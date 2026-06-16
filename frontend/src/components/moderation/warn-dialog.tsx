'use client'

import { useState } from 'react'
import { Loader2 } from 'lucide-react'

import { MAX_WARNING_MESSAGE_HINT, issueWarning } from '@/lib/reports'
import { useT } from '@/components/language-provider'
import { useToast } from '@/hooks/use-toast'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

interface WarnDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  targetUserId: string
  ticketId?: string
}

/**
 * Émission d'un avertissement par un modérateur. L'utilisateur ciblé verra le
 * message à sa prochaine connexion/action (interception via WarningsProvider).
 */
export function WarnDialog({ open, onOpenChange, targetUserId, ticketId }: WarnDialogProps) {
  const t = useT()
  const { toast } = useToast()
  const [message, setMessage] = useState('')
  const [submitting, setSubmitting] = useState(false)

  async function onSubmit() {
    const text = message.trim()
    if (!text) return
    setSubmitting(true)
    try {
      await issueWarning({ targetUserId, ticketId, message: text })
      toast({ title: t('warn.sent') })
      setMessage('')
      onOpenChange(false)
    } catch {
      toast({ title: t('warn.error'), variant: 'brand' })
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('warn.title')}</DialogTitle>
          <DialogDescription>{t('warn.desc')}</DialogDescription>
        </DialogHeader>
        <Textarea
          value={message}
          onChange={(e) => setMessage(e.target.value)}
          placeholder={t('warn.placeholder')}
          maxLength={MAX_WARNING_MESSAGE_HINT}
          rows={4}
        />
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)} disabled={submitting}>
            {t('common.cancel')}
          </Button>
          <Button onClick={() => void onSubmit()} disabled={submitting || !message.trim()}>
            {submitting ? <Loader2 className="h-4 w-4 animate-spin" /> : t('warn.submit')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
