import { truthFileName } from "../annotation";
import ToolButton, { FilePicker } from "./ToolButton";

const ZOOM_LEVELS = [
  { value: 1, label: "Fit" },
  { value: 2, label: "2×" },
  { value: 4, label: "4×" },
];

type ToolbarProps = {
  file: File | null;
  annotating: boolean;
  isDetecting: boolean;
  canSave: boolean;
  zoom: number;
  expanded: boolean;
  onSelectFiles: (files: File[]) => void;
  onDetect: () => void;
  onToggleAnnotating: () => void;
  onZoom: (zoom: number) => void;
  onSave: () => void;
  onToggleExpanded: () => void;
};

export default function Toolbar({
  file,
  annotating,
  isDetecting,
  canSave,
  zoom,
  expanded,
  onSelectFiles,
  onDetect,
  onToggleAnnotating,
  onZoom,
  onSave,
  onToggleExpanded,
}: ToolbarProps) {
  return (
    <div className="flex flex-wrap items-center gap-2">
      <FilePicker
        label={file ? "Change image" : "Choose image"}
        accept="image/png,image/jpeg"
        onPick={onSelectFiles}
      />
      <ToolButton variant="primary" disabled={!file} onClick={onDetect}>
        {isDetecting ? "Detecting…" : "Detect checkboxes"}
      </ToolButton>
      {annotating && (
        <ToolButton disabled={!canSave} onClick={onSave}>
          Save {file ? truthFileName(file.name) : "annotations"}
        </ToolButton>
      )}
      <div className="ml-auto flex flex-wrap items-center gap-2">
        <div
          className="flex items-center gap-1"
          role="group"
          aria-label="Zoom"
        >
          {ZOOM_LEVELS.map(({ value, label }) => (
            <ToolButton
              key={value}
              variant={value === zoom ? "active" : "plain"}
              aria-pressed={value === zoom}
              disabled={!file}
              onClick={() => onZoom(value)}
            >
              {label}
            </ToolButton>
          ))}
        </div>
        <ToolButton
          aria-pressed={expanded}
          title={
            expanded
              ? "Restore the page layout (Esc)"
              : "Fill the window with the workspace"
          }
          onClick={onToggleExpanded}
        >
          {expanded ? "Restore" : "Expand"}
        </ToolButton>
        <ToolButton
          variant={annotating ? "active" : "plain"}
          aria-pressed={annotating}
          onClick={onToggleAnnotating}
        >
          {annotating ? "Annotating" : "Annotate"}
        </ToolButton>
      </div>
    </div>
  );
}
