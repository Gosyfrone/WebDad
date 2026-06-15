import { describe, expect, it } from 'vitest'

import { isValidDisplayName } from '@/lib/display-name'

describe('isValidDisplayName', () => {
  it.each(['Jean Dupont', 'Élodie_75', 'Anne-Marie', '山田 太郎', 'Мария-2'])(
    'accepte %s',
    (name) => expect(isValidDisplayName(name)).toBe(true),
  )

  it.each(['Jean@Dupont', '#Jean', "O'Connor", 'Jean.Dupont', 'Jean 😊', '   '])(
    'refuse %s',
    (name) => expect(isValidDisplayName(name)).toBe(false),
  )
})
