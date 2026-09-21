# HomeVision

Detect and annotate checkboxes in document images with a React frontend and a Python/OpenCV backend.

Appraisal reports record many of their answers as checkboxes, so reading a scanned form starts with finding each box and telling whether it is marked. Given a PNG or JPEG page, `POST /detect` returns every checkbox as a pixel `bbox` with an `is_checked` flag, and the web app draws the result over the image so you can check it by eye.

Detection is classical computer vision with OpenCV rather than a trained model. It is deterministic, runs quickly on a CPU, and every decision comes from a named threshold in [`params.py`](backend/src/homevision/vision/params.py). The four supplied samples are hand-annotated, and the test suite fails on any wrong or spurious box. It finds 287 of 289 checkboxes; the two misses are described under [known limitations](#known-limitations).

[approach.md](approach.md) covers the architectural decisions and how the detector was validated. [backend/README.md](backend/README.md) and [frontend/README.md](frontend/README.md) cover the API, local development, and tests.

## Live demo

- App: [https://homevision.jfseijas.com.ar](https://homevision.jfseijas.com.ar). Upload a PNG or JPEG (the samples are in [`backend/testdata`](backend/testdata)) and click **Detect checkboxes**.
- API: [https://api.homevision.jfseijas.com.ar](https://api.homevision.jfseijas.com.ar)

```sh
curl -F image=@backend/testdata/sample1-urar-page1.png https://api.homevision.jfseijas.com.ar/detect
```

## Run with Docker

Install Docker with Compose (Docker Desktop includes both), then run from this directory:

```sh
docker compose up --build
```

Open [http://localhost:5173](http://localhost:5173), choose an image from `backend/testdata`, and click **Detect checkboxes**. The API is at [http://localhost:8080](http://localhost:8080).

Stop with Ctrl+C, then remove the containers with:

```sh
docker compose down
```

## How detection works

A checkbox is a small, nearly square hole enclosed by straight lines, and each step narrows the image down to that. The images below are a crop of `backend/testdata/sample1-urar-page1.png`.

**1. Grayscale.** The upload is validated and decoded to a single channel.

![Grayscale crop](docs/pipeline/1-grayscale.png)

**2. Adaptive threshold.** Each pixel is compared with its local neighborhood, so ink becomes white and paper black even with uneven scans or shading.

![Thresholded ink](docs/pipeline/2-threshold.png)

**3. Ruling mask.** Morphological opening with a thin horizontal and a thin vertical kernel keeps only straight runs of at least 12 px. Box borders and table rules survive; text and the diagonal strokes of the X marks do not.

![Ruling mask](docs/pipeline/3-ruling.png)

**4. Find the holes.** Contours are extracted from the ruling mask together with their hierarchy. OpenCV traces both the outside of every shape and the holes inside it, and records which contour sits inside which:

![Contour hierarchy](docs/pipeline/contour-hierarchy.png)

In the figure, blue contours are outermost and green ones are nested inside another. The empty interior of a box is always a hole in its border, like 5, 7, 3 and 2, while a solid square like 0 has no hole and can never be a checkbox. The detector uses `RETR_CCOMP`, which flattens the tree to two levels, so the outside of a nested shape (6) counts as top level and only holes have a parent. Keeping the contours that have a parent selects the interiors.

Holes are then filtered by size, squareness, rectangularity, and border thickness. Green holes pass; red ones, such as table cells and letter fragments, are rejected:

![Candidate holes](docs/pipeline/4-holes.png)

**5. Classify and clean up.** Each interior is expanded by its measured border to get the outer box, then trimmed slightly and checked against the thresholded image from step 2: if at least 4% of it is ink, the box is checked. Overlapping duplicates are removed and boxes are sorted top to bottom. Below, green is checked and red is unchecked:

![Detected boxes](docs/pipeline/5-result.png)

## Known limitations

Against the hand-made annotations of the four samples, the detector finds 287 of 289 checkboxes at IoU 0.5 with no false positives and every state correct.

- Both misses are in sample 2. The "Neighborhood Boundaries" box is too faint for the adaptive threshold (its border is about 30 gray levels from the paper), and the hatched box has no clean rectangular hole.
- Solid or densely hatched fills are missed for the same reason: without a hole there is no candidate (`test_misses_solid_fill` documents this). No sample contains one.
- Skew and rotation are untested. Tilted scans shorten the straight runs the ruling mask depends on, and the supported range has not been measured.
- The thresholds were set by inspecting the four samples, so agreement there is a regression check, not evidence of accuracy on unseen documents. Box sizes are absolute pixels and cover roughly 100-300 DPI letter pages.
- Authentication, per-client rate limiting, and confidence-based review are not implemented.
