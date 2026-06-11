import { describe, it, expect } from 'vitest'

import {
  estimateStrength,
  isPassphraseAcceptable,
  MIN_PASSPHRASE_LENGTH,
} from './passphrase-strength'

describe('estimateStrength', () => {
  it('renvoie « empty » pour une chaîne vide', () => {
    const r = estimateStrength('')
    expect(r.level).toBe('empty')
    expect(r.score).toBe(0)
  })

  it('note faible une passphrase courte/triviale', () => {
    expect(estimateStrength('azerty').level).toBe('weak')
    expect(estimateStrength('aaaaaa').level).toBe('weak')
  })

  it('pénalise les motifs triviaux', () => {
    const trivial = estimateStrength('password123')
    const varied = estimateStrength('Gx7!vQ2#mLz')
    expect(varied.bits).toBeGreaterThan(trivial.bits)
  })

  it('note fortement une passphrase longue et variée', () => {
    const r = estimateStrength('Cheval-Bleu_42!Tomate')
    expect(r.score).toBeGreaterThanOrEqual(3)
    expect(['good', 'strong']).toContain(r.level)
  })
})

describe('isPassphraseAcceptable', () => {
  it('refuse en dessous de la longueur minimale', () => {
    expect(isPassphraseAcceptable('Ab1!xy')).toBe(false)
    expect('Ab1!xy'.length).toBeLessThan(MIN_PASSPHRASE_LENGTH)
  })

  it('refuse une passphrase trop simple même assez longue', () => {
    expect(isPassphraseAcceptable('aaaaaaaaaaaa')).toBe(false)
  })

  it('accepte une passphrase longue et variée', () => {
    expect(isPassphraseAcceptable('Cheval-Bleu_42!')).toBe(true)
  })
})
