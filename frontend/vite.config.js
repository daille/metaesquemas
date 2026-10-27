import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// En desarrollo, /api se reenvía al backend Go (docker compose o `go run ./cmd/api`).
export default defineConfig({
  plugins: [vue()],
  server: { port: 5173, proxy: { '/api': { target: process.env.API_URL || 'http://localhost:8080', changeOrigin: false } } },
  build: { target: 'es2020', sourcemap: false, chunkSizeWarningLimit: 700 },
})
