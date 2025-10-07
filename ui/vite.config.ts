import path from 'path'
import tailwindcss from '@tailwindcss/vite'
import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  build: {
    outDir: '../static',
    emptyOutDir: false, // 不清空目录，保留 embed.go
    rollupOptions: {
      output: {
        manualChunks: {
          monaco: ['monaco-editor'],
          lodash: ['lodash'],
          recharts: ['recharts'],
        },
      },
    },
  },
  server: {
    watch: {
      ignored: ['**/.vscode/**'],
    },
    proxy: {
      '/api/': {
        changeOrigin: true,
        target: 'http://localhost:12306',
        ws: true, // Support WebSocket upgrade for /api/v1/vms/ws/* paths
      },
    },
  },
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  worker: {
    format: 'es',
  },
  define: {
    global: 'globalThis',
  },
})
