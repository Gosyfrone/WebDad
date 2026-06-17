'use client'

import { useEffect, useState } from 'react'

/**
 * Hauteur (px) masquée en bas de l'écran par le clavier logiciel, via l'API
 * `visualViewport`. Retourne `0` quand le clavier est fermé.
 *
 * Pourquoi : sur iOS Safari, l'ouverture du clavier rétrécit le **visual
 * viewport** mais PAS le **layout viewport** — donc `position:fixed`, `100dvh`
 * et `100vh` ignorent le clavier, et un composer collé au bas se retrouve
 * derrière lui (Safari scrolle alors la page pour révéler le champ, et ce
 * décalage reste coincé). Seul `window.visualViewport` reflète le clavier.
 *
 * Calcul : `window.innerHeight` (= layout viewport, inchangé par le clavier sur
 * iOS) − `visualViewport.height` (zone visible) − `visualViewport.offsetTop`
 * (décalage si Safari a scrollé) = hauteur du clavier en coordonnées layout.
 *
 * Sur Android Chrome, le clavier redimensionne déjà le layout viewport
 * (`innerHeight` rétrécit) → le calcul donne ~0 et le hook est un no-op : le
 * `fixed bottom` natif est déjà au-dessus du clavier. iOS uniquement, donc.
 *
 * `enabled` permet de n'attacher les écouteurs que là où c'est utile
 * (messagerie), sans surcoût sur les autres overlays.
 */
export function useKeyboardInset(enabled = true): number {
  const [inset, setInset] = useState(0)

  useEffect(() => {
    if (!enabled || typeof window === 'undefined') return
    const vv = window.visualViewport
    if (!vv) return

    const update = () => {
      const kb = window.innerHeight - vv.height - vv.offsetTop
      // Seuil anti-bruit : les micro-variations (barre d'adresse, arrondis) ne
      // doivent pas être prises pour un clavier. En dessous de ~120 px, on
      // considère qu'aucun clavier n'est ouvert.
      setInset(kb > 120 ? Math.round(kb) : 0)
    }

    update()
    vv.addEventListener('resize', update)
    vv.addEventListener('scroll', update)
    return () => {
      vv.removeEventListener('resize', update)
      vv.removeEventListener('scroll', update)
    }
  }, [enabled])

  return inset
}
