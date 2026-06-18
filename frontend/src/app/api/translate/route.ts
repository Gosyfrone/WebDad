import { NextRequest, NextResponse } from 'next/server'

type TranslateRequest = {
  text?: string
  targetLanguage?: string
}

type LibreTranslatePayload = {
  translatedText?: string
  detectedLanguage?: {
    language?: string
  }
  error?: string
}

const DEFAULT_TRANSLATION_API_URL = 'https://libretranslate.de/translate'
const MAX_TEXT_LENGTH = 5_000
const TRANSLATION_TIMEOUT_MS = 8_000

// BRZ-006 : throttle par IP. La traduction relaie vers une API externe
// (coût/abus potentiel) sans authentification. Limiteur en mémoire (fenêtre
// glissante) — suffisant pour le serveur Next standalone (process unique).
const RATE_LIMIT = 30 // requêtes
const RATE_WINDOW_MS = 60_000 // par minute, par IP
const hits = new Map<string, { count: number; resetAt: number }>()

function clientIp(request: NextRequest): string {
  const xff = request.headers.get('x-forwarded-for')
  if (xff) return xff.split(',')[0]!.trim()
  return request.headers.get('x-real-ip') ?? 'unknown'
}

function rateLimited(ip: string): boolean {
  const now = Date.now()
  const entry = hits.get(ip)
  if (!entry || now > entry.resetAt) {
    hits.set(ip, { count: 1, resetAt: now + RATE_WINDOW_MS })
    if (hits.size > 10_000) {
      // Garde-fou anti-fuite mémoire : purge les entrées expirées.
      for (const [k, v] of hits) if (now > v.resetAt) hits.delete(k)
    }
    return false
  }
  if (entry.count >= RATE_LIMIT) return true
  entry.count++
  return false
}

export async function POST(request: NextRequest) {
  if (rateLimited(clientIp(request))) {
    return NextResponse.json(
      { error: 'Trop de requêtes de traduction. Réessaie dans un instant.' },
      { status: 429 },
    )
  }

  let body: TranslateRequest

  try {
    body = await request.json()
  } catch {
    return NextResponse.json(
      { error: 'Le corps de la requête est invalide.' },
      { status: 400 },
    )
  }

  const text = body.text?.trim() ?? ''
  const targetLanguage = normalizeLanguage(body.targetLanguage)

  if (!text) {
    return NextResponse.json({ error: 'Le texte à traduire est requis.' }, { status: 400 })
  }

  if (text.length > MAX_TEXT_LENGTH) {
    return NextResponse.json(
      { error: `Le texte à traduire ne doit pas dépasser ${MAX_TEXT_LENGTH} caractères.` },
      { status: 413 },
    )
  }

  if (!targetLanguage) {
    return NextResponse.json({ error: 'La langue cible est invalide.' }, { status: 400 })
  }

  const translation = await translateText(text, targetLanguage)

  if (!translation?.translatedText) {
    return NextResponse.json(
      { error: 'Impossible de traduire ce texte pour le moment.' },
      { status: 502 },
    )
  }

  return NextResponse.json(translation)
}

async function translateText(
  text: string,
  targetLanguage: string,
): Promise<{ translatedText: string; detectedSourceLanguage: string } | null> {
  const usesConfiguredLibreTranslate =
    Boolean(process.env.TRANSLATION_API_KEY) ||
    Boolean(process.env.TRANSLATION_API_URL && process.env.TRANSLATION_API_URL !== DEFAULT_TRANSLATION_API_URL)

  if (usesConfiguredLibreTranslate) {
    return (
      (await translateWithLibreTranslate(text, targetLanguage)) ??
      (await translateWithGooglePublic(text, targetLanguage))
    )
  }

  return (
    (await translateWithGooglePublic(text, targetLanguage)) ??
    (await translateWithLibreTranslate(text, targetLanguage))
  )
}

async function translateWithLibreTranslate(
  text: string,
  targetLanguage: string,
): Promise<{ translatedText: string; detectedSourceLanguage: string } | null> {
  const apiUrl = process.env.TRANSLATION_API_URL ?? DEFAULT_TRANSLATION_API_URL
  const apiKey = process.env.TRANSLATION_API_KEY

  const upstreamResponse = await fetchWithTimeout(apiUrl, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      q: text,
      source: 'auto',
      target: targetLanguage,
      format: 'text',
      ...(apiKey ? { api_key: apiKey } : {}),
    }),
  }).catch(() => null)

  if (!upstreamResponse?.ok) return null

  const payload = (await upstreamResponse.json().catch(() => null)) as
    | LibreTranslatePayload
    | null
  const translatedText = payload?.translatedText?.trim() ?? ''

  if (!translatedText) return null

  return {
    translatedText,
    detectedSourceLanguage: normalizeLanguage(payload?.detectedLanguage?.language),
  }
}

async function translateWithGooglePublic(
  text: string,
  targetLanguage: string,
): Promise<{ translatedText: string; detectedSourceLanguage: string } | null> {
  const url = new URL('https://translate.googleapis.com/translate_a/single')
  url.searchParams.set('client', 'gtx')
  url.searchParams.set('sl', 'auto')
  url.searchParams.set('tl', targetLanguage)
  url.searchParams.set('dt', 't')
  url.searchParams.set('q', text)

  const upstreamResponse = await fetchWithTimeout(url.toString()).catch(() => null)
  if (!upstreamResponse?.ok) return null

  const payload = (await upstreamResponse.json().catch(() => null)) as unknown
  const translatedText = parseGoogleTranslatedText(payload)
  if (!translatedText) return null

  return {
    translatedText,
    detectedSourceLanguage: parseGoogleDetectedLanguage(payload),
  }
}

function fetchWithTimeout(input: string, init: RequestInit = {}): Promise<Response> {
  const controller = new AbortController()
  const timeout = setTimeout(() => controller.abort(), TRANSLATION_TIMEOUT_MS)

  return fetch(input, { ...init, signal: controller.signal }).finally(() => {
    clearTimeout(timeout)
  })
}

function parseGoogleTranslatedText(payload: unknown): string {
  if (!Array.isArray(payload) || !Array.isArray(payload[0])) return ''

  return payload[0]
    .map((chunk) => (Array.isArray(chunk) && typeof chunk[0] === 'string' ? chunk[0] : ''))
    .join('')
    .trim()
}

function parseGoogleDetectedLanguage(payload: unknown): string {
  if (!Array.isArray(payload)) return ''
  return typeof payload[2] === 'string' ? normalizeLanguage(payload[2]) : ''
}

function normalizeLanguage(language?: string | null): string {
  return (language ?? '').trim().toLowerCase().split('-')[0] ?? ''
}
