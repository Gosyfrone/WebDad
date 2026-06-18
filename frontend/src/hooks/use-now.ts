'use client'

import { useEffect, useState } from 'react'

/**
 * Force un re-render périodique (par défaut chaque seconde) afin de rafraîchir
 * les durées relatives affichées par `timeAgo` (« 1s, 2s, 3s… ») SANS recharger
 * les données. Renvoie l'instant courant (ms) — on peut l'ignorer si on ne s'en
 * sert que pour déclencher le rendu.
 */
export function useNow(intervalMs = 1000): number {
  const [now, setNow] = useState(() => Date.now())
  useEffect(() => {
    const id = setInterval(() => setNow(Date.now()), intervalMs)
    return () => clearInterval(id)
  }, [intervalMs])
  return now
}
