'use client'

import { useEffect, useState } from 'react'

import { apiFetch } from '@/lib/auth-client'
import { cn } from '@/lib/utils'

const POLL_MS = 10_000

type Presence = {
  lastLoginAt: string
  isOnline: boolean
}

type PresenceListener = (presence: Presence) => void

const cache = new Map<string, Presence>()
const listeners = new Map<string, Set<PresenceListener>>()
const timers = new Map<string, ReturnType<typeof setInterval>>()

export function ActivityPresenceDot({
  userId,
  initialLastLoginAt = '',
  initialIsOnline = false,
  className,
}: {
  userId?: string
  initialLastLoginAt?: string
  initialIsOnline?: boolean
  className?: string
}) {
  const [presence, setPresence] = useState<Presence>({
    lastLoginAt: initialLastLoginAt,
    isOnline: initialIsOnline,
  })

  useEffect(() => {
    if (!userId) return
    const initial = cache.get(userId) ?? {
      lastLoginAt: initialLastLoginAt,
      isOnline: initialIsOnline,
    }
    cache.set(userId, initial)
    setPresence(initial)
    return subscribePresence(userId, setPresence)
  }, [initialIsOnline, initialLastLoginAt, userId])

  const visible = presence.isOnline || Boolean(presence.lastLoginAt)
  if (!visible) return null

  return (
    <span
      aria-hidden
      className={cn(
        'pointer-events-none absolute -bottom-0.5 -left-0.5 z-20 h-3.5 w-3.5 rounded-full border-2 border-background shadow-[0_0_0_1px_rgba(15,23,42,0.18)]',
        presence.isOnline ? 'bg-emerald-500' : 'bg-slate-400',
        className,
      )}
    />
  )
}

function subscribePresence(userId: string, listener: PresenceListener): () => void {
  let set = listeners.get(userId)
  if (!set) {
    set = new Set()
    listeners.set(userId, set)
  }
  set.add(listener)

  if (!timers.has(userId)) {
    void refreshPresence(userId)
    timers.set(userId, setInterval(() => void refreshPresence(userId), POLL_MS))
  }

  return () => {
    const current = listeners.get(userId)
    current?.delete(listener)
    if (current && current.size === 0) {
      listeners.delete(userId)
      const timer = timers.get(userId)
      if (timer) clearInterval(timer)
      timers.delete(userId)
    }
  }
}

async function refreshPresence(userId: string): Promise<void> {
  try {
    const res = await apiFetch(`/profils/${encodeURIComponent(userId)}/activity`)
    if (!res.ok) return
    const payload = (await res.json().catch(() => null)) as {
      last_login_at?: string | null
      is_online?: boolean
    } | null
    const next: Presence = {
      lastLoginAt: payload?.last_login_at ?? '',
      isOnline: Boolean(payload?.is_online),
    }
    cache.set(userId, next)
    listeners.get(userId)?.forEach((listener) => listener(next))
  } catch {
    // Best-effort : l'état courant reste affiché jusqu'au prochain polling.
  }
}
