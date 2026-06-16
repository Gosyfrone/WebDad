/**
 * Client de signalement / modération (report-service, préfixe gateway `/reports`).
 *
 * Trois usages :
 *   - tout utilisateur : déposer un signalement (`createReport`), lire/acquitter
 *     SES avertissements (`fetchPendingWarnings`/`ackWarning`) ;
 *   - modérateur + admin : lister/consulter les tickets, répondre, changer le
 *     statut, émettre un avertissement ;
 *   - admin : transférer un ticket de bug vers la modération.
 *
 * Une pièce jointe (image ≤ 5 Mb) passe par le media-service (`uploadMedia`,
 * agnostique du contenu) ; on stocke l'id renvoyé dans le signalement.
 *
 * ⚠️ À usage CLIENT uniquement (`apiFetch` lit le token en localStorage).
 */

import { apiFetch } from '@/lib/auth-client'
import { mediaUrl } from '@/lib/media'

// ── Constantes alignées sur le back ─────────────────────────────────────────

export type ReportCategory = 'moderation' | 'bug'
export type ReportEntityType = 'post' | 'message' | 'group_message' | 'profile' | 'app'
export type TicketStatus = 'open' | 'closed' | 'reopened'
export type ReportReason = 'inappropriate' | 'offensive' | 'bug' | 'spam' | 'other'

/** Bornes de texte par catégorie (mêmes valeurs que le validateur back). */
export const MAX_MODERATION_TEXT = 255
export const MAX_BUG_TEXT = 500
/** Taille max d'une pièce jointe : 5 Mb. */
export const MAX_ATTACHMENT_BYTES = 5 * 1024 * 1024
/** Longueur max d'un message d'avertissement (alignée sur le back). */
export const MAX_WARNING_MESSAGE_HINT = 500

/**
 * Motifs proposés dans le formulaire (libellés traduits côté UI via i18n).
 * « Bug technique » fait partie des motifs : le choisir bascule le signalement
 * en catégorie BUG (limite 500, onglet Administration) ; les autres motifs →
 * catégorie MODÉRATION (limite 255, onglet Modération).
 */
export const ALL_REASONS: ReportReason[] = ['inappropriate', 'offensive', 'spam', 'bug', 'other']

/** Catégorie déduite du motif (« bug » → rapport technique, sinon modération). */
export function categoryForReason(reason: ReportReason): ReportCategory {
  return reason === 'bug' ? 'bug' : 'moderation'
}

export class ReportApiError extends Error {
  status: number
  constructor(message: string, status: number) {
    super(message)
    this.name = 'ReportApiError'
    this.status = status
  }
}

// ── Modèles front ────────────────────────────────────────────────────────────

export interface ChildReport {
  reporterId: string
  reason: ReportReason
  text: string
  /** Copie en clair d'un message chiffré divulguée par le signaleur (participant). */
  disclosedContent?: string
  /** URL gateway de la pièce jointe (résolue), si présente. */
  attachmentUrl?: string
  createdAt: string
}

export interface TicketAction {
  moderatorId: string
  type: 'reply' | 'status_change' | 'transfer' | 'auto_reopen' | 'content_removed'
  text: string
  status?: TicketStatus
  createdAt: string
}

export interface Ticket {
  id: string
  category: ReportCategory
  entityType: ReportEntityType
  entityId: string
  /** Propriétaire de l'entité (auteur du post/message, utilisateur du profil). */
  entityOwnerId: string
  status: TicketStatus
  reportCount: number
  /** motif → nombre d'occurrences (étiquettes récurrentes). */
  reasonTags: Record<string, number>
  reports: ChildReport[]
  actions: TicketAction[]
  lastReportedAt: string
  createdAt: string
}

export interface Warning {
  id: string
  ticketId?: string
  message: string
  issuedBy: string
  createdAt: string
}

// ── (de)sérialisation ─────────────────────────────────────────────────────────

interface ApiTicket {
  id: string
  category: ReportCategory
  entity_type: ReportEntityType
  entity_id?: string
  entity_owner_id?: string
  status: TicketStatus
  report_count: number
  reason_tags?: Record<string, number>
  reports?: { reporter_id: string; reason: ReportReason; text?: string; disclosed_content?: string; attachment_id?: string; created_at: string }[]
  actions?: { moderator_id: string; type: TicketAction['type']; text?: string; status?: TicketStatus; created_at: string }[]
  last_reported_at: string
  created_at: string
}

interface ApiWarning {
  id: string
  ticket_id?: string
  message: string
  issued_by: string
  created_at: string
}

function toTicket(t: ApiTicket): Ticket {
  return {
    id: t.id,
    category: t.category,
    entityType: t.entity_type,
    entityId: t.entity_id ?? '',
    entityOwnerId: t.entity_owner_id ?? '',
    status: t.status,
    reportCount: t.report_count,
    reasonTags: t.reason_tags ?? {},
    reports: (t.reports ?? []).map((r) => ({
      reporterId: r.reporter_id,
      reason: r.reason,
      text: r.text ?? '',
      disclosedContent: r.disclosed_content,
      attachmentUrl: r.attachment_id ? mediaUrl(r.attachment_id) : undefined,
      createdAt: r.created_at,
    })),
    actions: (t.actions ?? []).map((a) => ({
      moderatorId: a.moderator_id,
      type: a.type,
      text: a.text ?? '',
      status: a.status,
      createdAt: a.created_at,
    })),
    lastReportedAt: t.last_reported_at,
    createdAt: t.created_at,
  }
}

async function unwrap<T>(res: Response): Promise<T> {
  const body = (await res.json().catch(() => null)) as { data?: T; error?: string } | null
  if (!res.ok) {
    throw new ReportApiError(body?.error ?? `Erreur ${res.status}`, res.status)
  }
  return (body?.data ?? null) as T
}

async function expectOk(res: Response): Promise<void> {
  if (!res.ok) {
    const body = (await res.json().catch(() => null)) as { error?: string } | null
    throw new ReportApiError(body?.error ?? `Erreur ${res.status}`, res.status)
  }
}

// ── Dépôt d'un signalement (tout utilisateur) ────────────────────────────────

export interface CreateReportInput {
  category: ReportCategory
  /** Entité signalée (post/profil/message) — conservée même pour un bug. */
  entityType?: ReportEntityType
  entityId?: string
  /** Propriétaire de l'entité (pour pouvoir avertir l'auteur côté modération). */
  entityOwnerId?: string
  reason: ReportReason
  text: string
  /** Copie en clair d'un message chiffré, divulguée par le signaleur. */
  disclosedContent?: string
  /** id média d'une pièce jointe (cf. uploadMedia), optionnel. */
  attachmentId?: string
}

export async function createReport(input: CreateReportInput): Promise<void> {
  await expectOk(
    await apiFetch('/reports', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        category: input.category,
        entity_type: input.entityType,
        entity_id: input.entityId,
        entity_owner_id: input.entityOwnerId,
        reason: input.reason,
        text: input.text,
        disclosed_content: input.disclosedContent,
        attachment_id: input.attachmentId,
      }),
    }),
  )
}

// ── Tickets (modération + admin) ─────────────────────────────────────────────

export interface TicketFilters {
  category: ReportCategory
  status?: TicketStatus
  minReports?: number
  /** bornes ISO du dernier signalement. */
  since?: string
  until?: string
}

export async function listTickets(filters: TicketFilters): Promise<Ticket[]> {
  const params = new URLSearchParams({ category: filters.category })
  if (filters.status) params.set('status', filters.status)
  if (filters.minReports && filters.minReports > 0) params.set('min_reports', String(filters.minReports))
  if (filters.since) params.set('since', filters.since)
  if (filters.until) params.set('until', filters.until)
  const list = await unwrap<ApiTicket[]>(await apiFetch(`/reports/tickets?${params}`))
  return (list ?? []).map(toTicket)
}

export async function getTicket(id: string): Promise<Ticket> {
  return toTicket(await unwrap<ApiTicket>(await apiFetch(`/reports/tickets/${id}`)))
}

export async function replyTicket(id: string, text: string): Promise<Ticket> {
  return toTicket(
    await unwrap<ApiTicket>(
      await apiFetch(`/reports/tickets/${id}/replies`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ text }),
      }),
    ),
  )
}

export async function changeTicketStatus(id: string, status: TicketStatus): Promise<Ticket> {
  return toTicket(
    await unwrap<ApiTicket>(
      await apiFetch(`/reports/tickets/${id}/status`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ status }),
      }),
    ),
  )
}

/**
 * Journalise dans le ticket le RETRAIT du contenu signalé par la modération
 * (post masqué, message supprimé…). Ne change PAS le statut : la clôture reste
 * une décision manuelle du modérateur.
 */
export async function recordTicketRemoval(id: string): Promise<Ticket> {
  return toTicket(await unwrap<ApiTicket>(await apiFetch(`/reports/tickets/${id}/removal`, { method: 'POST' })))
}

/** Transfère un ticket de bug vers la modération (admin uniquement). */
export async function transferTicket(id: string): Promise<Ticket> {
  return toTicket(await unwrap<ApiTicket>(await apiFetch(`/reports/tickets/${id}/transfer`, { method: 'POST' })))
}

// ── Avertissements (warns) ────────────────────────────────────────────────────

export async function issueWarning(input: {
  targetUserId: string
  ticketId?: string
  message: string
}): Promise<void> {
  await expectOk(
    await apiFetch('/reports/warnings', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        target_user_id: input.targetUserId,
        ticket_id: input.ticketId,
        message: input.message,
      }),
    }),
  )
}

export async function fetchPendingWarnings(): Promise<Warning[]> {
  const list = await unwrap<ApiWarning[]>(await apiFetch('/reports/warnings/pending'))
  return (list ?? []).map((w) => ({
    id: w.id,
    ticketId: w.ticket_id,
    message: w.message,
    issuedBy: w.issued_by,
    createdAt: w.created_at,
  }))
}

export async function ackWarning(id: string): Promise<void> {
  await expectOk(await apiFetch(`/reports/warnings/${id}/ack`, { method: 'POST' }))
}
