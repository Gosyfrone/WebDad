'use client'

import { useState } from 'react'
import { Loader2 } from 'lucide-react'

import { startDM, type Conversation } from '@/lib/messages'
import type { RelationUser } from '@/types'
import { useToast } from '@/hooks/use-toast'
import { useLanguage } from '@/components/language-provider'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { UserSearch } from '@/components/messages/user-search'

interface NewDMDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  myId: string
  /** Conversation créée (ou retrouvée) — le parent la sélectionne. */
  onCreated: (conv: Conversation) => void
}

/** Modale « Nouveau message » : recherche une personne → ouvre/retrouve un DM. */
export function NewDMDialog({ open, onOpenChange, myId, onCreated }: NewDMDialogProps) {
  const { t } = useLanguage()
  const { toast } = useToast()
  const [pending, setPending] = useState(false)

  async function pick(user: RelationUser) {
    if (pending) return
    setPending(true)
    try {
      const conv = await startDM(user.id)
      onCreated(conv)
      onOpenChange(false)
    } catch {
      toast({ title: t('messages.dm_failed'), variant: 'destructive' })
    } finally {
      setPending(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>{t('messages.new_dm')}</DialogTitle>
          <DialogDescription>{t('messages.new_dm_desc')}</DialogDescription>
        </DialogHeader>
        {pending ? (
          <div className="flex justify-center py-10">
            <Loader2 className="h-6 w-6 animate-spin text-[#5B6CFF] dark:text-[#9aa6ff]" />
          </div>
        ) : (
          <UserSearch excludeIds={[myId]} onPick={pick} />
        )}
      </DialogContent>
    </Dialog>
  )
}
