import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

// Cœur d'`apiFetch` : Bearer + refresh single-flight sur 401 + redirection.
// On réinitialise le module entre chaque test (état single-flight / bootstrap).
const fetchMock = vi.fn()
let store: Record<string, string>
let events: string[]
let assigned: string[]

function setupWindow(token: string | null, pathname = '/feed') {
  store = {}
  events = []
  assigned = []
  if (token !== null) store['breezy-access-token'] = token
  globalThis.window = {
    localStorage: {
      getItem: (k: string) => store[k] ?? null,
      setItem: (k: string, v: string) => { store[k] = v },
      removeItem: (k: string) => { delete store[k] },
    },
    dispatchEvent: (e: Event) => { events.push(e.type); return true },
    location: { pathname, assign: (u: string) => assigned.push(u) },
  } as unknown as Window & typeof globalThis
  vi.stubGlobal('fetch', fetchMock)
}

const resp = (status: number, body: unknown = {}) => ({ status, ok: status >= 200 && status < 300, json: () => Promise.resolve(body) })

async function load() {
  vi.resetModules()
  return import('@/lib/auth-client')
}

beforeEach(() => fetchMock.mockReset())
afterEach(() => { vi.unstubAllGlobals(); delete (globalThis as { window?: unknown }).window })

describe('apiFetch', () => {
  it('ajoute le Bearer et renvoie la réponse 200', async () => {
    setupWindow('tok')
    fetchMock.mockResolvedValue(resp(200, { data: 1 }))
    const { apiFetch } = await load()
    const r = await apiFetch('/posts')
    expect(r.status).toBe(200)
    const [url, init] = fetchMock.mock.calls[0]
    expect(String(url)).toContain('/posts')
    expect((init.headers as Headers).get('Authorization')).toBe('Bearer tok')
  })

  it('sans token → pas d’en-tête Authorization', async () => {
    setupWindow(null)
    fetchMock.mockResolvedValue(resp(200))
    const { apiFetch } = await load()
    await apiFetch('/posts')
    expect((fetchMock.mock.calls[0][1].headers as Headers).get('Authorization')).toBeNull()
  })

  it('URL absolue conservée telle quelle', async () => {
    setupWindow('tok')
    fetchMock.mockResolvedValue(resp(200))
    const { apiFetch } = await load()
    await apiFetch('http://other/x')
    expect(String(fetchMock.mock.calls[0][0])).toBe('http://other/x')
  })

  it('401 → refresh réussi → rejoue la requête', async () => {
    setupWindow('old')
    fetchMock
      .mockResolvedValueOnce(resp(401))
      .mockResolvedValueOnce(resp(200, { accessToken: 'fresh' }))
      .mockResolvedValueOnce(resp(200, { data: 'ok' }))
    const { apiFetch } = await load()
    const r = await apiFetch('/posts')
    expect(r.status).toBe(200)
    expect(fetchMock.mock.calls[1][0]).toBe('/api/auth/refresh')
    expect((fetchMock.mock.calls[2][1].headers as Headers).get('Authorization')).toBe('Bearer fresh')
    expect(store['breezy-access-token']).toBe('fresh')
  })

  it('401 → refresh échoue → efface la session et redirige', async () => {
    setupWindow('old')
    fetchMock
      .mockResolvedValueOnce(resp(401))
      .mockResolvedValueOnce(resp(401)) // refresh KO
    const { apiFetch } = await load()
    const r = await apiFetch('/posts')
    expect(r.status).toBe(401)
    expect(store['breezy-access-token']).toBeUndefined()
    expect(assigned).toEqual(['/login'])
  })

  it('401 → token frais mais toujours 401 → session morte', async () => {
    setupWindow('old')
    fetchMock
      .mockResolvedValueOnce(resp(401))
      .mockResolvedValueOnce(resp(200, { accessToken: 'fresh' }))
      .mockResolvedValueOnce(resp(401))
    const { apiFetch } = await load()
    await apiFetch('/posts')
    expect(store['breezy-access-token']).toBeUndefined()
    expect(assigned).toEqual(['/login'])
  })

  it('refresh single-flight : 2 requêtes 401 → 1 seul appel /api/auth/refresh', async () => {
    setupWindow('old')
    fetchMock.mockImplementation(async (url: string, init?: { headers?: Headers }) => {
      if (url === '/api/auth/refresh') return resp(200, { accessToken: 'fresh' })
      // Avant refresh (Bearer old) → 401 ; rejoué (Bearer fresh) → 200.
      const auth = (init?.headers as Headers | undefined)?.get('Authorization')
      return auth === 'Bearer fresh' ? resp(200, { data: 'ok' }) : resp(401)
    })
    const { apiFetch } = await load()
    await Promise.all([apiFetch('/a'), apiFetch('/b')])
    const refreshCalls = fetchMock.mock.calls.filter((c) => c[0] === '/api/auth/refresh')
    expect(refreshCalls).toHaveLength(1)
  })
})

describe('logout', () => {
  it('révoque côté serveur, efface la session et redirige', async () => {
    setupWindow('tok')
    fetchMock.mockResolvedValue(resp(200))
    const { logout } = await load()
    await logout()
    expect(fetchMock.mock.calls[0][0]).toBe('/api/auth/logout')
    expect(store['breezy-access-token']).toBeUndefined()
    expect(assigned).toEqual(['/login'])
  })

  it('best-effort : nettoie même si l’appel réseau échoue', async () => {
    setupWindow('tok')
    vi.stubGlobal('fetch', vi.fn(() => Promise.reject(new Error('net'))))
    const { logout } = await load()
    await logout()
    expect(store['breezy-access-token']).toBeUndefined()
    expect(assigned).toEqual(['/login'])
  })
})

describe('bootstrapSession', () => {
  it('sans access token → tente un refresh une seule fois', async () => {
    setupWindow(null)
    fetchMock.mockResolvedValue(resp(200, { accessToken: 'fresh' }))
    const { bootstrapSession } = await load()
    await bootstrapSession()
    await bootstrapSession() // garde : pas de 2e refresh
    const refreshCalls = fetchMock.mock.calls.filter((c) => c[0] === '/api/auth/refresh')
    expect(refreshCalls).toHaveLength(1)
    expect(store['breezy-access-token']).toBe('fresh')
  })

  it('access token déjà présent → pas de refresh', async () => {
    setupWindow('tok')
    const { bootstrapSession } = await load()
    await bootstrapSession()
    expect(fetchMock).not.toHaveBeenCalled()
  })
})
