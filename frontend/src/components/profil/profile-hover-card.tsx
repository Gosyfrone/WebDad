'use client'

import { useEffect, useRef, useState } from 'react'
import Link from 'next/link'
import { Loader2, UserRound } from 'lucide-react'

import { getCommonFollowers } from '@/lib/api'
import { getPublicProfil, subscribeProfilUpdated } from '@/lib/profil-client'
import { cn, initialOf } from '@/lib/utils'
import { useFollow } from '@/lib/use-follow'
import type { ProfilDetails, RelationUser } from '@/types'
import { useLanguage } from '@/components/language-provider'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import { ActivityStatus } from '@/components/profil/activity-status'
import { ActivityPresenceDot } from '@/components/profil/activity-presence-dot'
import { CertificationBadge } from '@/components/profil/certification-badge'

type AnchorProps = Omit<React.AnchorHTMLAttributes<HTMLAnchorElement>, 'href'>

interface ProfileHoverCardProps extends AnchorProps {
  author: { id: string; username: string }
  href: string
  className?: string
  children: React.ReactNode
}

type LoadState =
  | {
      status: 'idle'
      profil: null
      commonFollowers: RelationUser[]
      commonLoading: false
    }
  | {
      status: 'loading'
      profil: null
      commonFollowers: RelationUser[]
      commonLoading: false
    }
  | {
      status: 'ready'
      profil: ProfilDetails
      commonFollowers: RelationUser[]
      commonLoading: boolean
    }
  | {
      status: 'error'
      profil: null
      commonFollowers: RelationUser[]
      commonLoading: false
    }

const profilePreviewCache = new Map<string, ProfilDetails>()
const commonFollowersCache = new Map<string, RelationUser[]>()

export function ProfileHoverCard({
  author,
  href,
  className,
  children,
  ...linkProps
}: ProfileHoverCardProps) {
  const { t } = useLanguage()
  const [open, setOpen] = useState(false)
  const [state, setState] = useState<LoadState>({
    status: 'idle',
    profil: null,
    commonFollowers: [],
    commonLoading: false,
  })
  const closeTimer = useRef<ReturnType<typeof setTimeout> | null>(null)
  const openTimer = useRef<ReturnType<typeof setTimeout> | null>(null)
  const triggerRef = useRef<HTMLAnchorElement | null>(null)
  const triggerHovered = useRef(false)
  const contentHovered = useRef(false)
  const pointerTriggered = useRef(false)
  const loadSeq = useRef(0)

  useEffect(() => {
    if (!open) return
    const cachedProfil = profilePreviewCache.get(author.username)
    const cachedCommonFollowers = commonFollowersCache.get(author.id) ?? []
    if (cachedProfil) {
      setState({
        status: 'ready',
        profil: cachedProfil,
        commonFollowers: cachedCommonFollowers,
        commonLoading: !commonFollowersCache.has(author.id),
      })
      if (!commonFollowersCache.has(author.id)) {
        void loadCommonFollowers(cachedProfil)
      }
      return
    }

    let cancelled = false
    const seq = loadSeq.current + 1
    loadSeq.current = seq
    setState({
      status: 'loading',
      profil: null,
      commonFollowers: [],
      commonLoading: false,
    })
    ;(async () => {
      try {
        const profil = await withTimeout(getPublicProfil(author.username), 6000)
        profilePreviewCache.set(author.username, profil)
        if (cancelled || loadSeq.current !== seq) return
        setState({
          status: 'ready',
          profil,
          commonFollowers: [],
          commonLoading: true,
        })
        void loadCommonFollowers(profil)
      } catch {
        if (!cancelled && loadSeq.current === seq) {
          setState({
            status: 'error',
            profil: null,
            commonFollowers: [],
            commonLoading: false,
          })
        }
      }
    })()
    return () => {
      cancelled = true
    }
    // `open` intentionally starts the load once; state changes must not refire it.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [author.id, author.username, open])

  useEffect(() => {
    return () => {
      clearOpenTimer()
      clearCloseTimer()
    }
  }, [])

  // Resync the preview after a self profile edit: refresh the module cache and,
  // if this card is currently showing the edited user, replace the live data.
  useEffect(
    () =>
      subscribeProfilUpdated((updated) => {
        profilePreviewCache.set(updated.username, updated)
        setState((current) =>
          current.status === 'ready' && current.profil.userId === updated.userId
            ? { ...current, profil: updated }
            : current,
        )
      }),
    [],
  )

  async function loadCommonFollowers(profil: ProfilDetails) {
    const cached = commonFollowersCache.get(profil.userId)
    if (cached) {
      setState((current) =>
        current.status === 'ready' && current.profil.userId === profil.userId
          ? { ...current, commonFollowers: cached, commonLoading: false }
          : current,
      )
      return
    }
    const seq = loadSeq.current
    try {
      const commonFollowers = await withTimeout(
        getCommonFollowers(profil.userId, 3),
        5000,
      )
      commonFollowersCache.set(profil.userId, commonFollowers)
      if (loadSeq.current !== seq) return
      setState((current) =>
        current.status === 'ready' && current.profil.userId === profil.userId
          ? { ...current, commonFollowers, commonLoading: false }
          : current,
      )
    } catch {
      if (loadSeq.current !== seq) return
      setState((current) =>
        current.status === 'ready' && current.profil.userId === profil.userId
          ? { ...current, commonLoading: false }
          : current,
      )
    }
  }

  function scheduleOpen(event: React.MouseEvent<HTMLAnchorElement>) {
    linkProps.onMouseEnter?.(event)
    pointerTriggered.current = true
    beginOpen()
  }

  function scheduleTriggerClose(event: React.MouseEvent<HTMLAnchorElement>) {
    linkProps.onMouseLeave?.(event)
    endTriggerHover()
  }

  function schedulePointerOpen(event: React.PointerEvent<HTMLAnchorElement>) {
    linkProps.onPointerEnter?.(event)
    if (event.pointerType === 'mouse') {
      pointerTriggered.current = true
      beginOpen()
    }
  }

  function schedulePointerClose(event: React.PointerEvent<HTMLAnchorElement>) {
    linkProps.onPointerLeave?.(event)
    if (event.pointerType === 'mouse') endTriggerHover()
  }

  function beginOpen() {
    if (!canHoverPreview()) return
    triggerHovered.current = true
    clearCloseTimer()
    clearOpenTimer()
    openTimer.current = setTimeout(() => {
      if (triggerHovered.current && triggerRef.current?.matches(':hover'))
        setOpen(true)
    }, 260)
  }

  function endTriggerHover() {
    triggerHovered.current = false
    blurTriggerIfPointerDriven(pointerTriggered, triggerRef)
    scheduleClose()
  }

  function scheduleClose() {
    clearOpenTimer()
    clearCloseTimer()
    closeTimer.current = setTimeout(() => {
      if (!triggerHovered.current && !contentHovered.current) setOpen(false)
    }, 120)
  }

  function clearOpenTimer() {
    if (openTimer.current) {
      clearTimeout(openTimer.current)
      openTimer.current = null
    }
  }

  function clearCloseTimer() {
    if (closeTimer.current) {
      clearTimeout(closeTimer.current)
      closeTimer.current = null
    }
  }

  function closeNow() {
    triggerHovered.current = false
    contentHovered.current = false
    clearOpenTimer()
    clearCloseTimer()
    setOpen(false)
    blurTriggerIfPointerDriven(pointerTriggered, triggerRef)
  }

  function handleOpenChange(nextOpen: boolean) {
    if (!nextOpen) closeNow()
  }

  return (
    <Popover open={open} onOpenChange={handleOpenChange}>
      <PopoverTrigger asChild>
        <Link
          {...linkProps}
          ref={triggerRef}
          href={href}
          className={cn(
            'inline-flex rounded-sm focus:outline-none focus-visible:outline-none',
            className,
          )}
          onMouseEnter={scheduleOpen}
          onMouseLeave={scheduleTriggerClose}
          onPointerEnter={schedulePointerOpen}
          onPointerLeave={schedulePointerClose}
          onBlur={closeNow}
        >
          {children}
        </Link>
      </PopoverTrigger>
      <PopoverContent
        align="start"
        side="bottom"
        sideOffset={10}
        onOpenAutoFocus={(event) => {
          event.preventDefault()
        }}
        onCloseAutoFocus={(event) => {
          event.preventDefault()
          requestAnimationFrame(() => {
            blurActiveElement(triggerRef.current)
            pointerTriggered.current = false
          })
        }}
        onMouseEnter={() => {
          contentHovered.current = true
          clearCloseTimer()
        }}
        onMouseLeave={() => {
          contentHovered.current = false
          scheduleClose()
        }}
        onEscapeKeyDown={closeNow}
        className="panel w-[min(22rem,calc(100vw-2rem))] overflow-hidden rounded-[22px] border p-0 shadow-[0_28px_80px_rgba(91,108,255,0.24)]"
      >
        {state.status === 'loading' || state.status === 'idle' ? (
          <div className="flex h-36 items-center justify-center">
            <Loader2
              className="h-5 w-5 animate-spin text-[#5B6CFF]"
              aria-hidden
            />
          </div>
        ) : state.status === 'error' ? (
          <div className="flex items-center gap-3 px-4 py-5 text-sm text-muted-foreground">
            <UserRound className="h-5 w-5 text-[#5B6CFF]" aria-hidden />
            {t('profile_hover.load_error')}
          </div>
        ) : (
          <ProfilePreview
            profil={state.profil}
            commonFollowers={state.commonFollowers}
            commonLoading={state.commonLoading}
            href={href}
          />
        )}
      </PopoverContent>
    </Popover>
  )
}

function ProfilePreview({
  profil,
  commonFollowers,
  commonLoading,
  href,
}: {
  profil: ProfilDetails
  commonFollowers: RelationUser[]
  commonLoading: boolean
  href: string
}) {
  const { t } = useLanguage()
  const followState = useFollow(true)
  const target = toRelationUser(profil)
  const isSelf = followState.currentUserId === profil.userId
  const following = followState.isFollowing(profil.userId)
  const requested = followState.isRequested(profil.userId)
  const pending = followState.isPending(profil.userId)

  async function toggleFollow() {
    if (isSelf || pending || requested || !followState.loaded) return
    await followState.toggle(target, !following)
  }

  return (
    <div className="p-4">
      <div className="flex items-start justify-between gap-3">
        <Avatar className="h-16 w-16 border-2 border-white shadow-[0_14px_34px_rgba(91,108,255,0.24)] dark:border-[#140c24]">
          {profil.avatarUrl && (
            <AvatarImage src={profil.avatarUrl} alt={profil.displayName} />
          )}
          <AvatarFallback className="bg-gradient-to-br from-[#8D3DFF] via-[#5B6CFF] to-[#47D9FF] text-xl font-bold text-white">
            {initialOf(profil.displayName, profil.username)}
          </AvatarFallback>
          <ActivityPresenceDot
            userId={profil.userId}
            initialLastLoginAt={profil.lastLoginAt}
            initialIsOnline={profil.isOnline}
            className="h-3.5 w-3.5"
          />
        </Avatar>
        {!isSelf && (
          <button
            type="button"
            onClick={() => void toggleFollow()}
            disabled={pending || requested || !followState.loaded}
            className={cn(
              'min-w-28 rounded-full px-4 py-2 text-sm font-extrabold shadow-sm transition disabled:cursor-not-allowed disabled:opacity-70',
              following
                ? 'border border-white/70 bg-white/80 text-foreground backdrop-blur hover:border-destructive/40 hover:bg-destructive/10 hover:text-destructive dark:border-white/15 dark:bg-white/10 dark:hover:bg-destructive/20'
                : 'bg-gradient-to-r from-[#8D3DFF] via-[#5B6CFF] to-[#47D9FF] text-white hover:brightness-105',
            )}
          >
            {!followState.loaded || pending ? (
              <Loader2 className="mx-auto h-4 w-4 animate-spin" aria-hidden />
            ) : requested ? (
              t('follow.requested')
            ) : following ? (
              t('profile_hover.following')
            ) : (
              t('profile_hover.follow')
            )}
          </button>
        )}
      </div>

      <div className="mt-3 min-w-0">
        <div className="flex min-w-0 items-center gap-1.5">
          <Link
            href={href}
            className="truncate text-lg font-extrabold text-foreground hover:underline"
          >
            {profil.displayName}
          </Link>
          <CertificationBadge certification={profil.certification} role={profil.role} />
        </div>
        <div className="flex min-w-0 items-center gap-2">
          <Link
            href={href}
            className="truncate text-sm text-muted-foreground hover:underline"
          >
            @{profil.username}
          </Link>
          <ActivityStatus
            userId={profil.userId}
            initialLastLoginAt={profil.lastLoginAt}
            initialIsOnline={profil.isOnline}
            className="shrink-0 text-xs"
          />
        </div>
      </div>

      {profil.bio && (
        <p className="mt-3 line-clamp-3 whitespace-pre-wrap text-sm leading-relaxed text-foreground/85">
          {profil.bio}
        </p>
      )}

      <div className="mt-3 flex gap-5 text-sm">
        <Stat value={profil.followingCount} label={t('profil.following')} />
        <Stat value={profil.followersCount} label={t('profil.followers')} />
      </div>

      {(commonFollowers.length > 0 || commonLoading) && (
        <div className="mt-4 flex items-center gap-3">
          <div className="flex min-w-7 -space-x-2">
            {commonFollowers.map((user) => (
              <Avatar
                key={user.id}
                className="h-7 w-7 border-2 border-white shadow-sm dark:border-[#140c24]"
              >
                {user.avatarUrl && (
                  <AvatarImage src={user.avatarUrl} alt={user.displayName} />
                )}
                <AvatarFallback className="bg-gradient-to-br from-[#8D3DFF] via-[#5B6CFF] to-[#47D9FF] text-[11px] font-bold text-white">
                  {initialOf(user.displayName, user.username)}
                </AvatarFallback>
                <ActivityPresenceDot userId={user.id} className="h-2.5 w-2.5" />
              </Avatar>
            ))}
            {commonLoading && commonFollowers.length === 0 && (
              <div className="grid h-7 w-7 place-items-center rounded-full border-2 border-white bg-white/80 shadow-sm backdrop-blur dark:border-[#140c24] dark:bg-white/10">
                <Loader2
                  className="h-3.5 w-3.5 animate-spin text-[#5B6CFF]"
                  aria-hidden
                />
              </div>
            )}
          </div>
          {commonFollowers.length > 0 && (
            <p className="min-w-0 text-sm leading-snug text-muted-foreground">
              {t('profile_hover.followed_by', {
                names: formatNames(commonFollowers),
              })}
            </p>
          )}
        </div>
      )}
    </div>
  )
}

function toRelationUser(profil: ProfilDetails): RelationUser {
  return {
    id: profil.userId,
    username: profil.username,
    displayName: profil.displayName,
    bio: profil.bio,
    avatarUrl: profil.avatarUrl,
    certification: profil.certification,
  }
}

function Stat({ value, label }: { value: number; label: string }) {
  return (
    <div className="flex gap-1">
      <span className="font-extrabold text-foreground">
        {formatCount(value)}
      </span>
      <span className="text-muted-foreground">{label}</span>
    </div>
  )
}

function formatNames(users: RelationUser[]): string {
  return users.map((user) => user.displayName || `@${user.username}`).join(', ')
}

function formatCount(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)} M`
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)} K`
  return String(n)
}

function canHoverPreview(): boolean {
  if (typeof window === 'undefined') return false
  return window.matchMedia('(hover: hover) and (pointer: fine)').matches
}

function blurActiveElement(element: HTMLAnchorElement | null): void {
  if (!element || document.activeElement !== element) return
  element.blur()
}

function blurTriggerIfPointerDriven(
  pointerTriggered: React.MutableRefObject<boolean>,
  triggerRef: React.MutableRefObject<HTMLAnchorElement | null>,
): void {
  if (!pointerTriggered.current) return
  pointerTriggered.current = false
  blurActiveElement(triggerRef.current)
}

async function withTimeout<T>(promise: Promise<T>, ms: number): Promise<T> {
  let timer: ReturnType<typeof setTimeout> | null = null
  try {
    return await Promise.race([
      promise,
      new Promise<T>((_, reject) => {
        timer = setTimeout(() => reject(new Error('timeout')), ms)
      }),
    ])
  } finally {
    if (timer) clearTimeout(timer)
  }
}
