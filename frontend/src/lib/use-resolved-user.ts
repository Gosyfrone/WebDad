'use client'

import { useEffect, useState } from 'react'

import { resolveUser, type ResolvedUser } from '@/lib/user-cache'

/**
 * Résout l'identité (username + décoratif) d'un `userId` pour l'affichage.
 * Renvoie `null` tant que la résolution n'est pas terminée (ou si `userId` est
 * vide). S'appuie sur le cache mémoïsé de `user-cache` (pas de refetch répété).
 */
export function useResolvedUser(userId: string | null | undefined): ResolvedUser | null {
  const [user, setUser] = useState<ResolvedUser | null>(null)

  useEffect(() => {
    if (!userId) {
      setUser(null)
      return
    }
    let cancelled = false
    resolveUser(userId).then((u) => {
      if (!cancelled) setUser(u)
    })
    return () => {
      cancelled = true
    }
  }, [userId])

  return user
}
