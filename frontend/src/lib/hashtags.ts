/**
 * Outils purs pour l'autocomplétion et le rendu visuel des hashtags.
 *
 * Les règles sont alignées sur le post-service : un hashtag est précédé d'un
 * début de texte ou d'un séparateur, puis contient lettres/chiffres/underscore.
 */

export interface HashtagCandidate {
  tag: string
  count?: number
}

export type HashtagSegment =
  | { type: 'text'; text: string }
  | { type: 'hashtag'; tag: string; raw: string }

export interface HashtagTypingContext {
  query: string
  hashIndex: number
  caretEnd: number
}

function hashtagRegex(): RegExp {
  return /(^|[^\p{L}\p{N}_#])#([\p{L}\p{N}_]{1,64})/gu
}

export function parseHashtagSegments(text: string): HashtagSegment[] {
  const segments: HashtagSegment[] = []
  const re = hashtagRegex()
  let last = 0
  let match: RegExpExecArray | null

  while ((match = re.exec(text)) !== null) {
    const boundary = match[1]
    const tag = match[2]
    const hashIndex = match.index + boundary.length
    if (hashIndex > last) segments.push({ type: 'text', text: text.slice(last, hashIndex) })
    segments.push({ type: 'hashtag', tag, raw: `#${tag}` })
    last = re.lastIndex
  }

  if (last < text.length) segments.push({ type: 'text', text: text.slice(last) })
  return segments
}

export function detectHashtagTyping(value: string, caret: number): HashtagTypingContext | null {
  const before = value.slice(0, caret)
  const match = before.match(/(?:^|[^\p{L}\p{N}_#])#([\p{L}\p{N}_]{0,64})$/u)
  if (!match) return null
  const query = match[1]
  return { query, hashIndex: caret - query.length - 1, caretEnd: caret }
}

export function applyHashtag(
  value: string,
  ctx: HashtagTypingContext,
  tag: string,
): { value: string; caret: number } {
  const clean = tag.trim().replace(/^#/, '')
  const inserted = `#${clean} `
  const next = value.slice(0, ctx.hashIndex) + inserted + value.slice(ctx.caretEnd)
  return { value: next, caret: ctx.hashIndex + inserted.length }
}
