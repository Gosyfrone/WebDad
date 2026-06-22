import { afterEach, describe, expect, it, vi } from 'vitest'

const apiFetch = vi.fn()
vi.mock('@/lib/auth-client', () => ({ apiFetch: (...a: unknown[]) => apiFetch(...a) }))
vi.mock('@/lib/session', () => ({ mapRole: (r: string) => (r === 'admin' ? 'administrator' : r) }))
const resolveUser = vi.fn(async (id: string) => ({ username: 'u_' + id, displayName: 'D ' + id, avatarUrl: '' }))
vi.mock('@/lib/user-cache', () => ({ resolveUser: (id: string) => resolveUser(id) }))

import * as admin from '@/lib/admin'

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
const starts = (p: string) => (url: string) => url.startsWith(p)

afterEach(() => vi.clearAllMocks())

describe('createAccount (orchestration 3 services)', () => {
  it('crée auth + user + profil et renvoie le username effectif', async () => {
    routes([
      [path('/auth/users'), json({ data: { id: 'id1', email: 'a@b.c' } })],
      [path('/users/admin'), json({ data: { id: 'id1', username: 'al', username_pending: true } })],
      [path('/profils/admin'), json({ data: {} })],
    ])
    const r = await admin.createAccount({ username: 'al', email: 'a@b.c', password: 'pw' })
    expect(r).toEqual({ id: 'id1', email: 'a@b.c', username: 'al', usernamePending: true })
  })

  it('échec de l’étape profil est toléré (best-effort)', async () => {
    routes([
      [path('/auth/users'), json({ data: { id: 'id1', email: 'a@b.c' } })],
      [path('/users/admin'), json({ data: { id: 'id1', username: 'al' } })],
      [path('/profils/admin'), json({}, { ok: false, status: 500 })],
    ])
    const r = await admin.createAccount({ username: 'al', email: 'a@b.c', password: 'pw' })
    expect(r.usernamePending).toBe(false)
  })

  it('échec auth → AdminApiError propagée', async () => {
    routes([[path('/auth/users'), json({ error: 'mail pris' }, { ok: false, status: 409 })]])
    await expect(admin.createAccount({ username: 'al', email: 'a@b.c', password: 'pw' }))
      .rejects.toMatchObject({ status: 409, name: 'AdminApiError' })
  })
})

describe('listAdminUsers', () => {
  it('enrichit chaque ligne auth du décoratif user/profil', async () => {
    routes([[starts('/auth/users'), json({ data: [{ id: 'a', email: 'a@b', role: 'admin', is_active: true, created_at: 'd' }] })]])
    const [u] = await admin.listAdminUsers('query')
    expect(u.role).toBe('administrator')
    expect(u.username).toBe('u_a')
    expect(u.displayName).toBe('D a')
    expect(String(apiFetch.mock.calls[0][0])).toContain('q=query')
  })
})

describe('updateUserRole', () => {
  it('mappe administrator → admin côté back', async () => {
    routes([[path('/auth/users/a/role'), json({}, { ok: true })]])
    await admin.updateUserRole('a', 'administrator')
    expect(JSON.parse(apiFetch.mock.calls[0][1].body)).toEqual({ role: 'admin' })
  })
  it('rôle non-admin inchangé, échec → erreur', async () => {
    routes([[path('/auth/users/a/role'), json({ error: 'no' }, { ok: false, status: 403 })]])
    await expect(admin.updateUserRole('a', 'moderator')).rejects.toMatchObject({ status: 403 })
  })
})

describe('setUserBanned (auth puis user)', () => {
  it('bannit : PATCH auth is_active=false puis user', async () => {
    routes([
      [path('/auth/users/a/status'), json({}, { ok: true })],
      [path('/users/a/status'), json({}, { ok: true })],
    ])
    await admin.setUserBanned('a', true)
    expect(JSON.parse(apiFetch.mock.calls[0][1].body)).toEqual({ is_active: false })
    expect(apiFetch.mock.calls.map((c) => String(c[0]))).toEqual(['/auth/users/a/status', '/users/a/status'])
  })

  it('tolère un 404 côté user-service', async () => {
    routes([
      [path('/auth/users/a/status'), json({}, { ok: true })],
      [path('/users/a/status'), json({}, { ok: false, status: 404 })],
    ])
    await expect(admin.setUserBanned('a', false)).resolves.toBeUndefined()
  })

  it('propage une autre erreur côté user-service', async () => {
    routes([
      [path('/auth/users/a/status'), json({}, { ok: true })],
      [path('/users/a/status'), json({ error: 'x' }, { ok: false, status: 500 })],
    ])
    await expect(admin.setUserBanned('a', true)).rejects.toMatchObject({ status: 500 })
  })

  it('échec auth critique → propagé sans toucher user', async () => {
    routes([[path('/auth/users/a/status'), json({}, { ok: false, status: 403 })]])
    await expect(admin.setUserBanned('a', true)).rejects.toMatchObject({ status: 403 })
    expect(apiFetch.mock.calls.some((c) => String(c[0]) === '/users/a/status')).toBe(false)
  })
})

describe('hardDeleteUser (effacement RGPD)', () => {
  it('parcourt toutes les étapes, 404 toléré', async () => {
    apiFetch.mockResolvedValue(json({}, { ok: true }))
    await expect(admin.hardDeleteUser('a')).resolves.toBeUndefined()
    const urls = apiFetch.mock.calls.map((c) => String(c[0]))
    expect(urls).toEqual([
      '/profils/a', '/posts/by-author/a', '/messages/users/a', '/media/owners/a', '/users/a/hard', '/auth/users/a',
    ])
  })

  it('liste les services en échec (≠404 ou exception)', async () => {
    apiFetch.mockImplementation(async (url: string) => {
      if (url === '/posts/by-author/a') return json({}, { ok: false, status: 500 })
      if (url === '/media/owners/a') throw new Error('net')
      return json({}, { ok: url.includes('/profils') ? false : true, status: url.includes('/profils') ? 404 : 200 })
    })
    await expect(admin.hardDeleteUser('a')).rejects.toThrow(/posts, media/)
  })
})
