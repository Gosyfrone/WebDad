import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { rememberLegalOrigin, getLegalOrigin, LEGAL_ORIGIN_KEY } from '@/lib/legal-origin'

let store: Record<string, string>
function stub(throwing = false) {
  store = {}
  vi.stubGlobal('window', {
    sessionStorage: {
      getItem: (k: string) => store[k] ?? null,
      setItem: (k: string, v: string) => { if (throwing) throw new Error('x'); store[k] = v },
    },
  })
}
beforeEach(() => stub())
afterEach(() => vi.unstubAllGlobals())

describe('legal-origin', () => {
  it('mémorise une page Breezy normale', () => {
    rememberLegalOrigin('/feed')
    expect(store[LEGAL_ORIGIN_KEY]).toBe('/feed')
    expect(getLegalOrigin()).toBe('/feed')
  })
  it('ignore les pages légales et le vide', () => {
    rememberLegalOrigin('/mentions-legales')
    rememberLegalOrigin('')
    expect(store[LEGAL_ORIGIN_KEY]).toBeUndefined()
  })
  it('getLegalOrigin null si aucune origine', () => {
    expect(getLegalOrigin()).toBeNull()
  })
  it('getLegalOrigin null si la valeur stockée est une page légale', () => {
    store[LEGAL_ORIGIN_KEY] = '/cgu'
    expect(getLegalOrigin()).toBeNull()
  })
  it('sessionStorage indisponible → silencieux', () => {
    stub(true)
    expect(() => rememberLegalOrigin('/feed')).not.toThrow()
  })
})
