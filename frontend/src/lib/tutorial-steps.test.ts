import { describe, expect, it } from 'vitest'

import {
  TUTORIAL_STEPS,
  TUTORIAL_STEP_COUNT,
  clampStepIndex,
  computeTooltipPosition,
  stepBodyKey,
  stepTitleKey,
  type Rect,
} from '@/lib/tutorial-steps'

// ─── TUTORIAL_STEPS ──────────────────────────────────────────────────────────

describe('TUTORIAL_STEPS', () => {
  it('contient au moins une étape et un compteur cohérent', () => {
    expect(TUTORIAL_STEPS.length).toBeGreaterThan(0)
    expect(TUTORIAL_STEP_COUNT).toBe(TUTORIAL_STEPS.length)
  })

  it('a des identifiants uniques', () => {
    const ids = TUTORIAL_STEPS.map((s) => s.id)
    expect(new Set(ids).size).toBe(ids.length)
  })

  it('chaque étape a une route', () => {
    for (const step of TUTORIAL_STEPS) {
      expect(typeof step.route).toBe('string')
      expect(step.route.length).toBeGreaterThan(0)
    }
  })
})

// ─── clampStepIndex ──────────────────────────────────────────────────────────

describe('clampStepIndex', () => {
  it('borne dans [0, dernier]', () => {
    expect(clampStepIndex(-5)).toBe(0)
    expect(clampStepIndex(0)).toBe(0)
    expect(clampStepIndex(TUTORIAL_STEPS.length + 10)).toBe(TUTORIAL_STEPS.length - 1)
  })

  it('gère NaN', () => {
    expect(clampStepIndex(Number.NaN)).toBe(0)
  })
})

// ─── stepTitleKey / stepBodyKey ──────────────────────────────────────────────

describe('clés i18n des étapes', () => {
  it('dérivent de l’id', () => {
    const step = TUTORIAL_STEPS[0]
    expect(stepTitleKey(step)).toBe(`tutorial.step.${step.id}.title`)
    expect(stepBodyKey(step)).toBe(`tutorial.step.${step.id}.body`)
  })
})

// ─── computeTooltipPosition ──────────────────────────────────────────────────

describe('computeTooltipPosition', () => {
  const viewport = { width: 1000, height: 800 }
  const tooltip = { width: 320, height: 160 }

  it('place à droite quand demandé et que ça rentre', () => {
    const target: Rect = { top: 300, left: 100, width: 40, height: 40 }
    const pos = computeTooltipPosition(target, tooltip, viewport, 'right')
    expect(pos.placement).toBe('right')
    expect(pos.left).toBeGreaterThanOrEqual(target.left + target.width)
  })

  it('bascule de côté si le côté préféré ne rentre pas', () => {
    // Cible collée au bord droit : « right » ne rentre pas → autre côté.
    const target: Rect = { top: 300, left: 980, width: 20, height: 20 }
    const pos = computeTooltipPosition(target, tooltip, viewport, 'right')
    expect(pos.placement).not.toBe('right')
  })

  it('borne la position dans le viewport', () => {
    const target: Rect = { top: 0, left: 0, width: 10, height: 10 }
    const pos = computeTooltipPosition(target, tooltip, viewport, 'auto')
    expect(pos.top).toBeGreaterThanOrEqual(0)
    expect(pos.left).toBeGreaterThanOrEqual(0)
    expect(pos.top + tooltip.height).toBeLessThanOrEqual(viewport.height)
    expect(pos.left + tooltip.width).toBeLessThanOrEqual(viewport.width)
  })

  it('choisit le côté avec le plus d’espace en mode auto', () => {
    // Beaucoup d'espace en bas, peu ailleurs.
    const target: Rect = { top: 40, left: 480, width: 40, height: 40 }
    const pos = computeTooltipPosition(target, tooltip, viewport, 'auto')
    expect(['bottom', 'right', 'left']).toContain(pos.placement)
  })
})
