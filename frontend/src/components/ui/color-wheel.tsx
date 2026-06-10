'use client'

import { useCallback, useEffect, useRef, useState } from 'react'

import { cn } from '@/lib/utils'
import {
  clamp,
  hexToRgb,
  hsvToRgb,
  rgbToHex,
  rgbToHsv,
  type Hsv,
} from '@/lib/color'
import { useT } from '@/components/language-provider'
import { Input } from '@/components/ui/input'

/**
 * Sélecteur de couleur sans dépendance externe :
 *   - une ROUE HSV (teinte = angle, saturation = rayon) avec curseur déplaçable ;
 *   - un curseur de LUMINOSITÉ (value) ;
 *   - un champ HEX et trois champs R / G / B synchronisés.
 *
 * Composant contrôlé : `value` (hex) entre, `onChange(hex)` sort. L'état interne
 * est gardé en HSV pour éviter les sauts du curseur quand S=0 ou V=0 (cas où la
 * teinte serait indéterminée par un aller-retour hex->hsv). On ne resynchronise
 * depuis `value` que lorsque la couleur reçue diffère réellement de la nôtre
 * (ex. reset externe), pas à chaque frappe.
 *
 * Géométrie de la roue : conic-gradient `from 0deg` (rouge en haut, sens horaire)
 * ; le placement du curseur utilise la même convention angulaire
 * (hue = atan2(dx, -dy)), garantissant que la couleur sous le curseur correspond.
 */

interface ColorWheelProps {
  /** Couleur courante (hex « #rrggbb »). */
  value: string
  onChange: (hex: string) => void
  className?: string
}

const WHEEL_SIZE = 200 // px (diamètre)
const KNOB = 18 // px (diamètre du curseur)

export function ColorWheel({ value, onChange, className }: ColorWheelProps) {
  const t = useT()
  const wheelRef = useRef<HTMLDivElement>(null)

  // Source de vérité interne, en HSV.
  const [hsv, setHsv] = useState<Hsv>(() => {
    const rgb = hexToRgb(value)
    return rgb ? rgbToHsv(rgb) : { h: 0, s: 0, v: 100 }
  })

  const currentHex = rgbToHex(hsvToRgb(hsv))

  // Resynchronise si la couleur externe change réellement (reset, autre cible…).
  useEffect(() => {
    if (value.toLowerCase() === currentHex.toLowerCase()) return
    const rgb = hexToRgb(value)
    if (rgb) setHsv(rgbToHsv(rgb))
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [value])

  /** Met à jour l'état HSV et notifie le parent (hex). */
  const update = useCallback(
    (next: Hsv) => {
      setHsv(next)
      onChange(rgbToHex(hsvToRgb(next)))
    },
    [onChange],
  )

  /** Convertit un point client (px) en teinte/saturation sur la roue. */
  const pointToHueSat = useCallback((clientX: number, clientY: number) => {
    const el = wheelRef.current
    if (!el) return null
    const rect = el.getBoundingClientRect()
    const cx = rect.left + rect.width / 2
    const cy = rect.top + rect.height / 2
    const dx = clientX - cx
    const dy = clientY - cy
    const radius = rect.width / 2
    const dist = Math.sqrt(dx * dx + dy * dy)
    // hue = angle horaire depuis le haut (cohérent avec conic-gradient from 0deg)
    const hue = ((Math.atan2(dx, -dy) * 180) / Math.PI + 360) % 360
    const sat = clamp((dist / radius) * 100, 0, 100)
    return { hue, sat }
  }, [])

  // Glisser-déposer sur la roue (souris + tactile via Pointer Events).
  const draggingRef = useRef(false)

  const onWheelPointerDown = useCallback(
    (e: React.PointerEvent<HTMLDivElement>) => {
      draggingRef.current = true
      e.currentTarget.setPointerCapture(e.pointerId)
      const hs = pointToHueSat(e.clientX, e.clientY)
      if (hs) update({ ...hsv, h: hs.hue, s: hs.sat })
    },
    [hsv, pointToHueSat, update],
  )

  const onWheelPointerMove = useCallback(
    (e: React.PointerEvent<HTMLDivElement>) => {
      if (!draggingRef.current) return
      const hs = pointToHueSat(e.clientX, e.clientY)
      if (hs) update({ ...hsv, h: hs.hue, s: hs.sat })
    },
    [hsv, pointToHueSat, update],
  )

  const stopDragging = useCallback((e: React.PointerEvent<HTMLDivElement>) => {
    draggingRef.current = false
    if (e.currentTarget.hasPointerCapture(e.pointerId)) {
      e.currentTarget.releasePointerCapture(e.pointerId)
    }
  }, [])

  // Position du curseur sur la roue (px depuis le coin haut-gauche).
  const radius = WHEEL_SIZE / 2
  const r = (hsv.s / 100) * radius
  const angle = (hsv.h * Math.PI) / 180
  const knobX = radius + r * Math.sin(angle)
  const knobY = radius - r * Math.cos(angle)

  /* ----------------------------------------------------------- Champs HEX / RGB */

  const rgb = hsvToRgb(hsv)
  const [hexDraft, setHexDraft] = useState(currentHex)

  // Garde le brouillon HEX aligné quand la couleur change par un autre moyen.
  useEffect(() => setHexDraft(currentHex), [currentHex])

  function commitHex(raw: string) {
    const parsed = hexToRgb(raw)
    if (parsed) update(rgbToHsv(parsed))
  }

  function setChannel(channel: 'r' | 'g' | 'b', raw: string) {
    const n = clamp(parseInt(raw || '0', 10) || 0, 0, 255)
    update(rgbToHsv({ ...rgb, [channel]: n }))
  }

  return (
    <div className={cn('flex flex-col items-center gap-4', className)}>
      {/* Roue HSV */}
      {/* Affordance pointeur (souris/tactile) ; l'accessibilité clavier passe
          par les champs Luminosité / HEX / RGB ci-dessous. */}
      <div
        ref={wheelRef}
        role="img"
        aria-label={`${t('theme.color_wheel_aria')} — ${currentHex}`}
        onPointerDown={onWheelPointerDown}
        onPointerMove={onWheelPointerMove}
        onPointerUp={stopDragging}
        onPointerCancel={stopDragging}
        className="relative cursor-crosshair touch-none rounded-full shadow-inner"
        style={{
          width: WHEEL_SIZE,
          height: WHEEL_SIZE,
          background:
            'radial-gradient(circle at center, #fff 0%, rgba(255,255,255,0) 100%), ' +
            'conic-gradient(from 0deg, ' +
            'hsl(0 100% 50%), hsl(30 100% 50%), hsl(60 100% 50%), hsl(90 100% 50%), ' +
            'hsl(120 100% 50%), hsl(150 100% 50%), hsl(180 100% 50%), hsl(210 100% 50%), ' +
            'hsl(240 100% 50%), hsl(270 100% 50%), hsl(300 100% 50%), hsl(330 100% 50%), hsl(360 100% 50%))',
        }}
      >
        {/* Voile noir = luminosité (value) plus basse */}
        <div
          className="pointer-events-none absolute inset-0 rounded-full bg-black"
          style={{ opacity: 1 - hsv.v / 100 }}
        />
        {/* Curseur */}
        <div
          className="pointer-events-none absolute rounded-full border-2 border-white shadow-[0_0_0_1px_rgba(0,0,0,0.4)]"
          style={{
            width: KNOB,
            height: KNOB,
            left: knobX - KNOB / 2,
            top: knobY - KNOB / 2,
            backgroundColor: currentHex,
          }}
        />
      </div>

      {/* Curseur de luminosité (value) */}
      <label className="flex w-full items-center gap-3">
        <span className="w-20 shrink-0 text-xs font-medium text-muted-foreground">
          {t('theme.brightness')}
        </span>
        <input
          type="range"
          min={0}
          max={100}
          value={Math.round(hsv.v)}
          onChange={(e) => update({ ...hsv, v: Number(e.target.value) })}
          aria-label={t('theme.brightness')}
          className="h-2 w-full cursor-pointer appearance-none rounded-full"
          style={{
            background: `linear-gradient(to right, #000, ${rgbToHex(
              hsvToRgb({ ...hsv, v: 100 }),
            )})`,
          }}
        />
      </label>

      {/* Aperçu + HEX */}
      <div className="flex w-full items-center gap-2">
        <span
          className="h-10 w-10 shrink-0 rounded-md border"
          style={{ backgroundColor: currentHex }}
          aria-hidden
        />
        <label className="flex flex-1 flex-col gap-1">
          <span className="text-xs font-medium text-muted-foreground">{t('theme.hex')}</span>
          <Input
            value={hexDraft}
            onChange={(e) => {
              setHexDraft(e.target.value)
              commitHex(e.target.value)
            }}
            spellCheck={false}
            aria-label={t('theme.hex')}
            className="h-9 font-mono uppercase"
          />
        </label>
      </div>

      {/* Champs R / G / B */}
      <div className="grid w-full grid-cols-3 gap-2">
        {(['r', 'g', 'b'] as const).map((ch) => (
          <label key={ch} className="flex flex-col gap-1">
            <span className="text-xs font-medium uppercase text-muted-foreground">{ch}</span>
            <Input
              type="number"
              min={0}
              max={255}
              value={rgb[ch]}
              onChange={(e) => setChannel(ch, e.target.value)}
              aria-label={`${t('theme.hex')} ${ch.toUpperCase()}`}
              className="h-9"
            />
          </label>
        ))}
      </div>
    </div>
  )
}
