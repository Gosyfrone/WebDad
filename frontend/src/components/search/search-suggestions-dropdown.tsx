'use client'

import { useEffect, useState } from 'react'
import { Hash, Loader2, Search } from 'lucide-react'

import { searchUsers } from '@/lib/api'
import { listHashtagTrends, type HashtagTrend } from '@/lib/posts'
import { hashtagHref, profilHref } from '@/lib/routes'
import type { RelationUser } from '@/types'
import { useT } from '@/components/language-provider'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'

interface SearchSuggestionsDropdownProps {
  query: string
  open: boolean
  onPick: (href: string) => void
}

export function SearchSuggestionsDropdown({
  query,
  open,
  onPick,
}: SearchSuggestionsDropdownProps) {
  const t = useT()
  const [debounced, setDebounced] = useState('')
  const [hashtags, setHashtags] = useState<HashtagTrend[]>([])
  const [users, setUsers] = useState<RelationUser[]>([])
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    const timer = setTimeout(() => setDebounced(query.trim()), 220)
    return () => clearTimeout(timer)
  }, [query])

  useEffect(() => {
    if (!open || !debounced) {
      setHashtags([])
      setUsers([])
      setLoading(false)
      return
    }

    let cancelled = false
    setLoading(true)
    Promise.all([
      listHashtagTrends(3, debounced).catch(() => []),
      searchUsers(debounced.replace(/^#/, '')).catch(() => []),
    ])
      .then(([nextHashtags, nextUsers]) => {
        if (cancelled) return
        setHashtags(nextHashtags)
        setUsers(nextUsers.slice(0, 6))
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })

    return () => {
      cancelled = true
    }
  }, [debounced, open])

  if (!open || !query.trim()) return null

  const hasContent = hashtags.length > 0 || users.length > 0

  return (
    <div className="absolute left-0 right-0 top-[calc(100%+0.45rem)] z-50 max-h-[70vh] overflow-y-auto rounded-[18px] border border-white/10 bg-black text-white shadow-2xl shadow-black/40 ring-1 ring-white/10">
      {loading && !hasContent ? (
        <div className="flex items-center justify-center py-8 text-white/70">
          <Loader2 className="h-5 w-5 animate-spin" aria-hidden />
        </div>
      ) : hasContent ? (
        <>
          {hashtags.map((trend) => (
            <SuggestionButton
              key={trend.tag}
              onPick={() => onPick(hashtagHref(trend.tag, 'top'))}
            >
              <span className="grid h-9 w-9 shrink-0 place-items-center rounded-full text-white">
                <Search className="h-6 w-6" aria-hidden />
              </span>
              <span className="min-w-0">
                <span className="block truncate text-[15px] font-bold leading-5">
                  {trend.tag}
                </span>
                <span className="block truncate text-[15px] leading-5 text-white/45">
                  {t('search_suggestions.trend')}
                </span>
              </span>
            </SuggestionButton>
          ))}
          {users.length > 0 && hashtags.length > 0 && <div className="border-t border-white/10" />}
          {users.map((user) => (
            <SuggestionButton
              key={user.id}
              onPick={() => onPick(profilHref(user.username))}
            >
              <Avatar className="h-10 w-10 shrink-0">
                {user.avatarUrl && <AvatarImage src={user.avatarUrl} alt={user.displayName} />}
                <AvatarFallback>{initials(user)}</AvatarFallback>
              </Avatar>
              <span className="min-w-0">
                <span className="block truncate text-[15px] font-bold leading-5">
                  {user.displayName}
                </span>
                <span className="block truncate text-[15px] leading-5 text-white/45">
                  @{user.username}
                </span>
                {user.bio && (
                  <span className="block truncate text-[13px] leading-5 text-white/45">
                    {user.bio}
                  </span>
                )}
              </span>
            </SuggestionButton>
          ))}
        </>
      ) : (
        <div className="flex items-center gap-3 px-5 py-5 text-sm text-white/55">
          <Hash className="h-5 w-5" aria-hidden />
          {t('search_suggestions.empty')}
        </div>
      )}
    </div>
  )
}

function SuggestionButton({
  children,
  onPick,
}: {
  children: React.ReactNode
  onPick: () => void
}) {
  return (
    <button
      type="button"
      onMouseDown={(event) => event.preventDefault()}
      onClick={onPick}
      className="flex w-full items-center gap-4 px-5 py-3 text-left transition-colors hover:bg-white/10 focus:bg-white/10 focus:outline-none"
    >
      {children}
    </button>
  )
}

function initials(user: RelationUser): string {
  return (user.displayName.charAt(0) || user.username.charAt(0) || '?').toUpperCase()
}
