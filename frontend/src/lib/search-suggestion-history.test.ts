import { describe, expect, it, beforeEach } from 'vitest'

import {
  readSuggestionHistory,
  addSuggestionHistoryEntry,
  clearSuggestionHistory,
  removeSuggestionHistoryEntry,
} from '@/lib/search-suggestion-history'
import type { SuggestionHistoryEntry } from '@/lib/search-suggestion-history'

function mockLocalStorage(): void {
  const store: Record<string, string> = {}
  globalThis.window = {
    localStorage: {
      getItem: (k: string) => store[k] ?? null,
      setItem: (k: string, v: string) => { store[k] = v },
      removeItem: (k: string) => { delete store[k] },
    },
  } as unknown as Window & typeof globalThis
}

function makeEntry(id: string, kind: 'hashtag' | 'profile' = 'hashtag'): Omit<SuggestionHistoryEntry, 'visitedAt'> {
  return { id, kind, label: `#${id}`, subtitle: '', href: `/feed?hashtag=${id}` }
}

describe('readSuggestionHistory', () => {
  beforeEach(mockLocalStorage)

  it('retourne un tableau vide au départ', () => {
    expect(readSuggestionHistory('u1')).toEqual([])
  })

  it('retourne les entrées stockées', () => {
    addSuggestionHistoryEntry('u1', makeEntry('go'))
    expect(readSuggestionHistory('u1')).toHaveLength(1)
  })

  it("isole par userId", () => {
    addSuggestionHistoryEntry('u1', makeEntry('go'))
    expect(readSuggestionHistory('u2')).toHaveLength(0)
  })
})

describe('addSuggestionHistoryEntry', () => {
  beforeEach(mockLocalStorage)

  it('place la nouvelle entrée en tête', () => {
    addSuggestionHistoryEntry('u1', makeEntry('a'))
    addSuggestionHistoryEntry('u1', makeEntry('b'))
    expect(readSuggestionHistory('u1')[0].id).toBe('b')
  })

  it('déplace en tête si doublon', () => {
    addSuggestionHistoryEntry('u1', makeEntry('a'))
    addSuggestionHistoryEntry('u1', makeEntry('b'))
    addSuggestionHistoryEntry('u1', makeEntry('a'))
    const history = readSuggestionHistory('u1')
    expect(history[0].id).toBe('a')
    expect(history).toHaveLength(2)
  })

  it('plafonne à 8 entrées', () => {
    for (let i = 0; i < 10; i++) addSuggestionHistoryEntry('u1', makeEntry(String(i)))
    expect(readSuggestionHistory('u1')).toHaveLength(8)
  })

  it('accepte kind profile', () => {
    addSuggestionHistoryEntry('u1', makeEntry('alice', 'profile'))
    expect(readSuggestionHistory('u1')[0].kind).toBe('profile')
  })
})

describe('clearSuggestionHistory', () => {
  beforeEach(mockLocalStorage)

  it("vide l'historique", () => {
    addSuggestionHistoryEntry('u1', makeEntry('go'))
    clearSuggestionHistory('u1')
    expect(readSuggestionHistory('u1')).toHaveLength(0)
  })
})

describe('removeSuggestionHistoryEntry', () => {
  beforeEach(mockLocalStorage)

  it('supprime une entrée par id', () => {
    addSuggestionHistoryEntry('u1', makeEntry('a'))
    addSuggestionHistoryEntry('u1', makeEntry('b'))
    removeSuggestionHistoryEntry('u1', 'a')
    const history = readSuggestionHistory('u1')
    expect(history).toHaveLength(1)
    expect(history[0].id).toBe('b')
  })

  it("ne plante pas si l'id est absent", () => {
    addSuggestionHistoryEntry('u1', makeEntry('a'))
    expect(() => removeSuggestionHistoryEntry('u1', 'inexistant')).not.toThrow()
  })
})
