import { defineConfig } from 'vitest/config'
import { fileURLToPath } from 'node:url'

// Tests unitaires de la couche technique (lib/*). Environnement Node : les
// primitives crypto utilisent WebCrypto (`globalThis.crypto`), disponible en
// Node 18+. L'alias `@` reflète le `paths` du tsconfig (@/* -> src/*).
export default defineConfig({
  test: {
    environment: 'node',
    include: ['src/**/*.test.ts'],
  },
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
})
