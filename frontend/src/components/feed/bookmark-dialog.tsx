'use client'

import { useEffect, useState } from 'react'
import { Bookmark, Check, Loader2, Plus } from 'lucide-react'

import { cn } from '@/lib/utils'
import {
  addBookmark,
  createCollection,
  getPostCollections,
  listCollections,
  removeBookmark,
  type BookmarkCollection,
} from '@/lib/bookmarks'
import { useToast } from '@/hooks/use-toast'
import { useT } from '@/components/language-provider'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'

interface BookmarkDialogProps {
  postId: string
  open: boolean
  onOpenChange: (open: boolean) => void
  /** Appelé quand l'appartenance change : true si le post est dans ≥1 playlist. */
  onMembershipChange?: (bookmarkedAnywhere: boolean) => void
}

/**
 * Sélecteur « Ranger dans… » : liste les playlists de l'utilisateur avec une
 * case cochée pour celles contenant le post, permet d'ajouter/retirer le post
 * par playlist (optimiste) et d'en créer une nouvelle à la volée.
 */
export function BookmarkDialog({ postId, open, onOpenChange, onMembershipChange }: BookmarkDialogProps) {
  const t = useT()
  const { toast } = useToast()

  const [collections, setCollections] = useState<BookmarkCollection[]>([])
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [loading, setLoading] = useState(true)
  const [busy, setBusy] = useState<Set<string>>(new Set())
  const [newName, setNewName] = useState('')
  const [creating, setCreating] = useState(false)

  // (Re)charge playlists + appartenance à chaque ouverture.
  useEffect(() => {
    if (!open) return
    let cancelled = false
    setLoading(true)
    Promise.all([listCollections(), getPostCollections(postId)])
      .then(([colls, memberIds]) => {
        if (cancelled) return
        setCollections(colls)
        setSelected(new Set(memberIds))
      })
      .catch(() => {
        if (!cancelled) toast({ title: t('bookmarks.load_error'), variant: 'destructive' })
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [open, postId, t, toast])

  function notify(next: Set<string>) {
    onMembershipChange?.(next.size > 0)
  }

  async function toggle(collectionId: string) {
    if (busy.has(collectionId)) return
    const isIn = selected.has(collectionId)
    const next = new Set(selected)
    if (isIn) next.delete(collectionId)
    else next.add(collectionId)
    setSelected(next)
    notify(next)
    setBusy((b) => new Set(b).add(collectionId))
    try {
      if (isIn) await removeBookmark(postId, collectionId)
      else await addBookmark(postId, collectionId)
      setCollections((prev) =>
        prev.map((c) =>
          c.id === collectionId
            ? { ...c, itemsCount: Math.max(0, c.itemsCount + (isIn ? -1 : 1)) }
            : c,
        ),
      )
    } catch {
      // Rollback.
      const rolled = new Set(selected)
      setSelected(rolled)
      notify(rolled)
      toast({ title: t('common.action_failed'), variant: 'destructive' })
    } finally {
      setBusy((b) => {
        const copy = new Set(b)
        copy.delete(collectionId)
        return copy
      })
    }
  }

  async function handleCreate() {
    const name = newName.trim()
    if (!name || creating) return
    setCreating(true)
    try {
      const coll = await createCollection(name)
      await addBookmark(postId, coll.id)
      setNewName('')
      setCollections((prev) => [{ ...coll, itemsCount: 1 }, ...prev])
      const next = new Set(selected).add(coll.id)
      setSelected(next)
      notify(next)
    } catch {
      toast({ title: t('common.action_failed'), variant: 'destructive' })
    } finally {
      setCreating(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="panel top-24 translate-y-0 border p-0 shadow-[0_28px_80px_rgba(91,108,255,0.24)] sm:max-w-md">
        <DialogHeader className="border-b px-5 py-4">
          <DialogTitle className="flex items-center gap-2 text-base">
            <Bookmark className="h-4 w-4 text-primary" />
            {t('bookmarks.save_to')}
          </DialogTitle>
          <DialogDescription className="sr-only">{t('bookmarks.save_to')}</DialogDescription>
        </DialogHeader>

        <div className="max-h-[50vh] overflow-y-auto px-3 py-2">
          {loading ? (
            <div className="flex justify-center py-10">
              <Loader2 className="h-5 w-5 animate-spin text-[#5B6CFF]" aria-hidden />
            </div>
          ) : collections.length === 0 ? (
            <p className="px-3 py-6 text-center text-sm text-muted-foreground">
              {t('bookmarks.no_playlists')}
            </p>
          ) : (
            <ul className="flex flex-col">
              {collections.map((c) => {
                const checked = selected.has(c.id)
                return (
                  <li key={c.id}>
                    <button
                      type="button"
                      onClick={() => toggle(c.id)}
                      disabled={busy.has(c.id)}
                      className="flex w-full items-center gap-3 rounded-xl px-3 py-3 text-left transition-colors hover:bg-accent"
                    >
                      <span
                        className={cn(
                          'grid h-5 w-5 shrink-0 place-items-center rounded-md border transition-colors',
                          checked
                            ? 'border-transparent bg-gradient-to-br from-[#8D3DFF] via-[#5B6CFF] to-[#47D9FF] text-white'
                            : 'border-border',
                        )}
                      >
                        {busy.has(c.id) ? (
                          <Loader2 className="h-3 w-3 animate-spin" />
                        ) : checked ? (
                          <Check className="h-3.5 w-3.5" />
                        ) : null}
                      </span>
                      <span className="min-w-0 flex-1 truncate text-sm font-medium">
                        {c.isDefault ? t('bookmarks.default_name') : c.name}
                      </span>
                      <span className="shrink-0 text-xs text-muted-foreground">{c.itemsCount}</span>
                    </button>
                  </li>
                )
              })}
            </ul>
          )}
        </div>

        {/* Créer une playlist */}
        <div className="flex items-center gap-2 border-t px-4 py-3">
          <Input
            value={newName}
            maxLength={60}
            onChange={(e) => setNewName(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                e.preventDefault()
                void handleCreate()
              }
            }}
            placeholder={t('bookmarks.new_playlist_placeholder')}
            className="h-9 flex-1"
          />
          <Button
            type="button"
            size="sm"
            onClick={() => void handleCreate()}
            disabled={!newName.trim() || creating}
            className="shrink-0 gap-1 bg-gradient-to-r from-[#8D3DFF] via-[#5B6CFF] to-[#47D9FF] text-white"
          >
            {creating ? <Loader2 className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />}
            {t('bookmarks.create')}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}
