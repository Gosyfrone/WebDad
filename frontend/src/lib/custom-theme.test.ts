import { describe, expect, it } from 'vitest'

import { buildCustomThemeVars, EMPTY_CUSTOM_THEME } from '@/lib/custom-theme'

describe('buildCustomThemeVars', () => {
  it('n\'émet aucune variable pour un thème vide', () => {
    expect(buildCustomThemeVars(EMPTY_CUSTOM_THEME)).toEqual({})
  })

  it('mappe le fond sur --background + --bg-page (solide), neutralise les voiles et repeint les surfaces', () => {
    const vars = buildCustomThemeVars({ ...EMPTY_CUSTOM_THEME, background: '#123456' })
    expect(vars['--bg-page']).toBe('#123456')
    expect(vars['--background']).toMatch(/^\d+ \d+% \d+%$/)
    expect(vars['--bg-glow-1']).toBe('none')
    expect(vars['--bg-glow-2']).toBe('none')
    // Surfaces de lecture (colonne centrale, header feed, cartes) suivent le fond.
    expect(vars['--column']).toBe('#123456')
    expect(vars['--glass']).toBe('#123456')
    expect(vars['--glass-strong']).toBe('#123456')
    // --panel(-y) sont en background-image → dégradé uni, pas un hex brut.
    expect(vars['--panel']).toBe('linear-gradient(#123456, #123456)')
    expect(vars['--panel-y']).toBe('linear-gradient(#123456, #123456)')
  })

  it('mappe le texte sur les trois variables de premier plan', () => {
    const vars = buildCustomThemeVars({ ...EMPTY_CUSTOM_THEME, text: '#ffffff' })
    expect(vars['--foreground']).toBe('0 0% 100%')
    expect(vars['--card-foreground']).toBe('0 0% 100%')
    expect(vars['--popover-foreground']).toBe('0 0% 100%')
  })

  it('mappe le primaire sur --primary/--ring + texte auto-contrasté + dégradé de marque', () => {
    const light = buildCustomThemeVars({ ...EMPTY_CUSTOM_THEME, primary: '#ffe14d' })
    expect(light['--primary']).toMatch(/^\d+ \d+% \d+%$/)
    expect(light['--ring']).toBe(light['--primary'])
    expect(light['--primary-foreground']).toBe('224 47% 11%') // texte foncé sur jaune clair
    // Les boutons d'action (dégradé de marque) se replient sur la couleur unie.
    expect(light['--brand-from']).toBe('#ffe14d')
    expect(light['--brand-via']).toBe('#ffe14d')
    expect(light['--brand-to']).toBe('#ffe14d')

    const dark = buildCustomThemeVars({ ...EMPTY_CUSTOM_THEME, primary: '#3b0a8a' })
    expect(dark['--primary-foreground']).toBe('0 0% 100%') // texte blanc sur violet foncé
  })

  it('ignore une couleur hex invalide', () => {
    expect(buildCustomThemeVars({ ...EMPTY_CUSTOM_THEME, primary: 'oops' })).toEqual({})
  })

  it('combine les trois cibles indépendamment', () => {
    const vars = buildCustomThemeVars({
      background: '#000000',
      text: '#ffffff',
      primary: '#e053ff',
    })
    expect(Object.keys(vars).sort()).toEqual(
      [
        '--background',
        '--bg-page',
        '--bg-glow-1',
        '--bg-glow-2',
        '--panel',
        '--panel-y',
        '--column',
        '--glass',
        '--glass-strong',
        '--foreground',
        '--card-foreground',
        '--popover-foreground',
        '--primary',
        '--ring',
        '--primary-foreground',
        '--brand-from',
        '--brand-via',
        '--brand-to',
      ].sort(),
    )
  })
})
