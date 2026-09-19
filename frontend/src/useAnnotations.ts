import { useEffect, useState } from "react";
import {
  serializeTruth,
  truthFileName,
  type AnnotatedBox,
} from "./annotation";

// Owns the boxes drawn over the open image: the detector's result while
// viewing, an editable draft while annotating, and the file saved from it.
// `image` is the open document, and names the file that "save" downloads.
export function useAnnotations(image: File | null) {
  const [boxes, setBoxes] = useState<AnnotatedBox[] | null>(null);
  const [selected, setSelected] = useState<number | null>(null);
  const [annotating, setAnnotating] = useState(false);
  const [dirty, setDirty] = useState(false);

  // Only hand-made edits are unsaved work; a detection can always be re-run.
  useEffect(() => {
    if (!dirty) return;
    const warn = (event: BeforeUnloadEvent) => event.preventDefault();
    window.addEventListener("beforeunload", warn);
    return () => window.removeEventListener("beforeunload", warn);
  }, [dirty]);

  useEffect(() => {
    if (!annotating || selected === null) return;
    const index = selected;
    function onKeyDown(event: KeyboardEvent) {
      if (event.target instanceof HTMLInputElement) return;
      if (event.key === "c" || event.key === " ") {
        toggleChecked(index);
      } else if (event.key === "a") {
        toggleAmbiguous(index);
      } else if (event.key === "Delete" || event.key === "Backspace") {
        remove(index);
      } else if (event.key === "Escape") {
        setSelected(null);
      } else {
        return;
      }
      event.preventDefault();
    }
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [annotating, selected]);

  function edit(index: number, change: (box: AnnotatedBox) => AnnotatedBox) {
    setBoxes((current) =>
      (current ?? []).map((box, i) => (i === index ? change(box) : box)),
    );
    setDirty(true);
  }

  function toggleChecked(index: number) {
    edit(index, (box) => ({ ...box, is_checked: !box.is_checked }));
  }

  function toggleAmbiguous(index: number) {
    edit(index, (box) => ({ ...box, ambiguous: !box.ambiguous }));
  }

  function remove(index: number) {
    setBoxes((current) => (current ?? []).filter((_, i) => i !== index));
    setSelected(null);
    setDirty(true);
  }

  function add(bbox: AnnotatedBox["bbox"]) {
    setBoxes((current) => [...(current ?? []), { bbox, is_checked: false }]);
    setSelected(boxes?.length ?? 0);
    setDirty(true);
  }

  // Replaces the boxes wholesale: a detection, a loaded file, or a new image.
  function replace(next: AnnotatedBox[] | null) {
    setBoxes(next);
    setSelected(null);
    setDirty(false);
  }

  // Annotating an image that was never detected starts from an empty draft.
  function toggleAnnotating() {
    setSelected(null);
    setAnnotating(!annotating);
    if (!annotating && image && boxes === null) setBoxes([]);
  }

  function confirmDiscard() {
    return !dirty || window.confirm("Discard the unsaved annotation edits?");
  }

  function save() {
    if (!image || !boxes) return;
    const url = URL.createObjectURL(
      new Blob([serializeTruth(boxes)], { type: "application/json" }),
    );
    const link = document.createElement("a");
    link.href = url;
    link.download = truthFileName(image.name);
    link.click();
    URL.revokeObjectURL(url);
    setDirty(false);
  }

  return {
    boxes,
    selected,
    annotating,
    dirty,
    canSave: boxes !== null && image !== null,
    select: setSelected,
    toggleChecked,
    toggleAmbiguous,
    remove,
    add,
    replace,
    toggleAnnotating,
    confirmDiscard,
    save,
  };
}
