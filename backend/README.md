# HomeVision backend

Python HTTP API (FastAPI) that detects checkboxes in a document image and reports whether each one is checked. Detection is classical computer vision through OpenCV; the [root README](../README.md#how-detection-works) walks through each step.

## Prerequisites

- [uv](https://docs.astral.sh/uv/getting-started/installation/) (`brew install uv`, or the installer on that page).

That is all: `uv sync` downloads Python 3.13 if it is missing and installs the locked dependencies into `.venv`. OpenCV comes from the prebuilt `opencv-python-headless` wheel, so there is no native toolchain to set up.

## Build, test, run

```sh
cd backend
uv sync                        # create .venv from uv.lock
uv run ruff format --check .   # formatting
uv run ruff check .            # lint
uv run mypy                    # strict type check
uv run pytest
uv run homevision-server       # listens on 0.0.0.0:8080
uv run homevision-server --host 127.0.0.1 --port 9000 --max-concurrent 2
```

`--max-concurrent` caps detections running at once (default: number of CPUs); extra requests wait up to 5 s and then receive 503. With the server running, interactive API docs are at http://localhost:8080/docs.

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
| 400 | Not multipart, missing `image` field, invalid `debug` value, or image data that fails to decode |
| 413 | Body over 20 MiB, or image area over 25 megapixels |
| 415 | File is not PNG or JPEG |
| 503 | All detection slots busy for 5 s (`Retry-After` is set) |

## Command-line detector

`homevision-detect` runs the same detector on a file and prints the same JSON, optionally drawing the result:

```sh
uv run homevision-detect testdata/sample1-urar-page1.png
uv run homevision-detect --debug --overlay /tmp/sample1.png testdata/sample1-urar-page1.png
```

Green rectangles are checked boxes, red are unchecked. Detection time and box count are printed to stderr.

## Annotation policy

The sample annotations use these labeling rules:

- **Checked:** a deliberate dark mark inside a box, including an X, tick, slash, partial stroke, or solid fill.
- **Unchecked:** an empty box or background shading without a distinct mark. The uniformly hatched "Electricity → Public" box in sample 2 is labeled unchecked.
- **Ignored:** colored watermark or signature ink, and hand-drawn loops or strokes outside a checkbox.

These rules define the intended labels, not guaranteed detector behavior; solid or densely hatched boxes can be missed.

## Layout

- `src/homevision/server.py`: server entry point, flags, uvicorn settings.
- `src/homevision/cli.py`: command-line runner and overlay writer.
- `src/homevision/api`: upload validation, limits, JSON contract. `app.py` is the endpoint; `limits.py` enforces the body size; `schemas.py` holds the response models.
- `src/homevision/vision`: the detector, with no HTTP dependency. `params.py` holds every tunable; `detector.py` is the pipeline; `candidates.py` filters and classifies; `boxes.py` clamps, deduplicates, and sorts; `image.py` validates uploads from their header.
- `tests`: `test_detector.py`, `test_api.py`, and `page.py`, the synthetic form drawer.
- `testdata`: the four sample documents from the challenge.

Tests draw synthetic forms with OpenCV to cover marks, box sizes, table grids, shading, heavy rules, nested borders, ordering, and invalid input, and check the four samples against their hand-made annotations (`testdata/*.truth.json`): every detection must match an annotated box at IoU 0.5 with the right state, and only the two known misses in sample 2 are tolerated. API tests cover the response contract, every error status, the upload limit with and without `Content-Length`, and the 503 path with a blocked detector. Production follow-ups are marked `TODO(prod)` in the code (`git grep 'TODO(prod)'`); the detector's known limitations are listed in the [root README](../README.md#known-limitations).
