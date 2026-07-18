import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// NoteHub 実装仕様書: WebSocketは REST API と同じバックエンド(8080番)へ接続する
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/ws': {
        target: 'ws://localhost:8080',
        ws: true,
      },
    },
  },
})
