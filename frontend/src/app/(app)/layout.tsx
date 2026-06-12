import { SidebarLeft } from '@/components/layout/sidebar-left'
import { SidebarRight } from '@/components/layout/sidebar-right'
import { MobileHeader } from '@/components/layout/mobile-header'
import { MobileTabBar } from '@/components/layout/mobile-tab-bar'
import { ComposeFab } from '@/components/layout/compose-fab'
import { NotificationsProvider } from '@/components/notifications-provider'
import { MessagesProvider } from '@/components/messages-provider'
import { OnboardingGate } from '@/components/onboarding/onboarding-gate'

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
 */
export default function AppLayout({ children }: { children: React.ReactNode }) {
  return (
    <NotificationsProvider>
      <MessagesProvider>
        <div className="bg-page relative flex min-h-screen justify-center overflow-x-clip">
          <div className="bg-page-glow-1 pointer-events-none fixed inset-0" />
          <div className="bg-page-glow-2 pointer-events-none fixed inset-0" />

          <div className="relative flex w-full max-w-[1265px]">
            <SidebarLeft />

            {/* Colonne centrale. overflow-x-clip : empêche tout défilement horizontal
                parasite sur mobile (sans créer de conteneur de scroll, donc sans
                casser les en-têtes sticky, contrairement à overflow-x-hidden). */}
            <div className="glass-column flex min-h-screen w-full min-w-0 flex-1 flex-col overflow-x-clip backdrop-blur-2xl lg:border-x">
              <MobileHeader />
              {/* pb-16 : dégage la barre d'onglets fixe (masquée ≥ lg) */}
              <main className="flex-1 pb-16 lg:pb-0">{children}</main>
            </div>

            <SidebarRight />
          </div>

          {/* Chrome mobile (masqué ≥ lg) */}
          <MobileTabBar />
          <ComposeFab />

          {/* Onboarding bloquant pour les comptes OAuth sans profil (cf. composant). */}
          <OnboardingGate />
        </div>
      </MessagesProvider>
    </NotificationsProvider>
  )
}
