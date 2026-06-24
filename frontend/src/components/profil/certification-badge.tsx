'use client'

import Image from 'next/image'
import { useEffect, useState } from 'react'

import { useLanguage } from '@/components/language-provider'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { getUserRole, subscribeIdentityUpdate } from '@/lib/user-roles'
import { cn } from '@/lib/utils'
import type { UserCertification, UserRole } from '@/types'

interface CertificationBadgeProps {
  userId?: string
  certification?: UserCertification
  role?: UserRole
  className?: string
}

export function CertificationBadge({
  userId = '',
  certification = 'none',
  role = 'user',
  className,
}: CertificationBadgeProps) {
  const { t } = useLanguage()
  const [currentCertification, setCurrentCertification] = useState<UserCertification>(certification)
  const [currentRole, setCurrentRole] = useState<UserRole>(role)

  useEffect(() => {
    setCurrentCertification(certification)
  }, [certification])

  useEffect(() => {
    setCurrentRole(role)
  }, [role])

  useEffect(() => {
    if (!userId) return
    let cancelled = false
    void getUserRole(userId)
      .then((resolvedRole) => {
        if (!cancelled) setCurrentRole(resolvedRole)
      })
      .catch(() => {})

    const unsubscribe = subscribeIdentityUpdate((update) => {
      if (update.userId !== userId) return
      if (update.certification) setCurrentCertification(update.certification)
      if (update.role) setCurrentRole(update.role)
    })
    return () => {
      cancelled = true
      unsubscribe()
    }
  }, [userId])

  const kind = badgeKind(currentCertification, currentRole)
  if (kind === 'none') return null

  const label = t(`certification.${kind}`)
  return (
    <Popover>
      <PopoverTrigger asChild>
        <button
          type="button"
          aria-label={label}
          title={label}
          className={cn(
            'inline-flex h-5 w-5 shrink-0 items-center justify-center rounded-full outline-none transition-transform hover:scale-105 focus-visible:ring-2 focus-visible:ring-[#5B6CFF]',
            className,
          )}
        >
          {kind === 'staff' ? (
            <Image
              src="/logo_breezy.png"
              alt=""
              width={18}
              height={18}
              className="h-5 w-5 rounded-full object-contain"
            />
          ) : (
            <VerifiedSeal kind={kind} />
          )}
        </button>
      </PopoverTrigger>
      <PopoverContent align="center" className="w-auto px-2 py-1 text-xs font-semibold">
        {label}
      </PopoverContent>
    </Popover>
  )
}

function VerifiedSeal({ kind }: { kind: 'political' | 'public_figure' }) {
  const fill = kind === 'political' ? '#F6B51E' : '#1D9BF0'

  return (
    <svg
      aria-hidden
      viewBox="0 0 24 24"
      className="h-5 w-5 drop-shadow-[0_1px_1px_rgba(15,23,42,0.18)]"
    >
      <path
        fill={fill}
        d="M12 1.25 14.35 3.1l2.98-.24 1.02 2.81 2.56 1.54-.82 2.88 1.1 2.78-2.4 1.79-.74 2.9-2.99.06L12 22.75l-3.06-2.13-2.99-.06-.74-2.9-2.4-1.79 1.1-2.78-.82-2.88 2.56-1.54 1.02-2.81 2.98.24L12 1.25Z"
      />
      <path
        fill="none"
        stroke="white"
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth="2.4"
        d="m7.75 12.25 2.65 2.65 5.95-6.05"
      />
    </svg>
  )
}

function badgeKind(certification: UserCertification, role: UserRole): UserCertification | 'staff' {
  if (role === 'administrator' || role === 'moderator') return 'staff'
  return certification
}
