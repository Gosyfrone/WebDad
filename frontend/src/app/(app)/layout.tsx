import { Suspense } from 'react'

import { SidebarLeft } from '@/components/layout/sidebar-left'
import { SidebarRight } from '@/components/layout/sidebar-right'
import { MobileHeader } from '@/components/layout/mobile-header'
import { VisitorThemeToggle } from '@/components/layout/visitor-theme-toggle'
import { MobileTabBar } from '@/components/layout/mobile-tab-bar'
import { ComposeFab } from '@/components/layout/compose-fab'
import { CurrentUserProvider } from '@/components/current-user-provider'
import { NotificationsProvider } from '@/components/notifications-provider'
import { MessagesProvider } from '@/components/messages-provider'
import { AuthPromptProvider } from '@/components/auth-prompt-provider'
import { OnboardingGate } from '@/components/onboarding/onboarding-gate'
import { PasswordChangeGate } from '@/components/account/password-change-gate'
import { TermsAcceptGate } from '@/components/account/terms-accept-gate'
import { UsernamePendingGate } from '@/components/account/username-pending-gate'
import { ExplorerFilterProvider } from '@/components/explorer/explorer-filter-context'
import { FeedView } from '@/components/feed/feed-view'
import { OverlayScrollLock } from '@/components/feed/overlay-scroll-lock'
import { ActivityLifecycle } from '@/components/activity/activity-lifecycle'
import { WarningsGate } from '@/components/moderation/warnings-gate'

/**
 * Layout de l'espace authentifié, responsive : en-tête mobile + tab bar < lg,
 * sidebar gauche ≥ lg, colonne de droite ≥ xl.
 *
 * **Feed persistant.** `FeedView` est monté une seule fois ici (colonne centrale) :
 * il porte le scroll de la fenêtre et conserve son état sur toute la navigation.
 * Les autres sections se rendent en overlay plein écran par-dessus, `/feed` se
 * rend en `null` (fond visible) — d'où l'abandon des parallel/intercepting routes
 * (`@modal`), source des bugs 404/refresh/scroll.
 */
export default function AppLayout({ children }: { children: React.ReactNode }) {
  return (
    <CurrentUserProvider>
      <AuthPromptProvider>
        <NotificationsProvider>
          <MessagesProvider>
            <ExplorerFilterProvider>
              <div className="bg-page relative flex min-h-screen justify-center overflow-x-clip">
                <div className="bg-page-glow-1 pointer-events-none fixed inset-0" />
                <div className="bg-page-glow-2 pointer-events-none fixed inset-0" />

                <div className="relative flex w-full max-w-[1265px]">
                  <SidebarLeft />

                  {/* Colonne centrale. overflow-x-clip : pas de scroll horizontal parasite
                      sur mobile, sans créer de conteneur de scroll (≠ overflow-x-hidden, qui
                      casserait les en-têtes sticky). */}
                  <div className="glass-column flex min-h-screen w-full min-w-0 flex-1 flex-col overflow-x-clip backdrop-blur-2xl lg:border-x">
                    {/* pt-14 / pb-16 : dégagent l'en-tête mobile fixe et la tab bar (masqués ≥ lg). */}
                    <main className="flex-1 pb-16 pt-14 lg:pb-0 lg:pt-0">
                      <Suspense fallback={null}>
                        <FeedView />
                      </Suspense>
                    </main>
                  </div>

                  <SidebarRight />
                </div>

                {/* Chrome mobile (masqué ≥ lg) */}
                <MobileTabBar />
                <ComposeFab />

                {/* Onboarding OAuth sans profil, puis mot de passe temporaire et username
                    provisoire (comptes créés par un admin). */}
                <OnboardingGate />
                <PasswordChangeGate />
                <TermsAcceptGate />
                <UsernamePendingGate />
                <ActivityLifecycle />
                <WarningsGate />

                {/* Gèle le défilement du feed quand un overlay est ouvert (≠ /feed). */}
                <OverlayScrollLock />

                {/* Overlays des sections, rendus au niveau racine (hors colonne centrale)
                    pour que leur `fixed` se réfère au viewport, pas au bloc créé par le
                    `backdrop-blur` de la colonne. `/feed` se rend en `null`. */}
                {children}

                {/* En-tête mobile (masqué ≥ lg), rendu APRÈS les overlays en `fixed z-[60]`
                    pour rester au-dessus d'eux → header unique sur toutes les pages. */}
                <MobileHeader />

                {/* Interrupteur clair/sombre visiteur (desktop uniquement ; sur mobile
                    c'est l'en-tête qui le porte, à la place de la cloche). */}
                <VisitorThemeToggle />
              </div>
            </ExplorerFilterProvider>
          </MessagesProvider>
        </NotificationsProvider>
      </AuthPromptProvider>
    </CurrentUserProvider>
  )
}
