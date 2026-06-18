'use client'

import { useCallback, useEffect, useState } from 'react'
import Link from 'next/link'
import { ArrowLeft, ArrowRightLeft, Ban, CheckCircle2, ExternalLink, Loader2, Lock, ShieldAlert, Trash2 } from 'lucide-react'

import {
  approveTicket,
  changeTicketStatus,
  getTicket,
  getUserWarningCount,
  issueWarning,
  recordTicketRemoval,
  replyTicket,
  transferTicket,
  ReportApiError,
  type Ticket,
  type TicketStatus,
} from '@/lib/reports'
import { setUserBanned } from '@/lib/admin'
import { deletePost, getPostById, type FeedPost } from '@/lib/posts'
import { moderateDeleteMessage } from '@/lib/messages'
import { resolveUser, type ResolvedUser } from '@/lib/user-cache'
import { useSession } from '@/lib/session'
import { postHref, profilHref } from '@/lib/routes'
import { timeAgo } from '@/lib/utils'
import { useLanguage } from '@/components/language-provider'
import { useToast } from '@/hooks/use-toast'
import { useNow } from '@/hooks/use-now'
import { WarnDialog } from '@/components/moderation/warn-dialog'
import { PostCard } from '@/components/feed/post-card'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

interface TicketDetailProps {
  ticketId: string
  /** L'acteur est administrateur (débloque le transfert bug → modération). */
  canTransfer: boolean
  /** Retour à la liste des tickets (flèche en haut à gauche). */
  onBack: () => void
  /** Notifie le parent qu'un ticket a changé (recharger la liste). */
  onChanged?: () => void
}

const STATUS_VARIANT: Record<TicketStatus, 'secondary' | 'destructive' | 'outline'> = {
  open: 'destructive',
  reopened: 'destructive',
  closed: 'secondary',
  approved: 'outline',
}

/**
 * Vue PLEINE d'un ticket (pas une modale) : on remplace la liste par le détail,
 * avec une flèche de retour en haut à gauche — même logique que le détail d'un
 * post. Affiche : le CONTENU ORIGINAL signalé (le post réel, pour juger sur
 * pièce), l'en-tête agrégé (volume, dernier signalement, motifs récurrents,
 * statut), le fil des signalements enfants (utilisateur + motif + texte + image),
 * le journal des actions de modération, et les actions : réponse interne,
 * changement de statut, avertissement de l'auteur, retrait du contenu (posts),
 * transfert vers la modération (admin, tickets de bug).
 */
export function TicketDetail({ ticketId, canTransfer, onBack, onChanged }: TicketDetailProps) {
  const { t, locale } = useLanguage()
  const { toast } = useToast()
  const session = useSession()
  useNow() // re-render chaque seconde → durées relatives (timeAgo) qui s'incrémentent
  const [ticket, setTicket] = useState<Ticket | null>(null)
  const [people, setPeople] = useState<Record<string, ResolvedUser>>({})
  const [loading, setLoading] = useState(false)
  const [reply, setReply] = useState('')
  const [busy, setBusy] = useState(false)
  // Contenu original signalé (post) + auteur ciblé (pour avertir / retirer).
  const [post, setPost] = useState<FeedPost | null>(null)
  const [postMissing, setPostMissing] = useState(false)
  const [messageRemoved, setMessageRemoved] = useState(false)
  const [authorId, setAuthorId] = useState<string | null>(null)
  const [warnOpen, setWarnOpen] = useState(false)
  // Message pré-rempli de la modale d'avertissement (justification d'un retrait).
  const [warnDefault, setWarnDefault] = useState('')
  // La modale d'avertissement est en mode « retrait » : valider = supprimer + avertir
  // + clôturer (atomique) ; sinon simple avertissement.
  const [pendingDelete, setPendingDelete] = useState(false)
  // Profil de risque de l'utilisateur ciblé : nb total d'avertissements reçus.
  const [warningCount, setWarningCount] = useState<number | null>(null)
  // Garde-fou de clôture (aucune action posée) et confirmation de bannissement.
  const [closeConfirmOpen, setCloseConfirmOpen] = useState(false)
  const [banConfirmOpen, setBanConfirmOpen] = useState(false)
  const [banning, setBanning] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const tk = await getTicket(ticketId)
      setTicket(tk)
      // Résout les identités (auteurs des signalements + modérateurs + l'utilisateur
      // ciblé pour l'encart « profil de risque » → nom + avatar). Le propriétaire de
      // l'entité (`entity_owner_id`) est inclus quand il est connu.
      const ids = Array.from(
        new Set(
          [
            ...tk.reports.map((r) => r.reporterId),
            ...tk.actions.map((a) => a.moderatorId),
            ...(tk.entityType === 'profile' ? [tk.entityId] : []),
            tk.entityOwnerId,
          ].filter(Boolean),
        ),
      )
      const resolved = await Promise.all(ids.map((id) => resolveUser(id)))
      const peopleMap: Record<string, ResolvedUser> = Object.fromEntries(ids.map((id, i) => [id, resolved[i]]))
      setPeople(peopleMap)

      // Cible de l'avertissement : le propriétaire de l'entité, capturé AU
      // signalement (`entity_owner_id`) → robuste, indépendant d'une relecture du
      // post (qui peut être masqué/supprimé) et disponible aussi pour les messages.
      // Repli pour d'anciens tickets sans ce champ : profil → l'id, post → auteur récupéré.
      let owner = tk.entityOwnerId || (tk.entityType === 'profile' ? tk.entityId : '')
      setAuthorId(owner || null)

      // Chargement du CONTENU original (affichage) pour un ticket de post.
      if (tk.entityType === 'post') {
        const p = await getPostById(tk.entityId).catch(() => null)
        setPost(p)
        setPostMissing(p === null)
        if (!owner) {
          owner = p?.author.id ?? '' // repli si owner non stocké
          setAuthorId(owner || null)
        }
      }

      // Profil de risque : identité de la cible (si pas déjà résolue) + nombre
      // d'avertissements déjà reçus.
      if (owner) {
        if (!peopleMap[owner]) {
          peopleMap[owner] = await resolveUser(owner)
          setPeople({ ...peopleMap })
        }
        setWarningCount(await getUserWarningCount(owner).catch(() => null))
      }
    } catch {
      toast({ title: t('tickets.error'), variant: 'brand' })
    } finally {
      setLoading(false)
    }
  }, [ticketId, t, toast])

  useEffect(() => {
    setTicket(null)
    setReply('')
    setPost(null)
    setPostMissing(false)
    setMessageRemoved(false)
    setAuthorId(null)
    setWarningCount(null)
    void load()
  }, [load])

  async function onReply() {
    if (!ticket || !reply.trim()) return
    setBusy(true)
    try {
      setTicket(await replyTicket(ticket.id, reply.trim()))
      setReply('')
      toast({ title: t('tickets.reply_added') })
      onChanged?.()
    } catch {
      toast({ title: t('tickets.action_failed'), variant: 'brand' })
    } finally {
      setBusy(false)
    }
  }

  async function onStatus(status: TicketStatus) {
    if (!ticket) return
    setBusy(true)
    try {
      setTicket(await changeTicketStatus(ticket.id, status))
      onChanged?.()
    } catch {
      toast({ title: t('tickets.action_failed'), variant: 'brand' })
    } finally {
      setBusy(false)
    }
  }

  async function onTransfer() {
    if (!ticket) return
    setBusy(true)
    try {
      setTicket(await transferTicket(ticket.id))
      toast({ title: t('tickets.transferred') })
      onChanged?.()
    } catch {
      toast({ title: t('tickets.action_failed'), variant: 'brand' })
    } finally {
      setBusy(false)
    }
  }

  // Validation « conforme » : décision TERMINALE. Démasque le post s'il avait été
  // auto-masqué et verrouille tout nouveau signalement de l'entité.
  async function onApprove() {
    if (!ticket) return
    setBusy(true)
    try {
      setTicket(await approveTicket(ticket.id))
      toast({ title: t('tickets.approved') })
      onChanged?.()
    } catch (err) {
      // Backstop du garde-fou serveur : 409 si une sanction a déjà été posée.
      const conflict = err instanceof ReportApiError && err.status === 409
      toast({ title: conflict ? t('tickets.cannot_approve_actioned') : t('tickets.action_failed'), variant: 'brand' })
      if (conflict) await load() // resynchronise (l'action manquait côté front)
    } finally {
      setBusy(false)
    }
  }

  // Ouvre la modale d'avertissement SIMPLE (avertir, sans autre effet).
  function openWarn(defaultMessage = '') {
    setPendingDelete(false)
    setWarnDefault(defaultMessage)
    setWarnOpen(true)
  }

  // Lance un RETRAIT de contenu : ouvre la modale d'avertissement en mode « retrait ».
  // RIEN n'est supprimé tant que le modérateur n'a pas validé → Annuler = aucun effet.
  function openRemovalWarn() {
    if (!ticket) return
    setPendingDelete(true)
    setWarnDefault(
      ticket.entityType === 'post'
        ? t('tickets.warn_after_delete_post')
        : t('tickets.warn_after_delete_message'),
    )
    setWarnOpen(true)
  }

  // Validation du mode « retrait » : suppression du contenu → journal du retrait →
  // avertissement de l'auteur → CLÔTURE automatique. Exécutée seulement à la
  // confirmation de la modale (jamais au simple clic sur « Supprimer »).
  // IDEMPOTENT : chaque étape est sautée si déjà journalisée → après un échec
  // partiel (ex. avertissement KO), un nouvel essai NE redéclenche PAS la
  // suppression ni un second avertissement ; il ne complète que ce qui manque.
  async function handleRemovalConfirm(message: string) {
    if (!ticket) return
    try {
      let current = ticket
      // 1) Retrait effectif du contenu — sauté si déjà retiré (`content_removed`).
      if (!current.actions.some((a) => a.type === 'content_removed')) {
        if (current.entityType === 'post') {
          await deletePost(current.entityId) // suppression douce → « Tweets supprimés »
          setPost(null)
          setPostMissing(true)
        } else {
          await moderateDeleteMessage(current.entityId) // tombstone E2EE-safe (serveur aveugle)
          setMessageRemoved(true)
        }
        current = await recordTicketRemoval(current.id) // journalise `content_removed`
        setTicket(current)
      }
      // 2) Avertissement de l'auteur — sauté s'il a déjà été averti (`warned`). On
      //    relit le ticket pour capter l'action `warned` (sinon un retry clôture-KO
      //    renverrait un second avertissement).
      if (authorId && !current.actions.some((a) => a.type === 'warned')) {
        await issueWarning({ targetUserId: authorId, ticketId: current.id, message })
        current = await getTicket(current.id)
        setTicket(current)
      }
      // 3) Clôture automatique du ticket.
      setTicket(await changeTicketStatus(current.id, 'closed'))
      toast({ title: t('tickets.removed_and_closed') })
      onChanged?.()
    } catch (err) {
      // Message propre au RETRAIT (≠ « avertissement » de WarnDialog). On relance pour
      // que la modale reste ouverte → l'essai suivant reprend là où il s'est arrêté.
      toast({ title: t('tickets.action_failed'), variant: 'brand' })
      throw err
    }
  }

  // Bannissement de l'utilisateur ciblé depuis le ticket (réutilise l'orchestration
  // auth+user de l'annuaire). Un modérateur ne peut bannir qu'un utilisateur simple :
  // le back renvoie 403 si la cible est mod/admin → message dédié.
  async function onBan() {
    if (!authorId) return
    setBanning(true)
    try {
      await setUserBanned(authorId, true)
      toast({ title: t('tickets.user_banned') })
      setBanConfirmOpen(false)
    } catch (err) {
      const restricted = (err as { status?: number })?.status === 403
      toast({ title: restricted ? t('moderation.ban_restricted') : t('tickets.action_failed'), variant: 'brand' })
    } finally {
      setBanning(false)
    }
  }

  function nameOf(id: string): string {
    const p = people[id]
    return p?.displayName || p?.username || t('common.user')
  }

  // Lien vers l'élément signalé (post → détail ; profil → page publique).
  const entityHref =
    ticket?.entityType === 'post'
      ? postHref(ticket.entityId)
      : ticket?.entityType === 'profile' && people[ticket.entityId]?.username
        ? profilHref(people[ticket.entityId].username)
        : null

  const topReasons = ticket
    ? Object.entries(ticket.reasonTags)
        .sort((a, b) => b[1] - a[1])
        .slice(0, 3)
    : []

  // Une sanction (retrait du contenu ou avertissement) a déjà été posée → on ne
  // peut plus « valider » l'entité comme conforme (miroir du garde-fou serveur).
  const hasSanction = !!ticket?.actions.some(
    (a) => a.type === 'content_removed' || a.type === 'warned',
  )
  // Le contenu a déjà été retiré par la modération (journal) — un modérateur VOIT
  // encore les posts masqués, donc `postMissing` ne suffit pas : on s'appuie sur le
  // journal pour ne pas re-proposer « Supprimer » sur un contenu déjà retiré.
  const alreadyRemoved = !!ticket?.actions.some((a) => a.type === 'content_removed')
  // Bannissement possible sur la cible, sauf soi-même (l'accès au panneau est déjà
  // réservé mod/admin ; le back borne les cibles selon le rôle de l'acteur).
  const canBan = !!authorId && authorId !== session?.userId

  return (
    <div className="flex flex-col">
      {/* En-tête de la vue : flèche de retour (haut à gauche) + titre. */}
      <header className="panel z-10 flex items-center gap-3 border-b px-4 py-3 lg:sticky lg:top-0">
        <button
          type="button"
          onClick={onBack}
          aria-label={t('common.back')}
          className="rounded-full p-1.5 text-muted-foreground transition-colors hover:bg-primary/10 hover:text-primary"
        >
          <ArrowLeft className="h-5 w-5" aria-hidden />
        </button>
        <h1 className="flex items-center gap-2 text-lg font-bold">
          <ShieldAlert className="h-5 w-5 text-[#5B6CFF]" aria-hidden />
          {t('tickets.detail_title')}
        </h1>
      </header>

      {loading || !ticket ? (
        <div className="flex items-center justify-center py-16 text-muted-foreground">
          <Loader2 className="h-5 w-5 animate-spin" aria-hidden />
        </div>
      ) : (
        <div className="flex flex-col gap-4 px-4 py-4">
          {/* En-tête agrégé */}
          <div className="flex flex-col gap-2 rounded-xl border p-3">
            <div className="flex flex-wrap items-center gap-2">
              <Badge variant="outline">{t(`tickets.entity.${ticket.entityType}`)}</Badge>
              <Badge variant={STATUS_VARIANT[ticket.status]}>{t(`tickets.status.${ticket.status}`)}</Badge>
              <span className="text-sm font-semibold">
                {t('tickets.reports_count', { count: ticket.reportCount })}
              </span>
              {entityHref && (
                <Link
                  href={entityHref}
                  target="_blank"
                  className="ml-auto inline-flex items-center gap-1 text-xs text-[#5B6CFF] hover:underline"
                >
                  <ExternalLink className="h-3.5 w-3.5" aria-hidden />
                  {t('tickets.open_entity')}
                </Link>
              )}
            </div>
            <p className="text-xs text-muted-foreground">
              {t('tickets.last_report', { when: timeAgo(ticket.lastReportedAt, locale) })}
            </p>
            {topReasons.length > 0 && (
              <div className="flex flex-wrap items-center gap-1.5">
                <span className="text-xs text-muted-foreground">{t('tickets.reason_tags')} :</span>
                {topReasons.map(([reason, count]) => (
                  <Badge key={reason} variant="secondary" className="text-[10px]">
                    {t(`report.reason.${reason}`)} · {count}
                  </Badge>
                ))}
              </div>
            )}
          </div>

          {/* Profil de risque de l'utilisateur ciblé : antécédents (nb d'avertissements)
              + action de bannissement directe depuis le ticket. */}
          {authorId && (
            <div className="flex flex-col gap-2 rounded-xl border p-3">
              <div className="flex items-center gap-2">
                <ShieldAlert className="h-4 w-4 shrink-0 text-[#5B6CFF]" aria-hidden />
                <span className="text-sm font-bold">{t('tickets.target_user')}</span>
              </div>
              <div className="flex flex-wrap items-center gap-2">
                <Avatar className="h-8 w-8 shrink-0">
                  {people[authorId]?.avatarUrl && <AvatarImage src={people[authorId].avatarUrl} alt="" />}
                  <AvatarFallback>{(nameOf(authorId) || 'U').charAt(0).toUpperCase()}</AvatarFallback>
                </Avatar>
                <span className="truncate text-sm font-semibold">{nameOf(authorId)}</span>
                <Badge variant={warningCount && warningCount > 0 ? 'destructive' : 'secondary'} className="text-[10px]">
                  {t('tickets.warning_count', { count: warningCount ?? 0 })}
                </Badge>
                {canBan && (
                  <Button
                    variant="destructive"
                    size="sm"
                    disabled={busy || banning}
                    onClick={() => setBanConfirmOpen(true)}
                    className="ml-auto shrink-0"
                  >
                    <Ban className="mr-1.5 h-4 w-4" aria-hidden />
                    {t('tickets.ban_user')}
                  </Button>
                )}
              </div>
            </div>
          )}

          {/* Contenu original signalé — pour juger sur pièce. */}
          <section className="flex flex-col gap-2">
            <h3 className="text-sm font-bold">{t('tickets.original_content')}</h3>
            {ticket.entityType === 'post' ? (
              post ? (
                <div className="rounded-xl border">
                  <PostCard post={post} embedded noNavigate />
                </div>
              ) : postMissing ? (
                <p className="rounded-xl border border-dashed p-4 text-sm text-muted-foreground">
                  {t('tickets.content_unavailable')}
                </p>
              ) : (
                <div className="flex items-center justify-center py-6 text-muted-foreground">
                  <Loader2 className="h-4 w-4 animate-spin" aria-hidden />
                </div>
              )
            ) : ticket.entityType === 'profile' ? (
              <Link
                href={entityHref ?? '#'}
                target="_blank"
                className="flex items-center gap-3 rounded-xl border p-3 transition-colors hover:border-[#5B6CFF]/50"
              >
                <Avatar className="h-10 w-10">
                  {people[ticket.entityId]?.avatarUrl && (
                    <AvatarImage src={people[ticket.entityId].avatarUrl} alt="" />
                  )}
                  <AvatarFallback>
                    {(nameOf(ticket.entityId) || 'U').charAt(0).toUpperCase()}
                  </AvatarFallback>
                </Avatar>
                <div className="flex min-w-0 flex-col">
                  <span className="truncate text-sm font-bold">{nameOf(ticket.entityId)}</span>
                  {people[ticket.entityId]?.username && (
                    <span className="truncate text-xs text-muted-foreground">
                      @{people[ticket.entityId].username}
                    </span>
                  )}
                </div>
                <ExternalLink className="ml-auto h-4 w-4 shrink-0 text-muted-foreground" aria-hidden />
              </Link>
            ) : (
              // Messages : serveur aveugle (E2EE) → contenu non lisible côté modération.
              <p className="rounded-xl border border-dashed p-4 text-sm text-muted-foreground">
                {t('tickets.content_encrypted')}
              </p>
            )}
          </section>

          {/* Fil des signalements enfants */}
          <div className="flex flex-col gap-2">
            <h3 className="text-sm font-bold">{t('tickets.thread_title', { count: ticket.reports.length })}</h3>
            {ticket.reports.map((r, i) => {
              const p = people[r.reporterId]
              const initial = (p?.displayName || p?.username || 'U').charAt(0).toUpperCase()
              return (
                <div key={i} className="flex gap-2 rounded-lg border p-2.5">
                  <Avatar className="h-8 w-8 shrink-0">
                    {p?.avatarUrl && <AvatarImage src={p.avatarUrl} alt={p.displayName} />}
                    <AvatarFallback>{initial}</AvatarFallback>
                  </Avatar>
                  <div className="flex min-w-0 flex-1 flex-col gap-1">
                    <div className="flex items-center gap-2">
                      <span className="truncate text-sm font-semibold">{nameOf(r.reporterId)}</span>
                      <Badge variant="outline" className="text-[10px]">{t(`report.reason.${r.reason}`)}</Badge>
                      <span className="ml-auto shrink-0 text-xs text-muted-foreground">{timeAgo(r.createdAt, locale)}</span>
                    </div>
                    {r.text && <p className="whitespace-pre-wrap break-words text-sm">{r.text}</p>}
                    {/* Copie divulguée d'un message chiffré (le signaleur l'a transmise). */}
                    {r.disclosedContent && (
                      <blockquote className="mt-1 border-l-2 border-[#5B6CFF]/50 bg-muted/40 px-2 py-1 text-sm">
                        <span className="mb-0.5 block text-[10px] font-semibold uppercase text-muted-foreground">
                          {t('tickets.disclosed_content')}
                        </span>
                        <span className="whitespace-pre-wrap break-words">{r.disclosedContent}</span>
                      </blockquote>
                    )}
                    {r.attachmentUrl && (
                      // eslint-disable-next-line @next/next/no-img-element
                      <img src={r.attachmentUrl} alt="" className="mt-1 max-h-40 w-auto rounded-md object-cover" />
                    )}
                  </div>
                </div>
              )
            })}
          </div>

          {/* Journal des actions de modération */}
          {ticket.actions.length > 0 && (
            <div className="flex flex-col gap-2">
              <h3 className="text-sm font-bold">{t('tickets.actions_title')}</h3>
              {ticket.actions.map((a, i) => (
                <div key={i} className="rounded-lg bg-muted/40 p-2 text-sm">
                  <div className="flex items-center gap-2 text-xs text-muted-foreground">
                    <span className="font-medium text-foreground">
                      {a.type === 'auto_reopen' || a.type === 'auto_hidden' ? t('tickets.system') : nameOf(a.moderatorId)}
                    </span>
                    <span>{timeAgo(a.createdAt, locale)}</span>
                  </div>
                  {a.type === 'reply' ? (
                    <p className="whitespace-pre-wrap break-words">{a.text}</p>
                  ) : a.type === 'status_change' ? (
                    <p className="text-muted-foreground">
                      {t('tickets.action.status_change', { status: t(`tickets.status.${a.status}`) })}
                    </p>
                  ) : a.type === 'auto_reopen' ? (
                    <p className="text-muted-foreground">{t('tickets.action.auto_reopen')}</p>
                  ) : a.type === 'content_removed' ? (
                    <p className="text-muted-foreground">{t('tickets.action.content_removed')}</p>
                  ) : a.type === 'auto_hidden' ? (
                    <p className="text-muted-foreground">{t('tickets.action.auto_hidden')}</p>
                  ) : a.type === 'approved' ? (
                    <p className="text-muted-foreground">{t('tickets.action.approved')}</p>
                  ) : a.type === 'warned' ? (
                    <p className="text-muted-foreground">{t('tickets.action.warned')}</p>
                  ) : (
                    <p className="text-muted-foreground">{t('tickets.action.transfer')}</p>
                  )}
                </div>
              ))}
            </div>
          )}

          {/* Réponse interne */}
          <div className="flex flex-col gap-2">
            <Textarea
              value={reply}
              onChange={(e) => setReply(e.target.value)}
              placeholder={t('tickets.reply_placeholder')}
              rows={2}
            />
            <Button size="sm" className="self-end" disabled={busy || !reply.trim()} onClick={() => void onReply()}>
              {busy ? <Loader2 className="h-4 w-4 animate-spin" /> : t('tickets.reply_submit')}
            </Button>
          </div>

          {/* Décision terminale : entité jugée conforme → re-signalement verrouillé. */}
          {ticket.status === 'approved' ? (
            <div className="flex items-center gap-2 rounded-xl border border-dashed bg-muted/40 p-3 text-sm text-muted-foreground">
              <Lock className="h-4 w-4 shrink-0" aria-hidden />
              {t('tickets.approved_locked')}
            </div>
          ) : (
          /* Actions de cycle de vie + modération */
          <div className="flex flex-wrap gap-2 border-t pt-3">
            {ticket.status === 'closed' ? (
              <Button variant="outline" size="sm" disabled={busy} onClick={() => void onStatus('reopened')}>
                {t('tickets.set_open')}
              </Button>
            ) : (
              <Button
                variant="outline"
                size="sm"
                disabled={busy}
                // Garde-fou : clôturer un ticket de modération SANS aucune sanction
                // demande confirmation (« non fondé ? ») ; sinon clôture directe.
                onClick={() =>
                  ticket.category === 'moderation' && !hasSanction
                    ? setCloseConfirmOpen(true)
                    : void onStatus('closed')
                }
              >
                {t('tickets.set_closed')}
              </Button>
            )}

            {/* Valider : l'entité est conforme (ne doit pas être signalée). Masqué
                dès qu'une sanction a été posée (retrait/avertissement) → cohérence. */}
            {ticket.category === 'moderation' && !hasSanction && (
              <Button variant="outline" size="sm" disabled={busy} onClick={() => void onApprove()} className="text-emerald-600">
                <CheckCircle2 className="mr-1.5 h-4 w-4" />
                {t('tickets.approve')}
              </Button>
            )}

            {authorId && (
              <Button variant="outline" size="sm" disabled={busy} onClick={() => openWarn()}>
                <ShieldAlert className="mr-1.5 h-4 w-4" />
                {t('tickets.warn')}
              </Button>
            )}

            {ticket.entityType === 'post' && !postMissing && !alreadyRemoved && (
              <Button variant="outline" size="sm" disabled={busy} onClick={openRemovalWarn} className="text-destructive">
                <Trash2 className="mr-1.5 h-4 w-4" />
                {t('post.delete')}
              </Button>
            )}

            {(ticket.entityType === 'message' || ticket.entityType === 'group_message') && !messageRemoved && !alreadyRemoved && (
              <Button variant="outline" size="sm" disabled={busy} onClick={openRemovalWarn} className="text-destructive">
                <Trash2 className="mr-1.5 h-4 w-4" />
                {t('tickets.delete_message')}
              </Button>
            )}

            {canTransfer && ticket.category === 'bug' && (
              <Button variant="default" size="sm" disabled={busy} onClick={() => void onTransfer()}>
                <ArrowRightLeft className="mr-1.5 h-4 w-4" />
                {t('tickets.transfer')}
              </Button>
            )}
          </div>
          )}
        </div>
      )}

      {authorId && (
        <WarnDialog
          open={warnOpen}
          onOpenChange={setWarnOpen}
          targetUserId={authorId}
          ticketId={ticket?.id}
          defaultMessage={warnDefault}
          // Mode « retrait » : la validation supprime + avertit + clôture, de façon
          // atomique (le contenu n'est touché qu'ici). Sinon, simple avertissement.
          title={pendingDelete ? t('tickets.delete_warn_title') : undefined}
          description={pendingDelete ? t('tickets.delete_warn_desc') : undefined}
          confirmLabel={pendingDelete ? t('tickets.delete_and_warn') : undefined}
          onConfirm={pendingDelete ? handleRemovalConfirm : undefined}
          onSubmitted={() => void load()}
        />
      )}

      {/* Garde-fou de clôture : aucune sanction posée → confirmer « non fondé ». */}
      <Dialog open={closeConfirmOpen} onOpenChange={setCloseConfirmOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('tickets.close_confirm_title')}</DialogTitle>
            <DialogDescription>{t('tickets.close_confirm_desc')}</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setCloseConfirmOpen(false)} disabled={busy}>
              {t('common.cancel')}
            </Button>
            <Button
              onClick={() => {
                setCloseConfirmOpen(false)
                void onStatus('closed')
              }}
              disabled={busy}
            >
              {t('tickets.close_confirm_submit')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Confirmation de bannissement de l'utilisateur ciblé. */}
      <Dialog open={banConfirmOpen} onOpenChange={(open) => !banning && setBanConfirmOpen(open)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('tickets.ban_confirm_title')}</DialogTitle>
            <DialogDescription>
              {t('tickets.ban_confirm_desc', { name: authorId ? nameOf(authorId) : '' })}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setBanConfirmOpen(false)} disabled={banning}>
              {t('common.cancel')}
            </Button>
            <Button variant="destructive" onClick={() => void onBan()} disabled={banning}>
              {banning ? <Loader2 className="h-4 w-4 animate-spin" aria-hidden /> : t('tickets.ban_user')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
