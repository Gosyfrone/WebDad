/**
 * Client typé des services métier (user-service, profil-service) via l'API
 * Gateway.
 *
 * Tous les appels passent par `apiFetch` (auth-client) : Bearer access token,
 * refresh single-flight sur 401, redirection /login si le refresh échoue.
 * ⚠️ À usage CLIENT uniquement (`apiFetch` lit le token en localStorage).
 *
 * Les réponses de l'API sont enveloppées dans `{ data }` (ou `{ error }`) :
 * `unwrap` déplie l'enveloppe et lève une `ApiError` sur statut non-2xx.
 *
 * Convention de noms : l'API renvoie du snake_case (`display_name`,
 * `follower_count`) ; ce module mappe vers le camelCase des types front.
 */

import { apiFetch } from '@/lib/auth-client'
import { resolveMediaUrl } from '@/lib/media'
import type { RelationUser } from '@/types'

/** Erreur d'appel API portant le code HTTP (pour distinguer 401/404/…). */
export class ApiError extends Error {
  status: number
  constructor(message: string, status: number) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

// --- Formes brutes renvoyées par l'API (snake_case) --------------------------

interface ApiUser {
  id: string
  username: string
  is_active: boolean
  created_at: string
  follower_count?: number
  following_count?: number
}

interface ApiProfil {
  user_id: string
  display_name: string
  bio: string
  avatar_url: string
  banner_url: string
}

/** Déplie l'enveloppe `{ data }` ; lève une `ApiError` sur statut non-2xx. */
async function unwrap<T>(res: Response): Promise<T> {
  const body = (await res.json().catch(() => null)) as
    | { data?: T; error?: string }
    | null
  if (!res.ok) {
    throw new ApiError(body?.error ?? `Erreur ${res.status}`, res.status)
  }
  return (body?.data ?? null) as T
}

// --- Utilisateur courant -----------------------------------------------------

/** Identité + compteurs de l'utilisateur courant (`GET /users/me`). */
export interface CurrentUser {
  id: string
  username: string
  followersCount: number
  followingCount: number
  joinedAt: string
}

export async function getMe(): Promise<CurrentUser> {
  const u = await unwrap<ApiUser>(await apiFetch('/users/me'))
  return {
    id: u.id,
    username: u.username,
    followersCount: u.follower_count ?? 0,
    followingCount: u.following_count ?? 0,
    joinedAt: u.created_at,
  }
}

/** Décoratif du profil de l'utilisateur courant (`GET /profils/me`). */
export interface MyProfil {
  displayName: string
  bio: string
  avatarUrl: string
  bannerUrl: string
}

/** Renvoie `null` si le profil n'existe pas encore (404). */
export async function getProfilMe(): Promise<MyProfil | null> {
  const res = await apiFetch('/profils/me')
  if (res.status === 404) return null
  const p = await unwrap<ApiProfil>(res)
  return {
    displayName: p.display_name,
    bio: p.bio,
    avatarUrl: resolveMediaUrl(p.avatar_url),
    bannerUrl: resolveMediaUrl(p.banner_url),
  }
}

/** Profil décoratif d'un utilisateur donné (best-effort : `null` si absent). */
async function getProfil(userId: string): Promise<ApiProfil | null> {
  const res = await apiFetch(`/profils/${userId}`)
  if (!res.ok) return null
  const body = (await res.json().catch(() => null)) as { data?: ApiProfil } | null
  return body?.data ?? null
}

/** Identité d'un utilisateur donné (best-effort : `null` si absent). */
async function getUser(userId: string): Promise<ApiUser | null> {
  const res = await apiFetch(`/users/${userId}`)
  if (!res.ok) return null
  const body = (await res.json().catch(() => null)) as { data?: ApiUser } | null
  return body?.data ?? null
}

/** ApiUser → RelationUser : enrichit l'identité du décoratif (profil-service). */
async function enrichFromUser(u: ApiUser): Promise<RelationUser> {
  const p = await getProfil(u.id)
  return {
    id: u.id,
    username: u.username,
    displayName: p?.display_name?.trim() || u.username,
    bio: p?.bio ?? '',
    avatarUrl: resolveMediaUrl(p?.avatar_url),
  }
}

/** ApiProfil → RelationUser : récupère l'username (user-service) ; null si absent. */
async function enrichFromProfil(p: ApiProfil): Promise<RelationUser | null> {
  const u = await getUser(p.user_id)
  if (!u) return null
  return {
    id: u.id,
    username: u.username,
    displayName: p.display_name?.trim() || u.username,
    bio: p.bio ?? '',
    avatarUrl: resolveMediaUrl(p.avatar_url),
  }
}

// --- Graphe social -----------------------------------------------------------

export type RelationKind = 'followers' | 'following'

/**
 * Liste les abonnés / abonnements d'un utilisateur, enrichis du décoratif via
 * profil-service. L'enrichissement est best-effort (N+1, acceptable à
 * l'échelle du projet) : si le profil manque, on retombe sur le username.
 */
export async function listRelations(
  userId: string,
  kind: RelationKind,
): Promise<RelationUser[]> {
  const users = await unwrap<ApiUser[]>(
    await apiFetch(`/users/${userId}/${kind}?limit=50`),
  )
  return Promise.all((users ?? []).map(enrichFromUser))
}

/**
 * Aperçu des abonnés communs : personnes qui suivent `targetUserId` et que
 * l'utilisateur courant suit aussi. Composition front sur les routes existantes.
 */
export async function getCommonFollowers(
  targetUserId: string,
  limit = 3,
): Promise<RelationUser[]> {
  const me = await getMe()
  if (me.id === targetUserId) return []

  const [followers, myFollowingIds] = await Promise.all([
    listRelations(targetUserId, 'followers'),
    getFollowingIds(me.id),
  ])

  return followers
    .filter((user) => user.id !== me.id && myFollowingIds.has(user.id))
    .slice(0, limit)
}

/**
 * Recherche d'utilisateurs. Le préfixe `@` bascule sur la recherche par
 * identifiant (user-service) ; sinon recherche par nom affiché (profil-service).
 * Dans les deux cas, le résultat est enrichi pour porter username + décoratif.
 */
export async function searchUsers(query: string): Promise<RelationUser[]> {
  const q = query.trim()
  if (!q) return []

  if (q.startsWith('@')) {
    const term = q.slice(1).trim()
    if (!term) return []
    const users = await unwrap<ApiUser[]>(
      await apiFetch(`/users/search?q=${encodeURIComponent(term)}`),
    )
    return Promise.all((users ?? []).map(enrichFromUser))
  }

  const profils = await unwrap<ApiProfil[]>(
    await apiFetch(`/profils/search?q=${encodeURIComponent(q)}`),
  )
  const enriched = await Promise.all((profils ?? []).map(enrichFromProfil))
  return enriched.filter((u): u is RelationUser => u !== null)
}

/**
 * Résout un utilisateur par son handle exact (`GET /users/by-username`), enrichi
 * du décoratif. `null` si le handle n'existe pas (404). Sert à la carte d'aperçu
 * d'une mention de non-membre dans la messagerie.
 */
export async function getUserByUsername(username: string): Promise<RelationUser | null> {
  const handle = username.trim()
  if (!handle) return null
  const res = await apiFetch(`/users/by-username/${encodeURIComponent(handle)}`)
  if (!res.ok) return null
  const body = (await res.json().catch(() => null)) as { data?: ApiUser } | null
  if (!body?.data) return null
  return enrichFromUser(body.data)
}

/** Comptes les plus suivis (« Qui suivre »), enrichis du décoratif. */
export async function getSuggestions(limit = 10): Promise<RelationUser[]> {
  const users = await unwrap<ApiUser[]>(
    await apiFetch(`/users/suggestions?limit=${limit}`),
  )
  return Promise.all((users ?? []).map(enrichFromUser))
}

/**
 * Ensemble des ids suivis par `userId` — sert à initialiser correctement
 * l'état des boutons Suivre/Abonné (l'API n'expose pas de flag `is_following`).
 */
export async function getFollowingIds(userId: string): Promise<Set<string>> {
  const users = await unwrap<ApiUser[]>(
    await apiFetch(`/users/${userId}/following?limit=200`),
  )
  return new Set((users ?? []).map((u) => u.id))
}

export type FollowStatus = 'following' | 'pending'

/** Suit un utilisateur ou crée une demande si son profil est privé. */
export async function follow(userId: string): Promise<FollowStatus> {
  const res = await apiFetch(`/users/${userId}/follow`, { method: 'POST' })
  if (!res.ok) throw new ApiError('Suivi impossible', res.status)
  const body = (await res.json().catch(() => null)) as { data?: { status?: FollowStatus } } | null
  return body?.data?.status ?? 'following'
}

export async function getPendingFollowRequestIds(): Promise<Set<string>> {
  const ids = await unwrap<string[]>(
    await apiFetch('/users/me/follow-requests/outgoing'),
  )
  return new Set(ids ?? [])
}

/** Se désabonne (`DELETE /users/:id/follow`, idempotent côté API). */
export async function unfollow(userId: string): Promise<void> {
  const res = await apiFetch(`/users/${userId}/follow`, { method: 'DELETE' })
  if (!res.ok) throw new ApiError('Désabonnement impossible', res.status)
}

export async function acceptFollowRequest(followerId: string): Promise<void> {
  const res = await apiFetch(`/users/follow-requests/${followerId}/accept`, { method: 'POST' })
  if (!res.ok) throw new ApiError('Acceptation impossible', res.status)
}

export async function rejectFollowRequest(followerId: string): Promise<void> {
  const res = await apiFetch(`/users/follow-requests/${followerId}/reject`, { method: 'POST' })
  if (!res.ok) throw new ApiError('Refus impossible', res.status)
}

/** Retire un utilisateur de mes abonnés (`DELETE /users/me/followers/:id`). */
export async function removeFollower(userId: string): Promise<void> {
  const res = await apiFetch(`/users/me/followers/${userId}`, { method: 'DELETE' })
  if (!res.ok) throw new ApiError('Retrait impossible', res.status)
}
