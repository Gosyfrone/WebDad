'use client'

import { useEffect, useState } from 'react'
import Image from 'next/image'
import Link from 'next/link'
import { usePathname } from 'next/navigation'
import { LogOut, Settings } from 'lucide-react'

import { cn } from '@/lib/utils'
import { logout } from '@/lib/auth-client'
import { ROUTES, navItemsForRole } from '@/lib/routes'
import type { UserRole } from '@/types'
import { ThemeToggle } from '@/components/theme-toggle'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import {
  Sheet,
  SheetClose,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from '@/components/ui/sheet'

interface MobileHeaderProps {
  role: UserRole | null
  username?: string
}

/** Largeur (px) de la zone de bord gauche sensible au swipe d'ouverture. */
const EDGE_ZONE = 24
/** Déplacement horizontal (px) minimal pour valider le swipe. */
const SWIPE_THRESHOLD = 60

/**
 * En-tête mobile (masqué ≥ lg) : photo de profil cliquable à gauche ouvrant un
 * tiroir latéral de navigation (Fil / Profil / Modération / Admin selon rôle),
 * logo Breezy centré.
 *
 * Le tiroir s'ouvre au clic sur l'avatar OU par un swipe depuis le bord gauche
 * de l'écran vers la droite (geste natif façon X.com).
 *
 * Les pages disposant déjà de leur propre en-tête (ex. profil) ne l'affichent
 * pas → on rend `null` sur ces routes (le swipe y est aussi désactivé).
 */
export function MobileHeader({ role, username = 'Utilisateur' }: MobileHeaderProps) {
  const pathname = usePathname()
  const hidden = pathname?.startsWith(ROUTES.profil) ?? false
  const [open, setOpen] = useState(false)

  // Ouverture par swipe depuis le bord gauche (→ droite).
  useEffect(() => {
    if (hidden) return

    let startX = 0
    let startY = 0
    let tracking = false

    function onTouchStart(e: TouchEvent) {
      const touch = e.touches[0]
      tracking = touch.clientX <= EDGE_ZONE
      startX = touch.clientX
      startY = touch.clientY
    }

    function onTouchEnd(e: TouchEvent) {
      if (!tracking) return
      tracking = false
      const touch = e.changedTouches[0]
      const dx = touch.clientX - startX
      const dy = Math.abs(touch.clientY - startY)
      // Mouvement majoritairement horizontal, vers la droite, suffisamment ample.
      if (dx > SWIPE_THRESHOLD && dy < dx) setOpen(true)
    }

    window.addEventListener('touchstart', onTouchStart, { passive: true })
    window.addEventListener('touchend', onTouchEnd, { passive: true })
    return () => {
      window.removeEventListener('touchstart', onTouchStart)
      window.removeEventListener('touchend', onTouchEnd)
    }
  }, [hidden])

  if (hidden) return null

  const navItems = navItemsForRole(role)

  return (
    <header className="panel sticky top-0 z-30 flex h-14 items-center border-b px-3 shadow-sm lg:hidden">
      {/* Photo de profil -> tiroir de navigation latéral gauche */}
      <Sheet open={open} onOpenChange={setOpen}>
        <SheetTrigger asChild>
          <button
            aria-label="Ouvrir le menu de navigation"
            className="rounded-full ring-offset-background transition-opacity hover:opacity-80 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
          >
            <Avatar className="h-8 w-8">
              <AvatarFallback>{username.charAt(0).toUpperCase()}</AvatarFallback>
            </Avatar>
          </button>
        </SheetTrigger>

        <SheetContent side="left" className="panel-y flex w-72 flex-col p-0 shadow-[0_24px_70px_rgba(91,108,255,0.22)]">
          {/* Identité */}
          <SheetHeader className="border-b p-4 text-left">
            <div className="flex items-center gap-3">
              <Avatar className="h-12 w-12">
                <AvatarFallback className="text-lg">
                  {username.charAt(0).toUpperCase()}
                </AvatarFallback>
              </Avatar>
              <div className="flex min-w-0 flex-col">
                <SheetTitle className="truncate">{username}</SheetTitle>
                <SheetDescription className="truncate">
                  @{username.toLowerCase()} · {role ?? 'non connecté'}
                </SheetDescription>
              </div>
            </div>
          </SheetHeader>

          {/* Navigation par section (filtrée par rôle) */}
          <nav className="flex flex-1 flex-col gap-1 p-2">
            {navItems.map((item) => {
              const active = pathname === item.href
              return (
                <SheetClose asChild key={item.href}>
                  <Link
                    href={item.href}
                    className={cn(
                      'rounded-2xl px-3 py-3 text-base transition-colors hover:bg-accent',
                      active && 'bg-accent font-bold text-[#5B6CFF] shadow-sm dark:text-[#9aa6ff]',
                    )}
                  >
                    {item.label}
                  </Link>
                </SheetClose>
              )
            })}
          </nav>

          {/* Thème + Paramètres + déconnexion */}
          <div className="border-t p-2">
            {/* Le sélecteur de thème ne ferme pas le tiroir (on garde le retour visuel) */}
            <ThemeToggle />
            <SheetClose asChild>
              <Link
                href={ROUTES.parametres}
                className="flex w-full items-center gap-3 rounded-2xl px-3 py-3 text-base transition-colors hover:bg-accent"
              >
                <Settings className="h-5 w-5" />
                Paramètres
              </Link>
            </SheetClose>
            <button
              type="button"
              onClick={() => void logout()}
              className="flex w-full items-center gap-3 rounded-md px-3 py-3 text-base transition-colors hover:bg-accent"
            >
              <LogOut className="h-5 w-5" />
              Se déconnecter
            </button>
          </div>
        </SheetContent>
      </Sheet>

      {/* Logo Breezy centré */}
      <Link
        href={ROUTES.feed}
        aria-label="Accueil"
        className="absolute left-1/2 -translate-x-1/2"
      >
        <Image
          src="/logo_only.png"
          alt="Breezy"
          width={512}
          height={512}
          className="h-8 w-8 object-contain"
          priority
        />
      </Link>

      {/* Contrepoids pour équilibrer le logo centré */}
      <span className="ml-auto h-8 w-8" aria-hidden />
    </header>
  )
}
