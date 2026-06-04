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
import { logout } from '@/lib/auth-client'
import { ROUTES } from '@/lib/routes'
import type { UserRole } from '@/types'
import { CreatePostDialog } from '@/components/feed/create-post-dialog'
import { ThemeToggle } from '@/components/theme-toggle'
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
              <Icon
                className={cn('h-6 w-6 shrink-0', active && 'stroke-[2.5]')}
                aria-hidden
              />
              <span>{item.label}</span>
            </Link>
          )
        })}

        {/* Post button : ouvre la popup de publication */}
        <CreatePostDialog>
          <Button
            size="lg"
            className="mt-4 w-[90%] rounded-full bg-gradient-to-r from-[#8D3DFF] via-[#5B6CFF] to-[#47D9FF] text-base font-bold text-white shadow-[0_18px_44px_rgba(91,108,255,0.3)] transition hover:scale-[1.015]"
          >
            Breezer
          </Button>
        </CreatePostDialog>
      </div>

      {/* Bas de la sidebar : sélecteur de thème + menu utilisateur */}
      <div className="flex flex-col gap-2">
        {/* Interrupteur clair/sombre (même composant que le tiroir mobile),
            posé sur un panneau pastel pour s'accorder à la card user. */}
        <div className="panel rounded-2xl border px-1 py-1 shadow-sm">
          <ThemeToggle />
        </div>

        {/* User menu */}
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <button className="panel flex w-full items-center gap-3 rounded-full border p-3 shadow-sm transition hover:shadow-[0_14px_34px_rgba(91,108,255,0.16)]">
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
          <DropdownMenuContent
            align="end"
            side="top"
            className="panel w-56 border shadow-[0_18px_44px_rgba(91,108,255,0.18)]"
          >
            <DropdownMenuLabel>
              <span className="block font-bold">{username}</span>
              <span className="block text-xs font-normal text-muted-foreground">
                {role ?? 'non connecté'}
              </span>
            </DropdownMenuLabel>
            <DropdownMenuSeparator />
            <DropdownMenuItem onSelect={() => void logout()}>
              <LogOut className="mr-2 h-4 w-4" />
              Se déconnecter
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </aside>
  )
}
