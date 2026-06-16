import { FeedOverlay } from '@/components/feed/feed-overlay'
import { OverlayWhenPath } from '@/components/feed/overlay-when-path'
import { NotificationsView } from '@/components/notifications/notifications-view'

// Intercepting route : Notifications en overlay au-dessus du feed (gardé monté).
// L'état/WebSocket vit dans <NotificationsProvider> (layout) → préservé.
// Accès direct/refresh → vraie page `(app)/notifications/page.tsx`.
export default function InterceptedNotificationsPage() {
  return (
    <OverlayWhenPath path="/notifications">
      <FeedOverlay>
        <NotificationsView />
      </FeedOverlay>
    </OverlayWhenPath>
  )
}
