import { Suspense } from 'react'

import { MessagesView } from '@/components/messages/messages-view'

// Messagerie privée chiffrée (E2EE) : DM, groupes et communautés via l'API
// Gateway. `MessagesView` est un Client Component (apiFetch/WebSocket/IndexedDB)
// ; le Suspense couvre `useSearchParams` (point d'entrée `?dm=<userId>`).
export default function MessagesPage() {
  return (
    <Suspense fallback={null}>
      <MessagesView />
    </Suspense>
  )
}
