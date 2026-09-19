import type { DetectedBox } from "./detection.ts";

export type AnnotatedBox = DetectedBox & { ambiguous?: boolean };

export type ImageSize = { width: number; height: number };

export type Point = { x: number; y: number };

export const MIN_BOX_SIDE = 8;

export function boxFromDrag(
  start: Point,
  end: Point,
  size: ImageSize,
): AnnotatedBox["bbox"] | null {
  const clampX = (x: number) =>
    Math.round(Math.min(Math.max(x, 0), size.width));
  const clampY = (y: number) =>
    Math.round(Math.min(Math.max(y, 0), size.height));
  const x1 = clampX(Math.min(start.x, end.x));
  const y1 = clampY(Math.min(start.y, end.y));
  const x2 = clampX(Math.max(start.x, end.x));
  const y2 = clampY(Math.max(start.y, end.y));
  if (x2 - x1 < MIN_BOX_SIDE || y2 - y1 < MIN_BOX_SIDE) return null;
  return [x1, y1, x2, y2];
}

export function fitScale(
  image: ImageSize,
  viewport: ImageSize,
  zoom: number,
  padding: number,
): number {
  const width = Math.max(viewport.width - 2 * padding, 1);
  const height = Math.max(viewport.height - 2 * padding, 1);
  return zoom * Math.min(width / image.width, height / image.height);
}

export function truthFileName(imageName: string): string {
  return `${imageName.replace(/\.[^.]+$/, "")}.truth.json`;
}

// One box per line, in the API's reading order, so edits produce small diffs.
export function serializeTruth(boxes: AnnotatedBox[]): string {
  const lines = boxes
    .toSorted((a, b) => a.bbox[1] - b.bbox[1] || a.bbox[0] - b.bbox[0])
    .map(({ bbox, is_checked, ambiguous }) =>
      JSON.stringify(
        ambiguous
          ? { bbox, is_checked, ambiguous: true }
          : { bbox, is_checked },
      ),
    );
  return `{"boxes": [\n${lines.map((line) => `  ${line}`).join(",\n")}\n]}\n`;
}
