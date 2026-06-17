import { FeedOverlay } from '@/components/feed/feed-overlay'
import { ModerationView } from '@/components/moderation/moderation-view'

/**
 * Centre de modération (rôles moderator + administrator) : corbeille des tweets
 * supprimés (restaurer / purger) et annuaire des comptes (bannir/réactiver ;
 * rôles + suppression de compte réservés aux admins). Garde côté client
 * (ModerationView) + back (403). L'« administration infra » vit sur /admin.
 * Rendu en overlay au-dessus du feed persistant du layout.
 */
export default function ModerationPage() {
  return (
    <FeedOverlay headerOffset={false}>
      <ModerationView />
    </FeedOverlay>
  )
}
