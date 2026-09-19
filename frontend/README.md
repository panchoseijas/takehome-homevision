# HomeVision frontend

A React + TypeScript image uploader styled with Tailwind CSS and using TanStack Query mutations for request state. Select or drop one PNG/JPEG, preview it, then upload it. Includes loading, success, retryable error states, and a JSON response viewer. After upload, the detected checkboxes are drawn over the image.

The visual design follows [HomeVision’s landing page](https://homevision.co/): Inter typography, indigo actions, neutral surfaces, and subtle grid details. The HomeVision logo and fonts are served locally; the Inter license is included in `public/fonts/LICENSE.txt`.

## Run

Requires Node.js 22.18+ (or a newer supported release) and npm.

```sh
cd frontend
npm ci
npm run dev
```

Open the local URL printed by Vite (normally http://localhost:5173). Selection and preview work without a backend; upload reports an error until a server is available.

## Detection overlay

After a successful upload, every box from the `POST /detect` response is drawn over the image, in the small preview and in the full-screen viewer. Checked boxes have a solid green outline and unchecked boxes a dashed red one, so the state does not depend on color alone; a legend shows the count of each. In the full-screen viewer, hovering a box shows its state and `bbox`.

The overlay is an SVG whose `viewBox` is the image's pixel grid, so `bbox` coordinates are used as-is and stay aligned at any display size or zoom level. `src/detection.ts` types the `{ boxes: [{ bbox: [x1, y1, x2, y2], is_checked }] }` contract; the raw JSON remains visible under "Endpoint response".

## API service

`src/services/api.service.ts` provides the generic `ApiService` class with `get`, `post`, `put`, and `delete` methods. `src/services/image.service.ts` owns the image-specific multipart request, so the UI only calls `imageService.uploadImage(file)`. TanStack Query manages request state, and `src/upload.ts` contains local file validation.

## Endpoint contract

- `POST /detect`, multipart form data with one file named `image`.
- Expects a JSON success response (for example `{"boxes":[]}`), or HTTP 204 without a body.
- Non-2xx responses display the HTTP status. Network errors and malformed responses display errors.
- The frontend accepts PNG/JPEG up to 10 MiB. This is a client usability limit, not server validation; the future server must validate input independently.
- Selecting another file or removing it clears the response. Requests do not automatically retry.

Vite proxies `/detect` to `http://localhost:8080` in development and preview. To change this, copy `.env.example` to `.env.local`, set `API_PROXY_TARGET`, and restart Vite. Set `VITE_API_BASE_URL` to use an absolute API origin. An absolute URL requires the server to allow the frontend origin through CORS. Production hosting must route `/detect` to the API, or set the API base URL at build time.

## Verify

```sh
npm test
npm run lint
npm run build
npm run preview
```

Tests cover validation, multipart request contents, JSON responses, and failure handling using mocked fetch; they require no server. To manually check the UI, choose an image from `../backend/testdata`, verify its preview, and upload. With no server running, verify a clear error and retry; remove or replace the image to clear it. Once a backend exists, verify success and its response, then check that the drawn boxes sit on the document's checkboxes, including in the zoomed full-screen viewer. Also try a non-image via drag and drop, multiple files, and a narrow viewport.
