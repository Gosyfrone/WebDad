'use client'

import Link from 'next/link'
import { Scale } from 'lucide-react'

import { ROUTES } from '@/lib/routes'
import { cn } from '@/lib/utils'
import { useT } from '@/components/language-provider'
import { LegalBackButton } from '@/components/legal/legal-back-button'

/**
 * Coquille commune aux pages légales (mentions légales, CGU, confidentialité).
 *
 * Reprend la direction artistique de la refonte : fond de page tokenisé
 * (`bg-page` + voiles `bg-page-glow-*`), carte « verre » (`glass-strong`), titre
 * en dégradé de marque (`brand-text`). Surfaces/tokens sémantiques → clair ET
 * sombre automatiques (l'interrupteur clair/sombre est monté par le layout
 * `(legal)` via `FloatingThemeToggle`).
 *
 * Client Component : les libellés statiques (badge, date, liens croisés) sont
 * localisés via `useT()` (FR/EN) ; le titre et la date sont passés déjà localisés
 * par l'appelant (cf. `legal-content.ts`).
 */

const LEGAL_LINKS = [
  { href: ROUTES.mentionsLegales, labelKey: 'legal.mentions' },
  { href: ROUTES.cgu, labelKey: 'legal.cgu' },
  { href: ROUTES.confidentialite, labelKey: 'legal.confidentialite' },
] as const

interface LegalShellProps {
  title: string
  /** Date « Dernière mise à jour » (texte libre, déjà localisé). */
  updatedAt: string
  /** Route de la page courante : son lien croisé est mis en évidence. */
  current: string
  children: React.ReactNode
}

export function LegalShell({ title, updatedAt, current, children }: LegalShellProps) {
  const t = useT()

  return (
    <main className="bg-page relative flex min-h-dvh w-full justify-center overflow-hidden px-4 py-10 sm:px-6">
      <div className="bg-page-glow-1 pointer-events-none absolute inset-0" />
      <div className="bg-page-glow-2 pointer-events-none absolute inset-0" />

      <section className="relative w-full max-w-3xl">
        <LegalBackButton />

        <article className="glass-strong relative mt-4 overflow-hidden rounded-[30px] border shadow-[0_30px_90px_rgba(0,0,0,0.18)] backdrop-blur-2xl">
          {/* Liseré dégradé de marque en tête de carte */}
          <div className="pointer-events-none absolute inset-x-0 top-0 h-1 bg-gradient-to-r from-[#8D3DFF] via-[#5B6CFF] to-[#47D9FF]" />

          <div className="px-6 py-9 sm:px-10 sm:py-11">
            <span className="inline-flex w-fit items-center gap-2 rounded-full border border-white/70 bg-white/80 px-3 py-1 text-xs font-semibold text-[#5B6CFF] shadow-sm dark:border-white/10 dark:bg-white/5 dark:text-[#9DA8FF]">
              <Scale className="h-3.5 w-3.5" aria-hidden />
              {t('legal.badge')}
            </span>

            <h1 className="brand-text mt-4 text-3xl font-bold leading-tight sm:text-4xl">
              {title}
            </h1>
            <p className="mt-2 text-xs text-muted-foreground">
              {t('legal.updated', { date: updatedAt })}
            </p>

            <div className="mt-8 space-y-8">{children}</div>
          </div>
        </article>

        {/* Liens croisés entre les pages légales */}
        <nav className="mt-6 flex flex-wrap items-center justify-center gap-x-2 gap-y-1 text-xs text-muted-foreground">
          {LEGAL_LINKS.map((link, index) => (
            <span key={link.href} className="flex items-center gap-2">
              <Link
                href={link.href}
                aria-current={link.href === current ? 'page' : undefined}
                className={cn(
                  'underline-offset-4 transition-colors hover:text-foreground hover:underline',
                  link.href === current && 'font-semibold text-foreground',
                )}
              >
                {t(link.labelKey)}
              </Link>
              {index < LEGAL_LINKS.length - 1 ? (
                <span aria-hidden className="text-muted-foreground/40">
                  ·
                </span>
              ) : null}
            </span>
          ))}
        </nav>
      </section>
    </main>
  )
}

/** Section de contenu : titre + corps de texte (tokens sémantiques). */
export function LegalSection({
  title,
  children,
}: {
  title: string
  children: React.ReactNode
}) {
  return (
    <section className="space-y-3">
      <h2 className="text-lg font-bold text-foreground sm:text-xl">{title}</h2>
      <div className="space-y-3 text-sm leading-relaxed text-muted-foreground">
        {children}
      </div>
    </section>
  )
}

/** Liste à puces stylée pour les pages légales. */
export function LegalList({ items }: { items: React.ReactNode[] }) {
  return (
    <ul className="list-disc space-y-1.5 pl-5">
      {items.map((item, index) => (
        <li key={index}>{item}</li>
      ))}
    </ul>
  )
}
