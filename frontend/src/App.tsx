import { useMutation } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import DocumentUpload from "./components/DocumentUpload";
import EndpointResponse from "./components/EndpointResponse";
import ImagePreview from "./components/ImagePreview";
import PageLayout from "./components/PageLayout";
import WorkspaceIntro from "./components/WorkspaceIntro";
import imageService from "./services/image.service";
import { validateImage } from "./upload";

function App() {
  const [file, setFile] = useState<File | null>(null);
  const [preview, setPreview] = useState("");
  const [validationError, setValidationError] = useState("");

  useEffect(() => {
    return () => {
      if (preview) URL.revokeObjectURL(preview);
    };
  }, [preview]);

  const mutation = useMutation({
    mutationFn: imageService.uploadImage,
    retry: false,
    networkMode: "always",
  });
  const error = validationError || mutation.error?.message;

  function clearSelection() {
    mutation.reset();
    setFile(null);
    setPreview("");
    setValidationError("");
  }

  function selectFile(files: File[]) {
    clearSelection();
    const message =
      files.length !== 1
        ? "Choose one image at a time."
        : validateImage(files[0]);
    if (message) {
      setValidationError(message);
    } else {
      setFile(files[0]);
      setPreview(URL.createObjectURL(files[0]));
    }
  }

  return (
    <PageLayout>
      <WorkspaceIntro />
      <div className="mt-9 bg-[#F7F8FC] grid grid-cols-1 overflow-hidden rounded-[20px] border border-[#e4e4e7] shadow-[0_6px_24px_#18181b08] min-[761px]:mt-12 min-[761px]:grid-cols-[1fr_1.2fr]">
        <DocumentUpload
          file={file}
          isPending={mutation.isPending}
          isSuccess={mutation.isSuccess}
          error={error}
          onSelectFiles={selectFile}
          onRemove={clearSelection}
          onUpload={() => {
            if (file && !mutation.isPending) mutation.mutate(file);
          }}
        />
        <ImagePreview
          file={file}
          preview={preview}
          onPreviewError={() => {
            clearSelection();
            setValidationError(
              "This image could not be displayed. Choose a valid PNG or JPEG.",
            );
          }}
        />
      </div>
      {mutation.isSuccess && <EndpointResponse data={mutation.data} />}
    </PageLayout>
  );
}

export default App;
