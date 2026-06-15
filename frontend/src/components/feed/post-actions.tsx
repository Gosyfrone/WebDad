'use client'

import { useEffect, useState } from 'react'
import { BarChart2, Heart, Loader2, MessageCircle, Repeat2, Share } from 'lucide-react'

import { cn } from '@/lib/utils'
import {
  likePost,
  notifyPostCreated,
  repostPost,
  unlikePost,
  unrepostPost,
  type FeedPost,
} from '@/lib/posts'
import { useToast } from '@/hooks/use-toast'
import { useAuthGate } from '@/components/auth-prompt-provider'
import { useLanguage } from '@/components/language-provider'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { AnimatedCount } from '@/components/feed/animated-count'
import { PostComposer } from '@/components/feed/post-composer'

interface PostActionsProps {
  post: FeedPost
  /** Compteur de commentaires (contrôlé par le parent, maj par `CommentSection`). */
  commentCount: number
  /** Le bouton commentaire est-il « actif » (commentaires ouverts dans la carte) ? */
  commentActive?: boolean
  /** Clic sur le bouton commentaire (carte : replier/déplier ; modale : no-op). */
  onComment: () => void
}

/**
 * Barre d'actions d'un post (commenter / repost / citer / liker / partager),
 * avec état optimiste + rollback. **Partagée** par la carte du fil (`PostCard`)
 * et la vue photo (`PostPhotoModal`) → une seule source de vérité pour la
 * logique like/repost. Le compteur de commentaires est piloté par le parent
 * (qui possède `CommentSection`).
 */
export function PostActions({ post, commentCount, commentActive = false, onComment }: PostActionsProps) {
  const { toast } = useToast()
  const { t } = useLanguage()
  const { isVisitor, promptLogin, requireAuth } = useAuthGate()

  const [liked, setLiked] = useState(post.liked)
  const [likeCount, setLikeCount] = useState(post.likesCount)
  const [likeBurst, setLikeBurst] = useState(0)

  const [reposted, setReposted] = useState(post.reposted)
  const [repostCount, setRepostCount] = useState(post.repostsCount)
  const [reposting, setReposting] = useState(false)
  const [repostMenuOpen, setRepostMenuOpen] = useState(false)
  const [quoteOpen, setQuoteOpen] = useState(false)

  useEffect(() => {
    setReposted(post.reposted)
    setRepostCount(post.repostsCount)
  }, [post.reposted, post.repostsCount])

  // Compteur de likes resynchronisé sur la prop (rafraîchissement dynamique).
  // L'état « liked » par moi reste piloté par mes actions locales.
  useEffect(() => {
    setLikeCount(post.likesCount)
  }, [post.likesCount])

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

  async function toggleRepost() {
    if (reposting) return
    const next = !reposted
    setReposting(true)
    setReposted(next)
    setRepostCount((prev) => Math.max(0, prev + (next ? 1 : -1)))
    setRepostMenuOpen(false)
    try {
      if (next) {
        const updated = await repostPost(post.id)
        setRepostCount(updated.repostsCount)
        notifyPostCreated(updated)
        toast({ title: 'Post reposté sur votre profil' })
      } else {
        const count = await unrepostPost(post.id)
        setRepostCount(count)
      }
    } catch {
      setReposted(!next)
      setRepostCount((prev) => Math.max(0, prev + (next ? -1 : 1)))
      toast({ title: 'Repost impossible', variant: 'destructive' })
    } finally {
      setReposting(false)
    }
  }

  return (
    <>
      <div className="-ml-2 flex items-center justify-between text-muted-foreground">
        <ActionButton
          icon={MessageCircle}
          count={commentCount}
          label={t('post.comment')}
          active={commentActive}
          onClick={onComment}
          className="hover:text-primary hover:bg-primary/10"
          activeClassName="text-primary"
        />
        <Popover
          open={repostMenuOpen}
          onOpenChange={(o) => {
            // Visiteur : invite à se connecter plutôt que d'ouvrir le menu repost.
            if (o && isVisitor) {
              promptLogin()
              return
            }
            setRepostMenuOpen(o)
          }}
        >
          <PopoverTrigger asChild>
            <button
              aria-label={t('post.repost')}
              className={cn(
                'flex items-center gap-1 rounded-full p-1.5 text-xs transition-colors hover:bg-green-500/10 hover:text-green-500',
                reposted && 'text-green-500',
              )}
            >
              {reposting ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Repeat2 className={cn('h-4 w-4', reposted && 'stroke-[2.6]')} />
              )}
              <AnimatedCount value={repostCount} />
            </button>
          </PopoverTrigger>
          <PopoverContent align="start" className="w-36 p-1">
            <button
              type="button"
              onClick={toggleRepost}
              className="flex w-full items-center gap-2 rounded-md px-3 py-2 text-sm font-medium transition-colors hover:bg-accent"
            >
              <Repeat2 className="h-4 w-4" />
              {reposted ? 'Annuler' : 'Repost'}
            </button>
            <button
              type="button"
              onClick={() => {
                setRepostMenuOpen(false)
                setQuoteOpen(true)
              }}
              className="flex w-full items-center gap-2 rounded-md px-3 py-2 text-sm font-medium transition-colors hover:bg-accent"
            >
              <MessageCircle className="h-4 w-4" />
              Citer
            </button>
          </PopoverContent>
        </Popover>
        <ActionButton
          icon={Heart}
          count={likeCount}
          label={t('post.like')}
          active={liked}
          onClick={requireAuth(toggleLike)}
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

      <Dialog open={quoteOpen} onOpenChange={setQuoteOpen}>
        <DialogContent className="panel top-24 translate-y-0 border p-4 shadow-[0_28px_80px_rgba(91,108,255,0.24)] sm:max-h-[calc(100vh-8rem)] sm:max-w-xl sm:overflow-y-auto">
          <DialogHeader className="sr-only">
            <DialogTitle>Citer le post</DialogTitle>
            <DialogDescription>Composer un post avec le post cité en dessous.</DialogDescription>
          </DialogHeader>
          <PostComposer
            autoFocus
            submitLabel="Citer"
            quotePost={post}
            className="pt-6"
            onPosted={() => setQuoteOpen(false)}
          />
        </DialogContent>
      </Dialog>
    </>
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
      <AnimatedCount value={count} />
    </button>
  )
}
