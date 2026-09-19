import { useEffect, useRef, useState, type PointerEvent } from "react";
import {
  boxFromDrag,
  fitScale,
  type AnnotatedBox,
  type ImageSize,
  type Point,
} from "../annotation";

export const CHECKED_COLOR = "#16a34a";
export const UNCHECKED_COLOR = "#dc2626";
const SELECTED_COLOR = "#4f46e5";

// Breathing room between the image and the stage border, in CSS pixels.
const PADDING = 16;

type ImageStageProps = {
  file: File | null;
  preview: string;
  size: ImageSize | null;
  boxes: AnnotatedBox[] | null;
  selected: number | null;
  zoom: number;
  annotating: boolean;
  onLoad: (size: ImageSize) => void;
  onError: () => void;
  onSelect: (index: number | null) => void;
  onAdd: (bbox: AnnotatedBox["bbox"]) => void;
  onSelectFiles: (files: File[]) => void;
};

export default function ImageStage({
  file,
  preview,
  size,
  boxes,
  selected,
  zoom,
  annotating,
  onLoad,
  onError,
  onSelect,
  onAdd,
  onSelectFiles,
}: ImageStageProps) {
  const stage = useRef<HTMLDivElement>(null);
  const svg = useRef<SVGSVGElement>(null);
  const input = useRef<HTMLInputElement>(null);
  const [viewport, setViewport] = useState<ImageSize | null>(null);
  const [hovering, setHovering] = useState(false);
  const [drag, setDrag] = useState<{ start: Point; end: Point } | null>(null);

  // The image is laid out in pixels rather than percentages so that zoom is a
  // multiple of the fitted size and the overlay always covers exactly the image.
  useEffect(() => {
    const node = stage.current;
    if (!node) return;
    const observer = new ResizeObserver(([entry]) =>
      setViewport({
        width: entry.contentRect.width,
        height: entry.contentRect.height,
      }),
    );
    observer.observe(node);
    return () => observer.disconnect();
  }, []);

  const scale =
    size && viewport ? fitScale(size, viewport, zoom, PADDING) : null;

  function toImagePoint(event: PointerEvent): Point {
    const rect = svg.current!.getBoundingClientRect();
    return {
      x: ((event.clientX - rect.left) * size!.width) / rect.width,
      y: ((event.clientY - rect.top) * size!.height) / rect.height,
    };
  }

  function finishDrag() {
    if (drag && size) {
      const bbox = boxFromDrag(drag.start, drag.end, size);
      if (bbox) onAdd(bbox);
    }
    setDrag(null);
  }

  const draft = drag && {
    x: Math.min(drag.start.x, drag.end.x),
    y: Math.min(drag.start.y, drag.end.y),
    width: Math.abs(drag.end.x - drag.start.x),
    height: Math.abs(drag.end.y - drag.start.y),
  };

  return (
    <div
      ref={stage}
      className="relative flex min-h-0 flex-1 overflow-auto rounded-xl border border-[#e4e4f0] bg-[#f8f9fc] bg-[image:radial-gradient(#c7d2fe80_0.8px,transparent_0.8px)] bg-size-[16px_16px]"
      onDragOver={(event) => {
        event.preventDefault();
        setHovering(true);
      }}
      onDragLeave={(event) => {
        if (!event.currentTarget.contains(event.relatedTarget as Node))
          setHovering(false);
      }}
      onDrop={(event) => {
        event.preventDefault();
        setHovering(false);
        onSelectFiles(Array.from(event.dataTransfer.files));
      }}
    >
      <input
        ref={input}
        className="sr-only"
        type="file"
        accept="image/png,image/jpeg"
        aria-label="Choose document image"
        onChange={(event) => {
          const files = Array.from(event.target.files ?? []);
          event.target.value = "";
          if (files.length) onSelectFiles(files);
        }}
      />
      {preview ? (
        <div
          className="m-auto flex shrink-0 items-center justify-center"
          style={{ padding: PADDING }}
        >
          <div
            className="relative"
            style={
              scale && size
                ? { width: size.width * scale, height: size.height * scale }
                : undefined
            }
          >
            <img
              className={`block select-none ${scale ? "h-full w-full" : "max-h-full max-w-full"}`}
              src={preview}
              alt={`Document ${file?.name}`}
              draggable={false}
              onLoad={(event) =>
                onLoad({
                  width: event.currentTarget.naturalWidth,
                  height: event.currentTarget.naturalHeight,
                })
              }
              onError={onError}
            />
            {size && boxes && (
              <svg
                ref={svg}
                className={`absolute inset-0 h-full w-full ${annotating ? "cursor-crosshair touch-none" : "pointer-events-none"}`}
                viewBox={`0 0 ${size.width} ${size.height}`}
                role="img"
                aria-label={
                  annotating
                    ? "Annotations; drag to add a box, click a box to select it"
                    : "Detected checkboxes drawn over the document"
                }
                onPointerDown={(event) => {
                  if (!annotating || event.button !== 0) return;
                  event.currentTarget.setPointerCapture(event.pointerId);
                  const point = toImagePoint(event);
                  onSelect(null);
                  setDrag({ start: point, end: point });
                }}
                onPointerMove={(event) => {
                  if (drag)
                    setDrag({ start: drag.start, end: toImagePoint(event) });
                }}
                onPointerUp={finishDrag}
                onPointerCancel={() => setDrag(null)}
              >
                {boxes.map(
                  ({ bbox: [x1, y1, x2, y2], is_checked, ambiguous }, index) => {
                    const color = is_checked ? CHECKED_COLOR : UNCHECKED_COLOR;
                    const isSelected = annotating && index === selected;
                    return (
                      <g key={index}>
                        <rect
                          className={annotating ? "cursor-pointer" : undefined}
                          x={x1}
                          y={y1}
                          width={x2 - x1}
                          height={y2 - y1}
                          fill={color}
                          fillOpacity={isSelected ? 0.35 : 0.15}
                          stroke={isSelected ? SELECTED_COLOR : color}
                          strokeWidth={isSelected ? 3 : 2}
                          strokeDasharray={is_checked ? undefined : "4 3"}
                          vectorEffect="non-scaling-stroke"
                          pointerEvents="all"
                          onPointerDown={(event) => {
                            if (!annotating) return;
                            event.stopPropagation();
                            onSelect(index);
                          }}
                        >
                          <title>
                            {`${is_checked ? "Checked" : "Unchecked"}${ambiguous ? ", ambiguous" : ""} [${x1}, ${y1}, ${x2}, ${y2}]`}
                          </title>
                        </rect>
                        {ambiguous && (
                          <text
                            className="pointer-events-none select-none"
                            x={x2 + 4}
                            y={y2}
                            fontSize={(y2 - y1) * 0.9}
                            fontWeight={700}
                            fill={color}
                          >
                            ?
                          </text>
                        )}
                      </g>
                    );
                  },
                )}
                {draft && (
                  <rect
                    {...draft}
                    fill="none"
                    stroke={SELECTED_COLOR}
                    strokeWidth={2}
                    vectorEffect="non-scaling-stroke"
                  />
                )}
              </svg>
            )}
          </div>
        </div>
      ) : (
        <button
          type="button"
          className="m-4 flex flex-1 flex-col items-center justify-center gap-2.5 rounded-lg border border-dashed border-[#c7d2fe] bg-[#fafaff]/70 px-6 text-[#52525b] hover:border-[#6366f1] hover:bg-[#eef2ff]"
          onClick={() => input.current?.click()}
        >
          <span
            className="mb-2 grid size-11.75 place-items-center rounded-xl border border-[#e0e7ff] bg-[#eef2ff] text-[27px] text-[#4f46e5]"
            aria-hidden="true"
          >
            ↑
          </span>
          <strong className="text-[15px] font-semibold">
            Drop your document here
          </strong>
          <span className="text-[13px]">
            or{" "}
            <span className="text-[#4f46e5] underline underline-offset-[3px]">
              browse files
            </span>
          </span>
          <small className="mt-3 text-[11px] text-[#71717a]">
            PNG or JPEG · up to 10 MB · one image
          </small>
        </button>
      )}
      {hovering && (
        <div
          className="pointer-events-none absolute inset-0 grid place-items-center rounded-xl border-2 border-dashed border-[#6366f1] bg-[#eef2ff]/85 text-sm font-semibold text-[#4f46e5]"
          aria-hidden="true"
        >
          Drop the image to open it
        </div>
      )}
    </div>
  );
}
