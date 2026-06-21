import type { PostPoll } from '@/lib/posts'

/**
 * Helpers d'affichage de sondage extraits de `PostCard` (logique pure, testable
 * sans rendu). Un sondage est clos s'il a une date de clôture explicite ou si sa
 * date de fin est dépassée.
 */
export function isPollClosedAt(poll: PostPoll, now: number): boolean {
  return Boolean(poll.closedAt) || Date.parse(poll.endsAt) <= now
}

/** Variante au temps courant (`Date.now()`). */
export function isPollClosed(poll: PostPoll): boolean {
  return isPollClosedAt(poll, Date.now())
}

/**
 * Bascule vers la vue « résultats » (barres type chart, façon Twitter) : on
 * n'affiche les résultats qu'une fois qu'on a le droit de les voir ET qu'on a
 * voté ou que le sondage est clos. Sinon on garde la vue de vote cliquable.
 */
export function pollResultsView(poll: PostPoll, closed: boolean): boolean {
  return poll.canViewResults && (Boolean(poll.votedChoiceId) || closed)
}

/**
 * Temps restant avant clôture, formaté de façon compacte (j/h/min/s). Renvoie
 * la plus grande unité significative + la suivante ; jamais de valeur négative.
 */
export function formatPollRemaining(endsAt: string, now: number): string {
  const remainingSeconds = Math.max(0, Math.ceil((Date.parse(endsAt) - now) / 1000))
  const days = Math.floor(remainingSeconds / 86400)
  const hours = Math.floor((remainingSeconds % 86400) / 3600)
  const minutes = Math.floor((remainingSeconds % 3600) / 60)
  const seconds = remainingSeconds % 60
  if (days > 0) return `${days} j ${hours} h`
  if (hours > 0) return `${hours} h ${minutes} min`
  if (minutes > 0) return `${minutes} min ${seconds} s`
  return `${seconds} s`
}
