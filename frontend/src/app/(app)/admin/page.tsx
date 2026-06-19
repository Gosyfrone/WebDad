import { AdminView } from '@/components/admin/admin-view'
import { FeedOverlay } from '@/components/feed/feed-overlay'

/**
 * Espace d'administration (rôle administrator). Onglets : « Signalements (bugs) »
 * (réception des rapports de bug, transfert possible vers la modération) et
 * « Infrastructure » (monitoring des microservices). La gouvernance des comptes
 * (annuaire, bannissement, rôles) vit dans le centre de Modération, partagé avec
 * les modérateurs. Garde côté client (AdminView) + back.
 * Rendu en overlay au-dessus du feed persistant du layout.
 */
export default function AdminPage() {
  return (
    <FeedOverlay>
      <AdminView />
    </FeedOverlay>
  )
}
