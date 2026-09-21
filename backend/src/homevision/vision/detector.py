import cv2
import numpy as np
from cv2.typing import MatLike

from homevision.vision.boxes import Box, Rect, finalize
from homevision.vision.candidates import find_candidates
from homevision.vision.image import CorruptImageError, validate_image
from homevision.vision.params import Params


class Detector:
    """Finds checkboxes in a scanned form page and classifies each as checked or not."""

    def __init__(self, params: Params | None = None) -> None:
        self.params = params or Params()

    def detect(self, data: bytes) -> list[Box]:
        """Return the checkboxes in a PNG or JPEG, in reading order.

        Raises ImageError for unsupported, oversized, or undecodable input.
        """
        validate_image(data, self.params.max_pixels)

        # Decode the image to grayscale
        gray = cv2.imdecode(np.frombuffer(data, dtype=np.uint8), cv2.IMREAD_GRAYSCALE)
        if gray is None or gray.size == 0:
            raise CorruptImageError
        height, width = gray.shape

        # Apply adaptive threshold to the grayscale image to create a binary image
        ink = cv2.adaptiveThreshold(
            gray,
            255,
            cv2.ADAPTIVE_THRESH_GAUSSIAN_C,
            cv2.THRESH_BINARY_INV,
            self.params.adaptive_block_size,
            self.params.adaptive_c,
        )
        ruling = ruling_mask(ink, self.params.line_kernel_length)

        candidates = find_candidates(ink, ruling, self.params)
        return finalize(candidates, Rect(0, 0, width, height), self.params.dedupe_iou)


def ruling_mask(ink: MatLike, kernel_length: int) -> MatLike:
    """Keep only pixels on straight horizontal or vertical runs of at least kernel_length."""
    horizontal_kernel = cv2.getStructuringElement(cv2.MORPH_RECT, (kernel_length, 1))
    vertical_kernel = cv2.getStructuringElement(cv2.MORPH_RECT, (1, kernel_length))
    close_kernel = cv2.getStructuringElement(cv2.MORPH_RECT, (3, 3))

    horizontal = cv2.morphologyEx(ink, cv2.MORPH_OPEN, horizontal_kernel)
    vertical = cv2.morphologyEx(ink, cv2.MORPH_OPEN, vertical_kernel)
    combined = cv2.bitwise_or(horizontal, vertical)
    return cv2.morphologyEx(combined, cv2.MORPH_CLOSE, close_kernel)
