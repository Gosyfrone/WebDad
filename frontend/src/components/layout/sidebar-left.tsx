'use client'

import { useEffect, useState } from 'react'
import Image from 'next/image'
import Link from 'next/link'
import { usePathname } from 'next/navigation'
import {
  Bell,
  Bookmark,
  Home,
  LogOut,
  Mail,
  MoreHorizontal,
  Search,
  Settings,
  Settings2,
  Shield,
  User,
} from 'lucide-react'

import { cn, initialOf } from '@/lib/utils'
import { logout } from '@/lib/auth-client'
import { ROUTES } from '@/lib/routes'
import { getSearchPath, isSearchSectionPath } from '@/lib/search-tab'
import { useCurrentUser } from '@/components/current-user-provider'
import { useAuthGate } from '@/components/auth-prompt-provider'
import { useNotifications } from '@/components/notifications-provider'
import { useMessages } from '@/components/messages-provider'
import { useT } from '@/components/language-provider'
import { ThemeToggle } from '@/components/theme-toggle'
import { CustomThemeDialog } from '@/components/custom-theme-dialog'
import type { UserRole } from '@/types'
import { CreatePostDialog } from '@/components/feed/create-post-dialog'
import { ActivityPresenceDot } from '@/components/profil/activity-presence-dot'
import { CertificationBadge } from '@/components/profil/certification-badge'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'

interface NavItem {
  /** Clé i18n du libellé (cf. lib/i18n.ts, namespace `nav`). */
  labelKey: string
  href: string
  icon: React.ElementType
  roles?: UserRole[]
  /** Valeur `data-tour` (cible du didacticiel), si l'item est étape du tour. */
  tour?: string
}

const NAV_ITEMS: NavItem[] = [
  { labelKey: 'nav.feed', href: ROUTES.feed, icon: Home, tour: 'nav-feed' },
  { labelKey: 'nav.explore', href: '/explorer', icon: Search, tour: 'nav-explorer' },
  { labelKey: 'nav.notifications', href: '/notifications', icon: Bell, tour: 'nav-notifications' },
  { labelKey: 'nav.messages', href: '/messages', icon: Mail, tour: 'nav-messages' },
  { labelKey: 'nav.bookmarks', href: ROUTES.bookmarks, icon: Bookmark, tour: 'nav-bookmarks' },
  { labelKey: 'nav.profil', href: ROUTES.profil, icon: User, tour: 'nav-profil' },
  {
    labelKey: 'nav.moderation',
    href: ROUTES.moderation,
    icon: Shield,
    roles: ['moderator', 'administrator'],
  },
  {
    labelKey: 'nav.admin',
    href: ROUTES.admin,
    icon: Settings2,
    roles: ['administrator'],
  },
]

export function SidebarLeft() {
  const t = useT()
  const pathname = usePathname()
  const { session, profil } = useCurrentUser()
  const { isVisitor } = useAuthGate()
  const { unreadCount } = useNotifications()
  const { unreadCount: msgUnread } = useMessages()
  const [themeDialogOpen, setThemeDialogOpen] = useState(false)

  // Rôle issu du JWT : `null` au 1er rendu (hydratation), puis renseigné →
  // les liens Modération/Admin apparaissent ensuite.
  const role = session?.role ?? null
  // Visiteur : seul « Accueil » (le fil public) reste accessible — les autres
  // entrées (recherche, notifications, messages, signets, profil, modération…)
  // requièrent une session.
  const visibleItems = isVisitor
    ? NAV_ITEMS.filter((item) => item.href === ROUTES.feed)
    : NAV_ITEMS.filter(
        (item) => !item.roles || (role !== null && item.roles.includes(role)),
      )
  // Loupe (Recherche) : `href` dynamique (mémoire de navigation par onglet).
  // Depuis un autre onglet → dernier chemin de la section recherche ; déjà dans la
  // section → recherche neuve (`/explorer`). Href natif d'ancre (fiable iOS).
  const [searchHref, setSearchHref] = useState<string>(ROUTES.explorer)
  useEffect(() => {
    setSearchHref(isSearchSectionPath(pathname) ? ROUTES.explorer : getSearchPath())
  }, [pathname])

  const fallbackInitial = initialOf(profil?.displayName, profil?.username)
  // Avant le chargement du profil (username vide) on affiche un libellé traduit.
  const shownName = profil?.username ? profil.displayName : t('common.user')
  const handle = profil?.username ? `@${profil.username}` : `@${t('common.username_fallback')}`
  const displayedRole = role

  return (
    <aside className="sticky top-0 hidden h-screen w-[275px] flex-col justify-between overflow-y-auto px-3 py-4 lg:flex">
      {/* Logo */}
      <div className="flex flex-col gap-1">
        <Link
          href={ROUTES.feed}
          scroll={false}
          className="mb-3 flex w-fit items-center rounded-2xl p-2 transition hover:scale-105"
        >
          <Image
            src="/logo_only.png"
            alt="Breezy"
            width={512}
            height={512}
            className="h-11 w-11 object-contain"
            priority
          />
        </Link>

        {/* Nav items */}
        {visibleItems.map((item) => {
          const Icon = item.icon
          const active = pathname === item.href
          const badgeCount =
            item.href === ROUTES.notifications
              ? unreadCount
              : item.href === ROUTES.messages
                ? msgUnread
                : 0
          const showBadge = badgeCount > 0
          const badgeAria =
            item.href === ROUTES.messages
              ? t('messages.badge_aria', { count: badgeCount })
              : t('notifications.unread_aria', { count: badgeCount })

          return (
            <Link
              key={item.href}
              href={item.href === ROUTES.explorer ? searchHref : item.href}
              scroll={false}
              data-tour={item.tour}
              className={cn(
                'flex w-fit items-center gap-4 rounded-full px-4 py-3 text-xl font-normal text-foreground/80 transition hover:bg-accent hover:text-[#5B6CFF] hover:shadow-sm dark:hover:text-[#9aa6ff]',
                active &&
                  'bg-accent font-bold text-[#5B6CFF] shadow-[0_14px_34px_rgba(91,108,255,0.16)] dark:text-[#9aa6ff]',
              )}
            >
              <span className="relative shrink-0">
                <Icon
                  className={cn('h-6 w-6', active && 'stroke-[2.5]')}
                  aria-hidden
                />
                {showBadge && (
                  <span
                    aria-label={badgeAria}
                    className="absolute -right-2 -top-1.5 flex h-[18px] min-w-[18px] items-center justify-center rounded-full bg-gradient-to-r from-[var(--brand-from)] via-[var(--brand-via)] to-[var(--brand-to)] px-1 text-[11px] font-bold leading-none text-white shadow"
                  >
                    {badgeCount > 99 ? '99+' : badgeCount}
                  </span>
                )}
              </span>
              <span>{t(item.labelKey)}</span>
            </Link>
          )
        })}

        {/* Post button : ouvre la popup de publication (masqué pour le visiteur). */}
        {!isVisitor && (
          <CreatePostDialog>
            <Button
              size="lg"
              data-tour="compose"
              className="mt-4 w-[90%] rounded-full bg-gradient-to-r from-[var(--brand-from)] via-[var(--brand-via)] to-[var(--brand-to)] text-base font-bold text-white shadow-[0_18px_44px_rgba(91,108,255,0.3)] transition hover:scale-[1.015]"
            >
              {t('nav.post')}
            </Button>
          </CreatePostDialog>
        )}
      </div>

      {/* Bas de la sidebar : menu utilisateur — ou appel à la connexion (visiteur) */}
      {isVisitor ? (
        <div className="flex flex-col gap-2" aria-label={t('visitor.cta_aria')}>
          <Button
            asChild
            size="lg"
            className="w-full rounded-full bg-gradient-to-r from-[var(--brand-from)] via-[var(--brand-via)] to-[var(--brand-to)] text-base font-bold text-white shadow-[0_18px_44px_rgba(91,108,255,0.3)] transition hover:scale-[1.015]"
          >
            <Link href={ROUTES.login}>{t('visitor.login')}</Link>
          </Button>
          <Button asChild size="lg" variant="outline" className="w-full rounded-full text-base font-bold">
            <Link href={ROUTES.register}>{t('visitor.register')}</Link>
          </Button>
        </div>
      ) : (
      <div className="flex flex-col gap-2">
        {/* User menu : identité réelle chargée via profil-service */}
        <DropdownMenu modal={false}>
          <DropdownMenuTrigger asChild>
            <button className="panel flex w-full items-center gap-3 rounded-full border p-3 shadow-sm transition hover:shadow-[0_14px_34px_rgba(91,108,255,0.16)]">
              <Avatar className="h-10 w-10 shrink-0">
                {profil?.avatarUrl && (
                  <AvatarImage src={profil.avatarUrl} alt={profil.displayName} />
                )}
                <AvatarFallback>{fallbackInitial}</AvatarFallback>
                <ActivityPresenceDot userId={profil?.userId ?? ''} />
              </Avatar>
              <div className="flex min-w-0 flex-1 flex-col text-left">
                <span className="flex min-w-0 items-center gap-1 text-sm font-bold">
                  <span className="truncate">{shownName}</span>
                  <CertificationBadge userId={profil?.userId} certification={profil?.certification} role={profil?.role} className="h-4 w-4" />
                </span>
                <span className="truncate text-sm text-muted-foreground">{handle}</span>
              </div>
              <MoreHorizontal className="h-5 w-5 shrink-0 text-muted-foreground" aria-hidden />
            </button>
          </DropdownMenuTrigger>
          <DropdownMenuContent
            align="end"
            side="top"
            className="panel z-40 w-56 border shadow-[0_18px_44px_rgba(91,108,255,0.18)]"
          >
            <DropdownMenuLabel>
              <span className="block font-bold">{shownName}</span>
              <span className="block text-xs font-normal text-muted-foreground">
                {displayedRole ? t(`role.${displayedRole}`) : t('common.not_connected')}
              </span>
            </DropdownMenuLabel>
            <DropdownMenuSeparator />
            <ThemeToggle
              onCustomize={() => {
                // Laisse le menu se fermer, puis ouvre la popup au tick suivant
                // (évite la course de focus Radix dropdown ↔ dialog).
                setTimeout(() => setThemeDialogOpen(true), 0)
              }}
            />
            <DropdownMenuSeparator />
            <DropdownMenuItem asChild>
              <Link href={ROUTES.parametres} scroll={false}>
                <Settings className="mr-2 h-4 w-4" />
                {t('nav.settings')}
              </Link>
            </DropdownMenuItem>
            <DropdownMenuItem onSelect={() => void logout()}>
              <LogOut className="mr-2 h-4 w-4" />
              {t('common.logout')}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>

        {/* Popup de thème personnalisé (frère du menu : ne se démonte pas
            quand le dropdown se ferme). */}
        <CustomThemeDialog open={themeDialogOpen} onOpenChange={setThemeDialogOpen} />
      </div>
      )}
    </aside>
  )
}
