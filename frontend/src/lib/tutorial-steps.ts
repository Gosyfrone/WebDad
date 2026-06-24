import { ROUTES } from '@/lib/routes'

/**
 * Définition (pure, testable) du didacticiel guidé de prise en main.
 *
 * Chaque étape pointe une CIBLE repérée par un attribut `data-tour="<target>"`
 * posé sur l'élément réel. La même valeur est posée sur les variantes desktop
 * (sidebar) ET mobile (tab bar / header) : le composant choisit l'instance
 * VISIBLE à l'écran (cf. tutorial-tooltip). Une étape sans cible visible se rend
 * en infobulle centrée (fallback : sections accessibles via un menu sur mobile).
 *
 * `route` : page sur laquelle l'étape doit se dérouler. À l'entrée dans l'étape,
 * le tour y navigue (router.push) si on n'y est pas déjà — l'utilisateur voit
 * ainsi chaque section. La navigation peut aussi se faire en cliquant la cible.
 */
export type TutorialPlacement = 'top' | 'bottom' | 'left' | 'right' | 'auto'

export interface TutorialStep {
  /** Identifiant stable de l'étape (clé i18n + valeur `data-tour`). */
  id: string
  /** Route où dérouler l'étape (le tour y navigue au besoin). */
  route: string
  /**
   * Sélecteur `data-tour` de la cible à mettre en surbrillance. `undefined` ⇒
   * infobulle centrée (écran d'accueil / de clôture, sans ancre).
   */
  target?: string
  /** Côté préféré pour l'infobulle (sinon placement auto selon l'espace). */
  placement: TutorialPlacement
}

/**
 * Étapes du tour, dans l'ordre. Les clés i18n dérivent de `id` :
 * `tutorial.step.<id>.title` / `tutorial.step.<id>.body`.
 */
export const TUTORIAL_STEPS: TutorialStep[] = [
  { id: 'welcome', route: ROUTES.feed, placement: 'auto' },
  { id: 'feed', route: ROUTES.feed, target: 'nav-feed', placement: 'right' },
  { id: 'explorer', route: ROUTES.explorer, target: 'nav-explorer', placement: 'right' },
  { id: 'notifications', route: ROUTES.notifications, target: 'nav-notifications', placement: 'right' },
  { id: 'messages', route: ROUTES.messages, target: 'nav-messages', placement: 'right' },
  { id: 'bookmarks', route: ROUTES.bookmarks, target: 'nav-bookmarks', placement: 'right' },
  { id: 'profil', route: ROUTES.profil, target: 'nav-profil', placement: 'right' },
  { id: 'compose', route: ROUTES.feed, target: 'compose', placement: 'auto' },
  { id: 'help', route: ROUTES.feed, target: 'help-button', placement: 'auto' },
] as const

export const TUTORIAL_STEP_COUNT = TUTORIAL_STEPS.length

/** Borne un index d'étape dans `[0, dernier]`. */
export function clampStepIndex(index: number): number {
  if (Number.isNaN(index) || index < 0) return 0
  if (index > TUTORIAL_STEPS.length - 1) return TUTORIAL_STEPS.length - 1
  return index
}

/** Clé i18n du titre d'une étape. */
export function stepTitleKey(step: TutorialStep): string {
  return `tutorial.step.${step.id}.title`
}

/** Clé i18n du corps d'une étape. */
export function stepBodyKey(step: TutorialStep): string {
  return `tutorial.step.${step.id}.body`
}

/** Rectangle minimal (sous-ensemble de DOMRect) suffisant au calcul de position. */
export interface Rect {
  top: number
  left: number
  width: number
  height: number
}

export interface Size {
  width: number
  height: number
}

export interface Viewport {
  width: number
  height: number
}

export interface TooltipPosition {
  top: number
  left: number
  /** Côté effectivement retenu (utile pour orienter la flèche). */
  placement: Exclude<TutorialPlacement, 'auto'>
}

/** Marge (px) entre la cible et l'infobulle, et entre l'infobulle et les bords. */
const GAP = 12
const MARGIN = 8

/**
 * Calcule la position de l'infobulle autour d'une cible (pur, donc testable sans
 * DOM). Choisit le côté préféré s'il rentre, sinon le côté offrant le plus
 * d'espace, puis borne la position dans le viewport. `placement: 'auto'` laisse
 * la fonction décider entièrement.
 */
export function computeTooltipPosition(
  target: Rect,
  tooltip: Size,
  viewport: Viewport,
  preferred: TutorialPlacement = 'auto',
): TooltipPosition {
  const space = {
    top: target.top,
    bottom: viewport.height - (target.top + target.height),
    left: target.left,
    right: viewport.width - (target.left + target.width),
  }
  const needV = tooltip.height + GAP
  const needH = tooltip.width + GAP

  const fits = (side: Exclude<TutorialPlacement, 'auto'>): boolean =>
    side === 'top' || side === 'bottom' ? space[side] >= needV : space[side] >= needH

  const order: Exclude<TutorialPlacement, 'auto'>[] =
    preferred !== 'auto' && fits(preferred)
      ? [preferred]
      : (['right', 'left', 'bottom', 'top'] as const)
          .slice()
          .sort((a, b) => space[b] - space[a])

  const placement = order[0]

  let top: number
  let left: number
  switch (placement) {
    case 'top':
      top = target.top - tooltip.height - GAP
      left = target.left + target.width / 2 - tooltip.width / 2
      break
    case 'bottom':
      top = target.top + target.height + GAP
      left = target.left + target.width / 2 - tooltip.width / 2
      break
    case 'left':
      top = target.top + target.height / 2 - tooltip.height / 2
      left = target.left - tooltip.width - GAP
      break
    case 'right':
    default:
      top = target.top + target.height / 2 - tooltip.height / 2
      left = target.left + target.width + GAP
      break
  }

  return {
    top: clamp(top, MARGIN, viewport.height - tooltip.height - MARGIN),
    left: clamp(left, MARGIN, viewport.width - tooltip.width - MARGIN),
    placement,
  }
}

function clamp(value: number, min: number, max: number): number {
  if (max < min) return min
  return Math.min(Math.max(value, min), max)
}
