import { AdminInfra } from '@/components/admin/admin-infra'

/**
 * Administration « infra » (rôle administrator) : monitoring des conteneurs
 * Docker / uptime des services (à venir, Stage ②). La gouvernance des comptes
 * (annuaire, bannissement, rôles) vit désormais dans le centre de Modération,
 * partagé avec les modérateurs. Garde côté client (AdminInfra) + back.
 */
export default function AdminPage() {
  return <AdminInfra />
}
