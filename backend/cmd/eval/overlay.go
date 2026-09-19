package main

import (
	"fmt"
	"image"
	"image/color"
	"strconv"

	"gocv.io/x/gocv"

	"github.com/panchoseijas/takehome-homevision/backend/internal/eval"
	"github.com/panchoseijas/takehome-homevision/backend/internal/vision"
)

var (
	correctColor       = color.RGBA{R: 0, G: 170, B: 0, A: 255}
	wrongStateColor    = color.RGBA{R: 255, G: 140, B: 0, A: 255}
	falsePositiveColor = color.RGBA{R: 220, G: 0, B: 0, A: 255}
	missedColor        = color.RGBA{R: 0, G: 90, B: 255, A: 255}
)

// writeOverlay draws the scored result: green for correct, orange for a wrong
// state, red for a false positive, and blue for a missed annotation. Errors
// carry the annotation index so they can be found in the truth file.
func writeOverlay(imagePath, outPath string, truth []eval.Truth, predicted []vision.Box, result eval.Result) error {
	img := gocv.IMRead(imagePath, gocv.IMReadColor)
	defer img.Close()
	if img.Empty() {
		return fmt.Errorf("read %s for overlay", imagePath)
	}

	for _, match := range result.Matches {
		if match.StateCorrect {
			if err := drawBox(&img, predicted[match.Predicted], correctColor, ""); err != nil {
				return err
			}
			continue
		}
		if err := drawBox(&img, predicted[match.Predicted], wrongStateColor, "#"+strconv.Itoa(match.Truth)); err != nil {
			return err
		}
	}
	for _, p := range result.FalsePositives {
		if err := drawBox(&img, predicted[p], falsePositiveColor, "fp"); err != nil {
			return err
		}
	}
	for _, t := range result.Missed {
		if err := drawBox(&img, truth[t].Box, missedColor, "#"+strconv.Itoa(t)); err != nil {
			return err
		}
	}

	if ok := gocv.IMWrite(outPath, img); !ok {
		return fmt.Errorf("write overlay %q", outPath)
	}
	return nil
}

func drawBox(img *gocv.Mat, box vision.Box, c color.RGBA, label string) error {
	// Exclusive right/bottom edges: draw on the last pixel of the box.
	rect := image.Rect(box.X1, box.Y1, box.X2-1, box.Y2-1)
	if err := gocv.Rectangle(img, rect, c, 2); err != nil {
		return fmt.Errorf("draw box: %w", err)
	}
	if label == "" {
		return nil
	}
	if err := gocv.PutText(img, label, image.Pt(box.X1, box.Y1-6), gocv.FontHersheySimplex, 0.7, c, 2); err != nil {
		return fmt.Errorf("draw label: %w", err)
	}
	return nil
}
