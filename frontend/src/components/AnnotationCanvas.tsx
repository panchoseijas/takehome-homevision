import { useRef, useState, type PointerEvent } from "react";
import { boxFromDrag, type AnnotatedBox, type Point } from "../annotation";
import { CHECKED_COLOR, UNCHECKED_COLOR, type ImageSize } from "./BoxOverlay";

const SELECTED_COLOR = "#4f46e5";

type AnnotationCanvasProps = {
  src: string;
  name: string;
  size: ImageSize | null;
  boxes: AnnotatedBox[];
  selected: number | null;
  zoom: number;
  onLoad: (size: ImageSize) => void;
  onSelect: (index: number | null) => void;
  onAdd: (bbox: AnnotatedBox["bbox"]) => void;
};

export default function AnnotationCanvas({
  src,
  name,
  size,
  boxes,
  selected,
  zoom,
  onLoad,
  onSelect,
  onAdd,
}: AnnotationCanvasProps) {
  const svg = useRef<SVGSVGElement>(null);
  const [drag, setDrag] = useState<{ start: Point; end: Point } | null>(null);

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
    <div className="h-[75vh] overflow-auto rounded-lg border border-[#e4e4f0] bg-[#f8f9fc]">
      <div className="relative" style={{ width: `${zoom * 100}%` }}>
        <img
          className="block w-full select-none"
          src={src}
          alt={name}
          draggable={false}
          onLoad={(event) =>
            onLoad({
              width: event.currentTarget.naturalWidth,
              height: event.currentTarget.naturalHeight,
            })
          }
        />
        {size && (
          <svg
            ref={svg}
            className="absolute inset-0 h-full w-full cursor-crosshair touch-none"
            viewBox={`0 0 ${size.width} ${size.height}`}
            role="img"
            aria-label="Annotations; drag to add a box, click a box to select it"
            onPointerDown={(event) => {
              if (event.button !== 0) return;
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
                const isSelected = index === selected;
                return (
                  <g key={index}>
                    <rect
                      className="cursor-pointer"
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
                      onPointerDown={(event) => {
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
  );
}
