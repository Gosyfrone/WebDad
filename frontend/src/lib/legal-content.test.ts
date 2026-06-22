import { describe, expect, it } from 'vitest'

import { legalContent } from '@/lib/legal-content'
import type { LegalSlug } from '@/lib/legal-content'
import { LOCALES } from '@/lib/i18n'

const SLUGS: LegalSlug[] = ['mentions-legales', 'cgu', 'confidentialite']
const ALL_LOCALES = LOCALES.map((l) => l.id)

describe('legalContent', () => {
  it('expose les 12 locales', () => {
    for (const locale of ALL_LOCALES) {
      expect(legalContent[locale], `locale ${locale} manquante`).toBeDefined()
    }
  })

  for (const locale of ALL_LOCALES) {
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

  // Parité structurelle : chaque locale doit refléter la structure du FR
  // (même nombre de sections, et même type/longueur de blocs par section),
  // pour garantir qu'aucune section/liste n'a été oubliée à la traduction.
  describe('parité structurelle avec le FR', () => {
    for (const slug of SLUGS) {
      const reference = legalContent.fr[slug]
      for (const locale of ALL_LOCALES) {
        if (locale === 'fr') continue
        describe(`${locale} / ${slug}`, () => {
          const doc = legalContent[locale][slug]

          it('a le même nombre de sections que le FR', () => {
            expect(doc.sections.length).toBe(reference.sections.length)
          })

          it('a la même structure de blocs par section', () => {
            reference.sections.forEach((refSection, i) => {
              const section = doc.sections[i]
              expect(section.blocks.length).toBe(refSection.blocks.length)
              refSection.blocks.forEach((refBlock, j) => {
                const block = section.blocks[j]
                if ('list' in refBlock) {
                  expect('list' in block, `bloc ${j} doit être une liste`).toBe(true)
                  if ('list' in block) {
                    expect(block.list.length).toBe(refBlock.list.length)
                  }
                } else {
                  expect('p' in block, `bloc ${j} doit être un paragraphe`).toBe(true)
                }
              })
            })
          })
        })
      }
    }
  })
})
