'use client'

import { useCallback, useEffect, useState } from 'react'
import { Loader2, MoreHorizontal, Search } from 'lucide-react'

import {
  hardDeleteUser,
  listAdminUsers,
  setUserBanned,
  updateUserRole,
  type AdminUser,
} from '@/lib/admin'
import { useCurrentUser } from '@/components/current-user-provider'
import { cn, initialOf, timeAgo } from '@/lib/utils'
import { ProfilLink } from '@/components/profil/profil-link'
import { ActivityPresenceDot } from '@/components/profil/activity-presence-dot'
import { useLanguage } from '@/components/language-provider'
import { useToast } from '@/hooks/use-toast'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import type { UserRole } from '@/types'

const ROLES: UserRole[] = ['user', 'moderator', 'administrator']

interface AccountsPanelProps {
  /**
   * Gouvernance = l'acteur est administrateur. Débloque le changement de rôle et
   * (à terme) la suppression définitive de compte (RGPD). Un modérateur (false)
   * ne voit ni les rôles ni l'effacement, et ne peut bannir QUE des utilisateurs
   * simples (le back refuse de toute façon un mod ciblant un mod/admin).
   */
  canGovern: boolean
}

/**
 * Annuaire des comptes, partagé par la Modération (mod + admin). C'est l'ancien
 * panneau d'administration, généralisé : la BASE reste l'auth-service
 * (`GET /auth/users`, autorité du rôle/état), enrichie côté front. Les boutons
 * sont filtrés par `canGovern` (cf. ci-dessus). La garde d'accès vit dans le
 * parent (ModerationView) ; le back renvoie 403 de toute façon.
 */
// ~5 ans − 30 j (en ms) : seuil d'« bientôt purgé » pour un compte banni, calé
// sur les défauts serveur (ACCOUNT_PURGE_AFTER − PURGE_WARN_BEFORE).
const ACCOUNT_PURGE_WARN_MS = (5 * 365 - 30) * 24 * 60 * 60 * 1000

type AccountFilter = 'all' | 'banned' | 'moderators' | 'admins'

export function AccountsPanel({ canGovern }: AccountsPanelProps) {
  const { t, locale } = useLanguage()
  const { toast } = useToast()
  const { session } = useCurrentUser()
  const [query, setQuery] = useState('')
  const [filter, setFilter] = useState<AccountFilter>('all')
  const [users, setUsers] = useState<AdminUser[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(false)
  const [busyId, setBusyId] = useState<string | null>(null)
  // Effacement RGPD : compte ciblé + texte de confirmation (re-saisie du username).
  const [eraseTarget, setEraseTarget] = useState<AdminUser | null>(null)
  const [confirmText, setConfirmText] = useState('')
  const [erasing, setErasing] = useState(false)

  const load = useCallback(async (q: string) => {
    setLoading(true)
    setError(false)
    try {
      setUsers(await listAdminUsers(q))
    } catch {
      setError(true)
    } finally {
      setLoading(false)
    }
  }, [])

  // Recherche debouncée (300 ms).
  useEffect(() => {
    const handle = setTimeout(() => void load(query.trim()), 300)
    return () => clearTimeout(handle)
  }, [query, load])

  async function onChangeRole(user: AdminUser, role: UserRole) {
    if (role === user.role) return
    setBusyId(user.id)
    try {
      await updateUserRole(user.id, role)
      setUsers((prev) => prev.map((u) => (u.id === user.id ? { ...u, role } : u)))
      toast({ title: t('admin.role_changed', { role: t(`role.${role}`) }) })
    } catch {
      toast({ title: t('admin.action_failed'), variant: 'destructive' })
    } finally {
      setBusyId(null)
    }
  }

  async function onToggleBan(user: AdminUser) {
    const banned = user.isActive // actif → on bannit
    setBusyId(user.id)
    try {
      await setUserBanned(user.id, banned)
      setUsers((prev) =>
        prev.map((u) => (u.id === user.id ? { ...u, isActive: !banned } : u)),
      )
      toast({ title: banned ? t('admin.banned_toast') : t('admin.unbanned_toast') })
    } catch {
      toast({ title: t('admin.action_failed'), variant: 'destructive' })
    } finally {
      setBusyId(null)
    }
  }

  function openErase(user: AdminUser) {
    setEraseTarget(user)
    setConfirmText('')
  }

  async function onConfirmErase() {
    if (!eraseTarget) return
    setErasing(true)
    try {
      await hardDeleteUser(eraseTarget.id)
      setUsers((prev) => prev.filter((u) => u.id !== eraseTarget.id))
      toast({ title: t('moderation.account_deleted_toast') })
      setEraseTarget(null)
    } catch {
      toast({ title: t('moderation.account_delete_failed'), variant: 'destructive' })
    } finally {
      setErasing(false)
    }
  }

  // Filtre par statut (cumulé à la recherche serveur, appliqué côté client sur
  // la page chargée) : tous / bannis / modérateurs / administrateurs.
  const visibleUsers = users.filter((u) => {
    switch (filter) {
      case 'banned':
        return !u.isActive
      case 'moderators':
        return u.role === 'moderator'
      case 'admins':
        return u.role === 'administrator'
      default:
        return true
    }
  })

  const filters: { key: AccountFilter; label: string }[] = [
    { key: 'all', label: t('accounts.filter_all') },
    { key: 'banned', label: t('accounts.filter_banned') },
    { key: 'moderators', label: t('accounts.filter_moderators') },
    { key: 'admins', label: t('accounts.filter_admins') },
  ]

  return (
    <div className="px-4 py-4">
      <div className="relative mb-3">
        <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" aria-hidden />
        <Input
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder={t('admin.search_placeholder')}
          className="pl-9"
          aria-label={t('admin.search_placeholder')}
        />
      </div>

      {/* Filtres de statut */}
      <div className="mb-4 flex flex-wrap gap-2">
        {filters.map((f) => (
          <button
            key={f.key}
            type="button"
            onClick={() => setFilter(f.key)}
            className={cn(
              'rounded-full border px-3 py-1 text-xs font-semibold transition-colors',
              filter === f.key
                ? 'border-[#5B6CFF] bg-[#5B6CFF]/10 text-[#5B6CFF]'
                : 'border-border text-muted-foreground hover:text-foreground',
            )}
          >
            {f.label}
          </button>
        ))}
      </div>

      {loading ? (
        <div className="flex items-center justify-center py-16 text-muted-foreground">
          <Loader2 className="h-5 w-5 animate-spin" aria-hidden />
        </div>
      ) : error ? (
        <p className="py-16 text-center text-sm text-muted-foreground">{t('admin.error')}</p>
      ) : visibleUsers.length === 0 ? (
        <p className="py-16 text-center text-sm text-muted-foreground">{t('admin.empty')}</p>
      ) : (
        <ul className="flex flex-col gap-2">
          {visibleUsers.map((user) => {
            const isSelf = user.id === session?.userId
            const busy = busyId === user.id
            // Un modérateur ne peut bannir qu'un utilisateur simple (le back le
            // refuse aussi). Un admin agit sur tout le monde sauf lui-même.
            const banAllowed = !isSelf && (canGovern || user.role === 'user')
            // Gouvernance (rôle + effacement) = admin, sauf sur soi-même.
            const canChangeRole = canGovern && !isSelf
            const canErase = canGovern && !isSelf
            // Au moins une action disponible → on affiche la bulle « … ».
            const hasActions = canChangeRole || banAllowed || canErase
            const initial = initialOf(user.displayName, user.username || user.email)
            return (
              <li
                key={user.id}
                className="panel flex items-center gap-3 rounded-2xl border p-3 shadow-sm"
              >
                <ProfilLink author={{ id: user.id, username: user.username }} className="shrink-0">
                  <Avatar className="h-10 w-10">
                    {user.avatarUrl && <AvatarImage src={user.avatarUrl} alt={user.displayName} />}
                    <AvatarFallback>{initial}</AvatarFallback>
                    <ActivityPresenceDot userId={user.id} />
                  </Avatar>
                </ProfilLink>

                <div className="flex min-w-0 flex-1 flex-col">
                  <span className="flex items-center gap-2 truncate text-sm font-bold">
                    {user.displayName || user.username || t('common.user')}
                    {isSelf && (
                      <Badge variant="outline" className="text-[10px]">
                        {t('admin.you')}
                      </Badge>
                    )}
                  </span>
                  <span className="truncate text-xs text-muted-foreground">
                    {user.username ? `@${user.username} · ` : ''}
                    {user.email}
                  </span>
                  {!user.isActive && user.deactivatedAt && (
                    <span className="flex items-center gap-1.5 text-xs text-muted-foreground">
                      {t('moderation.banned_since', { when: timeAgo(user.deactivatedAt, locale) })}
                      {Date.now() - new Date(user.deactivatedAt).getTime() >= ACCOUNT_PURGE_WARN_MS && (
                        <Badge variant="destructive" className="text-[10px]">
                          {t('moderation.expiring_soon')}
                        </Badge>
                      )}
                    </span>
                  )}
                </div>

                {/* Rôle élevé en badge (utilisateur simple → juste le statut). */}
                {user.role !== 'user' && (
                  <Badge variant="outline" className="shrink-0 text-[10px]">
                    {t(`role.${user.role}`)}
                  </Badge>
                )}

                {/* État du compte */}
                <Badge variant={user.isActive ? 'secondary' : 'destructive'} className="shrink-0">
                  {user.isActive ? t('admin.status_active') : t('admin.status_banned')}
                </Badge>

                {/* Toutes les actions repliées dans une bulle « … » → ligne compacte
                    sur mobile comme desktop (plus de débordement à droite). */}
                {hasActions && (
                  <DropdownMenu modal={false}>
                    <DropdownMenuTrigger asChild>
                      <Button
                        variant="ghost"
                        size="icon"
                        disabled={busy}
                        className="h-8 w-8 shrink-0"
                        aria-label={t('accounts.actions')}
                      >
                        {busy ? (
                          <Loader2 className="h-4 w-4 animate-spin" aria-hidden />
                        ) : (
                          <MoreHorizontal className="h-4 w-4" aria-hidden />
                        )}
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end" className="panel border z-40">
                      {/* Changement de rôle — gouvernance (admin) */}
                      {canChangeRole && (
                        <DropdownMenuSub>
                          <DropdownMenuSubTrigger>{t('admin.change_role')}</DropdownMenuSubTrigger>
                          <DropdownMenuSubContent className="panel border">
                            {ROLES.map((role) => (
                              <DropdownMenuItem
                                key={role}
                                disabled={role === user.role}
                                onSelect={() => void onChangeRole(user, role)}
                              >
                                {t(`role.${role}`)}
                              </DropdownMenuItem>
                            ))}
                          </DropdownMenuSubContent>
                        </DropdownMenuSub>
                      )}

                      {/* Bannir / réactiver — modération (mod + admin, cibles bornées) */}
                      {banAllowed && (
                        <DropdownMenuItem onSelect={() => void onToggleBan(user)}>
                          {user.isActive ? t('admin.action_ban') : t('admin.action_unban')}
                        </DropdownMenuItem>
                      )}

                      {/* Suppression définitive du compte (RGPD) — gouvernance admin,
                          effacement cross-service avec confirmation par re-saisie. */}
                      {canErase && (
                        <>
                          {(canChangeRole || banAllowed) && <DropdownMenuSeparator />}
                          <DropdownMenuItem
                            onSelect={() => openErase(user)}
                            className="text-destructive focus:text-destructive"
                          >
                            {t('moderation.account_delete')}
                          </DropdownMenuItem>
                        </>
                      )}
                    </DropdownMenuContent>
                  </DropdownMenu>
                )}
              </li>
            )
          })}
        </ul>
      )}

      <p className="mt-4 text-xs text-muted-foreground">{t('admin.role_note')}</p>

      {/* Confirmation d'effacement RGPD : re-saisie exacte du username. */}
      <Dialog
        open={eraseTarget !== null}
        onOpenChange={(open) => {
          if (!open) setEraseTarget(null)
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('moderation.account_delete_title')}</DialogTitle>
            <DialogDescription>
              {t('moderation.account_delete_desc', { username: eraseTarget?.username ?? '' })}
            </DialogDescription>
          </DialogHeader>
          <Input
            value={confirmText}
            onChange={(e) => setConfirmText(e.target.value)}
            placeholder={eraseTarget?.username ?? ''}
            aria-label={t('moderation.account_delete_confirm_label')}
            autoComplete="off"
          />
          <DialogFooter>
            <Button variant="outline" onClick={() => setEraseTarget(null)}>
              {t('common.cancel')}
            </Button>
            <Button
              variant="destructive"
              disabled={erasing || !eraseTarget?.username || confirmText !== eraseTarget.username}
              onClick={() => void onConfirmErase()}
            >
              {erasing ? (
                <Loader2 className="h-4 w-4 animate-spin" aria-hidden />
              ) : (
                t('moderation.account_delete')
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
