import { Suspense } from 'react'

import { FeedOverlay } from '@/components/feed/feed-overlay'
import { OverlayWhenPath } from '@/components/feed/overlay-when-path'
import { MessagesView } from '@/components/messages/messages-view'

// Intercepting route : Messagerie en overlay au-dessus du feed (gardé monté).
// Suspense pour `useSearchParams` (`?dm=<userId>`), comme la vraie page.
// Accès direct/refresh → vraie page `(app)/messages/page.tsx`.
// `OverlayWhenPath` referme l'overlay (et la PassphraseGate) dès qu'on quitte
// /messages, sans dépendre de la réinitialisation du slot parallèle.
export default function InterceptedMessagesPage() {
  return (
    <OverlayWhenPath path="/messages">
      <FeedOverlay wide>
        <Suspense fallback={null}>
          <MessagesView />
        </Suspense>
      </FeedOverlay>
    </OverlayWhenPath>
  )
}
