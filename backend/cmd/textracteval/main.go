// Command textracteval scores cached AWS Textract AnalyzeDocument (FORMS)
// responses against the hand-made annotations in testdata, using the same
// IoU >= 0.5 matching as TestDetectSamples. It makes no AWS calls: responses
// live in testdata/textract and were fetched once with the AWS CLI.
package main

import (
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
)

type box struct {
	x1, y1, x2, y2 int
	checked        bool
}

type truthFile struct {
	Boxes []struct {
		BBox    [4]int `json:"bbox"`
		Checked bool   `json:"is_checked"`
	} `json:"boxes"`
}

type textractFile struct {
	Blocks []struct {
		BlockType       string  `json:"BlockType"`
		Confidence      float64 `json:"Confidence"`
		SelectionStatus string  `json:"SelectionStatus"`
		Geometry        struct {
			BoundingBox struct {
				Left   float64 `json:"Left"`
				Top    float64 `json:"Top"`
				Width  float64 `json:"Width"`
				Height float64 `json:"Height"`
			} `json:"BoundingBox"`
		} `json:"Geometry"`
	} `json:"Blocks"`
}

func main() {
	samples := []string{
		"sample1-urar-page1.png",
		"sample2-neighborhood-site-crop.jpeg",
		"sample3-market-conditions-addendum.png",
		"sample4-manufactured-home-report.png",
	}
	testdata := "testdata"
	if len(os.Args) > 1 {
		testdata = os.Args[1]
	}

	var totalTruth, totalMatched, totalFalse, totalWrongState int
	for _, sample := range samples {
		base := strings.TrimSuffix(sample, filepath.Ext(sample))
		truth, err := readTruth(filepath.Join(testdata, base+".truth.json"))
		if err != nil {
			fatal(err)
		}
		w, h, err := imageSize(filepath.Join(testdata, sample))
		if err != nil {
			fatal(err)
		}
		detections, err := readTextract(filepath.Join(testdata, "textract", base+".textract.json"), w, h)
		if err != nil {
			fatal(err)
		}

		// Same greedy matching as TestDetectSamples: each detection claims
		// the unmatched annotation it overlaps best at IoU >= 0.5.
		matched := make([]bool, len(truth))
		falsePositives, wrongState := 0, 0
		for _, det := range detections {
			best, bestIoU := -1, 0.5
			for i, want := range truth {
				if overlap := iou(det, want); !matched[i] && overlap >= bestIoU {
					best, bestIoU = i, overlap
				}
			}
			if best < 0 {
				falsePositives++
				fmt.Printf("  false positive at (%d,%d)-(%d,%d) checked=%t\n", det.x1, det.y1, det.x2, det.y2, det.checked)
				continue
			}
			matched[best] = true
			if det.checked != truth[best].checked {
				wrongState++
				fmt.Printf("  wrong state at (%d,%d)-(%d,%d): got checked=%t, truth says %t\n",
					truth[best].x1, truth[best].y1, truth[best].x2, truth[best].y2, det.checked, truth[best].checked)
			}
		}
		missed := 0
		for i, ok := range matched {
			if !ok {
				missed++
				fmt.Printf("  missed annotation (%d,%d)-(%d,%d) checked=%t\n",
					truth[i].x1, truth[i].y1, truth[i].x2, truth[i].y2, truth[i].checked)
			}
		}

		found := len(truth) - missed
		fmt.Printf("%s: truth=%d textract=%d found=%d missed=%d false_positives=%d wrong_state=%d\n\n",
			sample, len(truth), len(detections), found, missed, falsePositives, wrongState)

		totalTruth += len(truth)
		totalMatched += found
		totalFalse += falsePositives
		totalWrongState += wrongState
	}

	fmt.Printf("TOTAL: truth=%d found=%d (%.1f%% recall) false_positives=%d wrong_state=%d (%.1f%% state accuracy on found)\n",
		totalTruth, totalMatched, 100*float64(totalMatched)/float64(totalTruth),
		totalFalse, totalWrongState, 100*float64(totalMatched-totalWrongState)/float64(totalMatched))
}

func readTruth(path string) ([]box, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var file truthFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	boxes := make([]box, len(file.Boxes))
	for i, b := range file.Boxes {
		boxes[i] = box{b.BBox[0], b.BBox[1], b.BBox[2], b.BBox[3], b.Checked}
	}
	return boxes, nil
}

func readTextract(path string, width, height int) ([]box, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var file textractFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	var boxes []box
	for _, b := range file.Blocks {
		if b.BlockType != "SELECTION_ELEMENT" {
			continue
		}
		bb := b.Geometry.BoundingBox
		boxes = append(boxes, box{
			x1:      int(bb.Left * float64(width)),
			y1:      int(bb.Top * float64(height)),
			x2:      int((bb.Left + bb.Width) * float64(width)),
			y2:      int((bb.Top + bb.Height) * float64(height)),
			checked: b.SelectionStatus == "SELECTED",
		})
	}
	return boxes, nil
}

func imageSize(path string) (int, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return 0, 0, fmt.Errorf("%s: %w", path, err)
	}
	return cfg.Width, cfg.Height, nil
}

func iou(a, b box) float64 {
	ix1, iy1 := max(a.x1, b.x1), max(a.y1, b.y1)
	ix2, iy2 := min(a.x2, b.x2), min(a.y2, b.y2)
	if ix2 <= ix1 || iy2 <= iy1 {
		return 0
	}
	inter := float64((ix2 - ix1) * (iy2 - iy1))
	areaA := float64((a.x2 - a.x1) * (a.y2 - a.y1))
	areaB := float64((b.x2 - b.x1) * (b.y2 - b.y1))
	return inter / (areaA + areaB - inter)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "textracteval:", err)
	os.Exit(1)
}
