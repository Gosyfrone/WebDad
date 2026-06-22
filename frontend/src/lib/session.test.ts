import { describe, expect, it, beforeEach } from 'vitest'

import {
  mapRole,
  decodeClaims,
  readSession,
  currentUserId,
  isAdmin,
  SESSION_CHANGED_EVENT,
  notifySessionChanged,
} from '@/lib/session'

// ─── helpers ─────────────────────────────────────────────────────────────────

function b64url(obj: object): string {
  const json = JSON.stringify(obj)
  // base64url (pas de +/= : Buffer→base64→replace)
  return Buffer.from(json).toString('base64').replace(/\+/g, '-').replace(/\//g, '_').replace(/=/g, '')
}

function makeJWT(claims: object): string {
  const header = b64url({ alg: 'HS256', typ: 'JWT' })
  const payload = b64url(claims)
  return `${header}.${payload}.fakesig`
}

function mockWindow(token: string | null): void {
  const store: Record<string, string> = {}
  if (token) store['breezy-access-token'] = token
  globalThis.window = {
    localStorage: {
      getItem: (k: string) => store[k] ?? null,
      setItem: (k: string, v: string) => { store[k] = v },
      removeItem: (k: string) => { delete store[k] },
    },
    atob: (s: string) => Buffer.from(s, 'base64').toString('utf-8'),
    dispatchEvent: () => true,
  } as unknown as Window & typeof globalThis
}

// ─── mapRole ─────────────────────────────────────────────────────────────────

describe('mapRole', () => {
  it('mappe admin → administrator', () => {
    expect(mapRole('admin')).toBe('administrator')
    expect(mapRole('administrator')).toBe('administrator')
  })

  it('mappe moderator → moderator', () => {
    expect(mapRole('moderator')).toBe('moderator')
  })

  it('mappe tout le reste → user', () => {
    expect(mapRole('user')).toBe('user')
    expect(mapRole('')).toBe('user')
    expect(mapRole(undefined)).toBe('user')
    expect(mapRole('superadmin')).toBe('user')
  })
})

// ─── decodeClaims ─────────────────────────────────────────────────────────────

describe('decodeClaims', () => {
  beforeEach(() => mockWindow(null))

  it('retourne null sans token', () => {
    expect(decodeClaims()).toBeNull()
  })

  it('retourne null avec token malformé', () => {
    mockWindow('not.a.jwt')
    // le payload « a » n'est pas un JSON valide
    expect(decodeClaims()).toBeNull()
  })

  it("décode les claims d'un JWT valide", () => {
    const claims = { user_id: 'u1', email: 'u1@breezy.dev', role: 'user' }
    mockWindow(makeJWT(claims))
    const decoded = decodeClaims()
    expect(decoded?.user_id).toBe('u1')
    expect(decoded?.email).toBe('u1@breezy.dev')
    expect(decoded?.role).toBe('user')
  })

  it('reconnaît must_change_password', () => {
    mockWindow(makeJWT({ user_id: 'u1', must_change_password: true }))
    expect(decodeClaims()?.must_change_password).toBe(true)
  })
})

// ─── readSession ─────────────────────────────────────────────────────────────

describe('readSession', () => {
  it('retourne null sans token', () => {
    mockWindow(null)
    expect(readSession()).toBeNull()
  })

  it('retourne null si user_id absent', () => {
    mockWindow(makeJWT({ email: 'x@y.com' }))
    expect(readSession()).toBeNull()
  })

  it('normalise les claims en session', () => {
    mockWindow(makeJWT({ user_id: 'u2', email: 'u2@breezy.dev', role: 'admin' }))
    const session = readSession()
    expect(session?.userId).toBe('u2')
    expect(session?.role).toBe('administrator')
    expect(session?.mustChangePassword).toBe(false)
  })

  it('reflète must_change_password', () => {
    mockWindow(makeJWT({ user_id: 'u3', must_change_password: true }))
    expect(readSession()?.mustChangePassword).toBe(true)
  })

  it('reflète terms_accepted (true) et le défaut false', () => {
    mockWindow(makeJWT({ user_id: 'u4', terms_accepted: true }))
    expect(readSession()?.termsAccepted).toBe(true)
    mockWindow(makeJWT({ user_id: 'u5' }))
    expect(readSession()?.termsAccepted).toBe(false)
  })
})

// ─── currentUserId / isAdmin ──────────────────────────────────────────────────

describe('currentUserId', () => {
  it('retourne chaîne vide sans session', () => {
    mockWindow(null)
    expect(currentUserId()).toBe('')
  })

  it('retourne le user_id de la session', () => {
    mockWindow(makeJWT({ user_id: 'abc' }))
    expect(currentUserId()).toBe('abc')
  })
})

describe('isAdmin', () => {
  it('retourne false sans session', () => {
    mockWindow(null)
    expect(isAdmin()).toBe(false)
  })

  it('retourne false pour role user', () => {
    mockWindow(makeJWT({ user_id: 'u', role: 'user' }))
    expect(isAdmin()).toBe(false)
  })

  it('retourne true pour role admin', () => {
    mockWindow(makeJWT({ user_id: 'u', role: 'admin' }))
    expect(isAdmin()).toBe(true)
  })
})

// ─── SESSION_CHANGED_EVENT / notifySessionChanged ────────────────────────────

describe('SESSION_CHANGED_EVENT', () => {
  it('est une chaîne non vide', () => {
    expect(typeof SESSION_CHANGED_EVENT).toBe('string')
    expect(SESSION_CHANGED_EVENT.length).toBeGreaterThan(0)
  })
})

describe('notifySessionChanged', () => {
  it('ne plante pas', () => {
    const dispatched: string[] = []
    globalThis.window = {
      dispatchEvent: (e: Event) => { dispatched.push(e.type); return true },
    } as unknown as Window & typeof globalThis
    notifySessionChanged()
    expect(dispatched).toContain(SESSION_CHANGED_EVENT)
  })
})
