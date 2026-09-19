import { test } from "node:test";
import assert from "node:assert/strict";
import {
  boxFromDrag,
  serializeTruth,
  truthFileName,
} from "../src/annotation.ts";

const size = { width: 100, height: 80 };

test("turns a drag in any direction into a clamped integer box", () => {
  assert.deepEqual(
    boxFromDrag({ x: 40.6, y: 50.2 }, { x: 10.4, y: 20.7 }, size),
    [10, 21, 41, 50],
  );
  assert.deepEqual(
    boxFromDrag({ x: 90, y: 70 }, { x: 140, y: 95 }, size),
    [90, 70, 100, 80],
  );
});

test("ignores drags too small to be a checkbox", () => {
  assert.equal(boxFromDrag({ x: 10, y: 10 }, { x: 12, y: 40 }, size), null);
});

test("names the truth file after the image", () => {
  assert.equal(truthFileName("sample2-crop.jpeg"), "sample2-crop.truth.json");
});

test("serializes boxes in reading order with ambiguous only when set", () => {
  const text = serializeTruth([
    { bbox: [50, 10, 70, 30], is_checked: true, ambiguous: false },
    { bbox: [5, 40, 25, 60], is_checked: false, ambiguous: true },
    { bbox: [10, 10, 30, 30], is_checked: false },
  ]);
  assert.deepEqual(JSON.parse(text), {
    boxes: [
      { bbox: [10, 10, 30, 30], is_checked: false },
      { bbox: [50, 10, 70, 30], is_checked: true },
      { bbox: [5, 40, 25, 60], is_checked: false, ambiguous: true },
    ],
  });
  assert.equal(text.split("\n").length, 6);
});
