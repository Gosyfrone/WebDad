/**
 * Helpers de partage (Web Share API / presse-papiers / URL absolue). Fonctions
 * PURES et sans dépendance UI → testables (vitest) et réutilisables par le
 * composant `ShareDialog` (post + profil).
 *
 * Le partage en message privé ne passe PAS par ici : il réutilise l'envoi de
 * message E2EE existant (`sendMessage`) côté composant.
 */

/**
 * Construit une URL absolue vers l'app (origine du navigateur + chemin relatif).
 * Côté serveur (`window` indéfini), renvoie le chemin tel quel.
 */
export function absoluteUrl(path: string): string {
  if (typeof window === 'undefined') return path
  return new URL(path, window.location.origin).toString()
}

/** L'API Web Share native est-elle disponible (mobile + certains desktops) ? */
export function canNativeShare(): boolean {
  return typeof navigator !== 'undefined' && typeof navigator.share === 'function'
}

/**
 * Ouvre la feuille de partage native. Renvoie `true` si le partage a abouti,
 * `false` si l'API est absente ou si l'utilisateur a annulé (`AbortError`).
 */
export async function nativeShare(data: { title?: string; text?: string; url: string }): Promise<boolean> {
  if (!canNativeShare()) return false
  try {
    await navigator.share(data)
    return true
  } catch {
    // Annulation utilisateur ou erreur → on laisse l'appelant gérer le repli.
    return false
  }
}

/**
 * Référence vers une ressource Breezy détectée dans un message (post / profil).
 * `id` = id du post OU `@handle` (username) du profil.
 */
export interface SharedRef {
  kind: 'post' | 'profile'
  id: string
}

/**
 * Extrait du texte la PREMIÈRE URL pointant vers un post ou un profil Breezy.
 * On reconnaît au chemin (`/posts/<id>` ou `/profil/<handle>`), quel que soit
 * l'hôte (un lien de prod prévisualise aussi en dev). Fonction PURE.
 */
export function extractSharedRef(text: string): SharedRef | null {
  const re = /(https?:\/\/[^\s]+)/g
  let match: RegExpExecArray | null
  while ((match = re.exec(text)) !== null) {
    const raw = match[0].replace(/[).,;!?]+$/, '')
    try {
      const u = new URL(raw)
      const post = u.pathname.match(/^\/posts\/([^/]+)\/?$/)
      if (post) return { kind: 'post', id: decodeURIComponent(post[1]) }
      const profile = u.pathname.match(/^\/profil\/([^/]+)\/?$/)
      if (profile) return { kind: 'profile', id: decodeURIComponent(profile[1]) }
    } catch {
      // URL malformée → on ignore et on continue.
    }
  }
  return null
}

const RECENT_KEY = 'breezy.share.recent'
const RECENT_MAX = 10

/** Minimum nécessaire pour afficher/relancer un partage récent. */
export interface RecentShareTarget {
  id: string
  username: string
  displayName: string
  avatarUrl: string
}

/** Destinataires de partage récents (plus récent d'abord), lus du localStorage. */
export function getRecentShareTargets(): RecentShareTarget[] {
  if (typeof window === 'undefined') return []
  try {
    const raw = window.localStorage.getItem(RECENT_KEY)
    const parsed = raw ? (JSON.parse(raw) as RecentShareTarget[]) : []
    return Array.isArray(parsed) ? parsed.filter((u) => u && u.id) : []
  } catch {
    return []
  }
}

/** Enregistre un destinataire en tête de liste (dédupliqué, plafonné). */
export function recordRecentShareTarget(user: RecentShareTarget): void {
  if (typeof window === 'undefined') return
  const entry: RecentShareTarget = {
    id: user.id,
    username: user.username,
    displayName: user.displayName,
    avatarUrl: user.avatarUrl,
  }
  const next = [entry, ...getRecentShareTargets().filter((u) => u.id !== entry.id)].slice(0, RECENT_MAX)
  try {
    window.localStorage.setItem(RECENT_KEY, JSON.stringify(next))
  } catch {
    // quota / mode privé → on ignore (le partage fonctionne sans historique).
  }
}

/** Copie un texte dans le presse-papiers. Renvoie `true` en cas de succès. */
export async function copyLink(text: string): Promise<boolean> {
  try {
    if (typeof navigator !== 'undefined' && navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(text)
      return true
    }
  } catch {
    // repli ci-dessous
  }
  return false
}
