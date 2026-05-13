import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)

// In Docker dev, backend container is reachable as "backend:3001"
// Locally, it's "localhost:3001"
const API_TARGET = process.env.VITE_API_TARGET ?? 'http://localhost:3001'

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
      '@/components/ui': path.resolve(__dirname, './src/shared/components/ui'),
      '@/components/layout': path.resolve(__dirname, './src/shared/components/layout'),
      '@/hooks': path.resolve(__dirname, './src/shared/hooks'),
      '@/lib': path.resolve(__dirname, './src/shared/libs'),
      '@/types': path.resolve(__dirname, './src/shared/types'),
      features: path.resolve(__dirname, './src/features'),
      shared: path.resolve(__dirname, './src/shared'),
    },
  },
  server: {
    port: 5173,
    host: true, // needed inside Docker to bind 0.0.0.0
    proxy: {
      '/api': {
        target: API_TARGET,
        changeOrigin: true,
      },
    },
  },
})
