import { describe, expect, it, beforeEach } from 'vitest'

import {
  readSearchHistory,
  addSearchHistoryEntry,
  clearSearchHistory,
  removeSearchHistoryEntry,
} from '@/lib/search-history'
import type { RelationUser } from '@/types'

function makeUser(id: string): RelationUser {
  return { id, username: `user${id}`, displayName: `User ${id}`, avatarUrl: null, isVerified: false }
}

function mockLocalStorage(): Record<string, string> {
  const store: Record<string, string> = {}
  globalThis.window = {
    localStorage: {
      getItem: (k: string) => store[k] ?? null,
      setItem: (k: string, v: string) => { store[k] = v },
      removeItem: (k: string) => { delete store[k] },
    },
  } as unknown as Window & typeof globalThis
  return store
}

describe('readSearchHistory', () => {
  beforeEach(() => { mockLocalStorage() })

  it('retourne un tableau vide si rien en mémoire', () => {
    expect(readSearchHistory('u1')).toEqual([])
  })

  it('retourne les entrées stockées', () => {
    addSearchHistoryEntry('u1', makeUser('a'))
    const history = readSearchHistory('u1')
    expect(history).toHaveLength(1)
    expect(history[0].id).toBe('a')
  })

  it("sépare l'historique par userId", () => {
    addSearchHistoryEntry('u1', makeUser('a'))
    expect(readSearchHistory('u2')).toHaveLength(0)
  })
})

describe('addSearchHistoryEntry', () => {
  beforeEach(() => { mockLocalStorage() })

  it('place la nouvelle entrée en tête', () => {
    addSearchHistoryEntry('u1', makeUser('a'))
    addSearchHistoryEntry('u1', makeUser('b'))
    const history = readSearchHistory('u1')
    expect(history[0].id).toBe('b')
  })

  it('déplace en tête si doublon', () => {
    addSearchHistoryEntry('u1', makeUser('a'))
    addSearchHistoryEntry('u1', makeUser('b'))
    addSearchHistoryEntry('u1', makeUser('a'))
    const history = readSearchHistory('u1')
    expect(history[0].id).toBe('a')
    expect(history).toHaveLength(2)
  })

  it('plafonne à 8 entrées', () => {
    for (let i = 0; i < 10; i++) addSearchHistoryEntry('u1', makeUser(String(i)))
    expect(readSearchHistory('u1')).toHaveLength(8)
  })
})

describe('clearSearchHistory', () => {
  beforeEach(() => { mockLocalStorage() })

  it("vide l'historique", () => {
    addSearchHistoryEntry('u1', makeUser('a'))
    clearSearchHistory('u1')
    expect(readSearchHistory('u1')).toHaveLength(0)
  })
})

describe('removeSearchHistoryEntry', () => {
  beforeEach(() => { mockLocalStorage() })

  it('supprime une entrée par id', () => {
    addSearchHistoryEntry('u1', makeUser('a'))
    addSearchHistoryEntry('u1', makeUser('b'))
    removeSearchHistoryEntry('u1', 'a')
    const history = readSearchHistory('u1')
    expect(history).toHaveLength(1)
    expect(history[0].id).toBe('b')
  })

  it("ne plante pas si l'id est absent", () => {
    addSearchHistoryEntry('u1', makeUser('a'))
    expect(() => removeSearchHistoryEntry('u1', 'inexistant')).not.toThrow()
  })
})
