import { test } from "node:test";
import assert from "node:assert/strict";
import { parseBoxes } from "../src/detection.ts";

test("reads boxes from a detection response", () => {
  const boxes = [
    { bbox: [10, 20, 30, 40], is_checked: true },
    { bbox: [50, 60, 70, 80], is_checked: false },
  ];
  assert.deepEqual(parseBoxes({ boxes }), boxes);
  assert.deepEqual(parseBoxes({ boxes: [] }), []);
});

test("ignores responses and entries that do not match the contract", () => {
  assert.deepEqual(parseBoxes(null), []);
  assert.deepEqual(parseBoxes({ received: "a.png", bytes: 3 }), []);
  assert.deepEqual(parseBoxes({ boxes: "none" }), []);

  const valid = { bbox: [1, 2, 3, 4], is_checked: true };
  assert.deepEqual(
    parseBoxes({
      boxes: [
        valid,
        null,
        { bbox: [1, 2, 3], is_checked: true },
        { bbox: [1, 2, 3, "4"], is_checked: true },
        { bbox: [1, 2, 3, 4], is_checked: "yes" },
        { bbox: [5, 5, 5, 9], is_checked: false },
      ],
    }),
    [valid],
  );
});
