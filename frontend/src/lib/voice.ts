'use client'

/**
 * Capture audio pour les messages vocaux (DM + posts).
 *
 * S'appuie sur `MediaRecorder` + `getUserMedia` (la permission micro est
 * demandée AU MOMENT du `start`, pas avant). Produit un `Blob` opaque que les
 * appelants traitent comme n'importe quel média :
 *   - DM  : chiffré puis `uploadEncryptedMedia` (serveur aveugle, cf. media.ts).
 *   - post: `uploadMedia` en clair ; le FRONT impose `type:'audio'` car la
 *           détection magic-bytes du WebM est ambiguë (cf. DECISIONS).
 *
 * La durée est plafonnée CÔTÉ CLIENT (`MAX_VOICE_MS`) : un DM est E2EE, le
 * serveur ne peut pas décoder l'audio pour la vérifier. Le cap de taille du
 * media-service reste la vraie barrière serveur.
 */

import { useCallback, useEffect, useRef, useState } from 'react'

/** Durée maximale d'un vocal (1 minute). Plafond appliqué côté client. */
export const MAX_VOICE_MS = 60_000

/** Un vocal enregistré, prêt à être prévisualisé / uploadé. */
export interface RecordedVoice {
  blob: Blob
  mime: string
  durationMs: number
  /** Object URL local pour une lecture immédiate (à révoquer par l'appelant). */
  url: string
}

/**
 * Candidats `mimeType` par ordre de préférence. On vise un conteneur dont les
 * octets sont reconnus comme audio par le media-service quand c'est possible
 * (ogg/mp4), avec repli WebM/Opus (Chrome/Firefox). Le dernier `''` laisse le
 * navigateur choisir son défaut.
 */
const MIME_CANDIDATES = [
  'audio/webm;codecs=opus',
  'audio/ogg;codecs=opus',
  'audio/webm',
  'audio/mp4',
  'audio/aac',
  '',
]

/** Premier `mimeType` supporté par `MediaRecorder` sur ce navigateur. */
function pickMimeType(): string {
  if (typeof MediaRecorder === 'undefined') return ''
  for (const c of MIME_CANDIDATES) {
    if (c === '' || MediaRecorder.isTypeSupported(c)) return c
  }
  return ''
}

/** Vrai si l'appareil a un pointeur grossier (tactile) → UX « appui maintenu ». */
export function isCoarsePointer(): boolean {
  if (typeof window === 'undefined' || !window.matchMedia) return false
  return window.matchMedia('(pointer: coarse)').matches
}

/** Formate une durée (ms) en `m:ss`. */
export function formatVoiceDuration(ms: number): string {
  const total = Math.max(0, Math.round(ms / 1000))
  const m = Math.floor(total / 60)
  const s = total % 60
  return `${m}:${s.toString().padStart(2, '0')}`
}

export type RecorderState = 'idle' | 'requesting' | 'recording' | 'error'

interface UseVoiceRecorderOptions {
  /** Appelé quand l'enregistrement est validé (durée > 0). */
  onComplete: (recorded: RecordedVoice) => void
  /** Appelé quand l'utilisateur annule (corbeille / glissé). */
  onCancel?: () => void
  /** Appelé si la permission est refusée ou l'enregistrement échoue. */
  onError?: (err: unknown) => void
  /** Plafond de durée (défaut `MAX_VOICE_MS`). */
  maxMs?: number
}

export interface VoiceRecorderControls {
  state: RecorderState
  /** Durée écoulée (ms), rafraîchie ~30 fps pendant l'enregistrement. */
  elapsedMs: number
  /** Niveau sonore instantané 0..1 (pour l'onde animée). */
  level: number
  /** Vrai pendant que la permission micro est demandée. */
  start: () => Promise<void>
  /** Valide et finalise → `onComplete` (sauf si durée nulle). */
  stop: () => void
  /** Annule et jette les octets → `onCancel`. */
  cancel: () => void
}

/**
 * Pilote `MediaRecorder` pour un vocal. Le composant d'UI gère les gestes
 * (clic PC vs appui-maintenu mobile) et appelle `start`/`stop`/`cancel`.
 */
export function useVoiceRecorder(options: UseVoiceRecorderOptions): VoiceRecorderControls {
  const { onComplete, onCancel, onError, maxMs = MAX_VOICE_MS } = options

  const [state, setState] = useState<RecorderState>('idle')
  const [elapsedMs, setElapsedMs] = useState(0)
  const [level, setLevel] = useState(0)

  const recorderRef = useRef<MediaRecorder | null>(null)
  const streamRef = useRef<MediaStream | null>(null)
  const chunksRef = useRef<BlobPart[]>([])
  const startedAtRef = useRef(0)
  const cancelledRef = useRef(false)
  const maxTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const rafRef = useRef<number | null>(null)
  const audioCtxRef = useRef<AudioContext | null>(null)
  const analyserRef = useRef<AnalyserNode | null>(null)

  // Garder les derniers callbacks sans recréer start/stop à chaque rendu.
  const cbRef = useRef({ onComplete, onCancel, onError })
  cbRef.current = { onComplete, onCancel, onError }

  const teardown = useCallback(() => {
    if (maxTimerRef.current) clearTimeout(maxTimerRef.current)
    maxTimerRef.current = null
    if (rafRef.current) cancelAnimationFrame(rafRef.current)
    rafRef.current = null
    streamRef.current?.getTracks().forEach((t) => t.stop())
    streamRef.current = null
    if (audioCtxRef.current && audioCtxRef.current.state !== 'closed') {
      void audioCtxRef.current.close()
    }
    audioCtxRef.current = null
    analyserRef.current = null
    recorderRef.current = null
    setLevel(0)
  }, [])

  // Métrage du niveau sonore (RMS) pour l'onde + horloge écoulée.
  const tick = useCallback(() => {
    const analyser = analyserRef.current
    if (analyser) {
      const buf = new Uint8Array(analyser.fftSize)
      analyser.getByteTimeDomainData(buf)
      let sum = 0
      for (let i = 0; i < buf.length; i++) {
        const v = (buf[i] - 128) / 128
        sum += v * v
      }
      const rms = Math.sqrt(sum / buf.length)
      // Compression douce pour rendre les petites variations visibles.
      setLevel(Math.min(1, rms * 3))
    }
    setElapsedMs(performance.now() - startedAtRef.current)
    rafRef.current = requestAnimationFrame(tick)
  }, [])

  const start = useCallback(async () => {
    if (state === 'recording' || state === 'requesting') return
    cancelledRef.current = false
    chunksRef.current = []
    setElapsedMs(0)
    setState('requesting')
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true })
      streamRef.current = stream

      const mime = pickMimeType()
      const recorder = mime ? new MediaRecorder(stream, { mimeType: mime }) : new MediaRecorder(stream)
      recorderRef.current = recorder

      recorder.ondataavailable = (e) => {
        if (e.data.size > 0) chunksRef.current.push(e.data)
      }
      recorder.onstop = () => {
        const elapsed = performance.now() - startedAtRef.current
        const wasCancelled = cancelledRef.current
        const type = recorder.mimeType || mime || 'audio/webm'
        const blob = new Blob(chunksRef.current, { type })
        teardown()
        setState('idle')
        setElapsedMs(0)
        if (wasCancelled || blob.size === 0 || elapsed < 300) {
          cbRef.current.onCancel?.()
          return
        }
        cbRef.current.onComplete({
          blob,
          mime: type,
          durationMs: Math.min(elapsed, maxMs),
          url: URL.createObjectURL(blob),
        })
      }

      // Analyse temps réel pour l'onde animée.
      const AudioCtx = window.AudioContext || (window as unknown as { webkitAudioContext: typeof AudioContext }).webkitAudioContext
      const ctx = new AudioCtx()
      audioCtxRef.current = ctx
      const source = ctx.createMediaStreamSource(stream)
      const analyser = ctx.createAnalyser()
      analyser.fftSize = 256
      source.connect(analyser)
      analyserRef.current = analyser

      startedAtRef.current = performance.now()
      recorder.start()
      setState('recording')
      rafRef.current = requestAnimationFrame(tick)
      // Arrêt automatique au plafond de durée → finalise comme un stop normal.
      maxTimerRef.current = setTimeout(() => {
        if (recorderRef.current?.state === 'recording') recorderRef.current.stop()
      }, maxMs)
    } catch (err) {
      teardown()
      setState('error')
      cbRef.current.onError?.(err)
    }
  }, [state, maxMs, tick, teardown])

  const stop = useCallback(() => {
    if (recorderRef.current?.state === 'recording') {
      recorderRef.current.stop()
    }
  }, [])

  const cancel = useCallback(() => {
    cancelledRef.current = true
    if (recorderRef.current?.state === 'recording') {
      recorderRef.current.stop()
    } else {
      teardown()
      setState('idle')
    }
  }, [teardown])

  // Nettoyage si le composant est démonté en pleine capture.
  useEffect(() => () => {
    cancelledRef.current = true
    if (recorderRef.current?.state === 'recording') recorderRef.current.stop()
    teardown()
  }, [teardown])

  return { state, elapsedMs, level, start, stop, cancel }
}
