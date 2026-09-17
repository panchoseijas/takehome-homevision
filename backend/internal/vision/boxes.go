package vision

import (
	"image"
	"sort"
)

// Box is one detected checkbox in original-image pixel coordinates. The
// origin is the top-left corner; X2 and Y2 are exclusive, matching
// image.Rectangle, so width is X2-X1 and height is Y2-Y1.
type Box struct {
	X1, Y1, X2, Y2 int
	Checked        bool
	Debug          Debug
}

// Debug carries per-box diagnostics that explain a classification.
type Debug struct {
	// FillRatio is the ink fraction of the trimmed interior compared against
	// Params.FillThreshold.
	FillRatio float64
	// InkPixels and InteriorArea are the numerator and denominator of FillRatio.
	InkPixels    int
	InteriorArea int
	// BorderPx is the ink thickness measured outward from the interior on
	// each side (left, top, right, bottom), used to expand the interior hole
	// to the reported box. It includes any table rule the box touches.
	BorderPx [4]int
}

// Rect returns the box as an image.Rectangle.
func (b Box) Rect() image.Rectangle {
	return image.Rect(b.X1, b.Y1, b.X2, b.Y2)
}

func (b Box) area() int {
	return (b.X2 - b.X1) * (b.Y2 - b.Y1)
}

// finalize clamps candidates to the image bounds, drops degenerate and
// duplicate boxes, and returns them in reading order (top to bottom, then left
// to right) so responses are deterministic.
func finalize(candidates []Box, bounds image.Rectangle, dedupeIoU float64) []Box {
	clamped := make([]Box, 0, len(candidates))
	for _, box := range candidates {
		rect := box.Rect().Intersect(bounds)
		if rect.Empty() {
			continue
		}
		box.X1, box.Y1, box.X2, box.Y2 = rect.Min.X, rect.Min.Y, rect.Max.X, rect.Max.Y
		clamped = append(clamped, box)
	}

	kept := dedupe(clamped, dedupeIoU)
	sort.Slice(kept, func(i, j int) bool {
		if kept[i].Y1 != kept[j].Y1 {
			return kept[i].Y1 < kept[j].Y1
		}
		return kept[i].X1 < kept[j].X1
	})
	return kept
}

// dedupe keeps the smaller of any two boxes whose IoU is at least threshold.
// The smaller box is preferred because nested duplicates come from double
// borders or tightly fitting table cells, and the innermost boundary is the
// one whose interior was classified.
func dedupe(boxes []Box, threshold float64) []Box {
	byArea := append([]Box(nil), boxes...)
	sort.SliceStable(byArea, func(i, j int) bool { return byArea[i].area() < byArea[j].area() })

	kept := make([]Box, 0, len(byArea))
	for _, candidate := range byArea {
		duplicate := false
		for _, existing := range kept {
			if iou(candidate, existing) >= threshold {
				duplicate = true
				break
			}
		}
		if !duplicate {
			kept = append(kept, candidate)
		}
	}
	return kept
}

// iou returns the intersection-over-union of two boxes.
func iou(a, b Box) float64 {
	inter := a.Rect().Intersect(b.Rect())
	if inter.Empty() {
		return 0
	}
	interArea := inter.Dx() * inter.Dy()
	union := a.area() + b.area() - interArea
	if union <= 0 {
		return 0
	}
	return float64(interArea) / float64(union)
}
