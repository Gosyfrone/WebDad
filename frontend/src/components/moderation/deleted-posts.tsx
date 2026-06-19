'use client'

import { useCallback, useEffect, useState } from 'react'
import { Loader2, RotateCcw, Search, Trash2, X } from 'lucide-react'

import {
  listDeletedPosts,
  restoreDeletedPost,
  purgeDeletedPost,
  type DeletedPost,
  type DeletedPostsFilter,
} from '@/lib/moderation'
import { searchUsers } from '@/lib/api'
import { resolveUser, type ResolvedUser } from '@/lib/user-cache'
import { initialOf, timeAgo } from '@/lib/utils'
import type { RelationUser } from '@/types'
import { ProfilLink } from '@/components/profil/profil-link'
import { ActivityPresenceDot } from '@/components/profil/activity-presence-dot'
import { useLanguage } from '@/components/language-provider'
import { useToast } from '@/hooks/use-toast'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

/** Taille de page (alignée sur le défaut serveur) pour le « Charger plus ». */
const PAGE_SIZE = 50

/**
 * Corbeille de modération : les tweets retirés en suppression douce (partagée
 * mod + admin). Pour chaque post : auteur + modérateur ayant retiré + date,
 * aperçu du contenu/média, et deux actions :
 *   - Restaurer (réversible, sans confirmation) ;
 *   - Supprimer définitivement (irréversible → confirmation, car la purge ne se
 *     rejoue pas, contrairement au retrait initial).
 *
 * Filtres SERVEUR cumulables (volume potentiellement élevé) : auteur (barre de
 * recherche → `author_id`) et plage de date de retrait (`since`/`until`).
 * Pagination par « Charger plus » (offset).
 */
export function DeletedPosts() {
  const { t, locale } = useLanguage()
  const { toast } = useToast()
  const [posts, setPosts] = useState<DeletedPost[]>([])
  const [people, setPeople] = useState<Record<string, ResolvedUser>>({})
  const [loading, setLoading] = useState(true)
  const [loadingMore, setLoadingMore] = useState(false)
  const [hasMore, setHasMore] = useState(false)
  const [error, setError] = useState(false)
  const [busyId, setBusyId] = useState<string | null>(null)
  const [purgeTarget, setPurgeTarget] = useState<DeletedPost | null>(null)

  // Filtres (serveur). `authorFilter` : utilisateur choisi dans les suggestions.
  const [authorFilter, setAuthorFilter] = useState<{ id: string; label: string } | null>(null)
  const [since, setSince] = useState('')
  const [until, setUntil] = useState('')
  // Recherche d'utilisateur (auteur) : saisie libre + suggestions résolues.
  const [userQuery, setUserQuery] = useState('')
  const [suggestions, setSuggestions] = useState<RelationUser[]>([])

  const hasFilters = authorFilter !== null || since !== '' || until !== ''

  // Construit la charge de filtres serveur (dates jour → bornes ISO début/fin).
  const buildFilter = useCallback(
    (offset: number): DeletedPostsFilter => ({
      limit: PAGE_SIZE,
      offset,
      authorId: authorFilter?.id,
      since: since ? new Date(`${since}T00:00:00`).toISOString() : undefined,
      until: until ? new Date(`${until}T23:59:59.999`).toISOString() : undefined,
    }),
    [authorFilter, since, until],
  )

  // Résout (mémoïsé) auteurs + modérateurs d'un lot et les fusionne dans `people`.
  const resolvePeopleFor = useCallback(async (list: DeletedPost[]) => {
    const ids = Array.from(new Set(list.flatMap((p) => [p.authorId, p.hiddenBy]).filter(Boolean)))
    const resolved = await Promise.all(ids.map((id) => resolveUser(id)))
    setPeople((prev) => ({ ...prev, ...Object.fromEntries(ids.map((id, i) => [id, resolved[i]])) }))
  }, [])

  // (Re)charge la première page selon les filtres courants.
  const load = useCallback(async () => {
    setLoading(true)
    setError(false)
    try {
      const list = await listDeletedPosts(buildFilter(0))
      setPosts(list)
      setHasMore(list.length === PAGE_SIZE)
      await resolvePeopleFor(list)
    } catch {
      setError(true)
    } finally {
      setLoading(false)
    }
  }, [buildFilter, resolvePeopleFor])

  // Recharge à chaque changement de filtre (load dépend de buildFilter).
  useEffect(() => {
    void load()
  }, [load])

  // Page suivante : empile à la suite (offset = nb de cartes déjà affichées).
  async function loadMore() {
    setLoadingMore(true)
    try {
      const list = await listDeletedPosts(buildFilter(posts.length))
      setPosts((prev) => [...prev, ...list])
      setHasMore(list.length === PAGE_SIZE)
      await resolvePeopleFor(list)
    } catch {
      toast({ title: t('admin.action_failed'), variant: 'destructive' })
    } finally {
      setLoadingMore(false)
    }
  }

  // Recherche d'utilisateur débouncée (300 ms) → suggestions (nom ou @username).
  useEffect(() => {
    const q = userQuery.trim()
    if (!q) {
      setSuggestions([])
      return
    }
    const handle = setTimeout(async () => {
      setSuggestions((await searchUsers(q).catch(() => [])).slice(0, 6))
    }, 300)
    return () => clearTimeout(handle)
  }, [userQuery])

  function selectAuthor(u: RelationUser) {
    setAuthorFilter({ id: u.id, label: u.username ? `@${u.username}` : u.displayName || t('common.user') })
    setUserQuery('')
    setSuggestions([])
  }

  function resetFilters() {
    setAuthorFilter(null)
    setSince('')
    setUntil('')
    setUserQuery('')
    setSuggestions([])
  }

  async function onRestore(post: DeletedPost) {
    setBusyId(post.id)
    try {
      await restoreDeletedPost(post.id)
      setPosts((prev) => prev.filter((p) => p.id !== post.id))
      toast({ title: t('moderation.restored_toast') })
    } catch {
      toast({ title: t('admin.action_failed'), variant: 'destructive' })
    } finally {
      setBusyId(null)
    }
  }

  async function onPurge(post: DeletedPost) {
    setBusyId(post.id)
    try {
      await purgeDeletedPost(post.id)
      setPosts((prev) => prev.filter((p) => p.id !== post.id))
      toast({ title: t('moderation.purged_toast') })
    } catch {
      toast({ title: t('admin.action_failed'), variant: 'destructive' })
    } finally {
      setBusyId(null)
      setPurgeTarget(null)
    }
  }

  // « Bientôt purgé » : la date de purge est à moins de ~30 jours (fenêtre de
  // préavis par défaut). Purement indicatif côté front.
  function isExpiringSoon(purgeAt: string): boolean {
    const ms = new Date(purgeAt).getTime() - Date.now()
    return ms <= 30 * 24 * 60 * 60 * 1000
  }

  return (
    <div className="flex flex-col gap-3 px-4 py-4">
      {/* ── Barre de filtres (serveur) ── */}
      <div className="flex flex-col gap-3">
        {/* Recherche par auteur → author_id */}
        {authorFilter ? (
          <div className="flex items-center gap-2">
            <Badge variant="secondary" className="flex items-center gap-1.5 py-1 pl-2.5 pr-1.5">
              {authorFilter.label}
              <button
                type="button"
                onClick={() => setAuthorFilter(null)}
                aria-label={t('common.clear')}
                className="rounded-full p-0.5 hover:bg-foreground/10"
              >
                <X className="h-3 w-3" aria-hidden />
              </button>
            </Badge>
          </div>
        ) : (
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" aria-hidden />
            <Input
              value={userQuery}
              onChange={(e) => setUserQuery(e.target.value)}
              placeholder={t('moderation.search_user')}
              className="pl-9"
              aria-label={t('moderation.search_user')}
            />
            {suggestions.length > 0 && (
              <ul className="panel absolute z-20 mt-1 max-h-64 w-full overflow-auto rounded-xl border p-1 shadow-md">
                {suggestions.map((u) => (
                  <li key={u.id}>
                    <button
                      type="button"
                      onClick={() => selectAuthor(u)}
                      className="flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-left hover:bg-primary/10"
                    >
                      <Avatar className="h-7 w-7 shrink-0">
                        {u.avatarUrl && <AvatarImage src={u.avatarUrl} alt="" />}
                        <AvatarFallback>{initialOf(u.displayName, u.username)}</AvatarFallback>
                      </Avatar>
                      <span className="flex min-w-0 flex-col">
                        <span className="truncate text-sm font-semibold">{u.displayName || u.username}</span>
                        {u.username && <span className="truncate text-xs text-muted-foreground">@{u.username}</span>}
                      </span>
                    </button>
                  </li>
                ))}
              </ul>
            )}
          </div>
        )}

        {/* Plage de date de retrait + réinitialisation */}
        <div className="flex flex-wrap items-end gap-3">
          <label className="flex flex-col gap-1 text-xs text-muted-foreground">
            {t('moderation.filter_since')}
            <Input type="date" value={since} max={until || undefined} onChange={(e) => setSince(e.target.value)} className="h-9 w-auto" />
          </label>
          <label className="flex flex-col gap-1 text-xs text-muted-foreground">
            {t('moderation.filter_until')}
            <Input type="date" value={until} min={since || undefined} onChange={(e) => setUntil(e.target.value)} className="h-9 w-auto" />
          </label>
          {hasFilters && (
            <Button variant="ghost" size="sm" onClick={resetFilters} className="text-muted-foreground">
              {t('moderation.filter_reset')}
            </Button>
          )}
        </div>
      </div>

      {/* ── Liste ── */}
      {loading ? (
        <div className="flex items-center justify-center py-16 text-muted-foreground">
          <Loader2 className="h-5 w-5 animate-spin" aria-hidden />
        </div>
      ) : error ? (
        <p className="py-16 text-center text-sm text-muted-foreground">{t('moderation.posts_error')}</p>
      ) : posts.length === 0 ? (
        <p className="py-16 text-center text-sm text-muted-foreground">
          {hasFilters ? t('moderation.no_match') : t('moderation.posts_empty')}
        </p>
      ) : (
        <>
          {posts.map((post) => {
            const author = people[post.authorId]
            const remover = people[post.hiddenBy]
            const busy = busyId === post.id
            const initial = initialOf(author?.displayName, author?.username)
            return (
              <article
                key={post.id}
                className="panel flex flex-col gap-2 rounded-2xl border p-3 shadow-sm"
              >
                <div className="flex items-center gap-2">
                  <ProfilLink author={{ id: post.authorId, username: author?.username ?? '' }} className="shrink-0">
                    <Avatar className="h-9 w-9">
                      {author?.avatarUrl && <AvatarImage src={author.avatarUrl} alt={author.displayName} />}
                      <AvatarFallback>{initial}</AvatarFallback>
                      <ActivityPresenceDot userId={post.authorId} />
                    </Avatar>
                  </ProfilLink>
                  <div className="flex min-w-0 flex-1 flex-col">
                    <span className="truncate text-sm font-bold">
                      {author?.displayName || author?.username || t('common.user')}
                    </span>
                    {author?.username && (
                      <span className="truncate text-xs text-muted-foreground">@{author.username}</span>
                    )}
                  </div>
                </div>

                {post.content && (
                  <p className="whitespace-pre-wrap break-words text-sm text-foreground">{post.content}</p>
                )}

                {post.media.length > 0 && (
                  <div className="flex flex-wrap gap-2">
                    {post.media.map((m, i) =>
                      m.type === 'image' ? (
                        // eslint-disable-next-line @next/next/no-img-element
                        <img
                          key={i}
                          src={m.url}
                          alt=""
                          className="h-20 w-20 rounded-lg object-cover"
                        />
                      ) : (
                        <video key={i} src={m.url} className="h-20 w-20 rounded-lg object-cover" />
                      ),
                    )}
                  </div>
                )}

                <p className="text-xs text-muted-foreground">
                  {t('moderation.removed_by', {
                    who: remover?.displayName || remover?.username || t('common.user'),
                    when: timeAgo(post.hiddenAt, locale),
                  })}
                </p>

                {post.purgeAt && (
                  <p className="flex items-center gap-2 text-xs text-muted-foreground">
                    <span>
                      {t('moderation.purge_scheduled', {
                        when: new Date(post.purgeAt).toLocaleDateString(locale === 'fr' ? 'fr-FR' : 'en-US'),
                      })}
                    </span>
                    {isExpiringSoon(post.purgeAt) && (
                      <Badge variant="destructive" className="text-[10px]">
                        {t('moderation.expiring_soon')}
                      </Badge>
                    )}
                  </p>
                )}

                <div className="flex justify-end gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={busy}
                    onClick={() => void onRestore(post)}
                  >
                    {busy ? (
                      <Loader2 className="h-4 w-4 animate-spin" aria-hidden />
                    ) : (
                      <RotateCcw className="h-4 w-4" aria-hidden />
                    )}
                    <span className="ml-1">{t('moderation.restore')}</span>
                  </Button>
                  <Button
                    variant="destructive"
                    size="sm"
                    disabled={busy}
                    onClick={() => setPurgeTarget(post)}
                  >
                    <Trash2 className="h-4 w-4" aria-hidden />
                    <span className="ml-1">{t('moderation.purge')}</span>
                  </Button>
                </div>
              </article>
            )
          })}

          {hasMore && (
            <Button variant="outline" size="sm" disabled={loadingMore} onClick={() => void loadMore()} className="self-center">
              {loadingMore ? <Loader2 className="h-4 w-4 animate-spin" aria-hidden /> : t('moderation.load_more')}
            </Button>
          )}
        </>
      )}

      {/* Confirmation de purge (irréversible). */}
      <Dialog open={purgeTarget !== null} onOpenChange={(open) => !open && setPurgeTarget(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('moderation.purge_title')}</DialogTitle>
            <DialogDescription>{t('moderation.purge_desc')}</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setPurgeTarget(null)}>
              {t('common.cancel')}
            </Button>
            <Button
              variant="destructive"
              disabled={busyId === purgeTarget?.id}
              onClick={() => purgeTarget && void onPurge(purgeTarget)}
            >
              {busyId === purgeTarget?.id ? (
                <Loader2 className="h-4 w-4 animate-spin" aria-hidden />
              ) : (
                t('moderation.purge')
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
