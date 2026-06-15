import type { Locale } from '@/lib/i18n'

export interface CountryOption {
  code: string
  name: string
}

export function countryName(code: string, locale: Locale): string {
  if (!code) return ''
  try {
    return new Intl.DisplayNames([locale], { type: 'region' }).of(code.toUpperCase()) ?? code
  } catch {
    return code
  }
}

export function countryFlag(code: string): string {
  const normalized = code.toUpperCase()
  if (!/^[A-Z]{2}$/.test(normalized)) return ''
  return String.fromCodePoint(...[...normalized].map((letter) => letter.charCodeAt(0) + 127397))
}

export function buildCountryOptions(codes: string[], locale: Locale): CountryOption[] {
  return codes
    .filter((code) => /^[A-Z]{2}$/.test(code))
    .map((code) => ({ code, name: countryName(code, locale) }))
    .sort((a, b) => a.name.localeCompare(b.name, locale, { sensitivity: 'base' }))
}

export function filterCountries(options: CountryOption[], query: string): CountryOption[] {
  const needle = normalize(query)
  if (!needle) return options
  return options.filter(
    ({ code, name }) => normalize(name).includes(needle) || code.toLowerCase().includes(needle),
  )
}

function normalize(value: string): string {
  return value
    .normalize('NFD')
    .replace(/[\u0300-\u036f]/g, '')
    .trim()
    .toLowerCase()
}
