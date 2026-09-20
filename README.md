# HomeVision

Detect and annotate checkboxes in document images with a React frontend and a Go/OpenCV backend.

## Run with Docker

Install Docker with Compose (Docker Desktop includes both), then run from this directory:

```sh
docker compose up --build
```

Open http://localhost:5173, choose an image from `backend/testdata`, and click **Detect checkboxes**. The API is also available at http://localhost:8080.

Compose starts two containers: `backend` builds and runs the Go server using a prebuilt OpenCV image; `frontend` runs Vite and forwards `/detect` requests to `http://backend:8080` over Compose's default network. Only Docker is needed locally. The first build downloads dependencies and compiles GoCV, so it can take several minutes; later builds reuse Docker's cache.

This is a local demo setup using Vite's development server. After editing source files, rerun the same command to rebuild. Stop with Ctrl+C, then remove the containers with:

```sh
docker compose down
```

The backend uses `linux/amd64` because the upstream ARM OpenCV image crashed during detection on Apple Silicon. Docker Desktop runs it through emulation on those Macs, which makes it slower than a native build.

See [backend/README.md](backend/README.md) and [frontend/README.md](frontend/README.md) for the API, local development, and tests.
