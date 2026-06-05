import { defineConfig } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";

// base relatif → hasil build bisa dibuka via file:// dan di-embed Wails.
export default defineConfig({
  plugins: [svelte()],
  base: "./",
  server: { port: 5173, strictPort: true },
});
