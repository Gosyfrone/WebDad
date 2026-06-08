import { NotificationsView } from '@/components/notifications/notifications-view'

// Flux de notifications temps réel (likes, commentaires, réponses, mentions),
// agrégées côté serveur. État et WebSocket gérés par <NotificationsProvider>
// (monté dans le layout de l'espace authentifié).
export default function NotificationsPage() {
  return <NotificationsView />
}
