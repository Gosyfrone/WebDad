/**
 * Client d'authentification (navigateur).
 *
 * Modèle : access token court (15 min) en localStorage + refresh token (24 h)
 * en cookie httpOnly (géré par le BFF Next). Les appels API partent
 * directement vers l'API Gateway avec `Authorization: Bearer <access>`.
 *
 * `apiFetch` catche les 401 : il déclenche UN refresh (partagé entre toutes les
 * requêtes concurrentes — single-flight), puis rejoue la requête. Si le refresh
 * échoue (cookie absent/expiré/révoqué), la session est effacée et l'utilisateur
 * redirigé vers /login.
 *
 * ⚠️ À usage CLIENT uniquement (utilise window/localStorage).
 */

import { API_URL } from '@/lib/config'
import { ROUTES } from '@/lib/routes'

const ACCESS_TOKEN_KEY = 'breezy-access-token'

export function getAccessToken(): string | null {
  if (typeof window === 'undefined') return null
  return window.localStorage.getItem(ACCESS_TOKEN_KEY)
}

export function setAccessToken(token: string): void {
  if (typeof window === 'undefined') return
  window.localStorage.setItem(ACCESS_TOKEN_KEY, token)
}

export function clearAccessToken(): void {
  if (typeof window === 'undefined') return
  window.localStorage.removeItem(ACCESS_TOKEN_KEY)
}

/**
 * Single-flight : une seule promesse de refresh à la fois. Les requêtes qui se
 * prennent un 401 en parallèle attendent toutes le MÊME refresh, puis rejouent.
 * Évite la tempête de refresh (N requêtes → N refresh).
 */
let refreshPromise: Promise<string | null> | null = null

function refreshAccessToken(): Promise<string | null> {
  if (!refreshPromise) {
    refreshPromise = (async () => {
      try {
        // same-origin → le cookie httpOnly refresh part automatiquement.
        const res = await fetch('/api/auth/refresh', { method: 'POST' })
        if (!res.ok) return null
        const payload = (await res.json().catch(() => null)) as
          | { accessToken?: string }
          | null
        const token = payload?.accessToken ?? null
        if (token) setAccessToken(token)
        return token
      } catch {
        return null
      } finally {
        refreshPromise = null
      }
    })()
  }
  return refreshPromise
}

function redirectToLogin(): void {
  if (typeof window === 'undefined') return
  if (window.location.pathname !== ROUTES.login) {
    window.location.assign(ROUTES.login)
  }
}

function resolveUrl(path: string): string {
  if (path.startsWith('http')) return path
  return `${API_URL}${path.startsWith('/') ? path : `/${path}`}`
}

/**
 * Fetch authentifié vers l'API Gateway. Ajoute le Bearer, et sur 401 :
 * refresh (partagé) + rejoue UNE fois. Échec du refresh → session effacée +
 * redirection /login.
 */
export async function apiFetch(
  path: string,
  init: RequestInit = {}
): Promise<Response> {
  const url = resolveUrl(path)

  const doFetch = (token: string | null): Promise<Response> => {
    const headers = new Headers(init.headers)
    if (token) headers.set('Authorization', `Bearer ${token}`)
    return fetch(url, { ...init, headers })
  }

  const response = await doFetch(getAccessToken())
  if (response.status !== 401) return response

  // 401 : on tente un refresh partagé, puis on rejoue une seule fois.
  const newToken = await refreshAccessToken()
  if (!newToken) {
    clearAccessToken()
    redirectToLogin()
    return response
  }

  const retried = await doFetch(newToken)
  if (retried.status === 401) {
    // Toujours 401 après un token frais → session morte.
    clearAccessToken()
    redirectToLogin()
  }
  return retried
}

/**
 * Déconnexion : révoque le refresh token côté serveur (BFF → gateway) et
 * efface la session locale, puis redirige vers /login. Best-effort : la
 * session locale est nettoyée même si l'appel réseau échoue.
 */
export async function logout(): Promise<void> {
  try {
    await fetch('/api/auth/logout', { method: 'POST' })
  } catch {
    // best-effort : on nettoie quand même côté client.
  } finally {
    clearAccessToken()
    if (typeof window !== 'undefined') {
      window.location.assign(ROUTES.login)
    }
  }
}
