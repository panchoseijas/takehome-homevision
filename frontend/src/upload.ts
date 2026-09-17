export const MAX_IMAGE_BYTES = 10 * 1024 * 1024;

export function validateImage(file: File): string | null {
  if (!["image/png", "image/jpeg"].includes(file.type)) {
    return "Choose a PNG or JPEG image.";
  }
  if (file.size === 0) return "This file is empty. Choose another image.";
  if (file.size > MAX_IMAGE_BYTES) return "Choose an image smaller than 10 MB.";
  return null;
}
