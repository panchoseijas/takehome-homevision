# Approach and tradeoffs

I built this as a single-document workflow: upload a page, detect its checkboxes, and inspect or correct the result. The architecture has two parts: a React interface for visual review and a stateless Go API for image processing.

## Architectural choices

- **Classical computer vision with GoCV/OpenCV.** The supplied forms have enough structure to detect box borders and classify the marks inside them without training a model. This makes the pipeline deterministic and inspectable, with no model-serving infrastructure. The tradeoff is sensitivity to scan quality and box geometry. OpenCV also adds a native build dependency; Docker Compose provides a reproducible way to run both services.
- **Go for both the API and detection.** I chose Go for its standard-library HTTP server, static typing, and explicit concurrency controls. I considered a separate Python service but intentionally avoided an additional network boundary because this workload does not yet justify an independent service.
- **A detector independent of HTTP.** The API handles uploads, validation, resource limits, and response formatting; the vision package owns detection and is shared by the server, CLI, and tests. That keeps the algorithm testable without a running service and allows it to evolve behind the same bounding-box contract.
- **Synchronous processing without persistence.** One request processes one image and returns the result. This keeps deployment simple and avoids introducing a database or background-job system for the current workflow. Upload and pixel limits, bounded detection concurrency, and a wait timeout constrain resource use. The cost is that clients must wait for processing, and there is no durable job history or recovery.

## Validation

The sample regression tests match detections against manually reviewed annotations. Synthetic tests cover individual detector behaviors. See the [main README](../README.md#known-limitations) for measured results and known limitations.
