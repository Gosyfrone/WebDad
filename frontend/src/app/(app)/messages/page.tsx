import { Suspense } from 'react'

import { FeedOverlay } from '@/components/feed/feed-overlay'
import { MessagesView } from '@/components/messages/messages-view'

// Messagerie privée chiffrée (E2EE) : DM, groupes et communautés via l'API
// Gateway. `MessagesView` est un Client Component (apiFetch/WebSocket/IndexedDB)
// ; le Suspense couvre `useSearchParams` (point d'entrée `?dm=<userId>`).
// Rendue en overlay (`wide` : occupe aussi la zone de la sidebar droite) au-dessus
// du feed persistant du layout.
export default function MessagesPage() {
  return (
    <FeedOverlay wide>
      <Suspense fallback={null}>
        <MessagesView />
      </Suspense>
    </FeedOverlay>
  )
}
