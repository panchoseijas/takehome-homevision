import type { DetectedBox } from "../detection";

export const CHECKED_COLOR = "#16a34a";
export const UNCHECKED_COLOR = "#dc2626";

export type ImageSize = { width: number; height: number };

type BoxOverlayProps = {
  boxes: DetectedBox[];
  size: ImageSize;
  interactive?: boolean;
};

// The viewBox is the image's pixel grid, and the default preserveAspectRatio
// (xMidYMid meet) matches a centered `object-contain` image, so bbox pixel
// coordinates line up with the picture at any display size.
export default function BoxOverlay({
  boxes,
  size,
  interactive = false,
}: BoxOverlayProps) {
  return (
    <svg
      className="pointer-events-none absolute inset-0 h-full w-full"
      viewBox={`0 0 ${size.width} ${size.height}`}
      aria-hidden="true"
    >
      {boxes.map(({ bbox: [x1, y1, x2, y2], is_checked }, index) => {
        const color = is_checked ? CHECKED_COLOR : UNCHECKED_COLOR;
        return (
          <rect
            key={index}
            x={x1}
            y={y1}
            width={x2 - x1}
            height={y2 - y1}
            fill={color}
            fillOpacity={0.15}
            stroke={color}
            strokeWidth={2}
            strokeDasharray={is_checked ? undefined : "4 3"}
            vectorEffect="non-scaling-stroke"
            pointerEvents={interactive ? "all" : undefined}
          >
            {interactive && (
              <title>
                {`${is_checked ? "Checked" : "Unchecked"} [${x1}, ${y1}, ${x2}, ${y2}]`}
              </title>
            )}
          </rect>
        );
      })}
    </svg>
  );
}
