import { SidebarLeft } from '@/components/layout/sidebar-left'
import { SidebarRight } from '@/components/layout/sidebar-right'
import type { UserRole } from '@/types'

/**
 * Layout de l'espace authentifié : 3 colonnes style X.com.
 *   - Gauche  : navigation (sidebar)
 *   - Centre  : contenu de la page
 *   - Droite  : suggestions / tendances
 *
 * TODO (issue auth) : récupérer la session (JWT) côté serveur, en déduire
 * `role` / `username`, et rediriger vers /login si l'utilisateur n'est pas connecté.
 */
const PLACEHOLDER_ROLE: UserRole = 'administrator'

export default function AppLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-screen justify-center">
      <div className="flex w-full max-w-[1265px]">
        <SidebarLeft role={PLACEHOLDER_ROLE} />
        <main className="min-h-screen flex-1 border-x">{children}</main>
        <SidebarRight />
      </div>
    </div>
  )
}
