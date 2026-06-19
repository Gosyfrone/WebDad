import { describe, expect, it } from 'vitest'

import {
  applyMention,
  detectMentionTyping,
  extractMentionHandles,
  parseMentionSegments,
  textMentionsUser,
} from '@/lib/mentions'

describe('parseMentionSegments', () => {
  it('isole les mentions du texte', () => {
    const segs = parseMentionSegments('hello @alice and @bob!')
    expect(segs).toEqual([
      { type: 'text', text: 'hello ' },
      { type: 'mention', handle: 'alice', raw: '@alice' },
      { type: 'text', text: ' and ' },
      { type: 'mention', handle: 'bob', raw: '@bob' },
      { type: 'text', text: '!' },
    ])
  })

  it("ne capture pas la partie locale d'un e-mail", () => {
    const segs = parseMentionSegments('write to jean@exemple.com please')
    expect(segs.every((s) => s.type === 'text')).toBe(true)
  })

  it('gère une mention en début de texte', () => {
    const segs = parseMentionSegments('@alice salut')
    expect(segs[0]).toEqual({ type: 'mention', handle: 'alice', raw: '@alice' })
  })

  it('ignore un handle trop court (< 3)', () => {
    const segs = parseMentionSegments('hey @ab')
    expect(segs.every((s) => s.type === 'text')).toBe(true)
  })

  it('capture un point interne au handle', () => {
    const segs = parseMentionSegments('coucou @jean.dupont !')
    expect(segs).toContainEqual({
      type: 'mention',
      handle: 'jean.dupont',
      raw: '@jean.dupont',
    })
  })

  it('exclut un point final de ponctuation', () => {
    const segs = parseMentionSegments('salut @bob.')
    expect(segs).toContainEqual({ type: 'mention', handle: 'bob', raw: '@bob' })
    expect(segs).toContainEqual({ type: 'text', text: '.' })
  })
})

describe('extractMentionHandles', () => {
  it('déduplique en minuscules', () => {
    expect(extractMentionHandles('@Alice @alice @BOB')).toEqual(['alice', 'bob'])
  })
})

describe('textMentionsUser', () => {
  it('détecte le handle, insensible à la casse', () => {
    expect(textMentionsUser('coucou @Alice', 'alice')).toBe(true)
    expect(textMentionsUser('coucou @bob', 'alice')).toBe(false)
    expect(textMentionsUser('coucou @bob', '')).toBe(false)
  })
})

describe('detectMentionTyping', () => {
  it('repère un token en cours de frappe', () => {
    const value = 'salut @ali'
    expect(detectMentionTyping(value, value.length)).toEqual({
      query: 'ali',
      atIndex: 6,
      caretEnd: 10,
    })
  })

  it("ouvre dès le « @ » seul", () => {
    expect(detectMentionTyping('hey @', 5)).toEqual({ query: '', atIndex: 4, caretEnd: 5 })
  })

  it('renvoie null hors mention', () => {
    expect(detectMentionTyping('hey there', 9)).toBeNull()
    expect(detectMentionTyping('mail jean@exemple', 17)).toBeNull()
  })
})

describe('applyMention', () => {
  it('remplace le token par @username + espace', () => {
    const value = 'salut @ali fin'
    const ctx = detectMentionTyping('salut @ali', 10)!
    expect(applyMention(value, ctx, 'alice')).toEqual({ value: 'salut @alice  fin', caret: 13 })
  })
})
