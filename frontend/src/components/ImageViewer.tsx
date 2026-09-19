import { useEffect, useLayoutEffect, useRef, useState } from "react";
import type { DetectedBox } from "../detection";
import BoxOverlay, { type ImageSize } from "./BoxOverlay";

type ImageViewerProps = {
  src: string;
  name: string;
  boxes: DetectedBox[];
  size: ImageSize | null;
  onClose: () => void;
};

export default function ImageViewer({
  src,
  name,
  boxes,
  size,
  onClose,
}: ImageViewerProps) {
  const dialog = useRef<HTMLDialogElement>(null);
  const viewport = useRef<HTMLDivElement>(null);
  const content = useRef<HTMLDivElement>(null);
  const focus = useRef({ x: 0.5, y: 0.5 });
  const [zoomed, setZoomed] = useState(false);

  useLayoutEffect(() => {
    const view = viewport.current;
    const inner = content.current;
    if (!view || !inner) return;
    const viewRect = view.getBoundingClientRect();
    const innerRect = inner.getBoundingClientRect();
    view.scrollLeft +=
      innerRect.left +
      focus.current.x * innerRect.width -
      (viewRect.left + view.clientWidth / 2);
    view.scrollTop +=
      innerRect.top +
      focus.current.y * innerRect.height -
      (viewRect.top + view.clientHeight / 2);
  }, [zoomed]);

  function toggleZoom(x = 0.5, y = 0.5) {
    focus.current = { x, y };
    setZoomed(!zoomed);
  }

  useEffect(() => {
    const element = dialog.current;
    const previousOverflow = document.body.style.overflow;
    element?.showModal();
    document.body.style.overflow = "hidden";
    return () => {
      element?.close();
      document.body.style.overflow = previousOverflow;
    };
  }, []);

  return (
    <dialog
      ref={dialog}
      className="fixed inset-0 m-0 h-dvh max-h-none w-screen max-w-none border-0 bg-[#18181b] p-0 text-white backdrop:bg-[#18181b]"
      aria-label={`Full-screen preview of ${name}`}
      onClose={(event) => {
        if (!event.currentTarget.open) onClose();
      }}
    >
      <div className="flex h-full flex-col">
        <div className="flex shrink-0 items-center gap-4 border-b border-white/15 px-5 py-4">
          <span className="min-w-0 flex-1 truncate text-sm">{name}</span>
          <button
            type="button"
            className="shrink-0 rounded-md bg-white/10 px-4 py-2 text-sm hover:bg-white/20"
            aria-pressed={zoomed}
            onClick={() => toggleZoom()}
          >
            {zoomed ? "Fit to screen" : "Zoom in"}
          </button>
          <button
            type="button"
            className="shrink-0 rounded-md bg-white/10 px-4 py-2 text-sm hover:bg-white/20"
            onClick={() => dialog.current?.close()}
            autoFocus
          >
            Close
          </button>
        </div>
        <div
          ref={viewport}
          className="min-h-0 flex-1 overflow-auto p-4"
          tabIndex={0}
          aria-label="Image; double-click to zoom, scroll to explore when zoomed"
        >
          <div
            ref={content}
            className={
              zoomed
                ? "relative h-[200%] w-[200%] cursor-zoom-out"
                : "relative h-full w-full cursor-zoom-in"
            }
            onDoubleClick={(event) => {
              const rect = event.currentTarget.getBoundingClientRect();
              toggleZoom(
                (event.clientX - rect.left) / rect.width,
                (event.clientY - rect.top) / rect.height,
              );
            }}
          >
            <img
              className="h-full w-full object-contain"
              src={src}
              alt={name}
            />
            {size && <BoxOverlay boxes={boxes} size={size} interactive />}
          </div>
        </div>
      </div>
    </dialog>
  );
}
