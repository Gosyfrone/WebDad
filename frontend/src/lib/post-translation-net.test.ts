import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { getPageLanguage, translatePostContent } from '@/lib/post-translation'

const fetchMock = vi.fn()
let lang: string
let navLang: string

beforeEach(() => {
  lang = ''
  navLang = 'en-US'
  vi.stubGlobal('window', {
    navigator: { get language() { return navLang } },
    setTimeout: () => 0,
    clearTimeout: () => {},
  })
  vi.stubGlobal('document', { documentElement: { get lang() { return lang } } })
  vi.stubGlobal('fetch', fetchMock)
  fetchMock.mockReset()
})
afterEach(() => vi.unstubAllGlobals())

const okJson = (body: unknown) => ({ ok: true, json: () => Promise.resolve(body) })
// texte clairement français → candidat à la traduction vers l'anglais
const FR = 'bonjour les gars comment ca va'

describe('getPageLanguage', () => {
  it('privilégie l’attribut lang du <html>', () => {
    lang = 'es'
    expect(getPageLanguage()).toBe('es')
  })
  it('retombe sur la langue du navigateur', () => {
    lang = ''
    navLang = 'de-DE'
    expect(getPageLanguage()).toBe('de')
  })
  it('défaut fr si rien', () => {
    lang = ''
    navLang = ''
    expect(getPageLanguage()).toBe('fr')
  })
})

describe('translatePostContent', () => {
  it('texte non candidat → null sans appel réseau', async () => {
    expect(await translatePostContent('p1', 'hi', 'en')).toBeNull()
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('candidat → POST /api/translate et renvoie la traduction', async () => {
    fetchMock.mockResolvedValue(okJson({ translatedText: 'hello guys how are you', detectedSourceLanguage: 'fr' }))
    const r = await translatePostContent('p2', FR, 'en')
    expect(r).toEqual({ translatedText: 'hello guys how are you', detectedSourceLanguage: 'fr' })
    expect(fetchMock.mock.calls[0][0]).toBe('/api/translate')
  })

  it('mémoïse : 2e appel identique → 1 seul fetch', async () => {
    fetchMock.mockResolvedValue(okJson({ translatedText: 'hello guys', detectedSourceLanguage: 'fr' }))
    await translatePostContent('p3', FR, 'en')
    await translatePostContent('p3', FR, 'en')
    expect(fetchMock).toHaveBeenCalledTimes(1)
  })

  it('réponse non-ok → null (et retire l’entrée du cache)', async () => {
    fetchMock.mockResolvedValue({ ok: false, json: () => Promise.resolve(null) })
    expect(await translatePostContent('p4', FR, 'en')).toBeNull()
  })

  it('traduction identique au texte → null', async () => {
    fetchMock.mockResolvedValue(okJson({ translatedText: FR, detectedSourceLanguage: 'fr' }))
    expect(await translatePostContent('p5', FR, 'en')).toBeNull()
  })

  it('source détectée = cible → null', async () => {
    fetchMock.mockResolvedValue(okJson({ translatedText: 'hello guys', detectedSourceLanguage: 'en' }))
    expect(await translatePostContent('p6', FR, 'en')).toBeNull()
  })

  it('erreur réseau (abort/throw) → null', async () => {
    fetchMock.mockRejectedValue(new Error('aborted'))
    expect(await translatePostContent('p7', FR, 'en')).toBeNull()
  })
})
