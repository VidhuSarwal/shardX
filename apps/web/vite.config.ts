import { defineConfig } from "vite";
import { configDefaults } from "vitest/config";
import react from "@vitejs/plugin-react-swc";
import path from "path";

// https://vitejs.dev/config/
export default defineConfig({
  server: {
    host: "::",
    port: 5173,
  },
  plugins: [react()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  test: {
    // Playwright's e2e/*.spec.ts files match vitest's default include glob too;
    // keep unit tests (vitest) and browser smoke tests (playwright) separate.
    exclude: [...configDefaults.exclude, "**/e2e/**"],
  },
});
