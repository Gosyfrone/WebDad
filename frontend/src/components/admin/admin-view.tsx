'use client'

import { useCallback, useEffect, useState } from 'react'
import { Loader2, Search, ShieldAlert } from 'lucide-react'

import {
  listAdminUsers,
  setUserBanned,
  updateUserRole,
  type AdminUser,
} from '@/lib/admin'
import { useSession } from '@/lib/session'
import { ProfilLink } from '@/components/profil/profil-link'
import { useT } from '@/components/language-provider'
import { useToast } from '@/hooks/use-toast'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import type { UserRole } from '@/types'

const ROLES: UserRole[] = ['user', 'moderator', 'administrator']

/**
 * Panneau d'administration : annuaire des comptes (base auth-service enrichie),
 * changement de rôle et bannissement/réactivation. Garde côté client (le back
 * renvoie 403 de toute façon) : refuse l'accès si le rôle n'est pas admin.
 */
export function AdminView() {
  const t = useT()
  const { toast } = useToast()
  const session = useSession()
  const [query, setQuery] = useState('')
  const [users, setUsers] = useState<AdminUser[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(false)
  const [busyId, setBusyId] = useState<string | null>(null)

  const isAdmin = session?.role === 'administrator'

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

  // Recherche debouncée (300 ms). Ne charge rien tant qu'on n'est pas admin
  // (session lue au montage : `undefined` → null/role en effet).
  useEffect(() => {
    if (!isAdmin) return
    const handle = setTimeout(() => void load(query.trim()), 300)
    return () => clearTimeout(handle)
  }, [query, isAdmin, load])

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

  if (session && !isAdmin) {
    return (
      <div className="flex flex-col items-center justify-center gap-3 px-4 py-20 text-center">
        <ShieldAlert className="h-10 w-10 text-muted-foreground" aria-hidden />
        <h1 className="text-xl font-bold">{t('admin.access_denied')}</h1>
        <p className="max-w-sm text-sm text-muted-foreground">{t('admin.access_denied_desc')}</p>
      </div>
    )
  }

  return (
    <div className="px-4 py-4">
      <header className="mb-4">
        <h1 className="text-xl font-bold">{t('nav.admin')}</h1>
        <p className="text-sm text-muted-foreground">{t('admin.subtitle')}</p>
      </header>

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
                </div>

                {/* État du compte */}
                <Badge variant={user.isActive ? 'secondary' : 'destructive'} className="shrink-0">
                  {user.isActive ? t('admin.status_active') : t('admin.status_banned')}
                </Badge>

                {/* Changement de rôle (interdit sur soi) */}
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

                {/* Bannir / réactiver (interdit sur soi) */}
                <Button
                  variant={user.isActive ? 'destructive' : 'outline'}
                  size="sm"
                  disabled={isSelf || busy}
                  onClick={() => void onToggleBan(user)}
                  className="shrink-0"
                >
                  {busy ? (
                    <Loader2 className="h-4 w-4 animate-spin" aria-hidden />
                  ) : user.isActive ? (
                    t('admin.action_ban')
                  ) : (
                    t('admin.action_unban')
                  )}
                </Button>
              </li>
            )
          })}
        </ul>
      )}

      <p className="mt-4 text-xs text-muted-foreground">{t('admin.role_note')}</p>
    </div>
  )
}
