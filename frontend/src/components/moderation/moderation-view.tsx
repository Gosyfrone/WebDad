'use client'

import { useState } from 'react'
import { ShieldAlert } from 'lucide-react'

import { useSession } from '@/lib/session'
import { useT } from '@/components/language-provider'
import { cn } from '@/lib/utils'
import { AccountsPanel } from '@/components/moderation/accounts-panel'
import { DeletedPosts } from '@/components/moderation/deleted-posts'

type ModerationTab = 'posts' | 'accounts'

/**
 * Centre de modération, partagé par les modérateurs ET les administrateurs
 * (« faire régner l'ordre sur l'appli »). Deux onglets :
 *   - Tweets supprimés : corbeille des posts retirés (restaurer / purger) ;
 *   - Comptes : annuaire + bannir/réactiver (et, pour un admin uniquement,
 *     changement de rôle + suppression définitive de compte).
 *
 * L'administration « infra » (monitoring Docker, uptime…) vit ailleurs, sur la
 * page /admin réservée aux admins. Garde d'accès ici : mod ou admin (le back
 * renvoie 403 de toute façon).
 */
export function ModerationView() {
  const t = useT()
  const session = useSession()
  const [tab, setTab] = useState<ModerationTab>('posts')

  const isModerator = session?.role === 'moderator' || session?.role === 'administrator'
  const isAdmin = session?.role === 'administrator'

  // Session lue au montage : `null` tant qu'on ne sait pas → on n'affiche le
  // refus que lorsqu'on a une session confirmée non habilitée.
  if (session && !isModerator) {
    return (
      <div className="flex flex-col items-center justify-center gap-3 px-4 py-20 text-center">
        <ShieldAlert className="h-10 w-10 text-muted-foreground" aria-hidden />
        <h1 className="text-xl font-bold">{t('moderation.access_denied')}</h1>
        <p className="max-w-sm text-sm text-muted-foreground">{t('moderation.access_denied_desc')}</p>
      </div>
    )
  }

  const tabs: { key: ModerationTab; label: string }[] = [
    { key: 'posts', label: t('moderation.tab_posts') },
    { key: 'accounts', label: t('moderation.tab_accounts') },
  ]

  return (
    <div className="flex flex-col">
      <header className="panel z-10 border-b px-4 py-3 lg:sticky lg:top-0">
        <h1 className="brand-text text-xl font-bold">{t('nav.moderation')}</h1>
        <p className="text-sm text-muted-foreground">{t('moderation.subtitle')}</p>
      </header>

      <div className="flex border-b" role="tablist">
        {tabs.map((item) => (
          <button
            key={item.key}
            type="button"
            role="tab"
            aria-selected={tab === item.key}
            onClick={() => setTab(item.key)}
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

      {tab === 'posts' ? <DeletedPosts /> : <AccountsPanel canGovern={isAdmin} />}
    </div>
  )
}
