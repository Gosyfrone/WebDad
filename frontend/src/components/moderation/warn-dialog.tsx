'use client'

import { useEffect, useState } from 'react'
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
  /** Message pré-rempli à l'ouverture (ex. justification d'un retrait de contenu). */
  defaultMessage?: string
  /** Titre/description personnalisés (ex. « Supprimer et avertir »). */
  title?: string
  description?: string
  /** Libellé du bouton de validation (défaut : « Avertir »). */
  confirmLabel?: string
  /**
   * Action de validation PERSONNALISÉE. Quand fournie, elle REMPLACE l'émission
   * d'avertissement par défaut : la modale délègue tout au parent (ex. supprimer
   * le contenu PUIS avertir PUIS clôturer, de façon atomique). Le contenu n'est
   * donc touché qu'à la validation — annuler n'a aucun effet de bord.
   */
  onConfirm?: (message: string) => Promise<void>
  /** Appelé après une validation réussie (ex. recharger le ticket). */
  onSubmitted?: () => void
}

/**
 * Émission d'un avertissement par un modérateur. L'utilisateur ciblé verra le
 * message à sa prochaine connexion/action (interception via WarningsProvider).
 */
export function WarnDialog({
  open,
  onOpenChange,
  targetUserId,
  ticketId,
  defaultMessage,
  title,
  description,
  confirmLabel,
  onConfirm,
  onSubmitted,
}: WarnDialogProps) {
  const t = useT()
  const { toast } = useToast()
  const [message, setMessage] = useState('')
  const [submitting, setSubmitting] = useState(false)

  // (Ré)initialise le texte à chaque ouverture sur le message pré-rempli fourni
  // (ex. « Votre publication a été supprimée… » après un retrait de contenu).
  useEffect(() => {
    if (open) setMessage(defaultMessage ?? '')
  }, [open, defaultMessage])

  async function onSubmit() {
    const text = message.trim()
    if (!text) return
    setSubmitting(true)
    try {
      if (onConfirm) {
        // Action atomique pilotée par le parent (ex. supprimer + avertir + clôturer) :
        // le contenu n'est touché qu'ici, à la validation. Le parent gère son toast.
        await onConfirm(text)
      } else {
        await issueWarning({ targetUserId, ticketId, message: text })
        toast({ title: t('warn.sent') })
      }
      setMessage('')
      onOpenChange(false)
      onSubmitted?.()
    } catch {
      // En mode délégué (`onConfirm`), le parent affiche déjà un message adapté à SON
      // action (ex. « retrait impossible ») et relance l'erreur → on ne double pas
      // avec un toast d'avertissement trompeur ; on garde juste la modale ouverte.
      if (!onConfirm) toast({ title: t('warn.error'), variant: 'brand' })
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{title ?? t('warn.title')}</DialogTitle>
          <DialogDescription>{description ?? t('warn.desc')}</DialogDescription>
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
            {submitting ? <Loader2 className="h-4 w-4 animate-spin" /> : (confirmLabel ?? t('warn.submit'))}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
