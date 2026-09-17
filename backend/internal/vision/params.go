package vision

// Params holds every tunable constant of the detector. Sizes are in pixels of
// the original image. The defaults target scanned or rendered US-letter forms
// between roughly 100 and 300 DPI, which covers the supplied samples (checkbox
// sides of about 20-40 px). They are deliberately absolute rather than relative
// to the image width: sample 2 is a crop of a page, so width-relative sizing
// would misjudge its scale.
type Params struct {
	// MaxPixels bounds the decoded image area (width*height) so a small upload
	// cannot force the server to allocate very large native buffers.
	MaxPixels int

	// AdaptiveBlockSize is the odd neighborhood size for adaptive
	// thresholding. It must be larger than the thickest stroke of interest so
	// the local mean is dominated by paper rather than ink.
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

	// MinBoxSide and MaxBoxSide bound the outer side length of a candidate.
	// The floor also rejects the bowls of small text glyphs (o, a, d, 8),
	// which survive the ruling opening at roughly 12-19 px in the samples.
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
		AdaptiveBlockSize:   31,
		AdaptiveC:           15,
		LineKernelLength:    12,
		MinBoxSide:          20,
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
