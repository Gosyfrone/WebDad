'use client'

import { useCallback, useEffect, useLayoutEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { usePathname, useRouter } from 'next/navigation'
import { Lightbulb } from 'lucide-react'

import {
  TUTORIAL_STEPS,
  TUTORIAL_STEP_COUNT,
  computeTooltipPosition,
  stepBodyKey,
  stepTitleKey,
  type Rect,
  type TooltipPosition,
} from '@/lib/tutorial-steps'
import { Button } from '@/components/ui/button'
import { useT } from '@/components/language-provider'
import { useTutorial } from '@/components/tutorial/tutorial-provider'

/** Marge (px) ajoutée autour de la cible pour aérer la surbrillance. */
const HIGHLIGHT_PADDING = 6

/** Renvoie l'instance VISIBLE d'une cible `data-tour` (desktop vs mobile). */
function findVisibleTarget(name: string): HTMLElement | null {
  const nodes = Array.from(
    document.querySelectorAll<HTMLElement>(`[data-tour="${name}"]`),
  )
  return nodes.find((el) => el.offsetParent !== null) ?? nodes[0] ?? null
}

/**
 * Rendu visuel du didacticiel : backdrop assombri + spotlight (découpe) autour de
 * la cible courante, et carte d'infobulle positionnée à côté (titre, texte,
 * compteur, Précédent / Suivant / Ignorer). Cliquer la cible = avancer.
 *
 * Le composant pilote la navigation : à chaque étape, on rejoint `step.route` si
 * besoin, puis on localise la cible (avec re-tentatives car le rendu de page est
 * asynchrone). Styling clair/sombre via les tokens du thème.
 */
export function TutorialTooltip() {
  const t = useT()
  const router = useRouter()
  const pathname = usePathname()
  const { isActive, stepIndex, next, prev, stop } = useTutorial()

  const [mounted, setMounted] = useState(false)
  const [rect, setRect] = useState<Rect | null>(null)
  const [position, setPosition] = useState<TooltipPosition | null>(null)
  const cardRef = useRef<HTMLDivElement>(null)
  const targetRef = useRef<HTMLElement | null>(null)

  const step = TUTORIAL_STEPS[stepIndex]
  const isLast = stepIndex >= TUTORIAL_STEP_COUNT - 1
  const isFirst = stepIndex === 0

  useEffect(() => setMounted(true), [])

  // Navigation vers la page de l'étape (le tour fait défiler les sections).
  useEffect(() => {
    if (!isActive || !step) return
    if (step.route && pathname !== step.route) router.push(step.route)
  }, [isActive, step, pathname, router])

  // Localisation de la cible : re-tentatives courtes (rAF) car la page peut ne
  // pas être encore montée juste après la navigation. Étape sans cible ⇒ centré.
  useEffect(() => {
    if (!isActive || !step) return
    targetRef.current = null
    setRect(null)

    if (!step.target) return

    let raf = 0
    const deadline = Date.now() + 2000
    const tryLocate = () => {
      const el = findVisibleTarget(step.target!)
      if (el) {
        targetRef.current = el
        el.scrollIntoView({ block: 'center', inline: 'center', behavior: 'smooth' })
        measure()
        return
      }
      if (Date.now() < deadline) raf = requestAnimationFrame(tryLocate)
    }
    raf = requestAnimationFrame(tryLocate)
    return () => cancelAnimationFrame(raf)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isActive, stepIndex, pathname])

  // Mesure cible + carte → position. Mémoïsé pour le rattacher aux events.
  const measure = useCallback(() => {
    const el = targetRef.current
    if (el) {
      const r = el.getBoundingClientRect()
      setRect({
        top: r.top - HIGHLIGHT_PADDING,
        left: r.left - HIGHLIGHT_PADDING,
        width: r.width + HIGHLIGHT_PADDING * 2,
        height: r.height + HIGHLIGHT_PADDING * 2,
      })
    }
  }, [])

  // Recalage au scroll / resize tant qu'une cible est suivie.
  useEffect(() => {
    if (!isActive) return
    const onMove = () => measure()
    window.addEventListener('scroll', onMove, true)
    window.addEventListener('resize', onMove)
    return () => {
      window.removeEventListener('scroll', onMove, true)
      window.removeEventListener('resize', onMove)
    }
  }, [isActive, measure])

  // Position de la carte une fois sa taille connue (et celle de la cible/écran).
  useLayoutEffect(() => {
    if (!isActive || !step) return
    const card = cardRef.current
    if (!card) return
    const size = { width: card.offsetWidth, height: card.offsetHeight }
    const viewport = { width: window.innerWidth, height: window.innerHeight }
    if (rect) {
      setPosition(computeTooltipPosition(rect, size, viewport, step.placement))
    } else {
      // Étape centrée (pas de cible) : carte au centre de l'écran.
      setPosition({
        top: viewport.height / 2 - size.height / 2,
        left: viewport.width / 2 - size.width / 2,
        placement: 'bottom',
      })
    }
  }, [isActive, step, rect])

  // Cliquer la cible mise en surbrillance = avancer (on neutralise l'action
  // native de l'élément : la navigation est pilotée par le tour, étape par étape).
  useEffect(() => {
    if (!isActive) return
    const el = targetRef.current
    if (!el || !rect) return
    const onClick = (e: Event) => {
      e.preventDefault()
      e.stopPropagation()
      next()
    }
    el.addEventListener('click', onClick, true)
    return () => el.removeEventListener('click', onClick, true)
  }, [isActive, rect, next, stepIndex])

  if (!mounted || !isActive || !step) return null

  const hidden = position === null
  const title = t(stepTitleKey(step))
  const body = t(stepBodyKey(step))

  return createPortal(
    // `pointer-events-none` sur l'enveloppe : indispensable pour que le clic
    // traverse jusqu'à la cible (sinon l'overlay plein écran capterait tout et
    // « cliquer la cible pour avancer » ne marcherait pas). Seuls la carte et le
    // voile plein écran (étape centrée, sans cible) réactivent les events.
    <div
      className="pointer-events-none fixed inset-0 z-[120]"
      role="dialog"
      aria-modal="true"
      aria-label={title}
    >
      {/* Spotlight : un cadre transparent autour de la cible projette une ombre
          immense qui assombrit tout le reste. pointer-events-none → le clic
          atteint la cible (intercepté par le listener ci-dessus). Sans cible :
          voile plein écran cliquable (rien à atteindre dessous). */}
      {rect ? (
        <div
          className="pointer-events-none absolute rounded-xl ring-2 ring-[var(--brand-via,#7C5CFF)] transition-all duration-200"
          style={{
            top: rect.top,
            left: rect.left,
            width: rect.width,
            height: rect.height,
            boxShadow: '0 0 0 9999px rgba(0,0,0,0.62)',
          }}
        />
      ) : (
        <div className="pointer-events-auto absolute inset-0 bg-black/60" />
      )}

      {/* Carte d'infobulle */}
      <div
        ref={cardRef}
        className="pointer-events-auto absolute w-[min(20rem,calc(100vw-1rem))] rounded-2xl border bg-background p-4 text-foreground shadow-xl"
        style={{
          top: position?.top ?? 0,
          left: position?.left ?? 0,
          visibility: hidden ? 'hidden' : 'visible',
        }}
      >
        <div className="mb-2 flex items-center gap-2">
          <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-gradient-to-r from-[var(--brand-from)] via-[var(--brand-via)] to-[var(--brand-to)] text-white">
            <Lightbulb className="h-4 w-4" aria-hidden />
          </span>
          <h2 className="text-sm font-bold">{title}</h2>
          <span className="ml-auto text-xs font-medium text-muted-foreground">
            {t('tutorial.step_counter', { current: stepIndex + 1, total: TUTORIAL_STEP_COUNT })}
          </span>
        </div>

        <p className="mb-4 text-sm leading-snug text-muted-foreground">{body}</p>

        <div className="flex items-center justify-between gap-2">
          <button
            type="button"
            onClick={stop}
            className="text-xs font-medium text-muted-foreground underline-offset-2 transition-colors hover:text-foreground hover:underline"
          >
            {t('tutorial.skip')}
          </button>
          <div className="flex items-center gap-2">
            {!isFirst && (
              <Button type="button" variant="outline" size="sm" className="rounded-full" onClick={prev}>
                {t('tutorial.prev')}
              </Button>
            )}
            <Button
              type="button"
              size="sm"
              onClick={next}
              className="rounded-full bg-gradient-to-r from-[var(--brand-from)] via-[var(--brand-via)] to-[var(--brand-to)] font-bold text-white"
            >
              {isLast ? t('tutorial.finish') : t('tutorial.next')}
            </Button>
          </div>
        </div>
      </div>
    </div>,
    document.body,
  )
}
