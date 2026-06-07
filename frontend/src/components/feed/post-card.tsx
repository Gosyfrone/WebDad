'use client'

import { useState } from 'react'
import {
  BarChart2,
  Heart,
  Loader2,
  MessageCircle,
  MoreHorizontal,
  Repeat2,
  Share,
  Trash2,
} from 'lucide-react'

import { cn, initialOf, timeAgo } from '@/lib/utils'
import { deletePost, likePost, unlikePost, type FeedPost } from '@/lib/posts'
import { useToast } from '@/hooks/use-toast'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { CommentSection } from '@/components/feed/comment-section'

interface PostCardProps {
  post: FeedPost
  /** Appelé après une suppression réussie (le parent retire le post du fil). */
  onDeleted?: (id: string) => void
}

/**
 * Carte d'un post : en-tête (auteur + horodatage + menu), contenu, barre
 * d'actions (commenter / liker / partager) et section commentaires repliable.
 *
 * Like et suppression sont câblés sur le post-service (optimistes + rollback).
 * Repost et vues restent décoratifs (pas d'API back).
 */
export function PostCard({ post, onDeleted }: PostCardProps) {
  const { toast } = useToast()

  const [liked, setLiked] = useState(post.liked)
  const [likeCount, setLikeCount] = useState(post.likesCount)
  const [commentCount, setCommentCount] = useState(post.commentsCount)
  const [showComments, setShowComments] = useState(false)
  const [deleting, setDeleting] = useState(false)

  // Décoratifs (pas de backend) : état purement local.
  const [reposted, setReposted] = useState(false)
  const [repostCount, setRepostCount] = useState(0)

  async function toggleLike() {
    const next = !liked
    // Optimiste.
    setLiked(next)
    setLikeCount((n) => n + (next ? 1 : -1))
    try {
      const count = next ? await likePost(post.id) : await unlikePost(post.id)
      setLikeCount(count) // reconcilie avec le compteur serveur
    } catch {
      // Rollback.
      setLiked(!next)
      setLikeCount((n) => n + (next ? -1 : 1))
      toast({ title: 'Action impossible', variant: 'destructive' })
    }
  }

  function toggleRepost() {
    setReposted((prev) => !prev)
    setRepostCount((prev) => (reposted ? prev - 1 : prev + 1))
  }

  async function handleDelete() {
    setDeleting(true)
    try {
      await deletePost(post.id)
      toast({ title: 'Post supprimé' })
      onDeleted?.(post.id)
    } catch {
      setDeleting(false)
      toast({ title: 'Suppression impossible', variant: 'destructive' })
    }
  }

  return (
    <article className="glass mx-3 my-3 flex gap-3 rounded-[24px] border px-4 py-3 backdrop-blur-xl transition hover:-translate-y-0.5 hover:bg-white/85 hover:shadow-[0_20px_56px_rgba(91,108,255,0.16)] dark:hover:bg-[#1f1633]/80">
      <Avatar className="mt-0.5 h-10 w-10 shrink-0">
        {post.author.avatarUrl && <AvatarImage src={post.author.avatarUrl} alt="" />}
        <AvatarFallback className="bg-gradient-to-br from-[#8D3DFF] via-[#5B6CFF] to-[#47D9FF] font-bold text-white">
          {initialOf(post.author.displayName)}
        </AvatarFallback>
      </Avatar>

      <div className="flex min-w-0 flex-1 flex-col gap-1">
        {/* Header */}
        <div className="flex items-center justify-between gap-2">
          <div className="flex min-w-0 items-center gap-1.5 text-sm">
            <span className="truncate font-bold text-foreground">{post.author.displayName}</span>
            {post.author.username && (
              <span className="shrink-0 text-muted-foreground">@{post.author.username}</span>
            )}
            <span className="shrink-0 text-muted-foreground">·</span>
            <span className="shrink-0 text-muted-foreground">{timeAgo(post.createdAt)}</span>
          </div>

          {post.canDelete && (
            <DropdownMenu>
              <DropdownMenuTrigger
                aria-label="Plus d'options"
                disabled={deleting}
                className="shrink-0 rounded-full p-1.5 text-muted-foreground transition-colors hover:bg-primary/10 hover:text-primary focus:outline-none"
              >
                {deleting ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <MoreHorizontal className="h-4 w-4" />
                )}
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem
                  onClick={handleDelete}
                  className="cursor-pointer text-red-500 focus:text-red-500"
                >
                  <Trash2 className="mr-2 h-4 w-4" />
                  Supprimer
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          )}
        </div>

        {/* Content */}
        <p className="whitespace-pre-wrap break-words text-sm leading-relaxed text-foreground/80">
          {post.content}
        </p>

        {/* Actions */}
        <div className="-ml-2 mt-1 flex items-center justify-between text-muted-foreground">
          <ActionButton
            icon={MessageCircle}
            count={commentCount}
            label="Commenter"
            active={showComments}
            onClick={() => setShowComments((v) => !v)}
            className="hover:text-primary hover:bg-primary/10"
            activeClassName="text-primary"
          />
          <ActionButton
            icon={Repeat2}
            count={repostCount}
            label="Reposter"
            active={reposted}
            onClick={toggleRepost}
            className="hover:text-green-500 hover:bg-green-500/10"
            activeClassName="text-green-500"
          />
          <ActionButton
            icon={Heart}
            count={likeCount}
            label="Aimer"
            active={liked}
            onClick={toggleLike}
            className="hover:text-red-500 hover:bg-red-500/10"
            activeClassName="text-red-500 fill-red-500"
          />
          <ActionButton
            icon={BarChart2}
            count={0}
            label="Vues"
            className="hover:text-primary hover:bg-primary/10"
          />
          <button
            aria-label="Partager"
            className="rounded-full p-1.5 transition-colors hover:bg-primary/10 hover:text-primary"
          >
            <Share className="h-4 w-4" />
          </button>
        </div>

        {/* Commentaires (repliable) */}
        {showComments && (
          <CommentSection
            postId={post.id}
            onCountChange={(delta) => setCommentCount((n) => Math.max(0, n + delta))}
          />
        )}
      </div>
    </article>
  )
}

interface ActionButtonProps {
  icon: React.ElementType
  count: number
  label: string
  active?: boolean
  onClick?: () => void
  className?: string
  activeClassName?: string
}

function ActionButton({
  icon: Icon,
  count,
  label,
  active = false,
  onClick,
  className,
  activeClassName,
}: ActionButtonProps) {
  return (
    <button
      aria-label={label}
      onClick={onClick}
      className={cn(
        'flex items-center gap-1 rounded-full p-1.5 text-xs transition-colors',
        className,
        active && activeClassName,
      )}
    >
      <Icon className={cn('h-4 w-4', active && activeClassName)} />
      {count > 0 && <span>{formatCount(count)}</span>}
    </button>
  )
}

function formatCount(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`
  return String(n)
}
