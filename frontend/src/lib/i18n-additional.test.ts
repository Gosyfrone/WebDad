import { describe, expect, it } from 'vitest'

import { additionalMessages } from '@/lib/i18n-additional'

// ─── additionalMessages ───────────────────────────────────────────────────────

describe('additionalMessages', () => {
  it('est un objet non vide', () => {
    expect(typeof additionalMessages).toBe('object')
    expect(additionalMessages).not.toBeNull()
    expect(Object.keys(additionalMessages).length).toBeGreaterThan(0)
  })

  it('contient des locales connues', () => {
    const locales = Object.keys(additionalMessages)
    // Au moins 3 locales supplémentaires attendues
    expect(locales.length).toBeGreaterThanOrEqual(3)
  })

  it('chaque locale est un dictionnaire string→string', () => {
    for (const [locale, dict] of Object.entries(additionalMessages)) {
      expect(typeof dict).toBe('object')
      for (const [key, value] of Object.entries(dict)) {
        expect(typeof key).toBe('string')
        expect(typeof value).toBe('string')
      }
      // Au moins quelques clés présentes par locale
      expect(Object.keys(dict).length).toBeGreaterThan(5)
      void locale
    }
  })

  it('les clés suivent le format namespace.key', () => {
    for (const dict of Object.values(additionalMessages)) {
      const keys = Object.keys(dict)
      const dotsCount = keys.filter((k) => k.includes('.')).length
      // La majorité des clés doit contenir un point
      expect(dotsCount).toBeGreaterThan(keys.length / 2)
    }
  })
})
