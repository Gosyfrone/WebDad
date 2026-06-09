'use client'

import { useEffect, useRef, useState } from 'react'
import { Gauge } from 'lucide-react'

import { cn } from '@/lib/utils'
import { useT } from '@/components/language-provider'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'

/** Vitesses de lecture proposées (façon Twitter/X). */
const SPEEDS = [0.5, 0.75, 1, 1.25, 1.5, 2] as const

interface FeedVideoProps {
  src: string
  /** Classes de la cellule (ratio / row-span de la grille `MediaGallery`). */
  className?: string
}

/**
 * Vidéo de fil façon Twitter/X :
 *   - **lecture auto en muet + boucle** dès qu'elle est suffisamment visible,
 *     mise en pause quand elle quitte le viewport (un seul `IntersectionObserver`
 *     à seuil ~60 %) ;
 *   - **chargement paresseux selon la pagination** : tant qu'on n'est pas
 *     *proche* (rootMargin large), l'élément `<video>` n'est même pas monté
 *     (juste un cadre noir) → aucun téléchargement ni boucle d'une vidéo loin
 *     dans le fil ;
 *   - **vitesse de lecture réglable** (overlay).
 *
 * L'autoplay non-muet étant bloqué par les navigateurs, la vidéo démarre en
 * muet ; les contrôles natifs permettent de réactiver le son / mettre en pause.
 */
export function FeedVideo({ src, className }: FeedVideoProps) {
  const t = useT()
  const containerRef = useRef<HTMLDivElement>(null)
  const videoRef = useRef<HTMLVideoElement>(null)
  const [near, setNear] = useState(false) // proche du viewport → on monte la <video>
  const [speed, setSpeed] = useState(1)

  // Observateur « proche » : monte/démonte la <video> selon la distance au
  // viewport (chargement paresseux lié à la pagination).
  useEffect(() => {
    const el = containerRef.current
    if (!el) return
    const io = new IntersectionObserver(([entry]) => setNear(entry.isIntersecting), {
      rootMargin: '400px',
    })
    io.observe(el)
    return () => io.disconnect()
  }, [])

  // Observateur « visible » : lecture auto quand ≥ 60 % visible, pause sinon.
  useEffect(() => {
    const v = videoRef.current
    if (!v) return
    const io = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting && entry.intersectionRatio >= 0.6) {
          void v.play().catch(() => {}) // peut échouer si l'onglet est masqué
        } else {
          v.pause()
        }
      },
      { threshold: [0, 0.6] },
    )
    io.observe(v)
    return () => io.disconnect()
  }, [near])

  // Applique la vitesse choisie (et la réapplique si la <video> est remontée).
  useEffect(() => {
    if (videoRef.current) videoRef.current.playbackRate = speed
  }, [speed, near])

  return (
    <div ref={containerRef} className={cn('relative overflow-hidden bg-black', className)}>
      {near && (
        <video
          ref={videoRef}
          src={src}
          muted
          loop
          playsInline
          controls
          preload="metadata"
          onLoadedMetadata={(e) => {
            e.currentTarget.playbackRate = speed
          }}
          className="h-full w-full object-contain"
        />
      )}

      {near && (
        <Popover>
          <PopoverTrigger asChild>
            <button
              type="button"
              aria-label={t('media.speed')}
              className="absolute right-2 top-2 z-10 flex items-center gap-1 rounded-full bg-black/55 px-2 py-1 text-xs font-semibold text-white backdrop-blur transition hover:bg-black/75"
            >
              <Gauge className="h-3.5 w-3.5" />
              {speed}×
            </button>
          </PopoverTrigger>
          <PopoverContent align="end" className="w-28 p-1">
            {SPEEDS.map((s) => (
              <button
                key={s}
                type="button"
                onClick={() => setSpeed(s)}
                className={cn(
                  'flex w-full items-center justify-between rounded-md px-3 py-1.5 text-sm transition-colors hover:bg-accent',
                  s === speed && 'font-bold text-primary',
                )}
              >
                {s}×{s === 1 && <span className="text-xs text-muted-foreground">{t('media.speed_normal')}</span>}
              </button>
            ))}
          </PopoverContent>
        </Popover>
      )}
    </div>
  )
}
