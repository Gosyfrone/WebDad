import { ExplorerView } from '@/components/explorer/explorer-view'
import { FeedOverlay } from '@/components/feed/feed-overlay'

// Recherche de comptes (user-service + profil-service via l'API Gateway).
// La recherche de posts arrivera quand l'API post sera branchée.
// Rendu en overlay au-dessus du feed persistant du layout.
export default function ExplorerPage() {
  return (
    <FeedOverlay>
      <ExplorerView />
    </FeedOverlay>
  )
}
