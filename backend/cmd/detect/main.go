// Command detect runs the checkbox detector on one image file and prints the
// same JSON that POST /detect returns. With -overlay it also writes a PNG with
// the boxes drawn on the original image (green: checked, red: unchecked).
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	"os"
	"time"

	"gocv.io/x/gocv"

	"github.com/panchoseijas/homevision/backend/internal/httpapi"
	"github.com/panchoseijas/homevision/backend/internal/vision"
)

func main() {
	debug := flag.Bool("debug", false, "include per-box diagnostics, like ?debug=1")
	overlay := flag.String("overlay", "", "write a PNG with the boxes drawn on the image")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s [-debug] [-overlay out.png] image.png\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}

	if err := run(flag.Arg(0), *debug, *overlay); err != nil {
		fmt.Fprintln(os.Stderr, "detect:", err)
		os.Exit(1)
	}
}

func run(path string, debug bool, overlayPath string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	detector := vision.NewDetector(vision.DefaultParams())
	start := time.Now()
	boxes, err := detector.Detect(context.Background(), data)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "%d boxes in %s\n", len(boxes), time.Since(start).Round(time.Millisecond))

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(httpapi.NewDetectResponse(boxes, debug)); err != nil {
		return err
	}

	if overlayPath == "" {
		return nil
	}
	return writeOverlay(data, boxes, overlayPath)
}

func writeOverlay(data []byte, boxes []vision.Box, path string) error {
	img, err := gocv.IMDecode(data, gocv.IMReadColor)
	if err != nil {
		return fmt.Errorf("decode for overlay: %w", err)
	}
	defer img.Close()
	if img.Empty() {
		return fmt.Errorf("decode for overlay: empty image")
	}

	checked := color.RGBA{R: 0, G: 170, B: 0, A: 255}
	unchecked := color.RGBA{R: 220, G: 0, B: 0, A: 255}
	for _, box := range boxes {
		c := unchecked
		if box.Checked {
			c = checked
		}
		// Exclusive right/bottom edges: draw on the last pixel of the box.
		rect := image.Rect(box.X1, box.Y1, box.X2-1, box.Y2-1)
		if err := gocv.Rectangle(&img, rect, c, 2); err != nil {
			return fmt.Errorf("draw box: %w", err)
		}
	}

	if ok := gocv.IMWrite(path, img); !ok {
		return fmt.Errorf("write overlay %s", path)
	}
	return nil
}
