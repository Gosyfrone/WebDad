import { apiFetch } from '@/lib/auth-client'

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
