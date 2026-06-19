'use client'

import * as React from 'react'

import { ROUTES } from '@/lib/routes'
import { legalContent, type LegalSlug } from '@/lib/legal-content'
import { useLanguage } from '@/components/language-provider'
import { LegalList, LegalSection, LegalShell } from '@/components/legal/legal-shell'
import { markTermsRead } from '@/lib/terms-consent'

const ROUTE_BY_SLUG: Record<LegalSlug, string> = {
  'mentions-legales': ROUTES.mentionsLegales,
  cgu: ROUTES.cgu,
  confidentialite: ROUTES.confidentialite,
}

/**
 * Rendu d'une page légale dans la langue courante (`useLanguage`) à partir du
 * registre `legal-content.ts` (FR/EN). Repli sur le FR si la locale manque.
 *
 * Le contenu est « plat » (paragraphes + listes) → on le projette sur
 * `LegalShell`/`LegalSection`/`LegalList`. La page (`page.tsx`) reste un Server
 * Component (pour les métadonnées) qui ne fait que monter ce composant client.
 */
export function LegalDocView({ slug }: { slug: LegalSlug }) {
  const { locale } = useLanguage()
  const doc = legalContent[locale]?.[slug] ?? legalContent.en[slug]

  React.useEffect(() => {
    if (slug !== 'cgu') return

    let done = false
    const markWhenRead = () => {
      if (done) return
      const root = document.documentElement
      const remaining = root.scrollHeight - window.scrollY - window.innerHeight
      if (remaining <= 24) {
        done = true
        markTermsRead()
      }
    }

    markWhenRead()
    window.addEventListener('scroll', markWhenRead, { passive: true })
    window.addEventListener('resize', markWhenRead)
    return () => {
      window.removeEventListener('scroll', markWhenRead)
      window.removeEventListener('resize', markWhenRead)
    }
  }, [slug])

  return (
    <LegalShell title={doc.title} updatedAt={doc.updatedAt} current={ROUTE_BY_SLUG[slug]}>
      <p className="text-sm leading-relaxed text-muted-foreground">{doc.intro}</p>

      {doc.sections.map((section, i) => (
        <LegalSection key={i} title={section.title}>
          {section.blocks.map((block, j) =>
            'list' in block ? (
              <LegalList key={j} items={block.list} />
            ) : (
              <p key={j}>{block.p}</p>
            ),
          )}
        </LegalSection>
      ))}
    </LegalShell>
  )
}
