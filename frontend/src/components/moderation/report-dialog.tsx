'use client'

import { type ChangeEvent, useRef, useState } from 'react'
import { ImagePlus, Loader2, X } from 'lucide-react'

import {
  ALL_REASONS,
  MAX_ATTACHMENT_BYTES,
  MAX_BUG_TEXT,
  MAX_MODERATION_TEXT,
  ReportApiError,
  categoryForReason,
  createReport,
  type ReportEntityType,
  type ReportReason,
} from '@/lib/reports'
import { uploadMedia } from '@/lib/media'
import { useT } from '@/components/language-provider'
import { useToast } from '@/hooks/use-toast'
import { cn } from '@/lib/utils'
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

interface ReportDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  /** Entité signalée (post/message/profil). Conservée même pour un motif « bug ». */
  entityType?: ReportEntityType
  entityId?: string
  /** Propriétaire de l'entité (auteur du post/message, utilisateur du profil). */
  entityOwnerId?: string
  /** Libellé optionnel de la cible (ex. « @alice »), affiché en sous-titre. */
  targetLabel?: string
  /**
   * Contenu EN CLAIR d'un message chiffré que le signaleur (participant) accepte
   * de divulguer à la modération. Fourni uniquement pour les signalements de
   * message (E2EE) : le serveur ne peut pas lire le message, seul le destinataire
   * peut le révéler pour CE signalement. Affiche un avis de transmission.
   */
  disclosedContent?: string
}

/**
 * Formulaire de signalement réutilisable (posts, messages, profils, bugs).
 * Le menu déroulant des MOTIFS inclut « Bug technique » : le choisir bascule le
 * signalement en catégorie bug (limite 500, onglet Administration) ; les autres
 * motifs → modération (limite 255, onglet Modération). Texte borné + compteur,
 * pièce jointe image optionnelle (≤ 5 Mb via media-service). Thème clair/sombre
 * via les tokens shadcn ; libellés i18n.
 */
export function ReportDialog({ open, onOpenChange, entityType, entityId, entityOwnerId, targetLabel, disclosedContent }: ReportDialogProps) {
  const t = useT()
  const { toast } = useToast()
  const fileInput = useRef<HTMLInputElement>(null)

  const [reason, setReason] = useState<ReportReason>(ALL_REASONS[0])
  const [text, setText] = useState('')
  const [file, setFile] = useState<File | null>(null)
  const [submitting, setSubmitting] = useState(false)

  // Catégorie + limite déduites du motif courant.
  const category = categoryForReason(reason)
  const isBug = category === 'bug'
  const maxText = isBug ? MAX_BUG_TEXT : MAX_MODERATION_TEXT

  function reset() {
    setReason(ALL_REASONS[0])
    setText('')
    setFile(null)
    if (fileInput.current) fileInput.current.value = ''
  }

  function handleClose(next: boolean) {
    if (!next) reset()
    onOpenChange(next)
  }

  function onPickFile(e: ChangeEvent<HTMLInputElement>) {
    const f = e.target.files?.[0]
    if (!f) return
    if (!f.type.startsWith('image/')) {
      toast({ title: t('report.attachment_type_error'), variant: 'brand' })
      e.target.value = ''
      return
    }
    if (f.size > MAX_ATTACHMENT_BYTES) {
      toast({ title: t('report.attachment_size_error'), variant: 'brand' })
      e.target.value = ''
      return
    }
    setFile(f)
  }

  async function onSubmit() {
    if (text.length > maxText) return
    setSubmitting(true)
    try {
      let attachmentId: string | undefined
      if (file) {
        attachmentId = (await uploadMedia(file)).id
      }
      await createReport({
        category,
        // L'entité est conservée même pour un bug (la modération/admin doit voir
        // l'élément signalé : tweet, profil…).
        entityType,
        entityId,
        entityOwnerId,
        reason,
        text: text.trim(),
        disclosedContent,
        attachmentId,
      })
      toast({ title: t('report.success') })
      handleClose(false)
    } catch (err) {
      // 409 a deux causes : (a) entité VALIDÉE par la modération → re-signalement
      // verrouillé (message serveur contient « validé ») ; (b) l'utilisateur a déjà
      // signalé cet élément (un seul signalement par entité).
      const conflict = err instanceof ReportApiError && err.status === 409
      const locked = conflict && err.message.includes('validé')
      const titleKey = locked ? 'report.locked' : conflict ? 'report.already' : 'report.error'
      toast({ title: t(titleKey), variant: 'brand' })
      if (conflict) handleClose(false)
    } finally {
      setSubmitting(false)
    }
  }

  const overLimit = text.length > maxText

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{category === 'bug' ? t('report.title_bug') : t('report.title')}</DialogTitle>
          <DialogDescription>
            {targetLabel ? t('report.subtitle_target', { target: targetLabel }) : t('report.subtitle')}
          </DialogDescription>
        </DialogHeader>

        <div className="flex flex-col gap-4">
          {/* Motif — menu déroulant */}
          <label className="flex flex-col gap-1.5 text-sm font-medium">
            {t('report.reason_label')}
            <select
              value={reason}
              onChange={(e) => setReason(e.target.value as ReportReason)}
              className="h-10 w-full rounded-md border border-input bg-background px-3 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
            >
              {ALL_REASONS.map((r) => (
                <option key={r} value={r}>
                  {t(`report.reason.${r}`)}
                </option>
              ))}
            </select>
          </label>

          {/* Texte libre — borné par catégorie */}
          <label className="flex flex-col gap-1.5 text-sm font-medium">
            {t('report.text_label')}
            <Textarea
              value={text}
              onChange={(e) => setText(e.target.value)}
              placeholder={t('report.text_placeholder')}
              rows={4}
              aria-invalid={overLimit}
            />
            <span className={cn('self-end text-xs', overLimit ? 'text-destructive' : 'text-muted-foreground')}>
              {text.length}/{maxText}
            </span>
          </label>

          {/* Pièce jointe image (≤ 5 Mb) */}
          <div className="flex flex-col gap-1.5">
            <input
              ref={fileInput}
              type="file"
              accept="image/*"
              onChange={onPickFile}
              className="hidden"
            />
            {file ? (
              <div className="flex items-center justify-between rounded-md border border-input bg-muted/40 px-3 py-2 text-sm">
                <span className="truncate">{file.name}</span>
                <button
                  type="button"
                  onClick={() => {
                    setFile(null)
                    if (fileInput.current) fileInput.current.value = ''
                  }}
                  className="ml-2 rounded-full p-1 text-muted-foreground hover:bg-primary/10 hover:text-primary"
                  aria-label={t('common.close')}
                >
                  <X className="h-4 w-4" />
                </button>
              </div>
            ) : (
              <Button type="button" variant="outline" size="sm" onClick={() => fileInput.current?.click()}>
                <ImagePlus className="mr-2 h-4 w-4" />
                {t('report.attachment_add')}
              </Button>
            )}
            <span className="text-xs text-muted-foreground">{t('report.attachment_hint')}</span>
          </div>

          {/* Message chiffré : avis de divulgation (E2EE → le serveur est aveugle ;
              le signaleur, destinataire, accepte de transmettre CE message). */}
          {disclosedContent !== undefined && (
            <div className="flex flex-col gap-1.5 rounded-md border border-dashed border-input bg-muted/40 p-3">
              <p className="text-xs text-muted-foreground">{t('report.disclose_notice')}</p>
              <p className="max-h-24 overflow-y-auto whitespace-pre-wrap break-words rounded bg-background p-2 text-sm">
                {disclosedContent || t('report.disclose_empty')}
              </p>
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => handleClose(false)} disabled={submitting}>
            {t('common.cancel')}
          </Button>
          <Button onClick={() => void onSubmit()} disabled={submitting || overLimit}>
            {submitting ? <Loader2 className="h-4 w-4 animate-spin" /> : t('report.submit')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
