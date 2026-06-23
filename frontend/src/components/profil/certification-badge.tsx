'use client'

import Image from 'next/image'
import { Check } from 'lucide-react'

import { useLanguage } from '@/components/language-provider'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { cn } from '@/lib/utils'
import type { UserCertification, UserRole } from '@/types'

interface CertificationBadgeProps {
  certification?: UserCertification
  role?: UserRole
  className?: string
}

export function CertificationBadge({
  certification = 'none',
  role = 'user',
  className,
}: CertificationBadgeProps) {
  const { t } = useLanguage()
  const kind = badgeKind(certification, role)
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
            <span
              className={cn(
                'inline-flex h-5 w-5 items-center justify-center rounded-full text-white shadow-sm',
                kind === 'political' ? 'bg-amber-400' : 'bg-[#1D9BF0]',
              )}
              aria-hidden
            >
              <Check className="h-3.5 w-3.5 stroke-[3]" />
            </span>
          )}
        </button>
      </PopoverTrigger>
      <PopoverContent align="center" className="w-auto px-2 py-1 text-xs font-semibold">
        {label}
      </PopoverContent>
    </Popover>
  )
}

function badgeKind(certification: UserCertification, role: UserRole): UserCertification | 'staff' {
  if (role === 'administrator' || role === 'moderator') return 'staff'
  return certification
}
