'use client'

import { useEffect, useState } from 'react'
import { Loader2, Search } from 'lucide-react'

import { searchUsers } from '@/lib/api'
import type { RelationUser } from '@/types'
import { initialOf } from '@/lib/utils'
import { useLanguage } from '@/components/language-provider'
import { ActivityPresenceDot } from '@/components/profil/activity-presence-dot'
import { CertificationBadge } from '@/components/profil/certification-badge'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'

interface UserSearchProps {
  /** Ids à exclure des résultats (soi-même, membres déjà présents…). */
  excludeIds?: string[]
  /** Appelé quand une personne est choisie dans les résultats. */
  onPick: (user: RelationUser) => void
  autoFocus?: boolean
  /**
   * Personnes prioritaires (ex. partages récents) : affichées quand le champ
   * est vide, et remontées en tête des résultats correspondants à la recherche.
   */
  recentUsers?: RelationUser[]
  /** Titre affiché au-dessus de la liste des récents (champ vide). */
  recentLabel?: string
}

/**
 * Champ de recherche d'utilisateurs (debounce ~300 ms) réutilisé par le nouveau
 * DM et l'ajout de membres. Réutilise `searchUsers` (user-service +
 * profil-service, bascule `@` = identifiant) ; chaque résultat est cliquable.
 */
export function UserSearch({
  excludeIds = [],
  onPick,
  autoFocus = true,
  recentUsers = [],
  recentLabel,
}: UserSearchProps) {
  const { t } = useLanguage()
  const [query, setQuery] = useState('')
  const [debounced, setDebounced] = useState('')
  const [results, setResults] = useState<RelationUser[]>([])
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    const timer = setTimeout(() => setDebounced(query.trim()), 300)
    return () => clearTimeout(timer)
  }, [query])

  useEffect(() => {
    if (!debounced) {
      setResults([])
      setLoading(false)
      return
    }
    let cancelled = false
    setLoading(true)
    searchUsers(debounced)
      .then((users) => {
        if (!cancelled) setResults(users)
      })
      .catch(() => {
        if (!cancelled) setResults([])
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [debounced])

  const exclude = new Set(excludeIds)
  // Rang d'un id parmi les récents (Infinity si absent) → tri stable « récents d'abord ».
  const recentRank = new Map(recentUsers.map((u, i) => [u.id, i]))
  const rankOf = (id: string) => recentRank.get(id) ?? Number.POSITIVE_INFINITY
  const visible = results
    .filter((u) => !exclude.has(u.id))
    .sort((a, b) => rankOf(a.id) - rankOf(b.id))

  // Champ vide : on propose les destinataires récents (filtrés) plutôt que rien.
  const recents = recentUsers.filter((u) => !exclude.has(u.id))
  const showRecents = !debounced && recents.length > 0

  return (
    <div className="flex flex-col">
      <div className="relative">
        <Search
          className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground"
          aria-hidden
        />
        <input
          type="search"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder={t('messages.search_user_placeholder')}
          autoFocus={autoFocus}
          className="w-full rounded-full border border-input bg-background/60 py-2.5 pl-10 pr-4 text-sm backdrop-blur placeholder:text-muted-foreground focus:border-[#5B6CFF] focus:outline-none"
        />
      </div>

      <div className="mt-2 max-h-72 overflow-y-auto">
        {loading ? (
          <div className="flex justify-center py-8">
            <Loader2 className="h-5 w-5 animate-spin text-[#5B6CFF] dark:text-[#9aa6ff]" />
          </div>
        ) : debounced && visible.length === 0 ? (
          <p className="py-8 text-center text-sm text-muted-foreground">
            {t('messages.no_users')}
          </p>
        ) : (
          <>
            {showRecents && recentLabel && (
              <p className="px-2 pb-1 text-xs font-semibold text-muted-foreground">{recentLabel}</p>
            )}
            <ul className="divide-y divide-border">
              {(showRecents ? recents : visible).map((user) => (
                <li key={user.id}>
                  <button
                    type="button"
                    onClick={() => onPick(user)}
                    className="flex w-full items-center gap-3 rounded-lg px-2 py-2.5 text-left transition-colors hover:bg-accent"
                  >
                    <Avatar className="h-9 w-9 shrink-0">
                      {user.avatarUrl && <AvatarImage src={user.avatarUrl} alt={user.displayName} />}
                      <AvatarFallback>
                        {initialOf(user.displayName, user.username)}
                      </AvatarFallback>
                      <ActivityPresenceDot userId={user.id} />
                    </Avatar>
                    <div className="flex min-w-0 flex-col">
                      <span className="flex min-w-0 items-center gap-1 text-sm font-bold text-foreground">
                        <span className="truncate">{user.displayName}</span>
                        <CertificationBadge userId={user.id} certification={user.certification} className="h-4 w-4" />
                      </span>
                      <span className="truncate text-xs text-muted-foreground">@{user.username}</span>
                    </div>
                  </button>
                </li>
              ))}
            </ul>
          </>
        )}
      </div>
    </div>
  )
}
