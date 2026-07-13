import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)

export default defineConfig({
  plugins: [react()],
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./vitest.setup.ts'],
    exclude: ['e2e/**', 'node_modules/**', 'dist/**'],
    // Node 26's built-in `globalThis.localStorage` requires the
    // `--localstorage-file` CLI flag and otherwise shadows jsdom's
    // window.localStorage by leaving it `undefined`. Asking jsdom for an
    // explicit in-memory URL gives us a real Storage on both
    // `globalThis.localStorage` and `window.localStorage` without depending
    // on Node CLI flags.
    environmentOptions: {
      jsdom: {
        url: 'http://localhost/',
      },
    },
  },
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
})
