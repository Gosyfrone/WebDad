'use client'

import { useEffect, useState } from 'react'
import { Loader2 } from 'lucide-react'

import { getSuggestions } from '@/lib/api'
import { useFollow } from '@/lib/use-follow'
import type { RelationUser } from '@/types'
import { useT } from '@/components/language-provider'
import { UserListItem } from '@/components/profil/user-list-item'

/** Nombre de suggestions affichées dans la carte. */
const VISIBLE = 3

/**
 * Carte « Qui suivre » : comptes les plus suivis (user-service), enrichis du
 * décoratif, hors soi-même et hors comptes déjà suivis. Rendu compact (sans
 * bio) via `UserListItem` ; bouton Suivre branché par `useFollow`.
 */
export function WhoToFollow() {
  const t = useT()
  const [users, setUsers] = useState<RelationUser[]>([])
  const [visibleUsers, setVisibleUsers] = useState<RelationUser[] | null>(null)
  const [loading, setLoading] = useState(true)

  const { currentUserId, loaded: followLoaded, isFollowing, isRequested, isPending, toggle } = useFollow()

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

  useEffect(() => {
    if (loading || !followLoaded || visibleUsers !== null) return
    setVisibleUsers(
      users
        .filter((u) => u.id !== currentUserId && !isFollowing(u.id) && !isRequested(u.id))
        .slice(0, VISIBLE),
    )
  }, [currentUserId, followLoaded, isFollowing, isRequested, loading, users, visibleUsers])

  const visible = visibleUsers ?? []

  return (
    <div className="glass overflow-hidden rounded-[24px] border backdrop-blur-xl">
      <h2 className="brand-text px-4 pb-1.5 pt-2.5 text-xl font-bold leading-tight">{t('who.title')}</h2>

      {loading || !followLoaded ? (
        <div className="flex justify-center py-6">
          <Loader2 className="h-5 w-5 animate-spin text-[#5B6CFF] dark:text-[#9aa6ff]" />
        </div>
      ) : visible.length === 0 ? (
        <p className="px-4 pb-4 text-sm text-muted-foreground">
          {t('who.empty')}
        </p>
      ) : (
        <div className="px-2 pb-2">
          {visible.map((user) => (
            <div key={user.id} className="min-w-0">
              <UserListItem
                user={user}
                isFollowing={isFollowing(user.id)}
                isRequested={isRequested(user.id)}
                isSelf={user.id === currentUserId}
                pending={isPending(user.id)}
                showBio={false}
                compact
                onToggleFollow={toggle}
              />
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
