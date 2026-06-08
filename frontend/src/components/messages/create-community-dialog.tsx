'use client'

import { useState } from 'react'
import { Loader2 } from 'lucide-react'

import { createCommunity, type Conversation } from '@/lib/messages'
import { useToast } from '@/hooks/use-toast'
import { useLanguage } from '@/components/language-provider'
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

interface CreateCommunityDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onCreated: (conv: Conversation) => void
}

/** Modale « Créer une communauté » : nom EN CLAIR (semi-public, annuaire). */
export function CreateCommunityDialog({ open, onOpenChange, onCreated }: CreateCommunityDialogProps) {
  const { t } = useLanguage()
  const { toast } = useToast()
  const [name, setName] = useState('')
  const [pending, setPending] = useState(false)

  function handleOpenChange(next: boolean) {
    if (!next) setName('')
    onOpenChange(next)
  }

  async function create() {
    if (!name.trim() || pending) return
    setPending(true)
    try {
      const conv = await createCommunity(name.trim())
      onCreated(conv)
      setName('')
      onOpenChange(false)
    } catch {
      toast({ title: t('messages.community_failed'), variant: 'destructive' })
    } finally {
      setPending(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>{t('messages.new_community')}</DialogTitle>
          <DialogDescription>{t('messages.community_note')}</DialogDescription>
        </DialogHeader>

        <Input
          value={name}
          onChange={(e) => setName(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && create()}
          placeholder={t('messages.community_name_placeholder')}
          maxLength={80}
          autoFocus
        />

        <DialogFooter>
          <Button type="button" onClick={create} disabled={!name.trim() || pending}>
            {pending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            {t('messages.create_community')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
