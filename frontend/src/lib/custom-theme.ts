/**
 * Thème personnalisé Breezy : 3 couleurs librement choisies par l'utilisateur,
 * appliquées par-dessus le mode clair/sombre et l'accent (cf. globals.css).
 *
 * Trois cibles (chacune indépendante, `null` = on garde le thème courant) :
 *   - background : fond de l'application  -> --background + --bg-page (solide)
 *   - text       : couleur des écritures  -> --foreground / --card-foreground / --popover-foreground
 *   - primary    : boutons d'action       -> --primary / --ring (+ --primary-foreground auto-contrasté)
 *
 * Persistance localStorage. On y stocke À LA FOIS les hex sources (pour rouvrir
 * l'éditeur sur la bonne couleur) ET la map de variables CSS pré-calculée
 * (`vars`), pour qu'un petit script inline (root layout) puisse l'appliquer
 * AVANT le premier paint sans embarquer la moindre conversion de couleur.
 *
 * `buildCustomThemeVars` est pure (aucun DOM) -> testable. `applyCustomTheme`
 * n'est que l'application au documentElement de ce qu'elle calcule.
 */

import { contrastingTextTriplet, hexToHslTriplet } from '@/lib/color'

/** Une cible personnalisable du thème. */
export type CustomThemeTarget = 'background' | 'text' | 'primary'

export const CUSTOM_THEME_TARGETS: CustomThemeTarget[] = ['background', 'text', 'primary']

/** Couleurs choisies (hex « #rrggbb »), `null` = non personnalisé. */
export interface CustomTheme {
  background: string | null
  text: string | null
  primary: string | null
}

export const EMPTY_CUSTOM_THEME: CustomTheme = {
  background: null,
  text: null,
  primary: null,
}

/** Clé localStorage de persistance. */
export const CUSTOM_THEME_STORAGE_KEY = 'breezy-custom-theme'

/** Variables CSS pilotées par chaque cible (pour set ET removeProperty). */
const TARGET_VARS: Record<CustomThemeTarget, string[]> = {
  // Le fond couvre aussi les SURFACES (colonne centrale, header de feed, cartes
  // verre) pour que « changer le fond » repeigne tout l'espace de lecture.
  background: [
    '--background',
    '--bg-page',
    '--bg-glow-1',
    '--bg-glow-2',
    '--panel',
    '--panel-y',
    '--column',
    '--glass',
    '--glass-strong',
  ],
  text: ['--foreground', '--card-foreground', '--popover-foreground'],
  // Le primaire couvre aussi le dégradé de marque des boutons d'action.
  primary: [
    '--primary',
    '--ring',
    '--primary-foreground',
    '--brand-from',
    '--brand-via',
    '--brand-to',
  ],
}

/** Toutes les variables CSS que la feature est susceptible de surcharger. */
export const ALL_CUSTOM_THEME_VARS: string[] = Object.values(TARGET_VARS).flat()

/**
 * Calcule la map de variables CSS à poser pour un thème donné (fonction pure).
 * Les hex invalides sont ignorés. Une cible `null` n'émet aucune variable
 * (la valeur de la feuille de style — donc le mode clair/sombre — reste active).
 */
export function buildCustomThemeVars(theme: CustomTheme): Record<string, string> {
  const vars: Record<string, string> = {}

  if (theme.background) {
    const triplet = hexToHslTriplet(theme.background)
    if (triplet) {
      vars['--background'] = triplet
      // Le fond visible de l'app est le dégradé --bg-page : on le rend solide,
      // et on neutralise les voiles superposés pour une couleur unie nette.
      vars['--bg-page'] = theme.background
      vars['--bg-glow-1'] = 'none'
      vars['--bg-glow-2'] = 'none'
      // Les surfaces de lecture (colonne centrale, header de feed, cartes verre)
      // adoptent la même couleur → le fond personnalisé repeint tout l'espace
      // central, pas seulement les marges.
      // --panel / --panel-y sont consommés en `background-image` (dégradé) : un
      // hex brut y serait invalide, on passe donc par un dégradé uni.
      const solidGradient = `linear-gradient(${theme.background}, ${theme.background})`
      vars['--panel'] = solidGradient
      vars['--panel-y'] = solidGradient
      // --column / --glass / --glass-strong sont en `background-color` : hex direct.
      vars['--column'] = theme.background
      vars['--glass'] = theme.background
      vars['--glass-strong'] = theme.background
    }
  }

  if (theme.text) {
    const triplet = hexToHslTriplet(theme.text)
    if (triplet) {
      vars['--foreground'] = triplet
      vars['--card-foreground'] = triplet
      vars['--popover-foreground'] = triplet
    }
  }

  if (theme.primary) {
    const triplet = hexToHslTriplet(theme.primary)
    if (triplet) {
      vars['--primary'] = triplet
      vars['--ring'] = triplet
      // Texte des boutons primaires auto-contrasté pour rester lisible.
      vars['--primary-foreground'] = contrastingTextTriplet(theme.primary)
      // Boutons d'action (Breezer/Suivre/FAB/badges) = dégradé de marque : on
      // replie ses 3 arrêts sur la couleur choisie → boutons unis colorés.
      vars['--brand-from'] = theme.primary
      vars['--brand-via'] = theme.primary
      vars['--brand-to'] = theme.primary
    }
  }

  return vars
}

/** Forme persistée : couleurs sources + variables pré-calculées. */
interface StoredCustomTheme extends CustomTheme {
  vars: Record<string, string>
}

/** Garde de type minimale d'un hex « #rrggbb » ou `null`. */
function isHexOrNull(v: unknown): v is string | null {
  return v === null || (typeof v === 'string' && /^#[0-9a-fA-F]{6}$/.test(v))
}

/** Lit le thème personnalisé depuis localStorage (sans planter côté serveur). */
export function readCustomTheme(): CustomTheme {
  if (typeof window === 'undefined') return { ...EMPTY_CUSTOM_THEME }
  try {
    const raw = window.localStorage.getItem(CUSTOM_THEME_STORAGE_KEY)
    if (!raw) return { ...EMPTY_CUSTOM_THEME }
    const parsed = JSON.parse(raw) as Partial<StoredCustomTheme>
    return {
      background: isHexOrNull(parsed.background) ? parsed.background : null,
      text: isHexOrNull(parsed.text) ? parsed.text : null,
      primary: isHexOrNull(parsed.primary) ? parsed.primary : null,
    }
  } catch {
    return { ...EMPTY_CUSTOM_THEME }
  }
}

/** Persiste le thème (couleurs + vars pré-calculées) dans localStorage. */
export function writeCustomTheme(theme: CustomTheme): void {
  if (typeof window === 'undefined') return
  const empty = !theme.background && !theme.text && !theme.primary
  if (empty) {
    window.localStorage.removeItem(CUSTOM_THEME_STORAGE_KEY)
    return
  }
  const stored: StoredCustomTheme = { ...theme, vars: buildCustomThemeVars(theme) }
  window.localStorage.setItem(CUSTOM_THEME_STORAGE_KEY, JSON.stringify(stored))
}

/**
 * Applique le thème au <html> : on RETIRE d'abord toutes nos surcharges
 * (retour à la feuille de style = mode clair/sombre/accent), puis on repose
 * uniquement les variables calculées. Idempotent.
 */
export function applyCustomTheme(theme: CustomTheme): void {
  if (typeof document === 'undefined') return
  const root = document.documentElement
  for (const name of ALL_CUSTOM_THEME_VARS) root.style.removeProperty(name)
  const vars = buildCustomThemeVars(theme)
  for (const [name, value] of Object.entries(vars)) {
    root.style.setProperty(name, value)
  }
}

/**
 * Snippet exécuté en inline dans le <head> (root layout) AVANT le premier paint :
 * lit les `vars` pré-calculées et les pose, évitant tout flash de couleur.
 * Volontairement sans dépendance (pas de conversion de couleur ici).
 */
export const CUSTOM_THEME_INLINE_SCRIPT = `(function(){try{var s=localStorage.getItem('${CUSTOM_THEME_STORAGE_KEY}');if(!s)return;var v=(JSON.parse(s)||{}).vars||{};var r=document.documentElement;for(var k in v){r.style.setProperty(k,v[k]);}}catch(e){}})();`
