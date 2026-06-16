'use client'

import { useState } from 'react'
import { Loader2, X } from 'lucide-react'

import { createGroup, PeerKeyMissingError, type Conversation } from '@/lib/messages'
import type { RelationUser } from '@/types'
import { useToast } from '@/hooks/use-toast'
import { useLanguage } from '@/components/language-provider'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { ActivityPresenceDot } from '@/components/profil/activity-presence-dot'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { UserSearch } from '@/components/messages/user-search'

interface NewGroupDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  myId: string
  onCreated: (conv: Conversation) => void
}

/** Modale « Nouveau groupe » : nom (chiffré) + sélection de membres initiaux. */
export function NewGroupDialog({ open, onOpenChange, myId, onCreated }: NewGroupDialogProps) {
  const { t } = useLanguage()
  const { toast } = useToast()
  const [name, setName] = useState('')
  const [members, setMembers] = useState<RelationUser[]>([])
  const [pending, setPending] = useState(false)

  function reset() {
    setName('')
    setMembers([])
  }

  function handleOpenChange(next: boolean) {
    if (!next) reset()
    onOpenChange(next)
  }

  function addMember(user: RelationUser) {
    setMembers((prev) => (prev.some((m) => m.id === user.id) ? prev : [...prev, user]))
  }

  function removeMember(id: string) {
    setMembers((prev) => prev.filter((m) => m.id !== id))
  }

  async function create() {
    if (!name.trim() || pending) return
    setPending(true)
    try {
      const conv = await createGroup(
        name.trim(),
        members.map((m) => m.id),
      )
      onCreated(conv)
      reset()
      onOpenChange(false)
    } catch (err) {
      if (err instanceof PeerKeyMissingError) {
        toast({ title: t('messages.peer_not_activated'), variant: 'brand' })
      } else {
        toast({ title: t('messages.group_failed'), variant: 'destructive' })
      }
    } finally {
      setPending(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>{t('messages.new_group')}</DialogTitle>
          <DialogDescription>{t('messages.new_group_desc')}</DialogDescription>
        </DialogHeader>

        <div className="flex flex-col gap-3">
          <Input
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder={t('messages.group_name_placeholder')}
            maxLength={80}
          />

          {members.length > 0 && (
            <div className="flex flex-wrap gap-2">
              {members.map((m) => (
                <span
                  key={m.id}
                  className="glass flex items-center gap-1.5 rounded-full border py-1 pl-1 pr-2 text-xs"
                >
                  <Avatar className="h-5 w-5">
                    {m.avatarUrl && <AvatarImage src={m.avatarUrl} alt={m.displayName} />}
                    <AvatarFallback className="text-[10px]">
                      {(m.displayName.charAt(0) || '?').toUpperCase()}
                    </AvatarFallback>
                    <ActivityPresenceDot userId={m.id} className="h-2 w-2 border" />
                  </Avatar>
                  <span className="font-semibold">{m.displayName}</span>
                  <button
                    type="button"
                    onClick={() => removeMember(m.id)}
                    aria-label={t('messages.remove_member')}
                    className="rounded-full p-0.5 hover:bg-accent"
                  >
                    <X className="h-3 w-3" />
                  </button>
                </span>
              ))}
            </div>
          )}

          <UserSearch
            excludeIds={[myId, ...members.map((m) => m.id)]}
            onPick={addMember}
            autoFocus={false}
          />
        </div>

        <DialogFooter>
          <Button type="button" onClick={create} disabled={!name.trim() || pending}>
            {pending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            {t('messages.create_group')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
