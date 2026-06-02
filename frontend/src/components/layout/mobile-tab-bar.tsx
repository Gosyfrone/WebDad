'use client'

import Image from 'next/image'
import Link from 'next/link'
import { usePathname } from 'next/navigation'
import { Bell, Mail, Search } from 'lucide-react'

import { cn } from '@/lib/utils'
import { ROUTES } from '@/lib/routes'

interface TabItem {
  href: string
  label: string
  /** Icône lucide ; `null` = logo Breezy (Accueil). */
  icon: React.ElementType | null
}

const TABS: TabItem[] = [
  { href: ROUTES.feed, label: 'Accueil', icon: null },
  { href: ROUTES.explorer, label: 'Recherche', icon: Search },
  { href: ROUTES.notifications, label: 'Notifications', icon: Bell },
  { href: ROUTES.messages, label: 'Messages', icon: Mail },
]

/**
 * Barre de navigation fixée en bas (masquée ≥ lg), façon X.com mobile.
 * « Accueil » utilise le logo Breezy seul (`logo_only.png`).
 */
export function MobileTabBar() {
  const pathname = usePathname()

  return (
    <nav
      aria-label="Navigation principale"
      className="fixed inset-x-0 bottom-0 z-40 flex h-14 border-t bg-background/95 backdrop-blur lg:hidden"
    >
      {TABS.map((tab) => {
        const active = pathname === tab.href
        const Icon = tab.icon

        return (
          <Link
            key={tab.href}
            href={tab.href}
            aria-label={tab.label}
            aria-current={active ? 'page' : undefined}
            className="flex flex-1 items-center justify-center transition-colors hover:bg-muted/40"
          >
            {Icon ? (
              <Icon
                className={cn('h-6 w-6', active ? 'text-primary' : 'text-muted-foreground')}
                aria-hidden
              />
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
