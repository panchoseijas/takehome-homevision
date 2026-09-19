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
	gray, err := decodeBrightestChannel(data)
	if err != nil {
		return nil, err
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
	ruling, err := rulingMask(ink, d.params.LineKernelLength, d.params.LineGapBridge)
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

// decodeBrightestChannel returns, per pixel, the maximum of the blue, green,
// and red channels. Black print is dark in all three and stays dark; a red
// watermark, blue signature, or tinted cell is bright in at least one and
// reads as paper, so only dark ink can form a border or a mark.
func decodeBrightestChannel(data []byte) (gocv.Mat, error) {
	color, err := gocv.IMDecode(data, gocv.IMReadColor)
	if err != nil {
		return gocv.Mat{}, fmt.Errorf("%w: %v", ErrCorrupt, err)
	}
	defer color.Close()
	if color.Empty() {
		return gocv.Mat{}, ErrCorrupt
	}

	channels := gocv.Split(color)
	defer func() {
		for _, channel := range channels {
			channel.Close()
		}
	}()
	brightest := gocv.NewMat()
	if err := gocv.Max(channels[0], channels[1], &brightest); err != nil {
		brightest.Close()
		return gocv.Mat{}, fmt.Errorf("brightest channel: %w", err)
	}
	if err := gocv.Max(brightest, channels[2], &brightest); err != nil {
		brightest.Close()
		return gocv.Mat{}, fmt.Errorf("brightest channel: %w", err)
	}
	return brightest, nil
}

// rulingMask keeps only pixels that belong to straight horizontal or vertical
// lines of at least kernelLength, then regrows each line along its own
// direction over ink within gapBridge of a dropout (see Params.LineGapBridge).
func rulingMask(ink gocv.Mat, kernelLength, gapBridge int) (gocv.Mat, error) {
	closeKernel := gocv.GetStructuringElement(gocv.MorphRect, image.Pt(3, 3))
	defer closeKernel.Close()

	horizontal, err := straightRuns(ink, image.Pt(kernelLength, 1), image.Pt(gapBridge+1, 1))
	if err != nil {
		return gocv.Mat{}, fmt.Errorf("horizontal ruling: %w", err)
	}
	defer horizontal.Close()
	vertical, err := straightRuns(ink, image.Pt(1, kernelLength), image.Pt(1, gapBridge+1))
	if err != nil {
		return gocv.Mat{}, fmt.Errorf("vertical ruling: %w", err)
	}
	defer vertical.Close()

	combined := gocv.NewMat()
	defer combined.Close()
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

// straightRuns opens ink with the run kernel, then extends the surviving runs
// by up to one run length in the same direction wherever the bridged ink
// continues. A diagonal stroke never survives the opening, so it has no run
// to extend; a border split by a dropout is rejoined from its long half.
func straightRuns(ink gocv.Mat, run, bridge image.Point) (gocv.Mat, error) {
	runKernel := gocv.GetStructuringElement(gocv.MorphRect, run)
	defer runKernel.Close()
	bridgeKernel := gocv.GetStructuringElement(gocv.MorphRect, bridge)
	defer bridgeKernel.Close()
	reachKernel := gocv.GetStructuringElement(gocv.MorphRect, image.Pt(2*run.X-1, 2*run.Y-1))
	defer reachKernel.Close()

	opened := gocv.NewMat()
	defer opened.Close()
	if err := gocv.MorphologyEx(ink, &opened, gocv.MorphOpen, runKernel); err != nil {
		return gocv.Mat{}, err
	}
	bridged := gocv.NewMat()
	defer bridged.Close()
	if err := gocv.MorphologyEx(ink, &bridged, gocv.MorphClose, bridgeKernel); err != nil {
		return gocv.Mat{}, err
	}
	reach := gocv.NewMat()
	defer reach.Close()
	if err := gocv.Dilate(opened, &reach, reachKernel); err != nil {
		return gocv.Mat{}, err
	}

	runs := gocv.NewMat()
	if err := gocv.BitwiseAnd(reach, bridged, &runs); err != nil {
		runs.Close()
		return gocv.Mat{}, err
	}
	return runs, nil
}
