import { afterEach, describe, expect, it, vi } from 'vitest'

// `search-tab` garde une mémoire EN MODULE (`memoryPath`) : on réimporte à neuf
// par test pour isoler l'état.
let store: Record<string, string>
function stubSession(throwing = false) {
  store = {}
  vi.stubGlobal('sessionStorage', {
    getItem: (k: string) => store[k] ?? null,
    setItem: (k: string, v: string) => {
      if (throwing) throw new Error('private mode')
      store[k] = v
    },
  })
}
async function load() {
  vi.resetModules()
  return import('@/lib/search-tab')
}
afterEach(() => vi.unstubAllGlobals())

describe('search-tab (mémoire de la loupe)', () => {
  it('défaut /explorer quand rien en mémoire ni en session', async () => {
    stubSession()
    const m = await load()
    expect(m.getSearchPath()).toBe(m.SEARCH_TAB_DEFAULT)
  })

  it('rememberSearchPath persiste et getSearchPath le relit', async () => {
    stubSession()
    const m = await load()
    m.rememberSearchPath('/explorer?q=js')
    expect(m.getSearchPath()).toBe('/explorer?q=js')
    expect(store['breezy.searchTabPath']).toBe('/explorer?q=js')
  })

  it('relit depuis sessionStorage quand la mémoire module est vide', async () => {
    stubSession()
    store['breezy.searchTabPath'] = '/profil/bob'
    const m = await load()
    expect(m.getSearchPath()).toBe('/profil/bob')
  })

  it('sessionStorage indisponible (nav privée) : la mémoire module suffit', async () => {
    stubSession(true)
    const m = await load()
    m.rememberSearchPath('/explorer?q=go')
    expect(m.getSearchPath()).toBe('/explorer?q=go')
  })
})
