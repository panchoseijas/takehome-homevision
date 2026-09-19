package eval

import (
	"math"
	"testing"

	"github.com/panchoseijas/takehome-homevision/backend/internal/vision"
)

func box(x, y, side int, checked bool) vision.Box {
	return vision.Box{X1: x, Y1: y, X2: x + side, Y2: y + side, Checked: checked}
}

func TestScoreCountsHitsMissesAndFalsePositives(t *testing.T) {
	truth := []Truth{
		{Box: box(0, 0, 40, true)},
		{Box: box(100, 0, 40, false)},
		{Box: box(200, 0, 40, false)},
	}
	predicted := []vision.Box{
		box(2, 2, 40, true),     // hit, right state
		box(101, 0, 40, true),   // hit, wrong state
		box(500, 500, 40, true), // false positive; truth[2] is missed
	}

	got := Score(truth, predicted, 0.5)

	if got.Matched != 2 || got.StateCorrect != 1 {
		t.Fatalf("matched = %d, state correct = %d, want 2 and 1", got.Matched, got.StateCorrect)
	}
	if len(got.Missed) != 1 || got.Missed[0] != 2 {
		t.Errorf("missed = %v, want [2]", got.Missed)
	}
	if len(got.FalsePositives) != 1 || got.FalsePositives[0] != 2 {
		t.Errorf("false positives = %v, want [2]", got.FalsePositives)
	}
	if want := 2.0 / 3.0; math.Abs(got.Precision()-want) > 1e-9 || math.Abs(got.Recall()-want) > 1e-9 {
		t.Errorf("precision = %v, recall = %v, want %v", got.Precision(), got.Recall(), want)
	}
	if got.StateAccuracy() != 0.5 {
		t.Errorf("state accuracy = %v, want 0.5", got.StateAccuracy())
	}
}

func TestScoreMatchesOneToOne(t *testing.T) {
	truth := []Truth{{Box: box(0, 0, 40, false)}}
	predicted := []vision.Box{box(4, 4, 40, false), box(1, 1, 40, false)}

	got := Score(truth, predicted, 0.5)

	if got.Matched != 1 || len(got.FalsePositives) != 1 {
		t.Fatalf("matched = %d, false positives = %v, want one of each", got.Matched, got.FalsePositives)
	}
	if got.Matches[0].Predicted != 1 {
		t.Errorf("matched prediction %d, want the closer one (1)", got.Matches[0].Predicted)
	}
}

func TestScoreRejectsOverlapBelowThreshold(t *testing.T) {
	truth := []Truth{{Box: box(0, 0, 40, false)}}
	predicted := []vision.Box{box(25, 25, 40, false)}

	got := Score(truth, predicted, 0.5)

	if got.Matched != 0 || len(got.Missed) != 1 || len(got.FalsePositives) != 1 {
		t.Errorf("got %+v, want a miss and a false positive", got)
	}
}

func TestScoreAcceptsEitherStateForAmbiguousBoxes(t *testing.T) {
	truth := []Truth{{Box: box(0, 0, 40, false), Ambiguous: true}}

	got := Score(truth, []vision.Box{box(0, 0, 40, true)}, 0.5)

	if got.StateCorrect != 1 || got.Ambiguous != 1 {
		t.Errorf("state correct = %d, ambiguous = %d, want 1 and 1", got.StateCorrect, got.Ambiguous)
	}
}

func TestScoreEmptyImageIsPerfect(t *testing.T) {
	got := Score(nil, nil, 0.5)

	if got.Precision() != 1 || got.Recall() != 1 {
		t.Errorf("precision = %v, recall = %v, want 1 and 1", got.Precision(), got.Recall())
	}
}
