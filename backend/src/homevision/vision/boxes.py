from dataclasses import dataclass, replace
from typing import NamedTuple


class Rect(NamedTuple):
    """Pixel rectangle with exclusive right and bottom edges."""

    x1: int
    y1: int
    x2: int
    y2: int

    @property
    def width(self) -> int:
        return self.x2 - self.x1

    @property
    def height(self) -> int:
        return self.y2 - self.y1

    @property
    def area(self) -> int:
        return self.width * self.height

    @property
    def empty(self) -> bool:
        return self.width <= 0 or self.height <= 0

    def inset(self, n: int) -> "Rect":
        return Rect(self.x1 + n, self.y1 + n, self.x2 - n, self.y2 - n)

    def intersect(self, other: "Rect") -> "Rect":
        return Rect(
            max(self.x1, other.x1),
            max(self.y1, other.y1),
            min(self.x2, other.x2),
            min(self.y2, other.y2),
        )


@dataclass(frozen=True)
class Debug:
    """Per-box diagnostics that explain a classification."""

    fill_ratio: float = 0.0
    ink_pixels: int = 0
    interior_area: int = 0
    border_px: tuple[int, int, int, int] = (0, 0, 0, 0)


@dataclass(frozen=True)
class Box:
    rect: Rect
    checked: bool
    debug: Debug = Debug()


def finalize(candidates: list[Box], bounds: Rect, dedupe_iou: float) -> list[Box]:
    """Clip boxes to the bounds, drop empty ones, and remove duplicates by IoU."""
    clamped = [
        replace(box, rect=rect)
        for box in candidates
        if not (rect := box.rect.intersect(bounds)).empty
    ]
    kept = dedupe(clamped, dedupe_iou)
    return sorted(kept, key=lambda box: (box.rect.y1, box.rect.x1))


def dedupe(boxes: list[Box], threshold: float) -> list[Box]:
    kept: list[Box] = []
    # Smallest first, so a nested double border resolves to the inner box.
    for candidate in sorted(boxes, key=lambda box: box.rect.area):
        if all(iou(candidate.rect, existing.rect) < threshold for existing in kept):
            kept.append(candidate)
    return kept


def iou(a: Rect, b: Rect) -> float:
    """Intersection over union of two rectangles."""
    inter = a.intersect(b)
    if inter.empty:
        return 0.0
    union = a.area + b.area - inter.area
    if union <= 0:
        return 0.0
    return inter.area / union
