import { useState } from "react";
import type { DetectedBox } from "../detection";
import BoxOverlay, {
  CHECKED_COLOR,
  UNCHECKED_COLOR,
  type ImageSize,
} from "./BoxOverlay";
import ImageViewer from "./ImageViewer";

type ImagePreviewProps = {
  file: File | null;
  preview: string;
  boxes: DetectedBox[] | null;
  onPreviewError: () => void;
};

export default function ImagePreview({
  file,
  preview,
  boxes,
  onPreviewError,
}: ImagePreviewProps) {
  const [expandedPreview, setExpandedPreview] = useState<string | null>(null);
  const [loaded, setLoaded] = useState<{ src: string; size: ImageSize }>();
  const size = loaded?.src === preview ? loaded.size : null;
  const visibleBoxes = boxes ?? [];
  const checked = visibleBoxes.filter((box) => box.is_checked).length;

  return (
    <section
      className="bg-[#ffffff] p-5.5 min-[761px]:p-8 min-[900px]:flex min-[900px]:flex-col"
      aria-labelledby="preview-heading"
    >
      <div className="mb-6.25 flex items-center gap-2.5">
        <span className="rounded bg-[#eef2ff] px-1.5 py-1.25 font-mono text-[11px] text-[#4f46e5]">
          02
        </span>
        <h2 className="text-sm font-[650]" id="preview-heading">
          Image preview
        </h2>
        {file && (
          <span className="ml-auto text-[9px] tracking-[1px] text-[#71717a]">
            {boxes ? "DETECTED" : "ORIGINAL"}
          </span>
        )}
      </div>
      <div
        className={`preview-canvas flex h-full min-h-70 max-h-107.5 items-center justify-center rounded-lg border border-[#e4e4f0] bg-[#f8f9fc] min-[761px]:min-h-83.75 min-[900px]:flex-1 ${preview ? "p-4" : ""}`}
      >
        {preview ? (
          <button
            type="button"
            className="relative flex max-h-full w-full cursor-zoom-in justify-center rounded-md"
            aria-label={`Open full-screen preview of ${file?.name}`}
            onClick={() => setExpandedPreview(preview)}
          >
            <img
              className="max-h-87.5 max-w-full object-contain min-[761px]:max-h-97.5"
              src={preview}
              alt={`Preview of ${file?.name}`}
              onLoad={(event) =>
                setLoaded({
                  src: preview,
                  size: {
                    width: event.currentTarget.naturalWidth,
                    height: event.currentTarget.naturalHeight,
                  },
                })
              }
              onError={onPreviewError}
            />
            {size && <BoxOverlay boxes={visibleBoxes} size={size} />}
            {visibleBoxes.length === 0 && (
              <span className="absolute right-2 bottom-2 rounded-md bg-[#18181b]/80 px-3 py-2 text-xs text-white">
                Click to expand
              </span>
            )}
          </button>
        ) : (
          <div className="text-center text-[#71717a]">
            <div
              className="mx-auto grid h-20 w-16 -rotate-6 grid-cols-[14px_1fr] items-center gap-1.5 rounded-[5px] border border-[#c7d2fe] bg-[#ffffff] px-2.5 py-3.25 text-[10px] shadow-[4px_5px_0_#e0e7ff]"
              aria-hidden="true"
            >
              <span>✓</span>
              <i className="h-0.5 bg-[#c7d2fe]" />
              <span>✓</span>
              <i className="h-0.5 bg-[#c7d2fe]" />
              <span>□</span>
              <i className="h-0.5 bg-[#c7d2fe]" />
            </div>
            <h3 className="mt-5.5 mb-2.5 text-sm font-medium text-[#3f3f46]">
              A closer look, right here.
            </h3>
            <p className="text-xs leading-[1.7]">
              Your image preview will appear
              <br />
              once you select a file.
            </p>
          </div>
        )}
      </div>
      {boxes && (
        <p
          className="mt-4 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-[#52525b]"
          aria-live="polite"
        >
          <span className="flex items-center gap-1.5">
            <i
              className="h-3 w-3 border-2"
              style={{ borderColor: CHECKED_COLOR }}
            />
            Checked ({checked})
          </span>
          <span className="flex items-center gap-1.5">
            <i
              className="h-3 w-3 border-2 border-dashed"
              style={{ borderColor: UNCHECKED_COLOR }}
            />
            Unchecked ({boxes.length - checked})
          </span>
          {boxes.length === 0 && <span>No checkboxes in the response.</span>}
        </p>
      )}
      {preview && expandedPreview === preview && (
        <ImageViewer
          src={preview}
          name={file?.name ?? "Document image"}
          boxes={visibleBoxes}
          size={size}
          onClose={() => setExpandedPreview(null)}
        />
      )}
    </section>
  );
}
