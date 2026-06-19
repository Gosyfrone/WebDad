import { describe, expect, it } from 'vitest'

import {
  MAX_MEDIA_BYTES,
  MAX_MEDIA_MB,
  mediaUrl,
  resolveMediaUrl,
  toStoredMedia,
} from '@/lib/media'
import { API_URL } from '@/lib/config'

// ─── constantes ──────────────────────────────────────────────────────────────

describe('MAX_MEDIA_BYTES', () => {
  it('vaut 5 Mo en octets', () => {
    expect(MAX_MEDIA_BYTES).toBe(5 * 1024 * 1024)
  })
})

describe('MAX_MEDIA_MB', () => {
  it('vaut 5', () => {
    expect(MAX_MEDIA_MB).toBe(5)
  })
})

// ─── mediaUrl ────────────────────────────────────────────────────────────────

describe('mediaUrl', () => {
  it('construit une URL absolue vers /media/<id>', () => {
    const url = mediaUrl('abc123')
    expect(url).toBe(`${API_URL}/media/abc123`)
  })

  it('préfixe toujours API_URL', () => {
    expect(mediaUrl('x').startsWith(API_URL)).toBe(true)
  })
})

// ─── resolveMediaUrl ─────────────────────────────────────────────────────────

describe('resolveMediaUrl', () => {
  it('retourne chaîne vide pour valeur falsy', () => {
    expect(resolveMediaUrl(null)).toBe('')
    expect(resolveMediaUrl(undefined)).toBe('')
    expect(resolveMediaUrl('')).toBe('')
  })

  it('laisse passer les URL http(s)', () => {
    expect(resolveMediaUrl('https://cdn.example.com/img.png')).toBe('https://cdn.example.com/img.png')
  })

  it('laisse passer les data:', () => {
    expect(resolveMediaUrl('data:image/png;base64,abc')).toBe('data:image/png;base64,abc')
  })

  it('laisse passer les blob:', () => {
    expect(resolveMediaUrl('blob:http://localhost/123')).toBe('blob:http://localhost/123')
  })

  it('préfixe /media/<id> si chemin absolu', () => {
    expect(resolveMediaUrl('/media/xyz')).toBe(`${API_URL}/media/xyz`)
  })

  it('ajoute /media/ si id seul (sans slash)', () => {
    expect(resolveMediaUrl('xyz')).toBe(`${API_URL}/media/xyz`)
  })
})

// ─── toStoredMedia ────────────────────────────────────────────────────────────

describe('toStoredMedia', () => {
  it('retourne chaîne vide pour valeur falsy', () => {
    expect(toStoredMedia(null)).toBe('')
    expect(toStoredMedia(undefined)).toBe('')
    expect(toStoredMedia('')).toBe('')
  })

  it('extrait le chemin relatif si URL gateway', () => {
    const stored = toStoredMedia(`${API_URL}/media/abc`)
    expect(stored).toBe('/media/abc')
  })

  it('laisse passer les URL externes', () => {
    const ext = 'https://cdn.example.com/img.png'
    expect(toStoredMedia(ext)).toBe(ext)
  })

  it('est idempotent sur un chemin déjà relatif', () => {
    // un chemin /media/x n'est pas préfixé API_URL → passe tel quel
    expect(toStoredMedia('/media/abc')).toBe('/media/abc')
  })
})
