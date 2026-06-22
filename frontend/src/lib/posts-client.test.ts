import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

// Couche réseau de `lib/posts` : tous les appels passent par `apiFetch`. On le
// mocke (routeur par URL) pour piloter les réponses et inspecter les requêtes.
// `session` (claims JWT) est aussi mocké pour piloter canDelete / canPin /
// canMarkNsfw sans vrai token.
const apiFetch = vi.fn()
const getAccessToken = vi.fn(() => 'tok')
vi.mock('@/lib/auth-client', () => ({
  apiFetch: (...args: unknown[]) => apiFetch(...args),
  getAccessToken: () => getAccessToken(),
}))

const currentUserId = vi.fn(() => 'me')
const decodeClaims = vi.fn(() => ({ user_id: 'me', role: 'user' }) as unknown)
vi.mock('@/lib/session', () => ({
  currentUserId: () => currentUserId(),
  decodeClaims: () => decodeClaims(),
}))

import * as posts from '@/lib/posts'

function json(body: unknown, init: { ok?: boolean; status?: number } = {}) {
  return {
    ok: init.ok ?? true,
    status: init.status ?? 200,
    json: () => Promise.resolve(body),
  }
}

/** Routeur : map [matcher → réponse] ; un matcher string = égalité sur le path. */
function routes(table: Array<[(url: string) => boolean, unknown]>) {
  apiFetch.mockImplementation(async (url: string) => {
    for (const [match, res] of table) if (match(url)) return res
    return json({ data: null }, { ok: false, status: 404 })
  })
}
const path = (p: string) => (url: string) => url.split('?')[0] === p
const starts = (p: string) => (url: string) => url.startsWith(p)

/** Réponses user + profil pour résoudre un auteur donné. */
function authorRoutes(id: string): Array<[(url: string) => boolean, unknown]> {
  return [
    [path(`/users/${id}`), json({ data: { id, username: `user_${id}` } })],
    [
      path(`/profils/${id}`),
      json({
        data: {
          user_id: id,
          display_name: `Name ${id}`,
          avatar_url: '/media/a.png',
          visibility: 'public',
          last_login_at: '2026-01-01T00:00:00Z',
          is_online: true,
        },
      }),
    ],
  ]
}

function rawPost(id: string, authorId: string, extra: Record<string, unknown> = {}) {
  return {
    id,
    author_id: authorId,
    content: 'hello',
    created_at: '2026-06-01T00:00:00Z',
    ...extra,
  }
}

beforeEach(() => {
  getAccessToken.mockReturnValue('tok')
  currentUserId.mockReturnValue('me')
  decodeClaims.mockReturnValue({ user_id: 'me', role: 'user' })
})
afterEach(() => {
  apiFetch.mockReset()
})

// ─── ids « moi » ──────────────────────────────────────────────────────────────

describe('getLikedIds / getRepostedIds / getBookmarkedIds', () => {
  it('renvoie un Set vide sans token (pas de redirection /login)', async () => {
    getAccessToken.mockReturnValue('')
    expect((await posts.getLikedIds()).size).toBe(0)
    expect((await posts.getRepostedIds()).size).toBe(0)
    expect((await posts.getBookmarkedIds()).size).toBe(0)
    expect(apiFetch).not.toHaveBeenCalled()
  })

  it('renvoie le Set des ids quand la réponse est ok', async () => {
    routes([[path('/posts/me/liked-ids'), json({ data: ['a', 'b'] })]])
    const set = await posts.getLikedIds()
    expect([...set]).toEqual(['a', 'b'])
  })

  it('renvoie un Set vide quand la réponse échoue', async () => {
    routes([[path('/posts/me/reposted-ids'), json({}, { ok: false, status: 500 })]])
    expect((await posts.getRepostedIds()).size).toBe(0)
  })

  it('tolère un data absent (Set vide)', async () => {
    routes([[path('/posts/me/bookmarked-ids'), json({})]])
    expect((await posts.getBookmarkedIds()).size).toBe(0)
  })
})

// ─── mapping complet via listFeed ────────────────────────────────────────────

describe('listFeed → mapping ApiPost → FeedPost', () => {
  it('mappe tous les champs, le sondage, les états « moi » et les droits', async () => {
    const p = rawPost('p1', 'au1', {
      hashtags: ['x'],
      media: [{ url: '/media/m.png', type: 'image' }],
      likes_count: 3,
      comments_count: 2,
      reposts_count: 1,
      reply_audience: 'followers',
      can_reply: false,
      nsfw: true,
      poll: {
        choices: [{ id: 'c1', label: 'A', votes_count: 2, image_url: '/media/i.png' }],
        ends_at: '2026-07-01T00:00:00Z',
        audience: 'followers',
        total_votes: 2,
        voted_choice_id: 'c1',
        winner_choice_ids: ['c1'],
        can_view_results: true,
        can_close: true,
      },
    })
    decodeClaims.mockReturnValue({ user_id: 'au1', role: 'admin' })
    currentUserId.mockReturnValue('au1')
    routes([
      [path('/posts'), json({ data: [p] })],
      [path('/posts/me/liked-ids'), json({ data: ['p1'] })],
      [path('/posts/me/reposted-ids'), json({ data: ['p1'] })],
      [path('/posts/me/bookmarked-ids'), json({ data: [] })],
      ...authorRoutes('au1'),
    ])

    const [fp] = await posts.listFeed()
    expect(fp.id).toBe('p1')
    expect(fp.author.username).toBe('user_au1')
    expect(fp.author.displayName).toBe('Name au1')
    expect(fp.hashtags).toEqual(['x'])
    expect(fp.media[0].type).toBe('image')
    expect(fp.replyAudience).toBe('followers')
    expect(fp.canReply).toBe(false)
    expect(fp.liked).toBe(true)
    expect(fp.reposted).toBe(true)
    expect(fp.bookmarked).toBe(false)
    expect(fp.nsfw).toBe(true)
    expect(fp.canMarkNsfw).toBe(true)
    expect(fp.canPin).toBe(true)
    expect(fp.poll?.choices[0].votesCount).toBe(2)
    expect(fp.poll?.audience).toBe('followers')
    expect(fp.poll?.winnerChoiceIds).toEqual(['c1'])
  })

  it('défauts: champs absents → valeurs neutres, droits refusés', async () => {
    decodeClaims.mockReturnValue(null)
    routes([
      [path('/posts'), json({ data: [rawPost('p2', 'au2')] })],
      [starts('/posts/me/'), json({ data: [] })],
      ...authorRoutes('au2'),
    ])
    const [fp] = await posts.listFeed(10, 5)
    expect(fp.hashtags).toEqual([])
    expect(fp.media).toEqual([])
    expect(fp.poll).toBeNull()
    expect(fp.canReply).toBe(true)
    expect(fp.canDelete).toBe(false)
    expect(fp.canMarkNsfw).toBe(false)
    expect(fp.likesCount).toBe(0)
    const url = apiFetch.mock.calls.find((c) => String(c[0]).startsWith('/posts?'))![0]
    const params = new URLSearchParams(String(url).split('?')[1])
    expect(params.get('limit')).toBe('10')
    expect(params.get('offset')).toBe('5')
  })

  it('passe hashtag + sort dans feedParams (# retiré, trim)', async () => {
    routes([
      [path('/posts'), json({ data: [] })],
      [starts('/posts/me/'), json({ data: [] })],
    ])
    await posts.listFeed(20, 0, ' #js ', 'top')
    const url = apiFetch.mock.calls.find((c) => String(c[0]).startsWith('/posts?'))![0]
    const params = new URLSearchParams(String(url).split('?')[1])
    expect(params.get('hashtag')).toBe('js')
    expect(params.get('sort')).toBe('top')
  })

  it('résout l’auteur de repli quand user et profil manquent', async () => {
    routes([
      [path('/posts'), json({ data: [rawPost('p3', 'ghost')] })],
      [starts('/posts/me/'), json({ data: [] })],
      // /users/ghost et /profils/ghost répondent 404 (auteur introuvable)
    ])
    const [fp] = await posts.listFeed()
    expect(fp.author.username).toBe('')
    expect(fp.author.displayName).toBe('Utilisateur')
    expect(fp.author.visibility).toBe('public')
  })
})

// ─── post unitaire + citation ────────────────────────────────────────────────

describe('getPostById', () => {
  it('charge un post et sa citation (profondeur 1)', async () => {
    routes([
      [path('/posts/p10'), json({ data: rawPost('p10', 'a10', { quote_post_id: 'p11' }) })],
      [path('/posts/p11'), json({ data: rawPost('p11', 'a11') })],
      [starts('/posts/me/'), json({ data: [] })],
      ...authorRoutes('a10'),
      ...authorRoutes('a11'),
    ])
    const fp = await posts.getPostById('p10')
    expect(fp?.quotePostId).toBe('p11')
    expect(fp?.quotedPost?.id).toBe('p11')
    // pas de récursion infinie : la citation n'a pas elle-même de quotedPost
    expect(fp?.quotedPost?.quotedPost).toBeNull()
  })

  it('renvoie null sur 404', async () => {
    routes([
      [starts('/posts/me/'), json({ data: [] })],
      [path('/posts/nope'), json({}, { ok: false, status: 404 })],
    ])
    expect(await posts.getPostById('nope')).toBeNull()
  })
})

// ─── listes ───────────────────────────────────────────────────────────────────

describe('listes de posts', () => {
  beforeEach(() => {
    routes([
      [path('/posts'), json({ data: [] })],
      [path('/posts/liked'), json({ data: [] })],
      [path('/posts/trends'), json({ data: [{ tag: 't', count: 1 }] })],
      [starts('/posts/me/'), json({ data: [] })],
      [starts('/users/me/following'), json({ data: [{ id: 'f1' }, { id: 'f2' }] })],
    ])
  })

  it('listFollowingFeed: aucun suivi → tableau vide sans requête /posts', async () => {
    routes([[starts('/users/me/following'), json({ data: [] })]])
    expect(await posts.listFollowingFeed()).toEqual([])
  })

  it('listFollowingFeed: ajoute author_ids', async () => {
    await posts.listFollowingFeed()
    const url = apiFetch.mock.calls.find((c) => String(c[0]).startsWith('/posts?'))![0]
    expect(new URLSearchParams(String(url).split('?')[1]).get('author_ids')).toBe('f1,f2')
  })

  it('listFollowingFeed: pas d’utilisateur courant → vide', async () => {
    currentUserId.mockReturnValue('')
    expect(await posts.listFollowingFeed()).toEqual([])
  })

  it('listByAuthor: ajoute author_id', async () => {
    await posts.listByAuthor('au')
    const url = apiFetch.mock.calls.find((c) => String(c[0]).startsWith('/posts?'))![0]
    expect(new URLSearchParams(String(url).split('?')[1]).get('author_id')).toBe('au')
  })

  it('listLikedByUser: 403 → erreur likes_private', async () => {
    routes([[path('/posts/liked'), json({}, { ok: false, status: 403 })]])
    await expect(posts.listLikedByUser('au')).rejects.toThrow('likes_private')
  })

  it('listLikedByUser: ok → mappe', async () => {
    expect(await posts.listLikedByUser('au')).toEqual([])
  })

  it('listHashtagTrends: nettoie le # et omet q vide', async () => {
    const t = await posts.listHashtagTrends(5, '  #js ')
    expect(t).toHaveLength(1)
    const url = apiFetch.mock.calls.find((c) => String(c[0]).startsWith('/posts/trends'))![0]
    expect(new URLSearchParams(String(url).split('?')[1]).get('q')).toBe('js')
  })

  it('listHashtaggedPosts: ajoute hashtag_any', async () => {
    await posts.listHashtaggedPosts()
    const url = apiFetch.mock.calls.find((c) => String(c[0]).startsWith('/posts?'))![0]
    expect(new URLSearchParams(String(url).split('?')[1]).get('hashtag_any')).toBe('true')
  })
})

// ─── écritures ────────────────────────────────────────────────────────────────

describe('createPost', () => {
  beforeEach(() => {
    routes([
      [path('/posts'), json({ data: rawPost('new', 'me') })],
      ...authorRoutes('me'),
    ])
  })

  it('payload minimal: seulement content', async () => {
    await posts.createPost('coucou')
    const body = JSON.parse(apiFetch.mock.calls.find((c) => c[1])![1].body)
    expect(body).toEqual({ content: 'coucou' })
  })

  it('payload complet: media, citation, sondage, audience, nsfw', async () => {
    await posts.createPost(
      'x',
      [{ url: 'u', type: 'image' }],
      'q1',
      {
        choices: [{ label: 'A', imageUrl: 'i' }, { label: 'B' }],
        durationMinutes: 60,
        audience: 'everyone',
      },
      'followers',
      true,
    )
    const body = JSON.parse(apiFetch.mock.calls.find((c) => c[1])![1].body)
    expect(body.media).toHaveLength(1)
    expect(body.quote_post_id).toBe('q1')
    expect(body.reply_audience).toBe('followers')
    expect(body.nsfw).toBe(true)
    expect(body.poll.choices[0]).toEqual({ label: 'A', image_url: 'i' })
    expect(body.poll.choices[1]).toEqual({ label: 'B' })
    expect(body.poll.duration_minutes).toBe(60)
  })
})

describe('actions sur un post (renvoient le post à jour)', () => {
  beforeEach(() => {
    routes([
      [path('/posts/p/poll/vote'), json({ data: rawPost('p', 'me') })],
      [path('/posts/p/poll/close'), json({ data: rawPost('p', 'me') })],
      [path('/posts/p/pin'), json({ data: rawPost('p', 'me') })],
      [path('/posts/p/nsfw'), json({ data: rawPost('p', 'me') })],
      [path('/posts/p/repost'), json({ data: rawPost('p', 'me') })],
      [starts('/posts/me/'), json({ data: [] })],
      ...authorRoutes('me'),
    ])
  })

  it('votePoll envoie choice_id', async () => {
    await posts.votePoll('p', 'c2')
    const call = apiFetch.mock.calls.find((c) => String(c[0]) === '/posts/p/poll/vote')!
    expect(JSON.parse(call[1].body)).toEqual({ choice_id: 'c2' })
  })

  it('closePoll POST', async () => {
    const fp = await posts.closePoll('p')
    expect(fp.id).toBe('p')
  })

  it('pinPost PATCH / unpinPost DELETE', async () => {
    await posts.pinPost('p')
    await posts.unpinPost('p')
    const methods = apiFetch.mock.calls
      .filter((c) => String(c[0]) === '/posts/p/pin')
      .map((c) => c[1].method)
    expect(methods).toEqual(['PATCH', 'DELETE'])
  })

  it('setPostNsfw envoie le flag', async () => {
    await posts.setPostNsfw('p', true)
    const call = apiFetch.mock.calls.find((c) => String(c[0]) === '/posts/p/nsfw')!
    expect(JSON.parse(call[1].body)).toEqual({ nsfw: true })
  })

  it('repostPost POST', async () => {
    const fp = await posts.repostPost('p')
    expect(fp.id).toBe('p')
  })
})

describe('deletePost / likes / reposts', () => {
  it('deletePost ok ne lève pas', async () => {
    routes([[path('/posts/p'), json({}, { ok: true })]])
    await expect(posts.deletePost('p')).resolves.toBeUndefined()
  })

  it('deletePost échec lève PostApiError', async () => {
    routes([[path('/posts/p'), json({}, { ok: false, status: 403 })]])
    await expect(posts.deletePost('p')).rejects.toMatchObject({ status: 403 })
  })

  it('likePost / unlikePost renvoient likes_count', async () => {
    routes([[path('/posts/p/like'), json({ data: { likes_count: 7 } })]])
    expect(await posts.likePost('p')).toBe(7)
    expect(await posts.unlikePost('p')).toBe(7)
  })

  it('unrepostPost renvoie reposts_count', async () => {
    routes([[path('/posts/p/repost'), json({ data: { reposts_count: 4 } })]])
    expect(await posts.unrepostPost('p')).toBe(4)
  })
})

// ─── commentaires ─────────────────────────────────────────────────────────────

describe('commentaires', () => {
  function rawComment(id: string, extra: Record<string, unknown> = {}) {
    return {
      id,
      post_id: 'p',
      author_id: 'ca',
      content: 'c',
      created_at: '2026-06-01T00:00:00Z',
      ...extra,
    }
  }
  beforeEach(() => {
    routes([
      [starts('/posts/me/'), json({ data: [] })],
      ...authorRoutes('ca'),
      ...authorRoutes('pa'),
    ])
  })

  it('listComments mappe les commentaires racine', async () => {
    routes([
      [starts('/posts/p/comments'), json({ data: [rawComment('c1', { likes_count: 2, liked: true, reply_count: 5 })] })],
      ...authorRoutes('ca'),
    ])
    const [c] = await posts.listComments('p')
    expect(c.id).toBe('c1')
    expect(c.likesCount).toBe(2)
    expect(c.liked).toBe(true)
    expect(c.replyCount).toBe(5)
  })

  it('listReplies cible la route replies', async () => {
    routes([
      [starts('/posts/p/comments/c1/replies'), json({ data: [rawComment('r1')] })],
      ...authorRoutes('ca'),
    ])
    const r = await posts.listReplies('p', 'c1')
    expect(r[0].id).toBe('r1')
  })

  it('listCommentsByAuthor enrichit le post et le commentaire parent', async () => {
    routes([
      [
        starts('/posts/comments'),
        json({
          data: [
            {
              ...rawComment('c2'),
              parent_post: rawPost('pp', 'pa'),
              parent_comment: rawComment('pc'),
            },
            { ...rawComment('c3') },
          ],
        }),
      ],
      [starts('/posts/me/'), json({ data: [] })],
      ...authorRoutes('ca'),
      ...authorRoutes('pa'),
    ])
    const ctx = await posts.listCommentsByAuthor('ca')
    expect(ctx[0].parentPost?.id).toBe('pp')
    expect(ctx[0].parentComment?.id).toBe('pc')
    expect(ctx[0].parentPostAuthor.username).toBe('user_pa')
    // sans parent_post → auteur de repli '...'
    expect(ctx[1].parentPost).toBeNull()
    expect(ctx[1].parentPostAuthor.displayName).toBe('...')
  })

  it('listCommentsByAuthor: data nul → tableau vide', async () => {
    routes([[starts('/posts/comments'), json({ data: null })]])
    expect(await posts.listCommentsByAuthor('ca')).toEqual([])
  })

  it('createComment: payload avec parent_id et media', async () => {
    routes([
      [starts('/posts/p/comments'), json({ data: rawComment('cN') })],
      ...authorRoutes('ca'),
    ])
    await posts.createComment('p', 'salut', 'c1', [{ url: 'u', type: 'image' }])
    const call = apiFetch.mock.calls.find((c) => c[1] && String(c[0]).includes('/comments'))!
    const body = JSON.parse(call[1].body)
    expect(body).toEqual({ content: 'salut', parent_id: 'c1', media: [{ url: 'u', type: 'image' }] })
  })

  it('deleteComment échec → PostApiError', async () => {
    routes([[starts('/posts/p/comments/c1'), json({}, { ok: false, status: 403 })]])
    await expect(posts.deleteComment('p', 'c1')).rejects.toThrow('Suppression impossible')
  })

  it('likeComment / unlikeComment renvoient likes_count', async () => {
    routes([[starts('/posts/p/comments/c1/like'), json({ data: { likes_count: 9 } })]])
    expect(await posts.likeComment('p', 'c1')).toBe(9)
    expect(await posts.unlikeComment('p', 'c1')).toBe(9)
  })
})

// ─── stats ────────────────────────────────────────────────────────────────────

describe('getPostsStats / getCommentsStats', () => {
  it('dédoublonne, filtre les vides et borne à 100', async () => {
    routes([
      [
        starts('/posts/stats'),
        json({ data: [{ id: 'a', likes_count: 1, comments_count: 2, reposts_count: 3 }] }),
      ],
    ])
    const m = await posts.getPostsStats(['a', 'a', '', 'a'])
    expect(m.get('a')).toEqual({ likesCount: 1, commentsCount: 2, repostsCount: 3 })
    const url = apiFetch.mock.calls[0][0]
    expect(new URLSearchParams(String(url).split('?')[1]).get('ids')).toBe('a')
  })

  it('liste vide → aucune requête', async () => {
    expect((await posts.getPostsStats([])).size).toBe(0)
    expect((await posts.getCommentsStats(['', ''])).size).toBe(0)
    expect(apiFetch).not.toHaveBeenCalled()
  })

  it('getCommentsStats mappe les likes', async () => {
    routes([[starts('/posts/comments/stats'), json({ data: [{ id: 'c', likes_count: 4 }] })]])
    const m = await posts.getCommentsStats(['c'])
    expect(m.get('c')).toEqual({ likesCount: 4 })
  })
})

describe('applyStatsToPost(s)', () => {
  const base = {
    id: 'p',
    likesCount: 1,
    commentsCount: 1,
    repostsCount: 1,
  } as posts.FeedPost

  it('renvoie la même référence si aucun compteur ne bouge', () => {
    const stats = new Map([['p', { likesCount: 1, commentsCount: 1, repostsCount: 1 }]])
    expect(posts.applyStatsToPost(base, stats)).toBe(base)
  })

  it('renvoie la même référence si le post est absent des stats', () => {
    expect(posts.applyStatsToPost(base, new Map())).toBe(base)
  })

  it('met à jour quand un compteur change', () => {
    const stats = new Map([['p', { likesCount: 5, commentsCount: 1, repostsCount: 1 }]])
    const next = posts.applyStatsToPost(base, stats)
    expect(next).not.toBe(base)
    expect(next.likesCount).toBe(5)
  })

  it('applyStatsToPosts: même liste si rien ne change, nouvelle sinon', () => {
    const list = [base]
    expect(posts.applyStatsToPosts(list, new Map())).toBe(list)
    const stats = new Map([['p', { likesCount: 2, commentsCount: 1, repostsCount: 1 }]])
    expect(posts.applyStatsToPosts(list, stats)).not.toBe(list)
  })
})

// ─── broadcast + temps réel (globaux stubés) ─────────────────────────────────

describe('notifyPostCreated / subscribePostCreated', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('notifyPostCreated ne fait rien sans window', () => {
    vi.stubGlobal('window', undefined)
    expect(() => posts.notifyPostCreated({ id: 'p' } as posts.FeedPost)).not.toThrow()
  })

  it('subscribePostCreated relaie l’évènement puis se désabonne', () => {
    const listeners = new Map<string, (e: Event) => void>()
    vi.stubGlobal('window', {
      addEventListener: (t: string, h: (e: Event) => void) => listeners.set(t, h),
      removeEventListener: (t: string) => listeners.delete(t),
      dispatchEvent: (e: Event) => listeners.get(e.type)?.(e),
    })
    vi.stubGlobal(
      'CustomEvent',
      class {
        type: string
        detail: unknown
        constructor(type: string, init: { detail: unknown }) {
          this.type = type
          this.detail = init.detail
        }
      },
    )
    const seen: unknown[] = []
    const off = posts.subscribePostCreated((p) => seen.push(p))
    posts.notifyPostCreated({ id: 'p9' } as posts.FeedPost)
    expect(seen).toEqual([{ id: 'p9' }])
    off()
    posts.notifyPostCreated({ id: 'p10' } as posts.FeedPost)
    expect(seen).toHaveLength(1)
  })
})

describe('connectFeedRealtime', () => {
  let sockets: FakeWS[]
  class FakeWS {
    url: string
    onmessage: ((e: { data: string }) => void) | null = null
    onopen: (() => void) | null = null
    onclose: (() => void) | null = null
    closed = false
    constructor(url: string) {
      this.url = url
      sockets.push(this)
    }
    close() {
      this.closed = true
    }
  }
  beforeEach(() => {
    sockets = []
    vi.stubGlobal('WebSocket', FakeWS as unknown as typeof WebSocket)
    vi.useFakeTimers()
  })
  afterEach(() => {
    vi.useRealTimers()
    vi.unstubAllGlobals()
  })

  it('n’ouvre pas de socket sans token', () => {
    getAccessToken.mockReturnValue('')
    posts.connectFeedRealtime(() => {})
    expect(sockets).toHaveLength(0)
  })

  it('ouvre la socket en ws:// avec le token et relaie les pings valides', () => {
    const pings: posts.NewPostPing[] = []
    posts.connectFeedRealtime((p) => pings.push(p))
    expect(sockets[0].url).toContain('/posts/ws?access_token=tok')
    expect(sockets[0].url.startsWith('ws')).toBe(true)
    sockets[0].onopen?.()
    sockets[0].onmessage?.({ data: 'not-json' }) // ignoré
    sockets[0].onmessage?.({ data: JSON.stringify({ type: 'other' }) }) // ignoré
    sockets[0].onmessage?.({
      data: JSON.stringify({ type: 'post_created', post_id: 'p1', author_id: 'a1' }),
    })
    expect(pings).toEqual([{ postId: 'p1', authorId: 'a1' }])
  })

  it('reconnecte après une fermeture non sollicitée', () => {
    posts.connectFeedRealtime(() => {})
    sockets[0].onclose?.()
    vi.advanceTimersByTime(1000)
    expect(sockets.length).toBeGreaterThan(1)
  })

  it('close() empêche la reconnexion', () => {
    const handle = posts.connectFeedRealtime(() => {})
    handle.close()
    expect(sockets[0].closed).toBe(true)
    sockets[0].onclose?.()
    vi.advanceTimersByTime(5000)
    expect(sockets).toHaveLength(1)
  })
})

describe('getPostAuthor', () => {
  it('résout l’auteur via user + profil', async () => {
    routes([...authorRoutes('z1')])
    const a = await posts.getPostAuthor('z1')
    expect(a.username).toBe('user_z1')
    expect(a.isOnline).toBe(true)
  })
})
