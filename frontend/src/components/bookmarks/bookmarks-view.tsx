'use client'

import { useCallback, useEffect, useRef, useState } from 'react'
import { Bookmark, Loader2, MoreHorizontal, Pencil, Plus, Trash2 } from 'lucide-react'

import { cn } from '@/lib/utils'
import { useInfiniteScroll } from '@/lib/use-infinite-scroll'
import {
  createCollection,
  deleteCollection,
  listAllBookmarks,
  listCollectionPosts,
  listCollections,
  renameCollection,
  type BookmarkCollection,
} from '@/lib/bookmarks'
import type { FeedPost } from '@/lib/posts'
import { useToast } from '@/hooks/use-toast'
import { useT } from '@/components/language-provider'
import { PostCard } from '@/components/feed/post-card'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Input } from '@/components/ui/input'

const PAGE = 10
const ALL = 'all'

function mergeUnique(current: FeedPost[], incoming: FeedPost[]): FeedPost[] {
  const seen = new Set(current.map((p) => p.id))
  return [...current, ...incoming.filter((p) => !seen.has(p.id))]
}

/**
 * Page « Signets » : barre de playlists (Tous + playlists nommées), gestion des
 * playlists (créer / renommer / supprimer) et liste des posts de la playlist
 * active (défilement infini, réutilise `PostCard`).
 */
export function BookmarksView() {
  const t = useT()
  const { toast } = useToast()

  const [collections, setCollections] = useState<BookmarkCollection[]>([])
  const [active, setActive] = useState<string>(ALL)
  const [posts, setPosts] = useState<FeedPost[]>([])
  const [loading, setLoading] = useState(true)
  const [loadingMore, setLoadingMore] = useState(false)
  const [hasMore, setHasMore] = useState(false)
  const offsetRef = useRef(0)

  // Édition de playlist (création / renommage / suppression).
  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<BookmarkCollection | null>(null)
  const [name, setName] = useState('')
  const [saving, setSaving] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState<BookmarkCollection | null>(null)

  const fetchPage = useCallback(
    (tab: string, offset: number) =>
      tab === ALL ? listAllBookmarks(PAGE, offset) : listCollectionPosts(tab, PAGE, offset),
    [],
  )

  const loadCollections = useCallback(() => {
    listCollections()
      .then(setCollections)
      .catch(() => undefined)
  }, [])

  useEffect(() => {
    loadCollections()
  }, [loadCollections])

  // Chargement initial / changement de playlist active.
  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setPosts([])
    offsetRef.current = 0
    fetchPage(active, 0)
      .then((list) => {
        if (cancelled) return
        setPosts(list)
        offsetRef.current = list.length
        setHasMore(list.length === PAGE)
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
  }, [active, fetchPage, t, toast])

  const loadMore = useCallback(async () => {
    setLoadingMore(true)
    try {
      const next = await fetchPage(active, offsetRef.current)
      offsetRef.current += next.length
      setPosts((prev) => mergeUnique(prev, next))
      setHasMore(next.length === PAGE)
    } catch {
      setHasMore(false)
    } finally {
      setLoadingMore(false)
    }
  }, [active, fetchPage])

  const sentinelRef = useInfiniteScroll(loadMore, { hasMore, loading: loading || loadingMore })

  const handleDeleted = useCallback((id: string) => {
    setPosts((prev) => prev.filter((p) => p.id !== id))
  }, [])

  // Un post désépinglé/épinglé met simplement à jour la carte concernée.
  const handleUpdated = useCallback((post: FeedPost) => {
    setPosts((prev) => prev.map((p) => (p.id === post.id ? post : p)))
  }, [])

  const activeCollection = collections.find((c) => c.id === active) ?? null

  function openCreate() {
    setEditing(null)
    setName('')
    setFormOpen(true)
  }

  function openRename(coll: BookmarkCollection) {
    setEditing(coll)
    setName(coll.name)
    setFormOpen(true)
  }

  async function submitForm() {
    const trimmed = name.trim()
    if (!trimmed || saving) return
    setSaving(true)
    try {
      if (editing) {
        const updated = await renameCollection(editing.id, trimmed)
        setCollections((prev) => prev.map((c) => (c.id === updated.id ? { ...c, name: updated.name } : c)))
      } else {
        const created = await createCollection(trimmed)
        setCollections((prev) => [created, ...prev])
      }
      setFormOpen(false)
    } catch {
      toast({ title: t('common.action_failed'), variant: 'destructive' })
    } finally {
      setSaving(false)
    }
  }

  async function confirmDelete() {
    if (!deleteTarget) return
    const target = deleteTarget
    setDeleteTarget(null)
    try {
      await deleteCollection(target.id)
      setCollections((prev) => prev.filter((c) => c.id !== target.id))
      if (active === target.id) setActive(ALL)
    } catch {
      toast({ title: t('common.action_failed'), variant: 'destructive' })
    }
  }

  return (
    <div className="flex flex-col">
      {/* En-tête + barre de playlists */}
      <div className="panel z-10 border-b lg:sticky lg:top-0">
        <div className="flex items-center justify-between gap-2 px-4 py-3">
          <h1 className="brand-text flex items-center gap-2 text-xl font-bold">
            <Bookmark className="h-5 w-5" />
            {t('bookmarks.title')}
          </h1>
          {activeCollection && !activeCollection.isDefault && (
            <DropdownMenu>
              <DropdownMenuTrigger
                aria-label={t('bookmarks.manage')}
                className="rounded-full p-1.5 text-muted-foreground transition-colors hover:bg-primary/10 hover:text-primary focus:outline-none"
              >
                <MoreHorizontal className="h-5 w-5" />
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onClick={() => openRename(activeCollection)} className="cursor-pointer">
                  <Pencil className="mr-2 h-4 w-4" />
                  {t('bookmarks.rename')}
                </DropdownMenuItem>
                <DropdownMenuItem
                  onClick={() => setDeleteTarget(activeCollection)}
                  className="cursor-pointer text-red-500 focus:text-red-500"
                >
                  <Trash2 className="mr-2 h-4 w-4" />
                  {t('bookmarks.delete')}
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          )}
        </div>

        <div className="flex items-center gap-2 overflow-x-auto px-3 pb-3">
          <Pill active={active === ALL} onClick={() => setActive(ALL)}>
            {t('bookmarks.all')}
          </Pill>
          {collections.map((c) => (
            <Pill key={c.id} active={active === c.id} onClick={() => setActive(c.id)}>
              {c.isDefault ? t('bookmarks.default_name') : c.name}
              <span className="ml-1.5 text-xs opacity-70">{c.itemsCount}</span>
            </Pill>
          ))}
          <button
            type="button"
            onClick={openCreate}
            aria-label={t('bookmarks.new_playlist')}
            className="flex shrink-0 items-center gap-1 rounded-full border border-dashed border-border px-3 py-1.5 text-sm text-muted-foreground transition-colors hover:border-primary hover:text-primary"
          >
            <Plus className="h-4 w-4" />
            {t('bookmarks.new_playlist')}
          </button>
        </div>
      </div>

      {/* Liste */}
      {loading ? (
        <div className="flex justify-center py-12">
          <Loader2 className="h-6 w-6 animate-spin text-[#5B6CFF]" aria-hidden />
        </div>
      ) : posts.length === 0 ? (
        <EmptyState
          title={t('bookmarks.empty_title')}
          message={active === ALL ? t('bookmarks.empty_message') : t('bookmarks.empty_playlist')}
        />
      ) : (
        <>
          <div className="divide-y divide-border">
            {posts.map((post) => (
              <PostCard key={post.id} post={post} onDeleted={handleDeleted} onUpdated={handleUpdated} />
            ))}
          </div>
          {hasMore && (
            <div ref={sentinelRef} className="flex justify-center py-6">
              {loadingMore && <Loader2 className="h-5 w-5 animate-spin text-[#5B6CFF]" aria-hidden />}
            </div>
          )}
        </>
      )}

      {/* Dialogue créer / renommer une playlist */}
      <Dialog open={formOpen} onOpenChange={setFormOpen}>
        <DialogContent className="panel top-32 translate-y-0 border shadow-[0_28px_80px_rgba(91,108,255,0.24)] sm:max-w-sm">
          <DialogHeader>
            <DialogTitle>{editing ? t('bookmarks.rename') : t('bookmarks.new_playlist')}</DialogTitle>
            <DialogDescription className="sr-only">
              {editing ? t('bookmarks.rename') : t('bookmarks.new_playlist')}
            </DialogDescription>
          </DialogHeader>
          <Input
            value={name}
            maxLength={60}
            autoFocus
            onChange={(e) => setName(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                e.preventDefault()
                void submitForm()
              }
            }}
            placeholder={t('bookmarks.new_playlist_placeholder')}
          />
          <DialogFooter>
            <Button
              onClick={() => void submitForm()}
              disabled={!name.trim() || saving}
              className="bg-gradient-to-r from-[#8D3DFF] via-[#5B6CFF] to-[#47D9FF] text-white"
            >
              {saving ? <Loader2 className="h-4 w-4 animate-spin" /> : t('bookmarks.save')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Confirmation de suppression */}
      <Dialog open={Boolean(deleteTarget)} onOpenChange={(o) => !o && setDeleteTarget(null)}>
        <DialogContent className="panel top-32 translate-y-0 border shadow-[0_28px_80px_rgba(91,108,255,0.24)] sm:max-w-sm">
          <DialogHeader>
            <DialogTitle>{t('bookmarks.delete_title')}</DialogTitle>
            <DialogDescription>{t('bookmarks.delete_confirm')}</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="ghost" onClick={() => setDeleteTarget(null)}>
              {t('common.cancel')}
            </Button>
            <Button variant="destructive" onClick={() => void confirmDelete()}>
              {t('bookmarks.delete')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

function Pill({
  active,
  onClick,
  children,
}: {
  active: boolean
  onClick: () => void
  children: React.ReactNode
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        'flex shrink-0 items-center rounded-full px-4 py-1.5 text-sm font-medium transition-colors',
        active
          ? 'bg-gradient-to-r from-[#8D3DFF] via-[#5B6CFF] to-[#47D9FF] text-white shadow-sm'
          : 'border border-border text-muted-foreground hover:bg-accent',
      )}
    >
      {children}
    </button>
  )
}

function EmptyState({ title, message }: { title: string; message: string }) {
  return (
    <div className="glass mx-4 mt-6 flex flex-col items-center gap-2 rounded-[26px] border px-8 py-16 text-center backdrop-blur-xl">
      <Bookmark className="h-10 w-10 text-[#5B6CFF] dark:text-[#9aa6ff]" aria-hidden />
      <h2 className="text-lg font-bold">{title}</h2>
      <p className="max-w-sm text-sm text-muted-foreground">{message}</p>
    </div>
  )
}
