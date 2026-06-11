'use client'

import { useEffect, useState } from 'react'
import Link from 'next/link'
import { usePathname } from 'next/navigation'
import { Search } from 'lucide-react'

import { listHashtagTrends, type HashtagTrend } from '@/lib/posts'
import { ROUTES } from '@/lib/routes'
import { useAuthGate } from '@/components/auth-prompt-provider'
import { useT } from '@/components/language-provider'
import { WhoToFollow } from '@/components/layout/who-to-follow'
import { LegalLinks } from '@/components/legal/legal-links'

export function SidebarRight() {
  const t = useT()
  const pathname = usePathname()
  const { isVisitor } = useAuthGate()
  const [trends, setTrends] = useState<HashtagTrend[]>([])

  useEffect(() => {
    let cancelled = false
    listHashtagTrends(5)
      .then((list) => {
        if (!cancelled) setTrends(list)
      })
      .catch(() => {
        if (!cancelled) setTrends([])
      })
    return () => {
      cancelled = true
    }
  }, [pathname])

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
        {trends.length > 0 ? (
          trends.map((trend) => (
            <Link
              key={trend.tag}
              href={`${ROUTES.feed}?hashtag=${encodeURIComponent(trend.tag)}`}
              className="flex flex-col gap-0.5 px-4 py-3 transition-colors hover:bg-accent"
            >
              <span className="text-xs text-muted-foreground">{t('trends.trending')}</span>
              <span className="font-bold text-foreground">#{trend.tag}</span>
              <span className="text-xs text-muted-foreground">
                {t(trend.count > 1 ? 'trends.posts_other' : 'trends.posts_one', {
                  count: formatTrendCount(trend.count),
                })}
              </span>
            </Link>
          ))
        ) : (
          <p className="px-4 pb-4 text-sm text-muted-foreground">{t('trends.empty')}</p>
        )}
      </div>

      {/* Qui suivre (réservé aux membres : appels au graphe social authentifiés). */}
      {!isVisitor && <WhoToFollow />}

      {/* Liens légaux, sous les suggestions */}
      <LegalLinks className="px-4 pb-2" />
    </aside>
  )
}

function formatTrendCount(count: number): string {
  if (count >= 1_000_000) return `${(count / 1_000_000).toFixed(1)}M`
  if (count >= 1_000) return `${(count / 1_000).toFixed(1)}K`
  return String(count)
}
