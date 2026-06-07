'use client'

import { useEffect, useRef, useState } from 'react'
import { Loader2, Trash2 } from 'lucide-react'

import { cn, initialOf, timeAgo } from '@/lib/utils'
import {
  createComment,
  deleteComment,
  listComments,
  type PostComment,
} from '@/lib/posts'
import { useToast } from '@/hooks/use-toast'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'

const MAX_CHARS = 280

interface CommentSectionProps {
  postId: string
  /** Notifie le parent d'une variation du nombre de commentaires (+1 / -1). */
  onCountChange?: (delta: number) => void
}

/**
 * Fil de commentaires repliable d'un post : mini-composer en tête + liste
 * chronologique. Chargé à la demande (au déploiement de la section).
 */
export function CommentSection({ postId, onCountChange }: CommentSectionProps) {
  const { toast } = useToast()
  const [comments, setComments] = useState<PostComment[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [content, setContent] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)

  const remaining = MAX_CHARS - content.length
  const canSubmit = content.trim().length > 0 && remaining >= 0 && !submitting

  useEffect(() => {
    let cancelled = false
    listComments(postId)
      .then((list) => {
        if (!cancelled) setComments(list)
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

  async function handleDelete(comment: PostComment) {
    const previous = comments
    setComments((prev) => prev.filter((c) => c.id !== comment.id))
    onCountChange?.(-1)
    try {
      await deleteComment(postId, comment.id)
    } catch {
      setComments(previous) // rollback
      onCountChange?.(1)
      toast({ title: 'Suppression impossible', variant: 'destructive' })
    }
  }

  return (
    <div className="mt-2 border-t border-border pt-3">
      {/* Mini-composer */}
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
          comments.map((c) => (
            <CommentRow key={c.id} comment={c} onDelete={() => handleDelete(c)} />
          ))
        )}
      </div>
    </div>
  )
}

function CommentRow({
  comment,
  onDelete,
}: {
  comment: PostComment
  onDelete: () => void
}) {
  return (
    <div className="group flex gap-2">
      <Avatar className="h-8 w-8 shrink-0">
        {comment.author.avatarUrl && <AvatarImage src={comment.author.avatarUrl} alt="" />}
        <AvatarFallback className="bg-gradient-to-br from-[#8D3DFF] via-[#5B6CFF] to-[#47D9FF] text-xs font-bold text-white">
          {initialOf(comment.author.displayName)}
        </AvatarFallback>
      </Avatar>

      <div className="flex min-w-0 flex-1 flex-col">
        <div className="flex items-center gap-1.5 text-xs">
          <span className="truncate font-bold text-foreground">{comment.author.displayName}</span>
          {comment.author.username && (
            <span className="shrink-0 text-muted-foreground">@{comment.author.username}</span>
          )}
          <span className="shrink-0 text-muted-foreground">·</span>
          <span className="shrink-0 text-muted-foreground">{timeAgo(comment.createdAt)}</span>
        </div>
        <p className="whitespace-pre-wrap break-words text-sm text-foreground/85">{comment.content}</p>
      </div>

      {comment.canDelete && (
        <button
          aria-label="Supprimer le commentaire"
          onClick={onDelete}
          className={cn(
            'shrink-0 rounded-full p-1.5 text-muted-foreground transition-colors',
            'hover:bg-red-500/10 hover:text-red-500 lg:opacity-0 lg:group-hover:opacity-100',
          )}
        >
          <Trash2 className="h-3.5 w-3.5" />
        </button>
      )}
    </div>
  )
}
