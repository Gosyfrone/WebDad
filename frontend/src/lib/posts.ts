/**
 * Client typé du post-service via l'API Gateway.
 *
 * Tous les appels passent par `apiFetch` (Bearer + refresh single-flight). Le
 * post-service ne stocke que `author_id` : ce module **résout l'auteur**
 * (username via user-service, displayName/avatar via profil-service) et
 * **mémoïse** le résultat (`authorCache`) pour ne pas refetch le même auteur à
 * chaque post d'un fil (N+1 best-effort, acceptable à l'échelle du projet).
 *
 * Les réponses de l'API sont enveloppées dans `{ data }` ; le snake_case de
 * l'API est mappé vers le camelCase des types front.
 */

import { apiFetch, getAccessToken } from '@/lib/auth-client'
import { API_URL } from '@/lib/config'
import { resolveMediaUrl } from '@/lib/media'
import type { ProfilDetails } from '@/types'
import { decodeClaims } from '@/lib/session'

/** Erreur d'appel API portant le code HTTP. */
export class PostApiError extends Error {
  status: number
  constructor(message: string, status: number) {
    super(message)
    this.name = 'PostApiError'
    this.status = status
  }
}

// --- Formes brutes (snake_case) ---------------------------------------------

export interface ApiPost {
  id: string
  author_id: string
  content: string
  hashtags?: string[]
  media?: { url: string; type: 'image' | 'video' }[]
  poll?: ApiPoll
  quote_post_id?: string
  reply_audience?: ReplyAudience
  can_reply?: boolean
  likes_count: number
  comments_count: number
  reposts_count?: number
  pinned_at?: string
  reposted_by_id?: string
  reposted_at?: string
  created_at: string
}

export interface ApiPoll {
  choices: ApiPollChoice[]
  ends_at: string
  closed_at?: string
  audience: PollAudience
  total_votes: number
  voted_choice_id?: string
  winner_choice_ids?: string[]
  can_view_results: boolean
  can_close: boolean
}

export interface ApiPollChoice {
  id: string
  label: string
  votes_count: number
  image_url?: string
}

interface ApiComment {
  id: string
  post_id: string
  parent_id?: string
  author_id: string
  content: string
  media?: { url: string; type: 'image' | 'video' }[]
  likes_count?: number
  liked?: boolean
  reply_count?: number
  created_at: string
}

interface ApiCommentWithPost extends ApiComment {
  parent_post?: ApiPost
  parent_comment?: ApiComment
}

interface ApiUser {
  id: string
  username: string
}

interface ApiProfil {
  user_id: string
  display_name?: string
  avatar_url?: string
  visibility?: 'public' | 'private'
  activity_visibility?: 'public' | 'private'
  last_login_at?: string
  is_online?: boolean
}

// --- Types front -------------------------------------------------------------

/** Auteur résolu (identité + décoratif) d'un post / commentaire. */
export interface PostAuthor {
  id: string
  username: string
  displayName: string
  avatarUrl: string
  visibility: 'public' | 'private'
  lastLoginAt: string
  isOnline: boolean
}

/** Média attaché à un post (URL prête à l'affichage + nature). */
export interface PostMedia {
  /** URL absolue (gateway) en lecture ; chemin relatif `/media/<id>` à la création. */
  url: string
  type: 'image' | 'video'
}

/** Post enrichi pour l'affichage. */
export interface FeedPost {
  id: string
  author: PostAuthor
  content: string
  hashtags: string[]
  media: PostMedia[]
  poll: PostPoll | null
  quotePostId: string
  quotedPost: FeedPost | null
  /** Qui peut répondre/commenter (`everyone` par défaut). */
  replyAudience: ReplyAudience
  /** L'utilisateur courant peut-il répondre/commenter ce post ? */
  canReply: boolean
  likesCount: number
  commentsCount: number
  repostsCount: number
  pinnedAt: string
  repostedById: string
  repostedAt: string
  createdAt: string
  isPinned: boolean
  /** L'utilisateur courant a-t-il liké ce post ? */
  liked: boolean
  /** L'utilisateur courant a-t-il reposté ce post ? */
  reposted: boolean
  /** L'utilisateur courant a-t-il signé ce post (dans au moins une collection) ? */
  bookmarked: boolean
  /** L'utilisateur courant peut-il supprimer (auteur ou mod/admin) ? */
  canDelete: boolean
  /** L'utilisateur courant peut-il épingler/désépingler ce post ? */
  canPin: boolean
}

export type PollAudience = 'everyone' | 'followers'

/** Audience des réponses d'un post : tout le monde, ou seulement les abonnés. */
export type ReplyAudience = 'everyone' | 'followers'

export interface PostPollChoice {
  id: string
  label: string
  votesCount: number
  /** Illustration optionnelle du choix (chemin relatif `/media/<id>`). */
  imageUrl?: string
}

export interface PostPoll {
  choices: PostPollChoice[]
  endsAt: string
  closedAt: string
  audience: PollAudience
  totalVotes: number
  votedChoiceId: string
  winnerChoiceIds: string[]
  canViewResults: boolean
  canClose: boolean
}

export interface CreatePollChoiceInput {
  label: string
  /** Chemin relatif `/media/<id>` d'une image déjà uploadée (optionnel). */
  imageUrl?: string
}

export interface CreatePollPayload {
  choices: CreatePollChoiceInput[]
  durationMinutes: number
  audience: PollAudience
}

export interface HashtagTrend {
  tag: string
  count: number
}

export type HashtagPostSort = 'top' | 'recent'

/** Commentaire de profil enrichi du post parent (onglet « Réponses »). */
export interface ReplyContext {
  comment: PostComment
  /** ID du post sur lequel porte le commentaire (pour la navigation). */
  parentPostId: string
  /** Auteur du post parent (pour le libellé « En réponse à @X »). */
  parentPostAuthor: PostAuthor
  /** Post parent complet (rendu au-dessus de la réponse). `null` si supprimé/inaccessible. */
  parentPost: FeedPost | null
  /** Commentaire parent, uniquement si la réponse répond à un autre commentaire
   *  (post → commentaire parent → réponse). `null` sinon. */
  parentComment: PostComment | null
}

/** Commentaire enrichi pour l'affichage. */
export interface PostComment {
  id: string
  postId: string
  /** Vide = commentaire racine ; sinon id du commentaire racine (réponse). */
  parentId: string
  author: PostAuthor
  content: string
  media: PostMedia[]
  likesCount: number
  liked: boolean
  /** Nombre de réponses (pertinent pour un commentaire racine). */
  replyCount: number
  createdAt: string
  canDelete: boolean
}

// --- Enveloppe ---------------------------------------------------------------

async function unwrap<T>(res: Response): Promise<T> {
  const body = (await res.json().catch(() => null)) as {
    data?: T
    error?: string
  } | null
  if (!res.ok) {
    throw new PostApiError(body?.error ?? `Erreur ${res.status}`, res.status)
  }
  return (body?.data ?? null) as T
}

async function expectOk(res: Response, message: string): Promise<void> {
  if (!res.ok) throw new PostApiError(message, res.status)
}

// --- Contexte utilisateur courant (claims JWT) ------------------------------

/** Id de l'utilisateur courant (ou '' si pas de session). Cf. `lib/session`. */
export function currentUserId(): string {
  return decodeClaims()?.user_id ?? ''
}

/** L'utilisateur courant peut-il supprimer un contenu de `authorId` ? */
function canDelete(authorId: string): boolean {
  const claims = decodeClaims()
  if (!claims) return false
  const role = claims.role
  return claims.user_id === authorId || role === 'moderator' || role === 'admin'
}

// --- Résolution d'auteur (mémoïsée) -----------------------------------------

const authorCache = new Map<string, Promise<PostAuthor>>()

function authorFromProfil(profil: ProfilDetails): PostAuthor {
  return {
    id: profil.userId,
    username: profil.username,
    displayName: profil.displayName || profil.username || 'Utilisateur',
    avatarUrl: profil.avatarUrl,
    visibility: profil.visibility,
    lastLoginAt: profil.lastLoginAt,
    isOnline: profil.isOnline,
  }
}

function syncAuthorCacheFromProfil(profil: ProfilDetails): PostAuthor {
  const author = authorFromProfil(profil)
  authorCache.set(profil.userId, Promise.resolve(author))
  return author
}

function applyAuthorUpdateToPost(post: FeedPost, author: PostAuthor): FeedPost {
  const quotedPost = post.quotedPost
    ? applyAuthorUpdateToPost(post.quotedPost, author)
    : null
  const authorChanged = post.author.id === author.id
  const quoteChanged = quotedPost !== post.quotedPost

  if (!authorChanged && !quoteChanged) return post

  return {
    ...post,
    author: authorChanged ? author : post.author,
    quotedPost,
  }
}

export function applyProfilUpdateToPosts(
  posts: FeedPost[],
  profil: ProfilDetails,
): FeedPost[] {
  const author = syncAuthorCacheFromProfil(profil)
  return posts.map((post) => applyAuthorUpdateToPost(post, author))
}

async function fetchUser(userId: string): Promise<ApiUser | null> {
  const res = await apiFetch(`/users/${userId}`)
  if (!res.ok) return null
  const body = (await res.json().catch(() => null)) as {
    data?: ApiUser
  } | null
  return body?.data ?? null
}

async function fetchProfil(userId: string): Promise<ApiProfil | null> {
  const res = await apiFetch(`/profils/${userId}`)
  if (!res.ok) return null
  const body = (await res.json().catch(() => null)) as {
    data?: ApiProfil
  } | null
  return body?.data ?? null
}

/** Résout (et cache) l'auteur : username + décoratif, repli si absent. */
function resolveAuthor(userId: string): Promise<PostAuthor> {
  const cached = authorCache.get(userId)
  if (cached) return cached

  const promise = (async (): Promise<PostAuthor> => {
    const [user, profil] = await Promise.all([
      fetchUser(userId),
      fetchProfil(userId),
    ])
    return {
      id: userId,
      username: user?.username ?? '',
      displayName:
        profil?.display_name?.trim() || user?.username || 'Utilisateur',
      avatarUrl: resolveMediaUrl(profil?.avatar_url),
      visibility: profil?.visibility === 'private' ? 'private' : 'public',
      lastLoginAt: profil?.last_login_at ?? '',
      isOnline: Boolean(profil?.is_online),
    }
  })()

  authorCache.set(userId, promise)
  return promise
}

// --- Mapping post / commentaire ---------------------------------------------

async function toFeedPost(
  p: ApiPost,
  likedIds: Set<string>,
  repostedIds: Set<string>,
  bookmarkedIds: Set<string> = new Set(),
  depth = 0,
): Promise<FeedPost> {
  const quotedPost =
    p.quote_post_id && depth < 1
      ? await getPost(
          p.quote_post_id,
          likedIds,
          repostedIds,
          bookmarkedIds,
          depth + 1,
        )
      : null
  return {
    id: p.id,
    author: await resolveAuthor(p.author_id),
    content: p.content,
    hashtags: p.hashtags ?? [],
    media: (p.media ?? []).map((m) => ({
      url: resolveMediaUrl(m.url),
      type: m.type,
    })),
    poll: p.poll ? toPostPoll(p.poll) : null,
    quotePostId: p.quote_post_id ?? '',
    quotedPost,
    replyAudience: p.reply_audience === 'followers' ? 'followers' : 'everyone',
    // Absent dans la réponse = non restreint ⇒ on autorise (le serveur reste
    // autoritaire à l'écriture). Seul un `false` explicite désactive le composer.
    canReply: p.can_reply !== false,
    likesCount: p.likes_count ?? 0,
    commentsCount: p.comments_count ?? 0,
    repostsCount: p.reposts_count ?? 0,
    pinnedAt: p.pinned_at ?? '',
    repostedById: p.reposted_by_id ?? '',
    repostedAt: p.reposted_at ?? '',
    createdAt: p.created_at,
    isPinned: Boolean(p.pinned_at),
    liked: likedIds.has(p.id),
    reposted: repostedIds.has(p.id),
    bookmarked: bookmarkedIds.has(p.id),
    canDelete: !p.reposted_by_id && canDelete(p.author_id),
    canPin: currentUserId() === p.author_id,
  }
}

function toPostPoll(poll: ApiPoll): PostPoll {
  return {
    choices: (poll.choices ?? []).map((choice) => ({
      id: choice.id,
      label: choice.label,
      votesCount: choice.votes_count ?? 0,
      imageUrl: choice.image_url || undefined,
    })),
    endsAt: poll.ends_at,
    closedAt: poll.closed_at ?? '',
    audience: poll.audience === 'followers' ? 'followers' : 'everyone',
    totalVotes: poll.total_votes ?? 0,
    votedChoiceId: poll.voted_choice_id ?? '',
    winnerChoiceIds: poll.winner_choice_ids ?? [],
    canViewResults: Boolean(poll.can_view_results),
    canClose: Boolean(poll.can_close),
  }
}

async function toComment(c: ApiComment): Promise<PostComment> {
  return {
    id: c.id,
    postId: c.post_id,
    parentId: c.parent_id ?? '',
    author: await resolveAuthor(c.author_id),
    content: c.content,
    media: (c.media ?? []).map((m) => ({
      url: resolveMediaUrl(m.url),
      type: m.type,
    })),
    likesCount: c.likes_count ?? 0,
    liked: Boolean(c.liked),
    replyCount: c.reply_count ?? 0,
    createdAt: c.created_at,
    canDelete: canDelete(c.author_id),
  }
}

// --- Posts -------------------------------------------------------------------

/** Ids des posts likés par l'utilisateur courant (pour l'état des cœurs). */
export async function getLikedIds(): Promise<Set<string>> {
  // Visiteur (pas de token) : ces routes sont protégées → un 401 déclencherait
  // une redirection forcée vers /login. On court-circuite (aucun like de toute façon).
  if (!getAccessToken()) return new Set()
  const res = await apiFetch('/posts/me/liked-ids')
  if (!res.ok) return new Set()
  const ids = await unwrap<string[]>(res)
  return new Set(ids ?? [])
}

/** Ids des posts repostés par l'utilisateur courant. */
export async function getRepostedIds(): Promise<Set<string>> {
  if (!getAccessToken()) return new Set()
  const res = await apiFetch('/posts/me/reposted-ids')
  if (!res.ok) return new Set()
  const ids = await unwrap<string[]>(res)
  return new Set(ids ?? [])
}

/** Ids des posts signés par l'utilisateur courant (état des boutons signet). */
export async function getBookmarkedIds(): Promise<Set<string>> {
  if (!getAccessToken()) return new Set()
  const res = await apiFetch('/posts/me/bookmarked-ids')
  if (!res.ok) return new Set()
  const ids = await unwrap<string[]>(res)
  return new Set(ids ?? [])
}

async function getPost(
  id: string,
  likedIds = new Set<string>(),
  repostedIds = new Set<string>(),
  bookmarkedIds = new Set<string>(),
  depth = 0,
): Promise<FeedPost | null> {
  const res = await apiFetch(`/posts/${id}`)
  if (!res.ok) return null
  const raw = await unwrap<ApiPost>(res)
  return toFeedPost(raw, likedIds, repostedIds, bookmarkedIds, depth)
}

/** Charge un post complet par son id (page détail, lien depuis une notification). */
export async function getPostById(id: string): Promise<FeedPost | null> {
  const [likedIds, repostedIds, bookmarkedIds] = await Promise.all([
    getLikedIds(),
    getRepostedIds(),
    getBookmarkedIds(),
  ])
  return getPost(id, likedIds, repostedIds, bookmarkedIds)
}

/**
 * Mappe une liste brute de posts vers des `FeedPost` enrichis (auteur résolu +
 * état liké/reposté/signé de l'utilisateur courant). Exporté pour les vues qui
 * lisent des posts via d'autres endpoints du post-service (ex. signets).
 */
export async function mapPosts(raw: ApiPost[]): Promise<FeedPost[]> {
  const [likedIds, repostedIds, bookmarkedIds] = await Promise.all([
    getLikedIds(),
    getRepostedIds(),
    getBookmarkedIds(),
  ])
  return Promise.all(
    (raw ?? []).map((p) => toFeedPost(p, likedIds, repostedIds, bookmarkedIds)),
  )
}

/** Fil global (« Pour toi »), paginé. */
export async function listFeed(
  limit = 20,
  offset = 0,
  hashtag = '',
  sort: HashtagPostSort = 'recent',
): Promise<FeedPost[]> {
  const params = feedParams(limit, offset, hashtag, sort)
  const raw = await unwrap<ApiPost[]>(await apiFetch(`/posts?${params}`))
  return mapPosts(raw)
}

/** Fil « Abonnements » : posts des comptes suivis (ids fournis par user-service). */
export async function listFollowingFeed(
  limit = 20,
  offset = 0,
  hashtag = '',
  sort: HashtagPostSort = 'recent',
): Promise<FeedPost[]> {
  const ids = await followingIds()
  if (ids.length === 0) return []
  const params = feedParams(limit, offset, hashtag, sort)
  params.set('author_ids', ids.join(','))
  const raw = await unwrap<ApiPost[]>(await apiFetch(`/posts?${params}`))
  return mapPosts(raw)
}

/** Posts d'un auteur (onglet « Posts » d'un profil). */
export async function listByAuthor(
  authorId: string,
  limit = 20,
  offset = 0,
): Promise<FeedPost[]> {
  const params = feedParams(limit, offset)
  params.set('author_id', authorId)
  const raw = await unwrap<ApiPost[]>(await apiFetch(`/posts?${params}`))
  return mapPosts(raw)
}

/** Posts likés par un utilisateur (403 si likes privés et non-propriétaire). */
export async function listLikedByUser(
  authorId: string,
  limit = 20,
  offset = 0,
): Promise<FeedPost[]> {
  const params = new URLSearchParams()
  params.set('author_id', authorId)
  params.set('limit', String(limit))
  params.set('offset', String(offset))
  const res = await apiFetch(`/posts/liked?${params}`)
  if (res.status === 403)
    throw Object.assign(new Error('likes_private'), { status: 403 })
  const raw = await unwrap<ApiPost[]>(res)
  return mapPosts(raw ?? [])
}

export async function listHashtagTrends(
  limit = 5,
  query = '',
): Promise<HashtagTrend[]> {
  const params = new URLSearchParams()
  params.set('limit', String(limit))
  const q = query.trim().replace(/^#/, '')
  if (q) params.set('q', q)
  const raw = await unwrap<HashtagTrend[]>(
    await apiFetch(`/posts/trends?${params}`),
  )
  return raw ?? []
}

/** Posts contenant au moins un hashtag, utilisés par la page Explorer. */
export async function listHashtaggedPosts(
  limit = 10,
  offset = 0,
): Promise<FeedPost[]> {
  const params = feedParams(limit, offset)
  params.set('hashtag_any', 'true')
  const raw = await unwrap<ApiPost[]>(await apiFetch(`/posts?${params}`))
  return mapPosts(raw)
}

/** Crée un post (auteur dérivé du JWT côté back). */
export async function createPost(
  content: string,
  media: PostMedia[] = [],
  quotePostId?: string,
  poll?: CreatePollPayload,
  replyAudience: ReplyAudience = 'everyone',
): Promise<FeedPost> {
  const payload: Record<string, unknown> = { content }
  if (media.length > 0) payload.media = media
  if (quotePostId) payload.quote_post_id = quotePostId
  // `everyone` est le défaut serveur → n'envoyer le champ que s'il est restreint.
  if (replyAudience === 'followers') payload.reply_audience = replyAudience
  if (poll) {
    payload.poll = {
      choices: poll.choices.map((c) => ({
        label: c.label,
        // N'envoyer image_url que si défini (choix texte seul = champ absent).
        ...(c.imageUrl ? { image_url: c.imageUrl } : {}),
      })),
      duration_minutes: poll.durationMinutes,
      audience: poll.audience,
    }
  }

  const created = await unwrap<ApiPost>(
    await apiFetch('/posts', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    }),
  )
  return toFeedPost(created, new Set(), new Set())
}

export async function votePoll(
  postId: string,
  choiceId: string,
): Promise<FeedPost> {
  const updated = await unwrap<ApiPost>(
    await apiFetch(`/posts/${postId}/poll/vote`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ choice_id: choiceId }),
    }),
  )
  const [likedIds, repostedIds, bookmarkedIds] = await Promise.all([
    getLikedIds(),
    getRepostedIds(),
    getBookmarkedIds(),
  ])
  return toFeedPost(updated, likedIds, repostedIds, bookmarkedIds)
}

export async function closePoll(postId: string): Promise<FeedPost> {
  const updated = await unwrap<ApiPost>(
    await apiFetch(`/posts/${postId}/poll/close`, { method: 'POST' }),
  )
  const [likedIds, repostedIds, bookmarkedIds] = await Promise.all([
    getLikedIds(),
    getRepostedIds(),
    getBookmarkedIds(),
  ])
  return toFeedPost(updated, likedIds, repostedIds, bookmarkedIds)
}

/** Épingle un post sur le profil de l'auteur courant ; renvoie le post à jour. */
export async function pinPost(id: string): Promise<FeedPost> {
  const updated = await unwrap<ApiPost>(
    await apiFetch(`/posts/${id}/pin`, { method: 'PATCH' }),
  )
  const [likedIds, repostedIds, bookmarkedIds] = await Promise.all([
    getLikedIds(),
    getRepostedIds(),
    getBookmarkedIds(),
  ])
  return toFeedPost(updated, likedIds, repostedIds, bookmarkedIds)
}

/** Désépingle un post ; renvoie le post à jour. */
export async function unpinPost(id: string): Promise<FeedPost> {
  const updated = await unwrap<ApiPost>(
    await apiFetch(`/posts/${id}/pin`, { method: 'DELETE' }),
  )
  const [likedIds, repostedIds, bookmarkedIds] = await Promise.all([
    getLikedIds(),
    getRepostedIds(),
    getBookmarkedIds(),
  ])
  return toFeedPost(updated, likedIds, repostedIds, bookmarkedIds)
}

/** Supprime un post (auteur ou mod/admin côté back). */
export async function deletePost(id: string): Promise<void> {
  await expectOk(
    await apiFetch(`/posts/${id}`, { method: 'DELETE' }),
    'Suppression impossible',
  )
}

// --- Likes -------------------------------------------------------------------

/** Like un post ; renvoie le nombre de likes à jour. */
export async function likePost(id: string): Promise<number> {
  const data = await unwrap<{ likes_count: number }>(
    await apiFetch(`/posts/${id}/like`, { method: 'POST' }),
  )
  return data.likes_count
}

/** Retire le like ; renvoie le nombre de likes à jour. */
export async function unlikePost(id: string): Promise<number> {
  const data = await unwrap<{ likes_count: number }>(
    await apiFetch(`/posts/${id}/like`, { method: 'DELETE' }),
  )
  return data.likes_count
}

// --- Reposts -----------------------------------------------------------------

/** Repost simple ; renvoie le post original annoté pour affichage profil. */
export async function repostPost(id: string): Promise<FeedPost> {
  const updated = await unwrap<ApiPost>(
    await apiFetch(`/posts/${id}/repost`, { method: 'POST' }),
  )
  const [likedIds, repostedIds, bookmarkedIds] = await Promise.all([
    getLikedIds(),
    getRepostedIds(),
    getBookmarkedIds(),
  ])
  return toFeedPost(updated, likedIds, repostedIds, bookmarkedIds)
}

/** Retire le repost ; renvoie le nombre de reposts à jour. */
export async function unrepostPost(id: string): Promise<number> {
  const data = await unwrap<{ reposts_count: number }>(
    await apiFetch(`/posts/${id}/repost`, { method: 'DELETE' }),
  )
  return data.reposts_count
}

// --- Commentaires ------------------------------------------------------------

/** Commentaires RACINE d'un post (chronologiques, paginés). */
export async function listComments(
  postId: string,
  limit = 10,
  offset = 0,
): Promise<PostComment[]> {
  const raw = await unwrap<ApiComment[]>(
    await apiFetch(`/posts/${postId}/comments?limit=${limit}&offset=${offset}`),
  )
  return Promise.all((raw ?? []).map(toComment))
}

/** Réponses d'un commentaire racine (chronologiques, paginées). */
export async function listReplies(
  postId: string,
  commentId: string,
  limit = 10,
  offset = 0,
): Promise<PostComment[]> {
  const raw = await unwrap<ApiComment[]>(
    await apiFetch(
      `/posts/${postId}/comments/${commentId}/replies?limit=${limit}&offset=${offset}`,
    ),
  )
  return Promise.all((raw ?? []).map(toComment))
}

/** Commentaires écrits par un utilisateur, enrichis du post parent (onglet « Réponses »). */
export async function listCommentsByAuthor(
  authorId: string,
  limit = 20,
  offset = 0,
): Promise<ReplyContext[]> {
  const raw = await unwrap<ApiCommentWithPost[]>(
    await apiFetch(
      `/posts/comments?author_id=${encodeURIComponent(authorId)}&limit=${limit}&offset=${offset}`,
    ),
  )
  if (!raw) return []
  const fallbackAuthor: PostAuthor = {
    id: '',
    username: '',
    displayName: '...',
    avatarUrl: '',
    visibility: 'public',
    lastLoginAt: '',
    isOnline: false,
  }
  // États (liké/reposté/signé) de l'utilisateur courant, récupérés une seule
  // fois pour enrichir tous les posts parents.
  const [likedIds, repostedIds, bookmarkedIds] = await Promise.all([
    getLikedIds(),
    getRepostedIds(),
    getBookmarkedIds(),
  ])
  return Promise.all(
    raw.map(async (item) => {
      const comment = await toComment(item)
      const parentPost = item.parent_post
        ? await toFeedPost(
            item.parent_post,
            likedIds,
            repostedIds,
            bookmarkedIds,
          )
        : null
      const parentPostAuthor = item.parent_post
        ? await resolveAuthor(item.parent_post.author_id)
        : fallbackAuthor
      const parentComment = item.parent_comment
        ? await toComment(item.parent_comment)
        : null
      return {
        comment,
        parentPostId: item.post_id,
        parentPostAuthor,
        parentPost,
        parentComment,
      }
    }),
  )
}

/**
 * Ajoute un commentaire (auteur dérivé du JWT côté back). `parentId` fourni →
 * c'est une réponse (rattachée à plat à la racine côté back).
 */
export async function createComment(
  postId: string,
  content: string,
  parentId?: string,
  media: PostMedia[] = [],
): Promise<PostComment> {
  const payload: Record<string, unknown> = { content }
  if (parentId) payload.parent_id = parentId
  if (media.length > 0) payload.media = media

  const created = await unwrap<ApiComment>(
    await apiFetch(`/posts/${postId}/comments`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    }),
  )
  return toComment(created)
}

/** Supprime un commentaire (auteur ou mod/admin côté back). */
export async function deleteComment(
  postId: string,
  commentId: string,
): Promise<void> {
  await expectOk(
    await apiFetch(`/posts/${postId}/comments/${commentId}`, {
      method: 'DELETE',
    }),
    'Suppression impossible',
  )
}

/** Like un commentaire ; renvoie le nombre de likes à jour. */
export async function likeComment(
  postId: string,
  commentId: string,
): Promise<number> {
  const data = await unwrap<{ likes_count: number }>(
    await apiFetch(`/posts/${postId}/comments/${commentId}/like`, {
      method: 'POST',
    }),
  )
  return data.likes_count
}

/** Retire le like d'un commentaire ; renvoie le nombre de likes à jour. */
export async function unlikeComment(
  postId: string,
  commentId: string,
): Promise<number> {
  const data = await unwrap<{ likes_count: number }>(
    await apiFetch(`/posts/${postId}/comments/${commentId}/like`, {
      method: 'DELETE',
    }),
  )
  return data.likes_count
}

// --- Graphe social (ids suivis pour le fil « Abonnements ») ------------------

async function followingIds(): Promise<string[]> {
  const me = currentUserId()
  if (!me) return []
  const res = await apiFetch(`/users/${me}/following?limit=200`)
  if (!res.ok) return []
  const users = await unwrap<{ id: string }[]>(res)
  return (users ?? []).map((u) => u.id)
}

function feedParams(
  limit: number,
  offset: number,
  hashtag = '',
  sort: HashtagPostSort = 'recent',
): URLSearchParams {
  const params = new URLSearchParams()
  params.set('limit', String(limit))
  params.set('offset', String(offset))
  const tag = hashtag.trim().replace(/^#/, '')
  if (tag) {
    params.set('hashtag', tag)
    params.set('sort', sort)
  }
  return params
}

// --- Broadcast « post créé » (le fil prépend sans refetch) -------------------

const POST_CREATED_EVENT = 'breezy:post-created'

export function notifyPostCreated(post: FeedPost): void {
  if (typeof window === 'undefined') return
  window.dispatchEvent(
    new CustomEvent<FeedPost>(POST_CREATED_EVENT, { detail: post }),
  )
}

export function subscribePostCreated(
  onCreate: (post: FeedPost) => void,
): () => void {
  function handle(event: Event) {
    onCreate((event as CustomEvent<FeedPost>).detail)
  }
  window.addEventListener(POST_CREATED_EVENT, handle)
  return () => window.removeEventListener(POST_CREATED_EVENT, handle)
}

/** Résout (et cache) l'auteur d'un post par son id — sert à l'aperçu temps réel
 * (avatar/nom du bandeau « a posté ») sans avoir à charger le post complet. */
export function getPostAuthor(userId: string): Promise<PostAuthor> {
  return resolveAuthor(userId)
}

// --- Compteurs dynamiques (polling périodique) ------------------------------
// Façon X : les compteurs (likes/commentaires/reposts) des posts AFFICHÉS sont
// rafraîchis par lots toutes les quelques secondes, sans recharger les posts ni
// toucher l'état « moi » (liked/reposted/bookmarked, piloté par les actions
// locales). Lecture seule via le fil authentifié normal → barrière de visibilité
// server-side (les posts invisibles sont simplement absents de la réponse).

/** Intervalle de rafraîchissement des compteurs (ms) — fil/profil (plusieurs posts). */
export const STATS_POLL_INTERVAL_MS = 7000

/** Intervalle plus court sur la vue détail (un seul post → coût négligeable, plus vif). */
export const STATS_POLL_INTERVAL_DETAIL_MS = 3500

/** Borne du lot d'ids par requête (alignée sur la borne serveur). */
const STATS_BATCH_MAX = 100

/** Compteurs dénormalisés d'un post (réponse de GET /posts/stats). */
export interface PostStats {
  likesCount: number
  commentsCount: number
  repostsCount: number
}

/** Compteurs dénormalisés d'un commentaire (likes uniquement). */
export interface CommentStats {
  likesCount: number
}

interface ApiPostStat {
  id: string
  likes_count: number
  comments_count: number
  reposts_count: number
}

interface ApiCommentStat {
  id: string
  likes_count: number
}

/**
 * Récupère les compteurs des posts demandés en un seul lot. Renvoie une map
 * `id → compteurs` ; les posts invisibles/supprimés en sont absents. Le lot est
 * borné à `STATS_BATCH_MAX` ids.
 */
export async function getPostsStats(
  ids: string[],
): Promise<Map<string, PostStats>> {
  const out = new Map<string, PostStats>()
  const wanted = Array.from(new Set(ids.filter(Boolean))).slice(
    0,
    STATS_BATCH_MAX,
  )
  if (wanted.length === 0) return out
  const params = new URLSearchParams({ ids: wanted.join(',') })
  const stats = await unwrap<ApiPostStat[]>(
    await apiFetch(`/posts/stats?${params}`),
  )
  for (const s of stats ?? []) {
    out.set(s.id, {
      likesCount: s.likes_count ?? 0,
      commentsCount: s.comments_count ?? 0,
      repostsCount: s.reposts_count ?? 0,
    })
  }
  return out
}

/**
 * Récupère les compteurs des commentaires demandés en un seul lot.
 * Renvoie une map `id → likes`.
 */
export async function getCommentsStats(
  ids: string[],
): Promise<Map<string, CommentStats>> {
  const out = new Map<string, CommentStats>()
  const wanted = Array.from(new Set(ids.filter(Boolean))).slice(
    0,
    STATS_BATCH_MAX,
  )
  if (wanted.length === 0) return out
  const params = new URLSearchParams({ ids: wanted.join(',') })
  const stats = await unwrap<ApiCommentStat[]>(
    await apiFetch(`/posts/comments/stats?${params}`),
  )
  for (const s of stats ?? []) {
    out.set(s.id, { likesCount: s.likes_count ?? 0 })
  }
  return out
}

/**
 * Applique des compteurs frais à un post SANS toucher son état « moi »
 * (liked/reposted/bookmarked). Renvoie le post inchangé (même référence) si
 * aucun compteur n'a bougé — évite les re-rendus inutiles.
 */
export function applyStatsToPost(
  post: FeedPost,
  stats: Map<string, PostStats>,
): FeedPost {
  const s = stats.get(post.id)
  if (!s) return post
  if (
    post.likesCount === s.likesCount &&
    post.commentsCount === s.commentsCount &&
    post.repostsCount === s.repostsCount
  ) {
    return post
  }
  return {
    ...post,
    likesCount: s.likesCount,
    commentsCount: s.commentsCount,
    repostsCount: s.repostsCount,
  }
}

/**
 * Applique des compteurs frais à une liste de posts. Renvoie la MÊME liste
 * (même référence) si aucun post n'a changé — un cycle de polling sans nouveauté
 * ne provoque alors aucun re-rendu.
 */
export function applyStatsToPosts(
  posts: FeedPost[],
  stats: Map<string, PostStats>,
): FeedPost[] {
  let changed = false
  const next = posts.map((p) => {
    const updated = applyStatsToPost(p, stats)
    if (updated !== p) changed = true
    return updated
  })
  return changed ? next : posts
}

// --- Fil temps réel (WebSocket) ---------------------------------------------
// Le post-service diffuse un « ping » léger à chaque nouveau post PUBLIC racine
// (id du post + id de l'auteur). On ne charge PAS le contenu via le WS : le fil
// authentifié normal reste la source (barrière de visibilité server-side). Le
// ping sert seulement à afficher le bandeau « X a posté » en haut du fil.

/** Ping minimal reçu du serveur à la création d'un post. */
export interface NewPostPing {
  postId: string
  authorId: string
}

/** Poignée de connexion temps réel du fil (fermeture propre). */
export interface FeedRealtimeHandle {
  close(): void
}

/**
 * Ouvre la connexion temps réel du fil (`/posts/ws`). Reconnexion automatique
 * avec backoff linéaire (calquée sur connectNotifications). Le token transite en
 * query param (le navigateur n'autorise pas d'en-tête sur un upgrade WS).
 */
export function connectFeedRealtime(
  onNewPost: (ping: NewPostPing) => void,
): FeedRealtimeHandle {
  let socket: WebSocket | null = null
  let closedByUs = false
  let retry = 0

  const connect = () => {
    const token = getAccessToken()
    if (!token) return
    const wsBase = API_URL.replace(/^http/, 'ws').replace(/\/+$/, '')
    socket = new WebSocket(
      `${wsBase}/posts/ws?access_token=${encodeURIComponent(token)}`,
    )

    socket.onmessage = (event) => {
      let payload: { type?: string; post_id?: string; author_id?: string }
      try {
        payload = JSON.parse(event.data as string)
      } catch {
        return
      }
      if (
        payload.type === 'post_created' &&
        payload.post_id &&
        payload.author_id
      ) {
        onNewPost({ postId: payload.post_id, authorId: payload.author_id })
      }
    }

    socket.onopen = () => {
      retry = 0
    }

    socket.onclose = () => {
      if (closedByUs) return
      retry = Math.min(retry + 1, 10)
      setTimeout(connect, retry * 1000)
    }
  }

  connect()

  return {
    close() {
      closedByUs = true
      socket?.close()
    },
  }
}
