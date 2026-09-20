package vision

import (
	"image"
	"math"
	"slices"

	"gocv.io/x/gocv"
)

func (d *Detector) findCandidates(gray, ink, ruling gocv.Mat) []Box {
	bounds := image.Rect(0, 0, gray.Cols(), gray.Rows())
	hierarchy := gocv.NewMat()
	defer hierarchy.Close()
	contours := gocv.FindContoursWithParams(ruling, &hierarchy, gocv.RetrievalCComp, gocv.ChainApproxSimple)
	defer contours.Close()

	candidates := []Box{}
	for i := range contours.Size() {
		// Hierarchy entries are [next, previous, firstChild, parent]
		// only holes have a parent.
		if hierarchy.GetVeciAt(0, i)[3] < 0 {
			continue
		}
		contour := contours.At(i)
		interior := gocv.BoundingRect(contour).Inset(1)
		if !d.plausibleInterior(interior) {
			continue
		}
		if rectangularity(gocv.ContourArea(contour), interior) < d.params.MinRectangularity {
			continue
		}

		border := d.measureBorder(ruling, interior)
		outer := expand(interior, border)
		if !d.plausibleBox(outer) {
			border = capAtMedian(border)
			outer = expand(interior, border)
		}
		if !d.plausibleBox(outer) || !d.mostlyHollow(interior, outer) || !d.onLightBackground(gray, outer, bounds) {
			continue
		}

		box := Box{X1: outer.Min.X, Y1: outer.Min.Y, X2: outer.Max.X, Y2: outer.Max.Y}
		box.Debug.BorderPx = border
		box.Checked = d.classify(ink, interior, &box.Debug)
		candidates = append(candidates, box)
	}
	return candidates
}

func expand(interior image.Rectangle, border [4]int) image.Rectangle {
	return image.Rect(
		interior.Min.X-border[0], interior.Min.Y-border[1],
		interior.Max.X+border[2], interior.Max.Y+border[3],
	)
}

// capAtMedian limits each thickness to the mean of the two middle values.
func capAtMedian(border [4]int) [4]int {
	sorted := border
	slices.Sort(sorted[:])
	median := (sorted[1] + sorted[2]) / 2
	for i := range border {
		border[i] = min(border[i], median)
	}
	return border
}

func (d *Detector) plausibleInterior(interior image.Rectangle) bool {
	return sidesWithin(interior, d.params.MinInteriorSide, d.params.MaxBoxSide)
}

func (d *Detector) plausibleBox(outer image.Rectangle) bool {
	return sidesWithin(outer, d.params.MinBoxSide, d.params.MaxBoxSide) && d.nearSquare(outer)
}

func sidesWithin(r image.Rectangle, minSide, maxSide int) bool {
	w, h := r.Dx(), r.Dy()
	return w >= minSide && h >= minSide && w <= maxSide && h <= maxSide
}

func (d *Detector) nearSquare(r image.Rectangle) bool {
	if r.Dx() <= 0 || r.Dy() <= 0 {
		return false
	}
	aspect := float64(r.Dx()) / float64(r.Dy())
	return aspect <= d.params.MaxAspectRatio && 1/aspect <= d.params.MaxAspectRatio
}

func rectangularity(contourArea float64, interior image.Rectangle) float64 {
	rectArea := float64(interior.Dx() * interior.Dy())
	if rectArea <= 0 {
		return 0
	}
	return math.Min(contourArea/rectArea, 1)
}

func (d *Detector) measureBorder(ruling gocv.Mat, interior image.Rectangle) [4]int {
	midX := (interior.Min.X + interior.Max.X) / 2
	midY := (interior.Min.Y + interior.Max.Y) / 2
	maxThickness := d.params.MaxBorderThickness

	return [4]int{
		runLength(ruling, interior.Min.X-1, midY, -1, 0, maxThickness),
		runLength(ruling, midX, interior.Min.Y-1, 0, -1, maxThickness),
		runLength(ruling, interior.Max.X, midY, 1, 0, maxThickness),
		runLength(ruling, midX, interior.Max.Y, 0, 1, maxThickness),
	}
}

func (d *Detector) mostlyHollow(interior, outer image.Rectangle) bool {
	outerArea := outer.Dx() * outer.Dy()
	if outerArea <= 0 {
		return false
	}
	return float64(interior.Dx()*interior.Dy())/float64(outerArea) >= d.params.MinInteriorFraction
}

func (d *Detector) onLightBackground(gray gocv.Mat, outer, bounds image.Rectangle) bool {
	neighborhood := outer.Inset(-d.params.SurroundBand).Intersect(bounds)
	inner := outer.Intersect(bounds)
	ringArea := neighborhood.Dx()*neighborhood.Dy() - inner.Dx()*inner.Dy()
	if ringArea <= 0 {
		return false
	}

	neighborhoodRegion := gray.Region(neighborhood)
	defer neighborhoodRegion.Close()
	innerRegion := gray.Region(inner)
	defer innerRegion.Close()

	neighborhoodSum := neighborhoodRegion.Mean().Val1 * float64(neighborhood.Dx()*neighborhood.Dy())
	innerSum := innerRegion.Mean().Val1 * float64(inner.Dx()*inner.Dy())
	return (neighborhoodSum-innerSum)/float64(ringArea) >= d.params.MinSurroundGray
}

func runLength(mask gocv.Mat, x, y, dx, dy, limit int) int {
	inMask := func(x, y int) bool {
		return x >= 0 && y >= 0 && x < mask.Cols() && y < mask.Rows()
	}

	count := 0
	for count < limit && inMask(x, y) && mask.GetUCharAt(y, x) != 0 {
		count++
		x += dx
		y += dy
	}
	if count == 0 {
		return 1
	}
	return count
}

func (d *Detector) classify(ink gocv.Mat, interior image.Rectangle, debug *Debug) bool {
	shorter := min(interior.Dx(), interior.Dy())
	margin := max(int(math.Round(float64(shorter)*d.params.InteriorMargin)), 1)
	trimmed := interior.Inset(margin)
	if trimmed.Empty() {
		trimmed = interior
	}

	region := ink.Region(trimmed)
	defer region.Close()

	debug.InkPixels = gocv.CountNonZero(region)
	debug.InteriorArea = trimmed.Dx() * trimmed.Dy()
	debug.FillRatio = float64(debug.InkPixels) / float64(debug.InteriorArea)
	return debug.FillRatio >= d.params.FillThreshold
}
