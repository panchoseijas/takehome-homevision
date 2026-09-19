import { useMutation } from "@tanstack/react-query";
import { useState } from "react";
import type { ImageSize } from "./annotation";
import EndpointResponse from "./components/EndpointResponse";
import ImageStage from "./components/ImageStage";
import PageLayout from "./components/PageLayout";
import StatusBar from "./components/StatusBar";
import Toolbar from "./components/Toolbar";
import imageService from "./services/image.service";
import { validateImage } from "./upload";
import { useAnnotations } from "./useAnnotations";
import { useExpanded } from "./useExpanded";

type OpenImage = { file: File; preview: string };

export default function App() {
  const [image, setImage] = useState<OpenImage | null>(null);
  const [size, setSize] = useState<ImageSize | null>(null);
  const [zoom, setZoom] = useState(1);
  const [message, setMessage] = useState("");
  const file = image?.file ?? null;
  const annotations = useAnnotations(file);
  const workspace = useExpanded({
    collapseOnEscape: annotations.selected === null,
  });

  const detection = useMutation({
    mutationFn: imageService.uploadImage,
    retry: false,
    networkMode: "always",
    onSuccess: (data) => annotations.replace(data?.boxes ?? []),
  });
  const error = message || detection.error?.message;

  // A blob URL outlives the File it came from, so release it as it is replaced.
  // The last one is released with the page.
  function openImage(next: File | null) {
    if (image) URL.revokeObjectURL(image.preview);
    setImage(next && { file: next, preview: URL.createObjectURL(next) });
    setSize(null);
    setZoom(1);
  }

  // A rejected file leaves the open image and its boxes untouched.
  function selectFiles(files: File[]) {
    if (files.length === 0) return;
    const invalid =
      files.length > 1
        ? "Choose one image at a time."
        : validateImage(files[0]);
    if (invalid) {
      setMessage(invalid);
      return;
    }
    if (!annotations.confirmDiscard()) return;
    detection.reset();
    setMessage("");
    annotations.replace(annotations.annotating ? [] : null);
    openImage(files[0]);
  }

  function detect() {
    if (!file || detection.isPending) return;
    if (
      annotations.dirty &&
      !window.confirm("Replace your edits with a fresh detector draft?")
    )
      return;
    setMessage("");
    detection.mutate(file);
  }

  return (
    <PageLayout>
      <section
        className={`flex flex-col border bg-white p-4 min-[761px]:p-5 ${annotations.annotating ? "border-[#c7d2fe]" : "border-[#e4e4e7]"} ${
          workspace.expanded
            ? "fixed inset-0 z-50 h-svh rounded-none"
            : `h-[calc(100svh-190px)] min-h-110 rounded-[20px] shadow-[0_6px_24px_#18181b08] ${annotations.annotating ? "ring-3 ring-[#eef2ff]" : ""}`
        }`}
        aria-label="Document workspace"
      >
        <Toolbar
          file={file}
          annotating={annotations.annotating}
          isDetecting={detection.isPending}
          canSave={annotations.canSave}
          zoom={zoom}
          expanded={workspace.expanded}
          onSelectFiles={selectFiles}
          onDetect={detect}
          onToggleAnnotating={annotations.toggleAnnotating}
          onZoom={setZoom}
          onSave={annotations.save}
          onToggleExpanded={workspace.toggle}
        />
        {error && (
          <p
            className="mt-3 rounded-md bg-[#fff1ed] p-3 text-xs leading-[1.6] text-[#a14029]"
            role="alert"
          >
            {error}
          </p>
        )}
        <StatusBar
          file={file}
          boxes={annotations.boxes}
          annotating={annotations.annotating}
          selected={annotations.selected}
          onToggleChecked={annotations.toggleChecked}
          onToggleAmbiguous={annotations.toggleAmbiguous}
          onDelete={annotations.remove}
        />
        <ImageStage
          file={file}
          preview={image?.preview ?? ""}
          size={size}
          boxes={annotations.boxes}
          selected={annotations.selected}
          zoom={zoom}
          annotating={annotations.annotating}
          onLoad={setSize}
          onError={() => {
            openImage(null);
            annotations.replace(null);
            setMessage(
              "This image could not be displayed. Choose a valid PNG or JPEG.",
            );
          }}
          onSelect={annotations.select}
          onAdd={annotations.add}
          onSelectFiles={selectFiles}
        />
      </section>
      {detection.isSuccess && <EndpointResponse data={detection.data} />}
    </PageLayout>
  );
}
