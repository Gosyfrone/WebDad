import { Suspense } from 'react'

import { SidebarLeft } from '@/components/layout/sidebar-left'
import { SidebarRight } from '@/components/layout/sidebar-right'
import { MobileHeader } from '@/components/layout/mobile-header'
import { MobileTabBar } from '@/components/layout/mobile-tab-bar'
import { ComposeFab } from '@/components/layout/compose-fab'
import { NotificationsProvider } from '@/components/notifications-provider'
import { MessagesProvider } from '@/components/messages-provider'
import { AuthPromptProvider } from '@/components/auth-prompt-provider'
import { OnboardingGate } from '@/components/onboarding/onboarding-gate'
import { PasswordChangeGate } from '@/components/account/password-change-gate'
import { UsernamePendingGate } from '@/components/account/username-pending-gate'
import { ExplorerFilterProvider } from '@/components/explorer/explorer-filter-context'
import { FeedView } from '@/components/feed/feed-view'
import { OverlayScrollLock } from '@/components/feed/overlay-scroll-lock'

/**
 * Layout de l'espace authentifié, responsive (mobile-first).
 *
 *   - < lg (téléphones, iPad portrait) : en-tête mobile (avatar + logo) +
 *     barre d'onglets fixe en bas + bouton « + » flottant sur le feed.
 *   - ≥ lg (iPad paysage, desktop) : colonne de navigation à gauche + contenu.
 *   - ≥ xl : ajout de la colonne de droite (suggestions / tendances).
 *
 * Le rôle réel est désormais dérivé du JWT par `useSession()` directement dans
 * `SidebarLeft` / `MobileHeader` (cf. lib/session.ts) — plus de placeholder.
 *
 * **Feed en arrière-plan persistant.** Le fil (`FeedView`) est monté **une seule
 * fois ici**, dans la colonne centrale : il porte le défilement de la fenêtre et
 * conserve son état (scroll, posts, WebSocket) sur toute la navigation. Chaque
 * autre section est une page classique qui se rend en overlay plein écran
 * (`FeedOverlay`) par-dessus, et `/feed` se rend en `null` (le fond transparaît).
 * On a ainsi abandonné le combo parallel + intercepting routes (`@modal`),
 * source des bugs 404/refresh/scroll : refresh et deep-link suivent désormais le
 * routing normal, sans cas particulier soft/hard.
 */
export default function AppLayout({ children }: { children: React.ReactNode }) {
  return (
    <AuthPromptProvider>
      <NotificationsProvider>
        <MessagesProvider>
          <ExplorerFilterProvider>
            <div className="bg-page relative flex min-h-screen justify-center overflow-x-clip">
              <div className="bg-page-glow-1 pointer-events-none fixed inset-0" />
              <div className="bg-page-glow-2 pointer-events-none fixed inset-0" />

              <div className="relative flex w-full max-w-[1265px]">
                <SidebarLeft />

                {/* Colonne centrale. overflow-x-clip : empêche tout défilement horizontal
                    parasite sur mobile (sans créer de conteneur de scroll, donc sans
                    casser les en-têtes sticky, contrairement à overflow-x-hidden). */}
                <div className="glass-column flex min-h-screen w-full min-w-0 flex-1 flex-col overflow-x-clip backdrop-blur-2xl lg:border-x">
                  {/* pt-14 : dégage l'en-tête mobile fixe (rendu au niveau racine,
                      au-dessus des overlays — masqué ≥ lg).
                      pb-16 : dégage la barre d'onglets fixe (masquée ≥ lg).
                      Feed persistant : reste monté derrière les overlays. */}
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

              {/* Onboarding bloquant pour les comptes OAuth sans profil (cf. composant). */}
              <OnboardingGate />
              {/* Comptes créés par un admin : changement du mot de passe temporaire
                  (prioritaire), puis du username provisoire si suffixé. */}
              <PasswordChangeGate />
              <UsernamePendingGate />

              {/* Gèle le défilement du feed quand un overlay est ouvert (≠ /feed). */}
              <OverlayScrollLock />

              {/* Overlays des sections. Rendus au niveau racine (hors colonne
                  centrale) pour que leur position `fixed` se réfère au viewport :
                  le `backdrop-blur` de la colonne créerait sinon un bloc conteneur
                  qui rognerait l'overlay. `/feed` se rend en `null` → fond visible. */}
              {children}

              {/* En-tête mobile (masqué ≥ lg). Rendu APRÈS les overlays, au niveau
                  racine et en `fixed z-[60]`, pour rester visible au-dessus d'eux
                  (sinon l'overlay `z-40` le recouvrirait) → header unique présent
                  sur toutes les pages. */}
              <MobileHeader />
            </div>
          </ExplorerFilterProvider>
        </MessagesProvider>
      </NotificationsProvider>
    </AuthPromptProvider>
  )
}
