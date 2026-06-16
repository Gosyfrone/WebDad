import { Suspense } from 'react'

import { MessagesView } from '@/components/messages/messages-view'

// Messagerie privée chiffrée (E2EE) : DM, groupes et communautés via l'API
// Gateway. `MessagesView` est un Client Component (apiFetch/WebSocket/IndexedDB)
// ; le Suspense couvre `useSearchParams` (point d'entrée `?dm=<userId>`).
// En accès direct/refresh (cette vraie page), la hauteur est portée par le div
// wrapper ; en soft-nav, c'est le panneau FeedOverlay (clavier-aware) qui la porte.
export default function MessagesPage() {
  return (
    <Suspense fallback={null}>
      {/* `MessagesView` est en `flex-1` : la hauteur est portée par ce conteneur
          (`flex flex-col`). En accès direct/refresh (cette vraie page, flux
          normal) on la fixe à l'espace entre l'en-tête et la tab bar mobiles ;
          en navigation soft, c'est le panneau `FeedOverlay` (collé à la tab bar,
          clavier-aware) qui la porte. */}
      <div className="flex h-[calc(100dvh-7rem)] flex-col lg:h-screen">
        <MessagesView />
      </div>
    </Suspense>
  )
}
