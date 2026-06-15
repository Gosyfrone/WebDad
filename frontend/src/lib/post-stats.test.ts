import { describe, expect, it } from 'vitest'

import {
  applyStatsToPost,
  applyStatsToPosts,
  type FeedPost,
  type PostStats,
} from '@/lib/posts'

function makePost(overrides: Partial<FeedPost> = {}): FeedPost {
  return {
    id: 'p1',
    author: { id: 'u1', username: 'u', displayName: 'U', avatarUrl: '', visibility: 'public' },
    content: '',
    hashtags: [],
    media: [],
    quotePostId: '',
    quotedPost: null,
    likesCount: 1,
    commentsCount: 2,
    repostsCount: 3,
    pinnedAt: '',
    repostedById: '',
    repostedAt: '',
    createdAt: '',
    isPinned: false,
    liked: true,
    reposted: true,
    bookmarked: true,
    canDelete: false,
    canPin: false,
    ...overrides,
  }
}

const stats = (s: Partial<PostStats> & { id?: string } = {}): Map<string, PostStats> =>
  new Map([
    [
      s.id ?? 'p1',
      { likesCount: s.likesCount ?? 10, commentsCount: s.commentsCount ?? 20, repostsCount: s.repostsCount ?? 30 },
    ],
  ])

describe('applyStatsToPost', () => {
  it('met à jour les compteurs sans toucher l’état « moi »', () => {
    const post = makePost()
    const next = applyStatsToPost(post, stats())
    expect(next.likesCount).toBe(10)
    expect(next.commentsCount).toBe(20)
    expect(next.repostsCount).toBe(30)
    // L'état local de l'utilisateur reste intact.
    expect(next.liked).toBe(true)
    expect(next.reposted).toBe(true)
    expect(next.bookmarked).toBe(true)
  })

  it('renvoie la même référence si les compteurs sont inchangés', () => {
    const post = makePost({ likesCount: 10, commentsCount: 20, repostsCount: 30 })
    expect(applyStatsToPost(post, stats())).toBe(post)
  })

  it('renvoie la même référence si le post n’est pas dans le lot', () => {
    const post = makePost({ id: 'other' })
    expect(applyStatsToPost(post, stats())).toBe(post)
  })
})

describe('applyStatsToPosts', () => {
  it('renvoie la même liste si rien n’a changé', () => {
    const list = [makePost({ likesCount: 10, commentsCount: 20, repostsCount: 30 })]
    expect(applyStatsToPosts(list, stats())).toBe(list)
  })

  it('renvoie une nouvelle liste dès qu’un post change', () => {
    const list = [makePost({ id: 'p1' }), makePost({ id: 'p2', likesCount: 99 })]
    const next = applyStatsToPosts(list, stats())
    expect(next).not.toBe(list)
    expect(next[0].likesCount).toBe(10)
    // Post hors du lot : référence préservée.
    expect(next[1]).toBe(list[1])
  })
})
