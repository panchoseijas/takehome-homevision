package vision

type Params struct {
	// MaxPixels bounds the decoded image area (width*height) so a small upload
	// cannot force the server to allocate very large native buffers.
	MaxPixels int

	// AdaptiveBlockSize is the odd neighborhood size for adaptive
	// thresholding. It must be wide enough that the local mean is dominated
	// by paper even beside a checkbox filled with a bold X (about 40 px at
	// 300 DPI); at 31 the faint borders of such boxes fell below threshold.
	AdaptiveBlockSize int
	// AdaptiveC is subtracted from the local mean before comparison. Higher
	// values ignore light shading edges and JPEG noise; lower values keep
	// fainter strokes.
	AdaptiveC float32

	// LineKernelLength is the length of the 1-px-wide horizontal and vertical
	// opening kernels that isolate straight ruling from text and marks. It
	// must be shorter than the smallest checkbox side or the box border
	// disappears, and longer than most glyph strokes.
	LineKernelLength int
	// LineGapBridge is the longest gap, in pixels along a line, that a
	// surviving run may cross when it is regrown after the opening. A bold
	// mark beside a faint 1 px border darkens the local mean enough to drop
	// border pixels from the binary image; without regrowth the short half
	// of the split border is discarded and the box is no longer enclosed.
	LineGapBridge int

	// MinBoxSide and MaxBoxSide bound the outer side length of a candidate.
	// The floor also rejects the bowls of text glyphs, which survive the
	// ruling opening at 12-19 px for body text and 20-21 px for capitals
	// (D, O, Q) in headings; the smallest annotated checkbox is 23 px.
	MinBoxSide int
	MaxBoxSide int
	// MinInteriorSide is the smallest interior (inside the border) that can
	// hold a legible mark. Specks enclosed by thick rules fall below it.
	MinInteriorSide int
	// MaxAspectRatio bounds width/height (and height/width) of both the
	// interior and the outer box. Checkboxes are square within a few pixels;
	// ordinary table cells rarely are, and the samples' near-square cells sit
	// at about 1.3. Applying it to the interior as well rejects the bowls of
	// bold glyphs (d, e, o), whose thick and thin strokes make the outer
	// shape square while the hole stays tall.
	MaxAspectRatio float64
	// MinRectangularity is the minimum ratio between the interior contour area
	// and its bounding-rectangle area; holes bounded by straight ruling are
	// close to 1.0.
	MinRectangularity float64
	// MaxBorderThickness caps the measured border width when expanding an
	// interior hole to the outer box edge, so a thick table rule or filled
	// header bar cannot inflate a box.
	MaxBorderThickness int
	// MinInteriorFraction is the minimum share of the outer box area taken by
	// the interior. A checkbox is a thin ring around empty space, so it is
	// mostly interior even when it shares an edge with a table rule. The
	// bowls of bold glyphs (o, d, a) and letters cut out of dark bars are
	// mostly stroke and score well below one half.
	MinInteriorFraction float64

	// SurroundBand is the width in pixels of the ring examined just outside
	// the outer box, and MinSurroundGray (0-255) is the minimum mean gray
	// level of that ring. Checkboxes sit on paper, so the ring is bright; a
	// hole punched into a dark header bar or sidebar (white text on a dark
	// background) is surrounded by ink.
	SurroundBand    int
	MinSurroundGray float64

	// InteriorMargin is the fraction of the shorter interior side trimmed from
	// each edge before measuring ink, so anti-aliased border pixels do not
	// count as a mark.
	InteriorMargin float64
	// FillThreshold is the minimum ink fraction of the trimmed interior for a
	// box to be reported as checked. Thin X marks land around 0.08-0.2; empty
	// boxes land below 0.01.
	FillThreshold float64

	// DedupeIoU is the intersection-over-union above which two candidates are
	// considered the same box; the smaller one is kept.
	DedupeIoU float64
}

// DefaultParams returns the defaults described on Params.
func DefaultParams() Params {
	return Params{
		MaxPixels:           25_000_000,
		AdaptiveBlockSize:   71,
		AdaptiveC:           15,
		LineKernelLength:    12,
		LineGapBridge:       1,
		MinBoxSide:          22,
		MaxBoxSide:          120,
		MinInteriorSide:     10,
		MaxAspectRatio:      1.25,
		MinRectangularity:   0.85,
		MaxBorderThickness:  8,
		MinInteriorFraction: 0.5,
		SurroundBand:        4,
		MinSurroundGray:     128,
		InteriorMargin:      0.12,
		FillThreshold:       0.04,
		DedupeIoU:           0.7,
	}
}
