import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

const apiFetch = vi.fn()
const isAdmin = vi.fn(() => false)
vi.mock('@/lib/auth-client', () => ({ apiFetch: (...a: unknown[]) => apiFetch(...a) }))
vi.mock('@/lib/session', () => ({ isAdmin: () => isAdmin() }))

import {
  exceedsMediaLimit, uploadedMediaUrl, uploadMedia, uploadEncryptedMedia, fetchMediaBytes,
  MAX_MEDIA_BYTES, type UploadedMedia,
} from '@/lib/media'

function json(body: unknown, init: { ok?: boolean; status?: number } = {}) {
  return { ok: init.ok ?? true, status: init.status ?? 200, json: () => Promise.resolve(body) }
}
beforeEach(() => { apiFetch.mockReset(); isAdmin.mockReturnValue(false) })
afterEach(() => vi.unstubAllGlobals())

describe('exceedsMediaLimit', () => {
  it('non-admin : true au-dessus du cap, false en dessous', () => {
    expect(exceedsMediaLimit(MAX_MEDIA_BYTES + 1)).toBe(true)
    expect(exceedsMediaLimit(MAX_MEDIA_BYTES)).toBe(false)
  })
  it('admin : jamais bloqué', () => {
    isAdmin.mockReturnValue(true)
    expect(exceedsMediaLimit(MAX_MEDIA_BYTES * 10)).toBe(false)
  })
})

describe('uploadedMediaUrl', () => {
  const base = { id: 'm', url: '/media/m', mime: 'image/png', kind: 'image', size: 1 } as UploadedMedia
  it('vidéo → toujours l’original', () => {
    expect(uploadedMediaUrl({ ...base, kind: 'video' })).toBe('/media/m')
  })
  it('préfère la variante demandée', () => {
    const m = { ...base, variants: { large: { url: 'L' } } } as unknown as UploadedMedia
    expect(uploadedMediaUrl(m, 'large')).toBe('L')
  })
  it('repli large → medium → original', () => {
    expect(uploadedMediaUrl({ ...base, variants: { medium: { url: 'M' } } } as unknown as UploadedMedia, 'thumb')).toBe('M')
    expect(uploadedMediaUrl(base, 'thumb')).toBe('/media/m')
  })
})

describe('uploadMedia / uploadEncryptedMedia', () => {
  it('uploadMedia POST multipart et renvoie data', async () => {
    apiFetch.mockResolvedValue(json({ data: { id: 'm1', url: '/media/m1' } }))
    const r = await uploadMedia(new Blob(['x']))
    expect(r.id).toBe('m1')
    const [path, init] = apiFetch.mock.calls[0]
    expect(path).toBe('/media')
    expect(init.method).toBe('POST')
    expect(init.body).toBeInstanceOf(FormData)
  })
  it('uploadMedia échec → message d’erreur', async () => {
    apiFetch.mockResolvedValue(json({ error: 'trop gros' }, { ok: false, status: 413 }))
    await expect(uploadMedia(new Blob(['x']))).rejects.toThrow('trop gros')
  })
  it('uploadMedia échec sans corps → message de repli', async () => {
    apiFetch.mockResolvedValue({ ok: false, status: 500, json: () => Promise.reject(new Error('x')) })
    await expect(uploadMedia(new Blob(['x']))).rejects.toThrow("échec de l'upload (500)")
  })
  it('uploadEncryptedMedia renvoie seulement l’id', async () => {
    apiFetch.mockResolvedValue(json({ data: { id: 'enc1' } }))
    expect(await uploadEncryptedMedia(new Blob(['c']))).toEqual({ id: 'enc1' })
    expect(apiFetch.mock.calls[0][0]).toBe('/media/encrypted')
  })
  it('uploadEncryptedMedia échec → erreur', async () => {
    apiFetch.mockResolvedValue(json({ error: 'ko' }, { ok: false, status: 400 }))
    await expect(uploadEncryptedMedia(new Blob(['c']))).rejects.toThrow('ko')
  })
})

describe('fetchMediaBytes', () => {
  it('télécharge les octets bruts (fetch public, sans Bearer)', async () => {
    const buf = new Uint8Array([1, 2, 3]).buffer
    vi.stubGlobal('fetch', vi.fn(async () => ({ ok: true, arrayBuffer: async () => buf })))
    const bytes = await fetchMediaBytes('id1')
    expect([...bytes]).toEqual([1, 2, 3])
  })
  it('média introuvable → erreur', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => ({ ok: false, status: 404 })))
    await expect(fetchMediaBytes('id1')).rejects.toThrow('média introuvable (404)')
  })
})
