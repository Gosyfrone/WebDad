'use client'

import {
  Fragment,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react'
import {
  ArrowLeft,
  Check,
  CheckCheck,
  Flag,
  ImageOff,
  Info,
  Loader2,
  Lock,
  MoreHorizontal,
  Paperclip,
  Pencil,
  Send,
  Trash2,
  X,
} from 'lucide-react'

import { cn, initialOf } from '@/lib/utils'
import { ReportDialog } from '@/components/moderation/report-dialog'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  computeDivider,
  decryptAttachment,
  deleteMessage,
  editMessage,
  listMessagesPage,
  planReceipts,
  sendMessage,
  sendTyping,
  type ChatAttachment,
  type ChatMessage,
  type Conversation,
  type ReceiptMark,
} from '@/lib/messages'
import { extractMentionHandles, type MentionCandidate } from '@/lib/mentions'
import { makeMemberFirstSearch } from '@/lib/mention-search'
import { useMention } from '@/lib/use-mention'
import { exceedsMediaLimit, MAX_MEDIA_MB } from '@/lib/media'
import { resolveUsers } from '@/lib/user-cache'
import { useResolvedUser } from '@/lib/use-resolved-user'
import { useToast } from '@/hooks/use-toast'
import { useLanguage } from '@/components/language-provider'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import { MentionAutocomplete } from '@/components/mention/mention-autocomplete'
import { MentionMessageText } from '@/components/mention/mention-text'
import { SharedLinkPreview } from '@/components/share/shared-link-preview'
import { extractSharedRef } from '@/lib/share'
import { MediaLightbox } from '@/components/ui/media-lightbox'
import {
  ConversationAvatar,
  conversationTitle,
} from '@/components/messages/conversation-meta'
import { ActivityStatus } from '@/components/profil/activity-status'
import { ActivityPresenceDot } from '@/components/profil/activity-presence-dot'

const PAGE = 30

interface ChatPaneProps {
  conversation: Conversation
  myId: string
  /** Dernier message reçu en temps réel pour CETTE conversation (ou null). */
  liveMessage: ChatMessage | null
  /** Message modifié reçu en temps réel pour CETTE conversation (ou null). */
  liveUpdatedMessage: ChatMessage | null
  /** Ids des membres « en train d'écrire » dans cette conversation (hors moi). */
  typingUserIds: string[]
  /** Curseur de lecture (`lastReadAt` ISO) capturé à l'ouverture (ancre
   *  « Nouveaux messages ») ; null si jamais lu. */
  dividerAnchor: string | null
  /** Retour à la liste (mobile). */
  onBack: () => void
  /** Ouvre le panneau d'infos / gestion (membres, renommer, quitter…). */
  onOpenInfo: () => void
  /** Notifie le parent d'un message envoyé ou modifié d'ici. */
  onLocalMessage: (msg: ChatMessage, edited?: boolean) => void
}

/** Fusionne des messages plus anciens en tête, en dédupliquant par id. */
function prependUnique(
  older: ChatMessage[],
  current: ChatMessage[],
): ChatMessage[] {
  const seen = new Set(current.map((m) => m.id))
  return [...older.filter((m) => !seen.has(m.id)), ...current]
}

/**
 * Volet de conversation : historique chiffré (déchiffré localement), défilement
 * infini vers le haut, envoi, messages live par WebSocket.
 *
 * Cas particuliers : clé de contenu absente sur cet appareil (`contentKey === null`,
 * identité créée ailleurs) → lecture/écriture désactivées ; communauté en lecture
 * seule (`viewer`) → composer désactivé.
 */
export function ChatPane({
  conversation,
  myId,
  liveMessage,
  liveUpdatedMessage,
  typingUserIds,
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
  const [attachments, setAttachments] = useState<File[]>([])
  const [sending, setSending] = useState(false)
  const [editingMessage, setEditingMessage] = useState<ChatMessage | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)
  // Id du message devant lequel afficher « Nouveaux messages » (gelé à l'ouverture).
  const [dividerBeforeId, setDividerBeforeId] = useState<string | null>(null)
  // Cible de portail des menus « … » des messages. On les rend DANS l'overlay
  // messagerie (`FeedOverlay`, `z-40`) plutôt que sur `document.body`, pour que
  // le composer sticky (`z-20`, lui aussi dans l'overlay) puisse passer DEVANT
  // eux — impossible autrement, un portail racine étant toujours au-dessus de
  // tout l'overlay ou caché derrière.
  const [menuPortal, setMenuPortal] = useState<HTMLDivElement | null>(null)

  const oldestIdRef = useRef<string | null>(null)
  const scrollRef = useRef<HTMLDivElement | null>(null)
  const topSentinelRef = useRef<HTMLDivElement | null>(null)
  const composerRef = useRef<HTMLTextAreaElement | null>(null)
  // Référence à la conversation courante (clé à jour pour chiffrer/déchiffrer).
  const convRef = useRef(conversation)
  convRef.current = conversation

  // Membres résolus (username + décoratif) : autocomplétion des mentions,
  // rendu des @handle (membre → profil ; non-membre → carte d'aperçu) et calcul
  // des ids mentionnés à l'envoi (notifications).
  const memberKey = conversation.memberIds.join(',')
  const [members, setMembers] = useState<MentionCandidate[]>([])
  useEffect(() => {
    let cancelled = false
    resolveUsers(conversation.memberIds)
      .then((list) => {
        if (!cancelled) setMembers(list)
      })
      .catch(() => {})
    return () => {
      cancelled = true
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [memberKey])

  const memberUsernames = useMemo(
    () => new Set(members.map((m) => m.username.toLowerCase()).filter(Boolean)),
    [members],
  )
  const memberByUsername = useMemo(() => {
    const map = new Map<string, string>()
    for (const m of members)
      if (m.username) map.set(m.username.toLowerCase(), m.id)
    return map
  }, [members])
  const mentionSearch = useMemo(
    () => makeMemberFirstSearch(members.filter((m) => m.id !== myId)),
    [members, myId],
  )
  const mention = useMention({
    inputRef: composerRef,
    onChange: setDraft,
    search: mentionSearch,
  })

  // Accusés de réception (remis/ouvert) à placer sous MES messages, recalculés
  // quand les messages ou les curseurs des autres membres changent (temps réel).
  const receiptMarks = useMemo(
    () => planReceipts(messages, conversation),
    [messages, conversation],
  )

  const keyMissing = conversation.contentKey === null
  const readOnly =
    conversation.type === 'community' && conversation.myRole === 'viewer'
  const canSend = !keyMissing && !readOnly
  const isEditing = editingMessage !== null
  // Modérateur : owner/admin d'un groupe ou d'une communauté peut supprimer les
  // messages des autres (contrôlé aussi côté serveur).
  const isModerator =
    (conversation.type === 'group' || conversation.type === 'community') &&
    (conversation.myRole === 'owner' || conversation.myRole === 'admin')

  // Ping « en train d'écrire » throttlé (au plus 1 / 3 s pendant la frappe).
  const lastTypingRef = useRef(0)
  const pingTyping = useCallback(() => {
    const now = Date.now()
    if (now - lastTypingRef.current < 3000) return
    lastTypingRef.current = now
    void sendTyping(convRef.current.id)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

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
    setEditingMessage(null)
    setDraft('')
    setAttachments([])
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
        if (!cancelled)
          toast({ title: t('messages.load_failed'), variant: 'destructive' })
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
      const page = await listMessagesPage(
        convRef.current,
        PAGE,
        oldestIdRef.current,
      )
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
    const nearBottom = el
      ? el.scrollHeight - el.scrollTop - el.clientHeight < 120
      : true
    setMessages((prev) =>
      prev.some((m) => m.id === liveMessage.id) ? prev : [...prev, liveMessage],
    )
    if (nearBottom) requestAnimationFrame(() => scrollToBottom('smooth'))
  }, [liveMessage, conversation.id, scrollToBottom])

  // Message modifié en temps réel : on remplace la version locale.
  useEffect(() => {
    if (
      !liveUpdatedMessage ||
      liveUpdatedMessage.conversationId !== conversation.id
    )
      return
    setMessages((prev) =>
      prev.map((m) =>
        m.id === liveUpdatedMessage.id ? liveUpdatedMessage : m,
      ),
    )
  }, [liveUpdatedMessage, conversation.id])

  async function handleSubmit() {
    const text = draft.trim()
    if (sending || !canSend) return
    if (isEditing && (!editingMessage || !text)) return
    if (!isEditing && !text && attachments.length === 0) return
    setSending(true)
    try {
      // Résout les @handle mentionnés en ids de MEMBRES (hors soi) → notifications.
      // Le serveur ne reçoit que des ids (jamais le texte) : E2EE intact.
      const mentionedIds = [
        ...new Set(
          extractMentionHandles(text)
            .map((h) => memberByUsername.get(h))
            .filter((id): id is string => Boolean(id) && id !== myId),
        ),
      ]
      const msg = editingMessage
        ? await editMessage(convRef.current, editingMessage, text, mentionedIds)
        : await sendMessage(convRef.current, text, attachments, mentionedIds)
      setMessages((prev) =>
        editingMessage
          ? prev.map((m) => (m.id === msg.id ? msg : m))
          : prev.some((m) => m.id === msg.id)
            ? prev
            : [...prev, msg],
      )
      onLocalMessage(msg, Boolean(editingMessage))
      setDraft('')
      setAttachments([])
      setEditingMessage(null)
      requestAnimationFrame(() => scrollToBottom('smooth'))
    } catch {
      toast({
        title: t(isEditing ? 'messages.edit_failed' : 'messages.send_failed'),
        variant: 'destructive',
      })
    } finally {
      setSending(false)
    }
  }

  function startEdit(message: ChatMessage) {
    if (!message.mine || !message.decrypted || sending || !canSend) return
    setEditingMessage(message)
    setDraft(message.text)
    setAttachments([])
    requestAnimationFrame(() => composerRef.current?.focus())
  }

  function cancelEdit() {
    setEditingMessage(null)
    setDraft('')
    setAttachments([])
  }

  async function handleDelete(message: ChatMessage) {
    if (message.deletedAt || sending) return
    if (!window.confirm(t('messages.delete_confirm'))) return
    try {
      const deleted = await deleteMessage(convRef.current, message.id)
      setMessages((prev) =>
        prev.map((m) => (m.id === deleted.id ? deleted : m)),
      )
      if (editingMessage?.id === deleted.id) cancelEdit()
      onLocalMessage(deleted, true) // edited=true → met à jour l'aperçu si concerné
    } catch {
      toast({ title: t('messages.delete_failed'), variant: 'destructive' })
    }
  }

  function handlePickFiles(e: React.ChangeEvent<HTMLInputElement>) {
    const picked = Array.from(e.target.files ?? [])
    e.target.value = ''
    if (picked.length === 0) return

    // Garde UX : pièces jointes trop lourdes écartées avant chiffrement/upload
    // (le cap est aussi appliqué côté serveur). Les admins ne sont pas plafonnés.
    const allowed = picked.filter((f) => !exceedsMediaLimit(f.size))
    if (allowed.length < picked.length) {
      toast({ title: t('media.too_large', { max: MAX_MEDIA_MB }), variant: 'brand' })
    }
    if (allowed.length > 0) setAttachments((prev) => [...prev, ...allowed])
  }

  return (
    <div className="flex h-full min-h-0 flex-col">
      {/* Cible de portail des menus « … » des messages (cf. `menuPortal`). */}
      <div ref={setMenuPortal} />
      {/* En-tête */}
      <ChatHeader
        conversation={conversation}
        myId={myId}
        onBack={onBack}
        onOpenInfo={onOpenInfo}
      />

      {/* Historique */}
      <div ref={scrollRef} className="flex-1 overflow-y-auto px-3 pb-6 pt-4 sm:px-4">
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
            <p className="text-sm font-bold text-foreground">
              {t('messages.no_messages_title')}
            </p>
            <p className="text-sm text-muted-foreground">
              {t('messages.no_messages_desc')}
            </p>
          </div>
        ) : (
          <ul className="flex flex-col gap-1">
            {messages.map((m, i) => (
              <Fragment key={m.id}>
                {m.id === dividerBeforeId && (
                  <NewMessagesDivider
                    label={t('messages.new_messages_divider')}
                  />
                )}
                <MessageBubble
                  message={m}
                  conversation={conversation}
                  memberUsernames={memberUsernames}
                  menuContainer={menuPortal}
                  receipt={m.mine ? receiptMarks.get(m.id) : undefined}
                  canDelete={!m.deletedAt && canSend && (m.mine || isModerator)}
                  // Affiche l'avatar/nom de l'expéditeur si l'auteur change (groupes/communautés).
                  showSender={
                    conversation.type !== 'dm' &&
                    !m.mine &&
                    (i === 0 || messages[i - 1].senderId !== m.senderId)
                  }
                  onEditStart={startEdit}
                  onDelete={handleDelete}
                />
              </Fragment>
            ))}
          </ul>
        )}
      </div>

      {/* Indicateur « en train d'écrire » */}
      <TypingIndicator conversation={conversation} userIds={typingUserIds} />

      {/* Composer */}
      <div className="panel sticky bottom-0 z-20 shrink-0 border-t px-3 pb-2.5 pt-2.5 shadow-[0_-14px_34px_rgba(91,108,255,0.10)] backdrop-blur-xl sm:px-4">
        {readOnly ? (
          <p className="py-2 text-center text-sm text-muted-foreground">
            {t('messages.read_only')}
          </p>
        ) : (
          <>
            {isEditing && (
              <div className="mb-2 flex items-center justify-between gap-2 rounded-xl border border-[#5B6CFF]/20 bg-[#5B6CFF]/10 px-3 py-2 text-xs text-foreground">
                <span className="truncate font-semibold">
                  {t('messages.editing')}
                </span>
                <button
                  type="button"
                  onClick={cancelEdit}
                  className="rounded-full p-1 text-muted-foreground transition hover:bg-background/60 hover:text-foreground"
                  aria-label={t('messages.cancel_edit')}
                >
                  <X className="h-3.5 w-3.5" />
                </button>
              </div>
            )}
            {!isEditing && attachments.length > 0 && (
              <PendingAttachments
                files={attachments}
                onRemove={(i) =>
                  setAttachments((prev) => prev.filter((_, idx) => idx !== i))
                }
                removeLabel={t('composer.media_remove')}
              />
            )}
            <div className="flex items-end gap-2">
              <input
                ref={fileInputRef}
                type="file"
                accept="image/*,video/*"
                multiple
                onChange={handlePickFiles}
                className="sr-only"
              />
              <Button
                type="button"
                variant="ghost"
                size="icon"
                onClick={() => fileInputRef.current?.click()}
                disabled={!canSend || sending || isEditing}
                aria-label={t('messages.add_attachment')}
                className="h-11 w-11 shrink-0 rounded-full text-[#5B6CFF]"
              >
                <Paperclip className="h-5 w-5" />
              </Button>
              <div className="relative flex-1">
                <textarea
                  ref={composerRef}
                  value={draft}
                  onChange={(e) => {
                    setDraft(e.target.value)
                    mention.sync()
                    if (canSend) pingTyping()
                  }}
                  onKeyUp={mention.sync}
                  onClick={mention.sync}
                  onKeyDown={(e) => {
                    mention.onKeyDown(e)
                    if (e.defaultPrevented) return
                    if (e.key === 'Enter' && !e.shiftKey) {
                      e.preventDefault()
                      handleSubmit()
                    }
                  }}
                  disabled={!canSend || sending}
                  rows={1}
                  placeholder={t('messages.composer_placeholder')}
                  className="glass max-h-32 min-h-[44px] w-full resize-none rounded-2xl border px-4 py-2.5 text-sm backdrop-blur placeholder:text-muted-foreground focus:border-[#5B6CFF] focus:outline-none disabled:cursor-not-allowed disabled:opacity-60"
                />
                <MentionAutocomplete controller={mention} placement="top" />
              </div>
              <Button
                type="button"
                size="icon"
                onClick={handleSubmit}
                disabled={
                  !canSend ||
                  sending ||
                  (isEditing
                    ? !draft.trim()
                    : !draft.trim() && attachments.length === 0)
                }
                aria-label={t(
                  isEditing ? 'messages.save_edit' : 'messages.send',
                )}
                className="h-11 w-11 shrink-0 rounded-full bg-gradient-to-r from-[var(--brand-from)] via-[var(--brand-via)] to-[var(--brand-to)] text-white"
              >
                {sending ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <Send className="h-4 w-4" />
                )}
              </Button>
            </div>
          </>
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
      ? (conversation.memberIds.find((id) => id !== myId) ?? null)
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

      <ConversationAvatar
        conversation={conversation}
        peer={peer}
        className="h-10 w-10"
      />

      <div className="flex min-w-0 flex-1 flex-col">
        <span className="truncate text-sm font-bold text-foreground">
          {title}
        </span>
        {subtitle && (
          <span className="flex min-w-0 items-center gap-2 text-xs text-muted-foreground">
            <span className="truncate">{subtitle}</span>
            {peerId && <ActivityStatus userId={peerId} className="shrink-0" />}
          </span>
        )}
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
  memberUsernames,
  receipt,
  canDelete,
  showSender,
  onEditStart,
  onDelete,
  menuContainer,
}: {
  message: ChatMessage
  conversation: Conversation
  memberUsernames: Set<string>
  /** Accusé de réception à afficher sous ce message (mes messages uniquement). */
  receipt?: ReceiptMark
  /** L'utilisateur courant peut-il supprimer ce message (auteur / modérateur) ? */
  canDelete: boolean
  showSender: boolean
  onEditStart: (message: ChatMessage) => void
  onDelete: (message: ChatMessage) => void
  /** Conteneur de portail du menu « … » (cf. `menuPortal` dans ChatPane). */
  menuContainer: HTMLElement | null
}) {
  const { t, locale } = useLanguage()
  const sender = useResolvedUser(showSender ? message.senderId : null)
  const time = formatTime(message.createdAt, locale)
  const isDeleted = Boolean(message.deletedAt)
  const canEdit = message.mine && message.decrypted && !isDeleted
  const [showOriginal, setShowOriginal] = useState(false)
  const hasOriginal =
    !isDeleted && message.decrypted && Boolean(message.originalText)
  // Signalement d'un message (hors message supprimé). Affiché aussi sur ses
  // propres messages pour que l'action soit toujours visible ; sécurité côté
  // back. Type d'entité selon la conversation : groupe/communauté → message de
  // groupe, sinon message privé.
  const canReport = !isDeleted
  const [reportOpen, setReportOpen] = useState(false)
  const reportEntityType =
    conversation.type === 'group' || conversation.type === 'community' ? 'group_message' : 'message'

  // Message supprimé « pour tout le monde » (tombstone) : rendu sobre, sans média,
  // sans actions, sans accusé.
  if (isDeleted) {
    return (
      <li
        className={cn(
          'flex flex-col',
          message.mine ? 'items-end' : 'items-start',
        )}
      >
        <div className="mt-1 max-w-[78%] rounded-2xl border border-dashed border-border bg-background/40 px-3.5 py-2 text-sm italic text-muted-foreground">
          <span className="flex items-center gap-1.5">
            <Trash2 className="h-3.5 w-3.5" aria-hidden />
            {message.deletedByModeration ? t('messages.deleted_by_moderation') : t('messages.deleted')}
          </span>
        </div>
        <span className="mt-0.5 px-1 text-[11px] text-muted-foreground">
          {time}
        </span>
      </li>
    )
  }

  return (
    <li
      className={cn(
        'flex flex-col',
        message.mine ? 'items-end' : 'items-start',
      )}
    >
      {showSender && (
        <div className="mb-0.5 ml-1 flex items-center gap-1.5">
          <Avatar className="h-5 w-5">
            {sender?.avatarUrl && (
              <AvatarImage src={sender.avatarUrl} alt={sender.displayName} />
            )}
            <AvatarFallback className="text-[10px]">
              {initialOf(sender?.displayName, sender?.username)}
            </AvatarFallback>
            <ActivityPresenceDot userId={sender?.id} className="h-2 w-2 border" />
          </Avatar>
          <span className="text-xs font-semibold text-muted-foreground">
            {sender?.displayName ?? '…'}
          </span>
        </div>
      )}
      {/* Pièces jointes (déchiffrées à la volée), hors bulle texte. Le menu d'actions
          « … » apparaît ici quand le message n'a PAS de texte (média seul) — sinon il
          est rendu dans la rangée texte plus bas. */}
      {message.decrypted && message.media.length > 0 && (
        <div
          className={cn(
            'group flex max-w-[78%] items-end gap-1.5',
            message.mine && 'flex-row-reverse',
          )}
        >
          <div
            className={cn(
              'flex flex-col gap-1.5',
              message.mine ? 'items-end' : 'items-start',
            )}
          >
            {message.media.map((att) => (
              <AttachmentView
                key={att.id}
                contentKey={conversation.contentKey}
                att={att}
              />
            ))}
          </div>
          {!message.text && (
            <MessageActionsMenu
              canEdit={canEdit}
              canDelete={canDelete}
              canReport={canReport}
              onEdit={() => onEditStart(message)}
              onDelete={() => onDelete(message)}
              onReport={() => setReportOpen(true)}
              menuContainer={menuContainer}
              className="mb-1"
            />
          )}
        </div>
      )}

      {/* Bulle texte : seulement s'il y a du texte, ou si le déchiffrement a échoué. */}
      {(!message.decrypted || message.text) && (
        <div
          className={cn(
            'group mt-1 flex max-w-[78%] items-end gap-1.5',
            message.mine && 'flex-row-reverse',
          )}
        >
          <div className="flex min-w-0 flex-col gap-1">
            {hasOriginal && (
              <button
                type="button"
                onClick={() => setShowOriginal((open) => !open)}
                className={cn(
                  'w-fit rounded-full px-2 py-0.5 text-[11px] font-semibold text-muted-foreground transition hover:bg-background/70 hover:text-foreground',
                  message.mine ? 'self-end' : 'self-start',
                )}
                aria-expanded={showOriginal}
              >
                {t(showOriginal ? 'messages.hide_original' : 'messages.edited')}
              </button>
            )}
            {hasOriginal && showOriginal && (
              <div className="max-w-full rounded-2xl border border-dashed border-border bg-background/55 px-3 py-1.5 text-xs italic text-muted-foreground opacity-60 shadow-none">
                <MentionMessageText
                  text={message.originalText}
                  memberUsernames={memberUsernames}
                  onAccent={false}
                  className="block whitespace-pre-wrap break-words"
                />
              </div>
            )}
            <div
              className={cn(
                'max-w-full rounded-2xl px-3.5 py-2 text-sm shadow-sm',
                message.mine
                  ? 'bg-gradient-to-r from-[var(--brand-from)] via-[var(--brand-via)] to-[var(--brand-to)] text-white'
                  : 'glass border text-foreground',
              )}
            >
              {message.decrypted ? (
                <>
                  <MentionMessageText
                    text={message.text}
                    memberUsernames={memberUsernames}
                    onAccent={message.mine}
                    className="block whitespace-pre-wrap break-words"
                  />
                </>
              ) : (
                <p className="flex items-center gap-1.5 italic opacity-80">
                  <Lock className="h-3.5 w-3.5" aria-hidden />
                  {t('messages.decrypt_failed')}
                </p>
              )}
            </div>
            {message.decrypted && (() => {
              const ref = extractSharedRef(message.text)
              return ref ? <SharedLinkPreview target={ref} /> : null
            })()}
          </div>
          <MessageActionsMenu
            canEdit={canEdit}
            canDelete={canDelete}
            canReport={canReport}
            onEdit={() => onEditStart(message)}
            onDelete={() => onDelete(message)}
            onReport={() => setReportOpen(true)}
            menuContainer={menuContainer}
            className="mb-1"
          />
        </div>
      )}
      <div className="mt-0.5 flex items-center gap-1 px-1 text-[11px] text-muted-foreground">
        <span>{time}</span>
        {receipt && <ReceiptIndicator receipt={receipt} />}
      </div>

      {canReport && (
        <ReportDialog
          open={reportOpen}
          onOpenChange={setReportOpen}
          entityType={reportEntityType}
          entityId={message.id}
          entityOwnerId={message.senderId}
          // E2EE : on transmet la copie EN CLAIR que CE destinataire a déchiffrée,
          // pour que la modération puisse juger (le serveur, lui, reste aveugle).
          disclosedContent={message.decrypted ? message.text || '' : ''}
        />
      )}
    </li>
  )
}

/**
 * Menu d'actions « … » d'un message (éditer / supprimer / signaler). Vrai `<button>`
 * Radix → tap fiable sur iOS. Rien si aucune action permise.
 */
function MessageActionsMenu({
  canEdit,
  canDelete,
  canReport,
  onEdit,
  onDelete,
  onReport,
  menuContainer,
  className,
}: {
  canEdit: boolean
  canDelete: boolean
  canReport: boolean
  onEdit: () => void
  onDelete: () => void
  onReport: () => void
  /** Conteneur de portail : l'overlay messagerie, pour passer SOUS le composer. */
  menuContainer: HTMLElement | null
  className?: string
}) {
  const { t } = useLanguage()
  if (!canEdit && !canDelete && !canReport) return null
  return (
    <DropdownMenu modal={false}>
      <DropdownMenuTrigger
        aria-label={t('messages.actions')}
        className={cn(
          'shrink-0 rounded-full p-1.5 text-muted-foreground transition hover:bg-background/70 hover:text-foreground focus:outline-none',
          'opacity-100 [@media(hover:hover)]:opacity-0 [@media(hover:hover)]:group-hover:opacity-100 [@media(hover:hover)]:group-focus-within:opacity-100 data-[state=open]:opacity-100',
          className,
        )}
      >
        <MoreHorizontal className="h-4 w-4" />
      </DropdownMenuTrigger>
      <DropdownMenuContent
        align="end"
        // Porté DANS l'overlay messagerie (`z-40`) et placé à `z-10` : au-dessus
        // des bulles (z-auto) mais SOUS le composer sticky (`z-20`), qui passe
        // donc devant. `collisionPadding` bas pour s'ouvrir vers le haut près du
        // composer plutôt que d'être masqué par lui.
        container={menuContainer}
        collisionPadding={{ bottom: 88 }}
        className="z-10"
      >
        {canEdit && (
          <DropdownMenuItem onClick={onEdit} className="cursor-pointer">
            <Pencil className="mr-2 h-4 w-4" />
            {t('messages.edit')}
          </DropdownMenuItem>
        )}
        {canDelete && (
          <DropdownMenuItem
            onClick={onDelete}
            className="cursor-pointer text-red-500 focus:text-red-500"
          >
            <Trash2 className="mr-2 h-4 w-4" />
            {t('messages.delete')}
          </DropdownMenuItem>
        )}
        {/* Signalement placé SOUS l'option de suppression (exigence fonctionnelle). */}
        {canReport && (
          <DropdownMenuItem onClick={onReport} className="cursor-pointer">
            <Flag className="mr-2 h-4 w-4" />
            {t('report.message_action')}
          </DropdownMenuItem>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

/**
 * Indicateur d'accusé de réception sous l'un de MES messages :
 *   - remis (pas encore lu) → 1 coche grise ;
 *   - ouvert (DM / groupe lu par tous) → 2 coches couleur Breezy ;
 *   - groupe lu par une partie → 1 coche couleur + nombre de lecteurs.
 *
 * Cliquable (tap PC + mobile) : un clic révèle le libellé « Remis » / « Ouvert »
 * à droite de la coche, un second le masque.
 */
function ReceiptIndicator({ receipt }: { receipt: ReceiptMark }) {
  const { t } = useLanguage()
  const [showLabel, setShowLabel] = useState(false)
  const brand = 'text-[#5B6CFF] dark:text-[#9aa6ff]'
  const isRead = receipt.kind !== 'delivered'
  const label = t(
    isRead ? 'messages.receipt_read' : 'messages.receipt_delivered',
  )

  return (
    <button
      type="button"
      onClick={(e) => {
        e.stopPropagation()
        setShowLabel((v) => !v)
      }}
      aria-label={label}
      aria-pressed={showLabel}
      title={label}
      className="flex items-center gap-0.5 rounded-full transition hover:opacity-80"
    >
      {receipt.kind === 'delivered' && <Check className="h-3.5 w-3.5" />}
      {receipt.kind === 'read-count' && (
        <span className={cn('flex items-center gap-0.5 font-semibold', brand)}>
          <Check className="h-3.5 w-3.5" />
          {receipt.count}
        </span>
      )}
      {(receipt.kind === 'read' || receipt.kind === 'read-all') && (
        <CheckCheck className={cn('h-3.5 w-3.5', brand)} />
      )}
      {showLabel && (
        <span
          className={cn(
            'text-[11px] font-medium',
            isRead ? brand : 'text-muted-foreground',
          )}
        >
          {label}
        </span>
      )}
    </button>
  )
}

/**
 * Indicateur « en train d'écrire » sous l'historique : DM → « écrit… » ; groupe →
 * « X écrit… » (ou « Plusieurs personnes écrivent… » à plusieurs). Rien si
 * personne ne tape. Éphémère : le parent gère l'expiration des signaux.
 */
function TypingIndicator({
  conversation,
  userIds,
}: {
  conversation: Conversation
  userIds: string[]
}) {
  const { t } = useLanguage()
  // Hook appelé inconditionnellement (règle des hooks) ; on ne résout le nom que
  // pour les groupes/communautés (DM = interlocuteur implicite).
  const first = useResolvedUser(
    conversation.type !== 'dm' ? (userIds[0] ?? null) : null,
  )
  if (userIds.length === 0) return null

  let label: string
  if (conversation.type === 'dm') label = t('messages.typing')
  else if (userIds.length === 1)
    label = t('messages.typing_user', { name: first?.displayName ?? '…' })
  else label = t('messages.typing_several')

  return (
    <div className="px-4 pb-1 pt-0.5 text-[12px] italic text-muted-foreground">
      <span className="inline-flex items-center gap-1.5">
        <span className="inline-flex gap-0.5" aria-hidden>
          <span className="h-1.5 w-1.5 animate-bounce rounded-full bg-current [animation-delay:-0.2s]" />
          <span className="h-1.5 w-1.5 animate-bounce rounded-full bg-current [animation-delay:-0.1s]" />
          <span className="h-1.5 w-1.5 animate-bounce rounded-full bg-current" />
        </span>
        {label}
      </span>
    </div>
  )
}

/**
 * Pièce jointe reçue : télécharge le blob CHIFFRÉ et le déchiffre localement
 * (avec la clé de contenu) en un `objectURL`, révoqué au démontage. Affiche un
 * spinner pendant, une icône si la clé manque / le déchiffrement échoue.
 */
function AttachmentView({
  contentKey,
  att,
}: {
  contentKey: Uint8Array | null
  att: ChatAttachment
}) {
  const { t } = useLanguage()
  const [url, setUrl] = useState<string | null>(null)
  const [failed, setFailed] = useState(false)
  const [lightbox, setLightbox] = useState(false)

  useEffect(() => {
    let revoked = false
    let objectUrl = ''
    if (!contentKey) {
      setFailed(true)
      return
    }
    setFailed(false)
    setUrl(null)
    decryptAttachment(contentKey, att)
      .then((blob) => {
        if (revoked) return
        objectUrl = URL.createObjectURL(blob)
        setUrl(objectUrl)
      })
      .catch(() => {
        if (!revoked) setFailed(true)
      })
    return () => {
      revoked = true
      if (objectUrl) URL.revokeObjectURL(objectUrl)
    }
  }, [contentKey, att])

  if (failed) {
    return (
      <div className="glass flex items-center gap-2 rounded-2xl border px-3 py-2 text-xs italic text-muted-foreground">
        <ImageOff className="h-4 w-4 shrink-0" aria-hidden />
        {t('messages.attachment_failed')}
      </div>
    )
  }
  if (!url) {
    return (
      <div className="glass flex h-40 w-40 items-center justify-center rounded-2xl border">
        <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
      </div>
    )
  }
  return (
    <div className="overflow-hidden rounded-2xl border border-border">
      {att.type === 'video' ? (
        <video src={url} controls playsInline className="max-h-80 max-w-full" />
      ) : (
        <>
          <button
            type="button"
            onClick={() => setLightbox(true)}
            className="block"
            aria-label={t('messages.open_image')}
          >
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img
              src={url}
              alt={att.name}
              className="max-h-80 max-w-full cursor-zoom-in object-contain"
            />
          </button>
          <MediaLightbox
            open={lightbox}
            onClose={() => setLightbox(false)}
            src={url}
            type="image"
            name={att.name}
            downloadable
          />
        </>
      )}
    </div>
  )
}

/** Aperçus des pièces jointes en attente d'envoi (avant chiffrement/upload). */
function PendingAttachments({
  files,
  onRemove,
  removeLabel,
}: {
  files: File[]
  onRemove: (index: number) => void
  removeLabel: string
}) {
  return (
    <div className="mb-2 flex flex-wrap gap-2">
      {files.map((file, i) => (
        <PendingThumb
          key={`${file.name}-${i}`}
          file={file}
          onRemove={() => onRemove(i)}
          removeLabel={removeLabel}
        />
      ))}
    </div>
  )
}

function PendingThumb({
  file,
  onRemove,
  removeLabel,
}: {
  file: File
  onRemove: () => void
  removeLabel: string
}) {
  const [url, setUrl] = useState('')
  useEffect(() => {
    const objectUrl = URL.createObjectURL(file)
    setUrl(objectUrl)
    return () => URL.revokeObjectURL(objectUrl)
  }, [file])

  return (
    <div className="group relative h-20 w-20 overflow-hidden rounded-xl border border-border bg-background/45">
      {file.type.startsWith('video/') ? (
        <video
          src={url}
          className="h-full w-full object-cover"
          muted
          playsInline
        />
      ) : (
        // eslint-disable-next-line @next/next/no-img-element
        <img src={url} alt="" className="h-full w-full object-cover" />
      )}
      <button
        type="button"
        aria-label={removeLabel}
        onClick={onRemove}
        className="absolute right-1 top-1 rounded-full bg-black/60 p-0.5 text-white transition hover:bg-black/80"
      >
        <X className="h-3.5 w-3.5" />
      </button>
    </div>
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
  return new Intl.DateTimeFormat(intl, {
    hour: '2-digit',
    minute: '2-digit',
  }).format(date)
}
