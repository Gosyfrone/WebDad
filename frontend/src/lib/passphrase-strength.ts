/**
 * Évaluation heuristique de la robustesse d'une phrase de passe (E2EE messagerie).
 *
 * Pas de dépendance externe (pas de zxcvbn) : on estime une *entropie* à partir
 * de la longueur et de la variété de l'alphabet, avec une pénalité pour les
 * motifs triviaux. Suffisant pour guider l'utilisateur via une barre de
 * complexité ; ce n'est PAS une garantie cryptographique.
 *
 * Module PUR → testable en Node. Le libellé est résolu côté UI via i18n à partir
 * de `level`.
 */

export type StrengthLevel = 'empty' | 'weak' | 'fair' | 'good' | 'strong'

export interface StrengthResult {
  /** Niveau discret (pilote couleur + libellé i18n). */
  level: StrengthLevel
  /** Score normalisé 0–4 (largeur de la barre). */
  score: number
  /** Entropie estimée en bits (indicatif). */
  bits: number
}

/** Taille de l'alphabet « deviné » selon les classes de caractères présentes. */
function alphabetSize(value: string): number {
  let size = 0
  if (/[a-z]/.test(value)) size += 26
  if (/[A-Z]/.test(value)) size += 26
  if (/[0-9]/.test(value)) size += 10
  if (/[^a-zA-Z0-9]/.test(value)) size += 33 // ponctuation/symboles ASCII courants
  return size
}

/** Détecte des motifs triviaux qui effondrent l'entropie réelle. */
function hasTrivialPattern(value: string): boolean {
  if (/(.)\1{2,}/.test(value)) return true // 3+ caractères identiques d'affilée
  const lower = value.toLowerCase()
  const sequences = ['azerty', 'qwerty', '123456', 'abcdef', 'motdepasse', 'password']
  return sequences.some((seq) => lower.includes(seq))
}

/**
 * Estime la robustesse d'une phrase de passe. Renvoie un niveau discret + un
 * score 0–4 + l'entropie estimée.
 */
export function estimateStrength(value: string): StrengthResult {
  if (!value) return { level: 'empty', score: 0, bits: 0 }

  const size = alphabetSize(value)
  let bits = value.length * Math.log2(Math.max(size, 1))
  if (hasTrivialPattern(value)) bits *= 0.5

  let level: StrengthLevel
  let score: number
  if (bits < 36) {
    level = 'weak'
    score = 1
  } else if (bits < 60) {
    level = 'fair'
    score = 2
  } else if (bits < 90) {
    level = 'good'
    score = 3
  } else {
    level = 'strong'
    score = 4
  }

  return { level, score, bits: Math.round(bits) }
}

/** Longueur minimale exigée pour autoriser la définition d'une passphrase. */
export const MIN_PASSPHRASE_LENGTH = 10

/**
 * Une passphrase est acceptable si elle atteint la longueur minimale ET un
 * niveau « fair » au moins (assez complexe mais retenable).
 */
export function isPassphraseAcceptable(value: string): boolean {
  if (value.length < MIN_PASSPHRASE_LENGTH) return false
  return estimateStrength(value).score >= 2
}
