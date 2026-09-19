package vision

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

const edgeTolerance = 2

func TestDetectBlankPageHasNoBoxes(t *testing.T) {
	p := newPage(t, 600, 400)
	if boxes := p.detect(); len(boxes) != 0 {
		t.Fatalf("boxes = %v, want none", boxes)
	}
}

func TestDetectTextOnlyPageHasNoBoxes(t *testing.T) {
	p := newPage(t, 900, 300)
	p.text("Borrower Homer Simpson 742 Evergreen Terrace", image.Pt(20, 80), 1.0)
	p.text("Occupant Owner Tenant Vacant 0 8 O D", image.Pt(20, 160), 1.4)
	p.text("bold Addendum Report", image.Pt(20, 260), 2.0)
	if boxes := p.detect(); len(boxes) != 0 {
		t.Fatalf("boxes = %v, want none", boxes)
	}
}

func TestDetectClassifiesMarks(t *testing.T) {
	tests := []struct {
		name        string
		mark        func(p *page, r image.Rectangle)
		wantChecked bool
	}{
		{"empty", func(*page, image.Rectangle) {}, false},
		{"x mark", (*page).xMark, true},
		{"tick", (*page).tick, true},
		{"diagonal scribble", func(p *page, r image.Rectangle) {
			for offset := -8; offset <= 8; offset += 8 {
				p.line(r.Min.Add(image.Pt(4+offset, 4)), r.Max.Add(image.Pt(-4+offset, -4)), defaultStroke)
			}
		}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newPage(t, 300, 200)
			want := image.Rect(100, 60, 140, 100)
			p.box(want)
			tt.mark(p, want)

			boxes := p.detect()
			if len(boxes) != 1 {
				t.Fatalf("got %d boxes, want 1: %v", len(boxes), boxes)
			}
			assertBox(t, boxes[0], want, tt.wantChecked, edgeTolerance)
		})
	}
}

// TestDetectMissesSolidFill documents a known limitation: a box filled solid,
// or nearly so, leaves no rectangular interior hole because the fill itself
// survives the ruling opening, so the hole-based candidate search cannot see
// it. None of the supplied samples contain such a box.
func TestDetectMissesSolidFill(t *testing.T) {
	for _, inset := range []int{0, 6} {
		p := newPage(t, 300, 200)
		want := image.Rect(100, 60, 140, 100)
		p.box(want)
		p.fill(want.Inset(inset), black)

		if boxes := p.detect(); len(boxes) != 0 {
			t.Fatalf("inset %d: got %v; if filled boxes are now detected, turn this into a positive test", inset, boxes)
		}
	}
}

func TestDetectHandlesBoxSizes(t *testing.T) {
	for _, side := range []int{22, 30, 56, 100} {
		p := newPage(t, 400, 300)
		want := image.Rect(100, 80, 100+side, 80+side)
		p.box(want)
		p.xMark(want)

		boxes := p.detect()
		if len(boxes) != 1 {
			t.Fatalf("side %d: got %d boxes, want 1: %v", side, len(boxes), boxes)
		}
		assertBox(t, boxes[0], want, true, edgeTolerance)
	}
}

func TestDetectBoxOnShadedCellIsUnchecked(t *testing.T) {
	p := newPage(t, 400, 200)
	p.fill(image.Rect(0, 40, 400, 160), shade)
	want := image.Rect(100, 70, 140, 110)
	p.box(want)

	boxes := p.detect()
	if len(boxes) != 1 {
		t.Fatalf("got %d boxes, want 1: %v", len(boxes), boxes)
	}
	assertBox(t, boxes[0], want, false, edgeTolerance)
}

func TestDetectIgnoresTableCellsButKeepsBoxesTouchingRules(t *testing.T) {
	p := newPage(t, 900, 400)
	// A table whose cells are wide rectangles, with row rules that the
	// checkboxes share as their top and bottom edges.
	p.grid([]int{50, 300, 550, 850}, []int{100, 150, 200, 250}, 2)
	checked := image.Rect(60, 150, 110, 200)
	unchecked := image.Rect(320, 200, 370, 250)
	p.box(checked)
	p.xMark(checked)
	p.box(unchecked)
	p.text("Yes", image.Pt(120, 190), 1.0)
	p.text("No", image.Pt(380, 240), 1.0)

	boxes := p.detect()
	if len(boxes) != 2 {
		t.Fatalf("got %d boxes, want 2: %v", len(boxes), boxes)
	}
	assertBox(t, boxes[0], checked, true, edgeTolerance)
	assertBox(t, boxes[1], unchecked, false, edgeTolerance)
}

func TestDetectMergesNestedDoubleBorder(t *testing.T) {
	p := newPage(t, 300, 200)
	outer := image.Rect(100, 60, 150, 110)
	p.box(outer)
	p.box(outer.Inset(4))
	p.xMark(outer.Inset(4))

	boxes := p.detect()
	if len(boxes) != 1 {
		t.Fatalf("got %d boxes, want 1: %v", len(boxes), boxes)
	}
	if !boxes[0].Checked {
		t.Errorf("nested box should be checked, got %+v", boxes[0])
	}
}

func TestDetectIgnoresHolesInDarkBars(t *testing.T) {
	p := newPage(t, 400, 300)
	// A sidebar like the black "SUBJECT" band in sample 1: white square
	// letters cut out of solid ink.
	p.fill(image.Rect(0, 0, 80, 300), black)
	for y := 20; y < 280; y += 60 {
		p.fill(image.Rect(20, y, 60, y+40), color.RGBA{R: 255, G: 255, B: 255, A: 255})
		p.fill(image.Rect(24, y+4, 56, y+36), black)
	}
	want := image.Rect(200, 100, 240, 140)
	p.box(want)

	boxes := p.detect()
	if len(boxes) != 1 {
		t.Fatalf("got %d boxes, want only the checkbox: %v", len(boxes), boxes)
	}
	assertBox(t, boxes[0], want, false, edgeTolerance)
}

func TestDetectReturnsReadingOrderAndStaysInBounds(t *testing.T) {
	p := newPage(t, 400, 300)
	rects := []image.Rectangle{
		image.Rect(0, 0, 40, 40), // touches the image origin
		image.Rect(300, 20, 340, 60),
		image.Rect(50, 200, 90, 240),
		image.Rect(360, 260, 400, 300), // touches the far corner
	}
	for _, r := range rects {
		p.box(r)
	}

	boxes := p.detect()
	if len(boxes) != len(rects) {
		t.Fatalf("got %d boxes, want %d: %v", len(boxes), len(rects), boxes)
	}
	for i, box := range boxes {
		if i > 0 && (box.Y1 < boxes[i-1].Y1 || (box.Y1 == boxes[i-1].Y1 && box.X1 < boxes[i-1].X1)) {
			t.Errorf("boxes not in reading order: %v", boxes)
		}
		if !box.Rect().In(image.Rect(0, 0, 400, 300)) {
			t.Errorf("box %v exceeds image bounds", box.Rect())
		}
	}
}

func TestDetectRejectsInvalidInput(t *testing.T) {
	detector := NewDetector(DefaultParams())
	tests := []struct {
		name string
		data []byte
		want error
	}{
		{"empty", nil, ErrUnsupportedFormat},
		{"text", []byte("plain text"), ErrUnsupportedFormat},
		{"gif", []byte("GIF89a\x01\x00\x01\x00\x00\x00\x00;"), ErrUnsupportedFormat},
		{"truncated png", newPage(t, 64, 64).png()[:40], ErrCorrupt},
		{"oversized png header", pngHeader(20_000, 20_000), ErrTooLarge},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := detector.Detect(t.Context(), tt.data)
			if !errors.Is(err, tt.want) {
				t.Errorf("err = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestDetectHonorsCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := NewDetector(DefaultParams()).Detect(ctx, newPage(t, 64, 64).png())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

// pngHeader returns the signature and IHDR chunk of a PNG with the given
// dimensions and nothing else, enough for image.DecodeConfig.
func pngHeader(width, height int) []byte {
	var buf bytes.Buffer
	buf.WriteString("\x89PNG\r\n\x1a\n")
	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:], uint32(width))
	binary.BigEndian.PutUint32(ihdr[4:], uint32(height))
	ihdr[8] = 8 // bit depth
	ihdr[9] = 2 // RGB
	_ = binary.Write(&buf, binary.BigEndian, uint32(len(ihdr)))
	chunk := append([]byte("IHDR"), ihdr...)
	buf.Write(chunk)
	_ = binary.Write(&buf, binary.BigEndian, crc32.ChecksumIEEE(chunk))
	return buf.Bytes()
}

// TestDetectSamples is a smoke test over the supplied documents: the detector
// must find boxes of both classes, keep them inside the image, and be
// deterministic. Accuracy against annotations is measured separately.
func TestDetectSamples(t *testing.T) {
	samples := []struct {
		file           string
		wantBothStates bool
	}{
		{"sample1-urar-page1.png", true},
		{"sample2-neighborhood-site-crop.jpeg", true},
		{"sample3-market-conditions-addendum.png", true},
		{"sample4-manufactured-home-report.png", true},
	}
	detector := NewDetector(DefaultParams())

	for _, sample := range samples {
		t.Run(sample.file, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("..", "..", "testdata", sample.file))
			if err != nil {
				t.Fatal(err)
			}
			config, err := ValidateImage(data, DefaultParams().MaxPixels)
			if err != nil {
				t.Fatal(err)
			}
			bounds := image.Rect(0, 0, config.Width, config.Height)

			boxes, err := detector.Detect(t.Context(), data)
			if err != nil {
				t.Fatal(err)
			}
			if len(boxes) < 10 {
				t.Fatalf("got %d boxes, want a form's worth", len(boxes))
			}

			checked, unchecked := 0, 0
			for _, box := range boxes {
				if !box.Rect().In(bounds) {
					t.Errorf("box %v exceeds %v", box.Rect(), bounds)
				}
				if box.Checked {
					checked++
				} else {
					unchecked++
				}
			}
			if sample.wantBothStates && (checked == 0 || unchecked == 0) {
				t.Errorf("checked = %d, unchecked = %d, want both", checked, unchecked)
			}

			again, err := detector.Detect(t.Context(), data)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(boxes, again) {
				t.Error("detection is not deterministic across runs")
			}
			t.Logf("%d boxes (%d checked, %d unchecked)", len(boxes), checked, unchecked)
		})
	}
}
