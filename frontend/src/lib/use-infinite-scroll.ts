'use client'

import { useEffect, useRef } from 'react'

interface InfiniteScrollOptions {
  /** Reste-t-il des pages à charger ? */
  hasMore: boolean
  /** Un chargement est-il déjà en cours ? (évite les appels concurrents) */
  loading: boolean
  /** Marge de pré-chargement avant que la sentinelle n'entre dans le viewport. */
  rootMargin?: string
}

/**
 * Défilement infini par sentinelle (`IntersectionObserver`). Renvoie une ref à
 * poser sur un élément placé en fin de liste : dès qu'il approche du viewport,
 * `onLoadMore` est appelé.
 *
 * L'observateur est recréé quand `hasMore`/`loading` changent : si la sentinelle
 * est toujours visible après un chargement, la page suivante part automatiquement
 * (scroll continu). Désactivé tant que `loading` (un seul appel par page).
 */
export function useInfiniteScroll<T extends HTMLElement = HTMLDivElement>(
  onLoadMore: () => void,
  { hasMore, loading, rootMargin = '300px' }: InfiniteScrollOptions,
) {
  const sentinelRef = useRef<T | null>(null)
  const callbackRef = useRef(onLoadMore)
  callbackRef.current = onLoadMore

  useEffect(() => {
    const el = sentinelRef.current
    if (!el || !hasMore || loading) return

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0]?.isIntersecting) callbackRef.current()
      },
      { rootMargin },
    )
    observer.observe(el)
    return () => observer.disconnect()
  }, [hasMore, loading, rootMargin])

  return sentinelRef
}
