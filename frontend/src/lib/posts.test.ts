import { describe, expect, it } from 'vitest'

import { applyProfilUpdateToPosts, PostApiError } from '@/lib/posts'
import type { FeedPost, PostAuthor } from '@/lib/posts'
import type { ProfilDetails } from '@/types'

// ─── helpers ─────────────────────────────────────────────────────────────────

function makeAuthor(overrides: Partial<PostAuthor> = {}): PostAuthor {
  return {
    id: 'u1',
    username: 'alice',
    displayName: 'Alice',
    avatarUrl: '',
    visibility: 'public',
    lastLoginAt: '',
    isOnline: false,
    ...overrides,
  }
}

function makePost(id: string, author: PostAuthor, overrides: Partial<FeedPost> = {}): FeedPost {
  return {
    id,
    author,
    content: 'test content',
    hashtags: [],
    media: [],
    poll: null,
    quotePostId: '',
    quotedPost: null,
    replyAudience: 'everyone',
    canReply: true,
    likesCount: 0,
    commentsCount: 0,
    repostsCount: 0,
    pinnedAt: '',
    repostedById: '',
    repostedAt: '',
    createdAt: '2024-01-01T00:00:00Z',
    isPinned: false,
    liked: false,
    reposted: false,
    bookmarked: false,
    canDelete: false,
    canPin: false,
    ...overrides,
  }
}

function makeProfil(overrides: Partial<ProfilDetails> = {}): ProfilDetails {
  return {
    userId: 'u1',
    displayName: 'Alice Updated',
    username: 'alice',
    role: 'user',
    isActive: true,
    bio: '',
    avatarUrl: '/media/avatar',
    bannerUrl: '',
    website: '',
    location: '',
    birthDate: '',
    gender: '',
    nationality: '',
    joinedAt: '',
    updatedAt: '',
    displayNameChangedAt: '',
    visibility: 'public',
    likesVisibility: 'public',
    activityVisibility: 'public',
    lastLoginAt: '',
    isOnline: true,
    followersCount: 0,
    followingCount: 0,
    postsCount: 0,
    profileExists: true,
    ...overrides,
  }
}

// ─── PostApiError ─────────────────────────────────────────────────────────────

describe('PostApiError', () => {
  it('contient le status HTTP', () => {
    const err = new PostApiError('Not found', 404)
    expect(err.status).toBe(404)
    expect(err.message).toBe('Not found')
    expect(err.name).toBe('PostApiError')
  })

  it('est une instance d\'Error', () => {
    const err = new PostApiError('Internal', 500)
    expect(err instanceof Error).toBe(true)
  })
})

// ─── applyProfilUpdateToPosts ─────────────────────────────────────────────────

describe('applyProfilUpdateToPosts', () => {
  it('retourne une liste vide si aucun post', () => {
    const profil = makeProfil()
    const result = applyProfilUpdateToPosts([], profil)
    expect(result).toEqual([])
  })

  it('met à jour l\'auteur des posts correspondants', () => {
    const author = makeAuthor({ id: 'u1', displayName: 'Alice Old' })
    const post = makePost('p1', author)
    const profil = makeProfil({ userId: 'u1', displayName: 'Alice New' })

    const result = applyProfilUpdateToPosts([post], profil)
    expect(result[0].author.displayName).toBe('Alice New')
  })

  it('ne touche pas les posts d\'un autre auteur', () => {
    const otherAuthor = makeAuthor({ id: 'u2', displayName: 'Bob' })
    const post = makePost('p1', otherAuthor)
    const profil = makeProfil({ userId: 'u1', displayName: 'Alice New' })

    const result = applyProfilUpdateToPosts([post], profil)
    expect(result[0].author.displayName).toBe('Bob')
    expect(result[0].author.id).toBe('u2')
  })

  it('retourne le même objet si rien ne change', () => {
    const otherAuthor = makeAuthor({ id: 'u99' })
    const post = makePost('p1', otherAuthor)
    const profil = makeProfil({ userId: 'u1' })

    const result = applyProfilUpdateToPosts([post], profil)
    expect(result[0]).toBe(post)
  })

  it('met à jour le post cité si l\'auteur correspond', () => {
    const author = makeAuthor({ id: 'u1', displayName: 'Alice Old' })
    const otherAuthor = makeAuthor({ id: 'u2', displayName: 'Bob' })
    const quotedPost = makePost('p0', author)
    const post = makePost('p1', otherAuthor, { quotedPost })
    const profil = makeProfil({ userId: 'u1', displayName: 'Alice New' })

    const result = applyProfilUpdateToPosts([post], profil)
    expect(result[0].quotedPost?.author.displayName).toBe('Alice New')
  })

  it('met à jour l\'isOnline dans l\'auteur', () => {
    const author = makeAuthor({ id: 'u1', isOnline: false })
    const post = makePost('p1', author)
    const profil = makeProfil({ userId: 'u1', isOnline: true })

    const result = applyProfilUpdateToPosts([post], profil)
    expect(result[0].author.isOnline).toBe(true)
  })
})
