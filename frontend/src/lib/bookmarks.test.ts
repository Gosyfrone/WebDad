import { afterEach, describe, expect, it, vi } from 'vitest'

const apiFetch = vi.fn()
vi.mock('@/lib/auth-client', () => ({ apiFetch: (...a: unknown[]) => apiFetch(...a) }))
// `mapPosts` est testé dans posts.ts ; on l'isole pour ne tester que bookmarks.
vi.mock('@/lib/posts', () => ({
  mapPosts: async (raw: unknown[]) => (raw ?? []).map((p) => ({ ...(p as object), mapped: true })),
  PostApiError: class extends Error {
    status: number
    constructor(m: string, s: number) { super(m); this.status = s }
  },
}))

import * as bm from '@/lib/bookmarks'

function json(body: unknown, init: { ok?: boolean; status?: number } = {}) {
  return { ok: init.ok ?? true, status: init.status ?? 200, json: () => Promise.resolve(body) }
}
const ok = (data: unknown) => json({ data })
afterEach(() => vi.clearAllMocks())

const apiColl = { id: 'c1', name: 'Lus', is_default: true, items_count: 3, created_at: 'd' }

describe('collections', () => {
  it('listCollections mappe', async () => {
    apiFetch.mockResolvedValue(ok([apiColl]))
    expect(await bm.listCollections()).toEqual([{ id: 'c1', name: 'Lus', isDefault: true, itemsCount: 3, createdAt: 'd' }])
  })
  it('createCollection POST + mappe (défauts)', async () => {
    apiFetch.mockResolvedValue(ok({ id: 'c2', name: 'New', created_at: 'd' }))
    const c = await bm.createCollection('New')
    expect(c).toEqual({ id: 'c2', name: 'New', isDefault: false, itemsCount: 0, createdAt: 'd' })
    expect(JSON.parse(apiFetch.mock.calls[0][1].body)).toEqual({ name: 'New' })
  })
  it('renameCollection PATCH', async () => {
    apiFetch.mockResolvedValue(ok(apiColl))
    await bm.renameCollection('c1', 'Renamed')
    expect(apiFetch.mock.calls[0][1].method).toBe('PATCH')
    expect(JSON.parse(apiFetch.mock.calls[0][1].body)).toEqual({ name: 'Renamed' })
  })
  it('deleteCollection ok / échec', async () => {
    apiFetch.mockResolvedValue(json({}, { ok: true }))
    await expect(bm.deleteCollection('c1')).resolves.toBeUndefined()
    apiFetch.mockResolvedValue(json({}, { ok: false, status: 403 }))
    await expect(bm.deleteCollection('c1')).rejects.toThrow('Suppression impossible')
  })
})

describe('signets', () => {
  it('quickBookmark filed → collection', async () => {
    apiFetch.mockResolvedValue(ok({ status: 'filed', collection: apiColl }))
    const r = await bm.quickBookmark('p1')
    expect(r).toEqual({ status: 'filed', collection: expect.objectContaining({ id: 'c1' }) })
  })
  it('quickBookmark needs_choice → liste de collections', async () => {
    apiFetch.mockResolvedValue(ok({ status: 'needs_choice', collections: [apiColl] }))
    const r = await bm.quickBookmark('p1')
    expect(r.status).toBe('needs_choice')
    if (r.status === 'needs_choice') expect(r.collections).toHaveLength(1)
  })
  it('quickBookmark filed sans collection → repli needs_choice', async () => {
    apiFetch.mockResolvedValue(ok({ status: 'filed' }))
    const r = await bm.quickBookmark('p1')
    expect(r.status).toBe('needs_choice')
  })
  it('addBookmark envoie collection_id et mappe (repli si collection absente)', async () => {
    apiFetch.mockResolvedValue(ok({ status: 'filed' }))
    const c = await bm.addBookmark('p1', 'c9')
    expect(c.id).toBe('c9')
    expect(JSON.parse(apiFetch.mock.calls[0][1].body)).toEqual({ collection_id: 'c9' })
  })
  it('removeBookmark DELETE avec body', async () => {
    apiFetch.mockResolvedValue(json({}, { ok: true }))
    await bm.removeBookmark('p1', 'c1')
    expect(apiFetch.mock.calls[0][1].method).toBe('DELETE')
    expect(JSON.parse(apiFetch.mock.calls[0][1].body)).toEqual({ collection_id: 'c1' })
  })
  it('removeBookmarkEverywhere DELETE sans body, échec → erreur', async () => {
    apiFetch.mockResolvedValue(json({}, { ok: true }))
    await expect(bm.removeBookmarkEverywhere('p1')).resolves.toBeUndefined()
    apiFetch.mockResolvedValue(json({}, { ok: false, status: 500 }))
    await expect(bm.removeBookmarkEverywhere('p1')).rejects.toThrow('Retrait impossible')
  })
  it('getPostCollections renvoie les ids (défaut [])', async () => {
    apiFetch.mockResolvedValue(ok(['c1', 'c2']))
    expect(await bm.getPostCollections('p1')).toEqual(['c1', 'c2'])
    apiFetch.mockResolvedValue(ok(null))
    expect(await bm.getPostCollections('p1')).toEqual([])
  })
})

describe('vues posts (via mapPosts)', () => {
  it('listAllBookmarks pagine et mappe', async () => {
    apiFetch.mockResolvedValue(ok([{ id: 'p1' }]))
    const posts = await bm.listAllBookmarks(10, 5)
    expect(posts[0]).toMatchObject({ id: 'p1', mapped: true })
    expect(String(apiFetch.mock.calls[0][0])).toContain('limit=10&offset=5')
  })
  it('listCollectionPosts cible la collection', async () => {
    apiFetch.mockResolvedValue(ok([{ id: 'p2' }]))
    const posts = await bm.listCollectionPosts('c1')
    expect(posts[0]).toMatchObject({ id: 'p2', mapped: true })
    expect(String(apiFetch.mock.calls[0][0])).toContain('/posts/bookmarks/collections/c1/posts')
  })
})
