import { test } from "node:test";
import assert from "node:assert/strict";
import { ApiService, ApiValidationError } from "../src/services/api.service.ts";
import imageService from "../src/services/image.service.ts";
import { MAX_IMAGE_BYTES, validateImage } from "../src/upload.ts";

const image = new File(["image bytes"], "document.png", { type: "image/png" });

test("validates supported, empty, and oversized files", () => {
  assert.equal(validateImage(image), null);
  assert.equal(
    validateImage(new File(["jpg"], "photo.jpg", { type: "image/jpeg" })),
    null,
  );
  assert.match(
    validateImage(new File(["pdf"], "file.pdf", { type: "application/pdf" })),
    /PNG or JPEG/,
  );
  assert.match(
    validateImage(new File([], "empty.png", { type: "image/png" })),
    /empty/,
  );
  assert.match(
    validateImage({ type: "image/png", size: MAX_IMAGE_BYTES + 1 }),
    /10 MB/,
  );
});

test("posts the image as multipart form data and returns JSON", async (t) => {
  t.mock.method(globalThis, "fetch", async (url, options) => {
    assert.equal(url, "/detect");
    assert.equal(options.method, "POST");
    assert.equal(options.signal, undefined);
    assert.equal(options.body.get("image").name, "document.png");
    assert.equal(await options.body.get("image").text(), "image bytes");
    assert.equal(options.headers, undefined);
    return Response.json({ boxes: [] });
  });
  assert.deepEqual(await imageService.uploadImage(image), { boxes: [] });
});

test("handles HTTP, network, and malformed response failures", async (t) => {
  const fetchMock = t.mock.method(
    globalThis,
    "fetch",
    async () => new Response(null, { status: 413 }),
  );
  await assert.rejects(imageService.uploadImage(image), (error) => {
    assert.ok(error instanceof ApiValidationError);
    assert.equal(error.status, 413);
    assert.equal(error.message, "Request failed (HTTP 413)");
    return true;
  });
  fetchMock.mock.mockImplementation(
    async () =>
      Response.json(
        { error: "uploaded file must be a PNG or JPEG image" },
        { status: 415 },
      ),
  );
  await assert.rejects(imageService.uploadImage(image), (error) => {
    assert.ok(error instanceof ApiValidationError);
    assert.equal(error.status, 415);
    assert.equal(error.message, "uploaded file must be a PNG or JPEG image");
    return true;
  });
  fetchMock.mock.mockImplementation(async () => {
    throw new TypeError("Failed to fetch");
  });
  await assert.rejects(imageService.uploadImage(image), /Failed to fetch/);
  fetchMock.mock.mockImplementation(
    async () =>
      new Response("<html>Vite fallback</html>", {
        headers: { "content-type": "text/html" },
      }),
  );
  await assert.rejects(
    imageService.uploadImage(image),
    (error) => error instanceof SyntaxError,
  );
  fetchMock.mock.mockImplementation(
    async () =>
      new Response("{", { headers: { "content-type": "application/json" } }),
  );
  await assert.rejects(
    imageService.uploadImage(image),
    (error) => error instanceof SyntaxError,
  );
  fetchMock.mock.mockImplementation(
    async () => new Response(null, { status: 204 }),
  );
  assert.equal(await imageService.uploadImage(image), null);
});

test("supports generic GET requests", async (t) => {
  t.mock.method(globalThis, "fetch", async (url, options) => {
    assert.equal(url, "/healthz");
    assert.equal(options.method, "GET");
    return Response.json({ status: "ok" });
  });

  const api = new ApiService();
  assert.deepEqual(await api.get("/healthz"), { status: "ok" });
});
