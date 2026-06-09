'use client'

import {
  Bell,
  BellOff,
  Compass,
  Loader2,
  MessageSquarePlus,
  MoreHorizontal,
  Pin,
  PinOff,
  Plus,
  Trash2,
  Users,
} from 'lucide-react'

import { cn } from '@/lib/utils'
import type { Conversation } from '@/lib/messages'
import { useResolvedUser } from '@/lib/use-resolved-user'
import { useLanguage } from '@/components/language-provider'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { ConversationAvatar, conversationTitle } from '@/components/messages/conversation-meta'

/** Aperçu du dernier message d'une conversation (sous-titre de la liste). */
export interface ConversationPreview {
  messageId: string
  text: string
  mine: boolean
  decrypted: boolean
}

interface ConversationListProps {
  conversations: Conversation[]
  selectedId: string | null
  myId: string
  loading: boolean
  /** Aperçu du dernier message par conversation (id → preview). */
  previews: Record<string, ConversationPreview>
  /** Conversations comportant des messages non lus (id → true). */
  unread: Record<string, boolean>
  onSelect: (id: string) => void
  onTogglePin: (conv: Conversation) => void
  onToggleMute: (conv: Conversation) => void
  onDelete: (conv: Conversation) => void
  onNewDM: () => void
  onNewGroup: () => void
  onNewCommunity: () => void
  onDiscover: () => void
}

/**
 * Volet gauche : en-tête + menu de création (DM / groupe / communauté /
 * découvrir) + liste des conversations triée par activité. Chaque entrée montre
 * l'aperçu du dernier message et une pastille mauve si non lu.
 */
export function ConversationList({
  conversations,
  selectedId,
  myId,
  loading,
  previews,
  unread,
  onSelect,
  onTogglePin,
  onToggleMute,
  onDelete,
  onNewDM,
  onNewGroup,
  onNewCommunity,
  onDiscover,
}: ConversationListProps) {
  const { t } = useLanguage()

  return (
    <div className="flex h-full min-h-0 flex-col">
      {/* En-tête */}
      <div className="panel flex items-center justify-between border-b px-4 py-3">
        <h1 className="brand-text text-xl font-bold">{t('messages.title')}</h1>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              size="icon"
              aria-label={t('messages.new')}
              className="h-9 w-9 rounded-full bg-gradient-to-r from-[#8D3DFF] via-[#5B6CFF] to-[#47D9FF] text-white"
            >
              <Plus className="h-5 w-5" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-56">
            <DropdownMenuItem onClick={onNewDM}>
              <MessageSquarePlus className="mr-2 h-4 w-4" />
              {t('messages.new_dm')}
            </DropdownMenuItem>
            <DropdownMenuItem onClick={onNewGroup}>
              <Users className="mr-2 h-4 w-4" />
              {t('messages.new_group')}
            </DropdownMenuItem>
            <DropdownMenuItem onClick={onNewCommunity}>
              <Plus className="mr-2 h-4 w-4" />
              {t('messages.new_community')}
            </DropdownMenuItem>
            <DropdownMenuItem onClick={onDiscover}>
              <Compass className="mr-2 h-4 w-4" />
              {t('messages.discover')}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>

      {/* Liste */}
      <div className="flex-1 overflow-y-auto">
        {loading ? (
          <div className="flex justify-center py-16">
            <Loader2 className="h-6 w-6 animate-spin text-[#5B6CFF] dark:text-[#9aa6ff]" />
          </div>
        ) : conversations.length === 0 ? (
          <div className="flex flex-col items-center gap-1 px-8 py-16 text-center">
            <p className="text-sm font-bold text-foreground">{t('messages.heading')}</p>
            <p className="text-sm text-muted-foreground">{t('messages.empty_desc')}</p>
          </div>
        ) : (
          <ul>
            {conversations.map((conv) => (
              <ConversationRow
                key={conv.id}
                conversation={conv}
                myId={myId}
                active={conv.id === selectedId}
                preview={previews[conv.id]}
                unread={Boolean(unread[conv.id])}
                onSelect={() => onSelect(conv.id)}
                onTogglePin={() => onTogglePin(conv)}
                onToggleMute={() => onToggleMute(conv)}
                onDelete={() => onDelete(conv)}
              />
            ))}
          </ul>
        )}
      </div>
    </div>
  )
}

function ConversationRow({
  conversation,
  myId,
  active,
  preview,
  unread,
  onSelect,
  onTogglePin,
  onToggleMute,
  onDelete,
}: {
  conversation: Conversation
  myId: string
  active: boolean
  preview?: ConversationPreview
  unread: boolean
  onSelect: () => void
  onTogglePin: () => void
  onToggleMute: () => void
  onDelete: () => void
}) {
  const { t, locale } = useLanguage()
  const peerId =
    conversation.type === 'dm'
      ? conversation.memberIds.find((id) => id !== myId) ?? null
      : null
  const peer = useResolvedUser(peerId)
  const title = conversationTitle(conversation, peer, t)
  const pinned = conversation.pinnedAt !== ''
  const muted = conversation.muted

  // Sous-titre : aperçu du dernier message si dispo, sinon @handle / type.
  let subtitle: string
  if (preview) {
    const body = preview.decrypted ? preview.text : t('messages.decrypt_failed')
    subtitle = preview.mine ? t('messages.you_prefix', { text: body }) : body
  } else if (conversation.type === 'dm') {
    subtitle = peer?.username ? `@${peer.username}` : ''
  } else {
    subtitle = t(`messages.type_${conversation.type}`)
  }

  // Ligne cliquable en <div role="button"> (et non <button>) pour pouvoir
  // imbriquer le menu « … » sans bouton dans un bouton. Côté droit en colonne :
  // heure en haut, puis pastille non-lue + menu « … » EN DESSOUS de l'heure
  // (jamais par-dessus). Le « … » est révélé au survol sur PC (espace réservé via
  // `invisible`), toujours visible sur mobile (pas de survol).
  return (
    <li className="group">
      <div
        role="button"
        tabIndex={0}
        onClick={onSelect}
        onKeyDown={(e) => {
          if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault()
            onSelect()
          }
        }}
        className={cn(
          'flex w-full cursor-pointer items-center gap-3 px-4 py-3 text-left transition-colors hover:bg-accent focus-visible:bg-accent focus-visible:outline-none',
          active && 'bg-accent',
        )}
      >
        <ConversationAvatar conversation={conversation} peer={peer} className="h-11 w-11" />

        <div className="flex min-w-0 flex-1 flex-col gap-0.5">
          <span
            className={cn(
              'flex min-w-0 items-center gap-1 text-sm text-foreground',
              unread ? 'font-extrabold' : 'font-bold',
            )}
          >
            {pinned && <Pin className="h-3 w-3 shrink-0 rotate-45 text-[#8D3DFF]" aria-hidden />}
            <span className="truncate">{title}</span>
            {muted && (
              <BellOff
                className="h-3 w-3 shrink-0 text-muted-foreground"
                aria-label={t('messages.muted_aria')}
              />
            )}
          </span>
          <span
            className={cn(
              'truncate text-xs',
              unread ? 'font-semibold text-foreground' : 'text-muted-foreground',
            )}
          >
            {subtitle}
          </span>
        </div>

        {/* Colonne droite : heure (haut), puis pastille + « … » (bas) */}
        <div className="flex shrink-0 flex-col items-end justify-between gap-1 self-stretch py-0.5">
          <span className="text-[11px] text-muted-foreground">
            {formatDay(conversation.updatedAt, locale)}
          </span>

          <div className="flex items-center gap-1.5">
            {unread && (
              <span
                aria-label={t('messages.unread_aria')}
                className="h-2.5 w-2.5 shrink-0 rounded-full bg-[#8D3DFF] shadow-[0_0_8px_rgba(141,61,255,0.6)]"
              />
            )}
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  aria-label={t('messages.actions_aria')}
                  onClick={(e) => e.stopPropagation()}
                  className="h-6 w-6 rounded-full text-muted-foreground hover:bg-accent lg:invisible lg:group-hover:visible lg:group-focus-within:visible"
                >
                  <MoreHorizontal className="h-4 w-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" onClick={(e) => e.stopPropagation()}>
                <DropdownMenuItem onClick={onTogglePin}>
                  {pinned ? (
                    <>
                      <PinOff className="mr-2 h-4 w-4" />
                      {t('messages.unpin')}
                    </>
                  ) : (
                    <>
                      <Pin className="mr-2 h-4 w-4" />
                      {t('messages.pin')}
                    </>
                  )}
                </DropdownMenuItem>
                <DropdownMenuItem onClick={onToggleMute}>
                  {muted ? (
                    <>
                      <Bell className="mr-2 h-4 w-4" />
                      {t('messages.unmute')}
                    </>
                  ) : (
                    <>
                      <BellOff className="mr-2 h-4 w-4" />
                      {t('messages.mute')}
                    </>
                  )}
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  onClick={onDelete}
                  className="text-destructive focus:text-destructive"
                >
                  <Trash2 className="mr-2 h-4 w-4" />
                  {t('messages.delete_for_me')}
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>
      </div>
    </li>
  )
}

/** Date courte (jour) de dernière activité. */
function formatDay(iso: string, locale: string): string {
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return ''
  const intl = locale === 'en' ? 'en-US' : 'fr-FR'
  const now = new Date()
  const sameDay = date.toDateString() === now.toDateString()
  if (sameDay) {
    return new Intl.DateTimeFormat(intl, { hour: '2-digit', minute: '2-digit' }).format(date)
  }
  return new Intl.DateTimeFormat(intl, { day: 'numeric', month: 'short' }).format(date)
}
