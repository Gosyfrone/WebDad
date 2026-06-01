/**
 * Configuration globale du frontend.
 *
 * Toute communication passe par l'API Gateway (cf. CLAUDE.md §1) :
 * aucun service backend n'est appelé directement depuis le client.
 */

/** URL de base de l'API Gateway. Surchargée via `NEXT_PUBLIC_API_URL`. */
export const API_URL =
  process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080'

/**
 * Construit une URL absolue vers l'API Gateway.
 * @example apiUrl('/auth/login') -> 'http://localhost:8080/auth/login'
 */
export function apiUrl(path: string): string {
  const normalized = path.startsWith('/') ? path : `/${path}`
  return `${API_URL}${normalized}`
}
