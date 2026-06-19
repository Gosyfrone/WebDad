import { describe, expect, it, beforeEach } from 'vitest'

import {
  isLegalPath,
  rememberLegalOrigin,
  getLegalOrigin,
  LEGAL_ORIGIN_KEY,
} from '@/lib/legal-origin'

describe('isLegalPath', () => {
  it('reconnaît les pages légales exactes', () => {
    expect(isLegalPath('/mentions-legales')).toBe(true)
    expect(isLegalPath('/cgu')).toBe(true)
    expect(isLegalPath('/confidentialite')).toBe(true)
  })

  it('reconnaît les sous-chemins légaux', () => {
    expect(isLegalPath('/mentions-legales/section-1')).toBe(true)
  })

  it('ne reconnaît pas les pages non légales', () => {
    expect(isLegalPath('/feed')).toBe(false)
    expect(isLegalPath('/')).toBe(false)
    expect(isLegalPath('/profil/alice')).toBe(false)
  })
})

describe('rememberLegalOrigin / getLegalOrigin', () => {
  beforeEach(() => {
    globalThis.window = {
      sessionStorage: (() => {
        const store: Record<string, string> = {}
        return {
          getItem: (k: string) => store[k] ?? null,
          setItem: (k: string, v: string) => { store[k] = v },
          removeItem: (k: string) => { delete store[k] },
        }
      })(),
    } as unknown as Window & typeof globalThis
  })

  it('mémorise une page non légale', () => {
    rememberLegalOrigin('/feed')
    expect(getLegalOrigin()).toBe('/feed')
  })

  it('ignore les pages légales', () => {
    window.sessionStorage.setItem(LEGAL_ORIGIN_KEY, '/feed')
    rememberLegalOrigin('/cgu')
    expect(getLegalOrigin()).toBe('/feed')
  })

  it('ignore les chemins vides', () => {
    rememberLegalOrigin('')
    expect(getLegalOrigin()).toBeNull()
  })

  it('retourne null si la valeur stockée est légale', () => {
    window.sessionStorage.setItem(LEGAL_ORIGIN_KEY, '/cgu')
    expect(getLegalOrigin()).toBeNull()
  })
})
