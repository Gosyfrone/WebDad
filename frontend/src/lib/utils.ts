import { type ClassValue, clsx } from 'clsx'
import { twMerge } from 'tailwind-merge'

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

/**
 * Horodatage relatif court façon X (« 12s », « 5min », « 3h », « 2j »), puis
 * date absolue au-delà d'une semaine. Tolère une date invalide (chaîne vide).
 */
export function timeAgo(iso: string): string {
  const date = new Date(iso)
  const ms = date.getTime()
  if (Number.isNaN(ms)) return ''

  const seconds = Math.max(0, Math.floor((Date.now() - ms) / 1000))
  if (seconds < 60) return `${seconds}s`
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return `${minutes}min`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}h`
  const days = Math.floor(hours / 24)
  if (days < 7) return `${days}j`

  return date.toLocaleDateString('fr-FR', { day: 'numeric', month: 'short' })
}

/** Première lettre (majuscule) d'un nom, pour les fallbacks d'avatar. */
export function initialOf(name: string): string {
  return (name.trim().charAt(0) || 'U').toUpperCase()
}
