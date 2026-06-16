import { AdminInfra } from '@/components/admin/admin-infra'
import { FeedOverlay } from '@/components/feed/feed-overlay'

/**
 * Administration « infra » (rôle administrator) : monitoring des conteneurs
 * Docker / uptime des services (à venir, Stage ②). La gouvernance des comptes
 * (annuaire, bannissement, rôles) vit désormais dans le centre de Modération,
 * partagé avec les modérateurs. Garde côté client (AdminInfra) + back.
 * Rendu en overlay au-dessus du feed persistant du layout.
 */
export default function AdminPage() {
  return (
    <FeedOverlay>
      <AdminInfra />
    </FeedOverlay>
  )
}
