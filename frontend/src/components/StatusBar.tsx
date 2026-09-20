import type { AnnotatedBox } from "../annotation";
import { CHECKED_COLOR, UNCHECKED_COLOR } from "./ImageStage";
import ToolButton from "./ToolButton";

type StatusBarProps = {
  file: File | null;
  boxes: AnnotatedBox[] | null;
  annotating: boolean;
  selected: number | null;
  onToggleChecked: (index: number) => void;
  onDelete: (index: number) => void;
};

function hint(
  file: File | null,
  boxes: AnnotatedBox[] | null,
  annotating: boolean,
): string {
  if (!file) return "Choose or drop a PNG or JPEG to begin.";
  if (annotating) return "Click a box to edit it, or drag to add a missing one.";
  if (!boxes) return "Run the detector to draw the checkboxes it finds.";
  return "Hover a box for its state and coordinates, or switch to Annotate to correct the result.";
}

export default function StatusBar({
  file,
  boxes,
  annotating,
  selected,
  onToggleChecked,
  onDelete,
}: StatusBarProps) {
  const box = boxes && selected !== null ? boxes[selected] : null;
  const checked = boxes?.filter((one) => one.is_checked).length ?? 0;

  return (
    <div className="my-3 flex min-h-9 flex-wrap items-center gap-x-4 gap-y-2 text-xs text-[#52525b]">
      {annotating && box && selected !== null ? (
        <div className="flex flex-wrap items-center gap-2">
          <span className="font-mono">[{box.bbox.join(", ")}]</span>
          <ToolButton onClick={() => onToggleChecked(selected)}>
            {box.is_checked ? "Mark unchecked" : "Mark checked"} (C)
          </ToolButton>
          <ToolButton onClick={() => onDelete(selected)}>
            Delete (⌫)
          </ToolButton>
        </div>
      ) : (
        <p className="max-w-[70ch] leading-[1.6]">
          {hint(file, boxes, annotating)}
        </p>
      )}
      {boxes && (
        <p
          className="ml-auto flex flex-wrap items-center gap-x-4 gap-y-1"
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
          {boxes.length === 0 && <span>No checkboxes yet.</span>}
        </p>
      )}
    </div>
  );
}
