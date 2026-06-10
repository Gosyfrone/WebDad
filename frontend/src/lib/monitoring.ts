/**
 * Client de monitoring infra (console admin).
 *
 * Le gateway agrège la santé de tous les microservices (`GET /admin/monitoring`,
 * garde admin) en sondant leur `/health` → statut, latence, uptime. Source
 * unique : on ne tape jamais les services en direct (règle « tout passe par la
 * gateway »).
 *
 * ⚠️ À usage CLIENT (`apiFetch` lit le token en localStorage).
 */

import { apiFetch } from '@/lib/auth-client'

/** Santé d'un composant (gateway ou service) à l'instant de la sonde. */
export interface ServiceHealth {
  name: string
  prefix?: string
  status: 'up' | 'down'
  latencyMs: number
  uptimeSeconds: number
  error?: string
}

/** Instantané renvoyé par le monitoring. */
export interface MonitoringSnapshot {
  generatedAt: string
  services: ServiceHealth[]
}

interface ApiServiceHealth {
  name: string
  prefix?: string
  status: string
  latency_ms: number
  uptime_seconds: number
  error?: string
}

/** Récupère l'état de santé agrégé de l'infrastructure (admin). */
export async function getMonitoring(): Promise<MonitoringSnapshot> {
  const res = await apiFetch('/admin/monitoring')
  const body = (await res.json().catch(() => null)) as
    | { data?: { generated_at?: string; services?: ApiServiceHealth[] }; error?: string }
    | null
  if (!res.ok) {
    throw new Error(body?.error ?? `Erreur ${res.status}`)
  }
  const data = body?.data ?? {}
  return {
    generatedAt: data.generated_at ?? '',
    services: (data.services ?? []).map((s) => ({
      name: s.name,
      prefix: s.prefix,
      status: s.status === 'up' ? 'up' : 'down',
      latencyMs: s.latency_ms ?? 0,
      uptimeSeconds: s.uptime_seconds ?? 0,
      error: s.error,
    })),
  }
}
