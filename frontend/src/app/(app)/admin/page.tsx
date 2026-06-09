import { AdminView } from '@/components/admin/admin-view'

/**
 * Panneau d'administration (rôle administrator) : annuaire des comptes,
 * changement de rôle (auth-service) et bannissement/réactivation (auth + user).
 * La garde d'accès est faite côté client dans AdminView et côté back (403).
 */
export default function AdminPage() {
  return <AdminView />
}
