'use client'

import { useEffect } from 'react'
import { usePathname } from 'next/navigation'

import { rememberLegalOrigin } from '@/lib/legal-origin'

/**
 * Enregistre en continu la dernière page « Breezy » (non légale) visitée, pour
 * que « Retour à Breezy » (pages légales) y revienne. Naviguer entre pages
 * légales ne touche pas l'origine (elles sont ignorées par `rememberLegalOrigin`).
 *
 * Monté une seule fois à la racine (couvre tous les groupes de routes). Ne rend
 * rien.
 */
export function RouteOriginTracker() {
  const pathname = usePathname()

  useEffect(() => {
    if (pathname) rememberLegalOrigin(pathname)
  }, [pathname])

  return null
}
