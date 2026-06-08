export interface PostTranslation {
  translatedText: string
  detectedSourceLanguage: string
}

interface TranslationPayload {
  translatedText?: string
  detectedSourceLanguage?: string | null
  error?: string
}

const translationCache = new Map<string, Promise<PostTranslation | null>>()
const CLIENT_TRANSLATION_TIMEOUT_MS = 12_000

export function getPageLanguage(): string {
  if (typeof window === 'undefined') return 'fr'

  const htmlLanguage = normalizeLanguage(document.documentElement.lang)
  if (htmlLanguage) return htmlLanguage

  const browserLanguage = normalizeLanguage(window.navigator.language)
  return browserLanguage || 'fr'
}

export function translatePostContent(
  postId: string,
  text: string,
  targetLanguage: string,
): Promise<PostTranslation | null> {
  const normalizedTarget = normalizeLanguage(targetLanguage)
  const normalizedText = text.trim()

  if (!normalizedText || normalizedText.length < 12 || !normalizedTarget) {
    return Promise.resolve(null)
  }

  const cacheKey = `${postId}:${normalizedTarget}:${hashText(normalizedText)}`
  const cached = translationCache.get(cacheKey)
  if (cached) return cached

  const promise = requestTranslation(normalizedText, normalizedTarget).then((result) => {
    if (!result) translationCache.delete(cacheKey)
    return result
  })
  translationCache.set(cacheKey, promise)
  return promise
}

async function requestTranslation(
  text: string,
  targetLanguage: string,
): Promise<PostTranslation | null> {
  const controller = new AbortController()
  const timeout = window.setTimeout(
    () => controller.abort(),
    CLIENT_TRANSLATION_TIMEOUT_MS,
  )

  try {
    const res = await fetch('/api/translate', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ text, targetLanguage }),
      signal: controller.signal,
    })

    if (!res.ok) return null

    const payload = (await res.json().catch(() => null)) as TranslationPayload | null
    const translatedText = payload?.translatedText?.trim() ?? ''
    const detectedSourceLanguage = normalizeLanguage(payload?.detectedSourceLanguage)

    if (!translatedText || textsAreSame(text, translatedText)) return null
    if (detectedSourceLanguage && detectedSourceLanguage === targetLanguage) return null

    return { translatedText, detectedSourceLanguage }
  } catch {
    return null
  } finally {
    window.clearTimeout(timeout)
  }
}

function normalizeLanguage(language?: string | null): string {
  return (language ?? '').trim().toLowerCase().split('-')[0] ?? ''
}

function textsAreSame(a: string, b: string): boolean {
  return a.trim().replace(/\s+/g, ' ') === b.trim().replace(/\s+/g, ' ')
}

function hashText(text: string): string {
  let hash = 0
  for (let i = 0; i < text.length; i += 1) {
    hash = (hash * 31 + text.charCodeAt(i)) | 0
  }
  return String(hash)
}
