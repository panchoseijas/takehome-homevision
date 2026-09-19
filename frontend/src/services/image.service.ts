import type { DetectResponse } from "../detection.ts";
import api from "./api.service.ts";

export class ImageService {
  async uploadImage(file: File): Promise<DetectResponse | null> {
    const body = new FormData();
    body.append("image", file);

    return api.post("/detect", body);
  }
}

const imageService = new ImageService();

export default imageService;
