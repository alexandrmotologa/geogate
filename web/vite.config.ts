import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    react(),
    tailwindcss(),
  ],
  base: './',
  server: {
    proxy: {
      '/v1': 'http://localhost:8080',
      '/health': 'http://localhost:8080',
      '/metrics': 'http://localhost:8080',
    },
  },
})
