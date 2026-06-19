import { describe, expect, it } from 'vitest'

import { legalContent } from '@/lib/legal-content'
import type { LegalSlug } from '@/lib/legal-content'

const SLUGS: LegalSlug[] = ['mentions-legales', 'cgu', 'confidentialite']
const LOCALES = ['fr', 'en'] as const

describe('legalContent', () => {
  it('expose les locales fr et en', () => {
    expect(legalContent.fr).toBeDefined()
    expect(legalContent.en).toBeDefined()
  })

  for (const locale of LOCALES) {
    describe(`locale ${locale}`, () => {
      for (const slug of SLUGS) {
        describe(slug, () => {
          it('a un titre non vide', () => {
            expect(legalContent[locale][slug].title.length).toBeGreaterThan(0)
          })

          it('a une date de mise à jour', () => {
            expect(legalContent[locale][slug].updatedAt.length).toBeGreaterThan(0)
          })

          it('a un intro non vide', () => {
            expect(legalContent[locale][slug].intro.length).toBeGreaterThan(0)
          })

          it('a au moins une section', () => {
            expect(legalContent[locale][slug].sections.length).toBeGreaterThan(0)
          })

          it('chaque section a un titre et au moins un bloc', () => {
            for (const section of legalContent[locale][slug].sections) {
              expect(section.title.length).toBeGreaterThan(0)
              expect(section.blocks.length).toBeGreaterThan(0)
            }
          })
        })
      }
    })
  }
})
