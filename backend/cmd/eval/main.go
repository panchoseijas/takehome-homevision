package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/panchoseijas/takehome-homevision/backend/internal/eval"
	"github.com/panchoseijas/takehome-homevision/backend/internal/vision"
)

type options struct {
	dataDir        string
	predictionsDir string
	overlayDir     string
	minIoU         float64
	verbose        bool
}

func main() {
	var opts options
	flag.StringVar(&opts.dataDir, "data", "testdata", "directory searched recursively for <name>.truth.json files")
	flag.StringVar(&opts.predictionsDir, "predictions", "", "score <name>.json files in this directory instead of running the built-in detector")
	flag.StringVar(&opts.overlayDir, "overlay", "", "write a PNG per image showing hits, misses, and false positives")
	flag.Float64Var(&opts.minIoU, "iou", 0.5, "minimum intersection-over-union for a prediction to match an annotation")
	flag.BoolVar(&opts.verbose, "v", false, "list every miss, false positive, and wrong state")
	flag.Parse()

	if err := run(opts); err != nil {
		fmt.Fprintln(os.Stderr, "eval:", err)
		os.Exit(1)
	}
}

func run(opts options) error {
	paths, err := findTruthFiles(opts.dataDir)
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		return fmt.Errorf("no %s files under %s; create them with the frontend's annotation editor", truthSuffix, opts.dataDir)
	}
	if opts.overlayDir != "" {
		if err := os.MkdirAll(opts.overlayDir, 0o755); err != nil {
			return err
		}
	}

	detector := vision.NewDetector(vision.DefaultParams())
	table := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(table, "image\ttruth\tfound\tprecision\trecall\tF1\tstate acc\tend-to-end F1\tmean IoU\ttime")

	var total eval.Result
	var details strings.Builder
	for _, path := range paths {
		name := strings.TrimSuffix(filepath.Base(path), truthSuffix)
		truth, err := readTruth(path)
		if err != nil {
			return err
		}
		imagePath, err := imageFor(path)
		if err != nil {
			return err
		}

		var predicted []vision.Box
		elapsed := "-"
		if opts.predictionsDir != "" {
			predicted, err = readPredictions(filepath.Join(opts.predictionsDir, name+".json"))
		} else {
			var took time.Duration
			predicted, took, err = detect(detector, imagePath)
			elapsed = took.Round(time.Millisecond).String()
		}
		if err != nil {
			return err
		}

		result := eval.Score(truth, predicted, opts.minIoU)
		total.Add(result)
		writeRow(table, name, result, elapsed)
		describeErrors(&details, name, truth, predicted, result)

		if opts.overlayDir != "" {
			out := filepath.Join(opts.overlayDir, name+".png")
			if err := writeOverlay(imagePath, out, truth, predicted, result); err != nil {
				return err
			}
		}
	}
	writeRow(table, "TOTAL", total, "")
	if err := table.Flush(); err != nil {
		return err
	}

	if total.Ambiguous > 0 {
		fmt.Printf("\nAmbiguous annotations, either state accepted: %d\n", total.Ambiguous)
	}
	if opts.verbose && details.Len() > 0 {
		fmt.Printf("\n%s", details.String())
	}
	return nil
}

func detect(detector *vision.Detector, imagePath string) ([]vision.Box, time.Duration, error) {
	data, err := os.ReadFile(imagePath)
	if err != nil {
		return nil, 0, err
	}
	start := time.Now()
	boxes, err := detector.Detect(context.Background(), data)
	if err != nil {
		return nil, 0, fmt.Errorf("%s: %w", imagePath, err)
	}
	return boxes, time.Since(start), nil
}

func writeRow(table *tabwriter.Writer, name string, r eval.Result, elapsed string) {
	fmt.Fprintf(table, "%s\t%d\t%d\t%.3f\t%.3f\t%.3f\t%.3f\t%.3f\t%.3f\t%s\n",
		name, r.Truth, r.Predicted, r.Precision(), r.Recall(), r.F1(),
		r.StateAccuracy(), r.EndToEndF1(), r.MeanIoU(), elapsed)
}

func describeErrors(out *strings.Builder, name string, truth []eval.Truth, predicted []vision.Box, r eval.Result) {
	for _, t := range r.Missed {
		fmt.Fprintf(out, "%s: missed #%d %v\n", name, t, truth[t].Box.Rect())
	}
	for _, p := range r.FalsePositives {
		fmt.Fprintf(out, "%s: false positive %v\n", name, predicted[p].Rect())
	}
	for _, m := range r.Matches {
		if !m.StateCorrect {
			fmt.Fprintf(out, "%s: wrong state #%d %v, want checked=%t\n",
				name, m.Truth, truth[m.Truth].Box.Rect(), truth[m.Truth].Box.Checked)
		}
	}
}
