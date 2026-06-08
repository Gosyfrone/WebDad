'use client'

import { usePathname } from 'next/navigation'
import { Search } from 'lucide-react'

import { ROUTES } from '@/lib/routes'
import { useT } from '@/components/language-provider'
import { WhoToFollow } from '@/components/layout/who-to-follow'
import { LegalLinks } from '@/components/legal/legal-links'

/**
 * Tendances décoratives (placeholder, pas de back). Les libellés viennent du
 * dictionnaire i18n ; les hashtags restent tels quels (identifiants de marque).
 */
const TRENDS = [
  { categoryKey: 'trends.t1.category', topic: '#Microservices', postsKey: 'trends.t1.posts' },
  { categoryKey: 'trends.t2.category', topic: '#NextJS', postsKey: 'trends.t2.posts' },
  { categoryKey: 'trends.t3.category', topic: '#Docker', postsKey: 'trends.t3.posts' },
]

export function SidebarRight() {
  const t = useT()
  const pathname = usePathname()

  // La messagerie occupe toute la largeur (chat à deux volets) : pas de colonne
  // « Qui suivre » sur /messages.
  if (pathname?.startsWith(ROUTES.messages)) return null

  return (
    <aside className="sticky top-0 hidden h-screen w-[350px] flex-col gap-4 overflow-y-auto px-4 py-4 xl:flex">
      {/* Search */}
      <div className="relative">
        <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" aria-hidden />
        <input
          type="search"
          placeholder={t('search.placeholder')}
          disabled
          className="glass w-full rounded-full border py-2.5 pl-10 pr-4 text-sm backdrop-blur placeholder:text-muted-foreground focus:border-[#5B6CFF] focus:bg-white focus:outline-none disabled:cursor-not-allowed dark:focus:bg-white/10"
        />
      </div>

      {/* Tendances */}
      <div className="glass overflow-hidden rounded-[24px] border backdrop-blur-xl">
        <h2 className="brand-text px-4 py-3 text-xl font-bold">{t('trends.title')}</h2>
        {TRENDS.map((trend) => (
          <div
            key={trend.topic}
            className="flex cursor-not-allowed flex-col gap-0.5 px-4 py-3 transition-colors hover:bg-accent"
          >
            <span className="text-xs text-muted-foreground">
              {t(trend.categoryKey)} · {t('trends.trending')}
            </span>
            <span className="font-bold text-foreground">{trend.topic}</span>
            <span className="text-xs text-muted-foreground">{t(trend.postsKey)}</span>
          </div>
        ))}
      </div>

      {/* Qui suivre */}
      <WhoToFollow />

      {/* Liens légaux, sous les suggestions */}
      <LegalLinks className="px-4 pb-2" />
    </aside>
  )
}
