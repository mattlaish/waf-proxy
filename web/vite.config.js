import { defineConfig } from "vite";
import preact from "@preact/preset-vite";

// Build output lands in ../static/app so the Go binary can serve/embed it.
// During dev, `vite` proxies /api to the running waf-proxy admin server.
export default defineConfig({
  // Using preact via aliases avoids needing @preact/preset-vite if you prefer;
  // the preset gives you Fast Refresh in dev. If you don't install the preset,
  // remove the import above and the plugins line below, and keep the alias.
  plugins: (() => {
    try {
      return [preact()];
    } catch {
      return [];
    }
  })(),
  resolve: {
    alias: {
      react: "preact/compat",
      "react-dom": "preact/compat",
    },
  },
  build: {
    outDir: "../static/app",
    emptyOutDir: true,
    rollupOptions: {
      output: { entryFileNames: "app.js", assetFileNames: "app[extname]" },
    },
  },
  server: {
    proxy: {
      "/api": "http://127.0.0.1:9090",
      "/healthz": "http://127.0.0.1:9090",
    },
  },
});
