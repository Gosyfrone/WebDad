import { describe, expect, it, beforeEach } from 'vitest'

import {
  absoluteUrl,
  canNativeShare,
  extractSharedRef,
  getRecentShareTargets,
  recordRecentShareTarget,
} from '@/lib/share'
import type { RecentShareTarget } from '@/lib/share'

// ─── absoluteUrl ─────────────────────────────────────────────────────────────

describe('absoluteUrl', () => {
  it('retourne le chemin tel quel si window est indéfini', () => {
    const saved = globalThis.window
    // @ts-expect-error
    delete globalThis.window
    expect(absoluteUrl('/posts/abc')).toBe('/posts/abc')
    globalThis.window = saved
  })

  it('construit une URL absolue avec window.location.origin', () => {
    globalThis.window = { location: { origin: 'http://localhost:3000' } } as unknown as Window & typeof globalThis
    expect(absoluteUrl('/posts/abc')).toBe('http://localhost:3000/posts/abc')
  })
})

// ─── canNativeShare ───────────────────────────────────────────────────────────

describe('canNativeShare', () => {
  it('retourne false si navigator.share est absent', () => {
    expect(canNativeShare()).toBe(false)
  })
})

// ─── extractSharedRef ─────────────────────────────────────────────────────────

describe('extractSharedRef', () => {
  it('extrait un lien de post', () => {
    const ref = extractSharedRef('Regarde ça : https://breezy.app/posts/abc123 !')
    expect(ref).toEqual({ kind: 'post', id: 'abc123' })
  })

  it('extrait un lien de profil', () => {
    const ref = extractSharedRef('Suis https://breezy.app/profil/alice.')
    expect(ref).toEqual({ kind: 'profile', id: 'alice' })
  })

  it('retourne null si aucun lien Breezy', () => {
    expect(extractSharedRef('Rien à voir ici')).toBeNull()
    expect(extractSharedRef('https://google.com/search')).toBeNull()
  })

  it('ignore la ponctuation finale', () => {
    const ref = extractSharedRef('Lien: https://breezy.app/posts/xyz.')
    expect(ref).toEqual({ kind: 'post', id: 'xyz' })
  })

  it("décode les caractères encodés dans l'id", () => {
    const ref = extractSharedRef('https://breezy.app/profil/alice%20bob')
    expect(ref).toEqual({ kind: 'profile', id: 'alice bob' })
  })

  it('retourne null pour une chaîne vide', () => {
    expect(extractSharedRef('')).toBeNull()
  })
})

// ─── getRecentShareTargets / recordRecentShareTarget ─────────────────────────

const mockTarget = (id: string): RecentShareTarget => ({
  id,
  username: `user${id}`,
  displayName: `User ${id}`,
  avatarUrl: '',
})

describe('getRecentShareTargets / recordRecentShareTarget', () => {
  beforeEach(() => {
    const store: Record<string, string> = {}
    globalThis.window = {
      localStorage: {
        getItem: (k: string) => store[k] ?? null,
        setItem: (k: string, v: string) => { store[k] = v },
        removeItem: (k: string) => { delete store[k] },
      },
    } as unknown as Window & typeof globalThis
  })

  it('retourne un tableau vide au départ', () => {
    expect(getRecentShareTargets()).toEqual([])
  })

  it('enregistre un destinataire', () => {
    recordRecentShareTarget(mockTarget('1'))
    expect(getRecentShareTargets()).toHaveLength(1)
    expect(getRecentShareTargets()[0].id).toBe('1')
  })

  it("déplace en tête lors d'un doublon", () => {
    recordRecentShareTarget(mockTarget('1'))
    recordRecentShareTarget(mockTarget('2'))
    recordRecentShareTarget(mockTarget('1'))
    const targets = getRecentShareTargets()
    expect(targets[0].id).toBe('1')
    expect(targets).toHaveLength(2)
  })

  it('respecte la limite de 10', () => {
    for (let i = 0; i < 12; i++) recordRecentShareTarget(mockTarget(String(i)))
    expect(getRecentShareTargets()).toHaveLength(10)
  })
})
