'use client'

import { useCallback, useEffect, useState } from 'react'
import { Activity, Loader2, ShieldAlert } from 'lucide-react'

import { getMonitoring, type MonitoringSnapshot } from '@/lib/monitoring'
import { useSession } from '@/lib/session'
import { useLanguage } from '@/components/language-provider'
import { CreateAccountDialog } from '@/components/admin/create-account-dialog'
import { timeAgo } from '@/lib/utils'

/** Intervalle de rafraîchissement du tableau de bord (ms). */
const REFRESH_MS = 5000

/**
 * Administration « infra » — réservée aux administrateurs (« la main sur
 * l'infra »). Tableau de bord de santé des microservices (statut / latence /
 * uptime), sondé par le gateway (`GET /admin/monitoring`) et rafraîchi en
 * continu. La gouvernance des comptes (annuaire, ban, rôles) vit dans le centre
 * de Modération, partagé avec les modérateurs.
 *
 * Perspective : métriques conteneur (CPU/mém/restarts) via l'API Docker.
 */
export function AdminInfra() {
  const { t, locale } = useLanguage()
  const session = useSession()
  const isAdmin = session?.role === 'administrator'

  const [snapshot, setSnapshot] = useState<MonitoringSnapshot | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(false)

  const refresh = useCallback(async () => {
    try {
      setSnapshot(await getMonitoring())
      setError(false)
    } catch {
      setError(true)
    } finally {
      setLoading(false)
    }
  }, [])

  // Sondage initial + rafraîchissement périodique tant qu'on est admin et monté.
  useEffect(() => {
    if (!isAdmin) return
    void refresh()
    const handle = setInterval(() => void refresh(), REFRESH_MS)
    return () => clearInterval(handle)
  }, [isAdmin, refresh])

  if (session && !isAdmin) {
    return (
      <div className="flex flex-col items-center justify-center gap-3 px-4 py-20 text-center">
        <ShieldAlert className="h-10 w-10 text-muted-foreground" aria-hidden />
        <h1 className="text-xl font-bold">{t('admin.access_denied')}</h1>
        <p className="max-w-sm text-sm text-muted-foreground">{t('admin.access_denied_desc')}</p>
      </div>
    )
  }

  return (
    <div className="flex flex-col">
      {/* En-tête interne (desktop) : sur mobile, le titre est porté par l'en-tête
          global type-feed → on le masque ici (le bouton « Créer un compte »
          descend alors dans le body, cf. plus bas). */}
      <header className="panel z-10 hidden items-start justify-between gap-3 border-b px-4 py-3 lg:sticky lg:top-0 lg:flex">
        <div>
          <h1 className="brand-text text-xl font-bold">{t('nav.admin')}</h1>
          <p className="text-sm text-muted-foreground">{t('admin.infra_subtitle')}</p>
        </div>
        {/* Création de compte de force — réservée aux administrateurs. */}
        <CreateAccountDialog />
      </header>

      {/* Action « Créer un compte » (mobile) : en haut du body, au-dessus de la
          liste des services, sans bande dédiée (l'en-tête global porte le titre). */}
      <div className="px-4 pt-3 lg:hidden">
        <CreateAccountDialog />
      </div>

      {loading ? (
        <div className="flex items-center justify-center py-16 text-muted-foreground">
          <Loader2 className="h-5 w-5 animate-spin" aria-hidden />
        </div>
      ) : error ? (
        <p className="py-16 text-center text-sm text-muted-foreground">{t('admin.monitoring_error')}</p>
      ) : (
        <div className="px-4 py-4">
          <div className="mb-3 flex items-center gap-2 text-sm text-muted-foreground">
            <Activity className="h-4 w-4 text-[#5B6CFF] dark:text-[#9aa6ff]" aria-hidden />
            <span>{t('admin.infra_heading')}</span>
            {snapshot?.generatedAt && (
              <span className="ml-auto text-xs">
                {t('admin.monitoring_updated', { when: timeAgo(snapshot.generatedAt, locale) })}
              </span>
            )}
          </div>

          <ul className="grid gap-2 sm:grid-cols-2">
            {(snapshot?.services ?? []).map((svc) => {
              const up = svc.status === 'up'
              return (
                <li
                  key={svc.name}
                  className="panel flex items-center gap-3 rounded-2xl border p-3 shadow-sm"
                >
                  <span
                    className={`h-2.5 w-2.5 shrink-0 rounded-full ${up ? 'bg-emerald-500' : 'bg-red-500'}`}
                    aria-hidden
                  />
                  <div className="flex min-w-0 flex-1 flex-col">
                    <span className="truncate text-sm font-bold">{svc.name}</span>
                    <span className="truncate text-xs text-muted-foreground">
                      {up
                        ? `${t('admin.status_up')} · ${t('admin.latency')} ${svc.latencyMs} ms · ${t('admin.uptime')} ${formatUptime(svc.uptimeSeconds, t)}`
                        : `${t('admin.status_down')}${svc.error ? ` · ${svc.error}` : ''}`}
                    </span>
                  </div>
                </li>
              )
            })}
          </ul>
        </div>
      )}
    </div>
  )
}

/**
 * Met en forme un uptime (secondes) en chaîne compacte localisée, au plus 2
 * unités significatives (ex. « 2j 3h », « 5min 12s »). `—` si nul/inconnu.
 */
function formatUptime(totalSeconds: number, t: (key: string) => string): string {
  if (totalSeconds <= 0) return '—'
  const d = Math.floor(totalSeconds / 86400)
  const h = Math.floor((totalSeconds % 86400) / 3600)
  const m = Math.floor((totalSeconds % 3600) / 60)
  const s = totalSeconds % 60

  const parts: string[] = []
  if (d) parts.push(`${d}${t('admin.uptime_d')}`)
  if (h) parts.push(`${h}${t('admin.uptime_h')}`)
  if (!d && m) parts.push(`${m}${t('admin.uptime_m')}`)
  if (!d && !h && s) parts.push(`${s}${t('admin.uptime_s')}`)
  return parts.slice(0, 2).join(' ') || `${s}${t('admin.uptime_s')}`
}
