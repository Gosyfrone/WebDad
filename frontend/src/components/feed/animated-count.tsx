'use client'

import { useEffect, useRef, useState } from 'react'

import { cn } from '@/lib/utils'

/** Formate un compteur façon X : 1.2K, 3.4M. */
export function formatCount(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`
  return String(n)
}

/**
 * Compteur d'action (likes/commentaires/reposts) avec une petite animation de
 * « roll » vertical à chaque changement de valeur — façon X quand un compteur
 * bouge via le rafraîchissement dynamique (polling) ou une action locale.
 *
 * Rien n'est rendu tant que la valeur est ≤ 0 (le compteur reste masqué tant
 * qu'il n'y a aucun like/commentaire/repost), mais le composant reste monté pour
 * que le passage 0 → 1 s'anime aussi. Le `key` force le rejeu de l'animation
 * même sur deux hausses consécutives.
 */
export function AnimatedCount({ value, className }: { value: number; className?: string }) {
  const [display, setDisplay] = useState(value)
  const [dir, setDir] = useState<'up' | 'down' | null>(null)
  const prev = useRef(value)

  useEffect(() => {
    if (value === prev.current) return
    setDir(value > prev.current ? 'up' : 'down')
    setDisplay(value)
    prev.current = value
    const id = setTimeout(() => setDir(null), 280)
    return () => clearTimeout(id)
  }, [value])

  if (display <= 0) return null

  return (
    <span className={cn('inline-flex overflow-hidden leading-none', className)}>
      <span
        key={display}
        className={cn(
          'inline-block',
          dir === 'up' && 'animate-count-up',
          dir === 'down' && 'animate-count-down',
        )}
      >
        {formatCount(display)}
      </span>
    </span>
  )
}
