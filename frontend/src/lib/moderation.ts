/**
 * Client de modération (réservé aux rôles modérateur + administrateur).
 *
 * Corbeille des tweets retirés en « suppression douce » : quand un mod/admin
 * supprime le post d'un AUTRE utilisateur, le post-service le MASQUE
 * (`is_hidden`) au lieu de l'effacer → il sort des fils mais reste restaurable
 * ici. Trois actions : lister la corbeille, restaurer, purger (effacer pour de
 * bon). L'auteur qui supprime SON post déclenche, lui, une vraie suppression.
 *
 * L'auteur du post et le modérateur qui l'a retiré sont enrichis côté front
 * (`resolveUser`, mémoïsé) — même pattern cross-service que l'annuaire admin.
 *
 * ⚠️ À usage CLIENT uniquement (`apiFetch` lit le token en localStorage).
 */

import { apiFetch } from '@/lib/auth-client'
import { resolveMediaUrl } from '@/lib/media'

/** Erreur d'appel API de modération portant le code HTTP. */
export class ModerationApiError extends Error {
  status: number
  constructor(message: string, status: number) {
    super(message)
    this.name = 'ModerationApiError'
    this.status = status
  }
}

interface ApiHiddenPost {
  id: string
  author_id: string
  content: string
  media?: { url: string; type: 'image' | 'video' }[]
  hidden_by?: string
  hidden_at?: string
  purge_at?: string
  created_at: string
}

/** Tweet dans la corbeille de modération (média déjà résolu en URL gateway). */
export interface DeletedPost {
  id: string
  authorId: string
  content: string
  media: { url: string; type: 'image' | 'video' }[]
  /** Id du modérateur/admin qui a retiré le post. */
  hiddenBy: string
  /** Date du retrait (point de départ de la purge RGPD). */
  hiddenAt: string
  /** Date prévue de purge définitive automatique (RGPD), si rétention configurée. */
  purgeAt?: string
  createdAt: string
}

async function unwrap<T>(res: Response): Promise<T> {
  const body = (await res.json().catch(() => null)) as
    | { data?: T; error?: string }
    | null
  if (!res.ok) {
    throw new ModerationApiError(body?.error ?? `Erreur ${res.status}`, res.status)
  }
  return (body?.data ?? null) as T
}

async function expectOk(res: Response): Promise<void> {
  if (!res.ok) {
    const body = (await res.json().catch(() => null)) as { error?: string } | null
    throw new ModerationApiError(body?.error ?? `Erreur ${res.status}`, res.status)
  }
}

function toDeletedPost(p: ApiHiddenPost): DeletedPost {
  return {
    id: p.id,
    authorId: p.author_id,
    content: p.content ?? '',
    media: (p.media ?? []).map((m) => ({ url: resolveMediaUrl(m.url), type: m.type })),
    hiddenBy: p.hidden_by ?? '',
    hiddenAt: p.hidden_at ?? p.created_at,
    purgeAt: p.purge_at,
    createdAt: p.created_at,
  }
}

/** Filtres de la corbeille de modération (tous cumulables et optionnels). */
export interface DeletedPostsFilter {
  limit?: number
  offset?: number
  /** Filtrer sur l'auteur du tweet retiré. */
  authorId?: string
  /** Bornes ISO (RFC3339) sur la date de retrait (`hidden_at`). */
  since?: string
  until?: string
}

/** Corbeille de modération (tweets masqués), du plus récemment retiré au plus ancien. */
export async function listDeletedPosts(filter: DeletedPostsFilter = {}): Promise<DeletedPost[]> {
  const { limit = 50, offset = 0, authorId, since, until } = filter
  const params = new URLSearchParams({ limit: String(limit), offset: String(offset) })
  if (authorId) params.set('author_id', authorId)
  if (since) params.set('since', since)
  if (until) params.set('until', until)
  const posts = await unwrap<ApiHiddenPost[]>(
    await apiFetch(`/posts/moderation/deleted?${params}`),
  )
  return (posts ?? []).map(toDeletedPost)
}

/** Restaure un tweet masqué (retour dans les fils). Action réversible, sans confirmation. */
export async function restoreDeletedPost(id: string): Promise<void> {
  await expectOk(await apiFetch(`/posts/${id}/restore`, { method: 'POST' }))
}

/** Efface DÉFINITIVEMENT un tweet depuis la corbeille. Irréversible. */
export async function purgeDeletedPost(id: string): Promise<void> {
  await expectOk(await apiFetch(`/posts/${id}/purge`, { method: 'DELETE' }))
}
