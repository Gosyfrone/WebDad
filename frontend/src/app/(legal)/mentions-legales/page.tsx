import type { Metadata } from 'next'

import { ROUTES } from '@/lib/routes'
import { LegalSection, LegalShell } from '@/components/legal/legal-shell'

export const metadata: Metadata = {
  title: 'Mentions légales — Breezy',
  description: 'Mentions légales du réseau social Breezy.',
}

export default function MentionsLegalesPage() {
  return (
    <LegalShell
      title="Mentions légales"
      updatedAt="5 juin 2026"
      current={ROUTES.mentionsLegales}
    >
      <p className="text-sm leading-relaxed text-muted-foreground">
        Breezy est un réseau social développé dans un cadre pédagogique (projet
        FISA INFO A3). Les informations ci-dessous sont fournies au titre de la
        transparence ; certaines mentions sont à compléter par l’éditeur.
      </p>

      <LegalSection title="Éditeur du site">
        <p>
          Le site et l’application Breezy (ci-après « le Service ») sont édités
          par&nbsp;: <strong>[Raison sociale / équipe — à compléter]</strong>.
        </p>
        <p>
          Adresse&nbsp;: <strong>[Adresse postale — à compléter]</strong>
          <br />
          Adresse e-mail&nbsp;:{' '}
          <strong>[contact@breezy.example — à compléter]</strong>
        </p>
      </LegalSection>

      <LegalSection title="Directeur de la publication">
        <p>
          Le directeur de la publication est&nbsp;:{' '}
          <strong>[Nom du responsable — à compléter]</strong>.
        </p>
      </LegalSection>

      <LegalSection title="Hébergement">
        <p>
          Le Service est hébergé par&nbsp;:{' '}
          <strong>[Nom de l’hébergeur — à compléter]</strong>,
          <br />
          <strong>[Adresse de l’hébergeur — à compléter]</strong>.
        </p>
      </LegalSection>

      <LegalSection title="Propriété intellectuelle">
        <p>
          La marque « Breezy », le logo, la charte graphique ainsi que la
          structure et le contenu éditorial du Service sont protégés par le droit
          de la propriété intellectuelle. Toute reproduction ou représentation,
          totale ou partielle, sans autorisation préalable est interdite.
        </p>
        <p>
          Les contenus publiés par les utilisateurs restent la propriété de leurs
          auteurs, dans les conditions prévues par les{' '}
          <strong>Conditions Générales d’Utilisation</strong>.
        </p>
      </LegalSection>

      <LegalSection title="Responsabilité">
        <p>
          L’éditeur s’efforce d’assurer l’exactitude des informations diffusées
          sur le Service mais ne saurait être tenu responsable des erreurs,
          d’une absence de disponibilité ou de la présence de contenus
          publiés par les utilisateurs. Le Service étant un projet pédagogique,
          il est fourni « en l’état », sans garantie de continuité.
        </p>
      </LegalSection>

      <LegalSection title="Contact">
        <p>
          Pour toute question relative aux présentes mentions légales, vous
          pouvez écrire à&nbsp;:{' '}
          <strong>[contact@breezy.example — à compléter]</strong>.
        </p>
      </LegalSection>
    </LegalShell>
  )
}
