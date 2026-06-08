/**
 * Mémorisation de la dernière page « Breezy » (non légale) visitée, pour que le
 * bouton « Retour à Breezy » des pages légales y revienne — plutôt qu'un simple
 * retour arrière navigateur (qui, entre deux pages légales, ramènerait juste à
 * la page légale précédente).
 *
 * Persisté en `sessionStorage` (par onglet) : un `RouteOriginTracker` monté à la
 * racine enregistre chaque pathname non légal ; `LegalBackButton` le relit.
 */

import { ROUTES } from '@/lib/routes'

export const LEGAL_ORIGIN_KEY = 'breezy-legal-origin'

const LEGAL_PATHS = [ROUTES.mentionsLegales, ROUTES.cgu, ROUTES.confidentialite]

/** La route donnée est-elle une page légale ? */
export function isLegalPath(pathname: string): boolean {
  return LEGAL_PATHS.some((p) => pathname === p || pathname.startsWith(`${p}/`))
}

/** Enregistre une page d'origine (ignore les pages légales et le vide). */
export function rememberLegalOrigin(pathname: string): void {
  if (typeof window === 'undefined' || !pathname || isLegalPath(pathname)) return
  try {
    window.sessionStorage.setItem(LEGAL_ORIGIN_KEY, pathname)
  } catch {
    /* sessionStorage indisponible (mode privé strict) : on ignore */
  }
}

/** Dernière page Breezy mémorisée (null si aucune / légale / indisponible). */
export function getLegalOrigin(): string | null {
  if (typeof window === 'undefined') return null
  try {
    const origin = window.sessionStorage.getItem(LEGAL_ORIGIN_KEY)
    return origin && !isLegalPath(origin) ? origin : null
  } catch {
    return null
  }
}
