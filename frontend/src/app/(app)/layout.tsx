import { SidebarLeft } from '@/components/layout/sidebar-left'
import { SidebarRight } from '@/components/layout/sidebar-right'
import { MobileHeader } from '@/components/layout/mobile-header'
import { MobileTabBar } from '@/components/layout/mobile-tab-bar'
import { ComposeFab } from '@/components/layout/compose-fab'
import type { UserRole } from '@/types'

/**
 * Layout de l'espace authentifié, responsive (mobile-first).
 *
 *   - < lg (téléphones, iPad portrait) : en-tête mobile (avatar + logo) +
 *     barre d'onglets fixe en bas + bouton « + » flottant. Sidebars masquées.
 *   - ≥ lg (iPad paysage, desktop) : colonne de navigation à gauche + contenu.
 *   - ≥ xl : ajout de la colonne de droite (suggestions / tendances).
 *
 * TODO (issue auth) : récupérer la session (JWT) côté serveur, en déduire
 * `role` / `username`, et rediriger vers /login si l'utilisateur n'est pas connecté.
 */
const PLACEHOLDER_ROLE: UserRole = 'administrator'

export default function AppLayout({ children }: { children: React.ReactNode }) {
  return (
    <div
      className="relative flex min-h-screen justify-center overflow-x-clip"
      style={{
        background:
          'linear-gradient(140deg, #f8f3ff 0%, #eadcff 28%, #d9c6ff 62%, #ebe8ff 100%)',
      }}
    >
      <div className="pointer-events-none fixed inset-0 bg-[linear-gradient(115deg,rgba(141,61,255,0.18)_0%,rgba(255,255,255,0.28)_34%,rgba(71,217,255,0.16)_100%)]" />
      <div className="pointer-events-none fixed inset-0 bg-[radial-gradient(circle_at_15%_15%,rgba(255,255,255,0.48),transparent_30%),radial-gradient(circle_at_82%_28%,rgba(141,61,255,0.16),transparent_32%),radial-gradient(circle_at_48%_92%,rgba(71,217,255,0.14),transparent_34%)]" />

      <div className="relative flex w-full max-w-[1265px]">
        <SidebarLeft role={PLACEHOLDER_ROLE} />

        {/* Colonne centrale. overflow-x-clip : empêche tout défilement horizontal
            parasite sur mobile (sans créer de conteneur de scroll, donc sans
            casser les en-têtes sticky, contrairement à overflow-x-hidden). */}
        <div className="flex min-h-screen w-full min-w-0 flex-1 flex-col overflow-x-clip border-white/45 bg-white/72 shadow-[0_30px_90px_rgba(91,108,255,0.12)] backdrop-blur-2xl lg:border-x">
          <MobileHeader role={PLACEHOLDER_ROLE} />
          {/* pb-16 : dégage la barre d'onglets fixe (masquée ≥ lg) */}
          <main className="flex-1 pb-16 lg:pb-0">{children}</main>
        </div>

        <SidebarRight />
      </div>

      {/* Chrome mobile (masqué ≥ lg) */}
      <MobileTabBar />
      <ComposeFab />
    </div>
  )
}
