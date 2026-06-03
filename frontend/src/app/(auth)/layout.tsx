import Image from 'next/image'
import Link from 'next/link'

import { ROUTES } from '@/lib/routes'

/**
 * Layout des pages publiques d'authentification (login / register).
 * Centre le contenu (formulaire) verticalement et horizontalement.
 */
export default function AuthLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-dvh flex-col items-stretch overflow-hidden bg-slate-50">
      <div className="h-full w-full">{children}</div>
    </div>
  )
}
