/**
 * Registre des thèmes Breezy.
 *
 * Deux dimensions indépendantes (cf. src/app/globals.css) :
 *   - MODE   : clair / sombre / système — piloté par next-themes (classe .dark).
 *   - ACCENT : couleur primaire de marque — piloté par l'attribut [data-accent]
 *              posé sur <html>. L'accent par défaut ('blue') est intégré à :root.
 *
 * Ce fichier sert de source de vérité pour un futur sélecteur de thème
 * (settings / menu utilisateur). L'application effective d'un accent se fera via
 * `document.documentElement.dataset.accent = id` (+ persistance localStorage).
 */

/** Modes de luminosité disponibles (valeurs next-themes). */
export const COLOR_MODES = ['light', 'dark', 'system'] as const
export type ColorMode = (typeof COLOR_MODES)[number]
export const DEFAULT_MODE: ColorMode = 'system'

/** Un accent = une couleur primaire de marque sélectionnable. */
export interface Accent {
  id: string
  /** Libellé affiché dans le sélecteur. */
  label: string
  /** Pastille de prévisualisation (couleur CSS). */
  swatch: string
}

/**
 * Accents disponibles. Le premier ('blue') est le défaut (intégré à :root,
 * donc sans attribut data-accent). Pour en ajouter un : créer le bloc CSS
 * correspondant dans globals.css puis l'ajouter ici.
 */
export const ACCENTS: Accent[] = [
  { id: 'pink', label: 'Breezy', swatch: '#e053ff' },
  { id: 'blue', label: 'Bleu', swatch: 'hsl(223 85% 60%)' },
  { id: 'cyan', label: 'Cyan', swatch: 'hsl(199 89% 48%)' },
]

export const DEFAULT_ACCENT = ACCENTS[0].id
export type AccentId = (typeof ACCENTS)[number]['id']
