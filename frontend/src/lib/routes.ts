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
  // Espace authentifié (route group (app))
  feed: '/feed',
  profil: '/profil',
  moderation: '/moderation',
  admin: '/admin',
} as const

export type Route = (typeof ROUTES)[keyof typeof ROUTES]

/** Un lien de navigation affiché dans la barre de navigation. */
export interface NavItem {
  label: string
  href: Route
  /** Rôles autorisés à voir ce lien. `undefined` = visible par tous. */
  roles?: UserRole[]
}

/**
 * Liens de l'espace authentifié, dans l'ordre d'affichage.
 * La visibilité par rôle reflète les 3 rôles du projet (cf. CLAUDE.md §1).
 */
export const APP_NAV: NavItem[] = [
  { label: 'Fil', href: ROUTES.feed },
  { label: 'Profil', href: ROUTES.profil },
  { label: 'Modération', href: ROUTES.moderation, roles: ['moderator', 'administrator'] },
  { label: 'Administration', href: ROUTES.admin, roles: ['administrator'] },
]

/** Filtre les liens de navigation selon le rôle de l'utilisateur courant. */
export function navItemsForRole(role: UserRole | null): NavItem[] {
  return APP_NAV.filter((item) => !item.roles || (role !== null && item.roles.includes(role)))
}
