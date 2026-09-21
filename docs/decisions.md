# Design decisions for `POST /detect`

Each entry records the options that were weighed, the choice, and what it costs. Numbers quoted here were measured on the four sample images in `backend/testdata/` while building the detector; D11 and D12 cover the hand-made annotations and what checking against them changed.

## D1. Detector stack: GoCV/OpenCV

Options: GoCV bindings to OpenCV; a pure Go implementation on the standard library; a Python sidecar process.

Chosen: GoCV, as the plan proposed. OpenCV supplies adaptive thresholding, morphology, and contour extraction with well-known semantics, so the detector is a short pipeline of named operations rather than hand-written image loops that would themselves need validation. Pure Go would remove the native dependency but replace it with a few hundred lines of custom code for the same primitives; a sidecar would split the service across two runtimes for one endpoint.

Cost: a native dependency. Building requires OpenCV installed on the host, and GoCV releases are tied to OpenCV versions. GoCV v0.43.0 documents OpenCV 4.12/4.13; the build here was verified against Homebrew OpenCV 4.14.0 (`gocv.OpenCVVersion()` reports `4.14.0`, all tests pass). The GoCV version is pinned in `go.mod`. Docker Compose is the documented run path so a reviewer needs no local OpenCV; its image builds against OpenCV 4.13.0 (`backend/Dockerfile`).

## D2. Binarization: adaptive Gaussian threshold

Options: global Otsu threshold; adaptive mean or Gaussian threshold.

Chosen: `AdaptiveThreshold(..., Gaussian, BinaryInv, block 31, C 15)`, so ink is 255 and paper is 0. Sample 3 has blue and gray shaded cells that a global threshold turns into solid ink, hiding the boxes on them, and sample 2 is a JPEG scan with uneven background. A local threshold treats shading as background because it is uniform within the block. The side-by-side Otsu comparison the plan listed for step 5 was not run beyond this observation.

Cost: solid regions wider than the block become hollow in the binary image (their centers are "paper" relative to the local mean). That is harmless here because candidates come from the ruling mask (D3), but it is one reason solid-filled boxes are not detected (D10).

## D3. Candidate source: holes in a straight-ruling mask

Options considered:

1. Contours of the raw binary image approximated to four vertices (`approxPolyDP`).
2. Holes in a mask of straight horizontal and vertical runs, built with two morphological openings (kernels 12x1 and 1x12), a bitwise OR, and a 3x3 closing, then `findContours` with `RETR_CCOMP`.
3. Hough line segments grouped into rectangles.

Chosen: option 2. An X or tick stroke that touches the border merges with it and breaks option 1; so does a box that shares an edge with a table rule, which happens in every sample. Straight-run filtering removes glyphs and diagonal marks before contours are taken, so the border stays closed and the interior remains a clean hole regardless of what is drawn inside. Option 3 needs segment grouping logic that is fragile on dense grids.

Cost: any enclosed rectangle of ruling is a candidate, so the filters in D4 carry the burden of rejecting table cells and glyph bowls. The opening kernel (12 px) must stay shorter than the smallest box side (20 px) and longer than most glyph strokes.

## D4. Candidate filters

All thresholds live in `vision.Params`; the reasons are recorded here. The filters, in order, and the sample behavior that motivated them:

| Filter | Default | Motivation |
| --- | --- | --- |
| Interior at least `MinInteriorSide` | 10 px | Specks enclosed by thick rules. |
| Rectangularity (contour area / bounding area) | 0.85 | Ragged or L-shaped holes. |
| Outer side within `MinBoxSide..MaxBoxSide` | 20..120 px | Bowls of small text glyphs (o, a, d, 8) at 12-19 px; the smallest annotated checkbox is 24x21 px. |
| Outer aspect ratio | 1.25 | Table cells; the samples' nearest square cells sit at 1.3. |
| Interior share of outer area `MinInteriorFraction` | 0.5 | Bowls of bold title glyphs: 22 px outer with 5-7 px strokes are one third interior; checkboxes are two thirds or more even when they share a rule. |

Sizes are absolute pixels rather than fractions of image width because sample 2 is a crop of a page; width-relative sizing would misjudge its scale. The defaults cover roughly 100-300 DPI letter forms. Rejected alternative: a fixed interior aspect test, which fails on sample 1 where boxes share thick top and bottom rules and the visible interior is 53x42.

Effect on the samples, boxes reported before and after the filters beyond size and aspect: sample 1 341 to 119, sample 2 62 to 41, sample 3 516 to 48, sample 4 164 to 77 (79 after the D12 change). Visual inspection of the overlays found no remaining glyph or sidebar false positives; the annotations in D11 later showed missed boxes, discussed in D12.

## D5. Classification: interior ink fraction

Options: ink fraction of the interior in the binary image; mean darkness of the grayscale interior; explicit stroke or diagonal detection.

Chosen: ink fraction of the interior after trimming 12% from each edge, measured on the binary image from D2 rather than on the ruling mask, with `FillThreshold` 0.04. Marks of any shape count, which matches the labeling policy (X, tick, slash, and fill are all "checked"). Grayscale darkness is sensitive to shading; diagonal detection over-fits X marks.

Observed separation on the samples: every unchecked box scored 0.000 and the lowest checked box scored 0.111, so the threshold has a wide margin on this data. The hatched box in sample 2 (Electricity, Public) has no clean rectangular hole and is not detected; the annotations label it `unchecked`.

## D6. Coordinates

Boxes use original-image pixels with the origin at the top-left; `x2` and `y2` are exclusive, matching Go's `image.Rectangle` and OpenCV's `Rect`, so width is `x2-x1`. Boxes are clamped to the image, duplicates at IoU >= 0.7 are removed keeping the smaller (innermost) box, and results are sorted top-to-bottom then left-to-right so responses are deterministic.

The reported box is the interior hole expanded by the ink thickness measured outward on each side (capped at 8 px). When a box shares an edge with a thicker table rule, the reported edge therefore includes the rule and extends a few pixels beyond the box's own stroke. Using the thinnest side as a uniform stroke was tried and rejected: sample 1's boxes then fail the aspect test because their visible interior is wider than tall. The over-extension is well inside an IoU 0.5 match.

## D7. Resolution and limits

Processing runs at native resolution. The largest sample is 10.7 MP and takes about 50 ms single-threaded (330 ms each with 12 concurrent requests on the development machine). Downscaling would speed this up but erases the 1-2 px strokes of sample 1's X marks. Memory is bounded by `MaxPixels` (25 MP, checked from the image header before decoding) and the 20 MiB body limit.

## D8. Validation and concurrency in the API

Format and dimensions are read with the standard library's `image.DecodeConfig` before OpenCV touches the bytes: PNG and JPEG are accepted (415 otherwise), oversized dimensions return 413, and a header that decodes but a body that does not returns 400. Only validated uploads compete for a detection slot.

OpenCV calls cannot be interrupted, so the handler bounds concurrency with a semaphore of `MaxConcurrent` slots (default `GOMAXPROCS`, flag `-max-concurrent`). A request waits up to 5 s for a slot and then receives 503 with `Retry-After`. The context is checked between pipeline stages so a disconnected client stops work at the next stage.

## D9. Error shape and `?debug=1`

Errors are `{"error": "message"}` with 400, 413, 415, 503, or 500; internal error details are logged, not returned. The success body is exactly `{"boxes":[{"bbox":[x1,y1,x2,y2],"is_checked":bool}]}` and `{"boxes":[]}` when nothing is found. With `?debug=1` (or `true`) each box gains a `debug` object with `fill_ratio`, `ink_pixels`, `interior_area`, and `border_px`; it is a pointer with `omitempty` so the default shape never carries the key.

The frontend's `ApiService.readError` reads the same `error` field and shows it for 4xx responses.

## D10. Known limitations and `TODO(prod)` follow-ups

Detector limitations:

- Boxes filled solid, or with dense horizontal/vertical hatching, have no rectangular hole and are missed (`TestDetectMissesSolidFill` documents this). No sample contains one.
- Hand-drawn marks beside a box rather than inside it, such as the quadrilateral next to "Water, Other" in sample 2, are not checkboxes and are ignored.
- Skewed or rotated scans reduce the straight-run mask; the supported skew range has not been measured yet.
- Thresholds were set by inspecting the four samples and have not been evaluated on held-out documents.

Production work deferred in code, each marked `TODO(prod)` where it applies (`git grep 'TODO(prod)'`):

- Authentication and per-client rate limiting on `POST /detect` (`backend/internal/httpapi/handler.go`).
- Persisting each input image (S3) and its result with the detector version (database) to build a held-out evaluation set, with encryption and a retention period because appraisal pages carry PII (`backend/internal/httpapi/handler.go`).
- A confidence score per box, so fill ratios near `FillThreshold` go to human review instead of being forced to checked or unchecked (`backend/internal/vision/candidates.go`).
- Frontend handling of a busy backend: 503 currently shows a generic "Server error"; it should say the server is busy and retry after `Retry-After` (`frontend/src/services/api.service.ts`).

## D11. Ground truth

Options for defining the correct result of an image: compare against a stored copy of the detector's own output (a regression check, not a measure of accuracy); annotate every box by hand in an external tool; or correct a detector draft in a purpose-built editor.

Chosen: the third. The correct result is a person's judgment under the mark classification policy in `docs/plan.md`, stored as `<image>.truth.json` in the `/detect` shape. The frontend's annotate mode seeds the draft from the detector and the annotator deletes false positives, flips states, and draws missed boxes, then saves the file. A draft starts from a detection, or empty when none has been run; the page does not reopen a saved file. Annotations exist for the four challenge samples.

Cost and caveats: a draft biases the annotator toward the current detector. Boxes the detector misses are absent from the draft and must be looked for deliberately, and a false positive in the draft can survive review. The four challenge samples tuned the thresholds, so agreement on them is a regression signal rather than a measure of generalization. The numbers in D12 came from a scoring command (`backend/cmd/eval`, one-to-one matching at IoU 0.5) that was later removed to keep the submission focused; it remains in the Git history before commit `22b53cc`, and the annotate mode and annotations stay. `TestDetectSamples` keeps the D12 result pinned: it matches detections to the annotations at the same IoU and fails on any false positive, wrong state, or new miss.

## D12. Changes driven by the annotations

Baseline against the annotations, scored at IoU 0.5: recall 0.986 (285 of 289) at precision 1.000.

| Finding | Cause | Change |
| --- | --- | --- |
| Sample 4 missed the "did / did not" pair. | Both sit directly under a heavy rule; its thickness was measured as the box's top border, giving an outer box of 30x38 that failed the aspect test. | When the measured box fails the size and aspect test, retry with every side capped at the median measured thickness, which drops the rule and keeps the box's own stroke. Boxes that already passed are untouched, so sample 1's boxes that legitimately share rules (D6) keep their coordinates. |

Result: recall 0.993 (287 of 289) at precision 1.000, state accuracy 1.000 on all four samples.

Known misses (there are no false positives), both in sample 2: the faint "Neighborhood Boundaries" box (border about 30 gray levels from paper, below `AdaptiveC`) and the hatched box, which has no clean rectangular hole.
