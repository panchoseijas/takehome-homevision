package vision

import (
	"cmp"
	"image"
	"slices"
)

type Box struct {
	X1, Y1, X2, Y2 int
	Checked        bool
	Debug          Debug
}

// Debug carries per-box diagnostics that explain a classification.
type Debug struct {
	FillRatio    float64
	InkPixels    int
	InteriorArea int
	BorderPx     [4]int
}

func (b Box) Rect() image.Rectangle {
	return image.Rect(b.X1, b.Y1, b.X2, b.Y2)
}

func (b Box) area() int {
	return (b.X2 - b.X1) * (b.Y2 - b.Y1)
}

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
	slices.SortFunc(kept, func(a, b Box) int {
		return cmp.Or(cmp.Compare(a.Y1, b.Y1), cmp.Compare(a.X1, b.X1))
	})
	return kept
}

func dedupe(boxes []Box, threshold float64) []Box {
	byArea := slices.Clone(boxes)
	slices.SortStableFunc(byArea, func(a, b Box) int { return cmp.Compare(a.area(), b.area()) })

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
