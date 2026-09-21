from collections.abc import Callable

import cv2
import numpy as np
import pytest
from fastapi.testclient import TestClient
from httpx2 import Response

from homevision.api.app import Config, create_app
from homevision.vision import (
    Box,
    CorruptImageError,
    Debug,
    Detector,
    ImageTooLargeError,
    Rect,
    UnsupportedFormatError,
)
from tests import TESTDATA

SAMPLE_DEBUG = Debug(fill_ratio=0.25, ink_pixels=100, interior_area=400, border_px=(2, 2, 2, 2))
SAMPLE_BOXES = [
    Box(rect=Rect(10, 20, 50, 60), checked=True, debug=SAMPLE_DEBUG),
    Box(rect=Rect(100, 20, 140, 60), checked=False),
]


class FakeDetector:
    """Returns canned results and records what it received."""

    def __init__(self, boxes: list[Box] | None = None, error: Exception | None = None) -> None:
        self.boxes = boxes or []
        self.error = error
        self.inputs: list[bytes] = []

    def detect(self, data: bytes) -> list[Box]:
        self.inputs.append(data)
        if self.error:
            raise self.error
        return self.boxes


def encode(extension: str, width: int, height: int) -> bytes:
    ok, encoded = cv2.imencode(extension, np.full((height, width), 255, dtype=np.uint8))
    assert ok
    return encoded.tobytes()


def upload(
    content: bytes, field: str = "image", filename: str = "form.png"
) -> dict[str, tuple[str, bytes]]:
    return {field: (filename, content)}


def test_returns_contract_shape() -> None:
    detector = FakeDetector(SAMPLE_BOXES)
    content = encode(".png", 64, 64)

    response = TestClient(create_app(detector)).post("/detect", files=upload(content))

    assert response.status_code == 200
    assert response.headers["content-type"] == "application/json"
    assert response.json() == {
        "boxes": [
            {"bbox": [10, 20, 50, 60], "is_checked": True},
            {"bbox": [100, 20, 140, 60], "is_checked": False},
        ]
    }
    assert detector.inputs == [content]


def test_empty_result_is_empty_array() -> None:
    client = TestClient(create_app(FakeDetector()))
    response = client.post("/detect", files=upload(encode(".jpg", 32, 32), filename="blank.jpg"))
    assert response.status_code == 200
    assert response.text == '{"boxes":[]}'


@pytest.mark.parametrize("value", ["1", "true"])
def test_debug_variant(value: str) -> None:
    client = TestClient(create_app(FakeDetector(SAMPLE_BOXES)))
    response = client.post(f"/detect?debug={value}", files=upload(encode(".png", 64, 64)))

    assert response.status_code == 200
    boxes = response.json()["boxes"]
    assert len(boxes) == 2 and all("debug" in box for box in boxes)
    assert boxes[0]["debug"] == {
        "fill_ratio": 0.25,
        "ink_pixels": 100,
        "interior_area": 400,
        "border_px": [2, 2, 2, 2],
    }


def test_debug_off_keeps_default_shape() -> None:
    client = TestClient(create_app(FakeDetector(SAMPLE_BOXES)))
    response = client.post("/detect?debug=0", files=upload(encode(".png", 64, 64)))
    assert "debug" not in response.text


def post_json(client: TestClient) -> Response:
    return client.post("/detect", content="{}", headers={"Content-Type": "application/json"})


def post_file(
    content: bytes, field: str = "image", filename: str = "form.png", debug: str = "0"
) -> Callable[[TestClient], Response]:
    return lambda client: client.post(
        "/detect", files=upload(content, field, filename), params={"debug": debug}
    )


@pytest.mark.parametrize(
    ("send", "want_status", "want_error"),
    [
        pytest.param(post_json, 400, "expected a multipart/form-data request", id="not multipart"),
        pytest.param(
            post_file(encode(".png", 8, 8), field="file"),
            400,
            'missing "image" file field',
            id="wrong field name",
        ),
        pytest.param(
            post_file(b"plain text", filename="notes.txt"),
            415,
            "uploaded file must be a PNG or JPEG image",
            id="not an image",
        ),
        pytest.param(
            post_file(b"GIF89a\x01\x00\x01\x00\x00\x00\x00;", filename="anim.gif"),
            415,
            "uploaded file must be a PNG or JPEG image",
            id="gif",
        ),
        pytest.param(
            post_file(encode(".png", 8, 8)[:20], filename="cut.png"),
            400,
            "image could not be decoded",
            id="truncated png",
        ),
        pytest.param(
            post_file(encode(".png", 200, 200), filename="huge.png"),
            413,
            "image dimensions exceed the supported size",
            id="oversized dimensions",
        ),
        pytest.param(
            post_file(bytes(5000), filename="big.png"),
            413,
            "request body exceeds the upload limit",
            id="oversized body",
        ),
        pytest.param(
            post_file(encode(".png", 8, 8), debug="maybe"),
            400,
            "invalid request parameters",
            id="bad debug flag",
        ),
    ],
)
def test_rejects_bad_uploads(
    send: Callable[[TestClient], Response], want_status: int, want_error: str
) -> None:
    detector = FakeDetector()
    client = TestClient(create_app(detector, Config(max_pixels=10_000, max_upload_bytes=4096)))

    response = send(client)

    assert response.status_code == want_status
    assert response.json() == {"error": want_error}
    assert detector.inputs == [], "detector ran for an invalid upload"


def test_upload_limit_counts_bodies_without_content_length() -> None:
    client = TestClient(create_app(FakeDetector(), Config(max_upload_bytes=4096)))

    # An iterator body is sent chunked, so the server never sees a Content-Length.
    response = client.post(
        "/detect",
        content=iter([bytes(1024)] * 8),
        headers={"Content-Type": "multipart/form-data; boundary=x"},
    )
    assert response.status_code == 413
    assert response.json() == {"error": "request body exceeds the upload limit"}


@pytest.mark.parametrize(
    ("method", "path", "want_status"),
    [
        ("GET", "/detect", 405),
        ("GET", "/", 404),
        ("GET", "/docs", 404),
        ("GET", "/redoc", 404),
        ("GET", "/openapi.json", 404),
    ],
)
def test_unknown_routes_return_json_errors(method: str, path: str, want_status: int) -> None:
    response = TestClient(create_app(FakeDetector())).request(method, path)
    assert response.status_code == want_status
    assert "error" in response.json()


@pytest.mark.parametrize(
    ("error", "want_status"),
    [
        (CorruptImageError(), 400),
        (UnsupportedFormatError(), 415),
        (ImageTooLargeError(), 413),
        (RuntimeError("opencv exploded"), 500),
    ],
)
def test_maps_detector_errors(error: Exception, want_status: int) -> None:
    client = TestClient(create_app(FakeDetector(error=error)))
    response = client.post("/detect", files=upload(encode(".png", 8, 8)))
    assert response.status_code == want_status
    assert "exploded" not in response.text, "internal error details leaked"


def test_detects_a_real_sample_end_to_end() -> None:
    sample = TESTDATA / "sample2-neighborhood-site-crop.jpeg"
    client = TestClient(create_app(Detector()))
    response = client.post("/detect", files=upload(sample.read_bytes(), filename=sample.name))

    assert response.status_code == 200
    boxes = response.json()["boxes"]
    assert boxes[0] == {"bbox": [155, 96, 180, 122], "is_checked": False}
    assert any(box["is_checked"] for box in boxes)
