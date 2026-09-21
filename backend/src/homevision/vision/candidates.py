import cv2
from cv2.typing import MatLike

from homevision.vision.boxes import Box, Debug, Rect
from homevision.vision.params import Params

type Border = tuple[int, int, int, int]  # left, top, right, bottom


def find_candidates(ink: MatLike, ruling: MatLike, params: Params) -> list[Box]:
    """Turn every box-shaped hole in the ruling mask into a classified Box.

    A checkbox is a small, near-rectangular hole enclosed by ruled lines. Each
    hole is filtered by size, shape, and border plausibility, then marked
    checked when enough ink falls inside it.
    """
    # RETR_CCOMP yields two levels: outer boundaries and the holes inside them.
    contours, hierarchy = cv2.findContours(ruling, cv2.RETR_CCOMP, cv2.CHAIN_APPROX_SIMPLE)

    candidates: list[Box] = []
    for i, contour in enumerate(contours):
        # Hierarchy entries are [next, previous, first_child, parent];
        # only holes have a parent.
        if hierarchy[0][i][3] < 0:
            continue
        # A hole's contour runs along the surrounding border pixels, so the
        # hole itself starts one pixel in.
        x, y, w, h = cv2.boundingRect(contour)
        interior = Rect(x, y, x + w, y + h).inset(1)
        if not sides_within(interior, params.min_interior_side, params.max_box_side):
            continue
        if rectangularity(cv2.contourArea(contour), interior) < params.min_rectangularity:
            continue

        border = measure_border(ruling, interior, params.max_border_thickness)
        outer = expand(interior, border)
        if not plausible_box(outer, params):
            border = cap_at_median(border)
            outer = expand(interior, border)
        if not plausible_box(outer, params):
            continue
        if interior.area / outer.area < params.min_interior_fraction:
            continue

        debug = measure_ink(ink, interior, border, params.interior_margin)
        # TODO(prod): return a confidence score and flag boxes near the threshold for human review
        checked = debug.fill_ratio >= params.fill_threshold
        candidates.append(Box(rect=outer, checked=checked, debug=debug))
    return candidates


def expand(interior: Rect, border: Border) -> Rect:
    """Grow the interior by its border thicknesses to get the box's outer edge."""
    left, top, right, bottom = border
    return Rect(interior.x1 - left, interior.y1 - top, interior.x2 + right, interior.y2 + bottom)


def cap_at_median(border: Border) -> Border:
    """Limit each thickness to the mean of the two middle values."""
    ordered = sorted(border)
    median = (ordered[1] + ordered[2]) // 2
    left, top, right, bottom = (min(side, median) for side in border)
    return left, top, right, bottom


def plausible_box(outer: Rect, params: Params) -> bool:
    """Whether the outer rectangle has checkbox-like size and aspect ratio."""
    if not sides_within(outer, params.min_box_side, params.max_box_side):
        return False
    aspect = outer.width / outer.height
    return aspect <= params.max_aspect_ratio and 1 / aspect <= params.max_aspect_ratio


def sides_within(rect: Rect, min_side: int, max_side: int) -> bool:
    return min_side <= rect.width <= max_side and min_side <= rect.height <= max_side


def rectangularity(contour_area: float, interior: Rect) -> float:
    """Fraction of the bounding rectangle the hole covers; 1.0 is a perfect rectangle."""
    return min(contour_area / interior.area, 1.0)


def measure_border(ruling: MatLike, interior: Rect, max_thickness: int) -> Border:
    """Measure each side's line thickness, walking outward from the side's midpoint."""
    mid_x = (interior.x1 + interior.x2) // 2
    mid_y = (interior.y1 + interior.y2) // 2
    return (
        run_length(ruling, interior.x1 - 1, mid_y, -1, 0, max_thickness),
        run_length(ruling, mid_x, interior.y1 - 1, 0, -1, max_thickness),
        run_length(ruling, interior.x2, mid_y, 1, 0, max_thickness),
        run_length(ruling, mid_x, interior.y2, 0, 1, max_thickness),
    )


def run_length(mask: MatLike, x: int, y: int, dx: int, dy: int, limit: int) -> int:
    """Count consecutive mask pixels from (x, y) along (dx, dy), between 1 and limit."""
    rows, cols = mask.shape
    count = 0
    while count < limit and 0 <= x < cols and 0 <= y < rows and mask[y, x] != 0:
        count += 1
        x += dx
        y += dy
    return max(count, 1)


def measure_ink(ink: MatLike, interior: Rect, border: Border, interior_margin: float) -> Debug:
    """Count ink inside the box, skipping a margin so border bleed is not read as a mark."""
    shorter = min(interior.width, interior.height)
    margin = max(round(shorter * interior_margin), 1)
    trimmed = interior.inset(margin)
    if trimmed.empty:
        trimmed = interior

    ink_pixels = cv2.countNonZero(ink[trimmed.y1 : trimmed.y2, trimmed.x1 : trimmed.x2])
    return Debug(
        fill_ratio=ink_pixels / trimmed.area,
        ink_pixels=ink_pixels,
        interior_area=trimmed.area,
        border_px=border,
    )
