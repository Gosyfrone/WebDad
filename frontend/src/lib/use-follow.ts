'use client'

import { useCallback, useEffect, useState } from 'react'

import { follow, getFollowingIds, getMe, getPendingFollowRequestIds, unfollow } from '@/lib/api'
import { subscribeFollowRequestDecision } from '@/lib/notifications'
import type { RelationUser } from '@/types'
import { useToast } from '@/hooks/use-toast'
import { useT } from '@/components/language-provider'

export type FollowToggleResult = 'following' | 'pending' | 'unfollowed' | 'failed'

/**
 * État de suivi partagé (modale des relations, Explorer, « Qui suivre »).
 *
 * Charge l'utilisateur courant + l'ensemble de ses abonnements (l'API n'expose
 * pas de flag `is_following`) pour initialiser correctement les boutons, et
 * fournit un `toggle` optimiste (rollback + toast en cas d'échec).
 *
 * `enabled=false` diffère le chargement (ex. modale fermée).
 */
export function useFollow(enabled = true) {
  const { toast } = useToast()
  const t = useT()
  const [currentUserId, setCurrentUserId] = useState<string | null>(null)
  const [followingIds, setFollowingIds] = useState<Set<string>>(new Set())
  const [requestedIds, setRequestedIds] = useState<Set<string>>(new Set())
  const [pending, setPending] = useState<Set<string>>(new Set())
  const [loaded, setLoaded] = useState(!enabled)

  useEffect(() => {
    if (!enabled) {
      setLoaded(true)
      return
    }
    let cancelled = false
    setLoaded(false)
    ;(async () => {
      try {
        const me = await getMe()
        if (cancelled) return
        setCurrentUserId(me.id)
        const ids = await getFollowingIds(me.id)
        const pendingIds = await getPendingFollowRequestIds()
        if (!cancelled) {
          setFollowingIds(ids)
          setRequestedIds(pendingIds)
        }
      } catch {
        /* best-effort : sans session, currentUserId reste null, boutons « Suivre » */
      } finally {
        if (!cancelled) setLoaded(true)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [enabled])

  useEffect(() => {
    if (!enabled) return undefined
    return subscribeFollowRequestDecision(({ actorId, status }) => {
      setRequestedIds((prev) => {
        const copy = new Set(prev)
        copy.delete(actorId)
        return copy
      })
      setFollowingIds((prev) => {
        const copy = new Set(prev)
        if (status === 'accepted') copy.add(actorId)
        else copy.delete(actorId)
        return copy
      })
    })
  }, [enabled])

  const toggle = useCallback(
    async (user: RelationUser, next: boolean) => {
      setPending((prev) => new Set(prev).add(user.id))
      setFollowingIds((prev) => {
        const copy = new Set(prev)
        if (next) copy.add(user.id)
        else copy.delete(user.id)
        return copy
      })
      try {
        if (next) {
          const status = await follow(user.id)
          setFollowingIds((prev) => {
            const copy = new Set(prev)
            if (status === 'following') copy.add(user.id)
            else copy.delete(user.id)
            return copy
          })
          setRequestedIds((prev) => {
            const copy = new Set(prev)
            if (status === 'pending') copy.add(user.id)
            else copy.delete(user.id)
            return copy
          })
          return status
        }
        await unfollow(user.id)
        setRequestedIds((prev) => {
          const copy = new Set(prev)
          copy.delete(user.id)
          return copy
        })
        return 'unfollowed'
      } catch {
        setFollowingIds((prev) => {
          const copy = new Set(prev)
          if (next) copy.delete(user.id)
          else copy.add(user.id)
          return copy
        })
        toast({
          title: next ? t('follow.fail_title') : t('follow.unfail_title'),
          description: t('follow.fail_desc'),
          variant: 'destructive',
        })
        return 'failed'
      } finally {
        setPending((prev) => {
          const copy = new Set(prev)
          copy.delete(user.id)
          return copy
        })
      }
    },
    [toast, t],
  )

  return {
    currentUserId,
    loaded,
    isFollowing: (id: string) => followingIds.has(id),
    isRequested: (id: string) => requestedIds.has(id),
    isPending: (id: string) => pending.has(id),
    toggle,
  }
}
