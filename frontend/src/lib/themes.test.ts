import { describe, expect, it } from 'vitest'

import { COLOR_MODES, DEFAULT_MODE, ACCENTS, DEFAULT_ACCENT } from '@/lib/themes'

describe('COLOR_MODES', () => {
  it('contient light, dark et system', () => {
    expect(COLOR_MODES).toContain('light')
    expect(COLOR_MODES).toContain('dark')
    expect(COLOR_MODES).toContain('system')
  })
})

describe('DEFAULT_MODE', () => {
  it('est un mode valide', () => {
    expect(COLOR_MODES).toContain(DEFAULT_MODE)
  })
})

describe('ACCENTS', () => {
  it('contient au moins un accent', () => {
    expect(ACCENTS.length).toBeGreaterThan(0)
  })

  it('chaque accent a id, label et swatch non vides', () => {
    for (const accent of ACCENTS) {
      expect(accent.id).toBeTruthy()
      expect(accent.label).toBeTruthy()
      expect(accent.swatch).toBeTruthy()
    }
  })

  it('ids sont uniques', () => {
    const ids = ACCENTS.map((a) => a.id)
    expect(new Set(ids).size).toBe(ids.length)
  })
})

describe('DEFAULT_ACCENT', () => {
  it('correspond à un id dans ACCENTS', () => {
    expect(ACCENTS.map((a) => a.id)).toContain(DEFAULT_ACCENT)
  })
})
