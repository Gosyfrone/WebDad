import type { Metadata } from 'next'
import Link from 'next/link'

import { ROUTES } from '@/lib/routes'
import { LegalList, LegalSection, LegalShell } from '@/components/legal/legal-shell'

export const metadata: Metadata = {
  title: 'Conditions Générales d’Utilisation — Breezy',
  description: 'Conditions Générales d’Utilisation du réseau social Breezy.',
}

export default function CguPage() {
  return (
    <LegalShell
      title="Conditions Générales d’Utilisation"
      updatedAt="5 juin 2026"
      current={ROUTES.cgu}
    >
      <p className="text-sm leading-relaxed text-muted-foreground">
        Les présentes Conditions Générales d’Utilisation (« CGU ») régissent
        l’accès et l’utilisation du réseau social Breezy (« le Service »). En
        créant un compte ou en utilisant le Service, vous acceptez les présentes
        CGU sans réserve.
      </p>

      <LegalSection title="Article 1 — Objet">
        <p>
          Breezy est un réseau social permettant de publier de courts messages,
          de suivre d’autres utilisateurs et d’interagir avec leurs publications.
          Les présentes CGU définissent les conditions d’utilisation du Service.
        </p>
      </LegalSection>

      <LegalSection title="Article 2 — Inscription et compte">
        <LegalList
          items={[
            'L’inscription nécessite une adresse e-mail valide et le choix d’un nom d’utilisateur (identifiant unique).',
            'Le nom d’utilisateur constitue l’identité du compte ; sa modification peut être limitée dans le temps.',
            'Vous êtes responsable de la confidentialité de vos identifiants et de toute activité réalisée depuis votre compte.',
            'Un compte est strictement personnel. Vous vous engagez à fournir des informations exactes.',
          ]}
        />
      </LegalSection>

      <LegalSection title="Article 3 — Accès au Service">
        <p>
          Le Service est accessible gratuitement. L’éditeur peut faire évoluer,
          suspendre ou interrompre tout ou partie du Service, notamment pour des
          raisons de maintenance, sans que sa responsabilité puisse être engagée.
        </p>
      </LegalSection>

      <LegalSection title="Article 4 — Règles de conduite">
        <p>En utilisant le Service, vous vous engagez à ne pas publier de contenu&nbsp;:</p>
        <LegalList
          items={[
            'illicite, diffamatoire, injurieux, haineux ou discriminatoire ;',
            'portant atteinte à la vie privée ou aux droits de tiers ;',
            'à caractère violent, pornographique ou incitant à la haine ;',
            'constituant du harcèlement, du spam ou de la fraude.',
          ]}
        />
      </LegalSection>

      <LegalSection title="Article 5 — Contenus publiés par les utilisateurs">
        <p>
          Vous conservez la propriété des contenus que vous publiez. Vous
          accordez au Service une licence non exclusive et gratuite permettant
          d’héberger et d’afficher ces contenus dans le cadre du fonctionnement
          du Service. Vous êtes seul responsable des contenus que vous publiez.
        </p>
      </LegalSection>

      <LegalSection title="Article 6 — Modération et rôles">
        <p>
          Le Service distingue trois rôles&nbsp;: <strong>Utilisateur</strong>,{' '}
          <strong>Modérateur</strong> et <strong>Administrateur</strong>. Les
          modérateurs et administrateurs peuvent masquer ou supprimer des
          contenus contraires aux présentes CGU et, le cas échéant, suspendre les
          comptes concernés.
        </p>
      </LegalSection>

      <LegalSection title="Article 7 — Responsabilité et disponibilité">
        <p>
          Le Service est fourni « en l’état », dans le cadre d’un projet
          pédagogique. L’éditeur ne garantit pas l’absence d’interruption ou
          d’erreur et ne saurait être tenu responsable des dommages résultant de
          l’utilisation ou de l’indisponibilité du Service.
        </p>
      </LegalSection>

      <LegalSection title="Article 8 — Suspension et résiliation">
        <p>
          Vous pouvez supprimer votre compte à tout moment. L’éditeur peut
          suspendre ou résilier un compte en cas de manquement aux présentes CGU,
          notamment aux règles de conduite (Article 4).
        </p>
      </LegalSection>

      <LegalSection title="Article 9 — Données personnelles">
        <p>
          Le traitement de vos données personnelles est décrit dans notre{' '}
          <Link
            href={ROUTES.confidentialite}
            className="font-semibold text-[#5B6CFF] underline-offset-4 transition-colors hover:text-[#8D3DFF] hover:underline dark:text-[#9DA8FF]"
          >
            Politique de confidentialité
          </Link>
          .
        </p>
      </LegalSection>

      <LegalSection title="Article 10 — Modification des CGU">
        <p>
          L’éditeur peut modifier les présentes CGU à tout moment. La version
          applicable est celle en vigueur lors de votre utilisation du Service.
        </p>
      </LegalSection>

      <LegalSection title="Article 11 — Droit applicable">
        <p>
          Les présentes CGU sont soumises au droit français. En cas de litige, et
          à défaut de résolution amiable, les tribunaux compétents seront ceux du
          ressort du siège de l’éditeur.
        </p>
      </LegalSection>
    </LegalShell>
  )
}
