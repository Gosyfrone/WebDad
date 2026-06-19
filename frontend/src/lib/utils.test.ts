import { describe, expect, it } from 'vitest'

import { timeAgo, initialOf } from '@/lib/utils'

// ─── timeAgo ────────────────────────────────────────────────────────────────

describe('timeAgo', () => {
  function isoAgo(seconds: number): string {
    return new Date(Date.now() - seconds * 1000).toISOString()
  }

  it('retourne une chaîne vide pour une date invalide', () => {
    expect(timeAgo('')).toBe('')
    expect(timeAgo('not-a-date')).toBe('')
  })

  it('affiche en secondes sous 60s', () => {
    expect(timeAgo(isoAgo(10))).toBe('10s')
    expect(timeAgo(isoAgo(0))).toBe('0s')
  })

  it('affiche en minutes entre 1 et 59min (fr)', () => {
    expect(timeAgo(isoAgo(90))).toBe('1min')
    expect(timeAgo(isoAgo(3540))).toBe('59min')
  })

  it('affiche en minutes (en)', () => {
    expect(timeAgo(isoAgo(90), 'en')).toBe('1m')
  })

  it('affiche en heures entre 1h et 23h', () => {
    expect(timeAgo(isoAgo(3600))).toBe('1h')
    expect(timeAgo(isoAgo(23 * 3600))).toBe('23h')
  })

  it('affiche en jours entre 1j et 6j (fr)', () => {
    expect(timeAgo(isoAgo(86400))).toBe('1j')
    expect(timeAgo(isoAgo(6 * 86400))).toBe('6j')
  })

  it('affiche en jours (en)', () => {
    expect(timeAgo(isoAgo(86400), 'en')).toBe('1d')
  })

  it('affiche une date absolue au-delà de 7 jours', () => {
    const result = timeAgo(isoAgo(8 * 86400))
    // doit contenir un chiffre (jour ou mois)
    expect(result).toMatch(/\d/)
    // ne doit pas être un format « Xs » ou « Xmin »
    expect(result).not.toMatch(/^\d+s$/)
    expect(result).not.toMatch(/^\d+min$/)
  })
})

// ─── initialOf ───────────────────────────────────────────────────────────────

describe('initialOf', () => {
  it('retourne la première lettre du displayName en majuscule', () => {
    expect(initialOf('Alice', 'al')).toBe('A')
  })

  it('repli sur le username si displayName vide', () => {
    expect(initialOf('', 'bob')).toBe('B')
    expect(initialOf(null, 'carol')).toBe('C')
  })

  it('retourne ? si les deux sont vides', () => {
    expect(initialOf(null, null)).toBe('?')
    expect(initialOf(undefined, undefined)).toBe('?')
  })

  it('gère les caractères unicode', () => {
    expect(initialOf('Ève', null)).toBe('È')
  })

  it('ignore les caractères non-lettre en tête', () => {
    expect(initialOf('123alice', null)).toBe('A')
  })
})
