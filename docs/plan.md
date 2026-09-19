# Checkbox detection build plan

The project will use a monorepo with a React + Vite + TypeScript frontend and a Go backend. The initial detector will use classical computer vision through GoCV/OpenCV. This document describes future work; the repository currently contains the directory skeleton, the four sample images under `backend/testdata/`, and this plan.

## Requirements and scope

| Source | Requirement | Planned verification |
| --- | --- | --- |
| Challenge | Detect filled and unfilled checkboxes in an uploaded document image. | Annotated examples covering both classes; localization precision and recall. |
| Challenge | Classify each detected checkbox as filled or unfilled. | Classification accuracy on matched detections, plus end-to-end correctly localized and classified results. |
| Challenge | Expose `POST /detect` with file upload input. | HTTP integration tests and a documented curl request. |
| Challenge | Return `boxes`, each with `bbox: [x1, y1, x2, y2]` and boolean `is_checked`. Coordinates describe top-left and bottom-right corners in image pixels. | Contract tests, coordinate bounds checks, and visual overlays on the original image. |
| Challenge | Provide build/run instructions and a brief approach, tradeoffs, and limitations writeup. | Follow the README from a clean extraction of the submission. |
| Challenge | Submit project files and writeup in a zip; explain the solution in a one-hour review. | Verify archive contents and prepare a short, repeatable walkthrough. |
| Project scope | Use `frontend/` and `backend/`, React + Vite + TypeScript, Go, and classical CV through GoCV/OpenCV. Run the frontend and backend as separate processes. | Keep the API independently usable and the frontend focused on inspecting its results. |

The challenge explicitly says perfect accuracy is not expected. It specifies no accuracy target, latency target, test coverage requirement, or image format list. Those omissions should be documented rather than replaced with invented evaluation criteria.

## Current structure

```text
./
├── frontend/
│   ├── public/
│   └── src/
├── backend/
│   ├── cmd/
│   │   └── server/
│   ├── internal/
│   │   └── vision/
│   └── testdata/
│       ├── sample1-urar-page1.png
│       ├── sample2-neighborhood-site-crop.jpeg
│       ├── sample3-market-conditions-addendum.png
│       └── sample4-manufactured-home-report.png
└── docs/
    └── plan.md
```

`cmd/server` will assemble the API server and detector. `internal/vision` will own image processing independently of HTTP. Add `internal/httpapi` when implementing upload validation, handlers, and JSON responses. Keep Go tests beside the packages they exercise and sample images/annotations under `backend/testdata`.

Organize frontend code around the upload-and-inspect flow as it develops. Keep its API types and request helper in the frontend; a shared cross-language package is unnecessary for this small contract. Use one Go module in `backend` and one JavaScript package in `frontend`. Add root run commands if they simplify setup.

## Selected technology

1. Frontend: React + Vite + TypeScript.
2. Backend: Go, with GoCV/OpenCV for the initial classical computer vision implementation.
3. Reproducible setup: plan a Docker run path for the backend to package the native OpenCV dependency. Validate compatible GoCV/OpenCV versions and the container build early.
4. Frontend serving: use Vite's dev server for development and local review, with requests proxied to the separately running Go backend. Document the commands and ports for both processes. Keep frontend and backend builds independent.

GoCV provides established image-processing primitives and reduces the amount of custom algorithm code to maintain and validate. Its native OpenCV dependency adds build and packaging work; that tradeoff is accepted for the initial approach.

Suggested API assumptions to document before implementation:

- One PNG or JPEG per request, sent as `multipart/form-data` using the field `image`.
- Synchronous processing, with no upload persistence.
- Successful detection of no checkboxes returns `{"boxes": []}`.
- Integer coordinates use the original decoded image dimensions, origin at the top-left; propose exclusive right/bottom edges and document that convention. Any resize or rotation inside the detector must be mapped back to this coordinate system.
- Define and test request-byte, decoded-pixel, concurrency, and timeout limits. Return clear JSON errors for missing, malformed, unsupported, or oversized input.
- Return exactly the specified success shape by default. With the opt-in query parameter `?debug=1`, each box additionally carries a `debug` object with per-box diagnostics (for example the interior fill score and the candidate source). The default response never includes it; test both shapes.
- Expose `GET /healthz` returning `200` with `{"status":"ok"}`. Use it for the container health check and for the frontend's connection state.

PDF upload, OCR, learned models, authentication, persistence, background jobs, and cloud infrastructure are outside the initial scope. The supplied PDF is the source of test images, not a new input format requirement.

## Mark classification policy

The samples contain more than empty boxes and X marks. Fix the labeling policy before annotating in step 5 so labels and detector behavior stay consistent:

| Observed in samples | Label | Notes |
| --- | --- | --- |
| X mark (samples 1, 3, 4; the common case) | checked | Strokes are thin (1–2 px at 2550 px width) in sample 1. |
| ✓ checkmark (sample 2, Electricity → Other) | checked | Any deliberate ink stroke inside the box counts. |
| Single diagonal slash or partial stroke (sample 2, Water → Other) | checked | Same rule as above. |
| Solid or near-solid dark fill | checked | Not present in the samples; included for completeness. |
| Empty box | unchecked | |
| Box over blue or gray shaded background, no stroke (sample 3 shaded rows) | unchecked | Shading is form styling, not a mark. |
| Uniform gray or hatched interior with no distinct stroke (sample 2, Electricity → Public) | unchecked, `ambiguous` | Likely scan artifact or partially printed fill. |
| Colored watermark or signature ink crossing a box (sample 4) | ignore the colored ink | Only dark ink inside the box counts as a mark. |
| Hand-drawn loops or strokes outside any box (sample 2, over the description text) | not a checkbox | Negative example for detection. |

Annotate ambiguous cases with an `ambiguous: true` flag and report metrics both including and excluding them. If a new mark type appears in later images, extend this table rather than deciding case by case.

## Build sequence

Throughout, mark deferred production work in code with `// TODO(prod): ...` comments. Use that one prefix consistently so the TODOs can be grepped and listed in the writeup's limitations section. The challenge explicitly invites this.

The order below deliberately puts a working end-to-end app before accuracy work. A crude detector wired through the API and frontend is worth more early than a precise one with no way to run it; measurement and tuning follow once the whole flow exists.

### 1. Establish the toolchain

- Scaffold the React + Vite + TypeScript frontend and Go module only after the planning stage. Keep generated boilerplate in a separate commit from original work.
- Pin compatible dependency versions and verify the backend can build and run through the documented setup, including GoCV and its native OpenCV dependency.
- The four original embedded images from assignment pages 3–6 are already stored under `backend/testdata/` as the raw embedded bytes, without resampling. Their dimensions are 2550×4200, 1586×846, 2550×4200, and 2550×3301. Sample 2 is a lossy JPEG and must not be re-encoded. Do not substitute screenshots of PDF pages.

### 2. Build a first classical CV detector

Implement the simplest pipeline that returns plausible boxes on the samples, without tuning for accuracy yet:

1. Decode and validate the image, retain the original pixels and dimensions, and derive grayscale/binary working images.
2. Find candidate checkbox boundaries using contours, plausible dimensions, and near-square geometry. Account for differing image resolutions; do not hardcode page coordinates or document templates.
3. Classify each candidate's interior, excluding its printed border, by ink occupancy. Treat X marks and ticks as checked, rather than requiring a solid fill.
4. Map boxes back to original-image coordinates, clamp them to image bounds, and return them in a deterministic order.

Keep preprocessing, candidate detection, and classification understandable and testable within the vision package. Favor a small pipeline over speculative abstractions or many unexplained thresholds; refinement waits for step 5.

### 3. Expose the Go HTTP API

- Use Go's HTTP facilities for the single required endpoint and keep transport code separate from detection.
- Implement the documented upload contract, decoding checks, bounded processing, errors, the exact success response, the `?debug=1` variant, and `GET /healthz`.
- Ensure native OpenCV image resources are released on all paths. Avoid shared mutable image buffers between requests.
- Exercise `POST /detect` independently with curl before connecting the frontend.

### 4. Build the inspection frontend

- Provide an image upload control and a detect action with loading, error, and empty-result states.
- Show the original image with checked/unchecked bounding-box overlays, a legend, and counts. Include zoom or another practical way to inspect the small boxes in full-page documents.
- Preserve overlay alignment while resizing the display; distinguish states through labels or styling as well as color.
- Make the exact response JSON available to inspect or download. Reset stale results when the input image changes.
- Request `?debug=1` from the inspector and show per-box diagnostics on hover or selection. Use `GET /healthz` to show backend connection state.
- Run the frontend separately through Vite and proxy `/detect` and `/healthz` requests to the Go backend for development and local review.

The frontend's purpose is to make correctness easy to review. At the end of this step the full upload → detection → overlay → JSON flow works, whatever the detector's accuracy.

### 5. Annotate, measure, and improve the detector

- Manually annotate checkbox bounds and states for the four samples following the mark classification policy above. Include ordinary table cells and text as negative examples.
- Add a reproducible evaluator with one-to-one matching at a documented intersection-over-union threshold, initially 0.5. Report localization precision/recall, classification accuracy for matched boxes, and correctly localized/classified detections. Report per-image results and runtime.
- Keep calibration and evaluation distinct where the four-image sample allows. Do not present performance on tuned examples or their synthetic variants as evidence of generalization to unseen documents.
- Compare global and adaptive thresholding on the samples. Avoid aggressive downscaling that erases small boxes or faint marks. Filter text and ordinary table cells, then merge duplicate detections from nested boundaries or multiple processing passes. Inspect failures involving faint strokes and shaded cells before adding rules; preserve enough interior area to classify small boxes reliably.
- The samples contain dense form grids, checkboxes touching table lines, blue shading, handwriting, and a colored watermark. These make simple square-contour and dark-pixel heuristics fallible. Evaluate each added rule against all examples. If table-line suppression is needed, compare candidates before and after suppression, validate their borders against the original, and preserve an unmodified image for mark classification. Consider line-based candidate recovery only if measured misses justify it.

### 6. Verify, document, and package

- Test the detector on all four annotated images and targeted synthetic cases: blank/text-only images, checked/unchecked boxes, duplicate contours, table intersections, changes in scale, shading, and noise. Probe skew to establish and document the supported range.
- Test the HTTP contract, empty detections, malformed uploads, unsupported types, request limits, decoded-image limits, coordinate mapping, the default response having no `debug` fields, the `?debug=1` variant, `GET /healthz`, and frontend-to-backend proxy routing.
- Exercise the actual upload → detection → overlay → JSON flow, including backend failure and replacement of the uploaded image.
- Run formatting, static checks, frontend type checking/linting/build, Go tests/vet, and any configured integration tests. Confirm concurrent requests do not corrupt results or exhaust native resources within the documented limits.
- Write a root README with prerequisites, exact build/run/test commands for the frontend and backend, their ports and proxy configuration, curl examples, supported inputs, and a short usage walkthrough. Record measured results, approach, dependency tradeoffs, observed limitations, and the list of `TODO(prod)` items in a concise writeup.
- Verify those instructions from a clean copy, inspect the zip contents, and exclude local dependencies, build output, credentials, and scratch files. Search for accidental chat artifacts and placeholder text before packaging.
- Keep commits organized around scaffold, detector, API, frontend, evaluation, and final verification/documentation.

Completion means the required API works, results have been measured honestly, important failures are understood, checks pass, and the project can be run and evaluated from the included instructions alone. Any detector limitations must be distinguished from software defects and described with examples.

## Sources

- Supplied *Backend Take Home Challenge Description - Checkbox Detection*, April 2026: requirements on pages 1–2, examples on pages 3–6, delivery guidance on page 7.
- [GoCV setup documentation](https://gocv.io/getting-started/) describes the required native OpenCV dependency.
- OpenCV documentation for [thresholding](https://docs.opencv.org/4.x/d7/d4d/tutorial_py_thresholding.html) and [contour extraction](https://docs.opencv.org/4.x/d4/d73/tutorial_py_contours_begin.html) supports the proposed baseline primitives; their suitability must be established through evaluation on the supplied images.
