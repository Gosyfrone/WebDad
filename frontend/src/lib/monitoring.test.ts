import { afterEach, describe, expect, it, vi } from 'vitest'

const apiFetch = vi.fn()
vi.mock('@/lib/auth-client', () => ({ apiFetch: (...a: unknown[]) => apiFetch(...a) }))

import { getMonitoring } from '@/lib/monitoring'

function json(body: unknown, init: { ok?: boolean; status?: number } = {}) {
  return { ok: init.ok ?? true, status: init.status ?? 200, json: () => Promise.resolve(body) }
}
afterEach(() => vi.clearAllMocks())

describe('getMonitoring', () => {
  it('mappe l’instantané (statut normalisé, défauts)', async () => {
    apiFetch.mockResolvedValue(json({ data: { generated_at: 'now', services: [
      { name: 'gw', prefix: '/', status: 'up', latency_ms: 12, uptime_seconds: 99, error: undefined },
      { name: 'post', status: 'weird' },
    ] } }))
    const snap = await getMonitoring()
    expect(snap.generatedAt).toBe('now')
    expect(snap.services[0]).toEqual({ name: 'gw', prefix: '/', status: 'up', latencyMs: 12, uptimeSeconds: 99, error: undefined })
    expect(snap.services[1].status).toBe('down')
    expect(snap.services[1].latencyMs).toBe(0)
  })

  it('data absent → instantané vide', async () => {
    apiFetch.mockResolvedValue(json({}))
    expect(await getMonitoring()).toEqual({ generatedAt: '', services: [] })
  })

  it('non-ok → erreur', async () => {
    apiFetch.mockResolvedValue(json({ error: 'forbidden' }, { ok: false, status: 403 }))
    await expect(getMonitoring()).rejects.toThrow('forbidden')
  })
})
