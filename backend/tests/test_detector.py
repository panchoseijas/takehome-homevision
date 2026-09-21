import json
import struct
import zlib
from collections.abc import Callable
from pathlib import Path

import pytest

from homevision.vision import (
    Box,
    CorruptImageError,
    Detector,
    ImageError,
    ImageTooLargeError,
    Rect,
    UnsupportedFormatError,
)
from homevision.vision.boxes import iou
from tests import TESTDATA
from tests.page import BLACK, SHADE, Page

EDGE_TOLERANCE = 2


def assert_box(got: Box, want: Rect, want_checked: bool) -> None:
    """Check that got matches want within EDGE_TOLERANCE pixels per edge."""
    assert all(abs(g - w) <= EDGE_TOLERANCE for g, w in zip(got.rect, want, strict=True)), (
        f"box {got.rect}, want {want} within {EDGE_TOLERANCE} px"
    )
    assert got.checked == want_checked, f"box {got.rect} fill {got.debug.fill_ratio:.3f}"


def test_blank_page_has_no_boxes() -> None:
    assert Page(600, 400).detect() == []


def test_text_only_page_has_no_boxes() -> None:
    page = Page(900, 300)
    page.text("Borrower Homer Simpson 742 Evergreen Terrace", (20, 80), 1.0)
    page.text("Occupant Owner Tenant Vacant 0 8 O D", (20, 160), 1.4)
    page.text("bold Addendum Report", (20, 260), 2.0)
    assert page.detect() == []


def diagonal_scribble(page: Page, r: Rect) -> None:
    for offset in (-8, 0, 8):
        page.line((r.x1 + 4 + offset, r.y1 + 4), (r.x2 - 4 + offset, r.y2 - 4))


@pytest.mark.parametrize(
    ("mark", "want_checked"),
    [
        pytest.param(lambda page, r: None, False, id="empty"),
        pytest.param(Page.x_mark, True, id="x mark"),
        pytest.param(Page.tick, True, id="tick"),
        pytest.param(diagonal_scribble, True, id="diagonal scribble"),
    ],
)
def test_classifies_marks(mark: Callable[[Page, Rect], None], want_checked: bool) -> None:
    page = Page(300, 200)
    want = Rect(100, 60, 140, 100)
    page.box(want)
    mark(page, want)

    boxes = page.detect()
    assert len(boxes) == 1
    assert_box(boxes[0], want, want_checked)


@pytest.mark.parametrize("inset", [0, 6])
def test_misses_solid_fill(inset: int) -> None:
    """Documents a known limitation.

    A box filled solid, or nearly so, leaves no rectangular interior hole
    because the fill itself survives the ruling opening, so the hole-based
    candidate search cannot see it. None of the supplied samples contain such
    a box. If filled boxes are now detected, turn this into a positive test.
    """
    page = Page(300, 200)
    want = Rect(100, 60, 140, 100)
    page.box(want)
    page.fill(want.inset(inset), BLACK)
    assert page.detect() == []


@pytest.mark.parametrize("side", [22, 30, 56, 100])
def test_handles_box_sizes(side: int) -> None:
    page = Page(400, 300)
    want = Rect(100, 80, 100 + side, 80 + side)
    page.box(want)
    page.x_mark(want)

    boxes = page.detect()
    assert len(boxes) == 1
    assert_box(boxes[0], want, True)


def test_box_on_shaded_cell_is_unchecked() -> None:
    page = Page(400, 200)
    page.fill(Rect(0, 40, 400, 160), SHADE)
    want = Rect(100, 70, 140, 110)
    page.box(want)

    boxes = page.detect()
    assert len(boxes) == 1
    assert_box(boxes[0], want, False)


def test_ignores_table_cells_but_keeps_boxes_touching_rules() -> None:
    page = Page(900, 400)
    # A table whose cells are wide rectangles, with row rules that the
    # checkboxes share as their top and bottom edges.
    page.grid([50, 300, 550, 850], [100, 150, 200, 250], 2)
    checked = Rect(60, 150, 110, 200)
    unchecked = Rect(320, 200, 370, 250)
    page.box(checked)
    page.x_mark(checked)
    page.box(unchecked)
    page.text("Yes", (120, 190), 1.0)
    page.text("No", (380, 240), 1.0)

    boxes = page.detect()
    assert len(boxes) == 2
    assert_box(boxes[0], checked, True)
    assert_box(boxes[1], unchecked, False)


def test_keeps_square_box_under_thick_rule() -> None:
    page = Page(400, 200)
    # Sample 4 prints boxes directly under a heavy section rule; the rule
    # must not count as the box's own border and spoil its aspect ratio.
    page.fill(Rect(0, 48, 400, 60), BLACK)
    want = Rect(100, 60, 122, 82)
    page.rect(want, BLACK, 1)

    boxes = page.detect()
    assert len(boxes) == 1
    assert_box(boxes[0], want, False)


def test_merges_nested_double_border() -> None:
    page = Page(300, 200)
    outer = Rect(100, 60, 150, 110)
    page.box(outer)
    page.box(outer.inset(4))
    page.x_mark(outer.inset(4))

    boxes = page.detect()
    assert len(boxes) == 1
    assert boxes[0].checked


def test_returns_reading_order_and_stays_in_bounds() -> None:
    page = Page(400, 300)
    rects = [
        Rect(0, 0, 40, 40),  # touches the image origin
        Rect(300, 20, 340, 60),
        Rect(50, 200, 90, 240),
        Rect(360, 260, 400, 300),  # touches the far corner
    ]
    for r in rects:
        page.box(r)

    boxes = page.detect()
    assert len(boxes) == len(rects)
    order = [(box.rect.y1, box.rect.x1) for box in boxes]
    assert order == sorted(order)
    bounds = Rect(0, 0, 400, 300)
    assert all(box.rect.intersect(bounds) == box.rect for box in boxes)


def png_chunk(kind: bytes, payload: bytes) -> bytes:
    body = kind + payload
    return struct.pack(">I", len(payload)) + body + struct.pack(">I", zlib.crc32(body))


def png_stub(width: int, height: int) -> bytes:
    """A PNG that declares its dimensions but carries no pixel data."""
    ihdr = struct.pack(">IIBBBBB", width, height, 8, 2, 0, 0, 0)
    return b"\x89PNG\r\n\x1a\n" + png_chunk(b"IHDR", ihdr) + png_chunk(b"IDAT", b"")


@pytest.mark.parametrize(
    ("data", "want"),
    [
        pytest.param(b"", UnsupportedFormatError, id="empty"),
        pytest.param(b"plain text", UnsupportedFormatError, id="text"),
        pytest.param(b"GIF89a\x01\x00\x01\x00\x00\x00\x00;", UnsupportedFormatError, id="gif"),
        pytest.param(Page(64, 64).png()[:40], CorruptImageError, id="truncated png"),
        pytest.param(png_stub(64, 64)[:20], CorruptImageError, id="truncated header"),
        pytest.param(png_stub(64, 64), CorruptImageError, id="no pixel data"),
        pytest.param(png_stub(6_000, 6_000), ImageTooLargeError, id="over the pixel limit"),
        pytest.param(png_stub(20_000, 20_000), ImageTooLargeError, id="decompression bomb"),
    ],
)
def test_rejects_invalid_input(data: bytes, want: type[ImageError]) -> None:
    with pytest.raises(want):
        Detector().detect(data)


def read_truth(path: Path) -> list[Box]:
    """Load an annotation file, which has the /detect response shape."""
    return [
        Box(rect=Rect(*item["bbox"]), checked=item["is_checked"])
        for item in json.loads(path.read_text())["boxes"]
    ]


@pytest.mark.parametrize(
    ("file", "want_missed"),
    [
        ("sample1-urar-page1.png", 0),
        ("sample2-neighborhood-site-crop.jpeg", 2),
        ("sample3-market-conditions-addendum.png", 0),
        ("sample4-manufactured-home-report.png", 0),
    ],
)
def test_samples(file: str, want_missed: int) -> None:
    """Check the detector against the hand-made annotations stored beside each sample.

    A detection matches an annotated box at IoU 0.5. Every detection must match
    with the right state; the only tolerated misses are the two known ones in
    sample 2.
    """
    data = (TESTDATA / file).read_bytes()
    truth = read_truth((TESTDATA / file).with_suffix(".truth.json"))
    detector = Detector()

    boxes = detector.detect(data)

    unmatched = list(truth)
    for box in boxes:
        best = max(unmatched, key=lambda want: iou(box.rect, want.rect), default=None)
        assert best is not None and iou(box.rect, best.rect) >= 0.5, (
            f"box {box.rect} matches no annotation"
        )
        unmatched.remove(best)
        assert box.checked == best.checked, f"box {box.rect} fill {box.debug.fill_ratio:.3f}"

    assert len(unmatched) == want_missed, f"missed {[want.rect for want in unmatched]}"
    assert detector.detect(data) == boxes, "detection is not deterministic across runs"
