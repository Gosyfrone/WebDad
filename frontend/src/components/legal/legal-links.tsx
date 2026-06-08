'use client'

import { Fragment } from 'react'
import Link from 'next/link'

import { ROUTES } from '@/lib/routes'
import { cn } from '@/lib/utils'
import { useT } from '@/components/language-provider'

/**
 * Liens vers les pages légales (mentions légales, CGU, confidentialité) + ligne
 * de copyright. Libellés localisés (`useT`, FR/EN). Tous les consommateurs
 * (sidebar droite, Paramètres, login/register) sont des Client Components.
 * Tokens sémantiques (`text-muted-foreground`/`hover:text-foreground`) →
 * lisible en mode clair ET sombre.
 */

const LEGAL_LINKS = [
  { href: ROUTES.mentionsLegales, labelKey: 'legal.mentions' },
  { href: ROUTES.cgu, labelKey: 'legal.cgu' },
  { href: ROUTES.confidentialite, labelKey: 'legal.confidentialite' },
] as const

export function LegalLinks({ className }: { className?: string }) {
  const t = useT()

  return (
    <div className={cn('flex flex-col gap-1.5 text-xs text-muted-foreground', className)}>
      <nav className="flex flex-wrap items-center gap-x-2 gap-y-1">
        {LEGAL_LINKS.map((link, index) => (
          <Fragment key={link.href}>
            <Link
              href={link.href}
              className="underline-offset-4 transition-colors hover:text-foreground hover:underline"
            >
              {t(link.labelKey)}
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
