'use client'

import { useEffect, useState } from 'react'
import { Loader2 } from 'lucide-react'

import { getSuggestions } from '@/lib/api'
import { useFollow } from '@/lib/use-follow'
import type { RelationUser } from '@/types'
import { UserListItem } from '@/components/profil/user-list-item'

/** Nombre de suggestions affichées dans la carte. */
const VISIBLE = 3

/**
 * Carte « Qui suivre » : comptes les plus suivis (user-service), enrichis du
 * décoratif, hors soi-même et hors comptes déjà suivis. Rendu compact (sans
 * bio) via `UserListItem` ; bouton Suivre branché par `useFollow`.
 */
export function WhoToFollow() {
  const [users, setUsers] = useState<RelationUser[]>([])
  const [loading, setLoading] = useState(true)

  const { currentUserId, isFollowing, isPending, toggle } = useFollow()

  useEffect(() => {
    let cancelled = false
    getSuggestions(10)
      .then((u) => {
        if (!cancelled) setUsers(u)
      })
      .catch(() => {
        /* best-effort : carte vide si la requête échoue */
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [])

  const visible = users
    .filter((u) => u.id !== currentUserId && !isFollowing(u.id))
    .slice(0, VISIBLE)

  return (
    <div className="glass overflow-hidden rounded-[24px] border backdrop-blur-xl">
      <h2 className="brand-text px-4 py-3 text-xl font-bold">Qui suivre</h2>

      {loading ? (
        <div className="flex justify-center py-6">
          <Loader2 className="h-5 w-5 animate-spin text-[#5B6CFF] dark:text-[#9aa6ff]" />
        </div>
      ) : visible.length === 0 ? (
        <p className="px-4 pb-4 text-sm text-muted-foreground">
          Aucune suggestion pour le moment.
        </p>
      ) : (
        <div className="divide-y divide-border">
          {visible.map((user) => (
            <UserListItem
              key={user.id}
              user={user}
              isFollowing={isFollowing(user.id)}
              isSelf={user.id === currentUserId}
              pending={isPending(user.id)}
              showBio={false}
              onToggleFollow={toggle}
            />
          ))}
        </div>
      )}
    </div>
  )
}
