'use client'

import { useEffect, useState } from 'react'
import { ShieldAlert } from 'lucide-react'

import { useCurrentUser } from '@/components/current-user-provider'
import { useT } from '@/components/language-provider'
import { cn } from '@/lib/utils'
import { AdminInfra } from '@/components/admin/admin-infra'
import { CreateAccountDialog } from '@/components/admin/create-account-dialog'
import { ModerationSettings } from '@/components/admin/moderation-settings'
import { TicketsPanel } from '@/components/moderation/tickets-panel'

type AdminTab = 'reports' | 'settings' | 'infra'

const TAB_STORAGE_KEY = 'breezy-admin-tab'
const ADMIN_TABS: AdminTab[] = ['reports', 'settings', 'infra']

/**
 * Espace d'administration (rôle administrator). Deux onglets :
 *   - Signalements (bugs) : tickets de RAPPORTS DE BUG, distincts des
 *     signalements de modération ; un admin peut TRANSFÉRER un ticket vers la
 *     modération (catégorie bug → moderation) ;
 *   - Infrastructure : monitoring des microservices (uptime/latence).
 *
 * Garde d'accès admin ici (le back renvoie 403 de toute façon).
 */
export function AdminView() {
  const t = useT()
  const { session, isAdmin } = useCurrentUser()
  const [tab, setTab] = useState<AdminTab>('reports')

  // Onglet persistant entre rafraîchissements (localStorage, lu après montage).
  useEffect(() => {
    const saved = localStorage.getItem(TAB_STORAGE_KEY) as AdminTab | null
    if (saved && ADMIN_TABS.includes(saved)) setTab(saved)
  }, [])

  function selectTab(key: AdminTab) {
    setTab(key)
    localStorage.setItem(TAB_STORAGE_KEY, key)
  }

  if (session && !isAdmin) {
    return (
      <div className="flex flex-col items-center justify-center gap-3 px-4 py-20 text-center">
        <ShieldAlert className="h-10 w-10 text-muted-foreground" aria-hidden />
        <h1 className="text-xl font-bold">{t('admin.access_denied')}</h1>
        <p className="max-w-sm text-sm text-muted-foreground">{t('admin.access_denied_desc')}</p>
      </div>
    )
  }

  const tabs: { key: AdminTab; label: string }[] = [
    { key: 'reports', label: t('admin.tab_reports') },
    { key: 'settings', label: t('admin.tab_settings') },
    { key: 'infra', label: t('admin.tab_infra') },
  ]

  return (
    <div className="flex flex-col">
      <header className="panel z-10 flex items-start justify-between gap-3 border-b px-4 py-3 lg:sticky lg:top-0">
        <div>
          <h1 className="brand-text text-xl font-bold">{t('nav.admin')}</h1>
          <p className="text-sm text-muted-foreground">{t('admin.infra_subtitle')}</p>
        </div>
        <CreateAccountDialog />
      </header>

      <div className="flex border-b" role="tablist">
        {tabs.map((item) => (
          <button
            key={item.key}
            type="button"
            role="tab"
            aria-selected={tab === item.key}
            onClick={() => selectTab(item.key)}
            className={cn(
              'flex-1 px-4 py-3 text-sm font-semibold transition-colors',
              tab === item.key
                ? 'border-b-2 border-[#5B6CFF] text-foreground'
                : 'text-muted-foreground hover:text-foreground',
            )}
          >
            {item.label}
          </button>
        ))}
      </div>

      {tab === 'reports' ? (
        <TicketsPanel category="bug" canTransfer />
      ) : tab === 'settings' ? (
        <ModerationSettings />
      ) : (
        <AdminInfra embedded />
      )}
    </div>
  )
}
