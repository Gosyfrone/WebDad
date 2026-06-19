'use client'

import { cn } from '@/lib/utils'
import { useKeyboardInset } from '@/hooks/use-keyboard-inset'
import { useMessages } from '@/components/messages-provider'

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
 *
 * `headerOffset` : réserve l'espace de l'en-tête mobile fixe (`pt-14`). À mettre
 * à `false` pour les vues qui n'affichent pas cet en-tête (ex. profil, qui a son
 * propre en-tête) afin d'éviter un vide de 56px en haut sur mobile.
 */
export function FeedOverlay({
  children,
  wide = false,
  headerOffset = true,
}: {
  children: React.ReactNode
  wide?: boolean
  headerOffset?: boolean
}) {
  // Clavier iOS (messagerie only) : remonte le composer au-dessus du clavier via
  // le padding bas du panneau. Sur iOS, `position:fixed`/`dvh` ignorent le
  // clavier (seul le visual viewport rétrécit) → sans ça le composer passe
  // derrière le clavier et un décalage reste coincé. `inset` = 0 quand le clavier
  // est fermé (et ~0 sur Android, qui redimensionne déjà le layout) → on retombe
  // sur `pb-14` (hauteur de la tab bar).
  const keyboardInset = useKeyboardInset(wide)

  // Vue large (messagerie) avec une conversation ouverte sur mobile : le
  // `MobileHeader` se masque (le ChatPane porte son propre en-tête) → on remonte
  // l'overlay à `top-0` au lieu de `top-14` pour ne pas laisser 56px de vide en
  // haut. Le bas reste inchangé (tab bar dégagée par `pb-14`).
  const { activeConversationId } = useMessages()
  const fillTop = wide && Boolean(activeConversationId)

  return (
    <div
      className={cn(
        'pointer-events-none fixed z-40 flex justify-center',
        // Vue « large » (messagerie) : sur mobile, le panneau utilise EXACTEMENT
        // le même ancrage bas que la tab bar — `bottom-0` — et non `bottom-14`.
        // Crucial sur iOS : `fixed; bottom:0` est calé sur le bas VISIBLE (comme
        // la tab bar), alors qu'un `bottom:Npx` se mesure depuis le bas du LAYOUT
        // viewport → les deux se désynchronisent quand la barre d'adresse bouge
        // (le décalage au swipe). En partageant `bottom-0`, panneau et tab bar
        // restent collés. Le composer est ensuite décalé de la hauteur de la tab
        // bar via le PADDING du panneau (cf. plus bas), pas via `bottom`.
        // ≥ lg : ni en-tête ni tab bar mobiles → plein écran (`inset-0`).
        // `fillTop` (conversation ouverte) : `top-0` pour combler le vide laissé
        // par le `MobileHeader` masqué.
        wide
          ? cn('inset-x-0 bottom-0 lg:inset-y-0', fillTop ? 'top-0' : 'top-14')
          : 'inset-0',
      )}
    >
      <div className="flex w-full max-w-[1265px]">
        {/* Espaceur = SidebarLeft (w-[275px], visible ≥ lg) */}
        <div className="hidden w-[275px] shrink-0 lg:block" />

        {/* Panneau central, opaque, aligné sur la colonne du feed.
            pt-14 : dégage l'en-tête mobile fixe (masqué ≥ lg) rendu par-dessus. */}
        <div
          className={cn(
            'bg-page pointer-events-auto flex min-w-0 flex-1 flex-col lg:border-x lg:pt-0',
            headerOffset && 'pt-14',
            !wide && 'overflow-y-auto pb-16 lg:pb-0',
            // Messagerie : padding bas = hauteur de la tab bar (h-14) → le
            // composer se pose pile au-dessus d'elle (le panneau, lui, descend
            // jusqu'à `bottom-0`). ≥ lg : pas de tab bar → pas de padding.
            wide && 'pb-14 lg:pb-0',
          )}
          // Clavier iOS ouvert : on remplace ce padding par la hauteur du clavier
          // → le composer remonte juste au-dessus du clavier. Fermé (ou Android,
          // ou desktop) : `inset` = 0 → on garde `pb-14`.
          style={wide && keyboardInset > 0 ? { paddingBottom: keyboardInset } : undefined}
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
