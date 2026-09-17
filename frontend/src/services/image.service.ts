import api from "./api.service.ts";

export class ImageService {
  async uploadImage(file: File): Promise<unknown> {
    const body = new FormData();
    body.append("image", file);

    return api.post("/detect", body);
  }
}

const imageService = new ImageService();

export default imageService;
