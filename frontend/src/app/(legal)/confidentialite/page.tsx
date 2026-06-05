import type { Metadata } from 'next'

import { ROUTES } from '@/lib/routes'
import { LegalList, LegalSection, LegalShell } from '@/components/legal/legal-shell'

export const metadata: Metadata = {
  title: 'Politique de confidentialité — Breezy',
  description: 'Politique de confidentialité et traitement des données du réseau social Breezy.',
}

export default function ConfidentialitePage() {
  return (
    <LegalShell
      title="Politique de confidentialité"
      updatedAt="5 juin 2026"
      current={ROUTES.confidentialite}
    >
      <p className="text-sm leading-relaxed text-muted-foreground">
        La présente politique décrit la manière dont Breezy (« le Service »)
        collecte et traite vos données personnelles, conformément au Règlement
        Général sur la Protection des Données (RGPD).
      </p>

      <LegalSection title="Responsable du traitement">
        <p>
          Le responsable du traitement est l’éditeur du Service&nbsp;:{' '}
          <strong>[Raison sociale / équipe — à compléter]</strong>. Pour toute
          question, contactez&nbsp;:{' '}
          <strong>[privacy@breezy.example — à compléter]</strong>.
        </p>
      </LegalSection>

      <LegalSection title="Données collectées">
        <LegalList
          items={[
            'Données de compte : adresse e-mail, nom d’utilisateur, mot de passe (stocké de façon hachée, jamais en clair).',
            'Données de profil : nom affiché, biographie, avatar, bannière et autres informations facultatives que vous renseignez.',
            'Contenus et interactions : publications, mentions « j’aime », abonnements (graphe social).',
            'Données techniques : jetons d’authentification (JWT et jeton de rafraîchissement) et journaux techniques nécessaires au fonctionnement du Service.',
          ]}
        />
      </LegalSection>

      <LegalSection title="Finalités et bases légales">
        <LegalList
          items={[
            'Fournir le Service et gérer votre compte (exécution du contrat / des CGU).',
            'Assurer la sécurité, prévenir la fraude et modérer les contenus (intérêt légitime).',
            'Améliorer le Service (intérêt légitime).',
          ]}
        />
      </LegalSection>

      <LegalSection title="Durées de conservation">
        <p>
          Vos données sont conservées tant que votre compte est actif. En cas de
          suppression du compte, elles sont supprimées ou anonymisées dans un
          délai raisonnable, sous réserve des obligations légales de conservation.
        </p>
      </LegalSection>

      <LegalSection title="Destinataires">
        <p>
          Vos données sont traitées par les différents services internes de
          l’application (authentification, utilisateurs, profils, publications) et
          ne sont pas vendues à des tiers. Elles peuvent être communiquées aux
          autorités compétentes lorsque la loi l’exige.
        </p>
      </LegalSection>

      <LegalSection title="Cookies et stockage local">
        <LegalList
          items={[
            'Un cookie strictement nécessaire et sécurisé (httpOnly) est utilisé pour maintenir votre session (jeton de rafraîchissement).',
            'Un jeton d’accès de courte durée peut être conservé dans le stockage local de votre navigateur pour vous authentifier auprès du Service.',
            'Aucun cookie publicitaire ou de suivi tiers n’est utilisé.',
          ]}
        />
      </LegalSection>

      <LegalSection title="Sécurité">
        <p>
          Les mots de passe sont hachés (bcrypt), l’authentification repose sur
          des jetons signés et révocables, et les échanges sont destinés à
          transiter via des connexions sécurisées. Aucun système n’étant
          infaillible, une sécurité absolue ne peut être garantie.
        </p>
      </LegalSection>

      <LegalSection title="Vos droits">
        <p>
          Conformément au RGPD, vous disposez des droits suivants&nbsp;:
        </p>
        <LegalList
          items={[
            'droit d’accès à vos données ;',
            'droit de rectification ;',
            'droit à l’effacement (« droit à l’oubli ») ;',
            'droit à la limitation et à l’opposition au traitement ;',
            'droit à la portabilité de vos données.',
          ]}
        />
        <p>
          Pour exercer ces droits, écrivez à&nbsp;:{' '}
          <strong>[privacy@breezy.example — à compléter]</strong>. Vous pouvez
          également introduire une réclamation auprès de la CNIL.
        </p>
      </LegalSection>

      <LegalSection title="Modification de la politique">
        <p>
          La présente politique peut être mise à jour. La version applicable est
          celle publiée sur le Service. Pour le détail de vos obligations
          d’utilisateur, consultez les Conditions Générales d’Utilisation
          (route&nbsp;: <code>{ROUTES.cgu}</code>).
        </p>
      </LegalSection>
    </LegalShell>
  )
}
