import cv2
import numpy as np

from homevision.vision import Box, Detector, Rect

type Color = tuple[int, int, int]  # BGR
type Point = tuple[int, int]

BLACK: Color = (0, 0, 0)
# Approximates the blue-gray cell shading in sample 3 once converted to grayscale.
SHADE: Color = (230, 200, 180)
DEFAULT_STROKE = 2


class Page:
    """A synthetic white document used to draw test fixtures.

    Drawing goes through OpenCV so the fixtures exercise the same anti-aliasing
    and rectangle conventions as real input.
    """

    def __init__(self, width: int, height: int) -> None:
        self.image = np.full((height, width, 3), 255, dtype=np.uint8)

    def box(self, r: Rect) -> None:
        """Draw an empty checkbox whose outer edges are r (exclusive max)."""
        self.rect(r, BLACK, DEFAULT_STROKE)

    def rect(self, r: Rect, color: Color, thickness: int) -> None:
        # OpenCV centers thick strokes on the ideal edge; offset inward so the
        # drawn ink stays inside r.
        inner = r.inset(thickness // 2)
        cv2.rectangle(self.image, (inner.x1, inner.y1, inner.width, inner.height), color, thickness)

    def fill(self, r: Rect, color: Color) -> None:
        cv2.rectangle(self.image, (r.x1, r.y1, r.width, r.height), color, -1)

    def line(self, start: Point, end: Point, thickness: int = DEFAULT_STROKE) -> None:
        cv2.line(self.image, start, end, BLACK, thickness)

    def x_mark(self, r: Rect) -> None:
        """Draw two diagonals inside r, leaving a small gap to the border."""
        inner = r.inset(4)
        self.line((inner.x1, inner.y1), (inner.x2, inner.y2))
        self.line((inner.x1, inner.y2), (inner.x2, inner.y1))

    def tick(self, r: Rect) -> None:
        inner = r.inset(4)
        low = (inner.x1 + inner.width * 2 // 5, inner.y2)
        self.line((inner.x1, inner.y1 + inner.height // 2), low)
        self.line(low, (inner.x2, inner.y1))

    def text(self, s: str, at: Point, scale: float) -> None:
        cv2.putText(self.image, s, at, cv2.FONT_HERSHEY_SIMPLEX, scale, BLACK, DEFAULT_STROKE)

    def grid(self, cols: list[int], rows: list[int], thickness: int) -> None:
        """Draw a table with the given column and row boundaries."""
        for x in cols:
            self.line((x, rows[0]), (x, rows[-1]), thickness)
        for y in rows:
            self.line((cols[0], y), (cols[-1], y), thickness)

    def png(self) -> bytes:
        ok, encoded = cv2.imencode(".png", self.image)
        assert ok
        return encoded.tobytes()

    def detect(self) -> list[Box]:
        return Detector().detect(self.png())
