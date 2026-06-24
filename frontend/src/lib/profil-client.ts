'use client'

import { apiFetch } from '@/lib/auth-client'
import { resolveMediaUrl, toStoredMedia } from '@/lib/media'
import { decodeClaims, mapRole } from '@/lib/session'
import type { ProfilDetails, ProfilEditableFields, UserCertification } from '@/types'

type ApiEnvelope<T> = { data?: T; error?: string; message?: string }

type ApiUser = {
  id: string
  username: string
  is_active?: boolean
  created_at: string
  updated_at?: string
  follower_count?: number
  following_count?: number
}

type ApiProfil = {
  user_id: string
  display_name?: string
  bio?: string
  avatar_url?: string
  banner_url?: string
  website?: string
  location?: string
  birth_date?: string
  gender?: 'male' | 'female'
  nationality?: string
  created_at?: string
  updated_at?: string
  display_name_changed_at?: string
  visibility?: 'public' | 'private'
  likes_visibility?: 'public' | 'private'
  activity_visibility?: 'public' | 'private'
  certification?: UserCertification
  last_login_at?: string
  is_online?: boolean
  nsfw_enabled?: boolean
  /** Didacticiel déjà vu (terminé/ignoré). Absent ⇒ false (vieux document). */
  tutorial_done?: boolean
  /** Calculés serveur, présents uniquement sur la vue privée /profils/me. */
  is_adult?: boolean
  nsfw_visible?: boolean
}

const PROFIL_UPDATED_EVENT = 'breezy:profil-updated'

export async function getMyProfil(): Promise<ProfilDetails> {
  const [user, profil] = await Promise.all([
    fetchApiData<ApiUser>('/users/me'),
    fetchApiData<ApiProfil | null>('/profils/me', { allowNotFound: true }),
  ])

  return mergeProfil(user, profil, true)
}

export async function getPublicProfil(
  username: string,
): Promise<ProfilDetails> {
  const user = await fetchApiData<ApiUser>(
    `/users/by-username/${encodeURIComponent(username)}`,
  )
  const profil = await fetchApiData<ApiProfil | null>(
    `/profils/${encodeURIComponent(user.id)}`,
    { allowNotFound: true },
  )

  return mergeProfil(user, profil, false)
}

export async function saveMyProfil(
  fields: ProfilEditableFields,
  profileExists: boolean,
): Promise<ProfilDetails> {
  if (!profileExists) {
    await fetchApiData<ApiProfil>('/profils', {
      method: 'POST',
      headers: jsonHeaders(),
      body: JSON.stringify({ display_name: fields.displayName }),
    })
  }

  await fetchApiData<ApiProfil>('/profils/me', {
    method: 'PATCH',
    headers: jsonHeaders(),
    body: JSON.stringify(toUpdatePayload(fields)),
  })

  const updated = await getMyProfil()
  notifyProfilUpdated(updated)
  return updated
}

export async function saveMyLikesVisibility(
  likesVisibility: ProfilDetails['likesVisibility'],
): Promise<ProfilDetails> {
  await fetchApiData<ApiProfil>('/profils/me', {
    method: 'PATCH',
    headers: jsonHeaders(),
    body: JSON.stringify({ likes_visibility: likesVisibility }),
  })

  const updated = await getMyProfil()
  notifyProfilUpdated(updated)
  return updated
}

export async function saveMyVisibility(
  visibility: ProfilDetails['visibility'],
): Promise<ProfilDetails> {
  await fetchApiData<ApiProfil>('/profils/me', {
    method: 'PATCH',
    headers: jsonHeaders(),
    body: JSON.stringify({ visibility }),
  })

  const updated = await getMyProfil()
  notifyProfilUpdated(updated)
  return updated
}

export async function saveMyActivityVisibility(
  activityVisibility: ProfilDetails['activityVisibility'],
): Promise<ProfilDetails> {
  await fetchApiData<ApiProfil>('/profils/me', {
    method: 'PATCH',
    headers: jsonHeaders(),
    body: JSON.stringify({ activity_visibility: activityVisibility }),
  })

  if (activityVisibility === 'public') {
    await markMyActivityOnline()
  }

  const updated = await getMyProfil()
  notifyProfilUpdated(updated)
  return updated
}

export async function saveMyNsfw(
  nsfwEnabled: boolean,
): Promise<ProfilDetails> {
  await fetchApiData<ApiProfil>('/profils/me', {
    method: 'PATCH',
    headers: jsonHeaders(),
    body: JSON.stringify({ nsfw_enabled: nsfwEnabled }),
  })

  const updated = await getMyProfil()
  notifyProfilUpdated(updated)
  return updated
}

/**
 * Marque le didacticiel comme vu (terminé ou ignoré) côté serveur. Persiste via
 * `PATCH /profils/me` (champ `tutorial_done`) pour que le tour ne soit plus
 * proposé automatiquement, sur tous les appareils. Best-effort : l'appelant
 * (TutorialProvider) ne bloque pas l'UX dessus.
 */
export async function saveMyTutorialDone(
  tutorialDone: boolean,
): Promise<ProfilDetails> {
  await fetchApiData<ApiProfil>('/profils/me', {
    method: 'PATCH',
    headers: jsonHeaders(),
    body: JSON.stringify({ tutorial_done: tutorialDone }),
  })

  const updated = await getMyProfil()
  notifyProfilUpdated(updated)
  return updated
}

export function subscribeProfilUpdated(
  onUpdate: (profil: ProfilDetails) => void,
): () => void {
  function handleUpdate(event: Event) {
    onUpdate((event as CustomEvent<ProfilDetails>).detail)
  }

  window.addEventListener(PROFIL_UPDATED_EVENT, handleUpdate)
  return () => window.removeEventListener(PROFIL_UPDATED_EVENT, handleUpdate)
}

function notifyProfilUpdated(profil: ProfilDetails): void {
  window.dispatchEvent(
    new CustomEvent<ProfilDetails>(PROFIL_UPDATED_EVENT, { detail: profil }),
  )
}

async function markMyActivityOnline(): Promise<void> {
  try {
    await apiFetch('/profils/me/activity', { method: 'PATCH' })
  } catch {
    // Best-effort : la préférence reste sauvegardée, le polling rattrapera.
  }
}

function mergeProfil(
  user: ApiUser,
  profil: ApiProfil | null,
  currentUser: boolean,
): ProfilDetails {
  const claims = currentUser ? decodeClaims() : null
  const role = currentUser ? mapRole(claims?.role) : 'user'
  const visibility = profil?.visibility === 'private' ? 'private' : 'public'
  const activityVisibility =
    profil?.activity_visibility === 'private' ? 'private' : 'public'

  return {
    userId: user.id,
    displayName: profil?.display_name?.trim() || user.username,
    username: user.username,
    role,
    isActive: user.is_active ?? true,
    bio: profil?.bio ?? '',
    avatarUrl: resolveMediaUrl(profil?.avatar_url),
    bannerUrl: resolveMediaUrl(profil?.banner_url),
    website: profil?.website ?? '',
    location: profil?.location ?? '',
    birthDate: profil?.birth_date ?? '',
    gender: profil?.gender ?? '',
    nationality: profil?.nationality ?? '',
    joinedAt: user.created_at,
    updatedAt: profil?.updated_at ?? user.updated_at ?? user.created_at,
    displayNameChangedAt: profil?.display_name_changed_at ?? '',
    visibility,
    likesVisibility:
      profil?.likes_visibility === 'private' ? 'private' : 'public',
    activityVisibility,
    certification: profil?.certification ?? 'none',
    lastLoginAt: profil?.last_login_at ?? '',
    isOnline: Boolean(profil?.is_online),
    // Défaut `true` : pref NSFW ON / adulte par défaut (cf. backend). Sur un profil
    // PUBLIC (currentUser=false) le serveur n'envoie pas is_adult/nsfw_visible —
    // ces champs ne sont consommés que pour le viewer courant (getMyProfil).
    nsfwEnabled: profil?.nsfw_enabled ?? true,
    isAdult: profil?.is_adult ?? true,
    nsfwVisible: profil?.nsfw_visible ?? true,
    // Défaut `false` : un profil sans le champ (vieux document / vue publique) est
    // considéré « didacticiel non vu » → le tour pourra être proposé une fois.
    tutorialDone: profil?.tutorial_done ?? false,
    followersCount: user.follower_count ?? 0,
    followingCount: user.following_count ?? 0,
    postsCount: 0,
    profileExists: profil !== null,
  }
}

async function fetchApiData<T>(
  path: string,
  init: RequestInit & { allowNotFound?: boolean } = {},
): Promise<T> {
  const { allowNotFound, ...requestInit } = init
  const response = await apiFetch(path, requestInit)

  if (allowNotFound && response.status === 404) {
    return null as T
  }

  const payload = (await response
    .json()
    .catch(() => null)) as ApiEnvelope<T> | null

  if (!response.ok) {
    throw new Error(
      payload?.error ?? payload?.message ?? 'Requête profil impossible.',
    )
  }

  if (!payload?.data) {
    throw new Error('Réponse API profil invalide.')
  }

  return payload.data
}

function toUpdatePayload(fields: ProfilEditableFields) {
  return {
    display_name: fields.displayName,
    bio: fields.bio,
    avatar_url: toStoredMedia(fields.avatarUrl),
    banner_url: toStoredMedia(fields.bannerUrl),
    website: fields.website,
    location: fields.location,
    gender: fields.gender || undefined,
    nationality: fields.nationality || undefined,
    birth_date: fields.birthDate
      ? new Date(fields.birthDate).toISOString()
      : undefined,
  }
}

function jsonHeaders(): HeadersInit {
  return { 'Content-Type': 'application/json' }
}
