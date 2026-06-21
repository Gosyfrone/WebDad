import { ROUTES } from '@/lib/routes'

/** Section courante de l'en-tête mobile, dérivée du pathname. */
export type MobileHeaderSection =
  | 'feed'
  | 'explorer'
  | 'messages'
  | 'notifications'
  | 'parametres'
  | 'admin'
  | 'moderation'
  | 'signets'
  | 'other'

export interface MobileHeaderNav {
  /** Section déduite du chemin courant. */
  section: MobileHeaderSection
  /** Clé i18n du titre centré, ou `null` pour afficher le logo Breezy. */
  titleKey: string | null
  /** Cloche notifications à droite (header type-feed). */
  showBell: boolean
  /** Flèche retour à gauche au lieu de l'avatar (pages secondaires). */
  showBack: boolean
}

/**
 * Dérive l'agencement de l'en-tête mobile (section, titre, cloche, flèche) à
 * partir du chemin courant. Logique pure extraite de `MobileHeader` pour être
 * testable sans rendu : l'ordre des branches est significatif (le feed est une
 * égalité stricte, les autres des préfixes). `null`/'' → section `other`.
 */
export function resolveMobileHeaderNav(pathname: string | null | undefined): MobileHeaderNav {
  const section: MobileHeaderSection =
    pathname === ROUTES.feed
      ? 'feed'
      : pathname?.startsWith(ROUTES.notifications)
        ? 'notifications'
        : pathname?.startsWith(ROUTES.explorer)
          ? 'explorer'
          : pathname?.startsWith(ROUTES.messages)
            ? 'messages'
            : pathname?.startsWith(ROUTES.parametres)
              ? 'parametres'
              : pathname?.startsWith(ROUTES.moderation)
                ? 'moderation'
                : pathname?.startsWith(ROUTES.admin)
                  ? 'admin'
                  : pathname?.startsWith(ROUTES.bookmarks)
                    ? 'signets'
                    : 'other'

  const titleKey =
    section === 'explorer'
      ? 'nav.explore'
      : section === 'messages'
        ? 'messages.title'
        : section === 'notifications'
          ? 'notifications.title'
          : section === 'parametres'
            ? 'settings.title'
            : section === 'moderation'
              ? 'nav.moderation'
              : section === 'admin'
                ? 'nav.admin'
                : section === 'signets'
                  ? 'bookmarks.title'
                  : null

  const showBell =
    section === 'feed' ||
    section === 'explorer' ||
    section === 'messages' ||
    section === 'admin' ||
    section === 'moderation' ||
    section === 'signets'

  const showBack = section === 'notifications' || section === 'parametres'

  return { section, titleKey, showBell, showBack }
}
