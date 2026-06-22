/**
 * Session de l'utilisateur courant, dérivée du JWT (access token).
 *
 * Source de vérité UNIQUE pour l'identité côté client : le rôle, l'id et l'email
 * vivent dans le JWT (le user-service ne stocke pas le rôle ; cf. CLAUDE.md §5).
 * Le client se contente de **lire** le payload du token (base64url) — la
 * signature reste vérifiée côté serveur. Ce module remplace les décodages JWT
 * jusqu'ici dupliqués (`posts.ts`, `profil-client.ts`).
 *
 * ⚠️ À usage CLIENT (lit `localStorage` via `getAccessToken`).
 */

import { useEffect, useState } from 'react'

import { getAccessToken } from '@/lib/auth-client'
import type { UserRole } from '@/types'

/** Claims bruts du JWT (snake_case, tels qu'émis par auth-service). */
export interface RawClaims {
  user_id?: string
  email?: string
  /** Rôle back : `user` | `moderator` | `admin`. */
  role?: string
  /** Mot de passe temporaire (compte créé par un admin) → changement imposé. */
  must_change_password?: boolean
  /** CGU en vigueur acceptées. Absent/false → modale d'acceptation imposée. */
  terms_accepted?: boolean
}

/** Session normalisée pour le front (rôle mappé vers `UserRole`). */
export interface Session {
  userId: string
  email: string
  role: UserRole
  /** Mot de passe temporaire à changer à la 1re connexion (porté par le JWT). */
  mustChangePassword: boolean
  /** CGU en vigueur acceptées (porté par le JWT). false → modale bloquante. */
  termsAccepted: boolean
}

/**
 * Normalise le rôle back (`admin`) vers le rôle front (`administrator`).
 * Le back utilise `admin`, le front `administrator` (cf. incohérence
 * historique) : ce mapping est le SEUL point de conversion.
 */
export function mapRole(role?: string): UserRole {
  if (role === 'admin' || role === 'administrator') return 'administrator'
  if (role === 'moderator') return 'moderator'
  return 'user'
}

function toBase64(base64Url: string): string {
  const base64 = base64Url.replace(/-/g, '+').replace(/_/g, '/')
  return base64.padEnd(base64.length + ((4 - (base64.length % 4)) % 4), '=')
}

/** Décode les claims bruts du JWT courant (ou `null` si pas de session). */
export function decodeClaims(): RawClaims | null {
  const token = getAccessToken()
  if (!token) return null
  const [, payload] = token.split('.')
  if (!payload) return null
  try {
    return JSON.parse(window.atob(toBase64(payload))) as RawClaims
  } catch {
    return null
  }
}

/** Session normalisée de l'utilisateur courant (ou `null` si déconnecté). */
export function readSession(): Session | null {
  const claims = decodeClaims()
  if (!claims?.user_id) return null
  return {
    userId: claims.user_id,
    email: claims.email ?? '',
    role: mapRole(claims.role),
    mustChangePassword: claims.must_change_password === true,
    termsAccepted: claims.terms_accepted === true,
  }
}

/** Id de l'utilisateur courant (`''` si pas de session). */
export function currentUserId(): string {
  return decodeClaims()?.user_id ?? ''
}

/** Rôle (front) de l'utilisateur courant (`null` si déconnecté). */
export function currentRole(): UserRole | null {
  return readSession()?.role ?? null
}

/** L'utilisateur courant est-il administrateur ? */
export function isAdmin(): boolean {
  return currentRole() === 'administrator'
}

/**
 * Événement diffusé quand la session change (login / refresh / logout) afin que
 * `useSession` se re-synchronise sans rechargement.
 */
export const SESSION_CHANGED_EVENT = 'breezy:session-changed'

/** Notifie les abonnés (`useSession`) d'un changement de session. */
export function notifySessionChanged(): void {
  if (typeof window === 'undefined') return
  window.dispatchEvent(new Event(SESSION_CHANGED_EVENT))
}

/**
 * Hook React : session courante de l'utilisateur.
 *
 * Rend `null` au 1er rendu (serveur + hydratation : le token n'existe pas côté
 * serveur), puis lit le JWT au montage → évite tout mismatch d'hydratation
 * (même compromis que le thème / la locale). Se re-synchronise sur
 * `breezy:session-changed` et sur les changements de `localStorage` (multi-onglet).
 */
export function useSession(): Session | null {
  const [session, setSession] = useState<Session | null>(null)

  useEffect(() => {
    const sync = () => setSession(readSession())
    sync()
    window.addEventListener(SESSION_CHANGED_EVENT, sync)
    window.addEventListener('storage', sync)
    return () => {
      window.removeEventListener(SESSION_CHANGED_EVENT, sync)
      window.removeEventListener('storage', sync)
    }
  }, [])

  return session
}
