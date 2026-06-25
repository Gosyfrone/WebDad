/**
 * Configuration globale du frontend.
 *
 * Toute communication passe par l'API Gateway (cf. la doc d'architecture) :
 * aucun service backend n'est appelé directement depuis le client.
 */

/** URL de base de l'API Gateway, vue depuis le navigateur (client). */
export const API_URL =
  process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080'

/**
 * Construit une URL absolue vers l'API Gateway.
 *
 * ⚠️ Côté/hôte différent selon le contexte :
 *   - **client (navigateur)** : la gateway est joignable via `localhost:8080`
 *     (port publié) → on utilise `NEXT_PUBLIC_API_URL`.
 *   - **serveur (route handlers Next, conteneur frontend)** : `localhost`
 *     désigne le conteneur lui-même ; la gateway est sur le réseau Docker
 *     (`http://api-gateway:8080`) → on utilise `API_INTERNAL_URL`.
 *
 * `API_INTERNAL_URL` n'est PAS préfixée `NEXT_PUBLIC_` : elle reste donc
 * invisible côté client (où `process.env.API_INTERNAL_URL` est `undefined`,
 * d'où le repli sur `API_URL`). En local sans Docker, laisser les deux sur
 * `localhost:8080`.
 *
 * @example apiUrl('/auth/login') -> 'http://api-gateway:8080/auth/login' (serveur)
 */
export function apiUrl(path: string): string {
  const base = process.env.API_INTERNAL_URL ?? API_URL
  const normalized = path.startsWith('/') ? path : `/${path}`
  return `${base}${normalized}`
}
