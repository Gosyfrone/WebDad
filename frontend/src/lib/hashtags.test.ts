import { describe, expect, it } from 'vitest'

import { applyHashtag, detectHashtagTyping, parseHashtagSegments } from '@/lib/hashtags'

describe('detectHashtagTyping', () => {
  it('détecte un hashtag en cours de saisie', () => {
    expect(detectHashtagTyping('hello #bre', 10)).toEqual({
      query: 'bre',
      hashIndex: 6,
      caretEnd: 10,
    })
  })

  it('accepte un hashtag vide juste après #', () => {
    expect(detectHashtagTyping('hello #', 7)).toEqual({
      query: '',
      hashIndex: 6,
      caretEnd: 7,
    })
  })

  it('ignore les # collés à un mot', () => {
    expect(detectHashtagTyping('hello#bre', 9)).toBeNull()
  })
})

describe('applyHashtag', () => {
  it('remplace le token et ajoute un espace', () => {
    const ctx = detectHashtagTyping('hello #br fin', 9)!
    expect(applyHashtag('hello #br fin', ctx, 'breezy')).toEqual({
      value: 'hello #breezy  fin',
      caret: 14,
    })
  })
})

describe('parseHashtagSegments', () => {
  it('isole les hashtags pour le rendu du composer', () => {
    expect(parseHashtagSegments('Go #Breezy et #dev_2026')).toEqual([
      { type: 'text', text: 'Go ' },
      { type: 'hashtag', tag: 'Breezy', raw: '#Breezy' },
      { type: 'text', text: ' et ' },
      { type: 'hashtag', tag: 'dev_2026', raw: '#dev_2026' },
    ])
  })
})
