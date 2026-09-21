import io

from PIL import Image

PNG_SIGNATURE = b"\x89PNG\r\n\x1a\n"
JPEG_SIGNATURE = b"\xff\xd8\xff"


class ImageError(Exception):
    """An uploaded image that cannot be processed."""


class UnsupportedFormatError(ImageError):
    def __init__(self) -> None:
        super().__init__("unsupported image format: expected PNG or JPEG")


class CorruptImageError(ImageError):
    def __init__(self) -> None:
        super().__init__("image could not be decoded")


class ImageTooLargeError(ImageError):
    def __init__(self) -> None:
        super().__init__("image dimensions exceed the supported size")


def validate_image(data: bytes, max_pixels: int) -> tuple[int, int]:
    """Check format and dimensions from the header alone, without decoding pixels.

    Returns (width, height).
    """
    if not data.startswith((PNG_SIGNATURE, JPEG_SIGNATURE)):
        raise UnsupportedFormatError

    try:
        with Image.open(io.BytesIO(data), formats=("PNG", "JPEG")) as image:
            width, height = image.size
    except Image.DecompressionBombError:
        raise ImageTooLargeError from None
    except Exception:
        raise CorruptImageError from None

    if width <= 0 or height <= 0:
        raise CorruptImageError
    if width * height > max_pixels:
        raise ImageTooLargeError
    return width, height
