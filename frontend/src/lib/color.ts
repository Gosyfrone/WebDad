/**
 * Conversions de couleurs (fonctions pures, testables).
 *
 * Sert le sélecteur de thème personnalisé (roue HSV) et l'application des
 * couleurs choisies aux variables CSS de Tailwind/shadcn.
 *
 * Deux modèles :
 *   - HSV : naturel pour la ROUE (angle = teinte, rayon = saturation) + un
 *           curseur de luminosité (value). Voir color-wheel.tsx.
 *   - HSL : format attendu par les variables CSS shadcn, stockées en triplet
 *           « H S% L% » (sans wrapper hsl(), Tailwind l'enveloppe lui-même via
 *           `hsl(var(--x))`). Voir custom-theme.ts.
 *
 * Toutes les fonctions sont pures : aucune dépendance au DOM.
 */

export interface Rgb {
  r: number // 0-255
  g: number
  b: number
}

export interface Hsl {
  h: number // 0-360
  s: number // 0-100
  l: number // 0-100
}

export interface Hsv {
  h: number // 0-360
  s: number // 0-100
  v: number // 0-100
}

/** Borne une valeur dans [min, max]. */
export function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value))
}

/* ------------------------------------------------------------------ HEX <-> RGB */

/**
 * Parse un code hexadécimal (#RGB, #RRGGBB, avec ou sans #) en RGB.
 * Renvoie `null` si la chaîne n'est pas un hex valide.
 */
export function hexToRgb(hex: string): Rgb | null {
  let h = hex.trim().replace(/^#/, '')
  if (h.length === 3) {
    h = h
      .split('')
      .map((c) => c + c)
      .join('')
  }
  if (!/^[0-9a-fA-F]{6}$/.test(h)) return null
  return {
    r: parseInt(h.slice(0, 2), 16),
    g: parseInt(h.slice(2, 4), 16),
    b: parseInt(h.slice(4, 6), 16),
  }
}

/** Formate un composant 0-255 en deux chiffres hex. */
function toHex2(n: number): string {
  return clamp(Math.round(n), 0, 255).toString(16).padStart(2, '0')
}

/** RGB -> code hexadécimal « #rrggbb » (minuscule). */
export function rgbToHex({ r, g, b }: Rgb): string {
  return `#${toHex2(r)}${toHex2(g)}${toHex2(b)}`
}

/* ------------------------------------------------------------------ RGB <-> HSL */

/** RGB (0-255) -> HSL (h 0-360, s/l 0-100). */
export function rgbToHsl({ r, g, b }: Rgb): Hsl {
  const rn = r / 255
  const gn = g / 255
  const bn = b / 255
  const max = Math.max(rn, gn, bn)
  const min = Math.min(rn, gn, bn)
  const d = max - min
  const l = (max + min) / 2

  let h = 0
  let s = 0
  if (d !== 0) {
    s = d / (1 - Math.abs(2 * l - 1))
    switch (max) {
      case rn:
        h = ((gn - bn) / d) % 6
        break
      case gn:
        h = (bn - rn) / d + 2
        break
      default:
        h = (rn - gn) / d + 4
    }
    h *= 60
    if (h < 0) h += 360
  }
  return { h, s: s * 100, l: l * 100 }
}

/** HSL (h 0-360, s/l 0-100) -> RGB (0-255). */
export function hslToRgb({ h, s, l }: Hsl): Rgb {
  const sn = s / 100
  const ln = l / 100
  const c = (1 - Math.abs(2 * ln - 1)) * sn
  const x = c * (1 - Math.abs(((h / 60) % 2) - 1))
  const m = ln - c / 2
  const [r, g, b] = sector(h, c, x)
  return {
    r: Math.round((r + m) * 255),
    g: Math.round((g + m) * 255),
    b: Math.round((b + m) * 255),
  }
}

/* ------------------------------------------------------------------ RGB <-> HSV */

/** RGB (0-255) -> HSV (h 0-360, s/v 0-100). */
export function rgbToHsv({ r, g, b }: Rgb): Hsv {
  const rn = r / 255
  const gn = g / 255
  const bn = b / 255
  const max = Math.max(rn, gn, bn)
  const min = Math.min(rn, gn, bn)
  const d = max - min

  let h = 0
  if (d !== 0) {
    switch (max) {
      case rn:
        h = ((gn - bn) / d) % 6
        break
      case gn:
        h = (bn - rn) / d + 2
        break
      default:
        h = (rn - gn) / d + 4
    }
    h *= 60
    if (h < 0) h += 360
  }
  const s = max === 0 ? 0 : d / max
  return { h, s: s * 100, v: max * 100 }
}

/** HSV (h 0-360, s/v 0-100) -> RGB (0-255). */
export function hsvToRgb({ h, s, v }: Hsv): Rgb {
  const sn = s / 100
  const vn = v / 100
  const c = vn * sn
  const x = c * (1 - Math.abs(((h / 60) % 2) - 1))
  const m = vn - c
  const [r, g, b] = sector(h, c, x)
  return {
    r: Math.round((r + m) * 255),
    g: Math.round((g + m) * 255),
    b: Math.round((b + m) * 255),
  }
}

/** Sélection du sextant de teinte (commun à HSL/HSV). */
function sector(h: number, c: number, x: number): [number, number, number] {
  const hh = ((h % 360) + 360) % 360
  if (hh < 60) return [c, x, 0]
  if (hh < 120) return [x, c, 0]
  if (hh < 180) return [0, c, x]
  if (hh < 240) return [0, x, c]
  if (hh < 300) return [x, 0, c]
  return [c, 0, x]
}

/* ------------------------------------------------------------------ Helpers CSS */

/**
 * Hex -> triplet CSS « H S% L% » (arrondi), tel qu'attendu par les variables
 * shadcn (`hsl(var(--x))`). Renvoie `null` si l'hex est invalide.
 */
export function hexToHslTriplet(hex: string): string | null {
  const rgb = hexToRgb(hex)
  if (!rgb) return null
  const { h, s, l } = rgbToHsl(rgb)
  return `${Math.round(h)} ${Math.round(s)}% ${Math.round(l)}%`
}

/**
 * Parse un triplet CSS « H S% L% » -> hex. Utilisé pour relire la couleur
 * courante d'une variable CSS (getComputedStyle) et la pré-remplir dans l'éditeur.
 * Renvoie `null` si le format n'est pas reconnu.
 */
export function hslTripletToHex(triplet: string): string | null {
  const m = triplet.trim().match(/^([\d.]+)\s+([\d.]+)%\s+([\d.]+)%$/)
  if (!m) return null
  const h = parseFloat(m[1])
  const s = parseFloat(m[2])
  const l = parseFloat(m[3])
  return rgbToHex(hslToRgb({ h, s, l }))
}

/**
 * Luminance relative perçue (sRGB, formule WCAG) d'une couleur hex, dans [0,1].
 * Sert à choisir une couleur de texte lisible sur un fond donné.
 */
export function relativeLuminance(hex: string): number {
  const rgb = hexToRgb(hex)
  if (!rgb) return 0
  const lin = (c: number) => {
    const cs = c / 255
    return cs <= 0.03928 ? cs / 12.92 : Math.pow((cs + 0.055) / 1.055, 2.4)
  }
  return 0.2126 * lin(rgb.r) + 0.7152 * lin(rgb.g) + 0.0722 * lin(rgb.b)
}

/**
 * Renvoie le triplet HSL d'une couleur de texte contrastée (blanc ou quasi-noir)
 * à poser sur le fond `hex`. Seuil ~0.4 : au-delà le fond est clair -> texte foncé.
 */
export function contrastingTextTriplet(hex: string): string {
  // Quasi-noir = --foreground clair de Breezy ; blanc pur sinon.
  return relativeLuminance(hex) > 0.4 ? '224 47% 11%' : '0 0% 100%'
}
