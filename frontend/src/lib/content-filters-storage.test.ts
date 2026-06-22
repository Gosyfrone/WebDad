import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { readMutedWords, saveMutedWords, subscribeMutedWords, MUTED_WORDS_STORAGE_KEY } from '@/lib/content-filters'

let store: Record<string, string>
let listeners: Map<string, Array<(e: Event) => void>>

beforeEach(() => {
  store = {}
  listeners = new Map()
  vi.stubGlobal('window', {
    localStorage: {
      getItem: (k: string) => store[k] ?? null,
      setItem: (k: string, v: string) => { store[k] = v },
      removeItem: (k: string) => { delete store[k] },
    },
    addEventListener: (t: string, h: (e: Event) => void) => {
      listeners.set(t, [...(listeners.get(t) ?? []), h])
    },
    removeEventListener: (t: string, h: (e: Event) => void) => {
      listeners.set(t, (listeners.get(t) ?? []).filter((x) => x !== h))
    },
    dispatchEvent: (e: Event) => { (listeners.get(e.type) ?? []).forEach((h) => h(e)); return true },
  })
  vi.stubGlobal('CustomEvent', class {
    type: string; detail: unknown
    constructor(type: string, init: { detail: unknown }) { this.type = type; this.detail = init.detail }
  })
})
afterEach(() => vi.unstubAllGlobals())

const key = (u: string) => `${MUTED_WORDS_STORAGE_KEY}:${u || 'anonymous'}`

describe('readMutedWords', () => {
  it('vide si aucune entrée', () => {
    expect(readMutedWords('u1')).toEqual([])
  })
  it('lit et assainit la liste stockée', () => {
    store[key('u1')] = JSON.stringify(['Spam', 'spam', '  ', 42, 'Hué'])
    expect(readMutedWords('u1')).toEqual(['Spam', 'Hué'])
  })
  it('JSON invalide → []', () => {
    store[key('u1')] = '{bad'
    expect(readMutedWords('u1')).toEqual([])
  })
  it('valeur non-tableau → []', () => {
    store[key('u1')] = JSON.stringify({ a: 1 })
    expect(readMutedWords('u1')).toEqual([])
  })
})

describe('saveMutedWords', () => {
  it('persiste la liste assainie et émet l’évènement', () => {
    let received: string[] | null = null
    subscribeMutedWords('u1', (w) => { received = w })
    const saved = saveMutedWords(['Foo', 'foo', 'Bar'], 'u1')
    expect(saved).toEqual(['Foo', 'Bar'])
    expect(JSON.parse(store[key('u1')])).toEqual(['Foo', 'Bar'])
    expect(received).toEqual(['Foo', 'Bar'])
  })
})

describe('subscribeMutedWords', () => {
  it('ignore les évènements d’un autre utilisateur et réagit au storage', () => {
    const seen: string[][] = []
    const off = subscribeMutedWords('u1', (w) => seen.push(w))
    // évènement local pour un autre user → ignoré
    saveMutedWords(['x'], 'u2')
    expect(seen).toHaveLength(0)
    // évènement storage sur la bonne clé → relit
    store[key('u1')] = JSON.stringify(['z'])
    listeners.get('storage')?.forEach((h) => h({ key: key('u1') } as unknown as Event))
    expect(seen.at(-1)).toEqual(['z'])
    off()
    saveMutedWords(['y'], 'u1')
    expect(seen.at(-1)).toEqual(['z']) // plus de notification après désabonnement
  })
})
