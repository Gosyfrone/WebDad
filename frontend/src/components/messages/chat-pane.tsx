'use client'

import { Fragment, useCallback, useEffect, useRef, useState } from 'react'
import { ArrowLeft, Info, Loader2, Lock, Send } from 'lucide-react'

import { cn } from '@/lib/utils'
import {
  computeDivider,
  listMessagesPage,
  sendMessage,
  type ChatMessage,
  type Conversation,
} from '@/lib/messages'
import { useResolvedUser } from '@/lib/use-resolved-user'
import { useToast } from '@/hooks/use-toast'
import { useLanguage } from '@/components/language-provider'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import { ConversationAvatar, conversationTitle } from '@/components/messages/conversation-meta'

const PAGE = 30

interface ChatPaneProps {
  conversation: Conversation
  myId: string
  /** Dernier message reçu en temps réel pour CETTE conversation (ou null). */
  liveMessage: ChatMessage | null
  /** Id du dernier message LU capturé à l'ouverture (ancre « Nouveaux messages »). */
  dividerAnchor: string | null
  /** Retour à la liste (mobile). */
  onBack: () => void
  /** Ouvre le panneau d'infos / gestion (membres, renommer, quitter…). */
  onOpenInfo: () => void
  /** Notifie le parent d'un message envoyé d'ici (aperçu + ordre de la liste). */
  onLocalMessage: (msg: ChatMessage) => void
}

/** Fusionne des messages plus anciens en tête, en dédupliquant par id. */
function prependUnique(older: ChatMessage[], current: ChatMessage[]): ChatMessage[] {
  const seen = new Set(current.map((m) => m.id))
  return [...older.filter((m) => !seen.has(m.id)), ...current]
}

/**
 * Volet de conversation : en-tête (interlocuteur / groupe), historique chiffré
 * (déchiffré localement) en défilement infini vers le HAUT (curseur `before`),
 * et zone de saisie. Les nouveaux messages arrivent par WebSocket (prop
 * `liveMessage`) ; l'envoi est géré ici (`sendMessage`).
 *
 * Cas particuliers gérés :
 *   - clé de contenu absente sur cet appareil (`contentKey === null`) → bandeau,
 *     lecture/écriture désactivées (identité créée sur un autre appareil) ;
 *   - communauté en lecture seule (`viewer`) → composer désactivé.
 */
export function ChatPane({
  conversation,
  myId,
  liveMessage,
  dividerAnchor,
  onBack,
  onOpenInfo,
  onLocalMessage,
}: ChatPaneProps) {
  const { t } = useLanguage()
  const { toast } = useToast()

  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [hasMore, setHasMore] = useState(false)
  const [initialLoading, setInitialLoading] = useState(true)
  const [loadingOlder, setLoadingOlder] = useState(false)
  const [draft, setDraft] = useState('')
  const [sending, setSending] = useState(false)
  // Id du message devant lequel afficher « Nouveaux messages » (gelé à l'ouverture).
  const [dividerBeforeId, setDividerBeforeId] = useState<string | null>(null)

  const oldestIdRef = useRef<string | null>(null)
  const scrollRef = useRef<HTMLDivElement | null>(null)
  const topSentinelRef = useRef<HTMLDivElement | null>(null)
  // Référence à la conversation courante (clé à jour pour chiffrer/déchiffrer).
  const convRef = useRef(conversation)
  convRef.current = conversation

  const keyMissing = conversation.contentKey === null
  const readOnly =
    conversation.type === 'community' && conversation.myRole === 'viewer'
  const canSend = !keyMissing && !readOnly

  const scrollToBottom = useCallback((behavior: ScrollBehavior = 'auto') => {
    const el = scrollRef.current
    if (el) el.scrollTo({ top: el.scrollHeight, behavior })
  }, [])

  // Chargement initial (et rechargement à chaque changement de conversation).
  useEffect(() => {
    let cancelled = false
    setInitialLoading(true)
    setMessages([])
    setHasMore(false)
    oldestIdRef.current = null

    listMessagesPage(convRef.current, PAGE)
      .then((page) => {
        if (cancelled) return
        setMessages(page.messages)
        setHasMore(page.hasMore)
        oldestIdRef.current = page.oldestId
        setDividerBeforeId(computeDivider(page.messages, dividerAnchor))
      })
      .catch(() => {
        if (!cancelled) toast({ title: t('messages.load_failed'), variant: 'destructive' })
      })
      .finally(() => {
        if (!cancelled) {
          setInitialLoading(false)
          requestAnimationFrame(() => scrollToBottom('auto'))
        }
      })

    return () => {
      cancelled = true
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [conversation.id])

  // Chargement des messages plus anciens (scroll vers le haut), avec
  // restauration de la position de défilement (on préfixe en tête).
  const loadOlder = useCallback(async () => {
    if (loadingOlder || !hasMore || !oldestIdRef.current) return
    setLoadingOlder(true)
    const el = scrollRef.current
    const prevHeight = el?.scrollHeight ?? 0
    try {
      const page = await listMessagesPage(convRef.current, PAGE, oldestIdRef.current)
      setMessages((prev) => prependUnique(page.messages, prev))
      setHasMore(page.hasMore)
      oldestIdRef.current = page.oldestId ?? oldestIdRef.current
      requestAnimationFrame(() => {
        if (el) el.scrollTop = el.scrollHeight - prevHeight
      })
    } catch {
      toast({ title: t('messages.load_failed'), variant: 'destructive' })
    } finally {
      setLoadingOlder(false)
    }
  }, [hasMore, loadingOlder, t, toast])

  // Sentinelle de défilement infini en HAUT de la liste.
  useEffect(() => {
    const el = topSentinelRef.current
    if (!el || !hasMore || loadingOlder || initialLoading) return
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0]?.isIntersecting) loadOlder()
      },
      { rootMargin: '200px' },
    )
    observer.observe(el)
    return () => observer.disconnect()
  }, [hasMore, loadingOlder, initialLoading, loadOlder])

  // Message temps réel pour cette conversation : on l'ajoute en bas (dédup).
  useEffect(() => {
    if (!liveMessage || liveMessage.conversationId !== conversation.id) return
    const el = scrollRef.current
    const nearBottom = el ? el.scrollHeight - el.scrollTop - el.clientHeight < 120 : true
    setMessages((prev) => (prev.some((m) => m.id === liveMessage.id) ? prev : [...prev, liveMessage]))
    if (nearBottom) requestAnimationFrame(() => scrollToBottom('smooth'))
  }, [liveMessage, conversation.id, scrollToBottom])

  async function handleSend() {
    const text = draft.trim()
    if (!text || sending || !canSend) return
    setSending(true)
    try {
      const msg = await sendMessage(convRef.current, text)
      setMessages((prev) => (prev.some((m) => m.id === msg.id) ? prev : [...prev, msg]))
      onLocalMessage(msg)
      setDraft('')
      requestAnimationFrame(() => scrollToBottom('smooth'))
    } catch {
      toast({ title: t('messages.send_failed'), variant: 'destructive' })
    } finally {
      setSending(false)
    }
  }

  return (
    <div className="flex h-full min-h-0 flex-col">
      {/* En-tête */}
      <ChatHeader conversation={conversation} myId={myId} onBack={onBack} onOpenInfo={onOpenInfo} />

      {/* Historique */}
      <div ref={scrollRef} className="flex-1 overflow-y-auto px-3 py-4 sm:px-4">
        <div ref={topSentinelRef} />
        {loadingOlder && (
          <div className="flex justify-center py-2">
            <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
          </div>
        )}

        {keyMissing && (
          <div className="glass mx-auto my-3 flex max-w-md items-center gap-2 rounded-2xl border px-4 py-3 text-sm text-muted-foreground">
            <Lock className="h-4 w-4 shrink-0" aria-hidden />
            {t('messages.key_missing')}
          </div>
        )}

        {initialLoading ? (
          <div className="flex justify-center py-16">
            <Loader2 className="h-6 w-6 animate-spin text-[#5B6CFF] dark:text-[#9aa6ff]" />
          </div>
        ) : messages.length === 0 && !keyMissing ? (
          <div className="flex flex-col items-center gap-1 py-16 text-center">
            <p className="text-sm font-bold text-foreground">{t('messages.no_messages_title')}</p>
            <p className="text-sm text-muted-foreground">{t('messages.no_messages_desc')}</p>
          </div>
        ) : (
          <ul className="flex flex-col gap-1">
            {messages.map((m, i) => (
              <Fragment key={m.id}>
                {m.id === dividerBeforeId && <NewMessagesDivider label={t('messages.new_messages_divider')} />}
                <MessageBubble
                  message={m}
                  conversation={conversation}
                  // Affiche l'avatar/nom de l'expéditeur si l'auteur change (groupes/communautés).
                  showSender={
                    conversation.type !== 'dm' &&
                    !m.mine &&
                    (i === 0 || messages[i - 1].senderId !== m.senderId)
                  }
                />
              </Fragment>
            ))}
          </ul>
        )}
      </div>

      {/* Composer */}
      <div className="panel border-t px-3 py-2.5 sm:px-4">
        {readOnly ? (
          <p className="py-2 text-center text-sm text-muted-foreground">{t('messages.read_only')}</p>
        ) : (
          <div className="flex items-end gap-2">
            <textarea
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && !e.shiftKey) {
                  e.preventDefault()
                  handleSend()
                }
              }}
              disabled={!canSend || sending}
              rows={1}
              placeholder={t('messages.composer_placeholder')}
              className="glass max-h-32 min-h-[44px] flex-1 resize-none rounded-2xl border px-4 py-2.5 text-sm backdrop-blur placeholder:text-muted-foreground focus:border-[#5B6CFF] focus:outline-none disabled:cursor-not-allowed disabled:opacity-60"
            />
            <Button
              type="button"
              size="icon"
              onClick={handleSend}
              disabled={!canSend || sending || !draft.trim()}
              aria-label={t('messages.send')}
              className="h-11 w-11 shrink-0 rounded-full bg-gradient-to-r from-[#8D3DFF] via-[#5B6CFF] to-[#47D9FF] text-white"
            >
              {sending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Send className="h-4 w-4" />}
            </Button>
          </div>
        )}
      </div>
    </div>
  )
}

/** En-tête du chat : avatar + titre (interlocuteur DM ou nom de groupe). */
function ChatHeader({
  conversation,
  myId,
  onBack,
  onOpenInfo,
}: {
  conversation: Conversation
  myId: string
  onBack: () => void
  onOpenInfo: () => void
}) {
  const { t } = useLanguage()
  const peerId =
    conversation.type === 'dm'
      ? conversation.memberIds.find((id) => id !== myId) ?? null
      : null
  const peer = useResolvedUser(peerId)
  const title = conversationTitle(conversation, peer, t)
  const subtitle =
    conversation.type === 'dm'
      ? peer?.username
        ? `@${peer.username}`
        : ''
      : t(`messages.type_${conversation.type}`)

  return (
    <header className="panel flex items-center gap-3 border-b px-3 py-2.5 sm:px-4">
      <Button
        type="button"
        variant="ghost"
        size="icon"
        onClick={onBack}
        aria-label={t('messages.back')}
        className="shrink-0 rounded-full lg:hidden"
      >
        <ArrowLeft className="h-5 w-5" />
      </Button>

      <ConversationAvatar conversation={conversation} peer={peer} className="h-10 w-10" />

      <div className="flex min-w-0 flex-1 flex-col">
        <span className="truncate text-sm font-bold text-foreground">{title}</span>
        {subtitle && <span className="truncate text-xs text-muted-foreground">{subtitle}</span>}
      </div>

      <Button
        type="button"
        variant="ghost"
        size="icon"
        onClick={onOpenInfo}
        aria-label={t('messages.info')}
        className="shrink-0 rounded-full"
      >
        <Info className="h-5 w-5" />
      </Button>
    </header>
  )
}

/** Bulle de message : alignée à droite (mienne) ou à gauche, avec expéditeur. */
function MessageBubble({
  message,
  conversation,
  showSender,
}: {
  message: ChatMessage
  conversation: Conversation
  showSender: boolean
}) {
  const { t, locale } = useLanguage()
  const sender = useResolvedUser(showSender ? message.senderId : null)
  const time = formatTime(message.createdAt, locale)

  return (
    <li className={cn('flex flex-col', message.mine ? 'items-end' : 'items-start')}>
      {showSender && (
        <div className="mb-0.5 ml-1 flex items-center gap-1.5">
          <Avatar className="h-5 w-5">
            {sender?.avatarUrl && <AvatarImage src={sender.avatarUrl} alt={sender.displayName} />}
            <AvatarFallback className="text-[10px]">
              {(sender?.displayName.charAt(0) || '?').toUpperCase()}
            </AvatarFallback>
          </Avatar>
          <span className="text-xs font-semibold text-muted-foreground">
            {sender?.displayName ?? '…'}
          </span>
        </div>
      )}
      <div
        className={cn(
          'max-w-[78%] rounded-2xl px-3.5 py-2 text-sm shadow-sm',
          message.mine
            ? 'bg-gradient-to-r from-[#8D3DFF] via-[#5B6CFF] to-[#47D9FF] text-white'
            : 'glass border text-foreground',
        )}
      >
        {message.decrypted ? (
          <p className="whitespace-pre-wrap break-words">{message.text}</p>
        ) : (
          <p className="flex items-center gap-1.5 italic opacity-80">
            <Lock className="h-3.5 w-3.5" aria-hidden />
            {t('messages.decrypt_failed')}
          </p>
        )}
      </div>
      <span className="mt-0.5 px-1 text-[11px] text-muted-foreground">{time}</span>
    </li>
  )
}

/** Séparateur « Nouveaux messages » (aux couleurs de marque). */
function NewMessagesDivider({ label }: { label: string }) {
  return (
    <li className="my-2 flex items-center gap-3" aria-label={label}>
      <span className="h-px flex-1 bg-[#8D3DFF]/40" />
      <span className="rounded-full bg-[#8D3DFF]/15 px-3 py-0.5 text-xs font-bold text-[#8D3DFF] dark:text-[#c9a3ff]">
        {label}
      </span>
      <span className="h-px flex-1 bg-[#8D3DFF]/40" />
    </li>
  )
}

function formatTime(iso: string, locale: string): string {
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return ''
  const intl = locale === 'en' ? 'en-US' : 'fr-FR'
  return new Intl.DateTimeFormat(intl, { hour: '2-digit', minute: '2-digit' }).format(date)
}
