/// <reference types="vitest" />
import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'
import { svelteTesting } from '@testing-library/svelte/vite'

export default defineConfig({
  plugins: [svelte(), svelteTesting()],
  server: {
    proxy: {
      // Браузер коннектится на ws://<host>/ws (тот же origin, что и SPA);
      // Vite проксирует на Go-сервер. changeOrigin НЕ ставим — иначе Host
      // разойдётся с Origin и coder/websocket вернёт 403 на хендшейке.
      '/ws': { target: 'ws://localhost:8080', ws: true },
    },
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./tests/setup.ts'],
  },
})
