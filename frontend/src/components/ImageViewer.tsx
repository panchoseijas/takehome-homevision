import { useEffect, useRef, useState } from "react";

type ImageViewerProps = {
  src: string;
  name: string;
  onClose: () => void;
};

export default function ImageViewer({ src, name, onClose }: ImageViewerProps) {
  const dialog = useRef<HTMLDialogElement>(null);
  const [zoomed, setZoomed] = useState(false);

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
            onClick={() => setZoomed(!zoomed)}
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
          className="min-h-0 flex-1 overflow-auto p-4"
          tabIndex={0}
          aria-label="Image; scroll to explore when zoomed"
        >
          <img
            className={
              zoomed
                ? "h-[200%] w-[200%] max-w-none object-contain"
                : "h-full w-full object-contain"
            }
            src={src}
            alt={name}
          />
        </div>
      </div>
    </dialog>
  );
}
