# HomeVision

Detect and annotate checkboxes in document images with a React frontend and a Go/OpenCV backend.

## Overview

Appraisal reports record many of their answers as checkboxes, so reading a scanned form automatically starts with finding each box and telling whether it is marked. HomeVision does that for a PNG or JPEG page: `POST /detect` returns every checkbox as a pixel `bbox` with an `is_checked` flag, and the web app draws the result over the image so it can be verified at a glance.

- **No trained model.** Detection is classical computer vision, so it is deterministic, fast on a CPU, and every decision traces back to a named threshold ([how it works](#how-detection-works)).
- **Measured, not eyeballed.** The four supplied samples are hand-annotated, and the test suite fails on any wrong or spurious box; the only misses are two known ones in sample 2 ([known limitations](#known-limitations)).
- **Easy to evaluate.** Try the live demo below, or start both services with one Docker command.

## Live demo

No installation needed:

- App: [https://homevision.jfseijas.com.ar](https://homevision.jfseijas.com.ar). Upload a PNG or JPEG (the samples are in `[backend/testdata](backend/testdata)`) and click **Detect checkboxes**.
- API: [https://api.homevision.jfseijas.com.ar](https://api.homevision.jfseijas.com.ar)

```sh
curl -F image=@backend/testdata/sample1-urar-page1.png https://api.homevision.jfseijas.com.ar/detect
```

## Run with Docker

Install Docker with Compose (Docker Desktop includes both), then run from this directory:

```sh
docker compose up --build
```

Open [http://localhost:5173](http://localhost:5173), choose an image from `backend/testdata`, and click **Detect checkboxes**. The API is also available at [http://localhost:8080](http://localhost:8080).

The first build takes a couple of minutes; later builds reuse Docker's cache. Stop with Ctrl+C, then remove the containers with:

```sh
docker compose down
```

## How detection works

The detector is classical computer vision with OpenCV, with no trained model. A checkbox is a small, nearly square hole enclosed by straight lines, and each step narrows the image down to that. The images below are a crop of `backend/testdata/sample1-urar-page1.png`.

**1. Grayscale.** The upload is validated and decoded to a single channel.

![Grayscale crop](docs/pipeline/1-grayscale.png)

**2. Adaptive threshold.** Each pixel is compared with its local neighborhood, so ink becomes white and paper black even with uneven scans or shading.

![Thresholded ink](docs/pipeline/2-threshold.png)

**3. Ruling mask.** Morphological opening with a thin horizontal and a thin vertical kernel keeps only straight runs of at least 12 px. Box borders and table rules survive; text and the diagonal strokes of the X marks fall apart.

![Ruling mask](docs/pipeline/3-ruling.png)

**4. Find the holes.** Contours are extracted from the ruling mask together with their hierarchy. OpenCV traces both the outside of every shape and the holes inside it, and records which contour sits inside which:

![Contour hierarchy](docs/pipeline/contour-hierarchy.png)

In the figure, blue contours are outermost and green ones are nested inside another. The empty interior of a box is always a hole in its border, like 5, 7, 3 and 2, while a solid square like 0 has no hole and can never be a checkbox. The detector uses `RETR_CCOMP`, which flattens the tree to two levels, so the outside of a nested shape (6) counts as top level and only holes have a parent. Keeping contours with a parent is what selects the interiors.

Holes are then filtered by size, squareness, rectangularity, and border thickness. Green holes pass; red ones, such as table cells and letter fragments, are rejected:

![Candidate holes](docs/pipeline/4-holes.png)

**5. Classify and clean up.** Each interior is expanded by its measured border to get the outer box, then trimmed slightly and checked against the thresholded image from step 2: if at least 4% of it is ink, the box is checked. Overlapping duplicates are removed and boxes are sorted top to bottom. Below, green is checked and red is unchecked:

![Detected boxes](docs/pipeline/5-result.png)

Every threshold lives in `[backend/internal/vision/params.go](backend/internal/vision/params.go)`.

## Known limitations

Against the hand-made annotations of the four samples, the detector finds 287 of 289 checkboxes at IoU 0.5 with no false positives and every state correct. Beyond that:

- **Two misses in sample 2.** The "Neighborhood Boundaries" box is too faint for the adaptive threshold (its border is about 30 gray levels from the paper), and the hatched box has no clean rectangular hole.
- **Solid or densely hatched fills are missed** for the same reason: without a hole there is no candidate (`TestDetectMissesSolidFill` documents this). No sample contains one.
- **Skew and rotation are untested.** Tilted scans shorten the straight runs the ruling mask depends on; the supported range has not been measured.
- **Tuned on the four samples.** The thresholds were set by inspecting them, so agreement there is a regression check, not evidence of accuracy on unseen documents. Box sizes are absolute pixels and cover roughly 100-300 DPI letter pages.

See [backend/README.md](backend/README.md) and [frontend/README.md](frontend/README.md) for the API, local development, and tests.
