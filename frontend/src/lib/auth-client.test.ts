import { describe, expect, it, beforeEach } from 'vitest'

import { getAccessToken, setAccessToken, clearAccessToken } from '@/lib/auth-client'

// ─── helpers ─────────────────────────────────────────────────────────────────

function mockWindow(initialToken: string | null = null): {
  store: Record<string, string>
  events: string[]
} {
  const store: Record<string, string> = {}
  if (initialToken !== null) store['breezy-access-token'] = initialToken
  const events: string[] = []

  globalThis.window = {
    localStorage: {
      getItem: (k: string) => store[k] ?? null,
      setItem: (k: string, v: string) => { store[k] = v },
      removeItem: (k: string) => { delete store[k] },
    },
    dispatchEvent: (e: Event) => { events.push(e.type); return true },
  } as unknown as Window & typeof globalThis

  return { store, events }
}

beforeEach(() => {
  mockWindow(null)
})

// ─── getAccessToken ───────────────────────────────────────────────────────────

describe('getAccessToken', () => {
  it('retourne null si aucun token', () => {
    mockWindow(null)
    expect(getAccessToken()).toBeNull()
  })

  it('retourne le token stocké', () => {
    mockWindow('tok.test.abc')
    expect(getAccessToken()).toBe('tok.test.abc')
  })

  it('retourne null si window est undefined', () => {
    const saved = globalThis.window
    // @ts-expect-error – simule SSR
    delete globalThis.window
    expect(getAccessToken()).toBeNull()
    globalThis.window = saved
  })
})

// ─── setAccessToken ───────────────────────────────────────────────────────────

describe('setAccessToken', () => {
  it('persiste le token dans localStorage', () => {
    const { store } = mockWindow(null)
    setAccessToken('new.token')
    expect(store['breezy-access-token']).toBe('new.token')
  })

  it('dispatche breezy:session-changed', () => {
    const { events } = mockWindow(null)
    setAccessToken('new.token')
    expect(events).toContain('breezy:session-changed')
  })

  it('ne plante pas si window est undefined', () => {
    const saved = globalThis.window
    // @ts-expect-error – simule SSR
    delete globalThis.window
    expect(() => setAccessToken('t')).not.toThrow()
    globalThis.window = saved
  })
})

// ─── clearAccessToken ─────────────────────────────────────────────────────────

describe('clearAccessToken', () => {
  it('supprime le token de localStorage', () => {
    const { store } = mockWindow('existing.token')
    clearAccessToken()
    expect(store['breezy-access-token']).toBeUndefined()
  })

  it('dispatche breezy:session-changed', () => {
    const { events } = mockWindow('existing.token')
    clearAccessToken()
    expect(events).toContain('breezy:session-changed')
  })

  it('ne plante pas si le token était déjà absent', () => {
    mockWindow(null)
    expect(() => clearAccessToken()).not.toThrow()
  })
})
