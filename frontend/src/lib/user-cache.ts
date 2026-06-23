/**
 * Résolution mémoïsée d'une identité utilisateur (username + décoratif) à partir
 * d'un `userId`, en croisant user-service (identité) et profil-service
 * (display_name / avatar).
 *
 * Plusieurs UI ne reçoivent qu'un `author_id` / `sender_id` / `member_id` et
 * doivent afficher un nom + un avatar : la messagerie (interlocuteur d'un DM,
 * expéditeurs, membres d'un groupe) en a un besoin transverse. Le cache
 * module-level évite de refetch le même utilisateur (anti N+1 au scroll).
 *
 * ⚠️ À usage CLIENT uniquement (`apiFetch` lit le token en localStorage).
 */

import { apiFetch } from '@/lib/auth-client'
import { resolveMediaUrl } from '@/lib/media'
import type { UserCertification } from '@/types'

/** Identité résolue, prête à l'affichage (repli sur le username puis un libellé). */
export interface ResolvedUser {
  id: string
  username: string
  displayName: string
  avatarUrl: string
  certification: UserCertification
}

interface ApiUser {
  id: string
  username: string
}

interface ApiProfil {
  user_id: string
  display_name: string
  avatar_url: string
  certification?: UserCertification
}

const cache = new Map<string, Promise<ResolvedUser>>()

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

/** Résout (et cache) l'identité d'un utilisateur ; repli best-effort si absent. */
export function resolveUser(userId: string): Promise<ResolvedUser> {
  const cached = cache.get(userId)
  if (cached) return cached

  const promise = (async (): Promise<ResolvedUser> => {
    const [user, profil] = await Promise.all([fetchUser(userId), fetchProfil(userId)])
    return {
      id: userId,
      username: user?.username ?? '',
      displayName: profil?.display_name?.trim() || user?.username || 'Utilisateur',
      avatarUrl: resolveMediaUrl(profil?.avatar_url),
      certification: profil?.certification ?? 'none',
    }
  })()

  cache.set(userId, promise)
  return promise
}

/** Résout plusieurs identités en parallèle (dédupliquées par le cache). */
export function resolveUsers(userIds: string[]): Promise<ResolvedUser[]> {
  return Promise.all([...new Set(userIds)].map(resolveUser))
}
