import { describe, expect, it } from 'vitest'

import { initialOf } from '@/lib/utils'

describe('initialOf', () => {
  it.each([
    [' Jean', undefined, 'J'],
    ['_Anne-Marie', undefined, 'A'],
    ['123 Élodie', undefined, 'É'],
    ['😊 山田', undefined, '山'],
    ['---', '_bob', 'B'],
    ['', '42-maria', 'M'],
    ['', '', '?'],
  ])('retourne la première lettre de %s / %s', (displayName, username, expected) => {
    expect(initialOf(displayName, username)).toBe(expected)
  })
})
