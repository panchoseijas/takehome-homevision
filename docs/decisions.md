# Design decisions for `POST /detect`

Each entry records the options that were weighed, the choice, and what it costs. Numbers quoted here were measured on the four sample images in `backend/testdata/` while building the detector; D11 and D12 cover the hand-made annotations and what checking against them changed.

## D1. Detector stack: GoCV/OpenCV

Options: GoCV bindings to OpenCV; a pure Go implementation on the standard library; a Python sidecar process.

Chosen: GoCV, as the plan proposed. OpenCV supplies adaptive thresholding, morphology, and contour extraction with well-known semantics, so the detector is a short pipeline of named operations rather than hand-written image loops that would themselves need validation. Pure Go would remove the native dependency but replace it with a few hundred lines of custom code for the same primitives; a sidecar would split the service across two runtimes for one endpoint.

Cost: a native dependency. Building requires OpenCV installed on the host, and GoCV releases are tied to OpenCV versions. GoCV v0.43.0 documents OpenCV 4.12/4.13; the build here was verified against Homebrew OpenCV 4.14.0 (`gocv.OpenCVVersion()` reports `4.14.0`, all tests pass). The version is pinned in `go.mod`, and a container image remains the fallback run path for the packaging step.

## D2. Binarization: adaptive Gaussian threshold

Options: global Otsu threshold; adaptive mean or Gaussian threshold.

Chosen: `AdaptiveThreshold(..., Gaussian, BinaryInv, block 71, C 15)` (block 31 until D12), so ink is 255 and paper is 0. Sample 3 has blue and gray shaded cells that a global threshold turns into solid ink, hiding the boxes on them, and sample 2 is a JPEG scan with uneven background. A local threshold treats shading as background because it is uniform within the block. The plan's Otsu comparison remains a step-5 task.

Cost: solid regions wider than the block become hollow in the binary image (their centers are "paper" relative to the local mean). That is harmless here because candidates come from the ruling mask (D3), but it is one reason solid-filled boxes are not detected (D10).

## D3. Candidate source: holes in a straight-ruling mask

Options considered:

1. Contours of the raw binary image approximated to four vertices (`approxPolyDP`).
2. Holes in a mask of straight horizontal and vertical runs, built with two morphological openings (kernels 12x1 and 1x12), a bitwise OR, and a 3x3 closing, then `findContours` with `RETR_CCOMP`.
3. Hough line segments grouped into rectangles.

Chosen: option 2. An X or tick stroke that touches the border merges with it and breaks option 1; so does a box that shares an edge with a table rule, which happens in every sample. Straight-run filtering removes glyphs and diagonal marks before contours are taken, so the border stays closed and the interior remains a clean hole regardless of what is drawn inside. Option 3 needs segment grouping logic that is fragile on dense grids.

Cost: any enclosed rectangle of ruling is a candidate, so the filters in D4 carry the burden of rejecting table cells, glyph bowls, and letters cut out of dark bars. The opening kernel (12 px) must stay shorter than the smallest box side (22 px) and longer than most glyph strokes.

## D4. Candidate filters

All thresholds live in `vision.Params` with a one-line reason each. The filters, in order, and the sample behavior that motivated them:

| Filter | Default | Motivation |
| --- | --- | --- |
| Interior at least `MinInteriorSide` | 10 px | Specks enclosed by thick rules. |
| Rectangularity (contour area / bounding area) | 0.85 | Ragged or L-shaped holes. |
| Outer side within `MinBoxSide..MaxBoxSide` | 22..120 px | Bowls of text glyphs at 12-19 px, and heading capitals (D, O, Q) at 20-21 px; the smallest annotated checkbox is 23 px. |
| Outer aspect ratio | 1.25 | Table cells; the samples' nearest square cells sit at 1.3. |
| Interior share of outer area `MinInteriorFraction` | 0.5 | Bowls of bold title glyphs: 22 px outer with 5-7 px strokes are one third interior; checkboxes are two thirds or more even when they share a rule. |
| Mean gray of a 4 px ring outside the box `MinSurroundGray` | 128 | White letters cut out of the black and blue sidebars in samples 1 and 3 pass every geometric test; their surroundings are ink, a checkbox's are paper. |

Sizes are absolute pixels rather than fractions of image width because sample 2 is a crop of a page; width-relative sizing would misjudge its scale. The defaults cover roughly 100-300 DPI letter forms. Rejected alternative: a fixed interior aspect test, which fails on sample 1 where boxes share thick top and bottom rules and the visible interior is 53x42.

Effect on the samples, boxes reported before and after the filters beyond size and aspect: sample 1 341 to 119, sample 2 62 to 41, sample 3 516 to 48, sample 4 164 to 77. Visual inspection of the overlays found no remaining glyph or sidebar false positives; the annotations in D11 later showed four missed boxes, addressed in D12.

## D5. Classification: interior ink fraction

Options: ink fraction of the interior in the binary image; mean darkness of the grayscale interior; explicit stroke or diagonal detection.

Chosen: ink fraction of the interior after trimming 12% from each edge, measured on the binary image from D2 rather than on the ruling mask, with `FillThreshold` 0.04. Marks of any shape count, which matches the labeling policy (X, tick, slash, and fill are all "checked"). Grayscale darkness is sensitive to shading; diagonal detection over-fits X marks.

Observed separation on the samples: every unchecked box scored 0.000 and the lowest checked box scored 0.111, so the threshold has a wide margin on this data. The hatched box in sample 2 (Electricity, Public) has no clean rectangular hole and is not detected; the annotations label it `unchecked, ambiguous`.

## D6. Coordinates

Boxes use original-image pixels with the origin at the top-left; `x2` and `y2` are exclusive, matching Go's `image.Rectangle` and OpenCV's `Rect`, so width is `x2-x1`. Boxes are clamped to the image, duplicates at IoU >= 0.7 are removed keeping the smaller (innermost) box, and results are sorted top-to-bottom then left-to-right so responses are deterministic.

The reported box is the interior hole expanded by the ink thickness measured outward on each side (capped at 8 px). When a box shares an edge with a thicker table rule, the reported edge therefore includes the rule and extends a few pixels beyond the box's own stroke. Using the thinnest side as a uniform stroke was tried and rejected: sample 1's boxes then fail the aspect test because their visible interior is wider than tall. The over-extension is well inside an IoU 0.5 match.

## D7. Resolution and limits

Processing runs at native resolution. The largest sample is 10.7 MP and takes about 70 ms single-threaded (330 ms each with 12 concurrent requests on the development machine). Downscaling would speed this up but erases the 1-2 px strokes of sample 1's X marks. Memory is bounded by `MaxPixels` (25 MP, checked from the image header before decoding) and the 20 MiB body limit.

## D8. Validation and concurrency in the API

Format and dimensions are read with the standard library's `image.DecodeConfig` before OpenCV touches the bytes: PNG and JPEG are accepted (415 otherwise), oversized dimensions return 413, and a header that decodes but a body that does not returns 400. Only validated uploads compete for a detection slot.

OpenCV calls cannot be interrupted, so the handler bounds concurrency with a semaphore of `MaxConcurrent` slots (default `GOMAXPROCS`, flag `-max-concurrent`). A request waits up to 5 s for a slot and then receives 503 with `Retry-After`. The context is checked between pipeline stages so a disconnected client stops work at the next stage.

## D9. Error shape and `?debug=1`

Errors are `{"error": "message"}` with 400, 413, 415, 503, or 500; internal error details are logged, not returned. The success body is exactly `{"boxes":[{"bbox":[x1,y1,x2,y2],"is_checked":bool}]}` and `{"boxes":[]}` when nothing is found. With `?debug=1` (or `true`) each box gains a `debug` object with `fill_ratio`, `ink_pixels`, `interior_area`, and `border_px`; it is a pointer with `omitempty` so the default shape never carries the key.

The frontend's `ApiService.readError` read a `message` field, so it was changed to read `error`; that is the only frontend change in this step.

## D10. Known limitations recorded as `TODO(prod)`

- Boxes filled solid, or with dense horizontal/vertical hatching, have no rectangular hole and are missed (`TestDetectMissesSolidFill` documents this). No sample contains one.
- Hand-drawn marks beside a box rather than inside it, such as the quadrilateral next to "Water, Other" in sample 2, are not checkboxes and are ignored.
- Skewed or rotated scans reduce the straight-run mask; the supported skew range has not been measured yet.
- Thresholds were set by inspecting the four samples and have not been evaluated on held-out documents.

## D11. Ground truth

Options for defining the correct result of an image: compare against a stored copy of the detector's own output (a regression check, not a measure of accuracy); annotate every box by hand in an external tool; or correct a detector draft in a purpose-built editor.

Chosen: the third. The correct result is a person's judgment under the mark classification policy in `docs/plan.md`, stored as `<image>.truth.json` in the `/detect` shape plus an `ambiguous` flag. The frontend's `/annotate` page seeds the file from the detector and the annotator deletes false positives, flips states, and draws missed boxes. Annotations exist for the four challenge samples and the four pages under `testdata/additional`.

Cost and caveats: a draft biases the annotator toward the current detector. Boxes the detector misses are absent from the draft and must be looked for deliberately, and a false positive in the draft can survive review. Ambiguous boxes must be found but accept either state. The four challenge samples tuned the thresholds, so agreement on them is a regression signal; the pages under `testdata/additional` were the held-out set. The numbers in D12 came from a scoring command (one-to-one matching at IoU 0.5) that was later removed to keep the submission focused; it remains in the Git history, and the editor and annotations stay.

## D12. Changes driven by the annotations

Baseline against the annotations: the four challenge samples scored recall 0.986 (285 of 289) at precision 1.000, but the four held-out REALVALS pages scored recall 0.809 and precision 0.941. Each change below was kept only if it did not lower any challenge sample; where the two sets pulled in different directions, the challenge samples won.

| Finding | Cause | Change |
| --- | --- | --- |
| 29 of 30 held-out misses were checked boxes with a bold X. | The border is a faint 1 px line; the X darkens the local mean of a 31 px block, border pixels beside it drop out of the binary image, the split border's short half fails the 12 px opening, and the hole leaks into the table cell. | `AdaptiveBlockSize` 31 to 71, and `straightRuns` regrows each surviving run along its own direction over ink (`LineGapBridge` 1). Closing the ink before the opening was tried first and rejected: it lets diagonal X strokes through and cost sample 2 five boxes. |
| Sample 4 missed the "did / did not" pair. | Both sit directly under a heavy rule; its thickness was measured as the box's top border, giving 30x38 and failing the aspect test. | When the measured box fails the size and aspect test, retry with each side capped at the median thickness. Boxes that already passed are untouched, so sample 1's boxes that legitimately share rules (D6) keep their coordinates. |
| Block 71 turned sample 4's red watermark into ink, enclosing two false boxes. | Luminance grayscale renders saturated red as mid-gray. | Binarize the per-pixel maximum of B, G, and R. Black print stays dark; colored ink and tinted cells read as paper, which is also what the mark policy asks for. |
| Eight held-out false positives on capital D, O, Q in headings. | Their bowls are 20-21 px wide, just above the old 20 px floor. | `MinBoxSide` 20 to 22. |

Result: challenge samples recall 0.993 (287 of 289), held-out recall 0.941 (144 of 153), precision 1.000 on all eight images, state accuracy 1.000. Four letters that the annotator had accepted from the detector draft were removed from the held-out annotations during this work, which is the draft bias D11 warns about.

Remaining misses. Sample 2: the faint "Neighborhood Boundaries" box (border about 30 gray levels from paper, below `AdaptiveC`) and the hatched ambiguous box. Held-out: nine bold-X boxes whose border dropouts exceed what regrowth recovers. `AdaptiveC` 10 recovers five of them at the cost of one false positive; it was not adopted because the block size had already been chosen with these pages in view, so they are no longer a clean held-out set for further tuning.
