import { describe, expect, it } from 'vitest'

import { buildCountryOptions, countryFlag, filterCountries } from '@/lib/countries'

describe('countries', () => {
  it('localise, trie et ignore les codes invalides', () => {
    const options = buildCountryOptions(['US', 'FR', 'INVALID'], 'fr')
    expect(options.map(({ code }) => code)).toEqual(['US', 'FR'])
  })

  it('recherche sans tenir compte des accents et accepte le code ISO', () => {
    const options = buildCountryOptions(['US', 'FR'], 'fr')
    expect(filterCountries(options, 'etats').map(({ code }) => code)).toEqual(['US'])
    expect(filterCountries(options, 'fr').map(({ code }) => code)).toEqual(['FR'])
  })

  it('convertit un code ISO alpha-2 en drapeau', () => {
    expect(countryFlag('FR')).toBe('🇫🇷')
    expect(countryFlag('invalid')).toBe('')
  })
})
