import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import {
  readCustomTheme, hasCustomThemeValues, readCustomThemeEnabled, writeCustomTheme,
  setCustomThemeEnabled, applyCustomTheme, CUSTOM_THEME_STORAGE_KEY, EMPTY_CUSTOM_THEME,
} from '@/lib/custom-theme'

let store: Record<string, string>
let events: string[]
let styleProps: Record<string, string>
let removed: string[]

function setupDom() {
  store = {}
  events = []
  styleProps = {}
  removed = []
  vi.stubGlobal('window', {
    localStorage: {
      getItem: (k: string) => store[k] ?? null,
      setItem: (k: string, v: string) => { store[k] = v },
      removeItem: (k: string) => { delete store[k] },
    },
    dispatchEvent: (e: Event) => { events.push(e.type); return true },
  })
  vi.stubGlobal('Event', class { type: string; constructor(t: string) { this.type = t } })
  vi.stubGlobal('document', {
    documentElement: {
      style: {
        setProperty: (k: string, v: string) => { styleProps[k] = v },
        removeProperty: (k: string) => { removed.push(k); delete styleProps[k] },
      },
    },
  })
}

beforeEach(setupDom)
afterEach(() => vi.unstubAllGlobals())

const blue = '#0a0a0a'

describe('readCustomTheme', () => {
  it('vide si rien en storage', () => {
    expect(readCustomTheme()).toEqual(EMPTY_CUSTOM_THEME)
  })
  it('lit les couleurs valides, ignore les hex invalides', () => {
    store[CUSTOM_THEME_STORAGE_KEY] = JSON.stringify({ background: blue, text: 'nothex', primary: '#ffffff' })
    expect(readCustomTheme()).toEqual({ background: blue, text: null, primary: '#ffffff' })
  })
  it('JSON cassé → vide', () => {
    store[CUSTOM_THEME_STORAGE_KEY] = '{bad'
    expect(readCustomTheme()).toEqual(EMPTY_CUSTOM_THEME)
  })
})

describe('hasCustomThemeValues / readCustomThemeEnabled', () => {
  it('hasCustomThemeValues', () => {
    expect(hasCustomThemeValues({ background: null, text: null, primary: null })).toBe(false)
    expect(hasCustomThemeValues({ background: blue, text: null, primary: null })).toBe(true)
  })
  it('enabled : true si valeurs + flag absent, false si enabled:false, false si vide', () => {
    expect(readCustomThemeEnabled()).toBe(false)
    store[CUSTOM_THEME_STORAGE_KEY] = JSON.stringify({ background: blue })
    expect(readCustomThemeEnabled()).toBe(true)
    store[CUSTOM_THEME_STORAGE_KEY] = JSON.stringify({ background: blue, enabled: false })
    expect(readCustomThemeEnabled()).toBe(false)
  })
  it('readCustomThemeEnabled JSON cassé → false', () => {
    store[CUSTOM_THEME_STORAGE_KEY] = '{bad'
    expect(readCustomThemeEnabled()).toBe(false)
  })
})

describe('writeCustomTheme', () => {
  it('thème vide → supprime + évènement', () => {
    store[CUSTOM_THEME_STORAGE_KEY] = 'x'
    writeCustomTheme({ background: null, text: null, primary: null })
    expect(store[CUSTOM_THEME_STORAGE_KEY]).toBeUndefined()
    expect(events).toContain('breezy-custom-theme-change')
  })
  it('thème non vide → persiste couleurs + vars pré-calculées', () => {
    writeCustomTheme({ background: blue, text: null, primary: null })
    const stored = JSON.parse(store[CUSTOM_THEME_STORAGE_KEY])
    expect(stored.background).toBe(blue)
    expect(stored.enabled).toBe(true)
    expect(stored.vars['--bg-page']).toBe(blue)
  })
})

describe('setCustomThemeEnabled / applyCustomTheme', () => {
  it('sans couleurs enregistrées → no-op', () => {
    setCustomThemeEnabled(true)
    expect(store[CUSTOM_THEME_STORAGE_KEY]).toBeUndefined()
  })
  it('désactive → applique le thème vide (variables retirées)', () => {
    store[CUSTOM_THEME_STORAGE_KEY] = JSON.stringify({ background: blue })
    setCustomThemeEnabled(false)
    expect(JSON.parse(store[CUSTOM_THEME_STORAGE_KEY]).enabled).toBe(false)
    expect(removed).toContain('--bg-page')
    expect(styleProps['--bg-page']).toBeUndefined()
  })
  it('applyCustomTheme pose les variables calculées', () => {
    applyCustomTheme({ background: blue, text: null, primary: null })
    expect(styleProps['--bg-page']).toBe(blue)
  })
  it('applyCustomTheme sans document → no-op', () => {
    vi.stubGlobal('document', undefined)
    expect(() => applyCustomTheme({ background: blue, text: null, primary: null })).not.toThrow()
  })
})
