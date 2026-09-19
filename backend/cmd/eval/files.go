package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/panchoseijas/takehome-homevision/backend/internal/eval"
	"github.com/panchoseijas/takehome-homevision/backend/internal/httpapi"
	"github.com/panchoseijas/takehome-homevision/backend/internal/vision"
)

const truthSuffix = ".truth.json"

// truthFile is the /detect response shape plus an optional ambiguous flag.
type truthFile struct {
	Boxes []struct {
		BBox      [4]int `json:"bbox"`
		IsChecked bool   `json:"is_checked"`
		Ambiguous bool   `json:"ambiguous"`
	} `json:"boxes"`
}

func findTruthFiles(root string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && strings.HasSuffix(path, truthSuffix) {
			paths = append(paths, path)
		}
		return nil
	})
	return paths, err
}

// imageFor returns the image annotated by a truth file: same directory, same
// name, with an image extension.
func imageFor(truthPath string) (string, error) {
	stem := strings.TrimSuffix(truthPath, truthSuffix)
	for _, extension := range []string{".png", ".jpeg", ".jpg"} {
		if _, err := os.Stat(stem + extension); err == nil {
			return stem + extension, nil
		}
	}
	return "", fmt.Errorf("no image beside %s", truthPath)
}

func readTruth(path string) ([]eval.Truth, error) {
	var file truthFile
	if err := readJSON(path, &file); err != nil {
		return nil, err
	}
	truth := make([]eval.Truth, 0, len(file.Boxes))
	for _, item := range file.Boxes {
		truth = append(truth, eval.Truth{
			Box:       newBox(item.BBox, item.IsChecked),
			Ambiguous: item.Ambiguous,
		})
	}
	return truth, nil
}

func readPredictions(path string) ([]vision.Box, error) {
	var response httpapi.DetectResponse
	if err := readJSON(path, &response); err != nil {
		return nil, err
	}
	boxes := make([]vision.Box, 0, len(response.Boxes))
	for _, item := range response.Boxes {
		boxes = append(boxes, newBox(item.BBox, item.IsChecked))
	}
	return boxes, nil
}

func newBox(bbox [4]int, checked bool) vision.Box {
	return vision.Box{X1: bbox[0], Y1: bbox[1], X2: bbox[2], Y2: bbox[3], Checked: checked}
}

func readJSON(path string, into any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, into); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}
