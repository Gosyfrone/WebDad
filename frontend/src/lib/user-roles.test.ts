import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

const apiFetch = vi.fn()
vi.mock('@/lib/auth-client', () => ({
  apiFetch: (...args: unknown[]) => apiFetch(...args),
}))

import {
  getUserRole,
  getUserRoles,
  mapPublicRole,
  dispatchIdentityUpdate,
  subscribeIdentityUpdate,
} from './user-roles'

function json(body: unknown, init: { ok?: boolean; status?: number } = {}) {
  return {
    ok: init.ok ?? true,
    status: init.status ?? 200,
    json: () => Promise.resolve(body),
  }
}

beforeEach(() => {
  apiFetch.mockReset()
})

afterEach(() => {
  apiFetch.mockReset()
})

describe('mapPublicRole', () => {
  it('normalise les rôles backend vers le front', () => {
    expect(mapPublicRole('admin')).toBe('administrator')
    expect(mapPublicRole('administrator')).toBe('administrator')
    expect(mapPublicRole('moderator')).toBe('moderator')
    expect(mapPublicRole('user')).toBe('user')
    expect(mapPublicRole('root')).toBe('user')
  })
})

describe('getUserRoles', () => {
  it('liste vide → map vide sans appel API', async () => {
    expect(await getUserRoles(['', '   '])).toEqual(new Map())
    expect(apiFetch).not.toHaveBeenCalled()
  })

  it('déduplique les ids et mappe la réponse', async () => {
    apiFetch.mockResolvedValue(json({ data: [{ id: 'u1', role: 'admin' }, { id: 'u2', role: 'moderator' }] }))
    const roles = await getUserRoles(['u1', 'u2', 'u1', ''])

    expect(String(apiFetch.mock.calls[0][0])).toContain('ids=u1%2Cu2')
    expect(roles.get('u1')).toBe('administrator')
    expect(roles.get('u2')).toBe('moderator')
  })

  it('retombe sur user si l’API échoue', async () => {
    apiFetch.mockResolvedValue(json({ error: 'nope' }, { ok: false, status: 500 }))
    const roles = await getUserRoles(['u'])
    expect(roles.get('u')).toBe('user')
  })
})

describe('getUserRole + identity event', () => {
  it('résout un rôle seul puis le cache', async () => {
    apiFetch.mockResolvedValue(json({ data: [{ id: 'single', role: 'admin' }] }))

    expect(await getUserRole('single')).toBe('administrator')
    expect(await getUserRole('single')).toBe('administrator')
    expect(apiFetch).toHaveBeenCalledTimes(1)
  })

  it('met à jour le cache même sans fenêtre navigateur', async () => {
    dispatchIdentityUpdate({ userId: 'staff', role: 'moderator', certification: 'political' })

    expect(await getUserRole('staff')).toBe('moderator')
  })

  it('notifie les abonnés quand window est disponible', () => {
    const target = new EventTarget()
    vi.stubGlobal('window', target)
    const seen: unknown[] = []
    const unsubscribe = subscribeIdentityUpdate((event) => seen.push(event))

    dispatchIdentityUpdate({ userId: 'live', certification: 'public_figure' })

    expect(seen).toEqual([{ userId: 'live', certification: 'public_figure' }])
    unsubscribe()
    dispatchIdentityUpdate({ userId: 'live', certification: 'political' })
    expect(seen).toHaveLength(1)
    vi.unstubAllGlobals()
  })
})
