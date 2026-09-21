package vision

type Params struct {
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
	// opening kernels that isolate straight ruling from text and marks.
	LineKernelLength int

	// MinBoxSide and MaxBoxSide bound the outer side length of a candidate.
	MinBoxSide int
	MaxBoxSide int
	// MinInteriorSide is the smallest interior (inside the border) that can
	// hold a legible mark. Specks enclosed by thick rules fall below it.
	MinInteriorSide int

	// MaxAspectRatio bounds width/height (and height/width) of the outer box.
	MaxAspectRatio float64

	// MinRectangularity is the minimum ratio between the interior contour area and its bounding-rectangle area
	MinRectangularity float64

	// MaxBorderThickness caps the measured border width when expanding an interior hole to the outer box edge
	MaxBorderThickness int

	// MinInteriorFraction is the minimum share of the outer box area taken by the interior
	MinInteriorFraction float64

	// InteriorMargin is the fraction of the shorter interior side trimmed from edge before measuring ink.
	InteriorMargin float64

	// FillThreshold is the minimum ink fraction of the trimmed interior for a box to be reported as checked.
	FillThreshold float64

	// DedupeIoU is the intersection-over-union above which two candidates are considered the same box
	DedupeIoU float64
}

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
		InteriorMargin:      0.12,
		FillThreshold:       0.04,
		DedupeIoU:           0.7,
	}
}
