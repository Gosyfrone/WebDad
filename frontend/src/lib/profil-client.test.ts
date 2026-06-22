import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

const apiFetch = vi.fn()
vi.mock('@/lib/auth-client', () => ({ apiFetch: (...a: unknown[]) => apiFetch(...a) }))

const decodeClaims = vi.fn(() => ({ role: 'admin' }) as unknown)
vi.mock('@/lib/session', () => ({
  decodeClaims: () => decodeClaims(),
  mapRole: (r: string) => (r === 'admin' ? 'administrator' : 'user'),
}))

import * as pc from '@/lib/profil-client'
import type { ProfilEditableFields } from '@/types'

function json(body: unknown, init: { ok?: boolean; status?: number } = {}) {
  return { ok: init.ok ?? true, status: init.status ?? 200, json: () => Promise.resolve(body) }
}
function routes(table: Array<[(url: string) => boolean, unknown]>) {
  apiFetch.mockImplementation(async (url: string) => {
    for (const [match, res] of table) if (match(url)) return res
    return json({ error: 'nf' }, { ok: false, status: 404 })
  })
}
const path = (p: string) => (url: string) => url.split('?')[0] === p

const apiUser = { id: 'u1', username: 'al', created_at: 'd', follower_count: 5, following_count: 2 }
const apiProfil = {
  user_id: 'u1', display_name: 'Alice', bio: 'hi', avatar_url: 'http://a', banner_url: 'http://b',
  website: 'w', location: 'l', visibility: 'private', likes_visibility: 'private',
  activity_visibility: 'private', nsfw_enabled: false, is_online: true,
}

beforeEach(() => {
  decodeClaims.mockReturnValue({ role: 'admin' })
  // window pour notify/subscribe
  const listeners = new Map<string, (e: Event) => void>()
  vi.stubGlobal('window', {
    addEventListener: (t: string, h: (e: Event) => void) => listeners.set(t, h),
    removeEventListener: (t: string) => listeners.delete(t),
    dispatchEvent: (e: Event) => listeners.get(e.type)?.(e),
  })
  vi.stubGlobal('CustomEvent', class {
    type: string; detail: unknown
    constructor(type: string, init: { detail: unknown }) { this.type = type; this.detail = init.detail }
  })
})
afterEach(() => {
  apiFetch.mockReset()
  vi.unstubAllGlobals()
})

describe('getMyProfil (fusion user + profil + rôle JWT)', () => {
  it('mappe tous les champs et le rôle admin', async () => {
    routes([
      [path('/users/me'), json({ data: apiUser })],
      [path('/profils/me'), json({ data: apiProfil })],
    ])
    const p = await pc.getMyProfil()
    expect(p.userId).toBe('u1')
    expect(p.displayName).toBe('Alice')
    expect(p.role).toBe('administrator')
    expect(p.visibility).toBe('private')
    expect(p.likesVisibility).toBe('private')
    expect(p.activityVisibility).toBe('private')
    expect(p.nsfwEnabled).toBe(false)
    expect(p.isOnline).toBe(true)
    expect(p.followersCount).toBe(5)
    expect(p.profileExists).toBe(true)
  })

  it('profil 404 → defaults publics, profileExists false', async () => {
    routes([
      [path('/users/me'), json({ data: { id: 'u1', username: 'al', created_at: 'd' } })],
      [path('/profils/me'), json({}, { ok: false, status: 404 })],
    ])
    const p = await pc.getMyProfil()
    expect(p.displayName).toBe('al')
    expect(p.visibility).toBe('public')
    expect(p.nsfwEnabled).toBe(true)
    expect(p.profileExists).toBe(false)
  })
})

describe('getPublicProfil', () => {
  it('résout par username, rôle forcé user', async () => {
    routes([
      [path('/users/by-username/bob'), json({ data: { id: 'u2', username: 'bob', created_at: 'd' } })],
      [path('/profils/u2'), json({ data: { user_id: 'u2', display_name: 'Bob' } })],
    ])
    const p = await pc.getPublicProfil('bob')
    expect(p.username).toBe('bob')
    expect(p.role).toBe('user')
  })
})

describe('fetchApiData (erreurs)', () => {
  it('réponse non-ok → message d’erreur de l’enveloppe', async () => {
    routes([[path('/users/me'), json({ error: 'boom' }, { ok: false, status: 500 })]])
    await expect(pc.getMyProfil()).rejects.toThrow('boom')
  })
  it('payload sans data → réponse invalide', async () => {
    routes([[path('/users/me'), json({})]])
    await expect(pc.getMyProfil()).rejects.toThrow('Réponse API profil invalide.')
  })
})

describe('saveMyProfil', () => {
  const fields: ProfilEditableFields = {
    displayName: 'Al', bio: 'b', avatarUrl: 'http://a', bannerUrl: 'http://b',
    website: 'w', location: 'l', gender: 'male', nationality: 'FR', birthDate: '2000-01-01',
  } as ProfilEditableFields

  it('crée le profil s’il n’existe pas puis PATCH, et notifie', async () => {
    routes([
      [path('/profils'), json({ data: {} })],
      [path('/profils/me'), json({ data: {} })],
      [path('/users/me'), json({ data: apiUser })],
    ])
    let notified = false
    const off = pc.subscribeProfilUpdated(() => { notified = true })
    await pc.saveMyProfil(fields, false)
    const posted = apiFetch.mock.calls.find((c) => c[1]?.method === 'POST')
    expect(posted).toBeTruthy()
    const patch = apiFetch.mock.calls.find((c) => c[1]?.method === 'PATCH')!
    const body = JSON.parse(patch[1].body)
    expect(body.gender).toBe('male')
    expect(body.birth_date).toContain('2000-01-01')
    expect(notified).toBe(true)
    off()
  })

  it('ne crée pas le profil s’il existe déjà', async () => {
    routes([
      [path('/profils/me'), json({ data: {} })],
      [path('/users/me'), json({ data: apiUser })],
    ])
    await pc.saveMyProfil(fields, true)
    expect(apiFetch.mock.calls.some((c) => c[1]?.method === 'POST')).toBe(false)
  })
})

describe('toggles de visibilité', () => {
  beforeEach(() => {
    routes([
      [path('/profils/me'), json({ data: {} })],
      [path('/profils/me/activity'), json({ data: {} })],
      [path('/users/me'), json({ data: apiUser })],
    ])
  })

  it('saveMyLikesVisibility PATCH', async () => {
    await pc.saveMyLikesVisibility('private')
    expect(JSON.parse(apiFetch.mock.calls[0][1].body)).toEqual({ likes_visibility: 'private' })
  })
  it('saveMyVisibility PATCH', async () => {
    await pc.saveMyVisibility('private')
    expect(JSON.parse(apiFetch.mock.calls[0][1].body)).toEqual({ visibility: 'private' })
  })
  it('saveMyActivityVisibility public → marque en ligne', async () => {
    await pc.saveMyActivityVisibility('public')
    expect(apiFetch.mock.calls.some((c) => String(c[0]) === '/profils/me/activity')).toBe(true)
  })
  it('saveMyActivityVisibility private → pas de marquage en ligne', async () => {
    await pc.saveMyActivityVisibility('private')
    expect(apiFetch.mock.calls.some((c) => String(c[0]) === '/profils/me/activity')).toBe(false)
  })
  it('saveMyActivityVisibility avale l’erreur de marquage en ligne (best-effort)', async () => {
    apiFetch.mockImplementation(async (url: string, init?: { method?: string }) => {
      if (url === '/profils/me/activity') throw new Error('net')
      if (url === '/users/me') return json({ data: apiUser })
      return json({ data: {} })
    })
    await expect(pc.saveMyActivityVisibility('public')).resolves.toBeDefined()
  })
  it('saveMyNsfw PATCH', async () => {
    await pc.saveMyNsfw(false)
    expect(JSON.parse(apiFetch.mock.calls[0][1].body)).toEqual({ nsfw_enabled: false })
  })
})
