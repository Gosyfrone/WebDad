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
  Search,
  Smile,
  Trash2,
  Users,
  X,
} from 'lucide-react'

import { cn, initialOf } from '@/lib/utils'
import { exceedsMediaLimit, MAX_MEDIA_MB, resolveMediaUrl, uploadedMediaUrl, uploadMedia } from '@/lib/media'
import {
  createPost,
  notifyPostCreated,
  pinPost,
  type FeedPost,
  type PollAudience,
  type PostMedia,
  type ReplyAudience,
} from '@/lib/posts'
import { usePollDraft, type PollChoiceDraft } from '@/lib/use-poll-draft'
import { searchGifs, type GiphyGif } from '@/lib/giphy'
import { useToast } from '@/hooks/use-toast'
import { useMention } from '@/lib/use-mention'
import { mentionSearchGlobal } from '@/lib/mention-search'
import { useHashtag } from '@/lib/use-hashtag'
import { hashtagSearchGlobal } from '@/lib/hashtag-search'
import { playAppSound } from '@/lib/sounds'
import { useT } from '@/components/language-provider'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
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
import { ActivityPresenceDot } from '@/components/profil/activity-presence-dot'
import { useCurrentUser } from '@/components/current-user-provider'

/** Limite de caractères par défaut (aligné sur le validateur post-service). */
const MAX_CHARS = 280
/** Limite relevée pour les modérateurs/admins (garde-fou absolu, aligné back). */
const MAX_CHARS_PRIVILEGED = 4000
/** Nombre maximal de médias par post (aligné sur le validateur post-service). */
const MAX_MEDIA = 4

interface PostComposerProps {
  className?: string
  autoFocus?: boolean
  submitLabel?: string
  onPosted?: (content: string) => void
  /** Post cité, affiché sous le champ et envoyé comme `quote_post_id`. */
  quotePost?: FeedPost
}

/** Formulaire de post partagé entre le fil inline et la popup sidebar (validation centralisée). */
export function PostComposer({
  className,
  autoFocus = false,
  submitLabel,
  onPosted,
  quotePost,
}: PostComposerProps) {
  const t = useT()
  const { toast } = useToast()
  const { profil, isModerator } = useCurrentUser()
  const label = submitLabel ?? t('nav.post')
  const [content, setContent] = useState('')
  const [media, setMedia] = useState<PostMedia[]>([])
  const [uploadingMedia, setUploadingMedia] = useState(false)
  const userId = profil?.userId ?? ''
  const avatarUrl = profil?.avatarUrl ?? ''
  const initial = initialOf(profil?.displayName, profil?.username)
  const [submitting, setSubmitting] = useState(false)
  const [pinOnProfile, setPinOnProfile] = useState(false)
  const [nsfw, setNsfw] = useState(false)
  const poll = usePollDraft()
  const [replyAudience, setReplyAudience] = useState<ReplyAudience>('everyone')
  const textareaRef = useRef<HTMLTextAreaElement>(null)
  const highlightRef = useRef<HTMLDivElement>(null)
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
  // Limite relevée pour modos/admins (l'enforcement réel vit côté post-service).
  const maxChars = isModerator ? MAX_CHARS_PRIVILEGED : MAX_CHARS
  const remaining = maxChars - content.length
  const isEmpty = content.trim().length === 0 && media.length === 0 && !poll.payload
  const isOver = remaining < 0

  // Le texte visible est peint par l'overlay `ComposerHighlight` (le textarea est
  // transparent). Quand le textarea scrolle au-delà de 3 lignes, on aligne le
  // scroll de l'overlay sinon le texte du bas devient invisible / se superpose.
  function syncHighlightScroll() {
    if (highlightRef.current && textareaRef.current) {
      highlightRef.current.scrollTop = textareaRef.current.scrollTop
    }
  }

  // Insertions programmatiques (emoji, mention/hashtag) modifient le scroll sans
  // déclencher `onScroll` → resync après chaque mise à jour du contenu.
  useEffect(() => {
    if (highlightRef.current && textareaRef.current) {
      highlightRef.current.scrollTop = textareaRef.current.scrollTop
    }
  }, [content])

  async function handleSubmit() {
    if (isEmpty || isOver || submitting || uploadingMedia) return
    setSubmitting(true)
    try {
      const created = await createPost(content.trim(), media, quotePost?.id, poll.payload, replyAudience, nsfw)
      const post = pinOnProfile ? await pinPost(created.id) : created
      notifyPostCreated(post) // le fil prépend sans refetch
      playAppSound('breeze_posted')
      onPosted?.(content)
      setContent('')
      setMedia([])
      setPinOnProfile(false)
      setNsfw(false)
      setReplyAudience('everyone')
      poll.reset()
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
          const uploaded = await uploadMedia(file)
          return { url: uploadedMediaUrl(uploaded, 'large'), type: uploaded.kind } as PostMedia
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

  function addGif(gif: GiphyGif) {
    if (media.length >= MAX_MEDIA) {
      toast({ title: t('composer.media_max', { count: MAX_MEDIA }), variant: 'destructive' })
      return
    }
    setMedia((prev) => [...prev, { url: gif.url, type: 'image' }])
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
        <ActivityPresenceDot userId={userId} />
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
            onScroll={syncHighlightScroll}
            placeholder={t('composer.placeholder')}
            rows={3}
            autoFocus={autoFocus}
            className={cn(
              'relative z-10 w-full cursor-text resize-none bg-transparent text-xl caret-[#5B6CFF] placeholder:text-muted-foreground focus:outline-none',
              content ? 'text-transparent' : 'text-foreground',
            )}
          />
          {content && <ComposerHighlight ref={highlightRef} text={content} />}
          <MentionAutocomplete controller={mention} placement="bottom" />
          <HashtagAutocomplete controller={hashtag} placement="bottom" className="left-24" />
        </div>

        {media.length > 0 && (
          <MediaPreviews media={media} onRemove={removeMedia} removeLabel={t('composer.media_remove')} />
        )}

        {poll.open && (
          <PollPanel
            choices={poll.choices}
            days={poll.days}
            hours={poll.hours}
            minutes={poll.minutes}
            minMinutes={poll.minMinutes}
            audience={poll.audience}
            uploadingChoice={poll.uploadingChoice}
            onChoiceChange={poll.updateChoice}
            onChoiceImage={poll.pickChoiceImage}
            onRemoveChoiceImage={poll.removeChoiceImage}
            onAddChoice={poll.addChoice}
            onRemoveChoice={poll.removeChoice}
            onDaysChange={poll.setDays}
            onHoursChange={poll.setHours}
            onMinutesChange={poll.setMinutes}
            onAudienceChange={poll.setAudience}
            onRemove={poll.reset}
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
            <GifPicker
              disabled={uploadingMedia || media.length >= MAX_MEDIA}
              onSelect={addGif}
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
              aria-pressed={poll.open}
              onClick={() => poll.setOpen((v) => !v)}
              className={cn(
                'rounded-full p-2 transition-colors hover:bg-primary/10',
                poll.open && 'bg-primary/10 text-primary',
              )}
            >
              <ListChecks className="h-5 w-5" />
            </button>
            <button
              type="button"
              title={t('nsfw.composer_tooltip')}
              aria-label={t('nsfw.composer_tooltip')}
              aria-pressed={nsfw}
              onClick={() => setNsfw((v) => !v)}
              className={cn(
                'rounded-full px-2.5 py-1 text-xs font-bold uppercase tracking-wide transition-colors hover:bg-primary/10',
                nsfw ? 'text-foreground' : 'text-muted-foreground/50',
              )}
            >
              {t('nsfw.post_badge')}
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

function GifPicker({
  disabled,
  onSelect,
}: {
  disabled?: boolean
  onSelect: (gif: GiphyGif) => void
}) {
  const t = useT()
  const { toast } = useToast()
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState('')
  const [gifs, setGifs] = useState<GiphyGif[]>([])
  const [loading, setLoading] = useState(false)
  const [failed, setFailed] = useState(false)

  useEffect(() => {
    if (!open) return
    let cancelled = false
    const handle = window.setTimeout(() => {
      setLoading(true)
      setFailed(false)
      searchGifs(query)
        .then((items) => {
          if (cancelled) return
          setGifs(items)
        })
        .catch(() => {
          if (cancelled) return
          setFailed(true)
          setGifs([])
        })
        .finally(() => {
          if (!cancelled) setLoading(false)
        })
    }, query.trim() ? 250 : 0)

    return () => {
      cancelled = true
      window.clearTimeout(handle)
    }
  }, [open, query])

  function choose(gif: GiphyGif) {
    onSelect(gif)
    setOpen(false)
    toast({ title: t('composer.gif_added') })
  }

  return (
    <Popover open={open} onOpenChange={(next) => !disabled && setOpen(next)}>
      <PopoverTrigger asChild>
        <button
          type="button"
          aria-label={t('composer.add_gif')}
          disabled={disabled}
          className="rounded-full px-2 py-2 text-sm font-black leading-none transition-colors hover:bg-primary/10 disabled:opacity-40"
        >
          GIF
        </button>
      </PopoverTrigger>
      <PopoverContent align="start" className="w-[min(22rem,calc(100vw-2rem))] p-3">
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder={t('composer.gif_search')}
            className="h-10 w-full rounded-md border border-border bg-background py-2 pl-9 pr-3 text-sm outline-none focus:border-primary"
            autoFocus
          />
        </div>

        <div className="mt-3 h-72 overflow-y-auto pr-1">
          {loading && gifs.length === 0 ? (
            <div className="grid h-full place-items-center text-muted-foreground">
              <Loader2 className="h-5 w-5 animate-spin" />
            </div>
          ) : failed ? (
            <div className="grid h-full place-items-center px-6 text-center text-sm text-muted-foreground">
              {t('composer.gif_failed')}
            </div>
          ) : gifs.length === 0 ? (
            <div className="grid h-full place-items-center px-6 text-center text-sm text-muted-foreground">
              {t('composer.gif_empty')}
            </div>
          ) : (
            <div className="grid grid-cols-2 gap-2">
              {gifs.map((gif) => (
                <button
                  key={gif.id}
                  type="button"
                  onClick={() => choose(gif)}
                  className="group overflow-hidden rounded-md border border-border bg-muted transition hover:border-primary"
                >
                  {/* eslint-disable-next-line @next/next/no-img-element */}
                  <img
                    src={gif.previewUrl}
                    alt={gif.title}
                    loading="lazy"
                    className="aspect-square h-full w-full object-cover transition group-hover:scale-[1.03]"
                  />
                </button>
              ))}
            </div>
          )}
        </div>

        <div className="mt-2 text-right text-[10px] font-bold uppercase tracking-wide text-muted-foreground">
          GIPHY
        </div>
      </PopoverContent>
    </Popover>
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

/**
 * Pilule « Qui peut répondre » du sondage, même UX que {@link ReplyAudiencePill}.
 * Menu Radix rendu dans le DOM (donc positionné correctement partout, y compris
 * en émulation mobile DevTools) — le `<select>` natif précédent sortait du champ.
 */
function PollAudiencePill({
  value,
  onChange,
}: {
  value: PollAudience
  onChange: (value: PollAudience) => void
}) {
  const t = useT()
  const Icon = value === 'followers' ? Users : Globe
  const label = value === 'followers' ? t('composer.poll_followers') : t('composer.poll_everyone')
  const options: { value: PollAudience; label: string; icon: typeof Globe }[] = [
    { value: 'everyone', label: t('composer.poll_everyone'), icon: Globe },
    { value: 'followers', label: t('composer.poll_followers'), icon: Users },
  ]
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          aria-label={t('composer.reply_audience')}
          className="inline-flex items-center gap-1.5 rounded-full px-2 py-1 text-sm font-semibold text-[#5B6CFF] transition-colors hover:bg-primary/10"
        >
          <Icon className="h-4 w-4" />
          <span>{label}</span>
          <ChevronDown className="h-4 w-4" />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="w-72">
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
  )
}

function PollPanel({
  choices,
  days,
  hours,
  minutes,
  minMinutes,
  audience,
  uploadingChoice,
  onChoiceChange,
  onChoiceImage,
  onRemoveChoiceImage,
  onAddChoice,
  onRemoveChoice,
  onDaysChange,
  onHoursChange,
  onMinutesChange,
  onAudienceChange,
  onRemove,
}: {
  choices: PollChoiceDraft[]
  days: number
  hours: number
  minutes: number
  minMinutes: number
  audience: PollAudience
  uploadingChoice: number | null
  onChoiceChange: (index: number, value: string) => void
  onChoiceImage: (index: number, file: File) => void
  onRemoveChoiceImage: (index: number) => void
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
          <PollChoiceRow
            key={index}
            choice={choice}
            index={index}
            uploading={uploadingChoice === index}
            canRemove={choices.length > 2}
            onChoiceChange={onChoiceChange}
            onChoiceImage={onChoiceImage}
            onRemoveChoiceImage={onRemoveChoiceImage}
            onRemoveChoice={onRemoveChoice}
          />
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
          <NumberSelect label={t('composer.poll_minutes')} value={minutes} min={minMinutes} max={59} onChange={onMinutesChange} />
        </div>
      </div>

      <div className="flex flex-col gap-3 border-t border-border p-3 sm:flex-row sm:items-center sm:justify-between">
        <PollAudiencePill value={audience} onChange={onAudienceChange} />
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

/**
 * Une ligne de choix de sondage : encadré image cliquable (upload via media-service,
 * cap 5 Mo géré par le parent) + champ libellé + bouton retirer le choix. L'image est
 * optionnelle ; un sondage peut mélanger choix illustrés et choix texte seul.
 */
function PollChoiceRow({
  choice,
  index,
  uploading,
  canRemove,
  onChoiceChange,
  onChoiceImage,
  onRemoveChoiceImage,
  onRemoveChoice,
}: {
  choice: PollChoiceDraft
  index: number
  uploading: boolean
  canRemove: boolean
  onChoiceChange: (index: number, value: string) => void
  onChoiceImage: (index: number, file: File) => void
  onRemoveChoiceImage: (index: number) => void
  onRemoveChoice: (index: number) => void
}) {
  const t = useT()
  const inputRef = useRef<HTMLInputElement>(null)
  return (
    <div className="flex items-center gap-2">
      <input
        ref={inputRef}
        type="file"
        accept="image/*"
        className="hidden"
        onChange={(e) => {
          const file = e.target.files?.[0]
          e.target.value = '' // permet de re-sélectionner le même fichier
          if (file) onChoiceImage(index, file)
        }}
      />
      <div className="relative h-12 w-12 shrink-0">
        <button
          type="button"
          aria-label={t('composer.poll_choice_image', { number: index + 1 })}
          onClick={() => inputRef.current?.click()}
          disabled={uploading}
          className="grid h-12 w-12 place-items-center overflow-hidden rounded-lg border border-dashed border-border bg-muted/60 text-muted-foreground transition hover:border-primary hover:text-primary disabled:opacity-60"
        >
          {uploading ? (
            <Loader2 className="h-5 w-5 animate-spin" />
          ) : choice.imageUrl ? (
            <img src={resolveMediaUrl(choice.imageUrl)} alt="" className="h-full w-full object-cover" />
          ) : (
            <ImageIcon className="h-5 w-5" />
          )}
        </button>
        {choice.imageUrl && !uploading && (
          <button
            type="button"
            aria-label={t('composer.poll_choice_image_remove')}
            onClick={() => onRemoveChoiceImage(index)}
            className="absolute -right-1.5 -top-1.5 grid h-5 w-5 place-items-center rounded-full bg-background text-muted-foreground shadow ring-1 ring-border transition hover:text-destructive"
          >
            <X className="h-3 w-3" />
          </button>
        )}
      </div>
      <input
        value={choice.label}
        onChange={(e) => onChoiceChange(index, e.target.value)}
        placeholder={t('composer.poll_choice', { number: index + 1 })}
        maxLength={80}
        className="h-12 min-w-0 flex-1 rounded-md border border-border bg-background px-3 text-sm outline-none focus:border-primary"
      />
      {canRemove && (
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
  )
}

function NumberSelect({
  label,
  value,
  min = 0,
  max,
  onChange,
}: {
  label: string
  value: number
  min?: number
  max: number
  onChange: (value: number) => void
}) {
  // État texte local : autorise la saisie transitoire (champ vidé pour retaper)
  // sans imposer le clamp à chaque frappe. Le clamp ne s'applique qu'à la valeur
  // remontée au parent. Resync si le parent change la valeur (ex. clamp auto des
  // minutes quand jours/heures repassent à 0).
  const [text, setText] = useState(String(value))
  useEffect(() => setText(String(value)), [value])

  function handleChange(next: string) {
    setText(next)
    if (next === '') return
    const n = Number(next)
    if (Number.isNaN(n)) return
    onChange(Math.min(max, Math.max(min, Math.trunc(n))))
  }

  return (
    <label className="flex min-w-0 flex-col gap-1 rounded-md border border-border bg-background px-3 py-2 text-xs text-muted-foreground">
      {label}
      <input
        type="number"
        inputMode="numeric"
        value={text}
        min={min}
        max={max}
        onChange={(e) => handleChange(e.target.value)}
        onBlur={() => setText(String(value))}
        className="bg-transparent text-base text-foreground outline-none"
      />
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
          key={`${m.url}-${i}`}
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
