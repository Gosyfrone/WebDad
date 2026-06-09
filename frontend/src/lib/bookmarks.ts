/**
 * Client typé des signets (bookmarks) du post-service via l'API Gateway.
 *
 * Les signets sont organisés en **collections** (many-to-many : un post peut être
 * rangé dans plusieurs collections). Tous les appels passent par `apiFetch`
 * (Bearer + refresh single-flight) et sont strictement privés (JWT).
 *
 * Modèle de « rafale » côté serveur : un clic court (`quickBookmark`) range
 * automatiquement dans la dernière collection si l'utilisateur enchaîne dans la
 * fenêtre glissante (statut `filed`) ; sinon le serveur renvoie `needs_choice`
 * et le front ouvre le sélecteur de collection (cf. CLAUDE.md §5).
 *
 * Les vues qui renvoient des posts (« Tous mes signets », contenu d'une
 * collection) réutilisent le mapper enrichi `mapPosts` de `lib/posts.ts`.
 */

import { apiFetch } from '@/lib/auth-client'
import { mapPosts, PostApiError, type ApiPost, type FeedPost } from '@/lib/posts'

// --- Types -------------------------------------------------------------------

/** Une collection de signets. */
export interface BookmarkCollection {
  id: string
  name: string
  /** Collection par défaut : toujours en tête, non supprimable / non renommable. */
  isDefault: boolean
  itemsCount: number
  createdAt: string
}

interface ApiCollection {
  id: string
  name: string
  is_default?: boolean
  items_count?: number
  created_at: string
}

function toCollection(c: ApiCollection): BookmarkCollection {
  return {
    id: c.id,
    name: c.name,
    isDefault: c.is_default ?? false,
    itemsCount: c.items_count ?? 0,
    createdAt: c.created_at,
  }
}

/**
 * Issue d'un clic court sur le bouton signet.
 *  - `filed`        : rangé automatiquement dans `collection` (fenêtre active) ;
 *  - `needs_choice` : ouverture de rafale → le front ouvre le sélecteur
 *    (`collections` = collections existantes proposées), rien n'est rangé.
 */
export type QuickBookmarkResult =
  | { status: 'filed'; collection: BookmarkCollection }
  | { status: 'needs_choice'; collections: BookmarkCollection[] }

interface ApiBookmarkResult {
  status: 'filed' | 'needs_choice'
  collection?: ApiCollection
  collections?: ApiCollection[]
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

// --- Collections ---------------------------------------------------------------

/** Collections de signets de l'utilisateur courant (avec compteur d'items). */
export async function listCollections(): Promise<BookmarkCollection[]> {
  const raw = await unwrap<ApiCollection[]>(await apiFetch('/posts/bookmarks/collections'))
  return (raw ?? []).map(toCollection)
}

/** Crée une collection. */
export async function createCollection(name: string): Promise<BookmarkCollection> {
  const created = await unwrap<ApiCollection>(
    await apiFetch('/posts/bookmarks/collections', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name }),
    }),
  )
  return toCollection(created)
}

/** Renomme une collection. */
export async function renameCollection(id: string, name: string): Promise<BookmarkCollection> {
  const updated = await unwrap<ApiCollection>(
    await apiFetch(`/posts/bookmarks/collections/${id}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name }),
    }),
  )
  return toCollection(updated)
}

/** Supprime une collection (et ses signets). */
export async function deleteCollection(id: string): Promise<void> {
  await expectOk(
    await apiFetch(`/posts/bookmarks/collections/${id}`, { method: 'DELETE' }),
    'Suppression impossible',
  )
}

// --- Signets -----------------------------------------------------------------

/** Clic court : laisse le serveur résoudre la fenêtre de rafale. */
export async function quickBookmark(postId: string): Promise<QuickBookmarkResult> {
  const r = await unwrap<ApiBookmarkResult>(
    await apiFetch(`/posts/${postId}/bookmark`, { method: 'POST' }),
  )
  if (r.status === 'filed' && r.collection) {
    return { status: 'filed', collection: toCollection(r.collection) }
  }
  return { status: 'needs_choice', collections: (r.collections ?? []).map(toCollection) }
}

/** Range explicitement un post dans une collection (sélecteur / appui long). */
export async function addBookmark(postId: string, collectionId: string): Promise<BookmarkCollection> {
  const r = await unwrap<ApiBookmarkResult>(
    await apiFetch(`/posts/${postId}/bookmark`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ collection_id: collectionId }),
    }),
  )
  return toCollection(r.collection ?? { id: collectionId, name: '', created_at: '' })
}

/** Retire un post d'une collection précise. */
export async function removeBookmark(postId: string, collectionId: string): Promise<void> {
  await expectOk(
    await apiFetch(`/posts/${postId}/bookmark`, {
      method: 'DELETE',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ collection_id: collectionId }),
    }),
    'Retrait impossible',
  )
}

/** Dé-signe : retire un post de TOUTES les collections. */
export async function removeBookmarkEverywhere(postId: string): Promise<void> {
  await expectOk(
    await apiFetch(`/posts/${postId}/bookmark`, { method: 'DELETE' }),
    'Retrait impossible',
  )
}

/** Ids des collections de l'utilisateur contenant ce post (coche le sélecteur). */
export async function getPostCollections(postId: string): Promise<string[]> {
  const ids = await unwrap<string[]>(await apiFetch(`/posts/${postId}/bookmark/collections`))
  return ids ?? []
}

// --- Vues (posts) ------------------------------------------------------------

/** Vue « Tous mes signets » (union dédupliquée), paginée. */
export async function listAllBookmarks(limit = 20, offset = 0): Promise<FeedPost[]> {
  const raw = await unwrap<ApiPost[]>(
    await apiFetch(`/posts/bookmarks?limit=${limit}&offset=${offset}`),
  )
  return mapPosts(raw)
}

/** Posts d'une collection (du plus récemment rangé au plus ancien), paginés. */
export async function listCollectionPosts(
  collectionId: string,
  limit = 20,
  offset = 0,
): Promise<FeedPost[]> {
  const raw = await unwrap<ApiPost[]>(
    await apiFetch(`/posts/bookmarks/collections/${collectionId}/posts?limit=${limit}&offset=${offset}`),
  )
  return mapPosts(raw)
}
