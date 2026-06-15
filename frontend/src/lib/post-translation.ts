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
const MIN_SCRIPT_LETTERS = 4
const MIN_SOURCE_MARKERS = 3

const LANGUAGE_MARKERS: Record<string, Set<string>> = {
  de: new Set([
    'aber',
    'auf',
    'das',
    'der',
    'die',
    'du',
    'ein',
    'eine',
    'es',
    'gut',
    'hallo',
    'ich',
    'ist',
    'mit',
    'nicht',
    'und',
    'von',
    'was',
    'wie',
    'wir',
    'zu',
  ]),
  en: new Set([
    'a',
    'about',
    'am',
    'and',
    'are',
    'as',
    'at',
    'be',
    'because',
    'but',
    'can',
    'do',
    'does',
    'for',
    'from',
    'go',
    'good',
    'have',
    'hello',
    'hey',
    'hi',
    'how',
    'i',
    'in',
    'is',
    'it',
    'like',
    'me',
    'my',
    'name',
    'not',
    'of',
    'on',
    'or',
    'that',
    'the',
    'this',
    'to',
    'want',
    'we',
    'what',
    'when',
    'where',
    'who',
    'why',
    'with',
    'you',
    'your',
  ]),
  fr: new Set([
    'a',
    'ai',
    'alors',
    'au',
    'avec',
    'bien',
    'bonjour',
    'ca',
    'ce',
    'ces',
    'comment',
    'dans',
    'de',
    'des',
    'du',
    'elle',
    'en',
    'est',
    'et',
    'faire',
    'gars',
    'il',
    'ils',
    'je',
    'jsuis',
    'la',
    'le',
    'les',
    'mais',
    'mes',
    'mon',
    'nom',
    'nous',
    'on',
    'ou',
    'pas',
    'pour',
    'que',
    'qui',
    'quoi',
    'salut',
    'suis',
    'trop',
    'sur',
    'ta',
    'te',
    'toi',
    'ton',
    'tu',
    'un',
    'une',
    'va',
    'vous',
  ]),
  es: new Set([
    'a',
    'como',
    'con',
    'de',
    'del',
    'el',
    'en',
    'es',
    'esta',
    'estoy',
    'hola',
    'la',
    'las',
    'los',
    'me',
    'mi',
    'muy',
    'no',
    'para',
    'pero',
    'por',
    'que',
    'si',
    'un',
    'una',
    'y',
    'yo',
  ]),
  it: new Set([
    'a',
    'che',
    'ciao',
    'come',
    'con',
    'di',
    'e',
    'il',
    'io',
    'la',
    'le',
    'mi',
    'molto',
    'non',
    'per',
    'sono',
    'un',
    'una',
  ]),
  nl: new Set([
    'de',
    'een',
    'en',
    'goed',
    'hallo',
    'het',
    'hoe',
    'ik',
    'is',
    'met',
    'niet',
    'op',
    'van',
    'wat',
    'we',
    'zijn',
  ]),
  pt: new Set([
    'a',
    'como',
    'com',
    'de',
    'e',
    'estou',
    'eu',
    'muito',
    'nao',
    'ola',
    'o',
    'os',
    'para',
    'por',
    'que',
    'um',
    'uma',
  ]),
}

const LANGUAGE_SCRIPTS: Record<string, string> = {
  am: 'ethiopic',
  ar: 'arabic',
  bg: 'cyrillic',
  bn: 'bengali',
  el: 'greek',
  fa: 'arabic',
  gu: 'gujarati',
  he: 'hebrew',
  hi: 'devanagari',
  hy: 'armenian',
  ja: 'kana',
  ka: 'georgian',
  km: 'khmer',
  ko: 'hangul',
  lo: 'lao',
  mr: 'devanagari',
  my: 'myanmar',
  pa: 'gurmukhi',
  ru: 'cyrillic',
  ta: 'tamil',
  te: 'telugu',
  th: 'thai',
  uk: 'cyrillic',
  ur: 'arabic',
  zh: 'han',
}

const SCRIPT_TESTS: Record<string, RegExp> = {
  arabic: /\p{Script=Arabic}/u,
  armenian: /\p{Script=Armenian}/u,
  bengali: /\p{Script=Bengali}/u,
  cyrillic: /\p{Script=Cyrillic}/u,
  devanagari: /\p{Script=Devanagari}/u,
  ethiopic: /\p{Script=Ethiopic}/u,
  georgian: /\p{Script=Georgian}/u,
  greek: /\p{Script=Greek}/u,
  gujarati: /\p{Script=Gujarati}/u,
  gurmukhi: /\p{Script=Gurmukhi}/u,
  hangul: /\p{Script=Hangul}/u,
  han: /\p{Script=Han}/u,
  hebrew: /\p{Script=Hebrew}/u,
  khmer: /\p{Script=Khmer}/u,
  kana: /[\p{Script=Hiragana}\p{Script=Katakana}]/u,
  lao: /\p{Script=Lao}/u,
  myanmar: /\p{Script=Myanmar}/u,
  tamil: /\p{Script=Tamil}/u,
  telugu: /\p{Script=Telugu}/u,
  thai: /\p{Script=Thai}/u,
}

const LETTER_RE = /\p{L}/u

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

  if (!isTranslationCandidate(normalizedText, normalizedTarget)) {
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

export function shouldAttemptTranslation(text: string, targetLanguage: string): boolean {
  const target = normalizeLanguage(targetLanguage)
  const scriptDecision = shouldTranslateByScript(text, target)
  if (scriptDecision !== null) return scriptDecision

  const words = extractWords(text)
  if (words.length < 3) return false

  const scores = scoreLanguages(words)
  const targetScore = scores[target] ?? 0
  const strongestForeignScore = Object.entries(scores)
    .filter(([language]) => language !== target)
    .sort((a, b) => b[1] - a[1])
    .at(0)?.[1] ?? 0

  // Veto local uniquement si le texte porte des marqueurs de la langue cible
  // sans signal étranger fort. Sinon le fournisseur, bien plus complet que nos
  // petits lexiques, détecte la source et la réponse est rejetée si source=cible.
  if (
    targetScore > 0 &&
    strongestForeignScore < MIN_SOURCE_MARKERS &&
    targetScore >= strongestForeignScore
  ) {
    return false
  }

  return true
}

export function isTranslationCandidate(text: string, targetLanguage: string): boolean {
  const normalizedTarget = normalizeLanguage(targetLanguage)
  const normalizedText = text.trim()

  return Boolean(normalizedText) &&
    normalizedText.length >= 12 &&
    Boolean(normalizedTarget) &&
    shouldAttemptTranslation(normalizedText, normalizedTarget)
}

function extractWords(text: string): string[] {
  return (text.toLowerCase().match(/[\p{L}]+/gu) ?? [])
    .map((word) => removeDiacritics(word))
    .filter((word) => word.length > 1)
}

function shouldTranslateByScript(text: string, targetLanguage: string): boolean | null {
  const counts = countScriptLetters(text)

  // Han seul = chinois ; la présence significative de kana = japonais.
  // Les séparer évite de bloquer chinois→japonais et japonais→chinois.
  const kanaCount = counts.kana ?? 0
  const hanCount = counts.han ?? 0
  if (kanaCount >= MIN_SCRIPT_LETTERS) return targetLanguage !== 'ja'
  if (hanCount >= MIN_SCRIPT_LETTERS) return targetLanguage !== 'zh'

  const totalScriptLetters = Object.values(counts).reduce((sum, count) => sum + count, 0)
  if (totalScriptLetters < MIN_SCRIPT_LETTERS) return null

  const [dominantScript = '', dominantCount = 0] = Object.entries(counts)
    .sort((a, b) => b[1] - a[1])[0] ?? []
  if (!dominantScript || dominantCount < MIN_SCRIPT_LETTERS) return null

  const dominance = dominantCount / totalScriptLetters
  if (dominance < 0.6) return null

  const targetScript = LANGUAGE_SCRIPTS[targetLanguage]
  return targetScript !== dominantScript
}

function countScriptLetters(text: string): Record<string, number> {
  const counts: Record<string, number> = {}

  for (const char of text) {
    if (!LETTER_RE.test(char)) continue
    for (const [script, pattern] of Object.entries(SCRIPT_TESTS)) {
      if (!pattern.test(char)) continue
      counts[script] = (counts[script] ?? 0) + 1
      break
    }
  }

  return counts
}

function scoreLanguages(words: string[]): Record<string, number> {
  const scores: Record<string, number> = {}
  for (const [language, markers] of Object.entries(LANGUAGE_MARKERS)) {
    scores[language] = words.reduce(
      (score, word) => score + (markers.has(word) ? 1 : 0),
      0,
    )
  }
  return scores
}

function removeDiacritics(value: string): string {
  return value.normalize('NFD').replace(/[\u0300-\u036f]/g, '')
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
