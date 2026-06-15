'use client'

import { useEffect, useRef } from 'react'

import {
  STATS_POLL_INTERVAL_MS,
  getPostsStats,
  type PostStats,
} from '@/lib/posts'

/**
 * Rafraîchit périodiquement (façon X, ~quelques secondes) les compteurs des
 * posts actuellement affichés, par lots, sans recharger les posts.
 *
 * - `getIds` est lu À CHAQUE tick (les ids affichés changent au scroll / par
 *   onglet), donc on ne re-souscrit pas quand la liste évolue.
 * - On ne sonde QUE lorsque l'onglet est visible (`document.hidden`) : un onglet
 *   en arrière-plan ne consomme rien ; un retour au premier plan déclenche un
 *   rafraîchissement immédiat (pour rattraper le temps masqué).
 * - Un seul appel en vol à la fois (garde `inFlight`) : un cycle lent ne se
 *   chevauche pas avec le suivant.
 */
export function usePostStatsPolling(
  getIds: () => string[],
  onStats: (stats: Map<string, PostStats>) => void,
  intervalMs: number = STATS_POLL_INTERVAL_MS,
): void {
  const getIdsRef = useRef(getIds)
  const onStatsRef = useRef(onStats)
  getIdsRef.current = getIds
  onStatsRef.current = onStats

  useEffect(() => {
    let cancelled = false
    let inFlight = false

    const tick = async () => {
      if (cancelled || inFlight || document.hidden) return
      const ids = getIdsRef.current()
      if (ids.length === 0) return
      inFlight = true
      try {
        const stats = await getPostsStats(ids)
        if (!cancelled && stats.size > 0) onStatsRef.current(stats)
      } catch {
        // best-effort : on retentera au prochain cycle
      } finally {
        inFlight = false
      }
    }

    const interval = setInterval(tick, intervalMs)
    // Retour au premier plan → rafraîchissement immédiat.
    const onVisible = () => {
      if (!document.hidden) void tick()
    }
    document.addEventListener('visibilitychange', onVisible)

    return () => {
      cancelled = true
      clearInterval(interval)
      document.removeEventListener('visibilitychange', onVisible)
    }
  }, [intervalMs])
}
