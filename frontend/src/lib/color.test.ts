import { describe, expect, it } from 'vitest'

import {
  contrastingTextTriplet,
  hexToHslTriplet,
  hexToRgb,
  hslTripletToHex,
  hslToRgb,
  hsvToRgb,
  rgbToHex,
  rgbToHsl,
  rgbToHsv,
} from '@/lib/color'

describe('hexToRgb', () => {
  it('parse #rrggbb', () => {
    expect(hexToRgb('#ff8800')).toEqual({ r: 255, g: 136, b: 0 })
  })
  it('parse la forme courte #rgb et sans #', () => {
    expect(hexToRgb('#f80')).toEqual({ r: 255, g: 136, b: 0 })
    expect(hexToRgb('00ff00')).toEqual({ r: 0, g: 255, b: 0 })
  })
  it('renvoie null sur une entrée invalide', () => {
    expect(hexToRgb('nope')).toBeNull()
    expect(hexToRgb('#12')).toBeNull()
  })
})

describe('rgbToHex', () => {
  it('formate et borne les composants', () => {
    expect(rgbToHex({ r: 255, g: 136, b: 0 })).toBe('#ff8800')
    expect(rgbToHex({ r: 300, g: -5, b: 0 })).toBe('#ff0000')
  })
})

describe('round-trips HSL / HSV', () => {
  const samples = ['#000000', '#ffffff', '#e053ff', '#4d7cf5', '#38bdf8', '#123456']
  it('hex -> hsl -> hex est stable', () => {
    for (const hex of samples) {
      const rgb = hexToRgb(hex)!
      const back = rgbToHex(hslToRgb(rgbToHsl(rgb)))
      expect(back).toBe(hex)
    }
  })
  it('hex -> hsv -> hex est stable', () => {
    for (const hex of samples) {
      const rgb = hexToRgb(hex)!
      const back = rgbToHex(hsvToRgb(rgbToHsv(rgb)))
      expect(back).toBe(hex)
    }
  })
})

describe('hexToHslTriplet / hslTripletToHex', () => {
  it('produit un triplet « H S% L% » arrondi', () => {
    expect(hexToHslTriplet('#ffffff')).toBe('0 0% 100%')
    expect(hexToHslTriplet('#000000')).toBe('0 0% 0%')
  })
  it('relit un triplet en hex', () => {
    expect(hslTripletToHex('0 0% 100%')).toBe('#ffffff')
    expect(hslTripletToHex('289 100% 66%')).toMatch(/^#[0-9a-f]{6}$/)
  })
  it('renvoie null sur format invalide', () => {
    expect(hexToHslTriplet('zzz')).toBeNull()
    expect(hslTripletToHex('pas un triplet')).toBeNull()
  })
})

describe('contrastingTextTriplet', () => {
  it('texte foncé sur fond clair, blanc sur fond foncé', () => {
    expect(contrastingTextTriplet('#ffffff')).toBe('224 47% 11%')
    expect(contrastingTextTriplet('#ffe14d')).toBe('224 47% 11%')
    expect(contrastingTextTriplet('#000000')).toBe('0 0% 100%')
    expect(contrastingTextTriplet('#1d1d6b')).toBe('0 0% 100%')
  })
})
