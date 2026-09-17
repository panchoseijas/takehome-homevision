// Package vision detects checkboxes in document images with classical
// computer vision (OpenCV through GoCV). It knows nothing about HTTP.
//
// Pipeline:
//
//  1. Decode to grayscale and binarize with an adaptive threshold so that ink
//     is white (255) and paper, including shaded cells, is black.
//  2. Keep only straight horizontal and vertical runs (morphological opening)
//     to obtain a mask of ruling. Glyphs, X marks, and ticks vanish from this
//     mask, so they cannot break a box border.
//  3. Treat each enclosed hole in the ruling mask as a candidate interior and
//     filter by size, aspect ratio, and rectangularity.
//     TODO(prod): a box filled solid (or with dense horizontal/vertical
//     hatching) has no rectangular hole and is missed; add a filled-square
//     candidate source if such marks appear in practice.
//  4. Expand each interior by its measured border thickness to get the outer
//     box, and classify it by the ink fraction of the interior in the
//     original binary image.
//  5. Clamp, deduplicate, and sort the boxes.
package vision

import (
	"context"
	"fmt"
	"image"

	"gocv.io/x/gocv"
)

// Detector runs the checkbox pipeline with fixed Params. It holds no image
// state, so one Detector is safe for concurrent use.
type Detector struct {
	params Params
}

// NewDetector returns a Detector configured with params.
func NewDetector(params Params) *Detector {
	return &Detector{params: params}
}

// Params returns the configuration in use.
func (d *Detector) Params() Params {
	return d.params
}

// Detect returns every checkbox found in a PNG or JPEG image. The context is
// checked between pipeline stages; individual OpenCV calls cannot be
// interrupted.
func (d *Detector) Detect(ctx context.Context, data []byte) ([]Box, error) {
	config, err := ValidateImage(data, d.params.MaxPixels)
	if err != nil {
		return nil, err
	}
	bounds := image.Rect(0, 0, config.Width, config.Height)

	gray, err := gocv.IMDecode(data, gocv.IMReadGrayScale)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCorrupt, err)
	}
	defer gray.Close()
	if gray.Empty() || gray.Cols() != config.Width || gray.Rows() != config.Height {
		return nil, ErrCorrupt
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	ink := gocv.NewMat()
	defer ink.Close()
	if err := gocv.AdaptiveThreshold(gray, &ink, 255, gocv.AdaptiveThresholdGaussian,
		gocv.ThresholdBinaryInv, d.params.AdaptiveBlockSize, d.params.AdaptiveC); err != nil {
		return nil, fmt.Errorf("adaptive threshold: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	ruling, err := rulingMask(ink, d.params.LineKernelLength)
	if err != nil {
		return nil, err
	}
	defer ruling.Close()
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	candidates := d.findCandidates(gray, ink, ruling)
	return finalize(candidates, bounds, d.params.DedupeIoU), nil
}

// rulingMask keeps only pixels that belong to straight horizontal or vertical
// runs at least kernelLength long, then closes 1-px gaps so box corners stay
// connected. The caller must Close the returned Mat.
func rulingMask(ink gocv.Mat, kernelLength int) (gocv.Mat, error) {
	horizontalKernel := gocv.GetStructuringElement(gocv.MorphRect, image.Pt(kernelLength, 1))
	defer horizontalKernel.Close()
	verticalKernel := gocv.GetStructuringElement(gocv.MorphRect, image.Pt(1, kernelLength))
	defer verticalKernel.Close()
	closeKernel := gocv.GetStructuringElement(gocv.MorphRect, image.Pt(3, 3))
	defer closeKernel.Close()

	horizontal := gocv.NewMat()
	defer horizontal.Close()
	vertical := gocv.NewMat()
	defer vertical.Close()
	combined := gocv.NewMat()
	defer combined.Close()

	if err := gocv.MorphologyEx(ink, &horizontal, gocv.MorphOpen, horizontalKernel); err != nil {
		return gocv.Mat{}, fmt.Errorf("horizontal opening: %w", err)
	}
	if err := gocv.MorphologyEx(ink, &vertical, gocv.MorphOpen, verticalKernel); err != nil {
		return gocv.Mat{}, fmt.Errorf("vertical opening: %w", err)
	}
	if err := gocv.BitwiseOr(horizontal, vertical, &combined); err != nil {
		return gocv.Mat{}, fmt.Errorf("combine rulings: %w", err)
	}

	ruling := gocv.NewMat()
	if err := gocv.MorphologyEx(combined, &ruling, gocv.MorphClose, closeKernel); err != nil {
		ruling.Close()
		return gocv.Mat{}, fmt.Errorf("close rulings: %w", err)
	}
	return ruling, nil
}
