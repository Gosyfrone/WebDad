'use client'

/**
 * Lecteur de message vocal (DM + posts), façon Instagram :
 *   - bouton play/pause,
 *   - onde cliquable / glissable pour se déplacer (seek),
 *   - horloge (position / durée),
 *   - vitesse cyclique 1x → 1.5x → 2x → 1x.
 *
 * `src` est une URL prête à lire : object URL local (prévisualisation) ou URL
 * gateway résolue (`resolveMediaUrl`) pour un post déjà publié. Pour un DM
 * chiffré, l'appelant déchiffre d'abord et passe l'object URL du clair.
 */

import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Pause, Play } from 'lucide-react'

import { cn } from '@/lib/utils'
import { formatVoiceDuration } from '@/lib/voice'

/** Vitesses de lecture cyclées par le bouton. */
const SPEEDS = [1, 1.5, 2] as const

interface VoiceMessageProps {
  src: string
  /** Durée connue (ms) — repli quand les métadonnées WebM annoncent Infinity. */
  durationMs?: number
  className?: string
}

/**
 * Barres « onde » décoratives mais STABLES pour une même source (hash du `src`),
 * pour éviter qu'elles sautent à chaque rendu. La portion lue est colorée.
 */
function useWaveformBars(src: string, count = 40): number[] {
  return useMemo(() => {
    let seed = 0
    for (let i = 0; i < src.length; i++) seed = (seed * 31 + src.charCodeAt(i)) >>> 0
    const bars: number[] = []
    for (let i = 0; i < count; i++) {
      seed = (seed * 1103515245 + 12345) & 0x7fffffff
      bars.push(0.25 + (seed % 1000) / 1000 * 0.75)
    }
    return bars
  }, [src, count])
}

export function VoiceMessage({ src, durationMs, className }: VoiceMessageProps) {
  const audioRef = useRef<HTMLAudioElement | null>(null)
  const barsRef = useRef<HTMLDivElement | null>(null)
  const bars = useWaveformBars(src)

  const [playing, setPlaying] = useState(false)
  const [current, setCurrent] = useState(0)
  const [duration, setDuration] = useState(durationMs ? durationMs / 1000 : 0)
  const [speedIdx, setSpeedIdx] = useState(0)
  const seekingRef = useRef(false)

  // Synchronise la vitesse choisie sur l'élément <audio>.
  useEffect(() => {
    if (audioRef.current) audioRef.current.playbackRate = SPEEDS[speedIdx]
  }, [speedIdx])

  const onLoadedMetadata = useCallback(() => {
    const a = audioRef.current
    if (!a) return
    // Bug Chrome/WebM : durée Infinity tant qu'on n'a pas « scrubbé » la fin.
    if (!Number.isFinite(a.duration)) {
      a.currentTime = 1e101
      return
    }
    setDuration(a.duration)
  }, [])

  const onTimeUpdate = useCallback(() => {
    const a = audioRef.current
    if (!a || seekingRef.current) return
    if (!Number.isFinite(a.duration)) return
    if (duration === 0 || (durationMs && Math.abs(a.duration - duration) > 0.5)) {
      setDuration(a.duration)
    }
    setCurrent(a.currentTime)
  }, [duration, durationMs])

  const toggle = useCallback(() => {
    const a = audioRef.current
    if (!a) return
    if (a.paused) {
      a.playbackRate = SPEEDS[speedIdx]
      void a.play()
    } else {
      a.pause()
    }
  }, [speedIdx])

  const cycleSpeed = useCallback(() => {
    setSpeedIdx((i) => (i + 1) % SPEEDS.length)
  }, [])

  // Seek depuis un clic/glissé sur l'onde.
  const seekFromClientX = useCallback(
    (clientX: number) => {
      const a = audioRef.current
      const el = barsRef.current
      if (!a || !el || !Number.isFinite(a.duration) || a.duration === 0) return
      const rect = el.getBoundingClientRect()
      const ratio = Math.min(1, Math.max(0, (clientX - rect.left) / rect.width))
      a.currentTime = ratio * a.duration
      setCurrent(a.currentTime)
    },
    [],
  )

  const onPointerDown = useCallback(
    (e: React.PointerEvent) => {
      seekingRef.current = true
      ;(e.target as Element).setPointerCapture?.(e.pointerId)
      seekFromClientX(e.clientX)
    },
    [seekFromClientX],
  )
  const onPointerMove = useCallback(
    (e: React.PointerEvent) => {
      if (seekingRef.current) seekFromClientX(e.clientX)
    },
    [seekFromClientX],
  )
  const onPointerUp = useCallback(() => {
    seekingRef.current = false
  }, [])

  const progress = duration > 0 ? current / duration : 0
  const remainingMs = Math.max(0, (duration - current) * 1000)
  // Pendant la lecture on affiche le temps restant ; à l'arrêt, la durée totale.
  const timeLabel = formatVoiceDuration(playing || current > 0 ? remainingMs : duration * 1000)

  return (
    <div className={cn('flex w-full max-w-xs items-center gap-2', className)}>
      <audio
        ref={audioRef}
        src={src}
        preload="metadata"
        onLoadedMetadata={onLoadedMetadata}
        onDurationChange={onLoadedMetadata}
        onTimeUpdate={onTimeUpdate}
        onPlay={() => setPlaying(true)}
        onPause={() => setPlaying(false)}
        onEnded={() => {
          setPlaying(false)
          setCurrent(0)
        }}
      />

      <button
        type="button"
        onClick={toggle}
        aria-label={playing ? 'pause' : 'play'}
        className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-primary text-primary-foreground transition hover:opacity-90"
      >
        {playing ? <Pause className="h-4 w-4" /> : <Play className="h-4 w-4 translate-x-[1px]" />}
      </button>

      <div
        ref={barsRef}
        onPointerDown={onPointerDown}
        onPointerMove={onPointerMove}
        onPointerUp={onPointerUp}
        className="flex h-8 flex-1 cursor-pointer touch-none items-center gap-[2px]"
      >
        {bars.map((h, i) => {
          const filled = i / bars.length <= progress
          return (
            <span
              key={i}
              className={cn(
                'w-[3px] shrink-0 rounded-full transition-colors',
                filled ? 'bg-primary' : 'bg-muted-foreground/40',
              )}
              style={{ height: `${Math.round(h * 100)}%` }}
            />
          )
        })}
      </div>

      <span className="w-9 shrink-0 text-right text-xs tabular-nums text-muted-foreground">
        {timeLabel}
      </span>

      <button
        type="button"
        onClick={cycleSpeed}
        aria-label="playback speed"
        className="shrink-0 rounded-full bg-muted px-2 py-0.5 text-xs font-semibold tabular-nums text-muted-foreground transition hover:bg-muted/70"
      >
        {SPEEDS[speedIdx]}x
      </button>
    </div>
  )
}
