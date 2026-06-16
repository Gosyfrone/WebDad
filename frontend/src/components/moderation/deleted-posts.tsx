'use client'

import { useCallback, useEffect, useState } from 'react'
import { Loader2, RotateCcw, Trash2 } from 'lucide-react'

import {
  listDeletedPosts,
  restoreDeletedPost,
  purgeDeletedPost,
  type DeletedPost,
} from '@/lib/moderation'
import { resolveUser, type ResolvedUser } from '@/lib/user-cache'
import { timeAgo } from '@/lib/utils'
import { ProfilLink } from '@/components/profil/profil-link'
import { ActivityPresenceDot } from '@/components/profil/activity-presence-dot'
import { useLanguage } from '@/components/language-provider'
import { useToast } from '@/hooks/use-toast'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

/**
 * Corbeille de modération : les tweets retirés en suppression douce (partagée
 * mod + admin). Pour chaque post : auteur + modérateur ayant retiré + date,
 * aperçu du contenu/média, et deux actions :
 *   - Restaurer (réversible, sans confirmation) ;
 *   - Supprimer définitivement (irréversible → confirmation, car la purge ne se
 *     rejoue pas, contrairement au retrait initial).
 */
export function DeletedPosts() {
  const { t, locale } = useLanguage()
  const { toast } = useToast()
  const [posts, setPosts] = useState<DeletedPost[]>([])
  const [people, setPeople] = useState<Record<string, ResolvedUser>>({})
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(false)
  const [busyId, setBusyId] = useState<string | null>(null)
  const [purgeTarget, setPurgeTarget] = useState<DeletedPost | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(false)
    try {
      const list = await listDeletedPosts()
      setPosts(list)
      // Résout (mémoïsé) auteurs + modérateurs en une passe.
      const ids = Array.from(
        new Set(list.flatMap((p) => [p.authorId, p.hiddenBy]).filter(Boolean)),
      )
      const resolved = await Promise.all(ids.map((id) => resolveUser(id)))
      setPeople(Object.fromEntries(ids.map((id, i) => [id, resolved[i]])))
    } catch {
      setError(true)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

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

  if (loading) {
    return (
      <div className="flex items-center justify-center py-16 text-muted-foreground">
        <Loader2 className="h-5 w-5 animate-spin" aria-hidden />
      </div>
    )
  }
  if (error) {
    return <p className="py-16 text-center text-sm text-muted-foreground">{t('moderation.posts_error')}</p>
  }
  if (posts.length === 0) {
    return <p className="py-16 text-center text-sm text-muted-foreground">{t('moderation.posts_empty')}</p>
  }

  return (
    <div className="flex flex-col gap-2 px-4 py-4">
      {posts.map((post) => {
        const author = people[post.authorId]
        const remover = people[post.hiddenBy]
        const busy = busyId === post.id
        const initial = (author?.displayName || author?.username || 'U').charAt(0).toUpperCase()
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
