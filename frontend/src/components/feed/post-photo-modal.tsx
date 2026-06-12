'use client'

import { useEffect, useState } from 'react'
import { createPortal } from 'react-dom'
import { ChevronLeft, ChevronRight, LockOpen, X } from 'lucide-react'

import { cn, timeAgo } from '@/lib/utils'
import { currentUserId, type FeedPost } from '@/lib/posts'
import { useLanguage } from '@/components/language-provider'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { CommentSection } from '@/components/feed/comment-section'
import { PostActions } from '@/components/feed/post-actions'
import { TranslatedContent } from '@/components/feed/translated-content'
import { ProfilLink } from '@/components/profil/profil-link'

interface PostPhotoModalProps {
  post: FeedPost
  /** Index du média cliqué (le composant est monté seulement à l'ouverture). */
  index: number
  onClose: () => void
}

/**
 * Vue « photo » d'un post, façon X (deux volets) :
 *   - gauche (fond sombre) : le média en grand + navigation ‹ › + barre de
 *     stats (`PostActions`) sous la photo ;
 *   - droite (panneau) : en-tête + texte du post en haut, commentaires en
 *     dessous (`CommentSection`).
 *
 * Monté uniquement quand une photo est ouverte (état `photoIndex` côté
 * `PostCard`) → l'état interne (index courant, compteur) repart propre.
 */
export function PostPhotoModal({ post, index, onClose }: PostPhotoModalProps) {
  const { t, locale } = useLanguage()
  const [current, setCurrent] = useState(index)
  const [commentCount, setCommentCount] = useState(post.commentsCount)

  const media = post.media
  const item = media[current]
  const hasMany = media.length > 1
  const showPrivateBadge =
    post.author.visibility === 'private' && post.author.id !== currentUserId()

  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose()
      if (e.key === 'ArrowLeft') setCurrent((i) => (i - 1 + media.length) % media.length)
      if (e.key === 'ArrowRight') setCurrent((i) => (i + 1) % media.length)
    }
    window.addEventListener('keydown', onKey)
    const previous = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    return () => {
      window.removeEventListener('keydown', onKey)
      document.body.style.overflow = previous
    }
  }, [onClose, media.length])

  if (!item || typeof document === 'undefined') return null

  // Portail vers <body> : le post est rendu dans une carte en `backdrop-blur` +
  // `transform` (hover), qui confinerait un `position: fixed`. Le portail fait
  // couvrir tout l'écran de l'app.
  return createPortal(
    <div className="fixed inset-0 z-[60] flex flex-col bg-black/85 backdrop-blur-sm lg:flex-row" role="dialog" aria-modal="true">
      {/* Volet média (gauche) */}
      <div className="relative flex min-h-0 flex-1 flex-col bg-black">
        {/* Fermer (haut-gauche, façon X) */}
        <button
          type="button"
          onClick={onClose}
          aria-label={t('common.close')}
          className="absolute left-3 top-3 z-10 rounded-full bg-white/10 p-2.5 text-white transition hover:bg-white/20"
        >
          <X className="h-5 w-5" />
        </button>

        <div className="flex flex-1 items-center justify-center overflow-hidden p-4">
          {item.type === 'video' ? (
            <video src={item.url} controls playsInline className="h-full w-full object-contain" />
          ) : (
            // eslint-disable-next-line @next/next/no-img-element
            <img src={item.url} alt="" className="h-full w-full object-contain" />
          )}
        </div>

        {/* Navigation entre médias */}
        {hasMany && (
          <>
            <NavArrow
              side="left"
              label={t('media.previous')}
              onClick={() => setCurrent((i) => (i - 1 + media.length) % media.length)}
            />
            <NavArrow
              side="right"
              label={t('media.next')}
              onClick={() => setCurrent((i) => (i + 1) % media.length)}
            />
            <div className="absolute bottom-16 left-1/2 -translate-x-1/2 rounded-full bg-black/50 px-3 py-1 text-xs font-medium text-white">
              {current + 1} / {media.length}
            </div>
          </>
        )}

        {/* Stats sous la photo */}
        <div className="border-t border-white/10 px-6 py-2 text-white/80">
          <PostActions post={post} commentCount={commentCount} onComment={() => {}} />
        </div>
      </div>

      {/* Volet infos (droite) : message en haut, commentaires en dessous */}
      <div className="panel flex min-h-0 w-full flex-col border-l lg:w-[400px]">
        <div className="flex items-start gap-3 border-b px-4 py-3">
          <ProfilLink author={post.author} className="shrink-0">
            <Avatar className="h-10 w-10">
              {post.author.avatarUrl && <AvatarImage src={post.author.avatarUrl} alt="" />}
              <AvatarFallback className="bg-gradient-to-br from-[var(--brand-from)] via-[var(--brand-via)] to-[var(--brand-to)] font-bold text-white">
                {(post.author.displayName.charAt(0) || '?').toUpperCase()}
              </AvatarFallback>
            </Avatar>
          </ProfilLink>
          <div className="flex min-w-0 flex-1 flex-col">
            <div className="flex min-w-0 items-center gap-1.5 text-sm">
              <ProfilLink author={post.author} className="truncate font-bold text-foreground hover:underline">
                {post.author.displayName}
              </ProfilLink>
              {post.author.username && (
                <ProfilLink author={post.author} className="shrink-0 text-muted-foreground hover:underline">
                  @{post.author.username}
                </ProfilLink>
              )}
              {showPrivateBadge && (
                <PrivateAuthorBadge label={t('post.private_account_tooltip')} />
              )}
              <span className="shrink-0 text-muted-foreground">·</span>
              <span className="shrink-0 text-muted-foreground">{timeAgo(post.createdAt, locale)}</span>
            </div>
            {post.content && (
              <TranslatedContent
                contentId={`post:${post.id}`}
                content={post.content}
                className="mt-1 whitespace-pre-wrap break-words text-sm leading-relaxed text-foreground/80"
              />
            )}
          </div>
        </div>

        {/* Commentaires (toujours visibles ici) */}
        <div className="min-h-0 flex-1 overflow-y-auto px-4 py-2">
          <CommentSection
            postId={post.id}
            onCountChange={(delta) => setCommentCount((n) => Math.max(0, n + delta))}
          />
        </div>
      </div>
    </div>,
    document.body,
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

function NavArrow({
  side,
  label,
  onClick,
}: {
  side: 'left' | 'right'
  label: string
  onClick: () => void
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-label={label}
      className={cn(
        'absolute top-1/2 -translate-y-1/2 rounded-full bg-black/50 p-2 text-white transition hover:bg-black/70',
        side === 'left' ? 'left-3' : 'right-3',
      )}
    >
      {side === 'left' ? <ChevronLeft className="h-6 w-6" /> : <ChevronRight className="h-6 w-6" />}
    </button>
  )
}
