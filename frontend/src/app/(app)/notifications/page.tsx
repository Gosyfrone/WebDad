import { FeedOverlay } from '@/components/feed/feed-overlay'
import { NotificationsView } from '@/components/notifications/notifications-view'

// Flux de notifications temps réel (likes, commentaires, réponses, mentions),
// agrégées côté serveur. État et WebSocket gérés par <NotificationsProvider>
// (monté dans le layout de l'espace authentifié). Rendu en overlay au-dessus
// du feed persistant du layout.
export default function NotificationsPage() {
  return (
    <FeedOverlay>
      <NotificationsView />
    </FeedOverlay>
  )
}
