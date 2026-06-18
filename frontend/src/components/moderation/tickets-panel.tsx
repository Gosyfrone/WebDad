'use client'

import { useCallback, useEffect, useState } from 'react'
import { Loader2, ShieldAlert } from 'lucide-react'

import { listTickets, type ReportCategory, type Ticket, type TicketStatus } from '@/lib/reports'
import { timeAgo } from '@/lib/utils'
import { cn } from '@/lib/utils'
import { useLanguage } from '@/components/language-provider'
import { useNow } from '@/hooks/use-now'
import { TicketDetail } from '@/components/moderation/ticket-detail'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

interface TicketsPanelProps {
  /** moderation (onglet Modération) ou bug (onglet Administration). */
  category: ReportCategory
  /** L'acteur est administrateur (transfert bug → modération). */
  canTransfer: boolean
}

const STATUS_VARIANT: Record<TicketStatus, 'secondary' | 'destructive'> = {
  open: 'destructive',
  reopened: 'destructive',
  closed: 'secondary',
  approved: 'secondary',
}

const STATUSES: (TicketStatus | '')[] = ['', 'open', 'reopened', 'closed', 'approved']

/**
 * Liste des tickets d'une catégorie, triés par volume de signalements
 * décroissant (tri serveur). Filtres CUMULABLES : statut, nombre minimal de
 * signalements, plage temporelle du dernier signalement. Clic → détail.
 */
export function TicketsPanel({ category, canTransfer }: TicketsPanelProps) {
  const { t, locale } = useLanguage()
  useNow() // re-render chaque seconde → « Dernier : Xs » qui s'incrémente
  const [tickets, setTickets] = useState<Ticket[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(false)
  const [selected, setSelected] = useState<string | null>(null)

  // Filtres cumulables.
  const [status, setStatus] = useState<TicketStatus | ''>('')
  const [minReports, setMinReports] = useState('')
  const [since, setSince] = useState('')
  const [until, setUntil] = useState('')

  const load = useCallback(async () => {
    setLoading(true)
    setError(false)
    try {
      setTickets(
        await listTickets({
          category,
          status: status || undefined,
          minReports: minReports ? Number(minReports) : undefined,
          // <input type=date> → borne de journée en ISO (00:00 / 23:59:59).
          since: since ? new Date(`${since}T00:00:00`).toISOString() : undefined,
          until: until ? new Date(`${until}T23:59:59`).toISOString() : undefined,
        }),
      )
    } catch {
      setError(true)
    } finally {
      setLoading(false)
    }
  }, [category, status, minReports, since, until])

  useEffect(() => {
    void load()
  }, [load])

  function resetFilters() {
    setStatus('')
    setMinReports('')
    setSince('')
    setUntil('')
  }

  // Vue PLEINE du ticket sélectionné (remplace la liste), avec retour à la liste.
  if (selected) {
    return (
      <TicketDetail
        ticketId={selected}
        canTransfer={canTransfer}
        onBack={() => setSelected(null)}
        onChanged={() => void load()}
      />
    )
  }

  return (
    <div className="flex flex-col gap-4 px-4 py-4">
      {/* Filtres cumulables */}
      <div className="flex flex-wrap items-end gap-3 rounded-xl border p-3">
        <label className="flex flex-col gap-1 text-xs font-medium text-muted-foreground">
          {t('tickets.filter_status')}
          <select
            value={status}
            onChange={(e) => setStatus(e.target.value as TicketStatus | '')}
            className="h-9 rounded-md border border-input bg-background px-2 text-sm text-foreground"
          >
            {STATUSES.map((s) => (
              <option key={s || 'all'} value={s}>
                {s ? t(`tickets.status.${s}`) : t('tickets.filter_status_all')}
              </option>
            ))}
          </select>
        </label>

        <label className="flex flex-col gap-1 text-xs font-medium text-muted-foreground">
          {t('tickets.filter_min_reports')}
          <Input
            type="number"
            min={0}
            value={minReports}
            onChange={(e) => setMinReports(e.target.value)}
            className="h-9 w-28"
          />
        </label>

        <label className="flex flex-col gap-1 text-xs font-medium text-muted-foreground">
          {t('tickets.filter_since')}
          <Input type="date" value={since} onChange={(e) => setSince(e.target.value)} className="h-9 w-40" />
        </label>

        <label className="flex flex-col gap-1 text-xs font-medium text-muted-foreground">
          {t('tickets.filter_until')}
          <Input type="date" value={until} onChange={(e) => setUntil(e.target.value)} className="h-9 w-40" />
        </label>

        <Button variant="ghost" size="sm" onClick={resetFilters}>
          {t('tickets.filter_reset')}
        </Button>
      </div>

      {loading ? (
        <div className="flex items-center justify-center py-16 text-muted-foreground">
          <Loader2 className="h-5 w-5 animate-spin" aria-hidden />
        </div>
      ) : error ? (
        <p className="py-16 text-center text-sm text-muted-foreground">{t('tickets.error')}</p>
      ) : tickets.length === 0 ? (
        <p className="py-16 text-center text-sm text-muted-foreground">
          {category === 'bug' ? t('tickets.empty_bug') : t('tickets.empty')}
        </p>
      ) : (
        <ul className="flex flex-col gap-2">
          {tickets.map((ticket) => (
            <li key={ticket.id}>
              <button
                type="button"
                onClick={() => setSelected(ticket.id)}
                className="panel flex w-full items-center gap-3 rounded-2xl border p-3 text-left shadow-sm transition-colors hover:border-[#5B6CFF]/50"
              >
                <span
                  className={cn(
                    'flex h-10 w-10 shrink-0 items-center justify-center rounded-full text-sm font-bold',
                    ticket.reportCount > 1 ? 'bg-[#5B6CFF]/15 text-[#5B6CFF]' : 'bg-muted text-muted-foreground',
                  )}
                >
                  {ticket.reportCount}
                </span>
                <div className="flex min-w-0 flex-1 flex-col gap-1">
                  <div className="flex flex-wrap items-center gap-2">
                    <Badge variant="outline" className="text-[10px]">{t(`tickets.entity.${ticket.entityType}`)}</Badge>
                    <Badge variant={STATUS_VARIANT[ticket.status]} className="text-[10px]">
                      {t(`tickets.status.${ticket.status}`)}
                    </Badge>
                  </div>
                  <span className="text-xs text-muted-foreground">
                    {t('tickets.reports_count', { count: ticket.reportCount })} ·{' '}
                    {t('tickets.last_report', { when: timeAgo(ticket.lastReportedAt, locale) })}
                  </span>
                </div>
                <ShieldAlert className="h-4 w-4 shrink-0 text-muted-foreground" aria-hidden />
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
