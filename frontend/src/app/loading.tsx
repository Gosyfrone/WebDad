import { DelayedRouteLoading } from '@/components/layout/delayed-route-loading'

/** UI de chargement globale (App Router : affichée pendant le streaming des routes). */
export default function Loading() {
  return <DelayedRouteLoading />
}
