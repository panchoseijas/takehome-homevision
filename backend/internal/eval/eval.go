// Package eval scores detected checkboxes against hand-verified annotations.
package eval

import (
	"cmp"
	"slices"

	"github.com/panchoseijas/takehome-homevision/backend/internal/vision"
)

// Truth is one annotated checkbox. Ambiguous marks a box whose state a careful
// reader could label either way (see the mark classification policy in
// docs/plan.md); it must still be found, but either state is accepted.
type Truth struct {
	Box       vision.Box
	Ambiguous bool
}

// Match pairs a truth index with a prediction index.
type Match struct {
	Truth, Predicted int
	IoU              float64
	StateCorrect     bool
}

// Result is the outcome for one image, or the sum over several.
type Result struct {
	Truth, Predicted int
	Matched          int
	StateCorrect     int
	Ambiguous        int
	IoUSum           float64

	Matches        []Match
	Missed         []int // truth indexes with no prediction
	FalsePositives []int // prediction indexes with no truth
}

// Score matches predictions to truth one-to-one, taking pairs in descending
// IoU order and accepting those at or above minIoU. Greedy matching is exact
// enough here because checkboxes do not overlap each other.
func Score(truth []Truth, predicted []vision.Box, minIoU float64) Result {
	var pairs []Match
	for t, want := range truth {
		for p, got := range predicted {
			if overlap := vision.IoU(want.Box, got); overlap >= minIoU {
				pairs = append(pairs, Match{Truth: t, Predicted: p, IoU: overlap})
			}
		}
	}
	slices.SortStableFunc(pairs, func(a, b Match) int { return cmp.Compare(b.IoU, a.IoU) })

	result := Result{Truth: len(truth), Predicted: len(predicted)}
	truthUsed := make([]bool, len(truth))
	predictedUsed := make([]bool, len(predicted))
	for _, pair := range pairs {
		if truthUsed[pair.Truth] || predictedUsed[pair.Predicted] {
			continue
		}
		truthUsed[pair.Truth], predictedUsed[pair.Predicted] = true, true

		want := truth[pair.Truth]
		pair.StateCorrect = want.Ambiguous || want.Box.Checked == predicted[pair.Predicted].Checked
		result.Matches = append(result.Matches, pair)
		result.Matched++
		result.IoUSum += pair.IoU
		if pair.StateCorrect {
			result.StateCorrect++
		}
	}

	for t, used := range truthUsed {
		if truth[t].Ambiguous {
			result.Ambiguous++
		}
		if !used {
			result.Missed = append(result.Missed, t)
		}
	}
	for p, used := range predictedUsed {
		if !used {
			result.FalsePositives = append(result.FalsePositives, p)
		}
	}
	return result
}

// Add accumulates counts for a total row. Per-box indexes are not merged
// because they are only meaningful within one image.
func (r *Result) Add(other Result) {
	r.Truth += other.Truth
	r.Predicted += other.Predicted
	r.Matched += other.Matched
	r.StateCorrect += other.StateCorrect
	r.Ambiguous += other.Ambiguous
	r.IoUSum += other.IoUSum
}

// Precision is the share of predictions that are real checkboxes.
func (r Result) Precision() float64 { return ratio(r.Matched, r.Predicted) }

// Recall is the share of real checkboxes that were found.
func (r Result) Recall() float64 { return ratio(r.Matched, r.Truth) }

// F1 is the harmonic mean of localization precision and recall.
func (r Result) F1() float64 { return harmonic(r.Precision(), r.Recall()) }

// StateAccuracy is the share of matched boxes with the right checked state.
func (r Result) StateAccuracy() float64 { return ratio(r.StateCorrect, r.Matched) }

// EndToEndF1 counts a prediction as correct only when it is both localized
// and classified correctly, which is what an API consumer experiences.
func (r Result) EndToEndF1() float64 {
	return harmonic(ratio(r.StateCorrect, r.Predicted), ratio(r.StateCorrect, r.Truth))
}

// MeanIoU is the average overlap of matched pairs.
func (r Result) MeanIoU() float64 {
	if r.Matched == 0 {
		return 0
	}
	return r.IoUSum / float64(r.Matched)
}

// ratio treats 0/0 as perfect: nothing was expected and nothing was reported.
func ratio(numerator, denominator int) float64 {
	if denominator == 0 {
		return 1
	}
	return float64(numerator) / float64(denominator)
}

func harmonic(a, b float64) float64 {
	if a+b == 0 {
		return 0
	}
	return 2 * a * b / (a + b)
}
