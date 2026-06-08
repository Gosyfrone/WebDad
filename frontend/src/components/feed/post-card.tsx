'use client'

import { useEffect, useState } from 'react'
import {
  BarChart2,
  Heart,
  Loader2,
  MessageCircle,
  MoreHorizontal,
  Pin,
  PinOff,
  Repeat2,
  Share,
  Trash2,
} from 'lucide-react'

import { cn, initialOf, timeAgo } from '@/lib/utils'
import { deletePost, likePost, pinPost, unlikePost, unpinPost, type FeedPost } from '@/lib/posts'
import { useToast } from '@/hooks/use-toast'
import { useLanguage } from '@/components/language-provider'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { CommentSection } from '@/components/feed/comment-section'
import { TranslatedContent } from '@/components/feed/translated-content'
import { ProfilLink } from '@/components/profil/profil-link'

interface PostCardProps {
  post: FeedPost
  /** Appelé après une suppression réussie (le parent retire le post du fil). */
  onDeleted?: (id: string) => void
  /** Appelé après une mise à jour réussie (pin/unpin, etc.). */
  onUpdated?: (post: FeedPost) => void
}

/**
 * Carte d'un post : en-tête (auteur + horodatage + menu), contenu, barre
 * d'actions (commenter / liker / partager) et section commentaires repliable.
 *
 * Like et suppression sont câblés sur le post-service (optimistes + rollback).
 * Repost et vues restent décoratifs (pas d'API back).
 */
export function PostCard({ post, onDeleted, onUpdated }: PostCardProps) {
  const { toast } = useToast()
  const { t, locale } = useLanguage()

  const [liked, setLiked] = useState(post.liked)
  const [likeCount, setLikeCount] = useState(post.likesCount)
  const [commentCount, setCommentCount] = useState(post.commentsCount)
  const [isPinned, setIsPinned] = useState(post.isPinned)
  const [pinning, setPinning] = useState(false)
  const [likeBurst, setLikeBurst] = useState(0)
  const [showComments, setShowComments] = useState(false)
  const [deleting, setDeleting] = useState(false)

  // Décoratifs (pas de backend) : état purement local.
  const [reposted, setReposted] = useState(false)
  const [repostCount, setRepostCount] = useState(0)

  useEffect(() => {
    setIsPinned(post.isPinned)
  }, [post.isPinned])

  async function toggleLike() {
    const next = !liked
    // Optimiste.
    setLiked(next)
    if (next) setLikeBurst((n) => n + 1)
    setLikeCount((n) => n + (next ? 1 : -1))
    try {
      const count = next ? await likePost(post.id) : await unlikePost(post.id)
      setLikeCount(count) // reconcilie avec le compteur serveur
    } catch {
      // Rollback.
      setLiked(!next)
      setLikeCount((n) => n + (next ? -1 : 1))
      toast({ title: t('common.action_failed'), variant: 'destructive' })
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
      toast({ title: t('post.deleted') })
      onDeleted?.(post.id)
    } catch {
      setDeleting(false)
      toast({ title: t('common.delete_failed'), variant: 'destructive' })
    }
  }

  async function togglePin() {
    if (pinning) return
    const next = !isPinned
    setPinning(true)
    setIsPinned(next)
    try {
      const updated = next ? await pinPost(post.id) : await unpinPost(post.id)
      setIsPinned(updated.isPinned)
      onUpdated?.(updated)
      toast({ title: next ? 'Post épinglé sur le profil' : 'Post désépinglé' })
    } catch {
      setIsPinned(!next)
      toast({ title: 'Épinglage impossible', variant: 'destructive' })
    } finally {
      setPinning(false)
    }
  }

  return (
    <article className="glass mx-3 my-3 flex gap-3 rounded-[24px] border px-4 py-3 backdrop-blur-xl transition hover:-translate-y-0.5 hover:bg-white/85 hover:shadow-[0_20px_56px_rgba(91,108,255,0.16)] dark:hover:bg-[#1f1633]/80">
      <ProfilLink author={post.author} className="mt-0.5 shrink-0 transition hover:opacity-90">
        <Avatar className="h-10 w-10">
          {post.author.avatarUrl && <AvatarImage src={post.author.avatarUrl} alt="" />}
          <AvatarFallback className="bg-gradient-to-br from-[#8D3DFF] via-[#5B6CFF] to-[#47D9FF] font-bold text-white">
            {initialOf(post.author.displayName)}
          </AvatarFallback>
        </Avatar>
      </ProfilLink>

      <div className="flex min-w-0 flex-1 flex-col gap-1">
        {/* Header */}
        <div className="flex items-center justify-between gap-2">
          <div className="flex min-w-0 items-center gap-1.5 text-sm">
            <ProfilLink
              author={post.author}
              className="truncate font-bold text-foreground hover:underline"
            >
              {post.author.displayName}
            </ProfilLink>
            {post.author.username && (
              <ProfilLink
                author={post.author}
                className="shrink-0 text-muted-foreground hover:underline"
              >
                @{post.author.username}
              </ProfilLink>
            )}
            <span className="shrink-0 text-muted-foreground">·</span>
            <span className="shrink-0 text-muted-foreground">{timeAgo(post.createdAt, locale)}</span>
          </div>

          {post.canDelete && (
            <DropdownMenu>
              <DropdownMenuTrigger
                aria-label={t('post.more_options')}
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
                {post.canPin && (
                  <DropdownMenuItem onClick={togglePin} disabled={pinning} className="cursor-pointer">
                    {isPinned ? (
                      <PinOff className="mr-2 h-4 w-4" />
                    ) : (
                      <Pin className="mr-2 h-4 w-4" />
                    )}
                    {isPinned ? 'Désépingler du profil' : 'Épingler sur le profil'}
                  </DropdownMenuItem>
                )}
                <DropdownMenuItem
                  onClick={handleDelete}
                  className="cursor-pointer text-red-500 focus:text-red-500"
                >
                  <Trash2 className="mr-2 h-4 w-4" />
                  {t('post.delete')}
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          )}
        </div>

        {isPinned && (
          <div className="mb-0.5 flex items-center gap-1 text-xs font-semibold text-primary">
            <Pin className="h-3.5 w-3.5 fill-current" />
            <span>Épinglé</span>
          </div>
        )}

        {/* Content */}
        <TranslatedContent
          contentId={`post:${post.id}`}
          content={post.content}
          className="whitespace-pre-wrap break-words text-sm leading-relaxed text-foreground/80"
        />

        {/* Actions */}
        <div className="-ml-2 mt-1 flex items-center justify-between text-muted-foreground">
          <ActionButton
            icon={MessageCircle}
            count={commentCount}
            label={t('post.comment')}
            active={showComments}
            onClick={() => setShowComments((v) => !v)}
            className="hover:text-primary hover:bg-primary/10"
            activeClassName="text-primary"
          />
          <ActionButton
            icon={Repeat2}
            count={repostCount}
            label={t('post.repost')}
            active={reposted}
            onClick={toggleRepost}
            className="hover:text-green-500 hover:bg-green-500/10"
            activeClassName="text-green-500"
          />
          <ActionButton
            icon={Heart}
            count={likeCount}
            label={t('post.like')}
            active={liked}
            onClick={toggleLike}
            burstKey={likeBurst}
            className="hover:text-red-500 hover:bg-red-500/10"
            activeClassName="text-red-500 fill-red-500"
          />
          <ActionButton
            icon={BarChart2}
            count={0}
            label={t('post.views')}
            className="hover:text-primary hover:bg-primary/10"
          />
          <button
            aria-label={t('post.share')}
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
  burstKey?: number
}

function ActionButton({
  icon: Icon,
  count,
  label,
  active = false,
  onClick,
  className,
  activeClassName,
  burstKey = 0,
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
      <span className="relative grid h-4 w-4 place-items-center">
        {burstKey > 0 && (
          <span
            key={burstKey}
            aria-hidden
            className="pointer-events-none absolute inset-[-8px] rounded-full border border-red-400/70 animate-like-burst"
          />
        )}
        <Icon
          className={cn(
            'h-4 w-4 transition-transform',
            active && 'animate-heart-pop',
            active && activeClassName,
          )}
        />
      </span>
      {count > 0 && <span>{formatCount(count)}</span>}
    </button>
  )
}

function formatCount(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`
  return String(n)
}
