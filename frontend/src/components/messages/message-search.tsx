'use client'

import { Fragment, type ReactNode, useEffect, useMemo, useState } from 'react'
import { Loader2, Search } from 'lucide-react'

import {
  listAllMessages,
  searchMessages,
  type ChatMessage,
  type Conversation,
} from '@/lib/messages'
import { useResolvedUser } from '@/lib/use-resolved-user'
import { initialOf } from '@/lib/utils'
import { useLanguage } from '@/components/language-provider'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { ActivityPresenceDot } from '@/components/profil/activity-presence-dot'
import { CertificationBadge } from '@/components/profil/certification-badge'

/**
 * Recherche de messages DANS une conversation (DM / groupe / communauté).
 *
 * E2EE oblige : le serveur ne voit que du chiffré, la recherche se fait donc
 * **côté client**. À la première recherche, on récupère l'historique complet
 * (déchiffré localement, borné) une seule fois, puis on filtre en mémoire.
 */
export function MessageSearch({ conversation }: { conversation: Conversation }) {
  const { t } = useLanguage()
  const [query, setQuery] = useState('')
  const [debounced, setDebounced] = useState('')
  const [all, setAll] = useState<ChatMessage[] | null>(null)
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    const id = setTimeout(() => setDebounced(query.trim()), 300)
    return () => clearTimeout(id)
  }, [query])

  // Charge l'historique complet à la 1re recherche (une seule fois).
  useEffect(() => {
    if (!debounced || all !== null || loading) return
    let cancelled = false
    setLoading(true)
    listAllMessages(conversation)
      .then((msgs) => {
        if (!cancelled) setAll(msgs)
      })
      .catch(() => {
        if (!cancelled) setAll([])
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [debounced, all, loading, conversation])

  // Résultats : plus récents d'abord.
  const results = useMemo(() => {
    if (!debounced || !all) return []
    return searchMessages(all, debounced).slice().reverse()
  }, [debounced, all])

  return (
    <div>
      <div className="relative">
        <Search
          className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground"
          aria-hidden
        />
        <input
          type="search"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder={t('messages.search_in_conversation')}
          className="glass w-full rounded-full border py-2.5 pl-10 pr-4 text-sm backdrop-blur placeholder:text-muted-foreground focus:border-[#5B6CFF] focus:outline-none"
        />
      </div>

      {debounced &&
        (loading && all === null ? (
          <div className="flex justify-center py-6">
            <Loader2 className="h-5 w-5 animate-spin text-[#5B6CFF] dark:text-[#9aa6ff]" />
          </div>
        ) : results.length === 0 ? (
          <p className="py-4 text-center text-sm text-muted-foreground">
            {t('messages.search_no_results')}
          </p>
        ) : (
          <ul className="mt-2 max-h-60 divide-y divide-border overflow-y-auto">
            {results.map((m) => (
              <SearchResult key={m.id} message={m} query={debounced} />
            ))}
          </ul>
        ))}
    </div>
  )
}

function SearchResult({ message, query }: { message: ChatMessage; query: string }) {
  const { locale } = useLanguage()
  const sender = useResolvedUser(message.senderId)

  return (
    <li className="flex items-start gap-3 py-2.5">
      <Avatar className="h-8 w-8 shrink-0">
        {sender?.avatarUrl && <AvatarImage src={sender.avatarUrl} alt={sender.displayName} />}
        <AvatarFallback className="text-xs">
          {initialOf(sender?.displayName, sender?.username)}
        </AvatarFallback>
        <ActivityPresenceDot userId={sender?.id} className="h-2.5 w-2.5" />
      </Avatar>
      <div className="flex min-w-0 flex-1 flex-col">
        <div className="flex items-baseline justify-between gap-2">
          <span className="flex min-w-0 items-center gap-1 text-sm font-semibold text-foreground">
            <span className="truncate">{sender?.displayName ?? '…'}</span>
            <CertificationBadge certification={sender?.certification} className="h-4 w-4" />
          </span>
          <span className="shrink-0 text-[11px] text-muted-foreground">
            {formatDateTime(message.createdAt, locale)}
          </span>
        </div>
        <p className="line-clamp-2 text-sm text-muted-foreground">
          {highlight(message.text, query)}
        </p>
      </div>
    </li>
  )
}

/** Surligne les occurrences (insensible à la casse) de `query` dans `text`. */
function highlight(text: string, query: string): ReactNode {
  const q = query.toLowerCase()
  if (!q) return text
  const lower = text.toLowerCase()
  const parts: ReactNode[] = []
  let i = 0
  let key = 0
  while (i < text.length) {
    const idx = lower.indexOf(q, i)
    if (idx < 0) {
      parts.push(<Fragment key={key++}>{text.slice(i)}</Fragment>)
      break
    }
    if (idx > i) parts.push(<Fragment key={key++}>{text.slice(i, idx)}</Fragment>)
    parts.push(
      <mark key={key++} className="rounded bg-[#8D3DFF]/20 px-0.5 text-foreground">
        {text.slice(idx, idx + q.length)}
      </mark>,
    )
    i = idx + q.length
  }
  return parts
}

function formatDateTime(iso: string, locale: string): string {
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return ''
  const intl = locale === 'en' ? 'en-US' : 'fr-FR'
  return new Intl.DateTimeFormat(intl, {
    day: 'numeric',
    month: 'short',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date)
}
