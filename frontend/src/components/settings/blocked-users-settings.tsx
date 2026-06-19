'use client'

import { useEffect, useState } from 'react'
import Link from 'next/link'
import { Loader2, UserCheck, UserX } from 'lucide-react'

import { useT } from '@/components/language-provider'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import { useToast } from '@/hooks/use-toast'
import { getBlockedUserIds, getUserById, unblockUser } from '@/lib/api'
import { profilHref } from '@/lib/routes'
import { emitBlockChange } from '@/lib/use-block'
import { initialOf } from '@/lib/utils'
import type { RelationUser } from '@/types'

export function BlockedUsersSettings() {
  const t = useT()
  const { toast } = useToast()
  const [users, setUsers] = useState<RelationUser[]>([])
  const [loading, setLoading] = useState(true)
  const [pending, setPending] = useState<Set<string>>(new Set())

  useEffect(() => {
    let cancelled = false

    async function loadBlockedUsers() {
      try {
        const ids = await getBlockedUserIds()
        const resolved = await Promise.all([...ids].map((id) => getUserById(id)))
        if (!cancelled) setUsers(resolved.filter((user): user is RelationUser => user !== null))
      } catch {
        if (!cancelled) {
          toast({ title: t('block.list_failed'), variant: 'destructive' })
        }
      } finally {
        if (!cancelled) setLoading(false)
      }
    }

    void loadBlockedUsers()
    return () => {
      cancelled = true
    }
  }, [toast, t])

  async function handleUnblock(user: RelationUser) {
    setPending((prev) => new Set(prev).add(user.id))
    try {
      await unblockUser(user.id)
      setUsers((prev) => prev.filter((item) => item.id !== user.id))
      emitBlockChange({ userId: user.id, blocked: false })
      toast({ title: t('block.unblocked') })
    } catch {
      toast({ title: t('block.unblock_failed'), variant: 'destructive' })
    } finally {
      setPending((prev) => {
        const copy = new Set(prev)
        copy.delete(user.id)
        return copy
      })
    }
  }

  if (loading) {
    return (
      <div className="panel flex min-h-24 items-center justify-center rounded-xl border px-3 py-3 shadow-sm">
        <Loader2 className="h-5 w-5 animate-spin text-primary" aria-hidden />
      </div>
    )
  }

  return (
    <div className="panel rounded-xl border px-3 py-3 shadow-sm">
      {users.length === 0 ? (
        <div className="flex items-start gap-3 rounded-lg px-1 py-1">
          <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-primary/10 text-primary">
            <UserCheck className="h-4 w-4" aria-hidden />
          </span>
          <p className="text-sm text-muted-foreground">{t('block.empty')}</p>
        </div>
      ) : (
        <div className="divide-y">
          {users.map((user) => {
            const busy = pending.has(user.id)
            return (
              <div key={user.id} className="flex items-center gap-3 py-3 first:pt-0 last:pb-0">
                <Link href={profilHref(user.username)} className="flex min-w-0 flex-1 items-center gap-3 rounded-lg transition hover:opacity-85">
                  <Avatar className="h-10 w-10 shrink-0">
                    {user.avatarUrl && <AvatarImage src={user.avatarUrl} alt="" />}
                    <AvatarFallback className="font-bold">
                      {initialOf(user.displayName, user.username)}
                    </AvatarFallback>
                  </Avatar>
                  <span className="min-w-0 flex-1">
                    <span className="block truncate text-sm font-semibold text-foreground">{user.displayName}</span>
                    <span className="block truncate text-xs text-muted-foreground">@{user.username}</span>
                  </span>
                </Link>
                <Button
                  type="button"
                  size="sm"
                  variant="outline"
                  disabled={busy}
                  onClick={() => void handleUnblock(user)}
                  className="shrink-0 rounded-full"
                >
                  {busy ? <Loader2 className="mr-2 h-3.5 w-3.5 animate-spin" /> : null}
                  {t('block.unblock_short')}
                </Button>
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}
