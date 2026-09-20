package vision

import (
	"image"
	"image/color"
	"testing"

	"gocv.io/x/gocv"
)

// page is a synthetic white document used to draw test fixtures. Drawing goes
// through OpenCV so the fixtures exercise the same anti-aliasing and rectangle
// conventions as real input.
type page struct {
	t   *testing.T
	mat gocv.Mat
}

var (
	black = color.RGBA{A: 255}
	// shade approximates the blue-gray cell shading in sample 3 once
	// converted to grayscale.
	shade = color.RGBA{R: 180, G: 200, B: 230, A: 255}
)

const defaultStroke = 2

func newPage(t *testing.T, width, height int) *page {
	t.Helper()
	mat := gocv.NewMatWithSizeFromScalar(gocv.NewScalar(255, 255, 255, 0), height, width, gocv.MatTypeCV8UC3)
	t.Cleanup(func() { mat.Close() })
	return &page{t: t, mat: mat}
}

// box draws an empty checkbox whose outer edges are r (exclusive max).
func (p *page) box(r image.Rectangle) {
	p.rect(r, black, defaultStroke)
}

func (p *page) rect(r image.Rectangle, c color.RGBA, thickness int) {
	p.t.Helper()
	// OpenCV centers thick strokes on the ideal edge; offset inward so the
	// drawn ink stays inside r.
	inner := r.Inset(thickness / 2)
	if err := gocv.Rectangle(&p.mat, inner, c, thickness); err != nil {
		p.t.Fatal(err)
	}
}

func (p *page) fill(r image.Rectangle, c color.RGBA) {
	p.t.Helper()
	if err := gocv.Rectangle(&p.mat, r, c, -1); err != nil {
		p.t.Fatal(err)
	}
}

func (p *page) line(from, to image.Point, thickness int) {
	p.t.Helper()
	if err := gocv.Line(&p.mat, from, to, black, thickness); err != nil {
		p.t.Fatal(err)
	}
}

// xMark draws two diagonals inside r, leaving a small gap to the border.
func (p *page) xMark(r image.Rectangle) {
	inner := r.Inset(4)
	p.line(inner.Min, inner.Max, defaultStroke)
	p.line(image.Pt(inner.Min.X, inner.Max.Y), image.Pt(inner.Max.X, inner.Min.Y), defaultStroke)
}

// tick draws a check mark inside r.
func (p *page) tick(r image.Rectangle) {
	inner := r.Inset(4)
	w, h := inner.Dx(), inner.Dy()
	low := image.Pt(inner.Min.X+w*2/5, inner.Max.Y)
	p.line(image.Pt(inner.Min.X, inner.Min.Y+h/2), low, defaultStroke)
	p.line(low, image.Pt(inner.Max.X, inner.Min.Y), defaultStroke)
}

func (p *page) text(s string, at image.Point, scale float64) {
	p.t.Helper()
	if err := gocv.PutText(&p.mat, s, at, gocv.FontHersheySimplex, scale, black, defaultStroke); err != nil {
		p.t.Fatal(err)
	}
}

// grid draws a table with the given column and row boundaries.
func (p *page) grid(cols, rows []int, thickness int) {
	top, bottom := rows[0], rows[len(rows)-1]
	left, right := cols[0], cols[len(cols)-1]
	for _, x := range cols {
		p.line(image.Pt(x, top), image.Pt(x, bottom), thickness)
	}
	for _, y := range rows {
		p.line(image.Pt(left, y), image.Pt(right, y), thickness)
	}
}

func (p *page) png() []byte {
	p.t.Helper()
	buf, err := gocv.IMEncode(gocv.PNGFileExt, p.mat)
	if err != nil {
		p.t.Fatal(err)
	}
	defer buf.Close()
	return append([]byte(nil), buf.GetBytes()...)
}

// detect runs the default detector on the page.
func (p *page) detect() []Box {
	p.t.Helper()
	boxes, err := NewDetector(DefaultParams()).Detect(p.t.Context(), p.png())
	if err != nil {
		p.t.Fatalf("Detect: %v", err)
	}
	return boxes
}

// assertBox checks that got matches want within tolerance pixels per edge.
func assertBox(t *testing.T, got Box, want image.Rectangle, wantChecked bool, tolerance int) {
	t.Helper()
	gotRect := got.Rect()
	edges := [][2]int{
		{gotRect.Min.X, want.Min.X}, {gotRect.Min.Y, want.Min.Y},
		{gotRect.Max.X, want.Max.X}, {gotRect.Max.Y, want.Max.Y},
	}
	for _, edge := range edges {
		if diff := edge[0] - edge[1]; diff > tolerance || diff < -tolerance {
			t.Errorf("box %v, want %v within %d px", gotRect, want, tolerance)
			break
		}
	}
	if got.Checked != wantChecked {
		t.Errorf("box %v checked = %t, want %t (fill %.3f)", gotRect, got.Checked, wantChecked, got.Debug.FillRatio)
	}
}
