'use client'

import { useEffect, useRef, useState } from 'react'
import { Image as ImageIcon, Smile, BarChart2, Loader2, Pin, X } from 'lucide-react'

import { cn } from '@/lib/utils'
import { getAccessToken } from '@/lib/auth-client'
import { getMyProfil, subscribeProfilUpdated } from '@/lib/profil-client'
import { resolveMediaUrl, uploadMedia } from '@/lib/media'
import { createPost, notifyPostCreated, pinPost, type FeedPost, type PostMedia } from '@/lib/posts'
import { useToast } from '@/hooks/use-toast'
import type { ProfilDetails } from '@/types'
import { useMention } from '@/lib/use-mention'
import { mentionSearchGlobal } from '@/lib/mention-search'
import { useHashtag } from '@/lib/use-hashtag'
import { hashtagSearchGlobal } from '@/lib/hashtag-search'
import { useT } from '@/components/language-provider'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import { EmojiPicker } from '@/components/feed/emoji-picker'
import { MentionAutocomplete } from '@/components/mention/mention-autocomplete'
import { HashtagAutocomplete } from '@/components/hashtag/hashtag-autocomplete'
import { ComposerHighlight } from '@/components/hashtag/composer-highlight'

const MAX_CHARS = 280
/** Nombre maximal de médias par post (aligné sur le validateur post-service). */
const MAX_MEDIA = 4

interface PostComposerProps {
  /** Classes du conteneur externe (padding/bordure gérés par le parent). */
  className?: string
  /** Place le curseur dans le champ dès le montage (utile en modale). */
  autoFocus?: boolean
  /** Libellé du bouton d'envoi (défaut : « Breezer » traduit). */
  submitLabel?: string
  /** Appelé après une publication réussie (ex. fermer la popup). */
  onPosted?: (content: string) => void
  /** Post cité, affiché sous le champ et envoyé comme quote_post_id. */
  quotePost?: FeedPost
}

/**
 * Formulaire de rédaction d'un post (avatar + zone de saisie + barre d'outils).
 *
 * Partagé entre la zone de composition inline du fil (`CreatePost`) et la
 * popup déclenchée depuis la sidebar (`CreatePostDialog`) — une seule source
 * de vérité pour la limite de caractères et la validation.
 */
export function PostComposer({
  className,
  autoFocus = false,
  submitLabel,
  onPosted,
  quotePost,
}: PostComposerProps) {
  const t = useT()
  const { toast } = useToast()
  const label = submitLabel ?? t('nav.post')
  const [content, setContent] = useState('')
  const [media, setMedia] = useState<PostMedia[]>([])
  const [uploadingMedia, setUploadingMedia] = useState(false)
  const [avatarUrl, setAvatarUrl] = useState('')
  const [initial, setInitial] = useState('U')
  const [submitting, setSubmitting] = useState(false)
  const [pinOnProfile, setPinOnProfile] = useState(false)
  const textareaRef = useRef<HTMLTextAreaElement>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const mention = useMention({
    inputRef: textareaRef,
    onChange: setContent,
    search: mentionSearchGlobal,
  })
  const hashtag = useHashtag({
    inputRef: textareaRef,
    onChange: setContent,
    search: hashtagSearchGlobal,
  })
  const remaining = MAX_CHARS - content.length
  const isEmpty = content.trim().length === 0 && media.length === 0
  const isOver = remaining < 0

  // Avatar de l'utilisateur courant (resync sur édition du profil, comme la
  // sidebar). Repli silencieux sur l'initiale si la session/le profil manque.
  useEffect(() => {
    let cancelled = false
    function apply(profil: ProfilDetails) {
      if (cancelled) return
      setAvatarUrl(profil.avatarUrl)
      setInitial((profil.displayName || profil.username || 'U').charAt(0).toUpperCase())
    }
    // Sans token (visiteur, ou course d'hydratation avant que `isVisitor` ne
    // bascule) : pas de profil à charger. `/profils/me` renverrait 401 →
    // refresh raté → redirection forcée vers /login.
    if (getAccessToken()) getMyProfil().then(apply).catch(() => {})
    const unsubscribe = subscribeProfilUpdated(apply)
    return () => {
      cancelled = true
      unsubscribe()
    }
  }, [])

  async function handleSubmit() {
    if (isEmpty || isOver || submitting || uploadingMedia) return
    setSubmitting(true)
    try {
      const created = await createPost(content.trim(), media, quotePost?.id)
      const post = pinOnProfile ? await pinPost(created.id) : created
      notifyPostCreated(post) // le fil prépend sans refetch
      onPosted?.(content)
      setContent('')
      setMedia([])
      setPinOnProfile(false)
    } catch {
      toast({ title: t('composer.post_failed'), variant: 'destructive' })
    } finally {
      setSubmitting(false)
    }
  }

  /** Uploade les fichiers choisis au media-service et les ajoute aux médias joints. */
  async function handleFiles(e: React.ChangeEvent<HTMLInputElement>) {
    const files = Array.from(e.target.files ?? [])
    e.target.value = '' // permet de re-sélectionner les mêmes fichiers
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

  /** Insère l'emoji à la position du curseur (ou à la fin) et restaure le focus. */
  function insertEmoji(emoji: string) {
    const el = textareaRef.current
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

  return (
    <div className={cn('flex gap-3', className)}>
      <Avatar className="mt-1 h-10 w-10 shrink-0 shadow-[0_12px_30px_rgba(91,108,255,0.22)]">
        {avatarUrl && <AvatarImage src={avatarUrl} alt="" />}
        <AvatarFallback className="bg-gradient-to-br from-[var(--brand-from)] via-[var(--brand-via)] to-[var(--brand-to)] font-bold text-white">
          {initial}
        </AvatarFallback>
      </Avatar>

      <div className="flex min-w-0 flex-1 flex-col gap-3">
        <div className="relative isolate">
          <textarea
            ref={textareaRef}
            value={content}
            onChange={(e) => {
              setContent(e.target.value)
              mention.sync()
              hashtag.sync()
            }}
            onKeyDown={(event) => {
              if (hashtag.onKeyDown(event)) return
              mention.onKeyDown(event)
            }}
            onKeyUp={() => {
              mention.sync()
              hashtag.sync()
            }}
            onClick={() => {
              mention.sync()
              hashtag.sync()
            }}
            placeholder={t('composer.placeholder')}
            rows={3}
            autoFocus={autoFocus}
            className={cn(
              'relative z-10 w-full cursor-text resize-none bg-transparent text-xl caret-[#5B6CFF] placeholder:text-muted-foreground focus:outline-none',
              content ? 'text-transparent' : 'text-foreground',
            )}
          />
          {content && <ComposerHighlight text={content} />}
          <MentionAutocomplete controller={mention} placement="bottom" />
          <HashtagAutocomplete controller={hashtag} placement="bottom" className="left-24" />
        </div>

        {media.length > 0 && (
          <MediaPreviews media={media} onRemove={removeMedia} removeLabel={t('composer.media_remove')} />
        )}

        {quotePost && (
          <QuotePreview post={quotePost} />
        )}

        <Separator className="bg-border" />

        {/* Toolbar */}
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-1 text-[#5B6CFF]">
            <button
              type="button"
              aria-label={t('composer.add_image')}
              onClick={() => fileInputRef.current?.click()}
              disabled={uploadingMedia || media.length >= MAX_MEDIA}
              className="rounded-full p-2 transition-colors hover:bg-primary/10 disabled:opacity-40"
            >
              {uploadingMedia ? (
                <Loader2 className="h-5 w-5 animate-spin" />
              ) : (
                <ImageIcon className="h-5 w-5" />
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
                className="rounded-full p-2 transition-colors hover:bg-primary/10"
              >
                <Smile className="h-5 w-5" />
              </button>
            </EmojiPicker>
            <ActionIcon icon={BarChart2} label={t('composer.add_poll')} />
            <button
              type="button"
              aria-label={t('composer.pin_profile')}
              aria-pressed={pinOnProfile}
              onClick={() => setPinOnProfile((v) => !v)}
              className={cn(
                'rounded-full p-2 transition-colors hover:bg-primary/10',
                pinOnProfile && 'bg-primary/10 text-primary',
              )}
            >
              <Pin className={cn('h-5 w-5', pinOnProfile && 'fill-current')} />
            </button>
          </div>

          <div className="flex items-center gap-3">
            {/* Compteur de caractères */}
            {content.length > 0 && (
              <span
                className={
                  isOver
                    ? 'text-sm font-bold text-destructive'
                    : remaining <= 20
                      ? 'text-sm text-amber-500'
                      : 'text-sm text-muted-foreground'
                }
              >
                {remaining}
              </span>
            )}

            <Button
              size="sm"
              className="rounded-full bg-gradient-to-r from-[var(--brand-from)] via-[var(--brand-via)] to-[var(--brand-to)] font-bold text-white shadow-[0_12px_30px_rgba(91,108,255,0.28)]"
              disabled={isEmpty || isOver || submitting}
              onClick={handleSubmit}
            >
              {submitting ? <Loader2 className="h-4 w-4 animate-spin" /> : label}
            </Button>
          </div>
        </div>
      </div>
    </div>
  )
}

function MediaPreviews({
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
        <div
          key={m.url}
          className="group relative overflow-hidden rounded-xl border border-border bg-background/45"
        >
          {m.type === 'video' ? (
            <video
              src={resolveMediaUrl(m.url)}
              className="max-h-72 w-full object-cover"
              muted
              playsInline
            />
          ) : (
            // eslint-disable-next-line @next/next/no-img-element
            <img src={resolveMediaUrl(m.url)} alt="" className="max-h-72 w-full object-cover" />
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

function QuotePreview({ post }: { post: FeedPost }) {
  return (
    <div className="rounded-xl border border-border bg-background/45 px-3 py-2 text-sm">
      <div className="mb-1 flex min-w-0 items-center gap-1.5 text-xs">
        <span className="truncate font-bold text-foreground">{post.author.displayName}</span>
        {post.author.username && (
          <span className="shrink-0 text-muted-foreground">@{post.author.username}</span>
        )}
      </div>
      <p className="line-clamp-4 whitespace-pre-wrap break-words text-foreground/75">
        {post.content}
      </p>
      {post.media.length > 0 && <QuoteMediaPreview media={post.media} />}
    </div>
  )
}

function QuoteMediaPreview({ media }: { media: PostMedia[] }) {
  return (
    <div
      className={cn(
        'mt-2 grid gap-1.5 overflow-hidden rounded-xl border border-border',
        media.length > 1 ? 'grid-cols-2' : 'grid-cols-1',
      )}
    >
      {media.map((m, i) => {
        const cellClass = cn(
          media.length === 1 ? 'max-h-64' : 'aspect-square',
          media.length === 3 && i === 0 && 'row-span-2 aspect-auto',
        )
        return (
          <div
            key={m.url}
            className={cn('overflow-hidden bg-background/50', media.length === 3 && i === 0 && 'row-span-2')}
          >
            {m.type === 'video' ? (
              <video
                src={resolveMediaUrl(m.url)}
                className={cn('h-full w-full object-cover', media.length === 1 ? 'max-h-64 aspect-video' : cellClass)}
                muted
                playsInline
                controls
              />
            ) : (
              // eslint-disable-next-line @next/next/no-img-element
              <img
                src={resolveMediaUrl(m.url)}
                alt=""
                loading="lazy"
                className={cn('h-full w-full object-cover', cellClass)}
              />
            )}
          </div>
        )
      })}
    </div>
  )
}

function ActionIcon({ icon: Icon, label }: { icon: React.ElementType; label: string }) {
  return (
    <button
      type="button"
      aria-label={label}
      className="rounded-full p-2 transition-colors hover:bg-primary/10"
    >
      <Icon className="h-5 w-5" />
    </button>
  )
}
