import { afterEach, describe, expect, it, vi } from 'vitest'

// `searchGifs`/`captureGif` s'appuient sur `apiFetch` (Bearer + refresh). On le
// mocke pour piloter la réponse réseau et inspecter la requête émise.
const apiFetch = vi.fn()
vi.mock('@/lib/auth-client', () => ({
  apiFetch: (...args: unknown[]) => apiFetch(...args),
}))

import { captureGif, searchGifs } from '@/lib/giphy'

function jsonResponse(body: unknown, init: { ok?: boolean; status?: number } = {}) {
  return {
    ok: init.ok ?? true,
    status: init.status ?? 200,
    json: () => Promise.resolve(body),
  }
}

afterEach(() => {
  apiFetch.mockReset()
})

describe('searchGifs', () => {
  it('construit la requête avec limit par défaut et terme nettoyé', async () => {
    apiFetch.mockResolvedValue(jsonResponse({ data: [] }))

    await searchGifs('  cats  ')

    const url = apiFetch.mock.calls[0][0] as string
    const params = new URLSearchParams(url.split('?')[1])
    expect(url.startsWith('/gifs/search?')).toBe(true)
    expect(params.get('limit')).toBe('24')
    expect(params.get('q')).toBe('cats')
  })

  it('omet le paramètre q quand la requête est vide', async () => {
    apiFetch.mockResolvedValue(jsonResponse({ data: [] }))

    await searchGifs('   ')

    const url = apiFetch.mock.calls[0][0] as string
    const params = new URLSearchParams(url.split('?')[1])
    expect(params.has('q')).toBe(false)
  })

  it('honore une limite personnalisée', async () => {
    apiFetch.mockResolvedValue(jsonResponse({ data: [] }))

    await searchGifs('dog', 5)

    const url = apiFetch.mock.calls[0][0] as string
    expect(new URLSearchParams(url.split('?')[1]).get('limit')).toBe('5')
  })

  it('mappe les champs snake_case de l’API vers le modèle interne', async () => {
    apiFetch.mockResolvedValue(
      jsonResponse({
        data: [
          { id: 'g1', title: 'Hello', url: 'u1', preview_url: 'p1', width: 200, height: 100 },
        ],
      }),
    )

    const gifs = await searchGifs('hi')

    expect(gifs).toEqual([
      { id: 'g1', title: 'Hello', url: 'u1', previewUrl: 'p1', width: 200, height: 100 },
    ])
  })

  it('met width/height à 0 quand l’API ne les fournit pas', async () => {
    apiFetch.mockResolvedValue(
      jsonResponse({ data: [{ id: 'g1', title: 't', url: 'u', preview_url: 'p' }] }),
    )

    const [gif] = await searchGifs('hi')

    expect(gif.width).toBe(0)
    expect(gif.height).toBe(0)
  })

  it('retourne un tableau vide quand data est absent', async () => {
    apiFetch.mockResolvedValue(jsonResponse({}))

    expect(await searchGifs('hi')).toEqual([])
  })

  it('lève l’erreur renvoyée par le corps quand la réponse échoue', async () => {
    apiFetch.mockResolvedValue(jsonResponse({ error: 'quota dépassé' }, { ok: false, status: 429 }))

    await expect(searchGifs('hi')).rejects.toThrow('quota dépassé')
  })

  it('lève un message de repli quand le corps d’erreur est illisible', async () => {
    apiFetch.mockResolvedValue({
      ok: false,
      status: 502,
      json: () => Promise.reject(new Error('not json')),
    })

    await expect(searchGifs('hi')).rejects.toThrow('giphy unavailable (502)')
  })
})

describe('captureGif', () => {
  it('POST l’URL giphy et renvoie le média capturé', async () => {
    const media = { id: 'm1', url: '/media/m1' }
    apiFetch.mockResolvedValue(jsonResponse({ data: media }))

    const result = await captureGif('https://giphy.com/x.gif')

    expect(result).toEqual(media)
    const [path, init] = apiFetch.mock.calls[0]
    expect(path).toBe('/gifs/capture')
    expect(init.method).toBe('POST')
    expect(init.headers['Content-Type']).toBe('application/json')
    expect(JSON.parse(init.body)).toEqual({ url: 'https://giphy.com/x.gif' })
  })

  it('lève l’erreur du corps quand la capture échoue', async () => {
    apiFetch.mockResolvedValue(jsonResponse({ error: 'capture KO' }, { ok: false, status: 500 }))

    await expect(captureGif('u')).rejects.toThrow('capture KO')
  })

  it('lève un message de repli quand le corps d’erreur est illisible', async () => {
    apiFetch.mockResolvedValue({
      ok: false,
      status: 503,
      json: () => Promise.reject(new Error('boom')),
    })

    await expect(captureGif('u')).rejects.toThrow('capture gif échouée (503)')
  })
})
