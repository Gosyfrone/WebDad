'use client'

import { Globe, Users } from 'lucide-react'

import { cn } from '@/lib/utils'
import type { Conversation } from '@/lib/messages'
import type { ResolvedUser } from '@/lib/user-cache'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'

type Translate = (key: string, params?: Record<string, string | number>) => string

/**
 * Titre d'affichage d'une conversation :
 *   - DM       → nom de l'interlocuteur (résolu) ;
 *   - groupe   → nom déchiffré (repli « Groupe ») ;
 *   - communauté → nom en clair (repli « Communauté »).
 */
export function conversationTitle(
  conv: Conversation,
  peer: ResolvedUser | null,
  t: Translate,
): string {
  if (conv.type === 'dm') return peer?.displayName ?? '…'
  if (conv.title) return conv.title
  return t(`messages.type_${conv.type}`)
}

/**
 * Avatar d'une conversation : photo de l'interlocuteur pour un DM, icône
 * thématique (groupe / communauté) sinon.
 */
export function ConversationAvatar({
  conversation,
  peer,
  className,
}: {
  conversation: Conversation
  peer: ResolvedUser | null
  className?: string
}) {
  if (conversation.type === 'dm') {
    const initials = (peer?.displayName.charAt(0) || '?').toUpperCase()
    return (
      <Avatar className={cn('shrink-0', className)}>
        {peer?.avatarUrl && <AvatarImage src={peer.avatarUrl} alt={peer.displayName} />}
        <AvatarFallback>{initials}</AvatarFallback>
      </Avatar>
    )
  }

  const Icon = conversation.type === 'community' ? Globe : Users
  return (
    <div
      className={cn(
        'flex shrink-0 items-center justify-center rounded-full bg-gradient-to-br from-[#8D3DFF] via-[#5B6CFF] to-[#47D9FF] text-white',
        className,
      )}
    >
      <Icon className="h-1/2 w-1/2" aria-hidden />
    </div>
  )
}
