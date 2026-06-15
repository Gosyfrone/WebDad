'use client'

import { useEffect, useState } from 'react'
import Image from 'next/image'
import Link from 'next/link'
import { usePathname } from 'next/navigation'
import { Bell, LogIn, Mail, Search } from 'lucide-react'

import { cn } from '@/lib/utils'
import { ROUTES } from '@/lib/routes'
import { getSearchPath, isSearchSectionPath } from '@/lib/search-tab'
import { useAuthGate } from '@/components/auth-prompt-provider'
import { useNotifications } from '@/components/notifications-provider'
import { useMessages } from '@/components/messages-provider'
import { useT } from '@/components/language-provider'

interface TabItem {
  href: string
  /** Clé i18n du libellé (cf. lib/i18n.ts, namespace `nav`). */
  labelKey: string
  /** Icône lucide ; `null` = logo Breezy (Accueil). */
  icon: React.ElementType | null
}

const TABS: TabItem[] = [
  { href: ROUTES.feed, labelKey: 'nav.home', icon: null },
  { href: ROUTES.explorer, labelKey: 'nav.search', icon: Search },
  { href: ROUTES.notifications, labelKey: 'nav.notifications', icon: Bell },
  { href: ROUTES.messages, labelKey: 'nav.messages', icon: Mail },
]

/** Onglets du visiteur : seul le fil public est accessible, + une entrée connexion. */
const VISITOR_TABS: TabItem[] = [
  { href: ROUTES.feed, labelKey: 'nav.home', icon: null },
  { href: ROUTES.login, labelKey: 'visitor.login', icon: LogIn },
]

/**
 * Barre de navigation fixée en bas (masquée ≥ lg), façon X.com mobile.
 * « Accueil » utilise le logo Breezy seul (`logo_only.png`).
 */
export function MobileTabBar() {
  const t = useT()
  const pathname = usePathname()
  const { isVisitor } = useAuthGate()

  // Loupe (Recherche) : `href` dynamique (mémoire de navigation par onglet).
  // Depuis un autre onglet → dernier chemin de la section recherche (profil,
  // résultats…) ; déjà dans la section → recherche neuve (`/explorer`). On utilise
  // l'href natif de l'ancre (fiable sur iOS, contrairement à un `preventDefault`).
  // Défaut `/explorer` au 1er rendu (SSR) puis recalcul client à chaque navigation.
  const [searchHref, setSearchHref] = useState<string>(ROUTES.explorer)
  useEffect(() => {
    setSearchHref(isSearchSectionPath(pathname) ? ROUTES.explorer : getSearchPath())
  }, [pathname])
  const { unreadCount } = useNotifications()
  const { unreadCount: msgUnread } = useMessages()
  const tabs = isVisitor ? VISITOR_TABS : TABS

  return (
    <nav
      aria-label={t('nav.main_aria')}
      className="panel fixed inset-x-0 bottom-0 z-50 flex h-14 border-t shadow-[0_-18px_44px_rgba(91,108,255,0.12)] backdrop-blur-2xl lg:hidden"
    >
      {tabs.map((tab) => {
        const active = pathname === tab.href
        const Icon = tab.icon
        const badgeCount =
          tab.href === ROUTES.notifications
            ? unreadCount
            : tab.href === ROUTES.messages
              ? msgUnread
              : 0
        const showBadge = badgeCount > 0
        const badgeAria =
          tab.href === ROUTES.messages
            ? t('messages.badge_aria', { count: badgeCount })
            : t('notifications.unread_aria', { count: badgeCount })

        return (
          <Link
            key={tab.href}
            href={tab.href === ROUTES.explorer ? searchHref : tab.href}
            scroll={false}
            aria-label={t(tab.labelKey)}
            aria-current={active ? 'page' : undefined}
            className="flex flex-1 items-center justify-center transition-colors hover:bg-accent"
          >
            {Icon ? (
              <span className="relative">
                <Icon
                  className={cn('h-6 w-6', active ? 'text-primary' : 'text-muted-foreground')}
                  aria-hidden
                />
                {showBadge && (
                  <span
                    aria-label={badgeAria}
                    className="absolute -right-2 -top-1.5 flex h-[16px] min-w-[16px] items-center justify-center rounded-full bg-gradient-to-r from-[var(--brand-from)] via-[var(--brand-via)] to-[var(--brand-to)] px-1 text-[10px] font-bold leading-none text-white shadow"
                  >
                    {badgeCount > 99 ? '99+' : badgeCount}
                  </span>
                )}
              </span>
            ) : (
              <Image
                src="/logo_only.png"
                alt=""
                width={512}
                height={512}
                className={cn(
                  'h-7 w-7 object-contain transition-opacity',
                  active ? 'opacity-100' : 'opacity-50',
                )}
              />
            )}
          </Link>
        )
      })}
    </nav>
  )
}
