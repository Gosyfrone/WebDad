import { Fragment } from 'react'
import Link from 'next/link'

import { ROUTES } from '@/lib/routes'
import { cn } from '@/lib/utils'

/**
 * Liens vers les pages légales (mentions légales, CGU, confidentialité) + ligne
 * de copyright. Présentational pur (pas de hook) → utilisable depuis un Server
 * Component (sidebar droite, page Paramètres) comme depuis un Client Component
 * (login / register). Tout en tokens sémantiques (`text-muted-foreground`,
 * `hover:text-foreground`) → lisible en mode clair ET sombre sans variante.
 */

const LEGAL_LINKS = [
  { href: ROUTES.mentionsLegales, label: 'Mentions légales' },
  { href: ROUTES.cgu, label: 'CGU' },
  { href: ROUTES.confidentialite, label: 'Confidentialité' },
] as const

export function LegalLinks({ className }: { className?: string }) {
  return (
    <div className={cn('flex flex-col gap-1.5 text-xs text-muted-foreground', className)}>
      <nav className="flex flex-wrap items-center gap-x-2 gap-y-1">
        {LEGAL_LINKS.map((link, index) => (
          <Fragment key={link.href}>
            <Link
              href={link.href}
              className="underline-offset-4 transition-colors hover:text-foreground hover:underline"
            >
              {link.label}
            </Link>
            {index < LEGAL_LINKS.length - 1 ? (
              <span aria-hidden className="text-muted-foreground/40">
                ·
              </span>
            ) : null}
          </Fragment>
        ))}
      </nav>
      <p>© {new Date().getFullYear()} Breezy</p>
    </div>
  )
}
