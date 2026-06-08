import type { Metadata } from 'next'

import { LegalDocView } from '@/components/legal/legal-doc-view'

export const metadata: Metadata = {
  title: 'Conditions Générales d’Utilisation — Breezy',
  description: 'Conditions Générales d’Utilisation du réseau social Breezy.',
}

export default function CguPage() {
  return <LegalDocView slug="cgu" />
}
