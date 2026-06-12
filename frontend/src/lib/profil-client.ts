'use client'

import { apiFetch } from '@/lib/auth-client'
import { resolveMediaUrl, toStoredMedia } from '@/lib/media'
import { decodeClaims, mapRole } from '@/lib/session'
import type { ProfilDetails, ProfilEditableFields } from '@/types'

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
  created_at?: string
  updated_at?: string
  display_name_changed_at?: string
  visibility?: 'public' | 'private'
}

const PROFIL_UPDATED_EVENT = 'breezy:profil-updated'

export async function getMyProfil(): Promise<ProfilDetails> {
  const [user, profil] = await Promise.all([
    fetchApiData<ApiUser>('/users/me'),
    fetchApiData<ApiProfil | null>('/profils/me', { allowNotFound: true }),
  ])

  return mergeProfil(user, profil, true)
}

export async function getPublicProfil(username: string): Promise<ProfilDetails> {
  const user = await fetchApiData<ApiUser>(
    `/users/by-username/${encodeURIComponent(username)}`
  )
  const profil = await fetchApiData<ApiProfil | null>(
    `/profils/${encodeURIComponent(user.id)}`,
    { allowNotFound: true }
  )

  return mergeProfil(user, profil, false)
}

export async function saveMyProfil(
  fields: ProfilEditableFields,
  profileExists: boolean
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

export async function saveMyVisibility(
  visibility: ProfilDetails['visibility']
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

export function subscribeProfilUpdated(
  onUpdate: (profil: ProfilDetails) => void
): () => void {
  function handleUpdate(event: Event) {
    onUpdate((event as CustomEvent<ProfilDetails>).detail)
  }

  window.addEventListener(PROFIL_UPDATED_EVENT, handleUpdate)
  return () => window.removeEventListener(PROFIL_UPDATED_EVENT, handleUpdate)
}

function notifyProfilUpdated(profil: ProfilDetails): void {
  window.dispatchEvent(
    new CustomEvent<ProfilDetails>(PROFIL_UPDATED_EVENT, { detail: profil })
  )
}

function mergeProfil(
  user: ApiUser,
  profil: ApiProfil | null,
  currentUser: boolean
): ProfilDetails {
  const claims = currentUser ? decodeClaims() : null
  const role = currentUser ? mapRole(claims?.role) : 'user'

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
    joinedAt: user.created_at,
    updatedAt: profil?.updated_at ?? user.updated_at ?? user.created_at,
    displayNameChangedAt: profil?.display_name_changed_at ?? '',
    visibility: profil?.visibility === 'private' ? 'private' : 'public',
    followersCount: user.follower_count ?? 0,
    followingCount: user.following_count ?? 0,
    postsCount: 0,
    profileExists: profil !== null,
  }
}

async function fetchApiData<T>(
  path: string,
  init: RequestInit & { allowNotFound?: boolean } = {}
): Promise<T> {
  const { allowNotFound, ...requestInit } = init
  const response = await apiFetch(path, requestInit)

  if (allowNotFound && response.status === 404) {
    return null as T
  }

  const payload = (await response.json().catch(() => null)) as ApiEnvelope<T> | null

  if (!response.ok) {
    throw new Error(payload?.error ?? payload?.message ?? 'Requête profil impossible.')
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
    birth_date: fields.birthDate ? new Date(fields.birthDate).toISOString() : undefined,
  }
}

function jsonHeaders(): HeadersInit {
  return { 'Content-Type': 'application/json' }
}
