'use client'

import { useEffect, useMemo, useState } from 'react'

import { useLanguage } from '@/components/language-provider'
import { apiFetch } from '@/lib/auth-client'
import { cn } from '@/lib/utils'

const POLL_MS = 10_000

export function ActivityStatus({
  userId,
  lastLoginAt,
  initialLastLoginAt,
  initialIsOnline = false,
  className,
}: {
  userId?: string
  lastLoginAt?: string
  initialLastLoginAt?: string
  initialIsOnline?: boolean
  className?: string
}) {
  const { t, locale } = useLanguage()
  const [activity, setActivity] = useState({
    lastLoginAt: initialLastLoginAt ?? lastLoginAt ?? '',
    isOnline: initialIsOnline,
  })

  useEffect(() => {
    setActivity({
      lastLoginAt: initialLastLoginAt ?? lastLoginAt ?? '',
      isOnline: initialIsOnline,
    })
  }, [initialIsOnline, initialLastLoginAt, lastLoginAt])

  useEffect(() => {
    if (!userId) return
    const targetUserId = userId
    let cancelled = false

    async function refresh() {
      try {
        const res = await apiFetch(
          `/profils/${encodeURIComponent(targetUserId)}/activity`,
        )
        if (!res.ok) return
        const payload = (await res.json().catch(() => null)) as {
          last_login_at?: string | null
          is_online?: boolean
        } | null
        if (!cancelled) {
          setActivity({
            lastLoginAt: payload?.last_login_at ?? '',
            isOnline: Boolean(payload?.is_online),
          })
        }
      } catch {
        // best-effort : l'état initial reste affiché.
      }
    }

    void refresh()
    const timer = window.setInterval(refresh, POLL_MS)
    return () => {
      cancelled = true
      window.clearInterval(timer)
    }
  }, [userId])

  const status = useMemo(
    () => activityStatus(activity.lastLoginAt, activity.isOnline, locale),
    [activity.isOnline, activity.lastLoginAt, locale],
  )

  if (!status.visible) return null

  return (
    <span
      className={cn(
        'inline-flex items-center gap-1.5 text-muted-foreground',
        className,
      )}
    >
      {status.online && (
        <span className="h-2 w-2 rounded-full bg-emerald-500" aria-hidden />
      )}
      {status.online
        ? t('activity.online')
        : t('activity.last_seen', { time: status.relative })}
    </span>
  )
}

function activityStatus(
  lastLoginAt: string,
  isOnline: boolean,
  locale: string,
): {
  visible: boolean
  online: boolean
  relative: string
} {
  if (isOnline) return { visible: true, online: true, relative: '' }
  if (!lastLoginAt) return { visible: false, online: false, relative: '' }
  const date = new Date(lastLoginAt)
  const time = date.getTime()
  if (Number.isNaN(time)) return { visible: false, online: false, relative: '' }
  const diff = Date.now() - time
  return {
    visible: true,
    online: false,
    relative: formatRelativeTime(diff, locale),
  }
}

function formatRelativeTime(diffMs: number, locale: string): string {
  const intl = locale === 'en' ? 'en-US' : 'fr-FR'
  const rtf = new Intl.RelativeTimeFormat(intl, { numeric: 'auto' })
  const abs = Math.max(0, diffMs)
  const minute = 60 * 1000
  const hour = 60 * minute
  const day = 24 * hour
  const month = 30 * day
  const year = 365 * day

  if (abs < hour)
    return rtf.format(-Math.max(1, Math.round(abs / minute)), 'minute')
  if (abs < day) return rtf.format(-Math.round(abs / hour), 'hour')
  if (abs < month) return rtf.format(-Math.round(abs / day), 'day')
  if (abs < year) return rtf.format(-Math.round(abs / month), 'month')
  return rtf.format(-Math.round(abs / year), 'year')
}
