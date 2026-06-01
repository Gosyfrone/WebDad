import Image from 'next/image'
import Link from 'next/link'

import { ROUTES } from '@/lib/routes'

/**
 * Layout des pages publiques d'authentification (login / register).
 * Centre le contenu (formulaire) verticalement et horizontalement.
 */
export default function AuthLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center px-4">
      <Link href={ROUTES.home} className="mb-8">
        <Image
          src="/logo_breezy.png"
          alt="Breezy"
          width={1106}
          height={336}
          className="h-14 w-auto object-contain"
          priority
        />
      </Link>
      <div className="w-full max-w-sm">{children}</div>
    </div>
  )
}
