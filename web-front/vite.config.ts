import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// In dev, the SPA runs on :5173 and proxies API calls to the Go server on
// :8080 so the browser sees a single origin (no CORS, cookies just work).
// In production, Caddy serves the built dist/ and proxies /api the same way.
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      "/api": { target: "http://127.0.0.1:8080", changeOrigin: false },
      "/healthz": { target: "http://127.0.0.1:8080", changeOrigin: false },
    },
  },
  build: {
    outDir: "dist",
    sourcemap: false,
  },
});
