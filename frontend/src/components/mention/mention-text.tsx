'use client'

import { Fragment, useState } from 'react'
import type { ReactNode } from 'react'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { Loader2 } from 'lucide-react'

import { getUserByUsername } from '@/lib/api'
import { parseMentionSegments } from '@/lib/mentions'
import { ROUTES, profilHref } from '@/lib/routes'
import { cn } from '@/lib/utils'
import type { RelationUser } from '@/types'
import { useLanguage } from '@/components/language-provider'
import { ActivityPresenceDot } from '@/components/profil/activity-presence-dot'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'

const MENTION_CLASS = 'font-semibold text-[#5B6CFF] hover:underline dark:text-[#9aa6ff]'
const HASHTAG_CLASS = 'font-semibold text-[#5B6CFF] hover:underline dark:text-[#9aa6ff]'
// Variante sur fond d'accent (bulle « mine » au dégradé violet/cyan, texte
// blanc) : le bleu de marque se fondrait dans le dégradé → blanc gras souligné.
const MENTION_CLASS_ACCENT =
  'font-semibold text-white underline underline-offset-2 hover:opacity-80'

/**
 * Rend un texte en transformant les @handle en liens vers le profil. Utilisé
 * pour les posts et les commentaires (convention « identité cliquable »).
 */
export function MentionText({ text, className }: { text: string; className?: string }) {
  const segments = parseMentionSegments(text)
  return (
    <span className={className}>
      {segments.map((seg, i) =>
        seg.type === 'text' ? (
          <Fragment key={i}>{renderHashtags(seg.text, `text-${i}`)}</Fragment>
        ) : (
          <Link key={i} href={profilHref(seg.handle)} className={MENTION_CLASS}>
            {seg.raw}
          </Link>
        ),
      )}
    </span>
  )
}

function renderHashtags(text: string, keyPrefix: string): ReactNode[] {
  const nodes: ReactNode[] = []
  const re = /(^|[^\p{L}\p{N}_#])#([\p{L}\p{N}_]{1,64})/gu
  let last = 0
  let match: RegExpExecArray | null

  while ((match = re.exec(text)) !== null) {
    const boundary = match[1]
    const tag = match[2]
    const hashIndex = match.index + boundary.length
    if (!/\p{L}/u.test(tag)) continue
    if (hashIndex > last) nodes.push(text.slice(last, hashIndex))
    nodes.push(
      <Link
        key={`${keyPrefix}-${hashIndex}`}
        href={`${ROUTES.feed}?hashtag=${encodeURIComponent(tag.toLowerCase())}`}
        className={HASHTAG_CLASS}
      >
        #{tag}
      </Link>,
    )
    last = re.lastIndex
  }
  if (last < text.length) nodes.push(text.slice(last))
  return nodes
}

/**
 * Variante messagerie : un @handle d'un MEMBRE de la conversation mène à son
 * profil (décision produit) ; un @handle d'un NON-membre ouvre une carte d'aperçu
 * (avatar / nom / @handle) dont le clic renvoie vers la recherche (loupe).
 *
 * `memberUsernames` = handles (minuscule) des membres de la conversation.
 */
export function MentionMessageText({
  text,
  memberUsernames,
  className,
  onAccent = false,
}: {
  text: string
  memberUsernames: Set<string>
  className?: string
  /** Rendu sur une bulle au fond d'accent (dégradé, texte blanc). */
  onAccent?: boolean
}) {
  const mentionClass = onAccent ? MENTION_CLASS_ACCENT : MENTION_CLASS
  const segments = parseMentionSegments(text)
  return (
    <span className={className}>
      {segments.map((seg, i) => {
        if (seg.type === 'text') return <Fragment key={i}>{seg.text}</Fragment>
        if (memberUsernames.has(seg.handle.toLowerCase())) {
          return (
            <Link key={i} href={profilHref(seg.handle)} className={mentionClass}>
              {seg.raw}
            </Link>
          )
        }
        return (
          <NonMemberMention key={i} handle={seg.handle} raw={seg.raw} triggerClass={mentionClass} />
        )
      })}
    </span>
  )
}

/** Mention d'un non-membre : carte d'aperçu au clic → recherche au clic suivant. */
function NonMemberMention({
  handle,
  raw,
  triggerClass = MENTION_CLASS,
}: {
  handle: string
  raw: string
  triggerClass?: string
}) {
  const { t } = useLanguage()
  const router = useRouter()
  const [user, setUser] = useState<RelationUser | null>(null)
  const [loading, setLoading] = useState(false)
  const [loaded, setLoaded] = useState(false)

  function onOpenChange(open: boolean) {
    if (open && !loaded) {
      setLoading(true)
      getUserByUsername(handle)
        .then((u) => setUser(u))
        .finally(() => {
          setLoading(false)
          setLoaded(true)
        })
    }
  }

  function goToSearch() {
    router.push(`${ROUTES.explorer}?q=${encodeURIComponent('@' + handle)}`)
  }

  return (
    <Popover onOpenChange={onOpenChange}>
      <PopoverTrigger asChild>
        <button type="button" className={cn(triggerClass, 'cursor-pointer')}>
          {raw}
        </button>
      </PopoverTrigger>
      <PopoverContent align="start" className="w-64 p-0">
        {loading ? (
          <div className="flex items-center justify-center py-6">
            <Loader2 className="h-4 w-4 animate-spin text-[#5B6CFF]" aria-hidden />
          </div>
        ) : user ? (
          <button
            type="button"
            onClick={goToSearch}
            className="flex w-full items-center gap-3 rounded-md p-3 text-left transition-colors hover:bg-accent"
          >
            <Avatar className="h-11 w-11 shrink-0">
              {user.avatarUrl && <AvatarImage src={user.avatarUrl} alt="" />}
              <AvatarFallback className="bg-gradient-to-br from-[var(--brand-from)] via-[var(--brand-via)] to-[var(--brand-to)] font-bold text-white">
                {(user.displayName || user.username || 'U').charAt(0).toUpperCase()}
              </AvatarFallback>
              <ActivityPresenceDot userId={user.id} />
            </Avatar>
            <div className="flex min-w-0 flex-col">
              <span className="truncate font-bold text-foreground">{user.displayName}</span>
              <span className="truncate text-sm text-muted-foreground">@{user.username}</span>
              <span className="mt-1 text-xs text-[#5B6CFF] dark:text-[#9aa6ff]">
                {t('mentions.view_in_search')}
              </span>
            </div>
          </button>
        ) : (
          <p className="px-3 py-4 text-center text-sm text-muted-foreground">
            {t('mentions.user_not_found')}
          </p>
        )}
      </PopoverContent>
    </Popover>
  )
}
