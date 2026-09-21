import asyncio
import logging
import os
import time
from dataclasses import dataclass, field
from typing import Annotated, Protocol

from fastapi import FastAPI, HTTPException, Query, Request, UploadFile, status
from fastapi.concurrency import run_in_threadpool
from fastapi.exceptions import RequestValidationError
from fastapi.responses import JSONResponse
from starlette.exceptions import HTTPException as StarletteHTTPException

from homevision.api.limits import UploadLimitMiddleware
from homevision.api.schemas import DetectResponse, ErrorResponse
from homevision.vision import (
    Box,
    CorruptImageError,
    ImageError,
    ImageTooLargeError,
    Params,
    UnsupportedFormatError,
    validate_image,
)

logger = logging.getLogger(__name__)


class DetectorLike(Protocol):
    def detect(self, data: bytes) -> list[Box]: ...


@dataclass(frozen=True)
class Config:
    max_upload_bytes: int = 20 << 20
    max_pixels: int = Params().max_pixels
    # Detections running at once. OpenCV releases the GIL, so they run in parallel threads.
    max_concurrent: int = field(default_factory=lambda: os.cpu_count() or 1)
    queue_timeout_seconds: float = 5.0


IMAGE_ERRORS: dict[type[ImageError], tuple[int, str]] = {
    UnsupportedFormatError: (
        status.HTTP_415_UNSUPPORTED_MEDIA_TYPE,
        "uploaded file must be a PNG or JPEG image",
    ),
    ImageTooLargeError: (
        status.HTTP_413_CONTENT_TOO_LARGE,
        "image dimensions exceed the supported size",
    ),
    CorruptImageError: (status.HTTP_400_BAD_REQUEST, "image could not be decoded"),
}

ERROR_RESPONSES: dict[int | str, dict[str, object]] = {
    code: {"model": ErrorResponse, "description": description}
    for code, description in {
        400: "Not multipart, missing `image` field, or image data that fails to decode",
        413: "Body over the upload limit, or image area over the pixel limit",
        415: "File is not PNG or JPEG",
        503: "All detection slots busy; `Retry-After` is set",
    }.items()
}


def create_app(detector: DetectorLike, config: Config | None = None) -> FastAPI:
    config = config or Config()
    slots = asyncio.Semaphore(config.max_concurrent)

    app = FastAPI(title="HomeVision checkbox detection")
    app.add_middleware(UploadLimitMiddleware, max_bytes=config.max_upload_bytes)

    @app.exception_handler(StarletteHTTPException)
    async def http_error(_: Request, exc: StarletteHTTPException) -> JSONResponse:
        return JSONResponse({"error": exc.detail}, exc.status_code, headers=exc.headers)

    @app.exception_handler(RequestValidationError)
    async def validation_error(request: Request, exc: RequestValidationError) -> JSONResponse:
        return JSONResponse({"error": upload_error_message(request, exc)}, 400)

    # TODO(prod): add authentication and rate limiting
    @app.post("/detect", response_model_exclude_none=True, responses=ERROR_RESPONSES)
    async def detect(
        image: UploadFile,
        debug: Annotated[bool, Query(description="Include per-box diagnostics")] = False,
    ) -> DetectResponse:
        """Detect checkboxes in one PNG or JPEG and report whether each is checked."""
        data = await image.read()
        try:
            validate_image(data, config.max_pixels)
        except ImageError as exc:
            raise image_http_error(exc) from None

        try:
            async with asyncio.timeout(config.queue_timeout_seconds):
                await slots.acquire()
        except TimeoutError:
            raise HTTPException(
                status.HTTP_503_SERVICE_UNAVAILABLE,
                "server is busy, retry shortly",
                headers={"Retry-After": str(round(config.queue_timeout_seconds))},
            ) from None

        started = time.perf_counter()
        try:
            boxes = await run_in_threadpool(detector.detect, data)
        except ImageError as exc:
            raise image_http_error(exc) from None
        except Exception:
            logger.exception("detection failed")
            raise HTTPException(status.HTTP_500_INTERNAL_SERVER_ERROR, "detection failed") from None
        finally:
            slots.release()
        logger.info(
            "detection complete boxes=%d bytes=%d duration_ms=%.1f",
            len(boxes),
            len(data),
            (time.perf_counter() - started) * 1000,
        )

        # TODO(prod): store the image (S3) and result with detector version (DB) for evaluation;
        # mind PII and retention
        return DetectResponse.from_boxes(boxes, include_debug=debug)

    return app


def image_http_error(exc: ImageError) -> HTTPException:
    status_code, message = IMAGE_ERRORS[type(exc)]
    return HTTPException(status_code, message)


def upload_error_message(request: Request, exc: RequestValidationError) -> str:
    if not any(error["loc"] == ("body", "image") for error in exc.errors()):
        return "invalid request parameters"
    if not request.headers.get("content-type", "").startswith("multipart/form-data"):
        return "expected a multipart/form-data request"
    return 'missing "image" file field'
