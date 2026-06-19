'use client'

import { useEffect, useState } from 'react'
import { Activity, Lock } from 'lucide-react'

import { useT } from '@/components/language-provider'
import { useToast } from '@/hooks/use-toast'
import {
  getMyProfil,
  saveMyActivityVisibility,
  saveMyLikesVisibility,
  saveMyVisibility,
} from '@/lib/profil-client'
import { cn } from '@/lib/utils'
import type { ProfilDetails } from '@/types'

type Visibility = ProfilDetails['visibility']

export function VisibilitySettings() {
  const t = useT()
  const { toast } = useToast()
  const [visibility, setVisibility] = useState<Visibility>('public')
  const [likesVisibility, setLikesVisibility] = useState<Visibility>('public')
  const [activityVisibility, setActivityVisibility] =
    useState<Visibility>('public')
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState<'account' | 'likes' | 'activity' | null>(
    null,
  )

  useEffect(() => {
    let cancelled = false

    async function loadVisibility() {
      try {
        const profil = await getMyProfil()
        if (!cancelled) {
          setVisibility(profil.visibility)
          setLikesVisibility(profil.likesVisibility)
          setActivityVisibility(profil.activityVisibility)
        }
      } catch {
        if (!cancelled) {
          toast({ title: t('visibility.load_failed'), variant: 'destructive' })
        }
      } finally {
        if (!cancelled) setLoading(false)
      }
    }

    void loadVisibility()
    return () => {
      cancelled = true
    }
  }, [toast, t])

  async function updateVisibility(next: Visibility) {
    if (next === visibility || saving) return
    const previous = visibility
    setVisibility(next)
    setSaving('account')
    try {
      const updated = await saveMyVisibility(next)
      setVisibility(updated.visibility)
      toast({ title: t('visibility.saved') })
    } catch {
      setVisibility(previous)
      toast({ title: t('visibility.save_failed'), variant: 'destructive' })
    } finally {
      setSaving(null)
    }
  }

  async function updateLikesVisibility(next: Visibility) {
    if (next === likesVisibility || saving) return
    const previous = likesVisibility
    setLikesVisibility(next)
    setSaving('likes')
    try {
      const updated = await saveMyLikesVisibility(next)
      setLikesVisibility(updated.likesVisibility)
      toast({ title: t('visibility.likes_saved') })
    } catch {
      setLikesVisibility(previous)
      toast({
        title: t('visibility.likes_save_failed'),
        variant: 'destructive',
      })
    } finally {
      setSaving(null)
    }
  }

  async function updateActivityVisibility(next: Visibility) {
    if (next === activityVisibility || saving) return
    const previous = activityVisibility
    setActivityVisibility(next)
    setSaving('activity')
    try {
      const updated = await saveMyActivityVisibility(next)
      setActivityVisibility(updated.activityVisibility)
      toast({ title: t('visibility.activity_saved') })
    } catch {
      setActivityVisibility(previous)
      toast({
        title: t('visibility.activity_save_failed'),
        variant: 'destructive',
      })
    } finally {
      setSaving(null)
    }
  }

  const isPrivate = visibility === 'private'
  const isLikesPrivate = likesVisibility === 'private'
  const isActivityPrivate = activityVisibility === 'private'
  const disabled = loading || saving !== null

  return (
    <div className="panel rounded-xl border px-3 py-3 shadow-sm">
      <div className="flex items-center justify-between gap-3 rounded-lg px-1 py-1">
        <div className="flex min-w-0 items-center gap-3">
          <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-primary/10 text-primary">
            <Lock className="h-4 w-4" aria-hidden />
          </span>
          <div className="min-w-0">
            <span className="block text-sm font-medium text-foreground">
              {t('visibility.private_account')}
            </span>
            <span className="mt-0.5 block text-xs leading-4 text-muted-foreground">
              {t('visibility.private_toggle_desc')}
            </span>
          </div>
        </div>

        <button
          type="button"
          role="switch"
          aria-checked={isPrivate}
          aria-label={t('visibility.toggle_aria')}
          disabled={disabled}
          onClick={() => updateVisibility(isPrivate ? 'public' : 'private')}
          className={cn(
            'relative inline-flex h-8 w-[3.25rem] shrink-0 items-center rounded-full border transition-colors',
            isPrivate
              ? 'border-primary bg-primary'
              : 'border-slate-400 bg-slate-300 dark:border-slate-500 dark:bg-slate-700',
            disabled && 'cursor-not-allowed opacity-60',
          )}
        >
          <span
            className={cn(
              'absolute left-1 z-0 flex h-6 w-6 items-center justify-center rounded-full bg-white shadow-sm ring-1 ring-black/10 transition-transform dark:bg-slate-100',
              isPrivate ? 'translate-x-5' : 'translate-x-0',
            )}
          >
            {isPrivate ? (
              <Lock className="h-3.5 w-3.5 text-primary" aria-hidden />
            ) : null}
          </span>
        </button>
      </div>

      <div className="mt-1 flex items-center justify-between gap-3 rounded-lg border-t px-1 pt-3">
        <div className="flex min-w-0 items-center gap-3">
          <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-primary/10 text-primary">
            <Lock className="h-4 w-4" aria-hidden />
          </span>
          <div className="min-w-0">
            <span className="block text-sm font-medium text-foreground">
              {t('visibility.likes_private_account')}
            </span>
            <span className="mt-0.5 block text-xs leading-4 text-muted-foreground">
              {t('visibility.likes_private_toggle_desc')}
            </span>
          </div>
        </div>

        <button
          type="button"
          role="switch"
          aria-checked={isLikesPrivate}
          aria-label={t('visibility.likes_toggle_aria')}
          disabled={disabled}
          onClick={() =>
            updateLikesVisibility(isLikesPrivate ? 'public' : 'private')
          }
          className={cn(
            'relative inline-flex h-8 w-[3.25rem] shrink-0 items-center rounded-full border transition-colors',
            isLikesPrivate
              ? 'border-primary bg-primary'
              : 'border-slate-400 bg-slate-300 dark:border-slate-500 dark:bg-slate-700',
            disabled && 'cursor-not-allowed opacity-60',
          )}
        >
          <span
            className={cn(
              'absolute left-1 z-0 flex h-6 w-6 items-center justify-center rounded-full bg-white shadow-sm ring-1 ring-black/10 transition-transform dark:bg-slate-100',
              isLikesPrivate ? 'translate-x-5' : 'translate-x-0',
            )}
          >
            {isLikesPrivate ? (
              <Lock className="h-3.5 w-3.5 text-primary" aria-hidden />
            ) : null}
          </span>
        </button>
      </div>

      <div className="mt-1 flex items-center justify-between gap-3 rounded-lg border-t px-1 pt-3">
        <div className="flex min-w-0 items-center gap-3">
          <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-primary/10 text-primary">
            <Activity className="h-4 w-4" aria-hidden />
          </span>
          <div className="min-w-0">
            <span className="block text-sm font-medium text-foreground">
              {t('visibility.activity_private_account')}
            </span>
            <span className="mt-0.5 block text-xs leading-4 text-muted-foreground">
              {isPrivate
                ? t('visibility.activity_private_followers_desc')
                : t('visibility.activity_private_toggle_desc')}
            </span>
          </div>
        </div>

        <button
          type="button"
          role="switch"
          aria-checked={isActivityPrivate}
          aria-label={t('visibility.activity_toggle_aria')}
          disabled={disabled}
          onClick={() =>
            updateActivityVisibility(isActivityPrivate ? 'public' : 'private')
          }
          className={cn(
            'relative inline-flex h-8 w-[3.25rem] shrink-0 items-center rounded-full border transition-colors',
            isActivityPrivate
              ? 'border-primary bg-primary'
              : 'border-slate-400 bg-slate-300 dark:border-slate-500 dark:bg-slate-700',
            disabled && 'cursor-not-allowed opacity-60',
          )}
        >
          <span
            className={cn(
              'absolute left-1 z-0 flex h-6 w-6 items-center justify-center rounded-full bg-white shadow-sm ring-1 ring-black/10 transition-transform dark:bg-slate-100',
              isActivityPrivate ? 'translate-x-5' : 'translate-x-0',
            )}
          >
            {isActivityPrivate ? (
              <Activity className="h-3.5 w-3.5 text-primary" aria-hidden />
            ) : null}
          </span>
        </button>
      </div>
    </div>
  )
}
