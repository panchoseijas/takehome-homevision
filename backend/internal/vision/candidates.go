package vision

import (
	"image"
	"math"

	"gocv.io/x/gocv"
)

// findCandidates turns each enclosed hole of the ruling mask into a classified
// Box. Holes are the second level of the RETR_CCOMP hierarchy; their contour
// points lie on the border pixels that surround the hole, so the true white
// interior is the contour's bounding rectangle shrunk by one pixel.
func (d *Detector) findCandidates(gray, ink, ruling gocv.Mat) []Box {
	bounds := image.Rect(0, 0, gray.Cols(), gray.Rows())
	hierarchy := gocv.NewMat()
	defer hierarchy.Close()
	contours := gocv.FindContoursWithParams(ruling, &hierarchy, gocv.RetrievalCComp, gocv.ChainApproxSimple)
	defer contours.Close()

	var candidates []Box
	for i := 0; i < contours.Size(); i++ {
		// Hierarchy entries are [next, previous, firstChild, parent]; only
		// holes have a parent.
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

		// The measured ink around the hole includes any table rule the box
		// shares an edge with, so such boxes extend a few pixels into the
		// rule. That is accepted: the sample forms print those boxes with the
		// rule as one of their edges, and the interior alone would fail the
		// aspect test there.
		border := d.measureBorder(ruling, interior)
		outer := image.Rect(
			interior.Min.X-border[0], interior.Min.Y-border[1],
			interior.Max.X+border[2], interior.Max.Y+border[3],
		)
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

// plausibleInterior filters the hole before border measurement.
func (d *Detector) plausibleInterior(interior image.Rectangle) bool {
	return d.sidesWithin(interior, d.params.MinInteriorSide, d.params.MaxBoxSide)
}

// plausibleBox applies the documented size and aspect limits to the outer box.
func (d *Detector) plausibleBox(outer image.Rectangle) bool {
	return d.sidesWithin(outer, d.params.MinBoxSide, d.params.MaxBoxSide) && d.nearSquare(outer)
}

func (d *Detector) sidesWithin(r image.Rectangle, minSide, maxSide int) bool {
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

// rectangularity compares the contour area with the interior rectangle area.
// A hole bounded by straight ruling scores close to 1; L-shaped or ragged
// holes score lower.
func rectangularity(contourArea float64, interior image.Rectangle) float64 {
	rectArea := float64(interior.Dx() * interior.Dy())
	if rectArea <= 0 {
		return 0
	}
	return math.Min(contourArea/rectArea, 1)
}

// measureBorder walks outward from the middle of each interior edge through
// the ruling mask and returns the border thickness as [left, top, right,
// bottom], each at least 1 and at most MaxBorderThickness.
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

// mostlyHollow reports whether the interior takes at least
// MinInteriorFraction of the outer box area.
func (d *Detector) mostlyHollow(interior, outer image.Rectangle) bool {
	outerArea := outer.Dx() * outer.Dy()
	if outerArea <= 0 {
		return false
	}
	return float64(interior.Dx()*interior.Dy())/float64(outerArea) >= d.params.MinInteriorFraction
}

// onLightBackground reports whether the SurroundBand-wide ring just outside
// the box is on average brighter than MinSurroundGray. White glyphs on a dark
// bar produce enclosed holes that pass every geometric test; their
// surroundings are ink, a checkbox's are paper.
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

// runLength counts consecutive non-zero pixels starting at (x, y) and stepping
// by (dx, dy), stopping at limit or the image edge. It returns at least 1
// because a hole is by construction enclosed by at least one ruling pixel.
func runLength(mask gocv.Mat, x, y, dx, dy, limit int) int {
	count := 0
	for count < limit && x >= 0 && y >= 0 && x < mask.Cols() && y < mask.Rows() && mask.GetUCharAt(y, x) != 0 {
		count++
		x += dx
		y += dy
	}
	if count == 0 {
		return 1
	}
	return count
}

// classify measures the ink fraction of the interior after trimming
// InteriorMargin from each edge. It reads the original binary image, not the
// ruling mask, so X marks and ticks are visible.
func (d *Detector) classify(ink gocv.Mat, interior image.Rectangle, debug *Debug) bool {
	shorter := interior.Dx()
	if interior.Dy() < shorter {
		shorter = interior.Dy()
	}
	margin := int(math.Round(float64(shorter) * d.params.InteriorMargin))
	if margin < 1 {
		margin = 1
	}
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
