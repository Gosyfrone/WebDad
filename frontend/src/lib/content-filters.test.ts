import { describe, expect, it } from 'vitest'

import {
  sanitizeMutedWords,
  postMatchesMutedWords,
  filterMutedPosts,
} from '@/lib/content-filters'
import type { FeedPost } from '@/lib/posts'

// ─── sanitizeMutedWords ───────────────────────────────────────────────────────

describe('sanitizeMutedWords', () => {
  it('supprime les doublons normalisés', () => {
    expect(sanitizeMutedWords(['Go', 'go', 'GO'])).toHaveLength(1)
  })

  it('supprime les entrées vides / espaces seuls', () => {
    expect(sanitizeMutedWords(['', '  ', 'ok'])).toHaveLength(1)
  })

  it('normalise les accents', () => {
    const result = sanitizeMutedWords(['café', 'cafe'])
    expect(result).toHaveLength(1)
  })

  it("conserve l'ordre de première apparition", () => {
    expect(sanitizeMutedWords(['b', 'a', 'b'])).toEqual(['b', 'a'])
  })

  it('retourne un tableau vide pour une entrée vide', () => {
    expect(sanitizeMutedWords([])).toEqual([])
  })
})

// ─── postMatchesMutedWords ───────────────────────────────────────────────────

function makePost(content: string, authorId = 'other', quotedContent?: string): FeedPost {
  return {
    id: 'p1',
    content,
    author: { id: authorId, username: 'alice', displayName: 'Alice', avatarUrl: null, isVerified: false },
    stats: { likes: 0, reposts: 0, replies: 0, bookmarks: 0, quotes: 0 },
    visibility: 'public',
    createdAt: new Date().toISOString(),
    quotedPost: quotedContent
      ? {
          id: 'q1',
          content: quotedContent,
          author: { id: 'bob', username: 'bob', displayName: 'Bob', avatarUrl: null, isVerified: false },
          stats: { likes: 0, reposts: 0, replies: 0, bookmarks: 0, quotes: 0 },
          visibility: 'public',
          createdAt: new Date().toISOString(),
        }
      : undefined,
  } as unknown as FeedPost
}

describe('postMatchesMutedWords', () => {
  it('retourne false si aucun mot muté', () => {
    expect(postMatchesMutedWords(makePost('typescript est top'), [])).toBe(false)
  })

  it('détecte un mot muté dans le contenu', () => {
    expect(postMatchesMutedWords(makePost('j adore typescript'), ['typescript'])).toBe(true)
  })

  it('est insensible à la casse', () => {
    expect(postMatchesMutedWords(makePost('TypeScript rocks'), ['typescript'])).toBe(true)
  })

  it('est insensible aux accents', () => {
    expect(postMatchesMutedWords(makePost('café crème'), ['cafe'])).toBe(true)
  })

  it("ne mute pas le post de l'auteur connecté", () => {
    const post = makePost('typescript', 'viewer-id')
    expect(postMatchesMutedWords(post, ['typescript'], 'viewer-id')).toBe(false)
  })

  it('détecte dans le quoted post', () => {
    const post = makePost('post normal', 'other', 'contient le mot muté')
    expect(postMatchesMutedWords(post, ['muté'])).toBe(true)
  })
})

// ─── filterMutedPosts ────────────────────────────────────────────────────────

describe('filterMutedPosts', () => {
  it('retourne tous les posts si mots vides', () => {
    const posts = [makePost('typescript'), makePost('go')]
    expect(filterMutedPosts(posts, [])).toHaveLength(2)
  })

  it('filtre les posts correspondants', () => {
    const posts = [makePost('j adore typescript'), makePost('go est cool')]
    const result = filterMutedPosts(posts, ['typescript'])
    expect(result).toHaveLength(1)
    expect(result[0].content).toBe('go est cool')
  })
})
