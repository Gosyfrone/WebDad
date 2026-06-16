'use client'

import { useEffect } from 'react'
import { usePathname } from 'next/navigation'

import { ROUTES } from '@/lib/routes'

/**
 * Le feed est monté en permanence dans le layout `(app)` (cf. `layout.tsx`) et
 * porte le défilement de la fenêtre. Toute section autre que `/feed` est rendue
 * en overlay plein écran (`FeedOverlay`, `fixed`) par-dessus. Sans verrou, le
 * feed continuerait de défiler derrière l'overlay (scroll de fond / chaînage
 * tactile sur mobile) — c'était l'un des bugs UX.
 *
 * Ce garde-fou pose `overflow:hidden` sur `<html>` dès qu'on quitte `/feed` :
 * la fenêtre ne défile plus, donc le feed se fige **à sa position** (overflow
 * hidden ne réinitialise pas `scrollTop`), et on la retrouve intacte au retour.
 * L'overlay gère son propre défilement interne (`overflow-y-auto`).
 */
export function OverlayScrollLock() {
  const pathname = usePathname()
  const overlayOpen = pathname !== ROUTES.feed

  useEffect(() => {
    if (!overlayOpen) return
    const root = document.documentElement
    const previous = root.style.overflow
    root.style.overflow = 'hidden'
    return () => {
      root.style.overflow = previous
    }
  }, [overlayOpen])

  return null
}
