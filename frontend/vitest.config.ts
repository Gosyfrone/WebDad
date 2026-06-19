import { defineConfig } from 'vitest/config'
import { fileURLToPath } from 'node:url'

// Tests unitaires de la couche technique (lib/*). Environnement Node : les
// primitives crypto utilisent WebCrypto (`globalThis.crypto`), disponible en
// Node 18+. L'alias `@` reflète le `paths` du tsconfig (@/* -> src/*).
export default defineConfig({
  test: {
    environment: 'node',
    include: ['src/**/*.test.ts'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'lcov'],
      // Périmètre = la couche technique réellement testée (lib/*). On exclut
      // le généré et le non-testé pour un % de couverture représentatif.
      include: ['src/lib/**'],
      exclude: ['src/**/*.test.ts', 'src/**/*.d.ts'],
    },
  },
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
})
