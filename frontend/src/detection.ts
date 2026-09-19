export type DetectedBox = {
  bbox: [number, number, number, number];
  is_checked: boolean;
};

// Boxes only drive the overlay, so a malformed entry is skipped rather
// than failing the whole response: the raw JSON stays visible for inspection.
export function parseBoxes(data: unknown): DetectedBox[] {
  if (typeof data !== "object" || data === null) return [];
  const boxes = (data as { boxes?: unknown }).boxes;
  if (!Array.isArray(boxes)) return [];
  return boxes.filter(isDetectedBox);
}

function isDetectedBox(value: unknown): value is DetectedBox {
  if (typeof value !== "object" || value === null) return false;
  const { bbox, is_checked } = value as Record<string, unknown>;
  if (typeof is_checked !== "boolean") return false;
  if (!Array.isArray(bbox) || bbox.length !== 4) return false;
  if (!bbox.every((n) => typeof n === "number" && Number.isFinite(n)))
    return false;
  return bbox[2] > bbox[0] && bbox[3] > bbox[1];
}
