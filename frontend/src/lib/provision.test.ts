import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

// `provision` appelle `fetch` directement (usage serveur). On stube le global.
vi.mock('@/lib/config', () => ({ apiUrl: (p: string) => `http://api${p}` }))

import { provisionUser, markLoginActivity, markLogoutActivity } from '@/lib/provision'

const fetchMock = vi.fn()
beforeEach(() => {
  fetchMock.mockReset()
  fetchMock.mockResolvedValue({ ok: true })
  vi.stubGlobal('fetch', fetchMock)
})
afterEach(() => vi.unstubAllGlobals())

const urls = () => fetchMock.mock.calls.map((c) => String(c[0]))

describe('provisionUser', () => {
  it('sans username → provisioning par email (GET /users/me)', async () => {
    await provisionUser('tok')
    expect(urls()).toEqual(['http://api/users/me'])
    expect(fetchMock.mock.calls[0][1].headers.Authorization).toBe('Bearer tok')
  })

  it('username vide (espaces) → repli email', async () => {
    await provisionUser('tok', { username: '   ' })
    expect(urls()).toEqual(['http://api/users/me'])
  })

  it('avec username → POST /users puis POST /profils (display_name=username)', async () => {
    await provisionUser('tok', { username: 'al', birthDate: '2000-01-01', gender: 'male' })
    expect(urls()).toEqual(['http://api/users', 'http://api/profils'])
    const profilBody = JSON.parse(fetchMock.mock.calls[1][1].body)
    expect(profilBody.display_name).toBe('al')
    expect(profilBody.gender).toBe('male')
    expect(profilBody.birth_date).toContain('2000-01-01')
  })

  it('POST /users non-ok → repli email (pas de POST /profils)', async () => {
    fetchMock.mockImplementation(async (url: string) => (url.endsWith('/users') ? { ok: false } : { ok: true }))
    await provisionUser('tok', { username: 'al' })
    expect(urls()).toEqual(['http://api/users', 'http://api/users/me'])
  })

  it('exception réseau sur POST /users → repli email', async () => {
    fetchMock.mockImplementation(async (url: string) => {
      if (url.endsWith('/users')) throw new Error('net')
      return { ok: true }
    })
    await provisionUser('tok', { username: 'al' })
    expect(urls()).toContain('http://api/users/me')
  })

  it('provisioning email avale les erreurs (best-effort)', async () => {
    fetchMock.mockRejectedValue(new Error('down'))
    await expect(provisionUser('tok')).resolves.toBeUndefined()
  })
})

describe('markLoginActivity / markLogoutActivity', () => {
  it('PATCH les routes d’activité', async () => {
    await markLoginActivity('tok')
    await markLogoutActivity('tok')
    expect(urls()).toEqual(['http://api/profils/me/activity', 'http://api/profils/me/activity/offline'])
    expect(fetchMock.mock.calls[0][1].method).toBe('PATCH')
  })
  it('avalent les erreurs (best-effort)', async () => {
    fetchMock.mockRejectedValue(new Error('x'))
    await expect(markLoginActivity('tok')).resolves.toBeUndefined()
    await expect(markLogoutActivity('tok')).resolves.toBeUndefined()
  })
})
