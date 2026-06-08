import type { Metadata } from 'next'

import { LegalDocView } from '@/components/legal/legal-doc-view'

export const metadata: Metadata = {
  title: 'Politique de confidentialité — Breezy',
  description: 'Politique de confidentialité et traitement des données du réseau social Breezy.',
}

// Server Component (métadonnées) → contenu bilingue rendu côté client selon la
// langue choisie (cf. legal-doc-view + legal-content).
export default function ConfidentialitePage() {
  return <LegalDocView slug="confidentialite" />
}
