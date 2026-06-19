'use client'

import { useCallback, useEffect, useState } from 'react'

import {
  blockUser,
  getBlockedUserIds,
  unblockUser,
  type ApiError,
} from '@/lib/api'
import { getAccessToken } from '@/lib/auth-client'
import { useToast } from '@/hooks/use-toast'
import { useT } from '@/components/language-provider'

export const BLOCK_CHANGE_EVENT = 'breezy:block-change'

export interface BlockChangeDetail {
  userId: string
  blocked: boolean
}

export function useBlock(enabled = true) {
  const { toast } = useToast()
  const t = useT()
  const [blockedIds, setBlockedIds] = useState<Set<string>>(new Set())
  const [pending, setPending] = useState<Set<string>>(new Set())
  const [loaded, setLoaded] = useState(!enabled)

  useEffect(() => {
    if (!enabled || !getAccessToken()) {
      setLoaded(true)
      return
    }
    let cancelled = false
    setLoaded(false)
    getBlockedUserIds()
      .then((ids) => {
        if (!cancelled) setBlockedIds(ids)
      })
      .catch(() => {
        if (!cancelled) setBlockedIds(new Set())
      })
      .finally(() => {
        if (!cancelled) setLoaded(true)
      })
    return () => {
      cancelled = true
    }
  }, [enabled])

  useEffect(() => {
    if (!enabled) return undefined
    function onChange(event: Event) {
      const { userId, blocked } = (event as CustomEvent<BlockChangeDetail>).detail
      setBlockedIds((prev) => {
        const copy = new Set(prev)
        if (blocked) copy.add(userId)
        else copy.delete(userId)
        return copy
      })
    }
    window.addEventListener(BLOCK_CHANGE_EVENT, onChange)
    return () => window.removeEventListener(BLOCK_CHANGE_EVENT, onChange)
  }, [enabled])

  const toggle = useCallback(
    async (userId: string, next: boolean) => {
      setPending((prev) => new Set(prev).add(userId))
      setBlockedIds((prev) => {
        const copy = new Set(prev)
        if (next) copy.add(userId)
        else copy.delete(userId)
        return copy
      })
      emitBlockChange({ userId, blocked: next })
      try {
        if (next) await blockUser(userId)
        else await unblockUser(userId)
        toast({ title: next ? t('block.blocked') : t('block.unblocked') })
      } catch (err) {
        setBlockedIds((prev) => {
          const copy = new Set(prev)
          if (next) copy.delete(userId)
          else copy.add(userId)
          return copy
        })
        emitBlockChange({ userId, blocked: !next })
        toast({
          title: next ? t('block.block_failed') : t('block.unblock_failed'),
          description: (err as ApiError | Error).message,
          variant: 'destructive',
        })
      } finally {
        setPending((prev) => {
          const copy = new Set(prev)
          copy.delete(userId)
          return copy
        })
      }
    },
    [toast, t],
  )

  return {
    loaded,
    isBlocked: (userId: string) => blockedIds.has(userId),
    isPending: (userId: string) => pending.has(userId),
    toggle,
  }
}

export function emitBlockChange(detail: BlockChangeDetail) {
  if (typeof window === 'undefined') return
  window.dispatchEvent(new CustomEvent<BlockChangeDetail>(BLOCK_CHANGE_EVENT, { detail }))
}
