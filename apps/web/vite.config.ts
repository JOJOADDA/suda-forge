import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// Keep browser requests same-origin in development so HttpOnly session cookies work.
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
      '/auth': 'http://localhost:8080',
      '/health': 'http://localhost:8080',
      '/ready': 'http://localhost:8080',
    },
  },
})
