import { cn } from '@/lib/utils'

/**
 * Coquille d'overlay rendue **pile au-dessus de la colonne centrale (le feed)**,
 * utilisée par les intercepting routes de l'espace `(app)` (post, explorer,
 * notifications, messages, signets, profil). Le feed reste monté en arrière-plan
 * (scroll, posts chargés et WebSocket « a posté » préservés) ; on y revient via
 * la navigation de gauche (toujours visible, en `scroll={false}` pour ne pas
 * bouger le feed) ou le bouton retour des vues — les routes sans overlay (feed,
 * paramètres, modération, admin) sont interceptées vers `null` pour vider ce
 * slot et révéler le feed au premier plan, à la position laissée.
 *
 * Reproduit la grille du layout `(app)` (rangée centrée `max-w-[1265px]` +
 * espaceurs aux largeurs exactes des sidebars) : le panneau opaque recouvre
 * exactement le feed tout en laissant les sidebars visibles et cliquables
 * (`pointer-events-none` sur l'enveloppe, `auto` sur le seul panneau).
 *
 * `wide` : pour les vues qui occupent aussi la zone de la sidebar droite (ex.
 * Messages, où `SidebarRight` se masque) → pas d'espaceur droit et c'est la vue
 * qui gère sa propre hauteur/défilement (pas de scroll ni `pb` imposés).
 */
export function FeedOverlay({
  children,
  wide = false,
}: {
  children: React.ReactNode
  wide?: boolean
}) {
  return (
    <div className="pointer-events-none fixed inset-0 z-40 flex justify-center">
      <div className="flex w-full max-w-[1265px]">
        {/* Espaceur = SidebarLeft (w-[275px], visible ≥ lg) */}
        <div className="hidden w-[275px] shrink-0 lg:block" />

        {/* Panneau central, opaque, aligné sur la colonne du feed */}
        <div
          className={cn(
            'bg-page pointer-events-auto flex min-w-0 flex-1 flex-col lg:border-x',
            !wide && 'overflow-y-auto pb-16 lg:pb-0',
          )}
        >
          {children}
        </div>

        {/* Espaceur = SidebarRight (masqué en mode large) */}
        {!wide && (
          <div className="hidden w-[min(350px,30vw)] min-w-[290px] shrink-0 xl:block" />
        )}
      </div>
    </div>
  )
}
