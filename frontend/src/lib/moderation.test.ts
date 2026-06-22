import { afterEach, describe, expect, it, vi } from 'vitest'

const apiFetch = vi.fn()
vi.mock('@/lib/auth-client', () => ({ apiFetch: (...a: unknown[]) => apiFetch(...a) }))
vi.mock('@/lib/media', () => ({ resolveMediaUrl: (u: string | undefined) => (u ? `R:${u}` : '') }))

import * as mod from '@/lib/moderation'

function json(body: unknown, init: { ok?: boolean; status?: number } = {}) {
  return { ok: init.ok ?? true, status: init.status ?? 200, json: () => Promise.resolve(body) }
}
const ok = (data: unknown) => json({ data })
afterEach(() => vi.clearAllMocks())

describe('listDeletedPosts', () => {
  it('mappe et passe tous les filtres', async () => {
    apiFetch.mockResolvedValue(ok([
      { id: 'p1', author_id: 'a', content: 'hi', media: [{ url: 'm.png', type: 'image' }], hidden_by: 'mod', hidden_at: 'h', purge_at: 'pg', created_at: 'c' },
    ]))
    const [p] = await mod.listDeletedPosts({ limit: 10, offset: 2, authorId: 'a', since: 's', until: 'u' })
    expect(p).toEqual({ id: 'p1', authorId: 'a', content: 'hi', media: [{ url: 'R:m.png', type: 'image' }], hiddenBy: 'mod', hiddenAt: 'h', purgeAt: 'pg', createdAt: 'c' })
    const params = new URLSearchParams(String(apiFetch.mock.calls[0][0]).split('?')[1])
    expect(params.get('limit')).toBe('10')
    expect(params.get('author_id')).toBe('a')
    expect(params.get('since')).toBe('s')
    expect(params.get('until')).toBe('u')
  })

  it('défauts : hiddenAt retombe sur created_at, media vide', async () => {
    apiFetch.mockResolvedValue(ok([{ id: 'p2', author_id: 'a', content: '', created_at: 'c' }]))
    const [p] = await mod.listDeletedPosts()
    expect(p.hiddenAt).toBe('c')
    expect(p.media).toEqual([])
    expect(p.hiddenBy).toBe('')
  })

  it('échec → ModerationApiError', async () => {
    apiFetch.mockResolvedValue(json({ error: 'no' }, { ok: false, status: 403 }))
    await expect(mod.listDeletedPosts()).rejects.toMatchObject({ status: 403, name: 'ModerationApiError' })
  })
})

describe('restoreDeletedPost / purgeDeletedPost', () => {
  it('restore POST', async () => {
    apiFetch.mockResolvedValue(json({}, { ok: true }))
    await mod.restoreDeletedPost('p1')
    expect(apiFetch.mock.calls[0]).toEqual(['/posts/p1/restore', { method: 'POST' }])
  })
  it('purge DELETE', async () => {
    apiFetch.mockResolvedValue(json({}, { ok: true }))
    await mod.purgeDeletedPost('p1')
    expect(apiFetch.mock.calls[0]).toEqual(['/posts/p1/purge', { method: 'DELETE' }])
  })
  it('échec → erreur', async () => {
    apiFetch.mockResolvedValue(json({ error: 'x' }, { ok: false, status: 500 }))
    await expect(mod.restoreDeletedPost('p1')).rejects.toMatchObject({ status: 500 })
  })
})
