'use client'

import { useEffect, useState } from 'react'
import { Check, Globe, Loader2, Search } from 'lucide-react'

import {
  joinCommunity,
  listCommunities,
  type CommunitySummary,
  type Conversation,
} from '@/lib/messages'
import { useToast } from '@/hooks/use-toast'
import { useLanguage } from '@/components/language-provider'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

interface DiscoverCommunitiesDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  /** Communauté rejointe → le parent l'ajoute à la liste et l'ouvre. */
  onJoined: (conv: Conversation) => void
}

/**
 * Annuaire public des communautés : recherche (nom en clair) + bouton Rejoindre
 * (auto-join en viewer, le serveur remet la clé). Les communautés déjà
 * rejointes sont marquées.
 */
export function DiscoverCommunitiesDialog({
  open,
  onOpenChange,
  onJoined,
}: DiscoverCommunitiesDialogProps) {
  const { t } = useLanguage()
  const { toast } = useToast()
  const [query, setQuery] = useState('')
  const [debounced, setDebounced] = useState('')
  const [items, setItems] = useState<CommunitySummary[]>([])
  const [loading, setLoading] = useState(false)
  const [joiningId, setJoiningId] = useState<string | null>(null)

  useEffect(() => {
    const timer = setTimeout(() => setDebounced(query.trim()), 300)
    return () => clearTimeout(timer)
  }, [query])

  // Charge l'annuaire à l'ouverture et à chaque recherche.
  useEffect(() => {
    if (!open) return
    let cancelled = false
    setLoading(true)
    listCommunities(debounced)
      .then((list) => {
        if (!cancelled) setItems(list)
      })
      .catch(() => {
        if (!cancelled) setItems([])
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [open, debounced])

  async function join(community: CommunitySummary) {
    if (joiningId) return
    setJoiningId(community.id)
    try {
      const conv = await joinCommunity(community.id)
      setItems((prev) =>
        prev.map((c) => (c.id === community.id ? { ...c, isMember: true } : c)),
      )
      onJoined(conv)
    } catch {
      toast({ title: t('messages.join_failed'), variant: 'destructive' })
    } finally {
      setJoiningId(null)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>{t('messages.discover')}</DialogTitle>
          <DialogDescription>{t('messages.discover_desc')}</DialogDescription>
        </DialogHeader>

        <div className="relative">
          <Search
            className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground"
            aria-hidden
          />
          <input
            type="search"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder={t('messages.search_community_placeholder')}
            className="glass w-full rounded-full border py-2.5 pl-10 pr-4 text-sm backdrop-blur placeholder:text-muted-foreground focus:border-[#5B6CFF] focus:outline-none"
          />
        </div>

        <div className="max-h-80 overflow-y-auto">
          {loading ? (
            <div className="flex justify-center py-10">
              <Loader2 className="h-6 w-6 animate-spin text-[#5B6CFF] dark:text-[#9aa6ff]" />
            </div>
          ) : items.length === 0 ? (
            <p className="py-10 text-center text-sm text-muted-foreground">
              {t('messages.no_communities')}
            </p>
          ) : (
            <ul className="divide-y divide-border">
              {items.map((c) => (
                <li key={c.id} className="flex items-center gap-3 py-3">
                  <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-gradient-to-br from-[#8D3DFF] via-[#5B6CFF] to-[#47D9FF] text-white">
                    <Globe className="h-5 w-5" aria-hidden />
                  </div>
                  <div className="flex min-w-0 flex-1 flex-col">
                    <span className="truncate text-sm font-bold text-foreground">{c.title}</span>
                    <span className="text-xs text-muted-foreground">
                      {t('messages.members_count', { count: c.memberCount })}
                    </span>
                  </div>
                  {c.isMember ? (
                    <span className="flex items-center gap-1 text-xs font-semibold text-muted-foreground">
                      <Check className="h-4 w-4" /> {t('messages.joined')}
                    </span>
                  ) : (
                    <Button
                      type="button"
                      size="sm"
                      onClick={() => join(c)}
                      disabled={joiningId === c.id}
                      className="rounded-full"
                    >
                      {joiningId === c.id ? (
                        <Loader2 className="h-4 w-4 animate-spin" />
                      ) : (
                        t('messages.join')
                      )}
                    </Button>
                  )}
                </li>
              ))}
            </ul>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}
