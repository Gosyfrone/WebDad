'use client'

import type { ReactNode } from 'react'
import { usePathname } from 'next/navigation'

/**
 * Garde déterministe pour les overlays de routes interceptées (`@modal`).
 *
 * Ne rend l'overlay QUE si l'URL courante correspond encore à sa route. On ne
 * dépend plus de la réinitialisation des slots parallèles de Next (notoirement
 * instable : un slot ne revient pas seul à `default.tsx` en navigation soft).
 * Dès que l'URL change (retour au feed, autre section), `usePathname()` ne
 * correspond plus → l'overlay disparaît immédiatement, même si Next garde le
 * slot monté. C'est ce qui referme proprement, p. ex., la PassphraseGate de la
 * messagerie quand on revient au feed via le logo.
 *
 * `exact` (défaut) : correspondance stricte. Sinon `pathname.startsWith(path)`,
 * pour les routes dynamiques (`/posts/<id>`, `/profil/<username>`).
 */
export function OverlayWhenPath({
  path,
  exact = true,
  children,
}: {
  path: string
  exact?: boolean
  children: ReactNode
}) {
  const pathname = usePathname()
  const matches = exact ? pathname === path : pathname.startsWith(path)
  return matches ? <>{children}</> : null
}
