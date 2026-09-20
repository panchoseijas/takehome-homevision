# HomeVision frontend

A React + TypeScript workspace for `POST /detect`, styled with Tailwind CSS and using TanStack Query mutations for request state. One page holds one document: choose or drop a PNG/JPEG, run the detector, and read the boxes it returns drawn over the image. Turning on annotate mode makes those boxes editable, so the same view records the correct result for an image.

The visual design follows [HomeVision’s landing page](https://homevision.co/): Inter typography, indigo actions, neutral surfaces, and subtle grid details. The HomeVision logo and fonts are served locally; the Inter license is included in `public/fonts/LICENSE.txt`.

## Run

Requires Node.js 22.18+ (or a newer supported release) and npm.

```sh
cd frontend
npm ci
npm run dev
```

Open the local URL printed by Vite (normally http://localhost:5173). Choosing an image, zooming, and annotating all work without a backend; detection reports an error until a server is available.

## The workspace

The document is the page. A toolbar sits above a stage that holds nothing but the image:

- **Choose image** (or drag a file onto the stage) opens a PNG or JPEG; **Detect checkboxes** sends it to `POST /detect`.
- **Fit / 2× / 4×** set the zoom. `Fit` shows the whole page; the higher levels are multiples of that scale and scroll inside the stage.
- **Expand** grows the workspace to fill the window for a closer read, and `Esc` or **Restore** brings the page back.
- **Annotate** turns the drawn boxes into an editable ground-truth draft; see below.

A rejected file (not PNG/JPEG, empty, over 10 MB, or several at once) reports the reason and leaves the open document and its boxes untouched.

## Detection overlay

After a successful detection, every box from the response is drawn over the image. Checked boxes have a solid green outline and unchecked boxes a dashed red one, so the state does not depend on color alone; a legend shows the count of each, and hovering a box shows its state and `bbox`.

The overlay is an SVG whose `viewBox` is the image's pixel grid, so `bbox` coordinates are used as-is and stay aligned at any zoom level. The image is laid out at an explicit pixel size (`fitScale` in `src/annotation.ts`) rather than a percentage, which keeps the overlay exactly on the image instead of on a letterboxed box around it. `src/detection.ts` types the `{ boxes: [{ bbox: [x1, y1, x2, y2], is_checked }] }` contract; the raw JSON stays visible under "Endpoint response", unchanged by any annotation edit.

## Annotate mode

Annotate mode records the correct result for an image: every checkbox and its state, as judged by a person. Press **Annotate** and the detector's boxes become an editable draft. Then correct it:

- click a box to select it; `C` or Space toggles checked, Delete removes it, Escape deselects;
- drag on the document to add a box the detector missed; to fix a misplaced one, delete it and draw it again;
- use 2× or 4× zoom on full pages, and look for checkboxes with no rectangle on them, since a draft can never contain the detector's own misses.

"Save" downloads `<image name>.truth.json`, the `/detect` response shape; move it beside the image in `backend/testdata`, where the four samples and the pages under `additional/` already have one. A saved file cannot be reopened in the page: a draft always starts from the detector, so correcting an existing annotation means editing the JSON by hand. Leaving the page with unsaved edits asks for confirmation, as does replacing them with a fresh detection; a detection on its own is never treated as unsaved work, since it can be re-run.

Annotating is a mode of the one page rather than a separate route: the same image, zoom, and boxes stay on screen when it is turned on, so there is no second upload and nothing to re-align.

The boxes and their edits live in `src/useAnnotations.ts`, a hook that owns the detector's result, the editable draft, the keyboard shortcuts, and the saved file; `src/useExpanded.ts` owns the expanded workspace and its scroll lock. `src/App.tsx` keeps only the open image, the view state, and the request, and holds no effects of its own.

## API service

`src/services/api.service.ts` provides the generic `ApiService` class with `get`, `post`, `put`, and `delete` methods. `src/services/image.service.ts` owns the image-specific multipart request, so the UI only calls `imageService.uploadImage(file)`. TanStack Query manages request state, and `src/upload.ts` contains local file validation.

## Endpoint contract

- `POST /detect`, multipart form data with one file named `image`.
- Expects a JSON success response (for example `{"boxes":[]}`), or HTTP 204 without a body.
- Non-2xx responses display the HTTP status. Network errors and malformed responses display errors.
- The frontend accepts PNG/JPEG up to 10 MiB. This is a client usability limit, not server validation; the future server must validate input independently.
- Selecting another file clears the response. Requests do not automatically retry.

Vite proxies `/detect` to `http://localhost:8080` in development and preview. To change this, copy `.env.example` to `.env.local`, set `API_PROXY_TARGET`, and restart Vite. Set `VITE_API_BASE_URL` to use an absolute API origin. An absolute URL requires the server to allow the frontend origin through CORS. Production hosting must route `/detect` to the API, or set the API base URL at build time.

## Verify

```sh
npm test
npm run lint
npm run build
npm run preview
```

Tests cover file validation, the fit-and-zoom scale, truth-file serialization, multipart request contents, JSON responses, and failure handling using mocked fetch; they require no server.

To check the UI by hand, with the backend running on :8080:

1. Drag `../backend/testdata/sample1-urar-page1.png` onto the stage and press **Detect checkboxes**. The legend should report 38 checked and 81 unchecked, and the boxes should sit on the document's checkboxes at `Fit`, `4×`, and inside **Expand**.
2. Press **Annotate**, click a box, and toggle it with `C`; the legend follows. Drag on an empty area to add a box, then Delete to remove it.
3. Press **Save**: the browser downloads `sample1-urar-page1.truth.json` with one box per line in reading order.
4. Stop the backend and press **Detect checkboxes**: the error appears and the drawn boxes stay. Try a PDF and a multi-file drop: both are refused without disturbing the open document.
5. Narrow the window to a phone width; the toolbar wraps and nothing overflows sideways.
