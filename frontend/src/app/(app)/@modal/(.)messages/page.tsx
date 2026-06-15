import { Suspense } from 'react'

import { FeedOverlay } from '@/components/feed/feed-overlay'
import { MessagesView } from '@/components/messages/messages-view'

// Intercepting route : Messagerie en overlay au-dessus du feed (gardé monté).
// Suspense pour `useSearchParams` (`?dm=<userId>`), comme la vraie page.
// Accès direct/refresh → vraie page `(app)/messages/page.tsx`.
export default function InterceptedMessagesPage() {
  return (
    <FeedOverlay wide>
      <Suspense fallback={null}>
        <MessagesView />
      </Suspense>
    </FeedOverlay>
  )
}
