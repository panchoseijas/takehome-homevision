import { useRef, useState } from "react";

type DocumentUploadProps = {
  file: File | null;
  isPending: boolean;
  isSuccess: boolean;
  error: string | undefined;
  onSelectFiles: (files: File[]) => void;
  onRemove: () => void;
  onUpload: () => void;
};

export default function DocumentUpload({
  file,
  isPending,
  isSuccess,
  error,
  onSelectFiles,
  onRemove,
  onUpload,
}: DocumentUploadProps) {
  const [dragging, setDragging] = useState(false);
  const input = useRef<HTMLInputElement>(null);

  return (
    <section
      className="border-b border-[#e4e4e7] p-5.5 min-[761px]:border-r min-[761px]:border-b-0 min-[761px]:p-8"
      aria-labelledby="upload-heading"
    >
      <div className="mb-6.25 flex items-center gap-2.5">
        <span className="rounded bg-[#eef2ff] px-1.5 py-1.25 font-mono text-[11px] text-[#4f46e5]">
          01
        </span>
        <h2 className="text-sm font-[650]" id="upload-heading">
          Your document
        </h2>
      </div>
      <input
        ref={input}
        className="sr-only"
        type="file"
        accept="image/png,image/jpeg"
        aria-label="Choose document image"
        onChange={(event) => {
          if (event.target.files?.length)
            onSelectFiles(Array.from(event.target.files));
          event.target.value = "";
        }}
      />
      <button
        type="button"
        className={`flex min-h-61 w-full flex-col items-center justify-center gap-2.5 rounded-lg border border-dashed text-[#52525b] hover:border-[#6366f1] hover:bg-[#eef2ff] ${dragging ? "border-[#6366f1] bg-[#eef2ff]" : "border-[#c7d2fe] bg-[#fafaff]"}`}
        onClick={() => input.current?.click()}
        onDragOver={(event) => {
          event.preventDefault();
          setDragging(true);
        }}
        onDragLeave={() => setDragging(false)}
        onDrop={(event) => {
          event.preventDefault();
          setDragging(false);
          onSelectFiles(Array.from(event.dataTransfer.files));
        }}
      >
        <span
          className="mb-2 grid size-11.75 place-items-center rounded-xl border border-[#e0e7ff] bg-[#eef2ff] text-[27px] text-[#4f46e5]"
          aria-hidden="true"
        >
          ↑
        </span>
        <strong className="text-[15px] font-semibold">
          {file ? "Choose a different image" : "Drop your image here"}
        </strong>
        <span className="text-[13px]">
          or{" "}
          <span className="text-[#4f46e5] underline underline-offset-[3px]">
            browse files
          </span>
        </span>
        <small className="mt-3 text-[11px] text-[#71717a]">
          PNG or JPEG · up to 10 MB · one image
        </small>
      </button>
      {file && (
        <div className="mt-4 flex min-w-0 items-center gap-3 rounded-md bg-[#f5f5ff] p-3">
          <span className="text-[23px] text-[#6366f1]" aria-hidden="true">
            ▧
          </span>
          <div className="min-w-0 flex-1">
            <strong className="block text-xs font-[550] wrap-anywhere">
              {file.name}
            </strong>
            <span className="mt-1 block text-[10px] text-[#71717a]">
              {(file.size / 1024 / 1024).toFixed(2)} MB · Ready to upload
            </span>
          </div>
          <button
            className="border-0 bg-transparent px-2 py-1 text-2xl text-[#71717a]"
            type="button"
            aria-label="Remove image"
            onClick={onRemove}
          >
            ×
          </button>
        </div>
      )}
      <button
        className="mt-6 flex w-full justify-between rounded-lg border-0 bg-[#4f46e5] px-4.5 py-3.75 font-semibold text-white enabled:hover:bg-[#4338ca] disabled:bg-[#e4e4f0] disabled:text-[#626279]"
        disabled={!file || isPending}
        onClick={onUpload}
      >
        {isPending ? "Uploading…" : "Upload image"}
        <span aria-hidden="true">→</span>
      </button>
      <p className="mt-3.5 text-center text-[11px] text-[#71717a]">
        Your image is sent only when you click upload.
      </p>
      <div aria-live="polite" aria-atomic="true">
        {isPending && (
          <p className="mt-4 rounded-md bg-[#eef2ff] p-3 text-xs leading-[1.6]">
            Sending your image. This may take a moment.
          </p>
        )}
        {isSuccess && (
          <p className="mt-4 rounded-md bg-[#edf5ec] p-3 text-xs leading-[1.6] text-[#2d6546]">
            Image uploaded successfully.
          </p>
        )}
      </div>
      {error && (
        <p
          className="mt-4 rounded-md bg-[#fff1ed] p-3 text-xs leading-[1.6] text-[#a14029]"
          role="alert"
        >
          {error}
        </p>
      )}
    </section>
  );
}
