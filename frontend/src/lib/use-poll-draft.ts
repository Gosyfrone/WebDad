'use client'

import { useEffect, useState, type Dispatch, type SetStateAction } from 'react'

import { exceedsMediaLimit, MAX_MEDIA_MB, uploadMedia, uploadedMediaUrl } from '@/lib/media'
import type { CreatePollPayload, PollAudience } from '@/lib/posts'
import { useToast } from '@/hooks/use-toast'
import { useT } from '@/components/language-provider'

/** Brouillon d'un choix de sondage : libellé + image optionnelle (chemin `/media/<id>`). */
export type PollChoiceDraft = { label: string; imageUrl?: string }

export interface PollDraft {
  open: boolean
  setOpen: Dispatch<SetStateAction<boolean>>
  choices: PollChoiceDraft[]
  days: number
  hours: number
  minutes: number
  /** Durée minimale : ≥ 1 min quand jours et heures sont à 0 (sinon durée nulle). */
  minMinutes: number
  audience: PollAudience
  uploadingChoice: number | null
  /** Payload prêt pour `createPost`, ou `undefined` si le sondage est incomplet. */
  payload: CreatePollPayload | undefined
  setDays: Dispatch<SetStateAction<number>>
  setHours: Dispatch<SetStateAction<number>>
  setMinutes: Dispatch<SetStateAction<number>>
  setAudience: Dispatch<SetStateAction<PollAudience>>
  updateChoice: (index: number, value: string) => void
  addChoice: () => void
  removeChoice: (index: number) => void
  pickChoiceImage: (index: number, file: File) => Promise<void>
  removeChoiceImage: (index: number) => void
  reset: () => void
}

/** État et actions du brouillon de sondage du composer (choix, durée, audience, image). */
export function usePollDraft(): PollDraft {
  const t = useT()
  const { toast } = useToast()
  const [open, setOpen] = useState(false)
  const [choices, setChoices] = useState<PollChoiceDraft[]>([{ label: '' }, { label: '' }])
  const [uploadingChoice, setUploadingChoice] = useState<number | null>(null)
  const [days, setDays] = useState(0)
  const [hours, setHours] = useState(0)
  const [minutes, setMinutes] = useState(10)
  const [audience, setAudience] = useState<PollAudience>('everyone')

  const minMinutes = days === 0 && hours === 0 ? 1 : 0
  useEffect(() => {
    if (minutes < minMinutes) setMinutes(minMinutes)
  }, [minMinutes, minutes])

  function reset() {
    setOpen(false)
    setChoices([{ label: '' }, { label: '' }])
    setUploadingChoice(null)
    setDays(0)
    setHours(0)
    setMinutes(10)
    setAudience('everyone')
  }

  function updateChoice(index: number, value: string) {
    setChoices((prev) => prev.map((c, i) => (i === index ? { ...c, label: value } : c)))
  }

  function addChoice() {
    setChoices((prev) => (prev.length >= 4 ? prev : [...prev, { label: '' }]))
  }

  function removeChoice(index: number) {
    setChoices((prev) => (prev.length <= 2 ? prev : prev.filter((_, i) => i !== index)))
  }

  /** Upload d'une image de choix (cap 5 Mo, admins exemptés) puis association au choix. */
  async function pickChoiceImage(index: number, file: File) {
    if (exceedsMediaLimit(file.size)) {
      toast({ title: t('media.too_large', { max: MAX_MEDIA_MB }), variant: 'brand' })
      return
    }
    setUploadingChoice(index)
    try {
      const uploaded = await uploadMedia(file)
      const imageUrl = uploadedMediaUrl(uploaded, 'medium')
      setChoices((prev) => prev.map((c, i) => (i === index ? { ...c, imageUrl } : c)))
    } catch {
      toast({ title: t('composer.media_failed'), variant: 'destructive' })
    } finally {
      setUploadingChoice(null)
    }
  }

  function removeChoiceImage(index: number) {
    setChoices((prev) => prev.map((c, i) => (i === index ? { ...c, imageUrl: undefined } : c)))
  }

  return {
    open,
    setOpen,
    choices,
    days,
    hours,
    minutes,
    minMinutes,
    audience,
    uploadingChoice,
    payload: buildPollPayload(open, choices, days, hours, minutes, audience),
    setDays,
    setHours,
    setMinutes,
    setAudience,
    updateChoice,
    addChoice,
    removeChoice,
    pickChoiceImage,
    removeChoiceImage,
    reset,
  }
}

function buildPollPayload(
  open: boolean,
  choices: PollChoiceDraft[],
  days: number,
  hours: number,
  minutes: number,
  audience: PollAudience,
): CreatePollPayload | undefined {
  if (!open) return undefined
  // Un choix est retenu s'il a un libellé (l'image seule ne suffit pas).
  const cleaned = choices
    .map((choice) => ({ label: choice.label.trim(), imageUrl: choice.imageUrl }))
    .filter((choice) => choice.label.length > 0)
  const durationMinutes = days * 24 * 60 + hours * 60 + minutes
  if (cleaned.length < 2 || durationMinutes < 1) return undefined
  return { choices: cleaned, durationMinutes, audience }
}
