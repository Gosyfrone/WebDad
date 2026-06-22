import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { hasReadTerms, markTermsRead, TERMS_READ_STORAGE_KEY } from '@/lib/terms-consent'

let store: Record<string, string>
beforeEach(() => {
  store = {}
  vi.stubGlobal('window', {
    localStorage: {
      getItem: (k: string) => store[k] ?? null,
      setItem: (k: string, v: string) => { store[k] = v },
    },
  })
})
afterEach(() => vi.unstubAllGlobals())

describe('terms-consent', () => {
  it('hasReadTerms false par défaut', () => {
    expect(hasReadTerms()).toBe(false)
  })
  it('markTermsRead persiste true', () => {
    markTermsRead()
    expect(store[TERMS_READ_STORAGE_KEY]).toBe('true')
    expect(hasReadTerms()).toBe(true)
  })
})
