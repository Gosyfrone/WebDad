import type { Metadata } from 'next'

import { LegalDocView } from '@/components/legal/legal-doc-view'

export const metadata: Metadata = {
  title: 'Mentions légales — Breezy',
  description: 'Mentions légales du réseau social Breezy.',
}

export default function MentionsLegalesPage() {
  return <LegalDocView slug="mentions-legales" />
}
