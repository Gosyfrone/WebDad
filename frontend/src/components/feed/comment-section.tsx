'use client'

import { useCallback, useEffect, useRef, useState } from 'react'
import { Image as ImageIcon, Loader2, Smile, Trash2, X } from 'lucide-react'

import { cn, initialOf, timeAgo } from '@/lib/utils'
import { getAccessToken } from '@/lib/auth-client'
import { useInfiniteScroll } from '@/lib/use-infinite-scroll'
import { resolveMediaUrl, uploadMedia } from '@/lib/media'
import {
  createComment,
  deleteComment,
  listComments,
  listReplies,
  type PostComment,
  type PostMedia,
} from '@/lib/posts'
import { useToast } from '@/hooks/use-toast'
import { useMention } from '@/lib/use-mention'
import { mentionSearchGlobal } from '@/lib/mention-search'
import { useAuthGate } from '@/components/auth-prompt-provider'
import { useLanguage } from '@/components/language-provider'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import { EmojiPicker } from '@/components/feed/emoji-picker'
import { MentionAutocomplete } from '@/components/mention/mention-autocomplete'
import { TranslatedContent } from '@/components/feed/translated-content'
import { ProfilLink } from '@/components/profil/profil-link'

const MAX_CHARS = 280
const MAX_MEDIA = 4
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
  const { t } = useLanguage()
  const { isVisitor, promptLogin } = useAuthGate()
  const [comments, setComments] = useState<PostComment[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [hasMore, setHasMore] = useState(false)
  const [loadingMore, setLoadingMore] = useState(false)
  const [content, setContent] = useState('')
  const [media, setMedia] = useState<PostMedia[]>([])
  const [uploadingMedia, setUploadingMedia] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const mention = useMention({
    inputRef,
    onChange: setContent,
    search: mentionSearchGlobal,
  })
  // Offset de pagination = nb chargé depuis le serveur (indépendant des
  // insertions/suppressions locales).
  const offsetRef = useRef(0)

  const remaining = MAX_CHARS - content.length
  const canSubmit = (content.trim().length > 0 || media.length > 0) && remaining >= 0 && !submitting && !uploadingMedia

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
        if (!cancelled) setError(t('comment.load_error'))
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [postId, t])

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
      const created = await createComment(postId, content.trim(), undefined, media)
      setComments((prev) => [...prev, created])
      setContent('')
      setMedia([])
      onCountChange?.(1)
      inputRef.current?.focus()
    } catch {
      toast({ title: t('comment.submit_failed'), variant: 'destructive' })
    } finally {
      setSubmitting(false)
    }
  }

  async function handleFiles(e: React.ChangeEvent<HTMLInputElement>) {
    const files = Array.from(e.target.files ?? [])
    e.target.value = ''
    if (files.length === 0) return

    const room = MAX_MEDIA - media.length
    if (room <= 0) {
      toast({ title: t('composer.media_max', { count: MAX_MEDIA }), variant: 'destructive' })
      return
    }

    setUploadingMedia(true)
    try {
      const uploaded = await Promise.all(
        files.slice(0, room).map(async (file) => {
          const { url, kind } = await uploadMedia(file)
          return { url, type: kind } as PostMedia
        }),
      )
      setMedia((prev) => [...prev, ...uploaded])
    } catch {
      toast({ title: t('composer.media_failed'), variant: 'destructive' })
    } finally {
      setUploadingMedia(false)
    }
  }

  function removeMedia(index: number) {
    setMedia((prev) => prev.filter((_, i) => i !== index))
  }

  function insertEmoji(emoji: string) {
    const el = inputRef.current
    const start = el?.selectionStart ?? content.length
    const end = el?.selectionEnd ?? content.length
    setContent(content.slice(0, start) + emoji + content.slice(end))
    requestAnimationFrame(() => {
      if (!el) return
      const caret = start + emoji.length
      el.focus()
      el.setSelectionRange(caret, caret)
    })
  }

  function handleRemoveRoot(id: string) {
    setComments((prev) => prev.filter((c) => c.id !== id))
  }

  return (
    <div className="mt-2 border-t border-border pt-3">
      {/* Composer racine — remplacé par une invite de connexion pour le visiteur. */}
      {isVisitor ? (
        <button
          type="button"
          onClick={promptLogin}
          className="w-full rounded-full border border-border bg-background/60 px-4 py-2 text-left text-sm text-muted-foreground transition-colors hover:bg-accent"
        >
          {t('comment.placeholder')}
        </button>
      ) : (
      <div className="flex flex-col gap-2">
        {media.length > 0 && (
          <CommentMediaPreviews media={media} onRemove={removeMedia} removeLabel={t('composer.media_remove')} />
        )}
        <div className="flex items-center gap-2">
          <div className="relative min-w-0 flex-1">
            <input
              ref={inputRef}
              value={content}
              onChange={(e) => {
                setContent(e.target.value)
                mention.sync()
              }}
              onKeyUp={mention.sync}
              onClick={mention.sync}
              onKeyDown={(e) => {
                mention.onKeyDown(e)
                if (e.defaultPrevented) return
                if (e.key === 'Enter' && !e.shiftKey) {
                  e.preventDefault()
                  void handleSubmit()
                }
              }}
              placeholder={t('comment.placeholder')}
              maxLength={MAX_CHARS + 20}
              className="w-full rounded-full border border-border bg-background/60 px-4 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:border-[#5B6CFF] focus:outline-none"
            />
            <MentionAutocomplete controller={mention} placement="top" />
          </div>
          <button
            type="button"
            aria-label={t('composer.add_image')}
            onClick={() => fileInputRef.current?.click()}
            disabled={uploadingMedia || media.length >= MAX_MEDIA}
            className="shrink-0 rounded-full p-2 text-[#5B6CFF] transition-colors hover:bg-primary/10 disabled:opacity-40"
          >
            {uploadingMedia ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <ImageIcon className="h-4 w-4" />
            )}
          </button>
          <input
            ref={fileInputRef}
            type="file"
            accept="image/*,video/*"
            multiple
            onChange={handleFiles}
            className="sr-only"
          />
          <EmojiPicker onSelect={insertEmoji}>
            <button
              type="button"
              aria-label={t('composer.add_emoji')}
              className="shrink-0 rounded-full p-2 text-[#5B6CFF] transition-colors hover:bg-primary/10"
            >
              <Smile className="h-4 w-4" />
            </button>
          </EmojiPicker>
          <Button
            size="sm"
            className="shrink-0 rounded-full bg-gradient-to-r from-[var(--brand-from)] via-[var(--brand-via)] to-[var(--brand-to)] font-bold text-white"
            disabled={!canSubmit}
            onClick={handleSubmit}
          >
            {submitting ? <Loader2 className="h-4 w-4 animate-spin" /> : t('comment.reply')}
          </Button>
        </div>
      </div>
      )}

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
            {t('comment.empty')}
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
  const { t } = useLanguage()
  const { promptLogin } = useAuthGate()
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
  const [media, setMedia] = useState<PostMedia[]>([])
  const [uploadingMedia, setUploadingMedia] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const replyInputRef = useRef<HTMLInputElement>(null)
  const replyFileInputRef = useRef<HTMLInputElement>(null)
  const mention = useMention({
    inputRef: replyInputRef,
    onChange: setContent,
    search: mentionSearchGlobal,
  })

  const hasMoreReplies = replies.length < replyCount
  const remaining = MAX_CHARS - content.length
  const canSubmit = (content.trim().length > 0 || media.length > 0) && remaining >= 0 && !submitting && !uploadingMedia

  async function loadReplies(offset: number) {
    setLoading(true)
    try {
      const next = await listReplies(postId, comment.id, REPLIES_PAGE, offset)
      setReplies((prev) => mergeUnique(prev, next))
    } catch {
      toast({ title: t('comment.load_failed'), variant: 'destructive' })
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

  function openReplyTo(to: { id: string; username: string }) {
    // Visiteur : invite à se connecter plutôt que d'ouvrir le composer de réponse.
    if (!getAccessToken()) {
      promptLogin()
      return
    }
    setTarget(to)
    setContent(to.username ? `@${to.username} ` : '')
    setComposerOpen(true)
    requestAnimationFrame(() => replyInputRef.current?.focus())
  }

  async function submitReply() {
    if (!canSubmit) return
    setSubmitting(true)
    try {
      const created = await createComment(postId, content.trim(), target.id, media)
      setReplies((prev) => [...prev, created])
      setReplyCount((n) => n + 1)
      setOpen(true)
      setContent('')
      setMedia([])
      setComposerOpen(false)
      onCountChange?.(1)
    } catch {
      toast({ title: t('comment.reply_failed'), variant: 'destructive' })
    } finally {
      setSubmitting(false)
    }
  }

  async function handleReplyFiles(e: React.ChangeEvent<HTMLInputElement>) {
    const files = Array.from(e.target.files ?? [])
    e.target.value = ''
    if (files.length === 0) return

    const room = MAX_MEDIA - media.length
    if (room <= 0) {
      toast({ title: t('composer.media_max', { count: MAX_MEDIA }), variant: 'destructive' })
      return
    }

    setUploadingMedia(true)
    try {
      const uploaded = await Promise.all(
        files.slice(0, room).map(async (file) => {
          const { url, kind } = await uploadMedia(file)
          return { url, type: kind } as PostMedia
        }),
      )
      setMedia((prev) => [...prev, ...uploaded])
    } catch {
      toast({ title: t('composer.media_failed'), variant: 'destructive' })
    } finally {
      setUploadingMedia(false)
    }
  }

  function removeReplyMedia(index: number) {
    setMedia((prev) => prev.filter((_, i) => i !== index))
  }

  function insertReplyEmoji(emoji: string) {
    const el = replyInputRef.current
    const start = el?.selectionStart ?? content.length
    const end = el?.selectionEnd ?? content.length
    setContent(content.slice(0, start) + emoji + content.slice(end))
    requestAnimationFrame(() => {
      if (!el) return
      const caret = start + emoji.length
      el.focus()
      el.setSelectionRange(caret, caret)
    })
  }

  async function deleteRoot() {
    try {
      await deleteComment(postId, comment.id)
      onCountChange?.(-(1 + replyCount)) // racine + ses réponses (cascade back)
      onRemove(comment.id)
    } catch {
      toast({ title: t('common.delete_failed'), variant: 'destructive' })
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
      toast({ title: t('common.delete_failed'), variant: 'destructive' })
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
              {t('comment.reply')}
            </button>
            {replyCount > 0 && (
              <button className="transition-colors hover:text-[#5B6CFF]" onClick={toggleReplies}>
                {open
                  ? t('comment.hide_replies')
                  : t(
                      replyCount > 1 ? 'comment.view_replies_other' : 'comment.view_replies_one',
                      { count: replyCount },
                    )}
              </button>
            )}
          </div>
        }
      />

      {/* Réponses indentées */}
      {(open || composerOpen) && (
        <div className="ml-5 mt-2 flex flex-col gap-3 border-l border-border pl-3">
          {composerOpen && (
            <div className="flex flex-col gap-2">
              {media.length > 0 && (
                <CommentMediaPreviews
                  media={media}
                  onRemove={removeReplyMedia}
                  removeLabel={t('composer.media_remove')}
                />
              )}
              <div className="flex items-center gap-2">
                <div className="relative min-w-0 flex-1">
                  <input
                    ref={replyInputRef}
                    value={content}
                    onChange={(e) => {
                      setContent(e.target.value)
                      mention.sync()
                    }}
                    onKeyUp={mention.sync}
                    onClick={mention.sync}
                    onKeyDown={(e) => {
                      mention.onKeyDown(e)
                      if (e.defaultPrevented) return
                      if (e.key === 'Enter' && !e.shiftKey) {
                        e.preventDefault()
                        void submitReply()
                      }
                    }}
                    placeholder={t('comment.reply_placeholder')}
                    maxLength={MAX_CHARS + 20}
                    className="w-full rounded-full border border-border bg-background/60 px-4 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:border-[#5B6CFF] focus:outline-none"
                  />
                  <MentionAutocomplete controller={mention} placement="top" />
                </div>
                <button
                  type="button"
                  aria-label={t('composer.add_image')}
                  onClick={() => replyFileInputRef.current?.click()}
                  disabled={uploadingMedia || media.length >= MAX_MEDIA}
                  className="shrink-0 rounded-full p-2 text-[#5B6CFF] transition-colors hover:bg-primary/10 disabled:opacity-40"
                >
                  {uploadingMedia ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <ImageIcon className="h-4 w-4" />
                  )}
                </button>
                <input
                  ref={replyFileInputRef}
                  type="file"
                  accept="image/*,video/*"
                  multiple
                  onChange={handleReplyFiles}
                  className="sr-only"
                />
                <EmojiPicker onSelect={insertReplyEmoji}>
                  <button
                    type="button"
                    aria-label={t('composer.add_emoji')}
                    className="shrink-0 rounded-full p-2 text-[#5B6CFF] transition-colors hover:bg-primary/10"
                  >
                    <Smile className="h-4 w-4" />
                  </button>
                </EmojiPicker>
                <Button
                  size="sm"
                  className="shrink-0 rounded-full bg-gradient-to-r from-[var(--brand-from)] via-[var(--brand-via)] to-[var(--brand-to)] font-bold text-white"
                  disabled={!canSubmit}
                  onClick={submitReply}
                >
                  {submitting ? <Loader2 className="h-4 w-4 animate-spin" /> : t('comment.reply')}
                </Button>
              </div>
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
                      {t('comment.reply')}
                    </button>
                  </div>
                }
              />
            ))}

          {open && hasMoreReplies && (
            <LoadMoreButton loading={loading} onClick={() => loadReplies(replies.length)}>
              {t('comment.view_more_replies')}
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
  const { t, locale } = useLanguage()
  return (
    <div className="group flex gap-2">
      <ProfilLink author={comment.author} className="shrink-0 transition hover:opacity-90">
        <Avatar className="h-8 w-8">
          {comment.author.avatarUrl && <AvatarImage src={comment.author.avatarUrl} alt="" />}
          <AvatarFallback className="bg-gradient-to-br from-[var(--brand-from)] via-[var(--brand-via)] to-[var(--brand-to)] text-xs font-bold text-white">
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
          <span className="shrink-0 text-muted-foreground">{timeAgo(comment.createdAt, locale)}</span>
        </div>
        {comment.content && (
          <TranslatedContent
            contentId={`comment:${comment.id}`}
            content={comment.content}
            className="whitespace-pre-wrap break-words text-sm text-foreground/85"
            indicatorClassName="min-h-5 text-[11px]"
          />
        )}
        {comment.media.length > 0 && <CommentMediaGallery media={comment.media} />}
        {footer}
      </div>

      {comment.canDelete && (
        <button
          aria-label={t('comment.delete_aria')}
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

function CommentMediaPreviews({
  media,
  onRemove,
  removeLabel,
}: {
  media: PostMedia[]
  onRemove: (index: number) => void
  removeLabel: string
}) {
  return (
    <div className={cn('grid gap-2', media.length > 1 ? 'grid-cols-2' : 'grid-cols-1')}>
      {media.map((m, i) => (
        <div key={m.url} className="group relative overflow-hidden rounded-xl border border-border bg-background/45">
          {m.type === 'video' ? (
            <video src={resolveMediaUrl(m.url)} className="max-h-52 w-full object-cover" muted playsInline />
          ) : (
            // eslint-disable-next-line @next/next/no-img-element
            <img src={resolveMediaUrl(m.url)} alt="" className="max-h-52 w-full object-cover" />
          )}
          <button
            type="button"
            aria-label={removeLabel}
            onClick={() => onRemove(i)}
            className="absolute right-1.5 top-1.5 rounded-full bg-black/60 p-1 text-white transition hover:bg-black/80"
          >
            <X className="h-4 w-4" />
          </button>
        </div>
      ))}
    </div>
  )
}

function CommentMediaGallery({ media }: { media: PostMedia[] }) {
  return (
    <div
      className={cn(
        'mt-2 grid max-w-md gap-1.5 overflow-hidden rounded-xl border border-border',
        media.length > 1 ? 'grid-cols-2' : 'grid-cols-1',
      )}
    >
      {media.map((m, i) => {
        const cellClass = cn(
          media.length === 1 ? 'max-h-72' : 'aspect-square',
          media.length === 3 && i === 0 && 'row-span-2 aspect-auto',
        )
        return (
          <div
            key={m.url}
            className={cn('overflow-hidden bg-background/50', media.length === 3 && i === 0 && 'row-span-2')}
          >
            {m.type === 'video' ? (
              <video
                src={m.url}
                className={cn('h-full w-full object-cover', media.length === 1 ? 'max-h-72 aspect-video' : cellClass)}
                controls
                playsInline
              />
            ) : (
              // eslint-disable-next-line @next/next/no-img-element
              <img src={m.url} alt="" loading="lazy" className={cn('h-full w-full object-cover', cellClass)} />
            )}
          </div>
        )
      })}
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
