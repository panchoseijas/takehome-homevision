import type { ComponentProps } from "react";

type Variant = "plain" | "primary" | "active";

const BASE =
  "rounded-md border px-3 py-2 text-xs font-semibold whitespace-nowrap";

const VARIANTS: Record<Variant, string> = {
  plain:
    "border-[#e4e4e7] bg-white text-[#3f3f46] enabled:hover:bg-[#eef2ff] disabled:text-[#a1a1aa]",
  primary:
    "border-transparent bg-[#4f46e5] text-white enabled:hover:bg-[#4338ca] disabled:bg-[#e4e4f0] disabled:text-[#626279]",
  active: "border-[#c7d2fe] bg-[#eef2ff] text-[#4f46e5]",
};

type ToolButtonProps = ComponentProps<"button"> & { variant?: Variant };

export default function ToolButton({
  variant = "plain",
  className = "",
  ...props
}: ToolButtonProps) {
  return (
    <button
      type="button"
      className={`${BASE} ${VARIANTS[variant]} ${className}`}
      {...props}
    />
  );
}

type FilePickerProps = {
  label: string;
  accept: string;
  variant?: Variant;
  onPick: (files: File[]) => void;
};

// A file input styled as a button; the input itself stays keyboard reachable.
export function FilePicker({
  label,
  accept,
  variant = "plain",
  onPick,
}: FilePickerProps) {
  return (
    <label
      className={`${BASE} ${VARIANTS[variant]} cursor-pointer focus-within:outline-3 focus-within:outline-offset-4 focus-within:outline-[#6366f1]`}
    >
      {label}
      <input
        className="sr-only"
        type="file"
        accept={accept}
        onChange={(event) => {
          const files = Array.from(event.target.files ?? []);
          event.target.value = "";
          if (files.length) onPick(files);
        }}
      />
    </label>
  );
}
