import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { nativeShare, copyLink, getRecentShareTargets, recordRecentShareTarget } from '@/lib/share'

let store: Record<string, string>
beforeEach(() => {
  store = {}
  vi.stubGlobal('window', {
    localStorage: {
      getItem: (k: string) => store[k] ?? null,
      setItem: (k: string, v: string) => { store[k] = v },
    },
  })
})
afterEach(() => vi.unstubAllGlobals())

const target = { id: 'u1', username: 'al', displayName: 'Al', avatarUrl: '' }

describe('nativeShare', () => {
  it('API absente → false', async () => {
    vi.stubGlobal('navigator', {})
    expect(await nativeShare({ url: 'x' })).toBe(false)
  })
  it('partage réussi → true', async () => {
    const share = vi.fn(async () => {})
    vi.stubGlobal('navigator', { share })
    expect(await nativeShare({ url: 'x', title: 't' })).toBe(true)
    expect(share).toHaveBeenCalledWith({ url: 'x', title: 't' })
  })
  it('annulation / erreur → false', async () => {
    vi.stubGlobal('navigator', { share: vi.fn(async () => { throw new Error('AbortError') }) })
    expect(await nativeShare({ url: 'x' })).toBe(false)
  })
})

describe('copyLink', () => {
  it('clipboard dispo → true', async () => {
    const writeText = vi.fn(async () => {})
    vi.stubGlobal('navigator', { clipboard: { writeText } })
    expect(await copyLink('hello')).toBe(true)
    expect(writeText).toHaveBeenCalledWith('hello')
  })
  it('clipboard absent → false', async () => {
    vi.stubGlobal('navigator', {})
    expect(await copyLink('hello')).toBe(false)
  })
  it('writeText échoue → false', async () => {
    vi.stubGlobal('navigator', { clipboard: { writeText: vi.fn(async () => { throw new Error('denied') }) } })
    expect(await copyLink('hello')).toBe(false)
  })
})

describe('destinataires récents', () => {
  it('liste vide par défaut', () => {
    expect(getRecentShareTargets()).toEqual([])
  })
  it('record met en tête, déduplique et plafonne', () => {
    recordRecentShareTarget(target)
    recordRecentShareTarget({ ...target, id: 'u2', username: 'bob' })
    recordRecentShareTarget(target) // remonte u1 en tête
    const list = getRecentShareTargets()
    expect(list.map((u) => u.id)).toEqual(['u1', 'u2'])
  })
  it('ignore les entrées corrompues à la lecture', () => {
    store['breezy.share.recent'] = JSON.stringify([{ id: 'ok' }, null, {}])
    expect(getRecentShareTargets().map((u) => u.id)).toEqual(['ok'])
  })
  it('JSON cassé → []', () => {
    store['breezy.share.recent'] = '{bad'
    expect(getRecentShareTargets()).toEqual([])
  })
})
