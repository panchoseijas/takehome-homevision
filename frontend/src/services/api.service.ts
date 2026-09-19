export class ApiValidationError extends Error {
  public field?: string;
  public status?: number;

  constructor(message: string, field?: string, status?: number) {
    super(message);
    this.name = "ApiValidationError";
    this.field = field;
    this.status = status;
  }
}

type JsonBody = Record<string, unknown> | unknown[];
type RequestBody = FormData | JsonBody;

export class ApiService {
  private readonly baseUrl = import.meta.env?.VITE_API_BASE_URL ?? "";

  async get<T>(endpoint: string): Promise<T> {
    return this.request<T>(endpoint, { method: "GET" });
  }

  async post<T>(endpoint: string, body: RequestBody): Promise<T> {
    return this.request<T>(endpoint, this.createRequest("POST", body));
  }

  async put<T>(endpoint: string, body: RequestBody): Promise<T> {
    return this.request<T>(endpoint, this.createRequest("PUT", body));
  }

  async delete<T>(endpoint: string): Promise<T> {
    return this.request<T>(endpoint, { method: "DELETE" });
  }

  private createRequest(
    method: "POST" | "PUT",
    body: RequestBody,
  ): RequestInit {
    if (body instanceof FormData) return { method, body };

    return {
      method,
      body: JSON.stringify(body),
      headers: { "Content-Type": "application/json" },
    };
  }

  private async request<T>(endpoint: string, init: RequestInit): Promise<T> {
    const response = await fetch(this.baseUrl + endpoint, init);

    if (response.status >= 500) {
      throw new Error("Server error");
    }

    if (response.status >= 400) {
      const error = await this.readError(response);
      throw new ApiValidationError(
        error.error ?? `Request failed (HTTP ${response.status})`,
        error.field,
        response.status,
      );
    }

    if (response.status === 204) return null as T;
    return response.json() as Promise<T>;
  }

  // The backend reports failures as {"error": "message"}.
  private async readError(
    response: Response,
  ): Promise<{ error?: string; field?: string }> {
    return response
      .json()
      .then((body) => body as { error?: string; field?: string })
      .catch(() => ({}));
  }
}

const api = new ApiService();

export default api;
