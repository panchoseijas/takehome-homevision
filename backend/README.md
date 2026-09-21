# HomeVision backend

Go HTTP API that detects checkboxes in a document image and reports whether each one is checked. Detection is classical computer vision through GoCV/OpenCV; see `../docs/decisions.md` for the reasoning behind each step.

## Prerequisites

- Go 1.26 or newer.
- OpenCV **4** with development headers, discoverable through `pkg-config`. GoCV v0.43.0 documents OpenCV 4.12/4.13; this project was built and tested against 4.14.0. Homebrew's plain `opencv` formula is now 5.0, which GoCV v0.43.0 does not support, so macOS needs `opencv@4` specifically.

macOS (Homebrew):

```sh
brew install opencv@4 pkgconf
export PKG_CONFIG_PATH="$(brew --prefix opencv@4)/lib/pkgconfig"
```

`opencv@4` is keg-only, so Homebrew leaves `opencv4.pc` off the default `pkg-config` search path. The export supplies it and is needed in every shell that runs `go build`, `go test`, or `go run` here; add it to your shell profile, or run `brew link --force opencv@4` once instead. (`pkgconf` provides the `pkg-config` command; skip it if you already have one.)

Debian/Ubuntu: follow the [GoCV installation guide](https://gocv.io/getting-started/linux/), which builds OpenCV 4 from source with `make install` and puts `opencv4.pc` on the default `pkg-config` search path, so no export is needed.

Check the toolchain before building. This must print a 4.x version:

```sh
pkg-config --modversion opencv4
```

The first `go build` compiles the GoCV cgo bindings and takes a few minutes; later builds are cached.

## Build, test, run

```sh
cd backend
go build ./...        # macOS: PKG_CONFIG_PATH must be set, see Prerequisites
go vet ./...
go test ./...
go run ./cmd/server            # listens on :8080
go run ./cmd/server -addr 127.0.0.1:9000 -max-concurrent 2
```

`-max-concurrent` caps detections running at once (default: number of CPUs); extra requests wait up to 5 s and then receive 503.

## API

### `POST /detect`

`multipart/form-data` with one PNG or JPEG file in the field `image`. Processing is synchronous and nothing is stored.

```sh
curl -F image=@testdata/sample2-neighborhood-site-crop.jpeg http://localhost:8080/detect
```

```json
{"boxes":[{"bbox":[155,96,180,122],"is_checked":false}, ...]}
```

- `bbox` is `[x1, y1, x2, y2]` in pixels of the uploaded image, origin top-left, `x2`/`y2` exclusive.
- `is_checked` is true when the interior carries a mark (X, tick, slash, or fill).
- Boxes are sorted top-to-bottom, then left-to-right. No checkboxes yields `{"boxes":[]}`.

Add `?debug=1` for per-box diagnostics that explain each classification (the frontend does not request them):

```sh
curl -F image=@testdata/sample2-neighborhood-site-crop.jpeg 'http://localhost:8080/detect?debug=1'
```

```json
{"boxes":[{"bbox":[155,96,180,122],"is_checked":false,"debug":{"fill_ratio":0,"ink_pixels":0,"interior_area":225,"border_px":[2,1,2,4]}}, ...]}
```

Errors are JSON, `{"error":"..."}`:

| Status | Cause |
| --- | --- |
| 400 | Not multipart, missing `image` field, or image data that fails to decode |
| 413 | Body over 20 MiB, or image area over 25 megapixels |
| 415 | File is not PNG or JPEG |
| 503 | All detection slots busy for 5 s (`Retry-After` is set) |

## Command-line detector

`cmd/detect` runs the same detector on a file and prints the same JSON, optionally drawing the result:

```sh
go run ./cmd/detect testdata/sample1-urar-page1.png
go run ./cmd/detect -debug -overlay /tmp/sample1.png testdata/sample1-urar-page1.png
```

Green rectangles are checked boxes, red are unchecked. Detection time and box count are printed to stderr.

## Layout

- `cmd/server`: HTTP server assembly, flags, graceful shutdown.
- `cmd/detect`: command-line runner and overlay writer.
- `internal/httpapi`: upload validation, limits, JSON contract.
- `internal/vision`: the detector. `params.go` holds every tunable (rationale in `../docs/decisions.md`, D2-D6); `detector.go` is the pipeline; `candidates.go` filters and classifies; `boxes.go` clamps, deduplicates, and sorts.
- `testdata`: the four sample documents from the challenge.

Tests draw synthetic forms with OpenCV to cover marks, box sizes, table grids, shading, heavy rules, nested borders, ordering, and invalid input, and check the four samples against their hand-made annotations (`testdata/*.truth.json`): every detection must match an annotated box at IoU 0.5 with the right state, and only the two known misses in sample 2 are tolerated. Production follow-ups are marked `TODO(prod)` in the code (`git grep 'TODO(prod)'`); the detector's known limitations are listed in `../docs/decisions.md` (D10).
