'use client'

import { apiFetch } from '@/lib/auth-client'
import type { UserCertification, UserRole } from '@/types'

export interface IdentityUpdate {
  userId: string
  certification?: UserCertification
  role?: UserRole
}

const IDENTITY_UPDATED_EVENT = 'breezy:identity-updated'
const roleCache = new Map<string, Promise<UserRole>>()

interface ApiPublicRole {
  id: string
  role: string
}

export function mapPublicRole(role?: string): UserRole {
  if (role === 'admin' || role === 'administrator') return 'administrator'
  if (role === 'moderator') return 'moderator'
  return 'user'
}

export function getUserRole(userId: string): Promise<UserRole> {
  if (!userId) return Promise.resolve('user')
  const cached = roleCache.get(userId)
  if (cached) return cached

  const promise = (async () => {
    const roles = await getUserRoles([userId])
    return roles.get(userId) ?? 'user'
  })()
  roleCache.set(userId, promise)
  return promise
}

export async function getUserRoles(userIds: string[]): Promise<Map<string, UserRole>> {
  const ids = [...new Set(userIds.map((id) => id.trim()).filter(Boolean))]
  const out = new Map<string, UserRole>()
  if (ids.length === 0) return out

  const res = await apiFetch(`/auth/users/roles?ids=${encodeURIComponent(ids.join(','))}`)
  if (!res.ok) {
    ids.forEach((id) => out.set(id, 'user'))
    return out
  }
  const body = (await res.json().catch(() => null)) as { data?: ApiPublicRole[] } | null
  for (const row of body?.data ?? []) {
    out.set(row.id, mapPublicRole(row.role))
  }
  ids.forEach((id) => {
    const role = out.get(id) ?? 'user'
    out.set(id, role)
    roleCache.set(id, Promise.resolve(role))
  })
  return out
}

export function dispatchIdentityUpdate(update: IdentityUpdate): void {
  if (update.role) roleCache.set(update.userId, Promise.resolve(update.role))
  if (typeof window === 'undefined') return
  window.dispatchEvent(new CustomEvent<IdentityUpdate>(IDENTITY_UPDATED_EVENT, { detail: update }))
}

export function subscribeIdentityUpdate(onUpdate: (update: IdentityUpdate) => void): () => void {
  if (typeof window === 'undefined') return () => {}
  const handler = (event: Event) => {
    onUpdate((event as CustomEvent<IdentityUpdate>).detail)
  }
  window.addEventListener(IDENTITY_UPDATED_EVENT, handler)
  return () => window.removeEventListener(IDENTITY_UPDATED_EVENT, handler)
}
