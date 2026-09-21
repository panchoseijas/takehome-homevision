# Approach and tradeoffs

I built this as a single-document workflow: upload a page, detect its checkboxes, and inspect or correct the result. The architecture has two parts: a React interface for visual review and a stateless Python API for image processing.

## Architectural choices

- **Classical computer vision with OpenCV.** The supplied forms have enough structure to detect box borders and classify the marks inside them without training a model. This makes the pipeline deterministic and inspectable, with no model-serving infrastructure. The tradeoff is sensitivity to scan quality and box geometry.
- **Python for both the API and detection.** `opencv-python-headless` ships prebuilt wheels, so setup is `uv sync` with no native toolchain. An earlier Go/GoCV version, preserved on the [`go-backend`](https://github.com/panchoseijas/takehome-homevision/tree/go-backend) branch, needed a source-built OpenCV 4 on every machine; the port returns identical boxes on all four samples. The tradeoff is the GIL: OpenCV releases it inside native calls, which dominate the runtime, so detections still run in parallel on a thread pool.
- **A detector independent of HTTP.** The API handles uploads, validation, resource limits, and response formatting; the vision package owns detection and is shared by the server, CLI, and tests. That keeps the algorithm testable without a running service and allows it to evolve behind the same bounding-box contract.
- **Synchronous processing without persistence.** One request processes one image and returns the result. This keeps deployment simple and avoids introducing a database or background-job system for the current workflow. Upload and pixel limits constrain per-request resource use; capping concurrent detections is a marked production follow-up. The cost is that clients must wait for processing, and there is no durable job history or recovery.

## Validation

The sample regression tests match detections against manually reviewed annotations. Synthetic tests cover individual detector behaviors. See the [main README](../README.md#known-limitations) for measured results and known limitations.
