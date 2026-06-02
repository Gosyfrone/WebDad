'use client'

import Image from 'next/image'
import Link from 'next/link'
import { usePathname } from 'next/navigation'
import {
  Bell,
  Home,
  LogOut,
  Mail,
  MoreHorizontal,
  Search,
  Settings2,
  Shield,
  User,
} from 'lucide-react'

import { cn } from '@/lib/utils'
import { ROUTES } from '@/lib/routes'
import type { UserRole } from '@/types'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
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
  label: string
  href: string
  icon: React.ElementType
  roles?: UserRole[]
}

const NAV_ITEMS: NavItem[] = [
  { label: 'Fil', href: ROUTES.feed, icon: Home },
  { label: 'Explorer', href: '/explorer', icon: Search },
  { label: 'Notifications', href: '/notifications', icon: Bell },
  { label: 'Messages', href: '/messages', icon: Mail },
  { label: 'Profil', href: ROUTES.profil, icon: User },
  {
    label: 'Modération',
    href: ROUTES.moderation,
    icon: Shield,
    roles: ['moderator', 'administrator'],
  },
  {
    label: 'Administration',
    href: ROUTES.admin,
    icon: Settings2,
    roles: ['administrator'],
  },
]

interface SidebarLeftProps {
  role: UserRole | null
  username?: string
}

export function SidebarLeft({ role, username = 'Utilisateur' }: SidebarLeftProps) {
  const pathname = usePathname()

  const visibleItems = NAV_ITEMS.filter(
    (item) => !item.roles || (role !== null && item.roles.includes(role)),
  )

  return (
    <aside className="sticky top-0 flex h-screen w-[275px] flex-col justify-between overflow-y-auto px-3 py-4">
      {/* Logo */}
      <div className="flex flex-col gap-1">
        <Link
          href={ROUTES.feed}
          className="mb-2 flex w-fit items-center rounded-xl px-2 py-1 transition-colors hover:bg-accent"
        >
          <Image
            src="/logo_breezy.png"
            alt="Breezy"
            width={1106}
            height={336}
            className="h-12 w-auto object-contain"
            priority
          />
        </Link>

        {/* Nav items */}
        {visibleItems.map((item) => {
          const Icon = item.icon
          const active = pathname === item.href

          return (
            <Link
              key={item.href}
              href={item.href}
              className={cn(
                'flex w-fit items-center gap-4 rounded-full px-4 py-3 text-xl font-normal transition-colors hover:bg-accent',
                active && 'font-bold',
              )}
            >
              <Icon
                className={cn('h-6 w-6 shrink-0', active && 'stroke-[2.5]')}
                aria-hidden
              />
              <span>{item.label}</span>
            </Link>
          )
        })}

        {/* Post button */}
        <Button
          size="lg"
          className="mt-4 w-[90%] rounded-full text-base font-bold"
          disabled
        >
          Poster
        </Button>
      </div>

      {/* User menu at bottom */}
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <button className="flex w-full items-center gap-3 rounded-full p-3 transition-colors hover:bg-accent">
            <Avatar className="h-10 w-10 shrink-0">
              <AvatarFallback>{username.charAt(0).toUpperCase()}</AvatarFallback>
            </Avatar>
            <div className="flex min-w-0 flex-1 flex-col text-left">
              <span className="truncate text-sm font-bold">{username}</span>
              <span className="truncate text-sm text-muted-foreground">
                @{username.toLowerCase()}
              </span>
            </div>
            <MoreHorizontal className="h-5 w-5 shrink-0 text-muted-foreground" aria-hidden />
          </button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" side="top" className="w-56">
          <DropdownMenuLabel>
            <span className="block font-bold">{username}</span>
            <span className="block text-xs font-normal text-muted-foreground">
              {role ?? 'non connecté'}
            </span>
          </DropdownMenuLabel>
          <DropdownMenuSeparator />
          {/* TODO (issue auth) : déconnexion réelle — effacer le JWT et rediriger vers /login */}
          <DropdownMenuItem disabled>
            <LogOut className="mr-2 h-4 w-4" />
            Se déconnecter
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </aside>
  )
}
