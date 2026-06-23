import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

const apiFetch = vi.fn()
vi.mock('@/lib/auth-client', () => ({
  apiFetch: (...a: unknown[]) => apiFetch(...a),
}))

import * as api from '@/lib/api'

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

beforeEach(() => apiFetch.mockReset())
afterEach(() => apiFetch.mockReset())

describe('getMe', () => {
  it('mappe l’utilisateur courant avec défauts', async () => {
    routes([[path('/users/me'), json({ data: { id: 'u1', username: 'al', created_at: 'd' } })]])
    expect(await api.getMe()).toEqual({
      id: 'u1', username: 'al', followersCount: 0, followingCount: 0,
      joinedAt: 'd', usernamePending: false, preferredLocale: null,
    })
  })
  it('mappe les compteurs et la locale fournis', async () => {
    routes([[path('/users/me'), json({ data: { id: 'u1', username: 'al', created_at: 'd', follower_count: 2, following_count: 3, username_pending: true, preferred_locale: 'fr' } })]])
    const me = await api.getMe()
    expect(me.followersCount).toBe(2)
    expect(me.preferredLocale).toBe('fr')
    expect(me.usernamePending).toBe(true)
  })
})

describe('updatePreferredLocale', () => {
  it('PATCH la locale', async () => {
    routes([[path('/users/me'), json({ data: {} })]])
    await api.updatePreferredLocale('en')
    expect(JSON.parse(apiFetch.mock.calls[0][1].body)).toEqual({ preferred_locale: 'en' })
  })
})

describe('updateMyUsername (résultat typé)', () => {
  it.each([
    [200, true, undefined],
    [409, false, 'taken'],
    [429, false, 'cooldown'],
    [400, false, 'invalid'],
    [500, false, 'generic'],
  ])('statut %i', async (status, ok, reason) => {
    routes([[path('/users/me'), json({}, { ok: status === 200, status })]])
    const r = await api.updateMyUsername('x')
    expect(r.ok).toBe(ok)
    if (!ok) expect((r as { reason: string }).reason).toBe(reason)
  })
})

describe('getProfilMe', () => {
  it('404 → null', async () => {
    routes([[path('/profils/me'), json({}, { ok: false, status: 404 })]])
    expect(await api.getProfilMe()).toBeNull()
  })
  it('mappe le décoratif', async () => {
    routes([[path('/profils/me'), json({ data: { user_id: 'u', display_name: 'Al', bio: 'hi', avatar_url: 'http://a', banner_url: 'http://b' } })]])
    expect(await api.getProfilMe()).toEqual({ displayName: 'Al', bio: 'hi', avatarUrl: 'http://a', bannerUrl: 'http://b' })
  })
})

describe('listRelations + enrichFromUser', () => {
  it('enrichit chaque relation du décoratif profil', async () => {
    routes([
      [starts('/users/u1/followers'), json({ data: [{ id: 'a', username: 'aa' }] })],
      [path('/profils/a'), json({ data: { user_id: 'a', display_name: 'Alpha', bio: 'b', avatar_url: 'http://x' } })],
    ])
    const rel = await api.listRelations('u1', 'followers')
    expect(rel[0]).toEqual({ id: 'a', username: 'aa', displayName: 'Alpha', bio: 'b', avatarUrl: 'http://x' })
  })
  it('profil absent → repli sur le username', async () => {
    routes([
      [starts('/users/u1/following'), json({ data: [{ id: 'a', username: 'aa' }] })],
      [path('/profils/a'), json({}, { ok: false, status: 404 })],
    ])
    const rel = await api.listRelations('u1', 'following')
    expect(rel[0].displayName).toBe('aa')
  })
})

describe('getCommonFollowers', () => {
  it('cible soi-même → vide', async () => {
    routes([[path('/users/me'), json({ data: { id: 'me', username: 'me', created_at: 'd' } })]])
    expect(await api.getCommonFollowers('me')).toEqual([])
  })
  it('intersection des abonnés et de mes abonnements, hors moi', async () => {
    routes([
      [path('/users/me'), json({ data: { id: 'me', username: 'me', created_at: 'd' } })],
      [starts('/users/t/followers'), json({ data: [{ id: 'a', username: 'a' }, { id: 'b', username: 'b' }, { id: 'me', username: 'me' }] })],
      [starts('/users/me/following'), json({ data: [{ id: 'a' }] })],
      [starts('/profils/'), json({}, { ok: false, status: 404 })],
    ])
    const common = await api.getCommonFollowers('t')
    expect(common.map((u) => u.id)).toEqual(['a'])
  })
})

describe('searchUsers', () => {
  it('requête vide → []', async () => {
    expect(await api.searchUsers('   ')).toEqual([])
    expect(await api.searchUsers('@  ')).toEqual([])
  })
  it('préfixe @ → recherche par identifiant', async () => {
    routes([
      [starts('/users/search'), json({ data: [{ id: 'a', username: 'al' }] })],
      [starts('/profils/'), json({}, { ok: false, status: 404 })],
    ])
    const r = await api.searchUsers('@al')
    expect(r[0].username).toBe('al')
    expect(String(apiFetch.mock.calls[0][0])).toContain('/users/search?q=al')
  })
  it('sans @ → fusion users+profils, dédoublonnée', async () => {
    routes([
      [starts('/users/search'), json({ data: [{ id: 'a', username: 'al' }] })],
      [starts('/profils/search'), json({ data: [{ user_id: 'a', display_name: 'Dup' }, { user_id: 'b', display_name: 'Bee' }] })],
      [path('/users/b'), json({ data: { id: 'b', username: 'bee' } })],
      [path('/profils/a'), json({}, { ok: false, status: 404 })],
    ])
    const r = await api.searchUsers('a')
    expect(r.map((u) => u.id)).toEqual(['a', 'b'])
  })
  it('profil sans user correspondant → filtré', async () => {
    routes([
      [starts('/users/search'), json({ data: [] })],
      [starts('/profils/search'), json({ data: [{ user_id: 'ghost', display_name: 'G' }] })],
      [path('/users/ghost'), json({}, { ok: false, status: 404 })],
    ])
    expect(await api.searchUsers('g')).toEqual([])
  })
})

describe('résolution par handle / id', () => {
  it('getUserByUsername: vide → null, 404 → null, ok → enrichi', async () => {
    expect(await api.getUserByUsername('  ')).toBeNull()
    routes([[starts('/users/by-username/'), json({}, { ok: false, status: 404 })]])
    expect(await api.getUserByUsername('x')).toBeNull()
    routes([
      [starts('/users/by-username/'), json({ data: { id: 'a', username: 'al' } })],
      [starts('/profils/'), json({}, { ok: false, status: 404 })],
    ])
    expect((await api.getUserByUsername('al'))?.id).toBe('a')
  })
  it('getUserById: vide → null, data absent → null, ok → enrichi', async () => {
    expect(await api.getUserById('')).toBeNull()
    routes([[path('/users/a'), json({ data: null })]])
    expect(await api.getUserById('a')).toBeNull()
    routes([
      [path('/users/a'), json({ data: { id: 'a', username: 'al' } })],
      [starts('/profils/'), json({}, { ok: false, status: 404 })],
    ])
    expect((await api.getUserById('a'))?.username).toBe('al')
  })
})

describe('suggestions + ids', () => {
  it('getSuggestions enrichit', async () => {
    routes([
      [starts('/users/suggestions'), json({ data: [{ id: 'a', username: 'al' }] })],
      [starts('/profils/'), json({}, { ok: false, status: 404 })],
    ])
    expect((await api.getSuggestions())[0].id).toBe('a')
  })
  it('getFollowingIds → Set', async () => {
    routes([[starts('/users/u/following'), json({ data: [{ id: 'a' }, { id: 'b' }] })]])
    expect([...(await api.getFollowingIds('u'))]).toEqual(['a', 'b'])
  })
  it('getPendingFollowRequestIds + getBlockedUserIds', async () => {
    routes([
      [path('/users/me/follow-requests/outgoing'), json({ data: ['a'] })],
      [path('/users/me/blocks'), json({ data: ['b'] })],
    ])
    expect([...(await api.getPendingFollowRequestIds())]).toEqual(['a'])
    expect([...(await api.getBlockedUserIds())]).toEqual(['b'])
  })
})

describe('actions sociales', () => {
  it('follow renvoie le statut (defaut following)', async () => {
    routes([[path('/users/u/follow'), json({ data: { status: 'pending' } })]])
    expect(await api.follow('u')).toBe('pending')
    routes([[path('/users/u/follow'), json({ data: {} })]])
    expect(await api.follow('u')).toBe('following')
  })
  it('follow échec → ApiError', async () => {
    routes([[path('/users/u/follow'), json({}, { ok: false, status: 403 })]])
    await expect(api.follow('u')).rejects.toMatchObject({ status: 403 })
  })
  it.each([
    ['unfollow', () => api.unfollow('u'), 'DELETE', '/users/u/follow'],
    ['blockUser', () => api.blockUser('u'), 'POST', '/users/u/block'],
    ['unblockUser', () => api.unblockUser('u'), 'DELETE', '/users/u/block'],
    ['acceptFollowRequest', () => api.acceptFollowRequest('u'), 'POST', '/users/follow-requests/u/accept'],
    ['rejectFollowRequest', () => api.rejectFollowRequest('u'), 'POST', '/users/follow-requests/u/reject'],
    ['removeFollower', () => api.removeFollower('u'), 'DELETE', '/users/me/followers/u'],
  ])('%s ok puis erreur', async (_n, fn, method, url) => {
    routes([[path(url), json({}, { ok: true })]])
    await expect(fn()).resolves.toBeUndefined()
    expect(apiFetch.mock.calls.at(-1)![1].method).toBe(method)
    routes([[path(url), json({}, { ok: false, status: 500 })]])
    await expect(fn()).rejects.toMatchObject({ status: 500 })
  })
})

describe('ApiError', () => {
  it('porte le statut', () => {
    const e = new api.ApiError('x', 418)
    expect(e.status).toBe(418)
    expect(e.name).toBe('ApiError')
  })
})
