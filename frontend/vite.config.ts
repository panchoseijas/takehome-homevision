import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { defineConfig, loadEnv } from "vite";

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), "");
  const proxy = {
    "/detect": {
      target: env.API_PROXY_TARGET || "http://localhost:8080",
      changeOrigin: true,
    },
  };
  return {
    plugins: [react(), tailwindcss()],
    server: { proxy },
    preview: { proxy },
  };
});
