import { defineConfig } from "vite"
import react from "@vitejs/plugin-react"
import tailwindcss from "@tailwindcss/vite"
import { fileURLToPath, URL } from "node:url"

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: { alias: { "@": fileURLToPath(new URL("./src", import.meta.url)) } },
  server: {
    proxy: {
      "/api": {
        target: "http://127.0.0.1:8787",
        changeOrigin: true,
        configure(proxy) {
          // Preserve the server's same-origin protection through the development proxy.
          proxy.on("proxyReq", (upstream, request) => {
            if (request.headers.origin === `http://${request.headers.host}`) {
              upstream.setHeader("Origin", "http://127.0.0.1:8787")
            }
          })
        },
      },
    },
  },
})
