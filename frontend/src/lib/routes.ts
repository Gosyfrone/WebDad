/**
 * Constantes de routes du frontend.
 *
 * Source de vérité unique pour les chemins de navigation : on évite ainsi
 * les chaînes en dur dispersées dans les composants (liens, redirections).
 */

import type { UserRole } from '@/types'

export const ROUTES = {
  home: '/',
  // Espace public (route group (auth))
  login: '/login',
  register: '/register',
  checkEmail: '/check-email',
  verifyEmail: '/verify-email',
  forgotPassword: '/forgot-password',
  resetPassword: '/reset-password',
  // Pages légales (route group (legal), publiques)
  mentionsLegales: '/mentions-legales',
  cgu: '/cgu',
  confidentialite: '/confidentialite',
  // Espace authentifié (route group (app))
  feed: '/feed',
  explorer: '/explorer',
  notifications: '/notifications',
  messages: '/messages',
  bookmarks: '/signets',
  profil: '/profil',
  parametres: '/parametres',
  moderation: '/moderation',
  admin: '/admin',
} as const

export type Route = (typeof ROUTES)[keyof typeof ROUTES]

/** Lien vers le profil public d'un utilisateur (`/profil/<username>`). */
export function profilHref(username: string): string {
  return `${ROUTES.profil}/${encodeURIComponent(username)}`
}

/** Lien vers le détail d'un post (`/posts/<id>`). */
export function postHref(id: string): string {
  return `/posts/${encodeURIComponent(id)}`
}

/** Lien vers les résultats d'un hashtag. */
export function hashtagHref(tag: string, tab: 'top' | 'recent' | 'media' = 'top'): string {
  const clean = tag.trim().replace(/^#/, '')
  return `${ROUTES.feed}?hashtag=${encodeURIComponent(clean)}&tab=${tab}`
}

/** Destination d'une recherche globale : #tag => feed hashtag, sinon Explorer. */
export function searchHref(query: string): string {
  const q = query.trim()
  if (!q) return ROUTES.explorer
  if (q.startsWith('#')) return hashtagHref(q, 'top')
  return `${ROUTES.explorer}?q=${encodeURIComponent(q)}`
}

/** Un lien de navigation affiché dans la barre de navigation. */
export interface NavItem {
  /** Clé i18n du libellé (cf. lib/i18n.ts, namespace `nav`). */
  labelKey: string
  href: Route
  /** Rôles autorisés à voir ce lien. `undefined` = visible par tous. */
  roles?: UserRole[]
}

/**
 * Liens de l'espace authentifié, dans l'ordre d'affichage.
 * La visibilité par rôle reflète les 3 rôles du projet (cf. CLAUDE.md §1).
 */
export const APP_NAV: NavItem[] = [
  { labelKey: 'nav.feed', href: ROUTES.feed },
  { labelKey: 'nav.bookmarks', href: ROUTES.bookmarks },
  { labelKey: 'nav.profil', href: ROUTES.profil },
  { labelKey: 'nav.moderation', href: ROUTES.moderation, roles: ['moderator', 'administrator'] },
  { labelKey: 'nav.admin', href: ROUTES.admin, roles: ['administrator'] },
]

/** Filtre les liens de navigation selon le rôle de l'utilisateur courant. */
export function navItemsForRole(role: UserRole | null): NavItem[] {
  return APP_NAV.filter((item) => !item.roles || (role !== null && item.roles.includes(role)))
}
