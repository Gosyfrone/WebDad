'use client'

import { useCallback, useEffect, useState } from 'react'

import { follow, getFollowingIds, getMe, unfollow } from '@/lib/api'
import type { RelationUser } from '@/types'
import { useToast } from '@/hooks/use-toast'

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
  const [currentUserId, setCurrentUserId] = useState<string | null>(null)
  const [followingIds, setFollowingIds] = useState<Set<string>>(new Set())
  const [pending, setPending] = useState<Set<string>>(new Set())

  useEffect(() => {
    if (!enabled) return
    let cancelled = false
    ;(async () => {
      try {
        const me = await getMe()
        if (cancelled) return
        setCurrentUserId(me.id)
        const ids = await getFollowingIds(me.id)
        if (!cancelled) setFollowingIds(ids)
      } catch {
        /* best-effort : sans session, currentUserId reste null, boutons « Suivre » */
      }
    })()
    return () => {
      cancelled = true
    }
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
        if (next) await follow(user.id)
        else await unfollow(user.id)
      } catch {
        setFollowingIds((prev) => {
          const copy = new Set(prev)
          if (next) copy.delete(user.id)
          else copy.add(user.id)
          return copy
        })
        toast({
          title: next ? 'Suivi impossible' : 'Désabonnement impossible',
          description: 'Connectez-vous pour gérer vos abonnements.',
          variant: 'destructive',
        })
      } finally {
        setPending((prev) => {
          const copy = new Set(prev)
          copy.delete(user.id)
          return copy
        })
      }
    },
    [toast],
  )

  return {
    currentUserId,
    isFollowing: (id: string) => followingIds.has(id),
    isPending: (id: string) => pending.has(id),
    toggle,
  }
}
