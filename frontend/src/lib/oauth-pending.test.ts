import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { savePendingOAuthSignup, loadPendingOAuthSignup, clearPendingOAuthSignup, OAUTH_PENDING_STORAGE_KEY } from '@/lib/oauth-pending'

let store: Record<string, string>
beforeEach(() => {
  store = {}
  vi.stubGlobal('window', {
    sessionStorage: {
      getItem: (k: string) => store[k] ?? null,
      setItem: (k: string, v: string) => { store[k] = v },
      removeItem: (k: string) => { delete store[k] },
    },
  })
})
afterEach(() => vi.unstubAllGlobals())

describe('oauth-pending', () => {
  it('save → load round-trip', () => {
    savePendingOAuthSignup({ provider: 'google', pendingToken: 'tk', email: 'a@b.c' })
    expect(JSON.parse(store[OAUTH_PENDING_STORAGE_KEY]).provider).toBe('google')
    expect(loadPendingOAuthSignup()).toEqual({ provider: 'google', pendingToken: 'tk', email: 'a@b.c' })
  })
  it('load null si absent', () => {
    expect(loadPendingOAuthSignup()).toBeNull()
  })
  it('load null si champs requis manquants', () => {
    store[OAUTH_PENDING_STORAGE_KEY] = JSON.stringify({ provider: 'google' })
    expect(loadPendingOAuthSignup()).toBeNull()
  })
  it('load comble email manquant par chaîne vide', () => {
    store[OAUTH_PENDING_STORAGE_KEY] = JSON.stringify({ provider: 'g', pendingToken: 't' })
    expect(loadPendingOAuthSignup()).toEqual({ provider: 'g', pendingToken: 't', email: '' })
  })
  it('load null si JSON invalide', () => {
    store[OAUTH_PENDING_STORAGE_KEY] = '{bad'
    expect(loadPendingOAuthSignup()).toBeNull()
  })
  it('clear supprime l’entrée', () => {
    store[OAUTH_PENDING_STORAGE_KEY] = 'x'
    clearPendingOAuthSignup()
    expect(store[OAUTH_PENDING_STORAGE_KEY]).toBeUndefined()
  })
})
