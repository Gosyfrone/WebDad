'use client'

import { useCallback, useEffect, useState } from 'react'
import {
  ArrowDownCircle,
  ArrowUpCircle,
  Loader2,
  LogOut,
  Trash2,
  UserPlus,
  UserMinus,
} from 'lucide-react'

import {
  deleteGroup,
  inviteToGroup,
  leaveGroup,
  listMembers,
  removeMember,
  renameCommunity,
  renameGroup,
  setMemberRole,
  type Conversation,
  type MemberInfo,
} from '@/lib/messages'
import type { RelationUser } from '@/types'
import { useResolvedUser } from '@/lib/use-resolved-user'
import { useToast } from '@/hooks/use-toast'
import { useLanguage } from '@/components/language-provider'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { MessageSearch } from '@/components/messages/message-search'
import { UserSearch } from '@/components/messages/user-search'

interface ConversationInfoDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  conversation: Conversation
  myId: string
  /** Conversation mise à jour (renommage). */
  onUpdated: (conv: Conversation) => void
  /** Conversation supprimée / quittée → le parent la retire et désélectionne. */
  onRemoved: (conversationId: string) => void
}

/**
 * Panneau d'infos & gestion d'une conversation.
 *   - DM : liste des deux membres.
 *   - Groupe : inviter (tout membre), renommer / exclure / supprimer (owner),
 *     quitter (membre).
 *   - Communauté : promouvoir/rétrograder talker↔viewer, renommer, exclure,
 *     supprimer (owner), quitter (membre). Rappel : lisible par un admin.
 */
export function ConversationInfoDialog({
  open,
  onOpenChange,
  conversation,
  myId,
  onUpdated,
  onRemoved,
}: ConversationInfoDialogProps) {
  const { t } = useLanguage()
  const { toast } = useToast()
  const [members, setMembers] = useState<MemberInfo[]>([])
  const [loading, setLoading] = useState(true)
  const [name, setName] = useState(conversation.title)
  const [busy, setBusy] = useState(false)

  const isOwner = conversation.myRole === 'owner'
  const isManageable = conversation.type === 'group' || conversation.type === 'community'

  const refresh = useCallback(async () => {
    setLoading(true)
    try {
      setMembers(await listMembers(conversation.id))
    } catch {
      setMembers([])
    } finally {
      setLoading(false)
    }
  }, [conversation.id])

  useEffect(() => {
    if (!open) return
    setName(conversation.title)
    refresh()
  }, [open, conversation.title, refresh])

  function fail() {
    toast({ title: t('messages.action_failed'), variant: 'destructive' })
  }

  async function rename() {
    const next = name.trim()
    if (!next || next === conversation.title || busy) return
    setBusy(true)
    try {
      const updated =
        conversation.type === 'community'
          ? await renameCommunity(conversation, next)
          : await renameGroup(conversation, next)
      onUpdated(updated)
    } catch {
      fail()
    } finally {
      setBusy(false)
    }
  }

  async function invite(user: RelationUser) {
    if (busy) return
    setBusy(true)
    try {
      await inviteToGroup(conversation, user.id)
      await refresh()
    } catch {
      fail()
    } finally {
      setBusy(false)
    }
  }

  async function kick(userId: string) {
    if (busy) return
    setBusy(true)
    try {
      await removeMember(conversation, userId)
      await refresh()
    } catch {
      fail()
    } finally {
      setBusy(false)
    }
  }

  async function changeRole(userId: string, role: 'talker' | 'viewer') {
    if (busy) return
    setBusy(true)
    try {
      await setMemberRole(conversation, userId, role)
      await refresh()
    } catch {
      fail()
    } finally {
      setBusy(false)
    }
  }

  async function leave() {
    if (busy) return
    setBusy(true)
    try {
      await leaveGroup(conversation)
      onRemoved(conversation.id)
      onOpenChange(false)
    } catch {
      fail()
    } finally {
      setBusy(false)
    }
  }

  async function destroy() {
    if (busy) return
    setBusy(true)
    try {
      await deleteGroup(conversation)
      onRemoved(conversation.id)
      onOpenChange(false)
    } catch {
      fail()
    } finally {
      setBusy(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[85vh] max-w-md overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{t('messages.info')}</DialogTitle>
          <DialogDescription>{t(`messages.type_${conversation.type}`)}</DialogDescription>
        </DialogHeader>

        {/* Recherche dans la conversation (côté client, E2EE) */}
        <MessageSearch conversation={conversation} />

        {/* Renommage (groupe / communauté, owner) */}
        {isManageable && isOwner && (
          <div className="flex items-end gap-2">
            <div className="flex-1">
              <Input
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder={t('messages.group_name_placeholder')}
                maxLength={80}
              />
            </div>
            <Button
              type="button"
              variant="outline"
              onClick={rename}
              disabled={busy || !name.trim() || name.trim() === conversation.title}
            >
              {t('messages.rename')}
            </Button>
          </div>
        )}

        {conversation.type === 'community' && (
          <p className="rounded-xl bg-accent px-3 py-2 text-xs text-muted-foreground">
            {t('messages.community_admin_note')}
          </p>
        )}

        {/* Membres */}
        <div>
          <h3 className="mb-1 text-sm font-bold text-foreground">{t('messages.members')}</h3>
          {loading ? (
            <div className="flex justify-center py-6">
              <Loader2 className="h-5 w-5 animate-spin text-[#5B6CFF] dark:text-[#9aa6ff]" />
            </div>
          ) : (
            <ul className="max-h-56 divide-y divide-border overflow-y-auto">
              {members.map((m) => (
                <MemberRow
                  key={m.userId}
                  member={m}
                  isSelf={m.userId === myId}
                  amOwner={isOwner}
                  conversationType={conversation.type}
                  busy={busy}
                  onKick={() => kick(m.userId)}
                  onPromote={() => changeRole(m.userId, 'talker')}
                  onDemote={() => changeRole(m.userId, 'viewer')}
                />
              ))}
            </ul>
          )}
        </div>

        {/* Inviter (groupes : tout membre) */}
        {conversation.type === 'group' && (
          <div>
            <h3 className="mb-1 flex items-center gap-1.5 text-sm font-bold text-foreground">
              <UserPlus className="h-4 w-4" /> {t('messages.invite')}
            </h3>
            <UserSearch
              excludeIds={[myId, ...members.map((m) => m.userId)]}
              onPick={invite}
              autoFocus={false}
            />
          </div>
        )}

        {/* Quitter / Supprimer */}
        <div className="flex justify-end gap-2 border-t pt-3">
          {isManageable && !isOwner && (
            <Button type="button" variant="outline" onClick={leave} disabled={busy}>
              <LogOut className="mr-2 h-4 w-4" /> {t('messages.leave')}
            </Button>
          )}
          {isManageable && isOwner && (
            <Button type="button" variant="destructive" onClick={destroy} disabled={busy}>
              <Trash2 className="mr-2 h-4 w-4" /> {t('messages.delete')}
            </Button>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}

function MemberRow({
  member,
  isSelf,
  amOwner,
  conversationType,
  busy,
  onKick,
  onPromote,
  onDemote,
}: {
  member: MemberInfo
  isSelf: boolean
  amOwner: boolean
  conversationType: Conversation['type']
  busy: boolean
  onKick: () => void
  onPromote: () => void
  onDemote: () => void
}) {
  const { t } = useLanguage()
  const user = useResolvedUser(member.userId)
  const isMemberOwner = member.role === 'owner'
  const canManage = amOwner && !isSelf && !isMemberOwner

  return (
    <li className="flex items-center gap-3 py-2.5">
      <Avatar className="h-9 w-9 shrink-0">
        {user?.avatarUrl && <AvatarImage src={user.avatarUrl} alt={user.displayName} />}
        <AvatarFallback>{(user?.displayName.charAt(0) || '?').toUpperCase()}</AvatarFallback>
      </Avatar>
      <div className="flex min-w-0 flex-1 flex-col">
        <span className="truncate text-sm font-semibold text-foreground">
          {user?.displayName ?? '…'}
          {isSelf && <span className="ml-1 text-xs text-muted-foreground">({t('messages.you')})</span>}
        </span>
        <Badge variant="secondary" className="mt-0.5 w-fit text-[10px]">
          {t(`messages.role_${member.role}`)}
        </Badge>
      </div>

      {canManage && (
        <div className="flex shrink-0 items-center gap-1">
          {conversationType === 'community' &&
            (member.role === 'viewer' ? (
              <Button
                type="button"
                variant="ghost"
                size="icon"
                onClick={onPromote}
                disabled={busy}
                aria-label={t('messages.promote')}
                title={t('messages.promote')}
                className="h-8 w-8 rounded-full"
              >
                <ArrowUpCircle className="h-4 w-4" />
              </Button>
            ) : member.role === 'talker' ? (
              <Button
                type="button"
                variant="ghost"
                size="icon"
                onClick={onDemote}
                disabled={busy}
                aria-label={t('messages.demote')}
                title={t('messages.demote')}
                className="h-8 w-8 rounded-full"
              >
                <ArrowDownCircle className="h-4 w-4" />
              </Button>
            ) : null)}
          <Button
            type="button"
            variant="ghost"
            size="icon"
            onClick={onKick}
            disabled={busy}
            aria-label={t('messages.remove_member')}
            title={t('messages.remove_member')}
            className="h-8 w-8 rounded-full text-destructive hover:bg-destructive/10"
          >
            <UserMinus className="h-4 w-4" />
          </Button>
        </div>
      )}
    </li>
  )
}
