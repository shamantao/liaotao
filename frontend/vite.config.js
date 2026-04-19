// vite.config.js — Vite build configuration for liaotao frontend.
// Compiles Svelte components to vanilla JS; outputs to dist/.

import { defineConfig } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";

export default defineConfig({
  plugins: [svelte()],
  root: ".",
  base: "./",
  build: {
    outDir: "dist",
    emptyOutDir: true,
    target: "es2020",
  },
  server: {
    port: 5173,
    strictPort: true,
  },
  test: {
    environment: "jsdom",
    globals: true,
    include: ["src/**/*.test.js"],
    setupFiles: ["src/__tests__/setup.js"],
  },
});
