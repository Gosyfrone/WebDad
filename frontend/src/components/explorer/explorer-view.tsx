'use client'

import { useEffect, useState } from 'react'
import Link from 'next/link'
import { useSearchParams } from 'next/navigation'
import { Clock3, Hash, Loader2, Search, Trash2, UserX, X } from 'lucide-react'

import { searchUsers } from '@/lib/api'
import { currentUserId as readCurrentUserId } from '@/lib/posts'
import { hashtagHref } from '@/lib/routes'
import {
  addSearchHistoryEntry,
  clearSearchHistory,
  readSearchHistory,
  removeSearchHistoryEntry,
  type SearchHistoryEntry,
} from '@/lib/search-history'
import { useFollow } from '@/lib/use-follow'
import type { RelationUser } from '@/types'
import { useT } from '@/components/language-provider'
import { UserListItem } from '@/components/profil/user-list-item'
import { Button } from '@/components/ui/button'

/**
 * Recherche Explorer. Saisie debouncée (~300ms) : recherche de comptes par nom
 * ou identifiant, et accès direct aux résultats d'un hashtag si la requête
 * commence par « # ».
 */
export function ExplorerView() {
  const t = useT()
  // Point d'entrée depuis une mention de non-membre (carte d'aperçu en
  // messagerie) : /explorer?q=@handle pré-remplit la recherche.
  const searchParams = useSearchParams()
  const [query, setQuery] = useState(() => searchParams.get('q') ?? '')
  const [debounced, setDebounced] = useState('')
  const [results, setResults] = useState<RelationUser[]>([])
  const [history, setHistory] = useState<SearchHistoryEntry[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const { currentUserId, isFollowing, isPending, toggle } = useFollow()
  const historyOwnerId = currentUserId ?? readCurrentUserId()

  // Debounce de la saisie.
  useEffect(() => {
    const timer = setTimeout(() => setDebounced(query.trim()), 300)
    return () => clearTimeout(timer)
  }, [query])

  // Historique des profils ouverts depuis Explorer, isolé par compte.
  useEffect(() => {
    setHistory(readSearchHistory(historyOwnerId))
  }, [historyOwnerId])

  // Recherche sur la valeur debouncée.
  useEffect(() => {
    if (!debounced) {
      setResults([])
      setError(null)
      setLoading(false)
      return
    }
    if (isHashtagQuery(debounced)) {
      setResults([])
      setError(null)
      setLoading(false)
      return
    }
    let cancelled = false
    setLoading(true)
    setError(null)
    searchUsers(debounced)
      .then((users) => {
        if (!cancelled) setResults(users)
      })
      .catch(() => {
        if (!cancelled) setError(t('explorer.search_failed_title'))
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [debounced, t])

  const visible = results.filter((u) => u.id !== currentUserId)
  const byHandle = debounced.startsWith('@')
  const hashtagQuery = isHashtagQuery(debounced) ? debounced.replace(/^#/, '') : ''
  const saveHistory = (user: RelationUser) => {
    setHistory(addSearchHistoryEntry(historyOwnerId, user))
  }
  const removeHistory = () => {
    clearSearchHistory(historyOwnerId)
    setHistory([])
  }
  const removeHistoryEntry = (entryId: string) => {
    setHistory(removeSearchHistoryEntry(historyOwnerId, entryId))
  }

  return (
    <div className="flex flex-col">
      {/* En-tête + champ de recherche */}
      <div className="panel z-10 border-b px-4 py-3 lg:sticky lg:top-0">
        <h1 className="brand-text mb-3 text-xl font-bold">{t('nav.explore')}</h1>
        <div className="relative">
          <Search
            className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground"
            aria-hidden
          />
          <input
            type="search"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder={t('explorer.search_placeholder')}
            autoFocus
            className="glass w-full rounded-full border py-2.5 pl-10 pr-4 text-sm backdrop-blur placeholder:text-muted-foreground focus:border-[#5B6CFF] focus:bg-white focus:outline-none dark:focus:bg-white/10"
          />
        </div>
        <p className="mt-2 px-1 text-xs text-muted-foreground">
          {t('explorer.hint_before')}{' '}
          <span className="font-bold">@</span> / <span className="font-bold">#</span>{' '}
          {t('explorer.hint_after')}
        </p>
      </div>

      {/* Résultats */}
      {!debounced ? (
        history.length > 0 ? (
          <div className="divide-y divide-border">
            <div className="flex items-center justify-between gap-3 px-4 py-3">
              <div className="flex min-w-0 items-center gap-2">
                <Clock3 className="h-4 w-4 shrink-0 text-[#5B6CFF] dark:text-[#9aa6ff]" />
                <h2 className="truncate text-sm font-bold text-foreground">
                  {t('explorer.history_title')}
                </h2>
              </div>
              <Button
                type="button"
                variant="ghost"
                size="icon"
                onClick={removeHistory}
                aria-label={t('explorer.history_clear')}
                title={t('explorer.history_clear')}
                className="h-8 w-8 shrink-0 rounded-full text-muted-foreground hover:text-destructive"
              >
                <Trash2 className="h-4 w-4" aria-hidden />
              </Button>
            </div>
            {history.map((user) => (
              <UserListItem
                key={user.id}
                user={user}
                isFollowing={isFollowing(user.id)}
                isSelf={user.id === currentUserId}
                pending={isPending(user.id)}
                onProfileOpen={saveHistory}
                trailingAction={
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    onClick={(e) => {
                      e.preventDefault()
                      e.stopPropagation()
                      removeHistoryEntry(user.id)
                    }}
                    aria-label={t('explorer.history_remove_one', {
                      name: user.displayName,
                    })}
                    title={t('explorer.history_remove_one', {
                      name: user.displayName,
                    })}
                    className="h-8 w-8 rounded-full text-muted-foreground hover:text-destructive"
                  >
                    <X className="h-4 w-4" aria-hidden />
                  </Button>
                }
                onToggleFollow={toggle}
              />
            ))}
          </div>
        ) : (
          <EmptyState
            icon={<Search className="h-10 w-10 text-[#5B6CFF] dark:text-[#9aa6ff]" aria-hidden />}
            title={t('explorer.empty_title')}
            message={t('explorer.empty_msg')}
          />
        )
      ) : hashtagQuery ? (
        <Link
          href={hashtagHref(hashtagQuery, 'top')}
          className="mx-3 my-3 flex items-center gap-3 rounded-[24px] border bg-background/70 px-4 py-3 transition-colors hover:bg-accent"
        >
          <span className="grid h-11 w-11 shrink-0 place-items-center rounded-full bg-[#5B6CFF]/10 text-[#5B6CFF] dark:bg-[#9aa6ff]/15 dark:text-[#9aa6ff]">
            <Hash className="h-5 w-5" aria-hidden />
          </span>
          <span className="flex min-w-0 flex-col">
            <span className="truncate font-bold text-foreground">#{hashtagQuery}</span>
            <span className="truncate text-sm text-muted-foreground">
              {t('explorer.hashtag_result')}
            </span>
          </span>
        </Link>
      ) : loading ? (
        <div className="flex justify-center py-16">
          <Loader2 className="h-6 w-6 animate-spin text-[#5B6CFF] dark:text-[#9aa6ff]" />
        </div>
      ) : error ? (
        <EmptyState
          icon={<UserX className="h-10 w-10 text-[#5B6CFF] dark:text-[#9aa6ff]" aria-hidden />}
          title={t('explorer.search_failed_title')}
          message={t('explorer.search_failed_msg')}
        />
      ) : visible.length === 0 ? (
        <EmptyState
          icon={<UserX className="h-10 w-10 text-[#5B6CFF] dark:text-[#9aa6ff]" aria-hidden />}
          title={t('explorer.no_results')}
          message={
            byHandle
              ? t('explorer.no_results_handle', { q: debounced })
              : t('explorer.no_results_name', { q: debounced })
          }
        />
      ) : (
        <div className="divide-y divide-border">
          {visible.map((user) => (
            <UserListItem
              key={user.id}
              user={user}
              isFollowing={isFollowing(user.id)}
              isSelf={user.id === currentUserId}
              pending={isPending(user.id)}
              onProfileOpen={saveHistory}
              onToggleFollow={toggle}
            />
          ))}
        </div>
      )}
    </div>
  )
}

function isHashtagQuery(value: string): boolean {
  return value.trim().startsWith('#') && value.trim().replace(/^#/, '').length > 0
}

function EmptyState({
  icon,
  title,
  message,
}: {
  icon: React.ReactNode
  title: string
  message: string
}) {
  return (
    <div className="glass mx-4 mt-6 flex flex-col items-center gap-2 rounded-[26px] border px-8 py-16 text-center backdrop-blur-xl">
      {icon}
      <h2 className="text-lg font-bold text-foreground">{title}</h2>
      <p className="max-w-sm text-sm text-muted-foreground">{message}</p>
    </div>
  )
}
