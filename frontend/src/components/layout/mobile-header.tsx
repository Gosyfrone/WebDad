'use client'

import { useEffect, useState } from 'react'
import Image from 'next/image'
import Link from 'next/link'
import { usePathname, useRouter } from 'next/navigation'
import { ArrowLeft, Bell, LogIn, LogOut, Settings } from 'lucide-react'

import { cn, initialOf } from '@/lib/utils'
import { logout } from '@/lib/auth-client'
import { ROUTES, navItemsForRole } from '@/lib/routes'
import { resolveMobileHeaderNav } from '@/lib/mobile-header-nav'
import { useCurrentUser } from '@/components/current-user-provider'
import { useAuthGate } from '@/components/auth-prompt-provider'
import { useNotifications } from '@/components/notifications-provider'
import { useMessages } from '@/components/messages-provider'
import { useT } from '@/components/language-provider'
import { ThemeToggle } from '@/components/theme-toggle'
import { CustomThemeDialog } from '@/components/custom-theme-dialog'
import { ActivityPresenceDot } from '@/components/profil/activity-presence-dot'
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
 * En-tête mobile contextuel (masqué ≥ lg). Selon la section courante :
 *   - gauche : photo de profil → tiroir de navigation, OU flèche ← (page Notifs)
 *     pour revenir au « menu normal » (page précédente, repli /feed) ;
 *   - centre : logo Breezy (feed / autres pages) ou titre de la section
 *     (« Explorer », « Messages », « Notifications ») ;
 *   - droite : bouton 🔔 vers /notifications (avec pastille non-lus) sur le
 *     feed / explorer / messages ; vide ailleurs.
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
  const router = useRouter()
  const { session, profil } = useCurrentUser()
  const { isVisitor } = useAuthGate()
  const { unreadCount } = useNotifications()
  const { activeConversationId } = useMessages()
  const hidden = pathname?.startsWith(ROUTES.profil) ?? false

  // Section courante (pilote les 3 zones gauche/centre/droite), titre centré,
  // cloche notifs et flèche retour : dérivés purs du pathname (cf. lib testée).
  const { section, titleKey, showBell, showBack } = resolveMobileHeaderNav(pathname)

  const [open, setOpen] = useState(false)
  const [themeDialogOpen, setThemeDialogOpen] = useState(false)

  // Rôle issu du JWT ; `null` au 1er rendu (hydratation).
  const role = session?.role ?? null
  const fallbackInitial = initialOf(profil?.displayName, profil?.username)
  // Avant le chargement du profil (username vide) on affiche un libellé traduit.
  const shownName = profil?.username ? profil.displayName : t('common.user')
  const handle = profil?.username ? `@${profil.username}` : `@${t('common.username_fallback')}`
  const displayedRole = role

  // Ouverture par swipe depuis le bord gauche (→ droite). Désactivée pour le
  // visiteur (pas de tiroir) et sur les pages à flèche retour (gauche ≠ avatar).
  useEffect(() => {
    if (hidden || isVisitor || showBack) return

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
  }, [hidden, isVisitor, section])

  // En-tête contextuel pour le feed, les sections de nav principale (explorer /
  // messages / notifications), les paramètres (variante flèche retour) et les
  // pages admin / modération / signets (header type-feed : avatar ← titre →
  // cloche). Les autres pages (profil) conservent leur propre en-tête → pas de
  // header global ni de décalage `pt-14` (cf. `headerOffset={false}`).
  if (hidden || section === 'other') return null
  // Conversation ouverte (mobile) : le ChatPane affiche son propre en-tête (nom +
  // retour) → on efface l'en-tête global pour ne pas le recouvrir/dédoubler.
  if (section === 'messages' && activeConversationId) return null

  const navItems = navItemsForRole(role)

  // « Menu normal » : revient à la page précédente, repli sur le feed s'il n'y
  // a pas d'historique de navigation (ouverture directe / deep-link).
  function goBack() {
    if (typeof window !== 'undefined' && window.history.length > 1) router.back()
    else router.push(ROUTES.feed)
  }

  return (
    <header
      className={cn(
        'panel fixed inset-x-0 top-0 isolate flex h-14 items-center border-b px-3 shadow-sm backdrop-blur-2xl lg:hidden',
        // Tiroir ouvert → header sous l'overlay du Sheet (z-50) pour ne pas masquer
        // l'identité affichée en haut du tiroir ; sinon au-dessus des overlays (z-40).
        open ? 'z-30' : 'z-[60]',
      )}
    >
      {/* Visiteur : pas de tiroir → lien direct vers la connexion. */}
      {isVisitor ? (
        <Link
          href={ROUTES.login}
          aria-label={t('visitor.login')}
          className="flex items-center gap-1.5 rounded-full px-3 py-1.5 text-sm font-bold text-[#5B6CFF] transition-colors hover:bg-accent dark:text-[#9aa6ff]"
        >
          <LogIn className="h-5 w-5" aria-hidden />
          <span>{t('visitor.login')}</span>
        </Link>
      ) : showBack ? (
        /* Pages secondaires (notifs / paramètres) : flèche retour vers le
           « menu normal » (page précédente). */
        <button
          type="button"
          onClick={goBack}
          aria-label={t('nav.back')}
          className="-ml-1 flex h-9 w-9 items-center justify-center rounded-full text-foreground transition-colors hover:bg-accent"
        >
          <ArrowLeft className="h-5 w-5" />
        </button>
      ) : (
      /* Photo de profil -> tiroir de navigation latéral gauche */
      <Sheet open={open} onOpenChange={setOpen}>
        <SheetTrigger asChild>
          <button
            aria-label={t('nav.open_menu')}
            className="rounded-full ring-offset-background transition-opacity hover:opacity-80 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
          >
            <Avatar className="h-8 w-8">
              {profil?.avatarUrl && (
                <AvatarImage src={profil.avatarUrl} alt={profil.displayName} />
              )}
              <AvatarFallback>{fallbackInitial}</AvatarFallback>
              <ActivityPresenceDot userId={profil?.userId ?? ''} className="h-2.5 w-2.5" />
            </Avatar>
          </button>
        </SheetTrigger>

        <SheetContent side="left" className="panel-y flex w-72 flex-col p-0 shadow-[0_24px_70px_rgba(91,108,255,0.22)]">
          {/* Identité */}
          <SheetHeader className="border-b p-4 text-left">
            <div className="flex items-center gap-3">
              <Avatar className="h-12 w-12">
                {profil?.avatarUrl && (
                  <AvatarImage src={profil.avatarUrl} alt={profil.displayName} />
                )}
                <AvatarFallback className="text-lg">{fallbackInitial}</AvatarFallback>
                <ActivityPresenceDot userId={profil?.userId ?? ''} />
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
              <ThemeToggle
                onCustomize={() => {
                  // Ferme le tiroir, puis ouvre la popup au tick suivant
                  // (évite la course de focus Sheet ↔ Dialog).
                  setOpen(false)
                  setTimeout(() => setThemeDialogOpen(true), 0)
                }}
              />
            </div>
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
      )}

      {/* Centre : titre de la section nommée, sinon logo Breezy (lien vers le feed). */}
      {titleKey ? (
        <h1 className="absolute left-1/2 -translate-x-1/2 text-lg font-bold">
          {t(titleKey)}
        </h1>
      ) : (
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
      )}

      {/* Droite : cloche vers les notifications (avec pastille), sinon contrepoids. */}
      {showBell ? (
        <Link
          href={ROUTES.notifications}
          scroll={false}
          aria-label={t('nav.notifications')}
          className="ml-auto flex h-9 w-9 items-center justify-center rounded-full text-foreground transition-colors hover:bg-accent"
        >
          <span className="relative">
            <Bell className="h-6 w-6" aria-hidden />
            {unreadCount > 0 && (
              <span
                aria-label={t('notifications.unread_aria', { count: unreadCount })}
                className="absolute -right-2 -top-1.5 flex h-[16px] min-w-[16px] items-center justify-center rounded-full bg-gradient-to-r from-[var(--brand-from)] via-[var(--brand-via)] to-[var(--brand-to)] px-1 text-[10px] font-bold leading-none text-white shadow"
              >
                {unreadCount > 99 ? '99+' : unreadCount}
              </span>
            )}
          </span>
        </Link>
      ) : (
        <span className="ml-auto h-9 w-9" aria-hidden />
      )}

      {/* Popup de thème personnalisé (frère du tiroir : ne se démonte pas
          quand le Sheet se ferme). */}
      <CustomThemeDialog open={themeDialogOpen} onOpenChange={setThemeDialogOpen} />
    </header>
  )
}
