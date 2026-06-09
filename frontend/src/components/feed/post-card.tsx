'use client'

import { useEffect, useRef, useState } from 'react'
import {
  BarChart2,
  Bookmark,
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
import {
  currentUserId,
  deletePost,
  likePost,
  notifyPostCreated,
  pinPost,
  repostPost,
  unlikePost,
  unrepostPost,
  unpinPost,
  type FeedPost,
  type PostMedia,
} from '@/lib/posts'
import { quickBookmark, removeBookmarkEverywhere } from '@/lib/bookmarks'
import { BookmarkDialog } from '@/components/feed/bookmark-dialog'
import { ToastAction } from '@/components/ui/toast'
import { useToast } from '@/hooks/use-toast'
import { useLanguage } from '@/components/language-provider'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { CommentSection } from '@/components/feed/comment-section'
import { FeedVideo } from '@/components/feed/feed-video'
import { PostComposer } from '@/components/feed/post-composer'
import { PostPhotoModal } from '@/components/feed/post-photo-modal'
import { TranslatedContent } from '@/components/feed/translated-content'
import { ProfilLink } from '@/components/profil/profil-link'

interface PostCardProps {
  post: FeedPost
  /** Affiche le badge public "Épinglé" (profil uniquement). */
  showPinBadge?: boolean
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
 * Repost simple et citation sont câblés sur le post-service.
 */
export function PostCard({ post, showPinBadge = false, onDeleted, onUpdated }: PostCardProps) {
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
  // Index du média ouvert en vue photo plein écran (null = fermé).
  const [photoIndex, setPhotoIndex] = useState<number | null>(null)

  const [reposted, setReposted] = useState(post.reposted)
  const [repostedById, setRepostedById] = useState(post.repostedById)
  const [repostCount, setRepostCount] = useState(post.repostsCount)
  const [reposting, setReposting] = useState(false)
  const [repostMenuOpen, setRepostMenuOpen] = useState(false)
  const [quoteOpen, setQuoteOpen] = useState(false)

  const [bookmarked, setBookmarked] = useState(post.bookmarked)
  const [bookmarking, setBookmarking] = useState(false)
  const [pickerOpen, setPickerOpen] = useState(false)
  const displayPinBadge = isPinned && (showPinBadge || post.canPin)
  // Détection de l'appui long (ouvre le sélecteur sans auto-classer).
  const longPress = useRef(false)
  const longPressTimer = useRef<ReturnType<typeof setTimeout> | null>(null)

  useEffect(() => {
    setBookmarked(post.bookmarked)
  }, [post.bookmarked])

  useEffect(() => {
    setIsPinned(post.isPinned)
  }, [post.isPinned])

  useEffect(() => {
    setReposted(post.reposted)
    setRepostedById(post.repostedById)
    setRepostCount(post.repostsCount)
  }, [post.reposted, post.repostedById, post.repostsCount])

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
    const previousRepostedById = repostedById
    setReposting(true)
    setReposted(next)
    setRepostCount((prev) => Math.max(0, prev + (next ? 1 : -1)))
    setRepostMenuOpen(false)
    try {
      if (next) {
        const updated = await repostPost(post.id)
        setRepostedById(updated.repostedById)
        setRepostCount(updated.repostsCount)
        notifyPostCreated(updated)
        toast({ title: 'Post reposté sur votre profil' })
      } else {
        const count = await unrepostPost(post.id)
        if (repostedById === currentUserId()) setRepostedById('')
        setRepostCount(count)
      }
    } catch {
      setReposted(!next)
      setRepostedById(previousRepostedById)
      setRepostCount((prev) => Math.max(0, prev + (next ? -1 : 1)))
      toast({ title: 'Repost impossible', variant: 'destructive' })
    } finally {
      setReposting(false)
    }
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

  /** Clic court : dé-signe si déjà signé, sinon laisse la rafale serveur décider. */
  async function quickToggleBookmark() {
    if (bookmarking) return
    if (bookmarked) {
      setBookmarking(true)
      setBookmarked(false)
      try {
        await removeBookmarkEverywhere(post.id)
        toast({ title: t('bookmarks.removed') })
      } catch {
        setBookmarked(true)
        toast({ title: t('common.action_failed'), variant: 'destructive' })
      } finally {
        setBookmarking(false)
      }
      return
    }
    setBookmarking(true)
    try {
      const result = await quickBookmark(post.id)
      if (result.status === 'filed') {
        setBookmarked(true)
        toast({
          title: t('bookmarks.saved_to', { name: result.collection.name }),
          action: (
            <ToastAction altText={t('bookmarks.organize')} onClick={() => setPickerOpen(true)}>
              {t('bookmarks.organize')}
            </ToastAction>
          ),
        })
      } else {
        // Ouverture de rafale : on laisse l'utilisateur choisir/créer la collection.
        setPickerOpen(true)
      }
    } catch {
      toast({ title: t('common.action_failed'), variant: 'destructive' })
    } finally {
      setBookmarking(false)
    }
  }

  function startLongPress() {
    longPress.current = false
    longPressTimer.current = setTimeout(() => {
      longPress.current = true
      setPickerOpen(true)
    }, 500)
  }

  function cancelLongPress() {
    if (longPressTimer.current) {
      clearTimeout(longPressTimer.current)
      longPressTimer.current = null
    }
  }

  function handleBookmarkClick() {
    // Un appui long a déjà ouvert le sélecteur → on n'enchaîne pas le clic court.
    if (longPress.current) {
      longPress.current = false
      return
    }
    void quickToggleBookmark()
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

        {(displayPinBadge || repostedById) && (
          <div className="mb-0.5 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs font-semibold">
            {displayPinBadge && (
              <div className="flex items-center gap-1 text-primary">
                <Pin className="h-3.5 w-3.5 fill-current" />
                <span>Épinglé</span>
              </div>
            )}
            {repostedById && (
              <div className="flex items-center gap-1 text-green-500">
                <Repeat2 className="h-3.5 w-3.5" />
                <span>Reposté</span>
              </div>
            )}
          </div>
        )}

        {/* Content */}
        {post.content && (
          <TranslatedContent
            contentId={`post:${post.id}`}
            content={post.content}
            className="whitespace-pre-wrap break-words text-sm leading-relaxed text-foreground/80"
          />
        )}

        {post.media.length > 0 && (
          <MediaGallery media={post.media} onOpen={(i) => setPhotoIndex(i)} />
        )}

        {post.quotedPost && (
          <QuotedPost post={post.quotedPost} />
        )}

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
          <Popover open={repostMenuOpen} onOpenChange={setRepostMenuOpen}>
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
                {repostCount > 0 && <span>{formatCount(repostCount)}</span>}
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
            aria-label={bookmarked ? t('bookmarks.remove_aria') : t('bookmarks.add_aria')}
            onClick={handleBookmarkClick}
            onPointerDown={startLongPress}
            onPointerUp={cancelLongPress}
            onPointerLeave={cancelLongPress}
            onContextMenu={(e) => e.preventDefault()}
            className={cn(
              'rounded-full p-1.5 transition-colors hover:bg-primary/10 hover:text-primary',
              bookmarked && 'text-primary',
            )}
          >
            {bookmarking ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Bookmark className={cn('h-4 w-4', bookmarked && 'fill-current')} />
            )}
          </button>
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

        <Dialog open={quoteOpen} onOpenChange={setQuoteOpen}>
          <DialogContent className="panel top-24 translate-y-0 border p-4 shadow-[0_28px_80px_rgba(91,108,255,0.24)] sm:max-w-xl">
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

        <BookmarkDialog
          postId={post.id}
          open={pickerOpen}
          onOpenChange={setPickerOpen}
          onMembershipChange={setBookmarked}
        />
      </div>

      {photoIndex !== null && (
        <PostPhotoModal post={post} index={photoIndex} onClose={() => setPhotoIndex(null)} />
      )}
    </article>
  )
}

/**
 * Galerie des médias d'un post (images + vidéos). Disposition façon X :
 * 1 média = pleine largeur ; 2-4 = grille 2 colonnes. Les URLs sont déjà
 * absolues (résolues dans `toFeedPost`). Cache navigateur géré par le
 * media-service (`Cache-Control: immutable`) ; lazy-loading des images.
 */
function MediaGallery({ media, onOpen }: { media: PostMedia[]; onOpen?: (index: number) => void }) {
  return (
    <div
      className={cn(
        'mt-2 grid gap-1.5 overflow-hidden rounded-2xl border border-border',
        media.length === 1 ? 'grid-cols-1' : 'grid-cols-2',
      )}
    >
      {media.map((m, i) => {
        const sizing = cn(
          media.length === 1 ? 'max-h-[32rem]' : 'aspect-square',
          media.length === 3 && i === 0 && 'row-span-2 aspect-auto',
        )
        // Cellule vidéo : un ratio défini est nécessaire (le `<video>` interne
        // est en `h-full`). 1 média = 16:9 ; sinon carré (grille).
        const videoCell = cn(
          media.length === 1 ? 'aspect-video' : 'aspect-square',
          media.length === 3 && i === 0 && 'row-span-2 aspect-auto',
        )
        // Vidéo : lecture auto en muet + boucle + vitesse (cf. FeedVideo), pas
        // d'ouverture en vue photo. L'image s'ouvre en grand au clic.
        return m.type === 'video' ? (
          <FeedVideo key={m.url} src={m.url} className={videoCell} />
        ) : (
          <button
            key={m.url}
            type="button"
            onClick={() => onOpen?.(i)}
            className={cn('group relative block overflow-hidden', media.length === 3 && i === 0 && 'row-span-2')}
            aria-label="Agrandir l'image"
          >
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img
              src={m.url}
              alt=""
              loading="lazy"
              className={cn('h-full w-full object-cover transition group-hover:brightness-95', sizing)}
            />
          </button>
        )
      })}
    </div>
  )
}

function QuotedPost({ post }: { post: FeedPost }) {
  return (
    <div className="mt-3 rounded-xl border border-border bg-background/45 px-3 py-2">
      <div className="mb-1 flex min-w-0 items-center gap-1.5 text-xs">
        <ProfilLink author={post.author} className="truncate font-bold text-foreground hover:underline">
          {post.author.displayName}
        </ProfilLink>
        {post.author.username && (
          <ProfilLink author={post.author} className="shrink-0 text-muted-foreground hover:underline">
            @{post.author.username}
          </ProfilLink>
        )}
      </div>
      <p className="line-clamp-5 whitespace-pre-wrap break-words text-sm text-foreground/75">
        {post.content}
      </p>
    </div>
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
