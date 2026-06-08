'use client'

import { useCallback, useEffect, useRef, useState } from 'react'
import { Loader2, Trash2 } from 'lucide-react'

import { cn, initialOf, timeAgo } from '@/lib/utils'
import { useInfiniteScroll } from '@/lib/use-infinite-scroll'
import {
  createComment,
  deleteComment,
  listComments,
  listReplies,
  type PostComment,
} from '@/lib/posts'
import { useToast } from '@/hooks/use-toast'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import { TranslatedContent } from '@/components/feed/translated-content'
import { ProfilLink } from '@/components/profil/profil-link'

const MAX_CHARS = 280
const COMMENTS_PAGE = 10
const REPLIES_PAGE = 6

/** Fusionne une nouvelle page en dédupliquant par id (anti double-chargement). */
function mergeUnique(current: PostComment[], incoming: PostComment[]): PostComment[] {
  const seen = new Set(current.map((c) => c.id))
  return [...current, ...incoming.filter((c) => !seen.has(c.id))]
}

interface CommentSectionProps {
  postId: string
  /** Notifie le parent d'une variation du nombre de commentaires (+1 / -N). */
  onCountChange?: (delta: number) => void
}

/**
 * Fil de commentaires repliable d'un post : composer racine + liste paginée de
 * commentaires racine (« Voir plus de commentaires »). Chaque commentaire gère
 * ses propres réponses (threading à 2 niveaux, cf. CommentThread).
 */
export function CommentSection({ postId, onCountChange }: CommentSectionProps) {
  const { toast } = useToast()
  const [comments, setComments] = useState<PostComment[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [hasMore, setHasMore] = useState(false)
  const [loadingMore, setLoadingMore] = useState(false)
  const [content, setContent] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)
  // Offset de pagination = nb chargé depuis le serveur (indépendant des
  // insertions/suppressions locales).
  const offsetRef = useRef(0)

  const remaining = MAX_CHARS - content.length
  const canSubmit = content.trim().length > 0 && remaining >= 0 && !submitting

  useEffect(() => {
    let cancelled = false
    listComments(postId, COMMENTS_PAGE, 0)
      .then((list) => {
        if (cancelled) return
        setComments(list)
        offsetRef.current = list.length
        setHasMore(list.length === COMMENTS_PAGE)
      })
      .catch(() => {
        if (!cancelled) setError('Impossible de charger les commentaires.')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [postId])

  const loadMore = useCallback(async () => {
    setLoadingMore(true)
    try {
      const next = await listComments(postId, COMMENTS_PAGE, offsetRef.current)
      offsetRef.current += next.length
      setComments((prev) => mergeUnique(prev, next))
      setHasMore(next.length === COMMENTS_PAGE)
    } catch {
      setHasMore(false)
    } finally {
      setLoadingMore(false)
    }
  }, [postId])

  const sentinelRef = useInfiniteScroll(loadMore, {
    hasMore,
    loading: loading || loadingMore,
  })

  async function handleSubmit() {
    if (!canSubmit) return
    setSubmitting(true)
    try {
      const created = await createComment(postId, content.trim())
      setComments((prev) => [...prev, created])
      setContent('')
      onCountChange?.(1)
      inputRef.current?.focus()
    } catch {
      toast({ title: 'Commentaire impossible', variant: 'destructive' })
    } finally {
      setSubmitting(false)
    }
  }

  function handleRemoveRoot(id: string) {
    setComments((prev) => prev.filter((c) => c.id !== id))
  }

  return (
    <div className="mt-2 border-t border-border pt-3">
      {/* Composer racine */}
      <div className="flex items-center gap-2">
        <input
          ref={inputRef}
          value={content}
          onChange={(e) => setContent(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && !e.shiftKey) {
              e.preventDefault()
              void handleSubmit()
            }
          }}
          placeholder="Écrire un commentaire…"
          maxLength={MAX_CHARS + 20}
          className="min-w-0 flex-1 rounded-full border border-border bg-background/60 px-4 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:border-[#5B6CFF] focus:outline-none"
        />
        <Button
          size="sm"
          className="shrink-0 rounded-full bg-gradient-to-r from-[#8D3DFF] via-[#5B6CFF] to-[#47D9FF] font-bold text-white"
          disabled={!canSubmit}
          onClick={handleSubmit}
        >
          {submitting ? <Loader2 className="h-4 w-4 animate-spin" /> : 'Répondre'}
        </Button>
      </div>

      {/* Liste */}
      <div className="mt-3 flex flex-col gap-3">
        {loading ? (
          <div className="flex justify-center py-4">
            <Loader2 className="h-4 w-4 animate-spin text-[#5B6CFF]" aria-hidden />
          </div>
        ) : error ? (
          <p className="py-2 text-center text-xs text-muted-foreground">{error}</p>
        ) : comments.length === 0 ? (
          <p className="py-2 text-center text-xs text-muted-foreground">
            Aucun commentaire. Soyez le premier à réagir.
          </p>
        ) : (
          <>
            {comments.map((c) => (
              <CommentThread
                key={c.id}
                postId={postId}
                comment={c}
                onRemove={handleRemoveRoot}
                onCountChange={onCountChange}
              />
            ))}
            {/* Sentinelle de défilement infini (commentaires racine). */}
            {hasMore && (
              <div ref={sentinelRef} className="flex justify-center py-2">
                {loadingMore && (
                  <Loader2 className="h-4 w-4 animate-spin text-[#5B6CFF]" aria-hidden />
                )}
              </div>
            )}
          </>
        )}
      </div>
    </div>
  )
}

interface CommentThreadProps {
  postId: string
  comment: PostComment
  onRemove: (id: string) => void
  onCountChange?: (delta: number) => void
}

/**
 * Un commentaire racine + ses réponses (repliées par défaut, indentées).
 * Réponses paginées (« Voir plus de réponses ») ; composer de réponse inline.
 */
function CommentThread({ postId, comment, onRemove, onCountChange }: CommentThreadProps) {
  const { toast } = useToast()
  const [replies, setReplies] = useState<PostComment[]>([])
  const [replyCount, setReplyCount] = useState(comment.replyCount)
  const [open, setOpen] = useState(false)
  const [loading, setLoading] = useState(false)

  const [composerOpen, setComposerOpen] = useState(false)
  const [target, setTarget] = useState<{ id: string; username: string }>({
    id: comment.id,
    username: comment.author.username,
  })
  const [content, setContent] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const replyInputRef = useRef<HTMLInputElement>(null)

  const hasMoreReplies = replies.length < replyCount
  const remaining = MAX_CHARS - content.length
  const canSubmit = content.trim().length > 0 && remaining >= 0 && !submitting

  async function loadReplies(offset: number) {
    setLoading(true)
    try {
      const next = await listReplies(postId, comment.id, REPLIES_PAGE, offset)
      setReplies((prev) => mergeUnique(prev, next))
    } catch {
      toast({ title: 'Chargement impossible', variant: 'destructive' })
    } finally {
      setLoading(false)
    }
  }

  async function toggleReplies() {
    if (!open && replies.length === 0 && replyCount > 0) {
      await loadReplies(0)
    }
    setOpen((v) => !v)
  }

  function openReplyTo(t: { id: string; username: string }) {
    setTarget(t)
    setContent(t.username ? `@${t.username} ` : '')
    setComposerOpen(true)
    requestAnimationFrame(() => replyInputRef.current?.focus())
  }

  async function submitReply() {
    if (!canSubmit) return
    setSubmitting(true)
    try {
      const created = await createComment(postId, content.trim(), target.id)
      setReplies((prev) => [...prev, created])
      setReplyCount((n) => n + 1)
      setOpen(true)
      setContent('')
      setComposerOpen(false)
      onCountChange?.(1)
    } catch {
      toast({ title: 'Réponse impossible', variant: 'destructive' })
    } finally {
      setSubmitting(false)
    }
  }

  async function deleteRoot() {
    try {
      await deleteComment(postId, comment.id)
      onCountChange?.(-(1 + replyCount)) // racine + ses réponses (cascade back)
      onRemove(comment.id)
    } catch {
      toast({ title: 'Suppression impossible', variant: 'destructive' })
    }
  }

  async function deleteReply(reply: PostComment) {
    const snapshot = replies
    setReplies((prev) => prev.filter((r) => r.id !== reply.id))
    setReplyCount((n) => Math.max(0, n - 1))
    onCountChange?.(-1)
    try {
      await deleteComment(postId, reply.id)
    } catch {
      setReplies(snapshot) // rollback
      setReplyCount((n) => n + 1)
      onCountChange?.(1)
      toast({ title: 'Suppression impossible', variant: 'destructive' })
    }
  }

  return (
    <div className="flex flex-col">
      <CommentRow
        comment={comment}
        onDelete={deleteRoot}
        footer={
          <div className="mt-1 flex items-center gap-3 text-xs font-semibold text-muted-foreground">
            <button
              className="transition-colors hover:text-[#5B6CFF]"
              onClick={() => openReplyTo({ id: comment.id, username: comment.author.username })}
            >
              Répondre
            </button>
            {replyCount > 0 && (
              <button className="transition-colors hover:text-[#5B6CFF]" onClick={toggleReplies}>
                {open
                  ? 'Masquer les réponses'
                  : `Voir les ${replyCount} réponse${replyCount > 1 ? 's' : ''}`}
              </button>
            )}
          </div>
        }
      />

      {/* Réponses indentées */}
      {(open || composerOpen) && (
        <div className="ml-5 mt-2 flex flex-col gap-3 border-l border-border pl-3">
          {composerOpen && (
            <div className="flex items-center gap-2">
              <input
                ref={replyInputRef}
                value={content}
                onChange={(e) => setContent(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' && !e.shiftKey) {
                    e.preventDefault()
                    void submitReply()
                  }
                }}
                placeholder="Écrire une réponse…"
                maxLength={MAX_CHARS + 20}
                className="min-w-0 flex-1 rounded-full border border-border bg-background/60 px-4 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:border-[#5B6CFF] focus:outline-none"
              />
              <Button
                size="sm"
                className="shrink-0 rounded-full bg-gradient-to-r from-[#8D3DFF] via-[#5B6CFF] to-[#47D9FF] font-bold text-white"
                disabled={!canSubmit}
                onClick={submitReply}
              >
                {submitting ? <Loader2 className="h-4 w-4 animate-spin" /> : 'Répondre'}
              </Button>
            </div>
          )}

          {open &&
            replies.map((r) => (
              <CommentRow
                key={r.id}
                comment={r}
                onDelete={() => deleteReply(r)}
                footer={
                  <div className="mt-1 text-xs font-semibold text-muted-foreground">
                    <button
                      className="transition-colors hover:text-[#5B6CFF]"
                      onClick={() => openReplyTo({ id: r.id, username: r.author.username })}
                    >
                      Répondre
                    </button>
                  </div>
                }
              />
            ))}

          {open && hasMoreReplies && (
            <LoadMoreButton loading={loading} onClick={() => loadReplies(replies.length)}>
              Voir plus de réponses
            </LoadMoreButton>
          )}
        </div>
      )}
    </div>
  )
}

function CommentRow({
  comment,
  onDelete,
  footer,
}: {
  comment: PostComment
  onDelete: () => void
  footer?: React.ReactNode
}) {
  return (
    <div className="group flex gap-2">
      <ProfilLink author={comment.author} className="shrink-0 transition hover:opacity-90">
        <Avatar className="h-8 w-8">
          {comment.author.avatarUrl && <AvatarImage src={comment.author.avatarUrl} alt="" />}
          <AvatarFallback className="bg-gradient-to-br from-[#8D3DFF] via-[#5B6CFF] to-[#47D9FF] text-xs font-bold text-white">
            {initialOf(comment.author.displayName)}
          </AvatarFallback>
        </Avatar>
      </ProfilLink>

      <div className="flex min-w-0 flex-1 flex-col">
        <div className="flex items-center gap-1.5 text-xs">
          <ProfilLink
            author={comment.author}
            className="truncate font-bold text-foreground hover:underline"
          >
            {comment.author.displayName}
          </ProfilLink>
          {comment.author.username && (
            <ProfilLink
              author={comment.author}
              className="shrink-0 text-muted-foreground hover:underline"
            >
              @{comment.author.username}
            </ProfilLink>
          )}
          <span className="shrink-0 text-muted-foreground">·</span>
          <span className="shrink-0 text-muted-foreground">{timeAgo(comment.createdAt)}</span>
        </div>
        <TranslatedContent
          contentId={`comment:${comment.id}`}
          content={comment.content}
          className="whitespace-pre-wrap break-words text-sm text-foreground/85"
          indicatorClassName="min-h-5 text-[11px]"
        />
        {footer}
      </div>

      {comment.canDelete && (
        <button
          aria-label="Supprimer le commentaire"
          onClick={onDelete}
          className={cn(
            'h-fit shrink-0 rounded-full p-1.5 text-muted-foreground transition-colors',
            'hover:bg-red-500/10 hover:text-red-500 lg:opacity-0 lg:group-hover:opacity-100',
          )}
        >
          <Trash2 className="h-3.5 w-3.5" />
        </button>
      )}
    </div>
  )
}

function LoadMoreButton({
  loading,
  onClick,
  children,
}: {
  loading: boolean
  onClick: () => void
  children: React.ReactNode
}) {
  return (
    <button
      onClick={onClick}
      disabled={loading}
      className="flex items-center justify-center gap-2 self-start rounded-full px-3 py-1 text-xs font-semibold text-[#5B6CFF] transition-colors hover:bg-primary/10 disabled:opacity-60 dark:text-[#9aa6ff]"
    >
      {loading && <Loader2 className="h-3.5 w-3.5 animate-spin" />}
      {children}
    </button>
  )
}
