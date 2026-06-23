import { afterEach, describe, expect, it, vi } from 'vitest'

const apiFetch = vi.fn()
vi.mock('@/lib/auth-client', () => ({ apiFetch: (...a: unknown[]) => apiFetch(...a) }))
vi.mock('@/lib/media', () => ({ resolveMediaUrl: (u: string | undefined) => (u ? `R:${u}` : '') }))

import { resolveUser, resolveUsers } from '@/lib/user-cache'

function json(body: unknown, init: { ok?: boolean; status?: number } = {}) {
  return { ok: init.ok ?? true, status: init.status ?? 200, json: () => Promise.resolve(body) }
}
function routes(table: Array<[(url: string) => boolean, unknown]>) {
  apiFetch.mockImplementation(async (url: string) => {
    for (const [match, res] of table) if (match(url)) return res
    return json({}, { ok: false, status: 404 })
  })
}
const path = (p: string) => (url: string) => url.split('?')[0] === p
afterEach(() => vi.clearAllMocks())

describe('resolveUser (mémoïsé)', () => {
  it('croise user + profil et résout l’avatar', async () => {
    routes([
      [path('/users/uA'), json({ data: { id: 'uA', username: 'al' } })],
      [path('/profils/uA'), json({ data: { user_id: 'uA', display_name: 'Alice', avatar_url: 'a.png' } })],
    ])
    expect(await resolveUser('uA')).toEqual({ id: 'uA', username: 'al', displayName: 'Alice', avatarUrl: 'R:a.png' })
  })

  it('repli sur le username puis libellé si profil/user absents', async () => {
    routes([[path('/users/uB'), json({ data: { id: 'uB', username: 'bob' } })]]) // profil 404
    expect((await resolveUser('uB')).displayName).toBe('bob')
    routes([]) // tout 404
    expect((await resolveUser('uC')).displayName).toBe('Utilisateur')
  })

  it('mémoïse : un 2e appel ne refait pas de requête', async () => {
    routes([
      [path('/users/uD'), json({ data: { id: 'uD', username: 'd' } })],
      [path('/profils/uD'), json({ data: { user_id: 'uD', display_name: 'D', avatar_url: '' } })],
    ])
    await resolveUser('uD')
    const before = apiFetch.mock.calls.length
    await resolveUser('uD')
    expect(apiFetch.mock.calls.length).toBe(before)
  })
})

describe('resolveUsers', () => {
  it('déduplique les ids', async () => {
    routes([
      [path('/users/uE'), json({ data: { id: 'uE', username: 'e' } })],
      [path('/profils/uE'), json({ data: { user_id: 'uE', display_name: 'E', avatar_url: '' } })],
    ])
    const list = await resolveUsers(['uE', 'uE'])
    expect(list).toHaveLength(1)
    expect(list[0].username).toBe('e')
  })
})
