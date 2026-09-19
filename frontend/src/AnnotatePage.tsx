import { useMutation } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import {
  serializeTruth,
  truthFileName,
  type AnnotatedBox,
  type TruthFile,
} from "./annotation";
import AnnotationCanvas from "./components/AnnotationCanvas";
import type { ImageSize } from "./components/BoxOverlay";
import PageLayout from "./components/PageLayout";
import imageService from "./services/image.service";
import { validateImage } from "./upload";

const ZOOM_LEVELS = [1, 2, 4];

const BUTTON =
  "rounded-md border border-[#e4e4e7] bg-white px-3 py-2 text-xs font-semibold text-[#3f3f46] enabled:hover:bg-[#eef2ff] disabled:text-[#a1a1aa]";
const PRIMARY_BUTTON =
  "rounded-md border-0 bg-[#4f46e5] px-3 py-2 text-xs font-semibold text-white enabled:hover:bg-[#4338ca] disabled:bg-[#e4e4f0] disabled:text-[#626279]";

export default function AnnotatePage() {
  const [file, setFile] = useState<File | null>(null);
  const [preview, setPreview] = useState("");
  const [size, setSize] = useState<ImageSize | null>(null);
  const [boxes, setBoxes] = useState<AnnotatedBox[]>([]);
  const [selected, setSelected] = useState<number | null>(null);
  const [zoom, setZoom] = useState(1);
  const [dirty, setDirty] = useState(false);
  const [message, setMessage] = useState("");

  const detection = useMutation({
    mutationFn: imageService.uploadImage,
    retry: false,
    networkMode: "always",
    onSuccess: (data) => replaceBoxes(data?.boxes ?? []),
  });
  const error = message || detection.error?.message;

  useEffect(() => {
    return () => {
      if (preview) URL.revokeObjectURL(preview);
    };
  }, [preview]);

  useEffect(() => {
    if (!dirty) return;
    const warn = (event: BeforeUnloadEvent) => event.preventDefault();
    window.addEventListener("beforeunload", warn);
    return () => window.removeEventListener("beforeunload", warn);
  }, [dirty]);

  useEffect(() => {
    if (selected === null) return;
    const index = selected;
    function onKeyDown(event: KeyboardEvent) {
      if (event.target instanceof HTMLInputElement) return;
      if (event.key === "c" || event.key === " ") {
        updateBox(index, (box) => ({ ...box, is_checked: !box.is_checked }));
      } else if (event.key === "a") {
        updateBox(index, (box) => ({ ...box, ambiguous: !box.ambiguous }));
      } else if (event.key === "Delete" || event.key === "Backspace") {
        deleteBox(index);
      } else if (event.key === "Escape") {
        setSelected(null);
      } else {
        return;
      }
      event.preventDefault();
    }
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [selected]);

  function confirmDiscard() {
    return !dirty || window.confirm("Discard the unsaved annotations?");
  }

  function replaceBoxes(next: AnnotatedBox[]) {
    setBoxes(next);
    setSelected(null);
    setDirty(next.length > 0);
  }

  function updateBox(
    index: number,
    change: (box: AnnotatedBox) => AnnotatedBox,
  ) {
    setBoxes((current) =>
      current.map((box, i) => (i === index ? change(box) : box)),
    );
    setDirty(true);
  }

  function deleteBox(index: number) {
    setBoxes((current) => current.filter((_, i) => i !== index));
    setSelected(null);
    setDirty(true);
  }

  function selectImage(next: File | undefined) {
    if (!next || !confirmDiscard()) return;
    detection.reset();
    const invalid = validateImage(next);
    setMessage(invalid ?? "");
    if (invalid) return;
    setFile(next);
    setPreview(URL.createObjectURL(next));
    setSize(null);
    setZoom(1);
    replaceBoxes([]);
  }

  async function loadAnnotations(next: File | undefined) {
    if (!next || !confirmDiscard()) return;
    try {
      const truth = JSON.parse(await next.text()) as TruthFile;
      replaceBoxes(truth.boxes);
      setDirty(false);
      setMessage("");
    } catch {
      setMessage(`${next.name} is not a valid annotation file.`);
    }
  }

  function download() {
    const url = URL.createObjectURL(
      new Blob([serializeTruth(boxes)], { type: "application/json" }),
    );
    const link = document.createElement("a");
    link.href = url;
    link.download = truthFileName(file!.name);
    link.click();
    URL.revokeObjectURL(url);
    setDirty(false);
  }

  const current = selected === null ? null : boxes[selected];
  const checked = boxes.filter((box) => box.is_checked).length;
  const ambiguous = boxes.filter((box) => box.ambiguous).length;

  return (
    <PageLayout>
      <div className="mx-auto max-w-[740px] text-center">
        <h1 className="text-[22px] font-bold tracking-[4px] text-[#4f46e5] min-[761px]:text-[30px] min-[761px]:tracking-[6px]">
          ANNOTATION EDITOR
        </h1>
        <p className="mt-4 text-xs leading-[1.7] text-[#52525b]">
          Build the ground truth used by <code>go run ./cmd/eval</code>. Start
          from the detector's draft, fix it, and save the file beside the image
          in <code>backend/testdata</code>.
        </p>
      </div>
      <section className="mt-9 rounded-[20px] border border-[#e4e4e7] bg-white p-5.5 shadow-[0_6px_24px_#18181b08] min-[761px]:mt-12 min-[761px]:p-8">
        <div className="flex flex-wrap items-center gap-2">
          <label className={`${BUTTON} cursor-pointer`}>
            {file ? "Change image" : "Choose image"}
            <input
              className="sr-only"
              type="file"
              accept="image/png,image/jpeg"
              onChange={(event) => {
                selectImage(event.target.files?.[0]);
                event.target.value = "";
              }}
            />
          </label>
          {file && (
            <>
              <button
                type="button"
                className={BUTTON}
                disabled={detection.isPending}
                onClick={() => {
                  if (confirmDiscard()) detection.mutate(file);
                }}
              >
                {detection.isPending ? "Detecting…" : "Draft from detector"}
              </button>
              <label className={`${BUTTON} cursor-pointer`}>
                Load annotations
                <input
                  className="sr-only"
                  type="file"
                  accept="application/json,.json"
                  onChange={(event) => {
                    void loadAnnotations(event.target.files?.[0]);
                    event.target.value = "";
                  }}
                />
              </label>
              <span className="ml-auto flex items-center gap-1">
                {ZOOM_LEVELS.map((level) => (
                  <button
                    key={level}
                    type="button"
                    className={level === zoom ? PRIMARY_BUTTON : BUTTON}
                    aria-pressed={level === zoom}
                    onClick={() => setZoom(level)}
                  >
                    {level}×
                  </button>
                ))}
              </span>
              <button
                type="button"
                className={PRIMARY_BUTTON}
                disabled={!dirty}
                onClick={download}
              >
                Save {truthFileName(file.name)}
              </button>
            </>
          )}
        </div>
        {error && (
          <p
            className="mt-4 rounded-md bg-[#fef2f2] p-3 text-xs leading-[1.6] text-[#b91c1c]"
            role="alert"
          >
            {error}
          </p>
        )}
        {file && preview ? (
          <>
            <div className="my-4 flex min-h-9 flex-wrap items-center gap-2 text-xs text-[#52525b]">
              {current && selected !== null ? (
                <>
                  <span className="font-mono">[{current.bbox.join(", ")}]</span>
                  <button
                    type="button"
                    className={BUTTON}
                    onClick={() =>
                      updateBox(selected, (box) => ({
                        ...box,
                        is_checked: !box.is_checked,
                      }))
                    }
                  >
                    {current.is_checked ? "Mark unchecked" : "Mark checked"} (C)
                  </button>
                  <button
                    type="button"
                    className={BUTTON}
                    aria-pressed={current.ambiguous ?? false}
                    onClick={() =>
                      updateBox(selected, (box) => ({
                        ...box,
                        ambiguous: !box.ambiguous,
                      }))
                    }
                  >
                    {current.ambiguous ? "Clear ambiguous" : "Mark ambiguous"}{" "}
                    (A)
                  </button>
                  <button
                    type="button"
                    className={BUTTON}
                    onClick={() => deleteBox(selected)}
                  >
                    Delete (⌫)
                  </button>
                </>
              ) : (
                <span>
                  Click a box to edit it. Drag on the page to add a missing box;
                  to fix a misplaced one, delete it and draw it again.
                </span>
              )}
              <span className="ml-auto" aria-live="polite">
                {boxes.length} boxes · {checked} checked ·{" "}
                {boxes.length - checked} unchecked · {ambiguous} ambiguous
              </span>
            </div>
            <AnnotationCanvas
              src={preview}
              name={file.name}
              size={size}
              boxes={boxes}
              selected={selected}
              zoom={zoom}
              onLoad={setSize}
              onSelect={setSelected}
              onAdd={(bbox) => {
                setBoxes([...boxes, { bbox, is_checked: false }]);
                setSelected(boxes.length);
                setDirty(true);
              }}
            />
          </>
        ) : (
          <p className="mt-6 text-xs leading-[1.7] text-[#71717a]">
            Choose a PNG or JPEG from <code>backend/testdata</code> to begin.
          </p>
        )}
      </section>
    </PageLayout>
  );
}
