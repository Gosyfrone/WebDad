import { afterEach, describe, expect, it, vi } from 'vitest'

const apiFetch = vi.fn()
vi.mock('@/lib/auth-client', () => ({ apiFetch: (...a: unknown[]) => apiFetch(...a) }))

import { getMfaStatus, setupMfa, enableMfa, disableMfa, MfaError } from '@/lib/mfa'

function json(body: unknown, init: { ok?: boolean; status?: number } = {}) {
  return { ok: init.ok ?? true, status: init.status ?? 200, json: () => Promise.resolve(body) }
}
afterEach(() => vi.clearAllMocks())

describe('getMfaStatus / setupMfa', () => {
  it('getMfaStatus renvoie data', async () => {
    apiFetch.mockResolvedValue(json({ data: { enabled: true, configured: true } }))
    expect(await getMfaStatus()).toEqual({ enabled: true, configured: true })
  })
  it('setupMfa POST et renvoie le secret', async () => {
    apiFetch.mockResolvedValue(json({ data: { secret: 's', otpauth_url: 'o', qr_data_uri: 'q' } }))
    const s = await setupMfa()
    expect(s.secret).toBe('s')
    expect(apiFetch.mock.calls[0][1].method).toBe('POST')
  })
})

describe('enableMfa / disableMfa', () => {
  it('enableMfa envoie le code', async () => {
    apiFetch.mockResolvedValue(json({ data: {} }))
    await enableMfa('123456')
    expect(JSON.parse(apiFetch.mock.calls[0][1].body)).toEqual({ code: '123456' })
  })
  it('disableMfa envoie la preuve (code ou mot de passe)', async () => {
    apiFetch.mockResolvedValue(json({ data: {} }))
    await disableMfa({ password: 'pw' })
    expect(JSON.parse(apiFetch.mock.calls[0][1].body)).toEqual({ password: 'pw' })
  })
})

describe('parse (erreurs)', () => {
  it('non-ok → MfaError portant le code machine', async () => {
    apiFetch.mockResolvedValue(json({ error: 'code invalide', code: 'invalid_mfa_code' }, { ok: false, status: 400 }))
    await expect(enableMfa('000000')).rejects.toMatchObject({ code: 'invalid_mfa_code' })
  })
  it('non-ok sans message → libellé par défaut', async () => {
    apiFetch.mockResolvedValue({ ok: false, status: 500, json: () => Promise.reject(new Error('x')) })
    await expect(getMfaStatus()).rejects.toThrow('Erreur MFA')
  })
  it('MfaError expose le code', () => {
    expect(new MfaError('m', 'c').code).toBe('c')
  })
})
