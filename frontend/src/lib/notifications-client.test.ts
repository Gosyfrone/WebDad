import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

const apiFetch = vi.fn()
const getAccessToken = vi.fn(() => 'tok')
vi.mock('@/lib/auth-client', () => ({
  apiFetch: (...a: unknown[]) => apiFetch(...a),
  getAccessToken: () => getAccessToken(),
}))

import * as notifs from '@/lib/notifications'

function json(body: unknown, init: { ok?: boolean; status?: number } = {}) {
  return { ok: init.ok ?? true, status: init.status ?? 200, json: () => Promise.resolve(body) }
}
function routes(table: Array<[(url: string) => boolean, unknown]>) {
  apiFetch.mockImplementation(async (url: string) => {
    for (const [match, res] of table) if (match(url)) return res
    return json({}, { ok: false, status: 404 })
  })
}
const path = (p: string) => (url: string) => url.split('?')[0] === p
const starts = (p: string) => (url: string) => url.startsWith(p)

function rawNotif(id: string, actor: string, over: Record<string, unknown> = {}) {
  return { id, type: 'like', last_actor_id: actor, count: 1, is_read: false, created_at: 'c', updated_at: 'u', ...over }
}

beforeEach(() => getAccessToken.mockReturnValue('tok'))
afterEach(() => { apiFetch.mockReset(); vi.unstubAllGlobals() })

describe('listNotifications (résolution d’acteur)', () => {
  it('résout l’acteur (user + profil) et passe le curseur before', async () => {
    routes([
      [path('/notifications'), json({ data: [rawNotif('n1', 'act1')] })],
      [path('/users/act1'), json({ data: { id: 'act1', username: 'zaid' } })],
      [path('/profils/act1'), json({ data: { display_name: 'Zaid', avatar_url: 'http://a' } })],
    ])
    const [n] = await notifs.listNotifications('cursorX')
    expect(n.actor.username).toBe('zaid')
    expect(n.actor.displayName).toBe('Zaid')
    expect(String(apiFetch.mock.calls[0][0])).toContain('before=cursorX')
  })

  it('acteur introuvable → repli (username vide, displayName défaut)', async () => {
    routes([
      [path('/notifications'), json({ data: [rawNotif('n2', 'ghost')] })],
      // /users/ghost et /profils/ghost → 404
    ])
    const [n] = await notifs.listNotifications()
    expect(n.actor.username).toBe('')
    expect(n.actor.displayName).toBe('Utilisateur')
  })
})

describe('getUnreadCount / markAllRead / markRead', () => {
  it('getUnreadCount renvoie le compteur, 0 si non-ok', async () => {
    routes([[path('/notifications/unread-count'), json({ data: { count: 9 } })]])
    expect(await notifs.getUnreadCount()).toBe(9)
    routes([[path('/notifications/unread-count'), json({}, { ok: false, status: 500 })]])
    expect(await notifs.getUnreadCount()).toBe(0)
  })
  it('markAllRead / markRead POST', async () => {
    routes([[starts('/notifications'), json({}, { ok: true })]])
    await notifs.markAllRead()
    await notifs.markRead('n1')
    const urls = apiFetch.mock.calls.map((c) => String(c[0]))
    expect(urls).toContain('/notifications/read')
    expect(urls).toContain('/notifications/n1/read')
  })
})

describe('dispatch/subscribe follow request decision', () => {
  it('sans window → no-op', () => {
    vi.stubGlobal('window', undefined)
    expect(() => notifs.dispatchFollowRequestDecision({ actorId: 'a', status: 'accepted' })).not.toThrow()
    expect(notifs.subscribeFollowRequestDecision(() => {})()).toBeUndefined()
  })
  it('relaie la décision puis se désabonne', () => {
    const listeners = new Map<string, (e: Event) => void>()
    vi.stubGlobal('window', {
      addEventListener: (t: string, h: (e: Event) => void) => listeners.set(t, h),
      removeEventListener: (t: string) => listeners.delete(t),
      dispatchEvent: (e: Event) => listeners.get(e.type)?.(e),
    })
    vi.stubGlobal('CustomEvent', class {
      type: string; detail: unknown
      constructor(type: string, init: { detail: unknown }) { this.type = type; this.detail = init.detail }
    })
    const seen: unknown[] = []
    const off = notifs.subscribeFollowRequestDecision((d) => seen.push(d))
    notifs.dispatchFollowRequestDecision({ actorId: 'a', status: 'accepted' })
    expect(seen).toEqual([{ actorId: 'a', status: 'accepted' }])
    off()
    notifs.dispatchFollowRequestDecision({ actorId: 'b', status: 'rejected' })
    expect(seen).toHaveLength(1)
  })
})

describe('connectNotifications', () => {
  let sockets: FakeWS[]
  class FakeWS {
    url: string
    onmessage: ((e: { data: string }) => void) | null = null
    onopen: (() => void) | null = null
    onclose: (() => void) | null = null
    closed = false
    constructor(url: string) { this.url = url; sockets.push(this) }
    close() { this.closed = true }
  }
  beforeEach(() => {
    sockets = []
    vi.stubGlobal('WebSocket', FakeWS as unknown as typeof WebSocket)
    vi.useFakeTimers()
    routes([
      [path('/users/act1'), json({ data: { id: 'act1', username: 'z' } })],
      [path('/profils/act1'), json({ data: { display_name: 'Z' } })],
    ])
  })
  afterEach(() => { vi.useRealTimers(); vi.unstubAllGlobals() })

  it('sans token → pas de socket', () => {
    getAccessToken.mockReturnValue('')
    notifs.connectNotifications({ onNotification: () => {}, onDeleted: () => {}, onRefresh: () => {} })
    expect(sockets).toHaveLength(0)
  })

  it('route chaque type d’évènement', async () => {
    const got = { notif: 0, deleted: '', refresh: 0, decision: null as unknown }
    notifs.connectNotifications({
      onNotification: () => { got.notif++ },
      onDeleted: (id) => { got.deleted = id },
      onRefresh: () => { got.refresh++ },
      onFollowRequestDecision: (d) => { got.decision = d },
    })
    sockets[0].onopen?.()
    sockets[0].onmessage?.({ data: 'oops' }) // non-JSON ignoré
    sockets[0].onmessage?.({ data: JSON.stringify({ type: 'notification', data: rawNotif('n1', 'act1') }) })
    sockets[0].onmessage?.({ data: JSON.stringify({ type: 'notification_deleted', data: { id: 'gone' } }) })
    sockets[0].onmessage?.({ data: JSON.stringify({ type: 'notification_refresh' }) })
    sockets[0].onmessage?.({ data: JSON.stringify({ type: 'follow_request_decision', data: { actor_id: 'a', status: 'accepted' } }) })
    sockets[0].onmessage?.({ data: JSON.stringify({ type: 'follow_request_decision', data: { status: 'weird' } }) }) // ignoré
    await vi.waitFor(() => expect(got.notif).toBe(1))
    expect(got.deleted).toBe('gone')
    expect(got.refresh).toBe(1)
    expect(got.decision).toEqual({ actorId: 'a', status: 'accepted' })
  })

  it('reconnecte après coupure ; close() coupe la reconnexion', () => {
    const h = notifs.connectNotifications({ onNotification: () => {}, onDeleted: () => {}, onRefresh: () => {} })
    sockets[0].onclose?.()
    vi.advanceTimersByTime(1000)
    expect(sockets.length).toBe(2)
    h.close()
    expect(sockets[1].closed).toBe(true)
    sockets[1].onclose?.()
    vi.advanceTimersByTime(10000)
    expect(sockets.length).toBe(2)
  })
})
