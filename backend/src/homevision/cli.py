import argparse
import sys
import time
from pathlib import Path

import cv2
import numpy as np

from homevision.api.schemas import DetectResponse
from homevision.vision import Box, Detector, ImageError

CHECKED_BGR = (0, 170, 0)
UNCHECKED_BGR = (0, 0, 220)


def main() -> None:
    parser = argparse.ArgumentParser(description="Detect checkboxes in one image file.")
    parser.add_argument("image", type=Path, help="PNG or JPEG file")
    parser.add_argument(
        "--debug", action="store_true", help="include per-box diagnostics, like ?debug=1"
    )
    parser.add_argument(
        "--overlay", type=Path, help="write a PNG with the boxes drawn on the image"
    )
    args = parser.parse_args()

    try:
        data = args.image.read_bytes()
        started = time.perf_counter()
        boxes = Detector().detect(data)
    except (OSError, ImageError) as exc:
        sys.exit(f"detect: {exc}")
    elapsed_ms = (time.perf_counter() - started) * 1000
    print(f"{len(boxes)} boxes in {elapsed_ms:.0f}ms", file=sys.stderr)

    response = DetectResponse.from_boxes(boxes, include_debug=args.debug)
    print(response.model_dump_json(indent=2, exclude_none=True))

    if args.overlay:
        write_overlay(data, boxes, args.overlay)


def write_overlay(data: bytes, boxes: list[Box], path: Path) -> None:
    image = cv2.imdecode(np.frombuffer(data, dtype=np.uint8), cv2.IMREAD_COLOR)
    if image is None:
        sys.exit("detect: could not decode the image for the overlay")
    for box in boxes:
        color = CHECKED_BGR if box.checked else UNCHECKED_BGR
        # Exclusive right/bottom edges: draw on the last pixel of the box.
        x1, y1, x2, y2 = box.rect
        cv2.rectangle(image, (x1, y1), (x2 - 1, y2 - 1), color, 2)
    if not cv2.imwrite(str(path), image):
        sys.exit(f"detect: could not write overlay {path}")


if __name__ == "__main__":
    main()
