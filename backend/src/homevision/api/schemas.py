from pydantic import BaseModel

from homevision.vision import Box


class DebugResponse(BaseModel):
    fill_ratio: float
    ink_pixels: int
    interior_area: int
    border_px: tuple[int, int, int, int]


class BoxResponse(BaseModel):
    bbox: tuple[int, int, int, int]
    is_checked: bool
    debug: DebugResponse | None = None


class DetectResponse(BaseModel):
    boxes: list[BoxResponse]

    @classmethod
    def from_boxes(cls, boxes: list[Box], include_debug: bool) -> "DetectResponse":
        return cls(
            boxes=[
                BoxResponse(
                    bbox=box.rect,
                    is_checked=box.checked,
                    debug=DebugResponse(
                        fill_ratio=box.debug.fill_ratio,
                        ink_pixels=box.debug.ink_pixels,
                        interior_area=box.debug.interior_area,
                        border_px=box.debug.border_px,
                    )
                    if include_debug
                    else None,
                )
                for box in boxes
            ]
        )
