'use client'

import { useCallback, useEffect, useState } from 'react'
import { Loader2, Search } from 'lucide-react'

import {
  hardDeleteUser,
  listAdminUsers,
  setUserBanned,
  updateUserRole,
  type AdminUser,
} from '@/lib/admin'
import { useSession } from '@/lib/session'
import { timeAgo } from '@/lib/utils'
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

export function AccountsPanel({ canGovern }: AccountsPanelProps) {
  const { t, locale } = useLanguage()
  const { toast } = useToast()
  const session = useSession()
  const [query, setQuery] = useState('')
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

  return (
    <div className="px-4 py-4">
      <div className="relative mb-4">
        <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" aria-hidden />
        <Input
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder={t('admin.search_placeholder')}
          className="pl-9"
          aria-label={t('admin.search_placeholder')}
        />
      </div>

      {loading ? (
        <div className="flex items-center justify-center py-16 text-muted-foreground">
          <Loader2 className="h-5 w-5 animate-spin" aria-hidden />
        </div>
      ) : error ? (
        <p className="py-16 text-center text-sm text-muted-foreground">{t('admin.error')}</p>
      ) : users.length === 0 ? (
        <p className="py-16 text-center text-sm text-muted-foreground">{t('admin.empty')}</p>
      ) : (
        <ul className="flex flex-col gap-2">
          {users.map((user) => {
            const isSelf = user.id === session?.userId
            const busy = busyId === user.id
            // Un modérateur ne peut bannir qu'un utilisateur simple (le back le
            // refuse aussi). Un admin agit sur tout le monde sauf lui-même.
            const banAllowed = !isSelf && (canGovern || user.role === 'user')
            const initial = (user.displayName || user.username || user.email || 'U')
              .charAt(0)
              .toUpperCase()
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

                {/* État du compte */}
                <Badge variant={user.isActive ? 'secondary' : 'destructive'} className="shrink-0">
                  {user.isActive ? t('admin.status_active') : t('admin.status_banned')}
                </Badge>

                {/* Changement de rôle — gouvernance (admin uniquement) */}
                {canGovern && (
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <Button
                        variant="outline"
                        size="sm"
                        disabled={isSelf || busy}
                        className="shrink-0"
                      >
                        {t(`role.${user.role}`)}
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end" className="panel border">
                      {ROLES.map((role) => (
                        <DropdownMenuItem
                          key={role}
                          disabled={role === user.role}
                          onSelect={() => void onChangeRole(user, role)}
                        >
                          {t(`role.${role}`)}
                        </DropdownMenuItem>
                      ))}
                    </DropdownMenuContent>
                  </DropdownMenu>
                )}

                {/* Bannir / réactiver — modération (mod + admin, cibles bornées) */}
                <Button
                  variant={user.isActive ? 'destructive' : 'outline'}
                  size="sm"
                  disabled={!banAllowed || busy}
                  onClick={() => void onToggleBan(user)}
                  className="shrink-0"
                  title={!banAllowed && !isSelf ? t('moderation.ban_restricted') : undefined}
                >
                  {busy ? (
                    <Loader2 className="h-4 w-4 animate-spin" aria-hidden />
                  ) : user.isActive ? (
                    t('admin.action_ban')
                  ) : (
                    t('admin.action_unban')
                  )}
                </Button>

                {/* Suppression définitive du compte (RGPD) — gouvernance admin,
                    effacement cross-service avec confirmation par re-saisie. */}
                {canGovern && (
                  <Button
                    variant="ghost"
                    size="sm"
                    disabled={isSelf || busy}
                    onClick={() => openErase(user)}
                    className="shrink-0 text-destructive hover:text-destructive"
                    title={t('moderation.account_delete')}
                  >
                    {t('moderation.account_delete')}
                  </Button>
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
