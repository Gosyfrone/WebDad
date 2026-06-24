'use client'

import { useEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { useRouter } from 'next/navigation'
import {
  Bookmark,
  Clipboard,
  Download,
  Flag,
  Heart,
  Loader2,
  LockOpen,
  MessageCircle,
  MoreHorizontal,
  Pin,
  PinOff,
  Repeat2,
  Share,
  ShieldAlert,
  Trash2,
  UserX,
} from 'lucide-react'

import { cn, initialOf, timeAgo } from '@/lib/utils'
import {
  closePoll,
  currentUserId,
  deletePost,
  getPostById,
  likePost,
  notifyPostCreated,
  pinPost,
  repostPost,
  setPostNsfw,
  unlikePost,
  unrepostPost,
  unpinPost,
  votePoll,
  type FeedPost,
  type PostMedia,
  type PostPoll,
} from '@/lib/posts'
import { formatPollRemaining, isPollClosed, isPollClosedAt, pollResultsView } from '@/lib/poll-view'
import { resolveMediaUrl } from '@/lib/media'
import { quickBookmark, removeBookmarkEverywhere } from '@/lib/bookmarks'
import { useLongPress } from '@/lib/use-long-press'
import { BookmarkDialog } from '@/components/feed/bookmark-dialog'
import { ToastAction } from '@/components/ui/toast'
import { useToast } from '@/hooks/use-toast'
import { useCurrentUser } from '@/components/current-user-provider'
import { useAuthGate } from '@/components/auth-prompt-provider'
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
import { AnimatedCount } from '@/components/feed/animated-count'
import { CommentSection } from '@/components/feed/comment-section'
import { FeedVideo } from '@/components/feed/feed-video'
import { PostComposer } from '@/components/feed/post-composer'
import { PostPhotoModal } from '@/components/feed/post-photo-modal'
import { TranslatedContent } from '@/components/feed/translated-content'
import { MentionText } from '@/components/mention/mention-text'
import { ShareDialog } from '@/components/share/share-dialog'
import { postHref } from '@/lib/routes'
import { blockUser } from '@/lib/api'
import { BLOCK_CHANGE_EVENT, emitBlockChange, type BlockChangeDetail } from '@/lib/use-block'
import { ActivityPresenceDot } from '@/components/profil/activity-presence-dot'
import { CertificationBadge } from '@/components/profil/certification-badge'
import { ProfilLink } from '@/components/profil/profil-link'
import { ReportDialog } from '@/components/moderation/report-dialog'

interface PostCardProps {
  post: FeedPost
  /** Affiche le badge public "Épinglé" (profil uniquement). */
  showPinBadge?: boolean
  /** Commentaire à mettre en avant (lien depuis une notification) : ouvre la
   *  section commentaires et y défile + surligne le commentaire ciblé. */
  focusCommentId?: string
  /** Rendu « intégré » (sans carte flottante : ni glass/ombre, ni marges, ni
   *  effet de survol). Utilisé quand le post est imbriqué dans une vue
   *  conversation (onglet Réponses). */
  embedded?: boolean
  /** Appelé après une suppression réussie (le parent retire le post du fil). */
  onDeleted?: (id: string) => void
  /** Appelé après une mise à jour réussie (pin/unpin, etc.). */
  onUpdated?: (post: FeedPost) => void
  /** Ouvre les commentaires par défaut sans interaction (page détail). */
  defaultShowComments?: boolean
  /** Remplace le toggle inline des commentaires (ex : navigate depuis la page réponses). */
  onCommentClick?: () => void
  /** Désactive le clic de navigation vers la page détail (page détail elle-même). */
  noNavigate?: boolean
}

/** Carte de post : en-tête, contenu, barre d'actions (like, repost, signet, suppression
 *  optimistes) et commentaires repliables. */
export function PostCard({ post, showPinBadge = false, focusCommentId, embedded = false, onDeleted, onUpdated, defaultShowComments = false, onCommentClick, noNavigate = false }: PostCardProps) {
  const router = useRouter()
  const { toast } = useToast()
  const { isVisitor, promptLogin, requireAuth } = useAuthGate()
  const { t, locale } = useLanguage()
  const { profil: viewerProfil } = useCurrentUser()

  const [liked, setLiked] = useState(post.liked)
  const [likeCount, setLikeCount] = useState(post.likesCount)
  const [commentCount, setCommentCount] = useState(post.commentsCount)
  const [isPinned, setIsPinned] = useState(post.isPinned)
  const [pinning, setPinning] = useState(false)
  const [nsfw, setNsfw] = useState(post.nsfw)
  const [nsfwSaving, setNsfwSaving] = useState(false)
  const [likeBurst, setLikeBurst] = useState(0)
  const [showComments, setShowComments] = useState(Boolean(focusCommentId) || defaultShowComments)
  const [deleting, setDeleting] = useState(false)
  const [hiddenByBlock, setHiddenByBlock] = useState(false)
  const [reportOpen, setReportOpen] = useState(false)
  // Index du média ouvert en vue photo plein écran (null = fermé).
  const [photoIndex, setPhotoIndex] = useState<number | null>(null)

  // Signalement : ouvert à tout utilisateur connecté (le visiteur passe par la
  // modale de connexion). On l'affiche aussi sur ses propres posts pour que
  // l'action soit toujours visible ; la sécurité réelle vit côté back.
  const canReport = !isVisitor && Boolean(currentUserId())
  const canBlock = canReport && currentUserId() !== post.author.id
  const [poll, setPoll] = useState(post.poll)
  const pollClosed = poll ? isPollClosed(poll) : false

  const [reposted, setReposted] = useState(post.reposted)
  const [repostedById, setRepostedById] = useState(post.repostedById)
  const [repostCount, setRepostCount] = useState(post.repostsCount)
  const [reposting, setReposting] = useState(false)
  const [repostMenuOpen, setRepostMenuOpen] = useState(false)
  const [shareOpen, setShareOpen] = useState(false)
  const [quoteOpen, setQuoteOpen] = useState(false)

  const [bookmarked, setBookmarked] = useState(post.bookmarked)
  const [bookmarking, setBookmarking] = useState(false)
  const [pickerOpen, setPickerOpen] = useState(false)
  const displayPinBadge = isPinned && (showPinBadge || post.canPin)
  const showPrivateBadge =
    post.author.visibility === 'private' && post.author.id !== currentUserId()
  // Appui long sur le signet → sélecteur de collection (sans auto-classer).
  const bookmarkLongPress = useLongPress(() => setPickerOpen(true), { enabled: !isVisitor })

  useEffect(() => {
    setBookmarked(post.bookmarked)
  }, [post.bookmarked])

  useEffect(() => {
    setIsPinned(post.isPinned)
  }, [post.isPinned])

  useEffect(() => {
    setNsfw(post.nsfw)
  }, [post.nsfw])

  // Floutage NSFW : un post marqué est masqué tant que le lecteur n'a pas la
  // préférence/majorité pour le voir. Visiteur non connecté → toujours flouté
  // (ni majorité ni consentement vérifiables). Connecté → `nsfw_visible`
  // calculé serveur, défaut `true` le temps que le profil charge (évite un
  // flash de floutage à chaque montage).
  const nsfwBlurred = nsfw && (isVisitor || !(viewerProfil?.nsfwVisible ?? true))

  useEffect(() => {
    setPoll(post.poll)
  }, [post.poll])

  useEffect(() => {
    if (!poll || !poll.canViewResults || pollClosed) return
    const timer = setInterval(() => {
      void refreshPoll()
    }, 5000)
    return () => clearInterval(timer)
    // `refreshPoll` intentionally stays outside deps; it reads the stable post id.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [poll?.canViewResults, pollClosed, post.id])

  useEffect(() => {
    setReposted(post.reposted)
    setRepostedById(post.repostedById)
    setRepostCount(post.repostsCount)
  }, [post.reposted, post.repostedById, post.repostsCount])

  // Resync du compteur serveur (polling) sans toucher l'état local « liked ».
  useEffect(() => {
    setLikeCount(post.likesCount)
  }, [post.likesCount])

  useEffect(() => {
    setCommentCount(post.commentsCount)
  }, [post.commentsCount])

  useEffect(() => {
    function onBlockChange(event: Event) {
      const { userId, blocked } = (event as CustomEvent<BlockChangeDetail>).detail
      if (userId === post.author.id && blocked) setHiddenByBlock(true)
    }
    window.addEventListener(BLOCK_CHANGE_EVENT, onBlockChange)
    return () => window.removeEventListener(BLOCK_CHANGE_EVENT, onBlockChange)
  }, [post.author.id])

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

  async function handleBlockAuthor() {
    try {
      await blockUser(post.author.id)
      emitBlockChange({ userId: post.author.id, blocked: true })
      setHiddenByBlock(true)
      toast({ title: t('block.blocked') })
    } catch {
      toast({ title: t('block.block_failed'), variant: 'destructive' })
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

  async function toggleNsfw() {
    if (nsfwSaving) return
    const next = !nsfw
    setNsfwSaving(true)
    setNsfw(next)
    try {
      const updated = await setPostNsfw(post.id, next)
      setNsfw(updated.nsfw)
      onUpdated?.(updated)
      toast({ title: next ? t('nsfw.marked') : t('nsfw.unmarked') })
    } catch {
      setNsfw(!next)
      toast({ title: t('nsfw.action_failed'), variant: 'destructive' })
    } finally {
      setNsfwSaving(false)
    }
  }

  async function handlePollVote(choiceId: string) {
    try {
      const updated = await votePoll(post.id, choiceId)
      setPoll(updated.poll)
      onUpdated?.(updated)
    } catch {
      toast({ title: t('post.poll_failed'), variant: 'destructive' })
    }
  }

  async function handleClosePoll() {
    try {
      const updated = await closePoll(post.id)
      setPoll(updated.poll)
      onUpdated?.(updated)
    } catch {
      toast({ title: t('post.poll_close_failed'), variant: 'destructive' })
    }
  }

  async function refreshPoll() {
    const updated = await getPostById(post.id)
    if (!updated) return
    setPoll(updated.poll)
    onUpdated?.(updated)
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

  function handleCardClick(e: React.MouseEvent) {
    const target = e.target as HTMLElement
    if (target.closest('button, a, [role="button"], [data-no-nav]')) return
    router.push(`/posts/${post.id}`)
  }

  function handleBookmarkClick() {
    // Un appui long a déjà ouvert le sélecteur → on n'enchaîne pas le clic court.
    if (bookmarkLongPress.consume()) return
    // Visiteur : invite à se connecter (signets réservés aux membres).
    if (isVisitor) {
      promptLogin()
      return
    }
    void quickToggleBookmark()
  }

  if (hiddenByBlock) return null

  return (
    <article
      onClick={noNavigate ? undefined : handleCardClick}
      className={cn(
        'flex gap-3 px-4 py-3',
        !noNavigate && 'cursor-pointer',
        embedded
          ? // Intégré (vue conversation) : pas de carte flottante ni d'ombre.
            'bg-transparent'
          : 'glass mx-3 my-3 rounded-[24px] border backdrop-blur-xl transition hover:-translate-y-0.5 hover:bg-white/85 hover:shadow-[0_20px_56px_rgba(91,108,255,0.16)] dark:hover:bg-[#1f1633]/80',
      )}
    >
      <ProfilLink author={post.author} className="mt-0.5 shrink-0 self-start transition hover:opacity-90">
        <Avatar className="h-10 w-10">
          {post.author.avatarUrl && <AvatarImage src={post.author.avatarUrl} alt="" />}
          <AvatarFallback className="bg-gradient-to-br from-[var(--brand-from)] via-[var(--brand-via)] to-[var(--brand-to)] font-bold text-white">
            {initialOf(post.author.displayName, post.author.username)}
          </AvatarFallback>
          <ActivityPresenceDot
            userId={post.author.id}
            initialLastLoginAt={post.author.lastLoginAt}
            initialIsOnline={post.author.isOnline}
          />
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
            <CertificationBadge certification={post.author.certification} className="h-4 w-4" />
            {post.author.username && (
              <ProfilLink
                author={post.author}
                className="shrink-0 text-muted-foreground hover:underline"
              >
                @{post.author.username}
              </ProfilLink>
            )}
            {showPrivateBadge && (
              <PrivateAuthorBadge label={t('post.private_account_tooltip')} />
            )}
            <span className="shrink-0 text-muted-foreground">·</span>
            <span className="shrink-0 text-muted-foreground">{timeAgo(post.createdAt, locale)}</span>
          </div>

          {(post.canDelete || canReport || canBlock || post.canMarkNsfw) && (
            <DropdownMenu modal={false}>
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
              <DropdownMenuContent align="end" className="z-40">
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
                {post.canMarkNsfw && (
                  <DropdownMenuItem
                    onClick={(e) => {
                      e.stopPropagation()
                      void toggleNsfw()
                    }}
                    disabled={nsfwSaving}
                    className="cursor-pointer"
                  >
                    <ShieldAlert className="mr-2 h-4 w-4" />
                    {nsfw ? t('nsfw.unmark') : t('nsfw.mark')}
                  </DropdownMenuItem>
                )}
                {canReport && (
                  <DropdownMenuItem
                    onClick={(e) => {
                      // Empêche le clic de remonter à la carte (qui navigue vers
                      // le détail du post) : on ouvre juste la modale, on reste au feed.
                      e.stopPropagation()
                      setReportOpen(true)
                    }}
                    className="cursor-pointer"
                  >
                    <Flag className="mr-2 h-4 w-4" />
                    {t('report.post_action')}
                  </DropdownMenuItem>
                )}
                {canBlock && (
                  <DropdownMenuItem
                    onClick={(e) => {
                      e.stopPropagation()
                      void handleBlockAuthor()
                    }}
                    className="cursor-pointer text-red-500 focus:text-red-500"
                  >
                    <UserX className="mr-2 h-4 w-4" />
                    {t('block.block_user')}
                  </DropdownMenuItem>
                )}
                {post.canDelete && (
                  <DropdownMenuItem
                    onClick={handleDelete}
                    className="cursor-pointer text-red-500 focus:text-red-500"
                  >
                    <Trash2 className="mr-2 h-4 w-4" />
                    {t('post.delete')}
                  </DropdownMenuItem>
                )}
              </DropdownMenuContent>
            </DropdownMenu>
          )}

          {canReport && (
            // Le contenu de la modale est porté (portal) mais reste enfant de la
            // carte dans l'arbre React → ses clics y remontent. On les arrête ici
            // pour ne jamais déclencher la navigation vers le détail du post.
            // `display:contents` (classe `contents`) → ce span ne génère AUCUNE
            // boîte : il ne compte pas comme un item flex (sinon il décale le « … »).
            <span className="contents" onClick={(e) => e.stopPropagation()}>
              <ReportDialog
                open={reportOpen}
                onOpenChange={setReportOpen}
                entityType="post"
                entityId={post.id}
                entityOwnerId={post.author.id}
                targetLabel={post.author.username ? `@${post.author.username}` : undefined}
              />
            </span>
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

        {/* Content (flouté si marqué NSFW et lecteur non autorisé/non consentant) */}
        <div className={cn('relative', nsfwBlurred && 'min-h-[9rem]')}>
          <div
            className={cn(
              nsfwBlurred && 'pointer-events-none select-none blur-xl',
            )}
            aria-hidden={nsfwBlurred}
          >
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

            {poll && (
              <PostPollCard
                poll={poll}
                onVote={requireAuth(handlePollVote)}
                onClose={poll.canClose ? handleClosePoll : undefined}
                onRefresh={refreshPoll}
              />
            )}

            {post.quotedPost && (
              <QuotedPost post={post.quotedPost} />
            )}
          </div>

          {nsfwBlurred && (
            <div
              className="absolute inset-0 flex flex-col items-center justify-center gap-2 rounded-xl bg-background/40 px-4 text-center backdrop-blur-sm"
              onClick={(e) => e.stopPropagation()}
            >
              <span className="inline-flex items-center gap-1.5 rounded-full bg-foreground/80 px-2.5 py-1 text-xs font-bold uppercase tracking-wide text-background">
                <ShieldAlert className="h-3.5 w-3.5" aria-hidden />
                {t('nsfw.post_badge')}
              </span>
              <p className="max-w-xs text-xs font-medium text-foreground">
                {isVisitor ? t('nsfw.post_hidden_visitor') : t('nsfw.post_hidden_desc')}
              </p>
              {isVisitor && (
                <button
                  type="button"
                  onClick={promptLogin}
                  className="mt-1 rounded-full bg-primary px-3 py-1 text-xs font-semibold text-primary-foreground transition hover:bg-primary/90"
                >
                  {t('visitor.login')}
                </button>
              )}
            </div>
          )}
        </div>

        {/* Actions — masquées tant que le post est flouté NSFW (le lecteur
            désactive d'abord le filtre dans les paramètres avant d'interagir). */}
        {!nsfwBlurred && (
        <div className="-ml-2 mt-1 flex items-center justify-between text-muted-foreground">
          <ActionButton
            icon={MessageCircle}
            count={commentCount}
            label={t('post.comment')}
            active={showComments}
            onClick={onCommentClick ?? (() => setShowComments((v) => !v))}
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
          <button
            aria-label={bookmarked ? t('bookmarks.remove_aria') : t('bookmarks.add_aria')}
            onClick={handleBookmarkClick}
            onPointerDown={bookmarkLongPress.start}
            onPointerUp={bookmarkLongPress.cancel}
            onPointerLeave={bookmarkLongPress.cancel}
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
            onClick={() => setShareOpen(true)}
            className="rounded-full p-1.5 transition-colors hover:bg-primary/10 hover:text-primary"
          >
            <Share className="h-4 w-4" />
          </button>
        </div>
        )}

        <ShareDialog
          open={shareOpen}
          onOpenChange={setShareOpen}
          url={postHref(post.id)}
          kind="post"
          title={t('post.share')}
        />

        {/* Commentaires (repliable) */}
        {showComments && (
          <div data-no-nav>
            <CommentSection
              postId={post.id}
              focusCommentId={focusCommentId}
              onCountChange={(delta) => setCommentCount((n) => Math.max(0, n + delta))}
              canReply={post.canReply}
            />
          </div>
        )}

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

function PrivateAuthorBadge({ label }: { label: string }) {
  return (
    <span className="group relative inline-flex shrink-0 items-center">
      <LockOpen
        tabIndex={0}
        className="h-3.5 w-3.5 text-primary outline-none"
        aria-label={label}
      />
      <span className="pointer-events-none absolute left-1/2 top-5 z-20 w-max max-w-[12rem] -translate-x-1/2 rounded-md border bg-popover px-2 py-1 text-xs font-medium text-popover-foreground opacity-0 shadow-md transition-opacity group-hover:opacity-100 group-focus-within:opacity-100">
        {label}
      </span>
    </span>
  )
}

/**
 * Galerie des médias d'un post (images + vidéos). Disposition façon X :
 * 1 média = pleine largeur ; 2-4 = grille 2 colonnes. Les URLs sont déjà
 * absolues (résolues dans `toFeedPost`). Cache navigateur géré par le
 * media-service (`Cache-Control: immutable`) ; lazy-loading des images.
 */
function MediaGallery({
  media,
  onOpen,
  compact = false,
}: {
  media: PostMedia[]
  onOpen?: (index: number) => void
  compact?: boolean
}) {
  const { t } = useLanguage()
  const { toast } = useToast()
  const [menu, setMenu] = useState<{ url: string; x: number; y: number } | null>(null)
  // Sur tactile (iOS/Android), on laisse le menu natif du navigateur gérer
  // l'appui long sur l'image ; le menu custom Breezy n'apparaît qu'au clic droit
  // souris (desktop). On mémorise le dernier type de pointeur pour distinguer.
  const lastPointerType = useRef<string>('mouse')

  useEffect(() => {
    if (!menu) return
    function close() {
      setMenu(null)
    }
    window.addEventListener('pointerdown', close)
    window.addEventListener('scroll', close, true)
    window.addEventListener('resize', close)
    return () => {
      window.removeEventListener('pointerdown', close)
      window.removeEventListener('scroll', close, true)
      window.removeEventListener('resize', close)
    }
  }, [menu])

  function handleImageContextMenu(e: React.MouseEvent, url: string) {
    // Tactile → menu natif du navigateur (« Enregistrer l'image »…). Le menu
    // Breezy (copier/enregistrer) reste au clic droit souris uniquement.
    if (lastPointerType.current !== 'mouse') return
    e.preventDefault()
    setMenu({ url, x: e.clientX, y: e.clientY })
  }

  async function copyImage(url: string) {
    try {
      await copyImageToClipboard(url)
      toast({ title: t('media.copy_success') })
    } catch {
      toast({ title: t('media.copy_failed'), variant: 'destructive' })
    } finally {
      setMenu(null)
    }
  }

  async function saveImage(url: string) {
    try {
      await downloadImage(url)
      toast({ title: t('media.save_started') })
    } catch {
      toast({ title: t('media.save_failed'), variant: 'destructive' })
    } finally {
      setMenu(null)
    }
  }

  return (
    <>
      <div
        className={cn(
          'mt-2 grid gap-1.5 overflow-hidden border border-border',
          compact ? 'rounded-xl' : 'rounded-2xl',
          media.length === 1 ? 'grid-cols-1' : 'grid-cols-2',
        )}
      >
        {media.map((m, i) => {
          const sizing = cn(
            media.length === 1 ? (compact ? 'max-h-64' : 'max-h-[32rem]') : 'aspect-square',
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
          if (m.type === 'video') {
            return <FeedVideo key={m.url} src={m.url} className={videoCell} />
          }

          const image = (
            // eslint-disable-next-line @next/next/no-img-element
            <img
              src={m.url}
              alt=""
              loading="lazy"
              className={cn('h-full w-full object-cover transition group-hover:brightness-95', sizing)}
            />
          )

          if (!onOpen) {
            return (
              <div
                key={m.url}
                data-no-nav
                onPointerDown={(e) => (lastPointerType.current = e.pointerType)}
                onContextMenu={(e) => handleImageContextMenu(e, m.url)}
                className={cn('group relative block overflow-hidden', media.length === 3 && i === 0 && 'row-span-2')}
              >
                {image}
              </div>
            )
          }

          return (
            <button
              key={m.url}
              type="button"
              data-no-nav
              onClick={() => onOpen?.(i)}
              onPointerDown={(e) => (lastPointerType.current = e.pointerType)}
              onContextMenu={(e) => handleImageContextMenu(e, m.url)}
              className={cn('group relative block overflow-hidden', media.length === 3 && i === 0 && 'row-span-2')}
              aria-label={t('media.open_image')}
            >
              {image}
            </button>
          )
        })}
      </div>
      {menu && (
        <ImageActionMenu
          x={menu.x}
          y={menu.y}
          onCopy={() => void copyImage(menu.url)}
          onSave={() => void saveImage(menu.url)}
          copyLabel={t('media.copy_image')}
          saveLabel={t('media.save_image')}
        />
      )}
    </>
  )
}

function ImageActionMenu({
  x,
  y,
  onCopy,
  onSave,
  copyLabel,
  saveLabel,
}: {
  x: number
  y: number
  onCopy: () => void
  onSave: () => void
  copyLabel: string
  saveLabel: string
}) {
  if (typeof document === 'undefined') return null

  const left = Math.min(Math.max(12, x - 88), window.innerWidth - 188)
  const top = Math.min(Math.max(12, y + 14), window.innerHeight - 112)

  return createPortal(
    <div
      data-no-nav
      onPointerDown={(e) => e.stopPropagation()}
      className="fixed z-[70] w-44 overflow-hidden rounded-xl border bg-popover p-1 text-popover-foreground shadow-2xl"
      style={{ left, top }}
      role="menu"
    >
      <button
        type="button"
        onClick={onCopy}
        className="flex w-full items-center gap-2 rounded-lg px-3 py-2 text-left text-sm font-medium transition-colors hover:bg-accent focus:bg-accent focus:outline-none"
        role="menuitem"
      >
        <Clipboard className="h-4 w-4" />
        {copyLabel}
      </button>
      <button
        type="button"
        onClick={onSave}
        className="flex w-full items-center gap-2 rounded-lg px-3 py-2 text-left text-sm font-medium transition-colors hover:bg-accent focus:bg-accent focus:outline-none"
        role="menuitem"
      >
        <Download className="h-4 w-4" />
        {saveLabel}
      </button>
    </div>,
    document.body,
  )
}

async function fetchImageBlob(url: string): Promise<Blob> {
  const res = await fetch(url)
  if (!res.ok) throw new Error('image_fetch_failed')
  const blob = await res.blob()
  if (!blob.type.startsWith('image/')) throw new Error('not_an_image')
  return blob
}

async function copyImageToClipboard(url: string): Promise<void> {
  if (!navigator.clipboard || typeof ClipboardItem === 'undefined') {
    throw new Error('clipboard_unavailable')
  }
  const blob = await fetchImageBlob(url)
  await navigator.clipboard.write([
    new ClipboardItem({
      [blob.type || 'image/png']: blob,
    }),
  ])
}

// Menu custom Breezy = desktop uniquement (clic droit) ; `<a download>` y est
// fiable. Sur tactile, c'est le menu natif du navigateur qui gère l'image.
async function downloadImage(url: string): Promise<void> {
  const blob = await fetchImageBlob(url)
  const objectUrl = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = objectUrl
  a.download = imageFileName(url, blob.type)
  document.body.appendChild(a)
  a.click()
  a.remove()
  setTimeout(() => URL.revokeObjectURL(objectUrl), 1000)
}

function imageFileName(url: string, mimeType: string): string {
  const extFromMime = mimeType.split('/')[1]?.split(';')[0]
  try {
    const parsed = new URL(url, window.location.href)
    const last = parsed.pathname.split('/').filter(Boolean).pop()
    if (last && /\.[a-z0-9]+$/i.test(last)) return last
  } catch {
    // Fallback below.
  }
  return `breezy-image.${extFromMime || 'png'}`
}

function QuotedPost({ post }: { post: FeedPost }) {
  const { t } = useLanguage()
  const router = useRouter()
  const showPrivateBadge =
    post.author.visibility === 'private' && post.author.id !== currentUserId()

  function handleClick(e: React.MouseEvent) {
    if ((e.target as HTMLElement).closest('a, button, [role="button"]')) return
    e.stopPropagation()
    router.push(`/posts/${post.id}`)
  }

  return (
    <div
      onClick={handleClick}
      className="mt-3 cursor-pointer rounded-xl border border-border bg-background/45 px-3 py-2 transition-colors hover:bg-background/70"
    >
      <div className="mb-1 flex min-w-0 items-center gap-1.5 text-xs">
        <ProfilLink author={post.author} className="truncate font-bold text-foreground hover:underline">
          {post.author.displayName}
        </ProfilLink>
        <CertificationBadge certification={post.author.certification} className="h-4 w-4" />
        {post.author.username && (
          <ProfilLink author={post.author} className="shrink-0 text-muted-foreground hover:underline">
            @{post.author.username}
          </ProfilLink>
        )}
        {showPrivateBadge && (
          <PrivateAuthorBadge label={t('post.private_account_tooltip')} />
        )}
      </div>
      <MentionText
        text={post.content}
        className="line-clamp-5 whitespace-pre-wrap break-words text-sm text-foreground/75"
      />
      {post.media.length > 0 && <MediaGallery media={post.media} compact />}
      {post.poll && <PostPollCard poll={post.poll} compact />}
    </div>
  )
}

function PostPollCard({
  poll,
  onVote,
  onClose,
  onRefresh,
  compact = false,
}: {
  poll: PostPoll
  onVote?: (choiceId: string) => Promise<void> | void
  onClose?: () => Promise<void> | void
  onRefresh?: () => Promise<void> | void
  compact?: boolean
}) {
  const { t } = useLanguage()
  const [voting, setVoting] = useState('')
  const [closing, setClosing] = useState(false)
  const [now, setNow] = useState(() => Date.now())
  const closed = isPollClosedAt(poll, now)
  const showResults = poll.canViewResults
  const total = Math.max(0, poll.totalVotes)
  // Bascule vers l'affichage « résultats » (barres type chart) une fois qu'on a
  // voté ou que le sondage est clos ; sinon on garde la vue de vote cliquable.
  const resultsView = pollResultsView(poll, closed)

  useEffect(() => {
    if (closed) return
    const timer = setInterval(() => {
      const next = Date.now()
      setNow(next)
      if (isPollClosedAt(poll, next)) {
        void onRefresh?.()
      }
    }, 1000)
    return () => clearInterval(timer)
  }, [closed, onRefresh, poll])

  async function vote(choiceId: string) {
    if (!onVote || voting || closed || poll.votedChoiceId) return
    setVoting(choiceId)
    try {
      await onVote(choiceId)
    } finally {
      setVoting('')
    }
  }

  async function closeNow() {
    if (!onClose || closing) return
    setClosing(true)
    try {
      await onClose()
    } finally {
      setClosing(false)
    }
  }

  return (
    <div
      className={cn(
        'mt-3',
        // En mode résultats : pas de cadre (façon Twitter). Sinon, on garde la carte.
        resultsView ? 'mt-2' : 'rounded-2xl border border-border bg-background/45 p-3',
        !resultsView && compact && 'rounded-xl p-2',
      )}
    >
      {onClose && !closed && !compact && (
        <div className="mb-2 flex justify-end">
          <button
            type="button"
            onClick={() => void closeNow()}
            disabled={closing}
            className="rounded-full px-3 py-1 text-xs font-semibold text-primary transition hover:bg-primary/10 disabled:opacity-60"
          >
            {closing ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : t('post.poll_close')}
          </button>
        </div>
      )}
      <div className={cn('space-y-2', resultsView && 'space-y-1.5')}>
        {poll.choices.map((choice) => {
          const percent = total > 0 ? Math.round((choice.votesCount / total) * 100) : 0
          const selected = poll.votedChoiceId === choice.id
          const winner = poll.winnerChoiceIds.includes(choice.id)

          // Vue résultats (façon Twitter) : barre proportionnelle non encadrée,
          // % à droite avec le nombre de votes en plus petit juste à sa gauche.
          if (resultsView) {
            return (
              <div
                key={choice.id}
                className="relative flex min-h-9 items-center justify-between gap-3 overflow-hidden rounded-md px-3 py-1.5 text-sm"
              >
                <span
                  aria-hidden
                  className={cn(
                    'absolute inset-y-0 left-0 rounded-md transition-all',
                    winner ? 'bg-primary/35' : 'bg-muted',
                  )}
                  style={{ width: `${Math.max(percent, 2)}%` }}
                />
                <span className="relative z-10 flex min-w-0 items-center gap-2">
                  {choice.imageUrl && (
                    <img
                      src={resolveMediaUrl(choice.imageUrl)}
                      alt=""
                      className="h-7 w-7 shrink-0 rounded object-cover"
                    />
                  )}
                  <span className={cn('min-w-0 truncate', winner ? 'font-bold' : 'font-medium')}>
                    {choice.label}
                  </span>
                </span>
                <span className="relative z-10 ml-3 flex shrink-0 items-baseline gap-1.5">
                  <span className="text-xs text-muted-foreground">{choice.votesCount}</span>
                  <span aria-hidden className="text-xs text-muted-foreground">·</span>
                  <span className={cn('font-semibold', winner && 'text-primary')}>{percent}%</span>
                </span>
              </div>
            )
          }

          return (
            <button
              key={choice.id}
              type="button"
              disabled={!onVote || closed || Boolean(poll.votedChoiceId) || Boolean(voting)}
              onClick={() => void vote(choice.id)}
              className={cn(
                'relative flex min-h-10 w-full items-center justify-between overflow-hidden rounded-lg border border-border px-3 py-2 text-left text-sm transition',
                onVote && !closed && !poll.votedChoiceId && 'hover:border-primary hover:bg-primary/5',
                selected && 'border-primary text-primary',
              )}
            >
              {showResults && (
                <span
                  aria-hidden
                  className="absolute inset-y-0 left-0 bg-primary/12 transition-all"
                  style={{ width: `${percent}%` }}
                />
              )}
              <span className="relative z-10 flex min-w-0 items-center gap-2">
                {choice.imageUrl && (
                  <img
                    src={resolveMediaUrl(choice.imageUrl)}
                    alt=""
                    className="h-8 w-8 shrink-0 rounded object-cover"
                  />
                )}
                <span className="min-w-0 truncate font-medium">{choice.label}</span>
              </span>
              <span className="relative z-10 ml-3 flex shrink-0 items-center gap-2 font-semibold">
                {voting === choice.id ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : showResults ? (
                  `${percent}%`
                ) : selected ? (
                  t('post.poll_voted')
                ) : (
                  t('post.poll_vote')
                )}
              </span>
            </button>
          )
        })}
      </div>
      <div className="mt-2 flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
        {showResults && (
          <>
            <span>{t('post.poll_votes', { count: total })}</span>
            <span>·</span>
          </>
        )}
        <span>{closed ? t('post.poll_closed') : t('post.poll_ends_in', { time: formatPollRemaining(poll.endsAt, now) })}</span>
        {poll.audience === 'followers' && (
          <>
            <span>·</span>
            <span>{t('post.poll_followers_only')}</span>
          </>
        )}
      </div>
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
      <AnimatedCount value={count} />
    </button>
  )
}
