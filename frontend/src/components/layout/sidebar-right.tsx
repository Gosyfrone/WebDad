'use client'

import { useEffect, useState, type FormEvent } from 'react'
import Link from 'next/link'
import { usePathname, useRouter } from 'next/navigation'
import { useSearchParams } from 'next/navigation'
import { Search } from 'lucide-react'

import { currentUserId, listHashtagTrends, type HashtagTrend } from '@/lib/posts'
import { ROUTES, hashtagHref, searchHref } from '@/lib/routes'
import { useAuthGate } from '@/components/auth-prompt-provider'
import {
  addSuggestionHistoryEntry,
  clearSuggestionHistory,
  readSuggestionHistory,
  removeSuggestionHistoryEntry,
  type SuggestionHistoryEntry,
} from '@/lib/search-suggestion-history'
import { useT } from '@/components/language-provider'
import { WhoToFollow } from '@/components/layout/who-to-follow'
import { LegalLinks } from '@/components/legal/legal-links'
import { SearchSuggestionsDropdown } from '@/components/search/search-suggestions-dropdown'
import { ExplorerFilterCard } from '@/components/explorer/explorer-filter-controls'
import { useExplorerFilters } from '@/components/explorer/explorer-filter-context'

export function SidebarRight() {
  const t = useT()
  const pathname = usePathname()
  const { isVisitor } = useAuthGate()
  const searchParams = useSearchParams()
  const router = useRouter()
  const [trends, setTrends] = useState<HashtagTrend[]>([])
  const [query, setQuery] = useState('')
  const [searchFocused, setSearchFocused] = useState(false)
  const [history, setHistory] = useState<SuggestionHistoryEntry[]>([])
  const historyOwnerId = currentUserId()
  const {
    showPublications,
    showUsers,
    submittedSearchActive,
    tryTogglePublications,
    tryToggleUsers,
  } = useExplorerFilters()
  const showExplorerFilters =
    pathname === ROUTES.explorer &&
    submittedSearchActive &&
    Boolean(searchParams.get('q')?.trim())

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

  useEffect(() => {
    setHistory(readSuggestionHistory(historyOwnerId))
  }, [historyOwnerId])

  // La messagerie occupe toute la largeur (chat à deux volets) : pas de colonne
  // « Qui suivre » sur /messages.
  if (pathname?.startsWith(ROUTES.messages)) return null

  function submitSearch(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const href = searchHref(query)
    if (href !== ROUTES.explorer) router.push(href)
  }

  function pickSearchSuggestion(
    href: string,
    entry?: Omit<SuggestionHistoryEntry, 'visitedAt'>,
  ) {
    if (entry) setHistory(addSuggestionHistoryEntry(historyOwnerId, entry))
    setSearchFocused(false)
    setQuery('')
    router.push(href)
  }

  function clearRecentSearches() {
    clearSuggestionHistory(historyOwnerId)
    setHistory([])
  }

  function removeRecentSearch(entryId: string) {
    setHistory(removeSuggestionHistoryEntry(historyOwnerId, entryId))
  }

  return (
    <aside className="sticky top-0 hidden max-h-screen w-[min(350px,30vw)] min-w-[290px] flex-col gap-2 overflow-y-auto overscroll-contain px-3 py-2 pb-8 xl:flex 2xl:px-4">
      {/* Search */}
      <form onSubmit={submitSearch} className="relative">
        <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" aria-hidden />
        <input
          type="search"
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          onFocus={() => setSearchFocused(true)}
          onBlur={() => setSearchFocused(false)}
          placeholder={t('search.placeholder')}
          className="glass w-full rounded-full border py-2.5 pl-10 pr-4 text-sm backdrop-blur placeholder:text-muted-foreground focus:border-[#5B6CFF] focus:bg-white focus:outline-none dark:focus:bg-white/10"
        />
        <SearchSuggestionsDropdown
          query={query}
          open={searchFocused}
          history={history}
          onPick={pickSearchSuggestion}
          onClearHistory={clearRecentSearches}
          onRemoveHistory={removeRecentSearch}
        />
      </form>

      {showExplorerFilters && (
        <ExplorerFilterCard
          showPublications={showPublications}
          showUsers={showUsers}
          onTogglePublications={tryTogglePublications}
          onToggleUsers={tryToggleUsers}
        />
      )}

      {/* Tendances */}
      <div className="glass overflow-hidden rounded-[24px] border backdrop-blur-xl">
        <h2 className="brand-text px-4 pb-1.5 pt-2.5 text-lg font-bold leading-tight">{t('trends.title')}</h2>
        {trends.length > 0 ? (
          <div className="px-2 pb-2">
            {trends.map((trend) => (
              <Link
                key={trend.tag}
                href={hashtagHref(trend.tag, 'top')}
                className="grid grid-cols-[minmax(0,1fr)_auto] gap-x-2 rounded-[18px] px-3 py-2 transition-colors hover:bg-accent"
              >
                <span className="col-span-2 text-[11px] leading-3 text-muted-foreground">{t('trends.trending')}</span>
                <span className="min-w-0 truncate text-sm font-bold leading-5 text-foreground">#{trend.tag}</span>
                <span className="justify-self-end whitespace-nowrap pl-2 text-[11px] leading-5 text-muted-foreground">
                  {t(trend.count > 1 ? 'trends.posts_other' : 'trends.posts_one', {
                    count: formatTrendCount(trend.count),
                  })}
                </span>
              </Link>
            ))}
          </div>
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
