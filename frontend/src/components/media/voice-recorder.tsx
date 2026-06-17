'use client'

/**
 * Bouton + UX d'enregistrement vocal, partagé DM et posts. Deux modes selon
 * l'appareil (détecté via le type de pointeur) :
 *
 *   - PC (pointeur fin)  : CLIC sur le micro → enregistre ; pendant la capture
 *     une barre couvre la zone de saisie (onde animée + chrono). FLÈCHE droite
 *     = valider/envoyer, CORBEILLE = annuler.
 *   - Mobile (tactile)   : APPUI MAINTENU sur le micro → enregistre ; on glisse
 *     vers la corbeille pour annuler, ou on RELÂCHE pour valider.
 *
 * Le composant rend, en cours d'enregistrement, une barre `absolute inset-0` :
 * le conteneur parent (toolbar / zone de saisie) DOIT être `position: relative`.
 *
 * La permission micro est demandée au premier `start` (cf. `useVoiceRecorder`).
 */

import { useCallback, useRef, useState } from 'react'
import { ArrowRight, Loader2, Mic, Trash2 } from 'lucide-react'

import { cn } from '@/lib/utils'
import { useT } from '@/components/language-provider'
import {
  formatVoiceDuration,
  isCoarsePointer,
  MAX_VOICE_MS,
  useVoiceRecorder,
  type RecordedVoice,
} from '@/lib/voice'

/** Glissé horizontal (px) au-delà duquel on annule en mode tactile. */
const CANCEL_DRAG_PX = 90

interface VoiceRecorderProps {
  onRecorded: (recorded: RecordedVoice) => void
  /** Permission refusée / erreur micro (l'appelant affiche un toast). */
  onError?: (err: unknown) => void
  disabled?: boolean
  className?: string
}

export function VoiceRecorder({ onRecorded, onError, disabled, className }: VoiceRecorderProps) {
  const t = useT()
  const [dragX, setDragX] = useState(0)
  const willCancelRef = useRef(false)
  const holdModeRef = useRef(false)
  const startXRef = useRef(0)

  const { state, elapsedMs, level, start, stop, cancel } = useVoiceRecorder({
    onComplete: onRecorded,
    onError,
  })

  const recording = state === 'recording'
  const requesting = state === 'requesting'

  // --- Mode PC : clic ↦ start ; boutons flèche/corbeille pour finir ----------
  const handleClickMic = useCallback(() => {
    // Sur tactile, le geste est géré par pointerdown/up (appui maintenu) ; le
    // click synthétique qui suit le relâchement ne doit PAS relancer la capture.
    if (isCoarsePointer()) return
    if (disabled || recording || requesting) return
    holdModeRef.current = false
    void start()
  }, [disabled, recording, requesting, start])

  // --- Mode mobile : appui maintenu + glissé ---------------------------------
  const handlePointerDown = useCallback(
    (e: React.PointerEvent) => {
      if (disabled || recording || requesting) return
      if (!isCoarsePointer()) return // le clic PC est géré par onClick
      holdModeRef.current = true
      willCancelRef.current = false
      startXRef.current = e.clientX
      setDragX(0)
      ;(e.target as Element).setPointerCapture?.(e.pointerId)
      void start()
    },
    [disabled, recording, requesting, start],
  )

  const handlePointerMove = useCallback((e: React.PointerEvent) => {
    if (!holdModeRef.current) return
    const dx = Math.abs(e.clientX - startXRef.current)
    setDragX(dx)
    willCancelRef.current = dx > CANCEL_DRAG_PX
  }, [])

  const handlePointerUp = useCallback(() => {
    if (!holdModeRef.current) return
    holdModeRef.current = false
    setDragX(0)
    if (willCancelRef.current) cancel()
    else stop()
  }, [cancel, stop])

  const progress = Math.min(1, elapsedMs / MAX_VOICE_MS)
  const nearCancel = dragX > CANCEL_DRAG_PX

  return (
    <div className={cn('contents', className)}>
      {/* Bouton micro (mode repos) */}
      <button
        type="button"
        disabled={disabled || requesting}
        onClick={handleClickMic}
        onPointerDown={handlePointerDown}
        onPointerMove={handlePointerMove}
        onPointerUp={handlePointerUp}
        onPointerCancel={handlePointerUp}
        aria-label={t('voice.record')}
        className={cn(
          'flex h-9 w-9 items-center justify-center rounded-full text-muted-foreground transition hover:bg-muted hover:text-foreground disabled:opacity-50',
          recording && 'text-red-500',
        )}
      >
        {requesting ? <Loader2 className="h-5 w-5 animate-spin" /> : <Mic className="h-5 w-5" />}
      </button>

      {/* Barre d'enregistrement : couvre la zone de saisie (parent relative) */}
      {recording && (
        <div className="absolute inset-0 z-20 flex items-center gap-3 rounded-md bg-background px-3">
          {/* Corbeille (clic en mode PC ; cible du glissé en mode mobile) */}
          <button
            type="button"
            onClick={cancel}
            aria-label={t('voice.cancel')}
            className={cn(
              'flex h-9 w-9 shrink-0 items-center justify-center rounded-full transition',
              nearCancel ? 'bg-red-500 text-white' : 'text-red-500 hover:bg-muted',
            )}
          >
            <Trash2 className="h-5 w-5" />
          </button>

          {/* Onde animée + chrono */}
          <div className="flex flex-1 items-center gap-2 overflow-hidden">
            <span className="h-2.5 w-2.5 shrink-0 animate-pulse rounded-full bg-red-500" />
            <div className="flex h-8 flex-1 items-center gap-[2px]">
              {Array.from({ length: 28 }).map((_, i) => {
                // Onde « vivante » : amplitude pilotée par le niveau micro.
                const phase = Math.sin(i * 0.7 + elapsedMs / 120)
                const h = 0.2 + Math.abs(phase) * level * 1.6
                return (
                  <span
                    key={i}
                    className="w-[3px] shrink-0 rounded-full bg-red-500/70"
                    style={{ height: `${Math.min(100, Math.round(h * 100))}%` }}
                  />
                )
              })}
            </div>
            <span className="shrink-0 text-xs tabular-nums text-muted-foreground">
              {formatVoiceDuration(elapsedMs)} / {formatVoiceDuration(MAX_VOICE_MS)}
            </span>
          </div>

          {/* Indice de glissé (mobile) ou bouton envoyer (PC) */}
          {holdModeRef.current ? (
            <span className="shrink-0 text-xs text-muted-foreground">
              {nearCancel ? t('voice.release_to_cancel') : t('voice.slide_to_cancel')}
            </span>
          ) : (
            <button
              type="button"
              onClick={stop}
              aria-label={t('voice.send')}
              className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-primary text-primary-foreground transition hover:opacity-90"
            >
              <ArrowRight className="h-5 w-5" />
            </button>
          )}

          {/* Liseré de progression vers le plafond 60 s */}
          <span
            className="pointer-events-none absolute bottom-0 left-0 h-0.5 bg-red-500/70"
            style={{ width: `${progress * 100}%` }}
          />
        </div>
      )}
    </div>
  )
}
