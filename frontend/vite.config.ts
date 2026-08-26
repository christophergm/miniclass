import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import path from 'path'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  // There is one .env in the repository, at the root. Pointing envDir there
  // makes that structural rather than conventional: a stray frontend/.env is
  // then never read, instead of being read and merely losing to an exported
  // value that happens to have the same name.
  envDir: path.resolve(__dirname, '..'),
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    port: parseInt(process.env.VITE_PORT || '5173'),
    proxy: {
      '/api': {
        // API_PROXY_TARGET is node-side only: it tells this dev server where to
        // forward /api. It is never exposed to the browser. The client bundle's
        // API base is VITE_API_URL, which is left empty in local development so
        // the browser requests a relative /api and travels through this proxy
        // (same-origin, so the CSP connect-src needs nothing beyond 'self').
        target: process.env.API_PROXY_TARGET || 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
  test: {
    environment: 'jsdom',
    setupFiles: './src/test/setup.ts',
  },
})
