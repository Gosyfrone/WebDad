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

import { cn } from '@/lib/utils'
import { logout } from '@/lib/auth-client'
import { getMyProfil, subscribeProfilUpdated } from '@/lib/profil-client'
import { useSession } from '@/lib/session'
import { ROUTES } from '@/lib/routes'
import { useNotifications } from '@/components/notifications-provider'
import { useMessages } from '@/components/messages-provider'
import { useT } from '@/components/language-provider'
import { ThemeToggle } from '@/components/theme-toggle'
import type { ProfilDetails, UserRole } from '@/types'
import { CreatePostDialog } from '@/components/feed/create-post-dialog'
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
}

const NAV_ITEMS: NavItem[] = [
  { labelKey: 'nav.feed', href: ROUTES.feed, icon: Home },
  { labelKey: 'nav.explore', href: '/explorer', icon: Search },
  { labelKey: 'nav.notifications', href: '/notifications', icon: Bell },
  { labelKey: 'nav.messages', href: '/messages', icon: Mail },
  { labelKey: 'nav.bookmarks', href: ROUTES.bookmarks, icon: Bookmark },
  { labelKey: 'nav.profil', href: ROUTES.profil, icon: User },
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
  const session = useSession()
  const { unreadCount } = useNotifications()
  const { unreadCount: msgUnread } = useMessages()
  const [account, setAccount] = useState({
    displayName: '',
    username: '',
    avatarUrl: '',
  })

  // Rôle réel issu du JWT (cf. lib/session). `null` au 1er rendu (hydratation),
  // puis renseigné au montage → les liens Modération/Admin apparaissent ensuite.
  const role = session?.role ?? null
  const visibleItems = NAV_ITEMS.filter(
    (item) => !item.roles || (role !== null && item.roles.includes(role)),
  )
  const fallbackInitial = (account.displayName || account.username || 'U')
    .charAt(0)
    .toUpperCase()
  // Avant le chargement du profil (username vide) on affiche un libellé traduit.
  const shownName = account.username ? account.displayName : t('common.user')
  const handle = account.username ? `@${account.username}` : `@${t('common.username_fallback')}`
  const displayedRole = role

  useEffect(() => {
    let cancelled = false

    function applyProfil(profil: ProfilDetails) {
      setAccount({
        displayName: profil.displayName,
        username: profil.username,
        avatarUrl: profil.avatarUrl,
      })
    }

    async function loadAccount() {
      try {
        const profil = await getMyProfil()
        if (!cancelled) {
          applyProfil(profil)
        }
      } catch {
        // Le layout reste utilisable avec le fallback pendant une session expirée.
      }
    }

    const unsubscribe = subscribeProfilUpdated(applyProfil)
    void loadAccount()
    return () => {
      cancelled = true
      unsubscribe()
    }
  }, [])

  return (
    <aside className="sticky top-0 hidden h-screen w-[275px] flex-col justify-between overflow-y-auto px-3 py-4 lg:flex">
      {/* Logo */}
      <div className="flex flex-col gap-1">
        <Link
          href={ROUTES.feed}
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
              href={item.href}
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
                    className="absolute -right-2 -top-1.5 flex h-[18px] min-w-[18px] items-center justify-center rounded-full bg-gradient-to-r from-[#8D3DFF] via-[#5B6CFF] to-[#47D9FF] px-1 text-[11px] font-bold leading-none text-white shadow"
                  >
                    {badgeCount > 99 ? '99+' : badgeCount}
                  </span>
                )}
              </span>
              <span>{t(item.labelKey)}</span>
            </Link>
          )
        })}

        {/* Post button : ouvre la popup de publication */}
        <CreatePostDialog>
          <Button
            size="lg"
            className="mt-4 w-[90%] rounded-full bg-gradient-to-r from-[#8D3DFF] via-[#5B6CFF] to-[#47D9FF] text-base font-bold text-white shadow-[0_18px_44px_rgba(91,108,255,0.3)] transition hover:scale-[1.015]"
          >
            {t('nav.post')}
          </Button>
        </CreatePostDialog>
      </div>

      {/* Bas de la sidebar : menu utilisateur */}
      <div className="flex flex-col gap-2">
        {/* User menu : identité réelle chargée via profil-service */}
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <button className="panel flex w-full items-center gap-3 rounded-full border p-3 shadow-sm transition hover:shadow-[0_14px_34px_rgba(91,108,255,0.16)]">
              <Avatar className="h-10 w-10 shrink-0">
                {account.avatarUrl && (
                  <AvatarImage src={account.avatarUrl} alt={account.displayName} />
                )}
                <AvatarFallback>{fallbackInitial}</AvatarFallback>
              </Avatar>
              <div className="flex min-w-0 flex-1 flex-col text-left">
                <span className="truncate text-sm font-bold">{shownName}</span>
                <span className="truncate text-sm text-muted-foreground">{handle}</span>
              </div>
              <MoreHorizontal className="h-5 w-5 shrink-0 text-muted-foreground" aria-hidden />
            </button>
          </DropdownMenuTrigger>
          <DropdownMenuContent
            align="end"
            side="top"
            className="panel w-56 border shadow-[0_18px_44px_rgba(91,108,255,0.18)]"
          >
            <DropdownMenuLabel>
              <span className="block font-bold">{shownName}</span>
              <span className="block text-xs font-normal text-muted-foreground">
                {displayedRole ? t(`role.${displayedRole}`) : t('common.not_connected')}
              </span>
            </DropdownMenuLabel>
            <DropdownMenuSeparator />
            <ThemeToggle />
            <DropdownMenuSeparator />
            <DropdownMenuItem asChild>
              <Link href={ROUTES.parametres}>
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
      </div>
    </aside>
  )
}
