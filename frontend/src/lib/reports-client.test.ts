import { afterEach, describe, expect, it, vi } from 'vitest'

const apiFetch = vi.fn()
vi.mock('@/lib/auth-client', () => ({ apiFetch: (...a: unknown[]) => apiFetch(...a) }))
vi.mock('@/lib/media', () => ({ mediaUrl: (id: string) => `/media/${id}` }))

import * as reports from '@/lib/reports'

function json(body: unknown, init: { ok?: boolean; status?: number } = {}) {
  return { ok: init.ok ?? true, status: init.status ?? 200, json: () => Promise.resolve(body) }
}
const ok = (data: unknown = {}) => json({ data })
afterEach(() => vi.clearAllMocks())

const apiTicket = {
  id: 't1', category: 'moderation', entity_type: 'post', status: 'open', report_count: 2,
  reason_tags: { spam: 2 }, last_reported_at: 'l', created_at: 'c',
  reports: [{ reporter_id: 'r', reason: 'spam', text: 'x', attachment_id: 'att1', created_at: 'c' }],
  actions: [{ moderator_id: 'm', type: 'reply', text: 'hi', created_at: 'c' }],
}

describe('createReport', () => {
  it('POST /reports avec le mapping snake_case', async () => {
    apiFetch.mockResolvedValue(json({}, { ok: true }))
    await reports.createReport({ category: 'moderation', entityType: 'post', entityId: 'p1', entityOwnerId: 'o', reason: 'spam', text: 't', attachmentId: 'a' })
    const body = JSON.parse(apiFetch.mock.calls[0][1].body)
    expect(body).toMatchObject({ category: 'moderation', entity_type: 'post', entity_id: 'p1', entity_owner_id: 'o', reason: 'spam', attachment_id: 'a' })
  })
  it('échec → ReportApiError', async () => {
    apiFetch.mockResolvedValue(json({ error: 'déjà signalé' }, { ok: false, status: 409 }))
    await expect(reports.createReport({ category: 'moderation', reason: 'spam', text: 't' })).rejects.toMatchObject({ status: 409, name: 'ReportApiError' })
  })
})

describe('tickets', () => {
  it('listTickets mappe et passe tous les filtres', async () => {
    apiFetch.mockResolvedValue(ok([apiTicket]))
    const [t] = await reports.listTickets({ category: 'moderation', status: 'open', minReports: 3, since: 's', until: 'u' })
    expect(t.id).toBe('t1')
    expect(t.reasonTags).toEqual({ spam: 2 })
    expect(t.reports[0].attachmentUrl).toBe('/media/att1')
    expect(t.actions[0].type).toBe('reply')
    const url = String(apiFetch.mock.calls[0][0])
    const p = new URLSearchParams(url.split('?')[1])
    expect(p.get('status')).toBe('open')
    expect(p.get('min_reports')).toBe('3')
    expect(p.get('since')).toBe('s')
    expect(p.get('until')).toBe('u')
  })
  it('listTickets gère défauts (reports/actions absents, minReports 0 ignoré)', async () => {
    apiFetch.mockResolvedValue(ok([{ id: 't', category: 'moderation', entity_type: 'post', status: 'open', report_count: 0, last_reported_at: 'l', created_at: 'c' }]))
    const [t] = await reports.listTickets({ category: 'moderation', minReports: 0 })
    expect(t.reports).toEqual([])
    expect(t.actions).toEqual([])
    expect(String(apiFetch.mock.calls[0][0])).not.toContain('min_reports')
  })
  it('getTicket / replyTicket / changeTicketStatus', async () => {
    apiFetch.mockResolvedValue(ok(apiTicket))
    expect((await reports.getTicket('t1')).id).toBe('t1')
    await reports.replyTicket('t1', 'salut')
    expect(JSON.parse(apiFetch.mock.calls[1][1].body)).toEqual({ text: 'salut' })
    await reports.changeTicketStatus('t1', 'closed')
    expect(JSON.parse(apiFetch.mock.calls[2][1].body)).toEqual({ status: 'closed' })
  })
  it('recordTicketRemoval / transferTicket / approveTicket POST', async () => {
    apiFetch.mockResolvedValue(ok(apiTicket))
    await reports.recordTicketRemoval('t1')
    await reports.transferTicket('t1')
    await reports.approveTicket('t1')
    const urls = apiFetch.mock.calls.map((c) => String(c[0]))
    expect(urls).toEqual(['/reports/tickets/t1/removal', '/reports/tickets/t1/transfer', '/reports/tickets/t1/approve'])
  })
})

describe('settings + warnings', () => {
  it('getReportSettings / updateReportSettings', async () => {
    apiFetch.mockResolvedValue(ok({ auto_hide_threshold: 5 }))
    expect(await reports.getReportSettings()).toEqual({ autoHideThreshold: 5 })
    await reports.updateReportSettings(7)
    expect(JSON.parse(apiFetch.mock.calls[1][1].body)).toEqual({ auto_hide_threshold: 7 })
  })
  it('getReportSettings tolère data absent (0)', async () => {
    apiFetch.mockResolvedValue(ok(null))
    expect(await reports.getReportSettings()).toEqual({ autoHideThreshold: 0 })
  })
  it('issueWarning POST le mapping', async () => {
    apiFetch.mockResolvedValue(json({}, { ok: true }))
    await reports.issueWarning({ targetUserId: 'u', ticketId: 't', message: 'm' })
    expect(JSON.parse(apiFetch.mock.calls[0][1].body)).toEqual({ target_user_id: 'u', ticket_id: 't', message: 'm' })
  })
  it('getUserWarningCount (défaut 0)', async () => {
    apiFetch.mockResolvedValue(ok({ count: 3 }))
    expect(await reports.getUserWarningCount('u')).toBe(3)
    apiFetch.mockResolvedValue(ok(null))
    expect(await reports.getUserWarningCount('u')).toBe(0)
  })
  it('fetchPendingWarnings mappe', async () => {
    apiFetch.mockResolvedValue(ok([{ id: 'w', ticket_id: 't', message: 'm', issued_by: 'mod', created_at: 'c' }]))
    expect((await reports.fetchPendingWarnings())[0]).toEqual({ id: 'w', ticketId: 't', message: 'm', issuedBy: 'mod', createdAt: 'c' })
  })
  it('ackWarning POST', async () => {
    apiFetch.mockResolvedValue(json({}, { ok: true }))
    await reports.ackWarning('w')
    expect(String(apiFetch.mock.calls[0][0])).toBe('/reports/warnings/w/ack')
  })
})
