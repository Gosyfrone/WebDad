'use client'

import { useEffect, useState } from 'react'
import { Loader2, Search, UserX } from 'lucide-react'

import { searchUsers } from '@/lib/api'
import { useFollow } from '@/lib/use-follow'
import type { RelationUser } from '@/types'
import { useT } from '@/components/language-provider'
import { UserListItem } from '@/components/profil/user-list-item'

/**
 * Recherche de comptes (Explorer). Saisie debouncée (~300ms) : recherche par
 * nom affiché, ou par identifiant si la requête commence par « @ ». Les
 * résultats (alimentés user-service + profil-service) sont rendus via
 * `UserListItem` ; l'utilisateur courant est exclu de la liste.
 */
export function ExplorerView() {
  const t = useT()
  const [query, setQuery] = useState('')
  const [debounced, setDebounced] = useState('')
  const [results, setResults] = useState<RelationUser[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const { currentUserId, isFollowing, isPending, toggle } = useFollow()

  // Debounce de la saisie.
  useEffect(() => {
    const timer = setTimeout(() => setDebounced(query.trim()), 300)
    return () => clearTimeout(timer)
  }, [query])

  // Recherche sur la valeur debouncée.
  useEffect(() => {
    if (!debounced) {
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
          {t('explorer.hint_before')} <span className="font-bold">@</span>{' '}
          {t('explorer.hint_after')}
        </p>
      </div>

      {/* Résultats */}
      {!debounced ? (
        <EmptyState
          icon={<Search className="h-10 w-10 text-[#5B6CFF] dark:text-[#9aa6ff]" aria-hidden />}
          title={t('explorer.empty_title')}
          message={t('explorer.empty_msg')}
        />
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
              onToggleFollow={toggle}
            />
          ))}
        </div>
      )}
    </div>
  )
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
