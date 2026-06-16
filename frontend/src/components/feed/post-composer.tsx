'use client'

import { useEffect, useRef, useState } from 'react'
import {
  Check,
  ChevronDown,
  Globe,
  Image as ImageIcon,
  ListChecks,
  Loader2,
  Pin,
  Plus,
  Smile,
  Trash2,
  Users,
  X,
} from 'lucide-react'

import { cn } from '@/lib/utils'
import { getAccessToken } from '@/lib/auth-client'
import { getMyProfil, subscribeProfilUpdated } from '@/lib/profil-client'
import { exceedsMediaLimit, MAX_MEDIA_MB, resolveMediaUrl, uploadMedia } from '@/lib/media'
import {
  createPost,
  notifyPostCreated,
  pinPost,
  type CreatePollPayload,
  type FeedPost,
  type PollAudience,
  type PostMedia,
  type ReplyAudience,
} from '@/lib/posts'
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
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
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
  const [pollOpen, setPollOpen] = useState(false)
  const [pollChoices, setPollChoices] = useState(['', ''])
  const [pollDays, setPollDays] = useState(0)
  const [pollHours, setPollHours] = useState(0)
  const [pollMinutes, setPollMinutes] = useState(10)
  const [pollAudience, setPollAudience] = useState<PollAudience>('everyone')
  const [replyAudience, setReplyAudience] = useState<ReplyAudience>('everyone')
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
  const pollPayload = buildPollPayload(pollOpen, pollChoices, pollDays, pollHours, pollMinutes, pollAudience)
  const isEmpty = content.trim().length === 0 && media.length === 0 && !pollPayload
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
      const created = await createPost(content.trim(), media, quotePost?.id, pollPayload, replyAudience)
      const post = pinOnProfile ? await pinPost(created.id) : created
      notifyPostCreated(post) // le fil prépend sans refetch
      onPosted?.(content)
      setContent('')
      setMedia([])
      setPinOnProfile(false)
      setReplyAudience('everyone')
      resetPoll()
    } catch {
      toast({ title: t('composer.post_failed'), variant: 'destructive' })
    } finally {
      setSubmitting(false)
    }
  }

  /** Uploade les fichiers choisis au media-service et les ajoute aux médias joints. */
  async function handleFiles(e: React.ChangeEvent<HTMLInputElement>) {
    const picked = Array.from(e.target.files ?? [])
    e.target.value = '' // permet de re-sélectionner les mêmes fichiers
    if (picked.length === 0) return

    // Garde UX : on écarte les fichiers trop lourds avant tout upload (le cap
    // est aussi appliqué côté serveur). Les admins ne sont pas plafonnés.
    const files = picked.filter((f) => !exceedsMediaLimit(f.size))
    if (files.length < picked.length) {
      toast({ title: t('media.too_large', { max: MAX_MEDIA_MB }), variant: 'brand' })
    }
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

  function resetPoll() {
    setPollOpen(false)
    setPollChoices(['', ''])
    setPollDays(0)
    setPollHours(0)
    setPollMinutes(10)
    setPollAudience('everyone')
  }

  function updatePollChoice(index: number, value: string) {
    setPollChoices((prev) => prev.map((choice, i) => (i === index ? value : choice)))
  }

  function addPollChoice() {
    setPollChoices((prev) => (prev.length >= 4 ? prev : [...prev, '']))
  }

  function removePollChoice(index: number) {
    setPollChoices((prev) => (prev.length <= 2 ? prev : prev.filter((_, i) => i !== index)))
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

        {pollOpen && (
          <PollPanel
            choices={pollChoices}
            days={pollDays}
            hours={pollHours}
            minutes={pollMinutes}
            audience={pollAudience}
            onChoiceChange={updatePollChoice}
            onAddChoice={addPollChoice}
            onRemoveChoice={removePollChoice}
            onDaysChange={setPollDays}
            onHoursChange={setPollHours}
            onMinutesChange={setPollMinutes}
            onAudienceChange={setPollAudience}
            onRemove={resetPoll}
          />
        )}

        {quotePost && (
          <QuotePreview post={quotePost} />
        )}

        <ReplyAudiencePill value={replyAudience} onChange={setReplyAudience} />

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
            <button
              type="button"
              aria-label={t('composer.add_poll')}
              aria-pressed={pollOpen}
              onClick={() => setPollOpen((v) => !v)}
              className={cn(
                'rounded-full p-2 transition-colors hover:bg-primary/10',
                pollOpen && 'bg-primary/10 text-primary',
              )}
            >
              <ListChecks className="h-5 w-5" />
            </button>
            <EmojiPicker onSelect={insertEmoji}>
              <button
                type="button"
                aria-label={t('composer.add_emoji')}
                className="rounded-full p-2 transition-colors hover:bg-primary/10"
              >
                <Smile className="h-5 w-5" />
              </button>
            </EmojiPicker>
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
              disabled={isEmpty || isOver || submitting || uploadingMedia}
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

/**
 * Pilule « Qui peut répondre » (façon X) : bouton toujours visible sous le texte
 * qui ouvre un menu Tout le monde / Abonnés. Le défaut est `everyone`.
 */
function ReplyAudiencePill({
  value,
  onChange,
}: {
  value: ReplyAudience
  onChange: (value: ReplyAudience) => void
}) {
  const t = useT()
  const Icon = value === 'followers' ? Users : Globe
  const pill = value === 'followers' ? t('composer.reply_pill_followers') : t('composer.reply_pill_everyone')
  const options: { value: ReplyAudience; label: string; icon: typeof Globe }[] = [
    { value: 'everyone', label: t('composer.reply_everyone'), icon: Globe },
    { value: 'followers', label: t('composer.reply_followers'), icon: Users },
  ]
  return (
    <div className="-mt-1">
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <button
            type="button"
            aria-label={t('composer.reply_audience')}
            className="inline-flex items-center gap-1.5 rounded-full px-2 py-1 text-sm font-semibold text-[#5B6CFF] transition-colors hover:bg-primary/10"
          >
            <Icon className="h-4 w-4" />
            <span>{pill}</span>
            <ChevronDown className="h-4 w-4" />
          </button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="start" className="w-60">
          <p className="px-2 py-1.5 text-sm font-bold">{t('composer.reply_audience')}</p>
          {options.map((opt) => (
            <DropdownMenuItem
              key={opt.value}
              onClick={() => onChange(opt.value)}
              className="flex items-center gap-2"
            >
              <opt.icon className="h-4 w-4 text-muted-foreground" />
              <span className="flex-1">{opt.label}</span>
              {value === opt.value && <Check className="h-4 w-4 text-[#5B6CFF]" />}
            </DropdownMenuItem>
          ))}
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  )
}

function buildPollPayload(
  open: boolean,
  choices: string[],
  days: number,
  hours: number,
  minutes: number,
  audience: PollAudience,
): CreatePollPayload | undefined {
  if (!open) return undefined
  const cleaned = choices.map((choice) => choice.trim()).filter(Boolean)
  const durationMinutes = days * 24 * 60 + hours * 60 + minutes
  if (cleaned.length < 2 || durationMinutes < 1) return undefined
  return { choices: cleaned, durationMinutes, audience }
}

function PollPanel({
  choices,
  days,
  hours,
  minutes,
  audience,
  onChoiceChange,
  onAddChoice,
  onRemoveChoice,
  onDaysChange,
  onHoursChange,
  onMinutesChange,
  onAudienceChange,
  onRemove,
}: {
  choices: string[]
  days: number
  hours: number
  minutes: number
  audience: PollAudience
  onChoiceChange: (index: number, value: string) => void
  onAddChoice: () => void
  onRemoveChoice: (index: number) => void
  onDaysChange: (value: number) => void
  onHoursChange: (value: number) => void
  onMinutesChange: (value: number) => void
  onAudienceChange: (value: PollAudience) => void
  onRemove: () => void
}) {
  const t = useT()
  return (
    <div className="overflow-hidden rounded-2xl border border-border bg-background/35">
      <div className="space-y-3 p-3">
        {choices.map((choice, index) => (
          <div key={index} className="flex items-center gap-2">
            <div className="grid h-12 w-12 shrink-0 place-items-center rounded-lg border border-dashed border-border bg-muted/60 text-muted-foreground">
              <ImageIcon className="h-5 w-5" />
            </div>
            <input
              value={choice}
              onChange={(e) => onChoiceChange(index, e.target.value)}
              placeholder={t('composer.poll_choice', { number: index + 1 })}
              maxLength={80}
              className="h-12 min-w-0 flex-1 rounded-md border border-border bg-background px-3 text-sm outline-none focus:border-primary"
            />
            {choices.length > 2 && (
              <button
                type="button"
                aria-label={t('composer.poll_remove_choice')}
                onClick={() => onRemoveChoice(index)}
                className="rounded-full p-2 text-muted-foreground transition hover:bg-destructive/10 hover:text-destructive"
              >
                <X className="h-4 w-4" />
              </button>
            )}
          </div>
        ))}
        {choices.length < 4 && (
          <button
            type="button"
            onClick={onAddChoice}
            className="ml-auto flex items-center gap-1 rounded-full px-2 py-1 text-sm font-semibold text-primary transition hover:bg-primary/10"
          >
            <Plus className="h-4 w-4" />
            {t('composer.poll_add_choice')}
          </button>
        )}
      </div>

      <div className="border-t border-border p-3">
        <p className="mb-2 text-sm font-semibold">{t('composer.poll_duration')}</p>
        <div className="grid grid-cols-3 gap-2">
          <NumberSelect label={t('composer.poll_days')} value={days} max={7} onChange={onDaysChange} />
          <NumberSelect label={t('composer.poll_hours')} value={hours} max={23} onChange={onHoursChange} />
          <NumberSelect label={t('composer.poll_minutes')} value={minutes} max={59} onChange={onMinutesChange} />
        </div>
      </div>

      <div className="flex flex-col gap-3 border-t border-border p-3 sm:flex-row sm:items-center sm:justify-between">
        <label className="flex items-center gap-2 text-sm font-semibold text-primary">
          <select
            value={audience}
            onChange={(e) => onAudienceChange(e.target.value as PollAudience)}
            className="rounded-md border border-border bg-background px-2 py-1 text-foreground outline-none focus:border-primary"
          >
            <option value="everyone">{t('composer.poll_everyone')}</option>
            <option value="followers">{t('composer.poll_followers')}</option>
          </select>
        </label>
        <button
          type="button"
          onClick={onRemove}
          className="flex items-center justify-center gap-2 rounded-full px-3 py-1.5 text-sm font-semibold text-red-500 transition hover:bg-red-500/10"
        >
          <Trash2 className="h-4 w-4" />
          {t('composer.poll_remove')}
        </button>
      </div>
    </div>
  )
}

function NumberSelect({
  label,
  value,
  max,
  onChange,
}: {
  label: string
  value: number
  max: number
  onChange: (value: number) => void
}) {
  return (
    <label className="flex min-w-0 flex-col gap-1 rounded-md border border-border bg-background px-3 py-2 text-xs text-muted-foreground">
      {label}
      <select
        value={value}
        onChange={(e) => onChange(Number(e.target.value))}
        className="bg-transparent text-base text-foreground outline-none"
      >
        {Array.from({ length: max + 1 }, (_, n) => (
          <option key={n} value={n}>
            {n}
          </option>
        ))}
      </select>
    </label>
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
