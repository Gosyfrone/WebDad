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
  quote_post_id?: string
  likes_count: number
  comments_count: number
  reposts_count?: number
  pinned_at?: string
  reposted_by_id?: string
  reposted_at?: string
  created_at: string
}

interface ApiComment {
  id: string
  post_id: string
  parent_id?: string
  author_id: string
  content: string
  media?: { url: string; type: 'image' | 'video' }[]
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
}

// --- Types front -------------------------------------------------------------

/** Auteur résolu (identité + décoratif) d'un post / commentaire. */
export interface PostAuthor {
  id: string
  username: string
  displayName: string
  avatarUrl: string
  visibility: 'public' | 'private'
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
  quotePostId: string
  quotedPost: FeedPost | null
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
  /** Nombre de réponses (pertinent pour un commentaire racine). */
  replyCount: number
  createdAt: string
  canDelete: boolean
}

// --- Enveloppe ---------------------------------------------------------------

async function unwrap<T>(res: Response): Promise<T> {
  const body = (await res.json().catch(() => null)) as
    | { data?: T; error?: string }
    | null
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
  const body = (await res.json().catch(() => null)) as { data?: ApiUser } | null
  return body?.data ?? null
}

async function fetchProfil(userId: string): Promise<ApiProfil | null> {
  const res = await apiFetch(`/profils/${userId}`)
  if (!res.ok) return null
  const body = (await res.json().catch(() => null)) as { data?: ApiProfil } | null
  return body?.data ?? null
}

/** Résout (et cache) l'auteur : username + décoratif, repli si absent. */
function resolveAuthor(userId: string): Promise<PostAuthor> {
  const cached = authorCache.get(userId)
  if (cached) return cached

  const promise = (async (): Promise<PostAuthor> => {
    const [user, profil] = await Promise.all([fetchUser(userId), fetchProfil(userId)])
    return {
      id: userId,
      username: user?.username ?? '',
      displayName: profil?.display_name?.trim() || user?.username || 'Utilisateur',
      avatarUrl: resolveMediaUrl(profil?.avatar_url),
      visibility: profil?.visibility === 'private' ? 'private' : 'public',
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
      ? await getPost(p.quote_post_id, likedIds, repostedIds, bookmarkedIds, depth + 1)
      : null
  return {
    id: p.id,
    author: await resolveAuthor(p.author_id),
    content: p.content,
    hashtags: p.hashtags ?? [],
    media: (p.media ?? []).map((m) => ({ url: resolveMediaUrl(m.url), type: m.type })),
    quotePostId: p.quote_post_id ?? '',
    quotedPost,
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

async function toComment(c: ApiComment): Promise<PostComment> {
  return {
    id: c.id,
    postId: c.post_id,
    parentId: c.parent_id ?? '',
    author: await resolveAuthor(c.author_id),
    content: c.content,
    media: (c.media ?? []).map((m) => ({ url: resolveMediaUrl(m.url), type: m.type })),
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
  return Promise.all((raw ?? []).map((p) => toFeedPost(p, likedIds, repostedIds, bookmarkedIds)))
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
export async function listByAuthor(authorId: string, limit = 20, offset = 0): Promise<FeedPost[]> {
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
  if (res.status === 403) throw Object.assign(new Error('likes_private'), { status: 403 })
  const raw = await unwrap<ApiPost[]>(res)
  return mapPosts(raw ?? [])
}

export async function listHashtagTrends(limit = 5, query = ''): Promise<HashtagTrend[]> {
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
export async function listHashtaggedPosts(limit = 10, offset = 0): Promise<FeedPost[]> {
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
): Promise<FeedPost> {
  const payload: Record<string, unknown> = { content }
  if (media.length > 0) payload.media = media
  if (quotePostId) payload.quote_post_id = quotePostId

  const created = await unwrap<ApiPost>(
    await apiFetch('/posts', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    }),
  )
  return toFeedPost(created, new Set(), new Set())
}

/** Épingle un post sur le profil de l'auteur courant ; renvoie le post à jour. */
export async function pinPost(id: string): Promise<FeedPost> {
  const updated = await unwrap<ApiPost>(await apiFetch(`/posts/${id}/pin`, { method: 'PATCH' }))
  const [likedIds, repostedIds, bookmarkedIds] = await Promise.all([
    getLikedIds(),
    getRepostedIds(),
    getBookmarkedIds(),
  ])
  return toFeedPost(updated, likedIds, repostedIds, bookmarkedIds)
}

/** Désépingle un post ; renvoie le post à jour. */
export async function unpinPost(id: string): Promise<FeedPost> {
  const updated = await unwrap<ApiPost>(await apiFetch(`/posts/${id}/pin`, { method: 'DELETE' }))
  const [likedIds, repostedIds, bookmarkedIds] = await Promise.all([
    getLikedIds(),
    getRepostedIds(),
    getBookmarkedIds(),
  ])
  return toFeedPost(updated, likedIds, repostedIds, bookmarkedIds)
}

/** Supprime un post (auteur ou mod/admin côté back). */
export async function deletePost(id: string): Promise<void> {
  await expectOk(await apiFetch(`/posts/${id}`, { method: 'DELETE' }), 'Suppression impossible')
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
  const updated = await unwrap<ApiPost>(await apiFetch(`/posts/${id}/repost`, { method: 'POST' }))
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
export async function listComments(postId: string, limit = 10, offset = 0): Promise<PostComment[]> {
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
    await apiFetch(`/posts/${postId}/comments/${commentId}/replies?limit=${limit}&offset=${offset}`),
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
        ? await toFeedPost(item.parent_post, likedIds, repostedIds, bookmarkedIds)
        : null
      const parentPostAuthor = item.parent_post
        ? await resolveAuthor(item.parent_post.author_id)
        : fallbackAuthor
      const parentComment = item.parent_comment ? await toComment(item.parent_comment) : null
      return { comment, parentPostId: item.post_id, parentPostAuthor, parentPost, parentComment }
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
export async function deleteComment(postId: string, commentId: string): Promise<void> {
  await expectOk(
    await apiFetch(`/posts/${postId}/comments/${commentId}`, { method: 'DELETE' }),
    'Suppression impossible',
  )
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
  window.dispatchEvent(new CustomEvent<FeedPost>(POST_CREATED_EVENT, { detail: post }))
}

export function subscribePostCreated(onCreate: (post: FeedPost) => void): () => void {
  function handle(event: Event) {
    onCreate((event as CustomEvent<FeedPost>).detail)
  }
  window.addEventListener(POST_CREATED_EVENT, handle)
  return () => window.removeEventListener(POST_CREATED_EVENT, handle)
}
