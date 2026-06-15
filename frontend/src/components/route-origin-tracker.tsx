'use client'

import { useEffect } from 'react'
import { usePathname } from 'next/navigation'

import { rememberLegalOrigin } from '@/lib/legal-origin'
import { isSearchSectionPath, rememberSearchPath } from '@/lib/search-tab'

/**
 * Enregistre en continu la dernière page « Breezy » (non légale) visitée, pour
 * que « Retour à Breezy » (pages légales) y revienne. Naviguer entre pages
 * légales ne touche pas l'origine (elles sont ignorées par `rememberLegalOrigin`).
 *
 * Mémorise aussi le dernier chemin de la « section recherche » (Explorer +
 * profils) pour la mémoire de navigation de la loupe (cf. lib/search-tab). On lit
 * `window.location.search` pour conserver la requête `?q=` (les changements de
 * query SANS changement de pathname sont gérés en plus dans l'Explorer lui-même).
 *
 * Monté une seule fois à la racine (couvre tous les groupes de routes). Ne rend
 * rien.
 */
export function RouteOriginTracker() {
  const pathname = usePathname()

  useEffect(() => {
    if (!pathname) return
    rememberLegalOrigin(pathname)
    if (isSearchSectionPath(pathname)) {
      rememberSearchPath(pathname + window.location.search)
    }
  }, [pathname])

  return null
}
