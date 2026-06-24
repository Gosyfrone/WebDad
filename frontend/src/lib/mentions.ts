/**
 * Outils partagés pour les mentions @handle (posts, commentaires, messages).
 *
 * Règles alignées sur le back (`post-service/internal/notifier`) : un handle =
 * lettres/chiffres/underscore avec un point INTERNE autorisé (≥ 3 caractères,
 * premier et dernier = caractère-mot → un point final de ponctuation est exclu),
 * précédé d'un début de texte ou d'un caractère NON-mot (évite de capturer la
 * partie locale d'une adresse e-mail `jean@exemple.com`).
 *
 * Module PUR (aucun import client) → testable et réutilisable côté rendu comme
 * côté détection (calcul des ids mentionnés, « vous a mentionné »).
 */

import type { UserCertification } from '@/types'

/** Candidat proposé dans la pop-up d'autocomplétion. */
export interface MentionCandidate {
  id: string
  username: string
  displayName: string
  avatarUrl: string
  certification: UserCertification
}

/** Segment de texte tokenisé : texte brut ou mention cliquable. */
export type MentionSegment =
  | { type: 'text'; text: string }
  | { type: 'mention'; handle: string; raw: string }

// Frontière + handle (mêmes règles que le back). Le groupe 1 capture la
// frontière consommée (début → '' ; sinon un caractère non-mot), le groupe 2 le
// handle. `g` pour itérer ; on recrée le RegExp à chaque appel (lastIndex).
function mentionRegex(): RegExp {
  return /(^|[^\w@])@(\w[\w.]+\w)/g
}

/**
 * Découpe un texte en segments texte / mention. Conserve le texte exact
 * (frontières comprises) pour un rendu fidèle. Fonction PURE.
 */
export function parseMentionSegments(text: string): MentionSegment[] {
  const segments: MentionSegment[] = []
  const re = mentionRegex()
  let last = 0
  let m: RegExpExecArray | null
  while ((m = re.exec(text)) !== null) {
    const boundary = m[1]
    const handle = m[2]
    const atIndex = m.index + boundary.length // position du « @ »
    if (atIndex > last) {
      segments.push({ type: 'text', text: text.slice(last, atIndex) })
    }
    segments.push({ type: 'mention', handle, raw: '@' + handle })
    last = re.lastIndex
  }
  if (last < text.length) {
    segments.push({ type: 'text', text: text.slice(last) })
  }
  return segments
}

/**
 * Handles mentionnés dans un texte, dédupliqués et en minuscules. Fonction PURE.
 */
export function extractMentionHandles(text: string): string[] {
  const re = mentionRegex()
  const seen = new Set<string>()
  let m: RegExpExecArray | null
  while ((m = re.exec(text)) !== null) {
    seen.add(m[2].toLowerCase())
  }
  return [...seen]
}

/**
 * Vrai si `text` mentionne `username` (insensible à la casse). Sert au libellé
 * « X vous a mentionné » de l'aperçu de conversation. Fonction PURE.
 */
export function textMentionsUser(text: string, username: string): boolean {
  if (!username) return false
  const target = username.toLowerCase()
  return extractMentionHandles(text).includes(target)
}

/** Contexte de saisie d'une mention en cours (token sous le curseur). */
export interface MentionTypingContext {
  /** Texte partiel saisi après le « @ » (peut être vide). */
  query: string
  /** Index du « @ » dans la valeur. */
  atIndex: number
  /** Position du curseur (fin du token). */
  caretEnd: number
}

/**
 * Détecte si le curseur est dans une mention en cours de frappe. Renvoie le
 * token (query + bornes) ou null. Plus permissif que `parseMentionSegments`
 * (accepte un handle partiel, même vide → ouvre la pop-up dès le « @ »).
 * Fonction PURE.
 */
export function detectMentionTyping(value: string, caret: number): MentionTypingContext | null {
  const before = value.slice(0, caret)
  const m = before.match(/(?:^|[^\w@])@([\w.]{0,50})$/)
  if (!m) return null
  const query = m[1]
  return { query, atIndex: caret - query.length - 1, caretEnd: caret }
}

/**
 * Remplace le token de mention par `@username ` et renvoie la nouvelle valeur +
 * la position du curseur. Fonction PURE.
 */
export function applyMention(
  value: string,
  ctx: MentionTypingContext,
  username: string,
): { value: string; caret: number } {
  const inserted = '@' + username + ' '
  const next = value.slice(0, ctx.atIndex) + inserted + value.slice(ctx.caretEnd)
  return { value: next, caret: ctx.atIndex + inserted.length }
}
