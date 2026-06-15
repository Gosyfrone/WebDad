import { type ClassValue, clsx } from 'clsx'
import { twMerge } from 'tailwind-merge'

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

/**
 * Horodatage relatif court façon X (« 12s », « 5min », « 3h », « 2j »), puis
 * date absolue au-delà d'une semaine. Tolère une date invalide (chaîne vide).
 *
 * Les suffixes minute/jour et le format de date absolue suivent la `locale`
 * (défaut `fr`) : en anglais → « 5m », « 2d » + date `en-US`.
 */
export function timeAgo(iso: string, locale: string = 'fr'): string {
  const date = new Date(iso)
  const ms = date.getTime()
  if (Number.isNaN(ms)) return ''

  const en = locale === 'en'
  const seconds = Math.max(0, Math.floor((Date.now() - ms) / 1000))
  if (seconds < 60) return `${seconds}s`
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return `${minutes}${en ? 'm' : 'min'}`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}h`
  const days = Math.floor(hours / 24)
  if (days < 7) return `${days}${en ? 'd' : 'j'}`

  return date.toLocaleDateString(en ? 'en-US' : 'fr-FR', { day: 'numeric', month: 'short' })
}

/** Première vraie lettre Unicode du nom, puis du username, pour les avatars. */
export function initialOf(displayName?: string | null, username?: string | null): string {
  for (const value of [displayName, username]) {
    const letter = value?.match(/\p{L}/u)?.[0]
    if (letter) return letter.toLocaleUpperCase()
  }
  return '?'
}
