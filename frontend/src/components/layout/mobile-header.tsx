'use client'

import { useEffect, useState } from 'react'
import Image from 'next/image'
import Link from 'next/link'
import { usePathname } from 'next/navigation'
import { LogOut, Palette, Settings } from 'lucide-react'

import { cn } from '@/lib/utils'
import { logout } from '@/lib/auth-client'
import { getMyProfil, subscribeProfilUpdated } from '@/lib/profil-client'
import { useSession } from '@/lib/session'
import { ROUTES, navItemsForRole } from '@/lib/routes'
import type { ProfilDetails } from '@/types'
import { useT } from '@/components/language-provider'
import { ThemeToggle } from '@/components/theme-toggle'
import { CustomThemeDialog } from '@/components/custom-theme-dialog'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import {
  Sheet,
  SheetClose,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from '@/components/ui/sheet'

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
export function MobileHeader() {
  const t = useT()
  const pathname = usePathname()
  const session = useSession()
  const hidden = pathname?.startsWith(ROUTES.profil) ?? false
  const [open, setOpen] = useState(false)
  const [themeDialogOpen, setThemeDialogOpen] = useState(false)
  const [account, setAccount] = useState({
    displayName: '',
    username: '',
    avatarUrl: '',
  })

  // Rôle réel issu du JWT (cf. lib/session) ; `null` au 1er rendu (hydratation).
  const role = session?.role ?? null
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
        if (!cancelled) applyProfil(profil)
      } catch {
        // Le header conserve le fallback si la session est expirée.
      }
    }

    const unsubscribe = subscribeProfilUpdated(applyProfil)
    void loadAccount()
    return () => {
      cancelled = true
      unsubscribe()
    }
  }, [])

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
    <header className="panel sticky top-0 z-[60] isolate flex h-14 items-center border-b px-3 shadow-sm backdrop-blur-2xl lg:hidden">
      {/* Photo de profil -> tiroir de navigation latéral gauche */}
      <Sheet open={open} onOpenChange={setOpen}>
        <SheetTrigger asChild>
          <button
            aria-label={t('nav.open_menu')}
            className="rounded-full ring-offset-background transition-opacity hover:opacity-80 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
          >
            <Avatar className="h-8 w-8">
              {account.avatarUrl && (
                <AvatarImage src={account.avatarUrl} alt={account.displayName} />
              )}
              <AvatarFallback>{fallbackInitial}</AvatarFallback>
            </Avatar>
          </button>
        </SheetTrigger>

        <SheetContent side="left" className="panel-y flex w-72 flex-col p-0 shadow-[0_24px_70px_rgba(91,108,255,0.22)]">
          {/* Identité */}
          <SheetHeader className="border-b p-4 text-left">
            <div className="flex items-center gap-3">
              <Avatar className="h-12 w-12">
                {account.avatarUrl && (
                  <AvatarImage src={account.avatarUrl} alt={account.displayName} />
                )}
                <AvatarFallback className="text-lg">{fallbackInitial}</AvatarFallback>
              </Avatar>
              <div className="flex min-w-0 flex-col">
                <SheetTitle className="truncate">{shownName}</SheetTitle>
                <SheetDescription className="truncate">
                  {handle} · {displayedRole ? t(`role.${displayedRole}`) : t('common.not_connected')}
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
                    {t(item.labelKey)}
                  </Link>
                </SheetClose>
              )
            })}
          </nav>

          {/* Paramètres + déconnexion */}
          <div className="border-t p-2">
            <div className="mb-1 rounded-2xl px-2 py-2">
              <ThemeToggle />
            </div>
            <button
              type="button"
              onClick={() => {
                // Ferme le tiroir, puis ouvre la popup au tick suivant
                // (évite la course de focus Sheet ↔ Dialog).
                setOpen(false)
                setTimeout(() => setThemeDialogOpen(true), 0)
              }}
              className="flex w-full items-center gap-3 rounded-2xl px-3 py-3 text-base transition-colors hover:bg-accent"
            >
              <Palette className="h-5 w-5" />
              {t('theme.customize')}
            </button>
            <SheetClose asChild>
              <Link
                href={ROUTES.parametres}
                className="flex w-full items-center gap-3 rounded-2xl px-3 py-3 text-base transition-colors hover:bg-accent"
              >
                <Settings className="h-5 w-5" />
                {t('nav.settings')}
              </Link>
            </SheetClose>
            <button
              type="button"
              onClick={() => void logout()}
              className="flex w-full items-center gap-3 rounded-md px-3 py-3 text-base transition-colors hover:bg-accent"
            >
              <LogOut className="h-5 w-5" />
              {t('common.logout')}
            </button>
          </div>
        </SheetContent>
      </Sheet>

      {/* Logo Breezy centré */}
      <Link
        href={ROUTES.feed}
        aria-label={t('nav.home')}
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

      {/* Popup de thème personnalisé (frère du tiroir : ne se démonte pas
          quand le Sheet se ferme). */}
      <CustomThemeDialog open={themeDialogOpen} onOpenChange={setThemeDialogOpen} />
    </header>
  )
}
