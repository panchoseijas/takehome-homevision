package vision

import (
	"context"
	"fmt"
	"image"

	"gocv.io/x/gocv"
)

type Detector struct {
	params Params
}

func NewDetector(params Params) *Detector {
	return &Detector{params: params}
}

func (d *Detector) Params() Params {
	return d.params
}

func (d *Detector) Detect(ctx context.Context, data []byte) ([]Box, error) {
	config, err := ValidateImage(data, d.params.MaxPixels)
	if err != nil {
		return nil, err
	}
	bounds := image.Rect(0, 0, config.Width, config.Height)
	// Decode the image to grayscale
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
	// Apply adaptive threshold to the grayscale image to create a binary image
	err = gocv.AdaptiveThreshold(
		gray,
		&ink,
		255,
		gocv.AdaptiveThresholdGaussian,
		gocv.ThresholdBinaryInv,
		d.params.AdaptiveBlockSize,
		d.params.AdaptiveC,
	)
	if err != nil {
		return nil, fmt.Errorf("adaptive threshold: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// Create a ruling mask by keeping only pixels that belong to straight horizontal or vertical runs at least kernelLength long
	ruling, err := rulingMask(ink, d.params.LineKernelLength)
	if err != nil {
		return nil, err
	}
	defer ruling.Close()
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	candidates := d.findCandidates(ink, ruling)
	return finalize(candidates, bounds, d.params.DedupeIoU), nil
}

// keeps only pixels that belong to straight horizontal or vertical lines of at least kernelLength length.
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
