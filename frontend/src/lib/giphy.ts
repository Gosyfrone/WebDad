import { apiFetch } from '@/lib/auth-client'
import type { UploadedMedia } from '@/lib/media'

export interface GiphyGif {
  id: string
  title: string
  url: string
  previewUrl: string
  width: number
  height: number
}

interface ApiGiphyGif {
  id: string
  title: string
  url: string
  preview_url: string
  width: number
  height: number
}

export async function searchGifs(query: string, limit = 24): Promise<GiphyGif[]> {
  const params = new URLSearchParams()
  params.set('limit', String(limit))
  const q = query.trim()
  if (q) params.set('q', q)

  const res = await apiFetch(`/gifs/search?${params}`)
  if (!res.ok) {
    const body = (await res.json().catch(() => null)) as { error?: string } | null
    throw new Error(body?.error ?? `giphy unavailable (${res.status})`)
  }
  const body = (await res.json()) as { data?: ApiGiphyGif[] }
  return (body.data ?? []).map((gif) => ({
    id: gif.id,
    title: gif.title,
    url: gif.url,
    previewUrl: gif.preview_url,
    width: gif.width ?? 0,
    height: gif.height ?? 0,
  }))
}

/**
 * Capture un GIF GIPHY côté serveur : le media-service télécharge les octets
 * depuis le CDN giphy et les range dans MinIO, puis renvoie un média servi par
 * la gateway (`/media/<id>`). On stocke ce chemin dans le post plutôt que l'URL
 * giphy externe → pas de hotlink (403/404 du CDN), compatible CSP `self`.
 */
export async function captureGif(url: string): Promise<UploadedMedia> {
  const res = await apiFetch('/gifs/capture', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ url }),
  })
  if (!res.ok) {
    const body = (await res.json().catch(() => null)) as { error?: string } | null
    throw new Error(body?.error ?? `capture gif échouée (${res.status})`)
  }
  const body = (await res.json()) as { data: UploadedMedia }
  return body.data
}
