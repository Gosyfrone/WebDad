/**
 * Contenu des pages légales (mentions légales, CGU, confidentialité), bilingue
 * FR/EN, **adapté aux fonctionnalités réelles de Breezy** : publications courtes,
 * commentaires, likes, reposts/citations, réactions emoji, graphe d'abonnements,
 * profils, recherche, rôles (utilisateur/modérateur/administrateur) et
 * **messagerie chiffrée de bout en bout** (DM, groupes, communautés).
 *
 * Modèle volontairement « plat » (paragraphes + listes en texte) → rendu par
 * `LegalShell`/`LegalSection`/`LegalList`. La navigation entre les 3 pages est
 * assurée par les liens croisés de `LegalShell` (pas de lien inline à gérer ici).
 */

import type { Locale } from '@/lib/i18n'

export type LegalSlug = 'mentions-legales' | 'cgu' | 'confidentialite'

/** Un bloc de contenu : un paragraphe (`p`) ou une liste à puces (`list`). */
export type LegalBlock = { p: string } | { list: string[] }

export interface LegalSectionData {
  title: string
  blocks: LegalBlock[]
}

export interface LegalDoc {
  title: string
  /** Texte libre « Dernière mise à jour » (déjà localisé). */
  updatedAt: string
  /** Chapô introductif (avant la 1re section). */
  intro: string
  sections: LegalSectionData[]
}

// --- Français ----------------------------------------------------------------

const FR: Record<LegalSlug, LegalDoc> = {
  'mentions-legales': {
    title: 'Mentions légales',
    updatedAt: '8 juin 2026',
    intro:
      'Breezy est un réseau social développé dans un cadre pédagogique (projet FISA INFO A3). Les informations ci-dessous sont fournies au titre de la transparence ; certaines mentions sont à compléter par l’éditeur lors d’un déploiement public.',
    sections: [
      {
        title: 'Éditeur du Service',
        blocks: [
          {
            p: 'Le site et l’application Breezy (ci-après « le Service ») sont édités par l’équipe projet Breezy (Zaid, Perujan, Candis, Théo) dans le cadre d’un projet étudiant. Contact : contact@breezy.example.',
          },
        ],
      },
      {
        title: 'Directeur de la publication',
        blocks: [
          { p: 'Le directeur de la publication est le porteur du projet pour l’équipe Breezy.' },
        ],
      },
      {
        title: 'Hébergement',
        blocks: [
          {
            p: 'Le Service est conteneurisé (Docker) et destiné à être hébergé chez un hébergeur à désigner lors d’un déploiement public. En environnement pédagogique, il s’exécute en local.',
          },
        ],
      },
      {
        title: 'Propriété intellectuelle',
        blocks: [
          {
            p: 'La marque « Breezy », le logo, la charte graphique ainsi que la structure et le contenu éditorial du Service sont protégés par le droit de la propriété intellectuelle. Toute reproduction ou représentation, totale ou partielle, sans autorisation préalable est interdite.',
          },
          {
            p: 'Les contenus publiés par les utilisateurs (publications, commentaires, messages) restent la propriété de leurs auteurs, dans les conditions prévues par les Conditions Générales d’Utilisation.',
          },
        ],
      },
      {
        title: 'Responsabilité',
        blocks: [
          {
            p: 'L’éditeur s’efforce d’assurer l’exactitude des informations diffusées mais ne saurait être tenu responsable des erreurs, d’une absence de disponibilité ou des contenus publiés par les utilisateurs. Le Service étant un projet pédagogique, il est fourni « en l’état », sans garantie de continuité.',
          },
        ],
      },
      {
        title: 'Contact',
        blocks: [
          {
            p: 'Pour toute question relative aux présentes mentions légales, vous pouvez écrire à : contact@breezy.example.',
          },
        ],
      },
    ],
  },

  cgu: {
    title: 'Conditions Générales d’Utilisation',
    updatedAt: '8 juin 2026',
    intro:
      'Les présentes Conditions Générales d’Utilisation (« CGU ») régissent l’accès et l’utilisation du réseau social Breezy (« le Service »). En créant un compte ou en utilisant le Service, vous acceptez les présentes CGU sans réserve.',
    sections: [
      {
        title: 'Article 1 — Objet',
        blocks: [
          {
            p: 'Breezy est un réseau social permettant de publier de courts messages, de commenter, d’aimer, de reposter ou citer des publications, de suivre d’autres utilisateurs, d’échanger des messages privés chiffrés et de participer à des groupes et communautés.',
          },
        ],
      },
      {
        title: 'Article 2 — Inscription et compte',
        blocks: [
          {
            list: [
              'L’inscription nécessite une adresse e-mail valide et le choix d’un nom d’utilisateur (identifiant unique).',
              'Le nom d’utilisateur constitue l’identité du compte ; sa modification peut être limitée dans le temps.',
              'Vous êtes responsable de la confidentialité de vos identifiants et de toute activité réalisée depuis votre compte.',
              'Un compte est strictement personnel. Vous vous engagez à fournir des informations exactes.',
            ],
          },
        ],
      },
      {
        title: 'Article 3 — Accès au Service',
        blocks: [
          {
            p: 'Le Service est accessible gratuitement. L’éditeur peut faire évoluer, suspendre ou interrompre tout ou partie du Service, notamment pour des raisons de maintenance, sans que sa responsabilité puisse être engagée.',
          },
        ],
      },
      {
        title: 'Article 4 — Publications et interactions',
        blocks: [
          {
            p: 'Vous pouvez publier des messages courts, y répondre par des commentaires (y compris avec des emojis), aimer, reposter ou citer des publications, et épingler l’une de vos publications sur votre profil. Une traduction automatique des publications et commentaires peut être proposée à titre indicatif.',
          },
          {
            p: 'Vous conservez la propriété de vos contenus et accordez au Service une licence non exclusive et gratuite pour les héberger et les afficher dans le cadre de son fonctionnement. Vous êtes seul responsable de ce que vous publiez.',
          },
        ],
      },
      {
        title: 'Article 5 — Messagerie privée, groupes et communautés',
        blocks: [
          {
            list: [
              'Les messages privés et les groupes sont chiffrés de bout en bout : seuls les participants peuvent en lire le contenu. L’éditeur n’a pas accès au contenu de ces échanges.',
              'Les communautés sont semi-publiques : leur nom est public et figure dans un annuaire, et leur contenu peut être accessible à un administrateur. Vous y publiez en connaissance de cette différence.',
              'La clé d’identité de chiffrement est conservée sur votre appareil ; un nouvel appareil ne peut pas déchiffrer l’historique antérieur.',
              'Les règles de conduite (Article 6) s’appliquent à tous les échanges, y compris privés.',
            ],
          },
        ],
      },
      {
        title: 'Article 6 — Règles de conduite',
        blocks: [
          { p: 'En utilisant le Service, vous vous engagez à ne pas publier de contenu :' },
          {
            list: [
              'illicite, diffamatoire, injurieux, haineux ou discriminatoire ;',
              'portant atteinte à la vie privée ou aux droits de tiers ;',
              'à caractère violent, pornographique ou incitant à la haine ;',
              'constituant du harcèlement, du spam ou de la fraude.',
            ],
          },
        ],
      },
      {
        title: 'Article 7 — Modération et rôles',
        blocks: [
          {
            p: 'Le Service distingue trois rôles : Utilisateur, Modérateur et Administrateur. Les modérateurs et administrateurs peuvent masquer ou supprimer des contenus publics contraires aux présentes CGU et, le cas échéant, suspendre les comptes concernés. Le chiffrement de bout en bout des messages privés et des groupes limite la modération de ces contenus à ce qui est techniquement accessible.',
          },
        ],
      },
      {
        title: 'Article 8 — Responsabilité et disponibilité',
        blocks: [
          {
            p: 'Le Service est fourni « en l’état », dans le cadre d’un projet pédagogique. L’éditeur ne garantit pas l’absence d’interruption ou d’erreur et ne saurait être tenu responsable des dommages résultant de l’utilisation ou de l’indisponibilité du Service.',
          },
        ],
      },
      {
        title: 'Article 9 — Suspension et résiliation',
        blocks: [
          {
            p: 'Vous pouvez supprimer votre compte à tout moment. L’éditeur peut suspendre ou résilier un compte en cas de manquement aux présentes CGU, notamment aux règles de conduite (Article 6).',
          },
        ],
      },
      {
        title: 'Article 10 — Données personnelles',
        blocks: [
          {
            p: 'Le traitement de vos données personnelles est décrit dans notre Politique de confidentialité (accessible via les liens en bas de page).',
          },
        ],
      },
      {
        title: 'Article 11 — Modification des CGU et droit applicable',
        blocks: [
          {
            p: 'L’éditeur peut modifier les présentes CGU à tout moment ; la version applicable est celle en vigueur lors de votre utilisation. Les présentes CGU sont soumises au droit français ; à défaut de résolution amiable, les tribunaux compétents seront ceux du ressort du siège de l’éditeur.',
          },
        ],
      },
    ],
  },

  confidentialite: {
    title: 'Politique de confidentialité',
    updatedAt: '8 juin 2026',
    intro:
      'La présente politique décrit la manière dont Breezy (« le Service ») collecte et traite vos données personnelles, conformément au Règlement Général sur la Protection des Données (RGPD).',
    sections: [
      {
        title: 'Responsable du traitement',
        blocks: [
          {
            p: 'Le responsable du traitement est l’éditeur du Service (équipe projet Breezy). Pour toute question : privacy@breezy.example.',
          },
        ],
      },
      {
        title: 'Données collectées',
        blocks: [
          {
            list: [
              'Données de compte : adresse e-mail, nom d’utilisateur, mot de passe (stocké haché, jamais en clair).',
              'Données de profil : nom affiché, biographie, avatar, bannière, site web, localisation, date de naissance, genre et nationalité (facultatifs).',
              'Contenus et interactions publics : publications, commentaires, likes, reposts/citations, abonnements (graphe social).',
              'Messagerie : pour les messages privés et les groupes, le serveur ne stocke que des données chiffrées (texte chiffré + clé emballée par destinataire) et n’a pas accès à leur contenu ; pour les communautés (semi-publiques), la clé de contenu est détenue par le serveur.',
              'Données techniques : jetons d’authentification (JWT et jeton de rafraîchissement) et journaux nécessaires au fonctionnement.',
            ],
          },
        ],
      },
      {
        title: 'Finalités et bases légales',
        blocks: [
          {
            list: [
              'Fournir le Service et gérer votre compte (exécution des CGU).',
              'Assurer la sécurité, prévenir la fraude et modérer les contenus publics (intérêt légitime).',
              'Améliorer le Service (intérêt légitime).',
            ],
          },
        ],
      },
      {
        title: 'Durées de conservation',
        blocks: [
          {
            p: 'Vos données sont conservées tant que votre compte est actif. En cas de suppression du compte, elles sont supprimées ou anonymisées dans un délai raisonnable, sous réserve des obligations légales de conservation.',
          },
        ],
      },
      {
        title: 'Destinataires',
        blocks: [
          {
            p: 'Vos données sont traitées par les services internes de l’application (authentification, utilisateurs, profils, publications, messagerie) et ne sont pas vendues à des tiers. Le contenu des messages privés et des groupes, chiffré de bout en bout, n’est techniquement pas accessible à l’éditeur. Vos données peuvent être communiquées aux autorités compétentes lorsque la loi l’exige.',
          },
        ],
      },
      {
        title: 'Cookies et stockage local',
        blocks: [
          {
            list: [
              'Un cookie strictement nécessaire et sécurisé (httpOnly) maintient votre session (jeton de rafraîchissement).',
              'Un jeton d’accès de courte durée peut être conservé dans le stockage local de votre navigateur pour vous authentifier.',
              'Votre clé privée de messagerie est stockée localement sur votre appareil (IndexedDB) et ne quitte jamais le navigateur.',
              'Aucun cookie publicitaire ou de suivi tiers n’est utilisé.',
            ],
          },
        ],
      },
      {
        title: 'Sécurité',
        blocks: [
          {
            p: 'Les mots de passe sont hachés (bcrypt), l’authentification repose sur des jetons signés et révocables, et la messagerie privée utilise un chiffrement de bout en bout (X25519 + XChaCha20-Poly1305) côté client. Aucun système n’étant infaillible, une sécurité absolue ne peut être garantie.',
          },
        ],
      },
      {
        title: 'Vos droits',
        blocks: [
          { p: 'Conformément au RGPD, vous disposez des droits suivants :' },
          {
            list: [
              'droit d’accès à vos données ;',
              'droit de rectification ;',
              'droit à l’effacement (« droit à l’oubli ») ;',
              'droit à la limitation et à l’opposition au traitement ;',
              'droit à la portabilité de vos données.',
            ],
          },
          {
            p: 'Pour exercer ces droits, écrivez à privacy@breezy.example. Vous pouvez également introduire une réclamation auprès de la CNIL.',
          },
        ],
      },
      {
        title: 'Modification de la politique',
        blocks: [
          {
            p: 'La présente politique peut être mise à jour. La version applicable est celle publiée sur le Service. Pour le détail de vos obligations, consultez les Conditions Générales d’Utilisation.',
          },
        ],
      },
    ],
  },
}

// --- English -----------------------------------------------------------------

const EN: Record<LegalSlug, LegalDoc> = {
  'mentions-legales': {
    title: 'Legal Notice',
    updatedAt: 'June 8, 2026',
    intro:
      'Breezy is a social network built in an academic context (FISA INFO A3 project). The information below is provided for transparency; some entries are to be completed by the publisher for a public deployment.',
    sections: [
      {
        title: 'Publisher',
        blocks: [
          {
            p: 'The Breezy website and application (the “Service”) are published by the Breezy project team (Zaid, Perujan, Candis, Théo) as a student project. Contact: contact@breezy.example.',
          },
        ],
      },
      {
        title: 'Publication director',
        blocks: [{ p: 'The publication director is the project lead for the Breezy team.' }],
      },
      {
        title: 'Hosting',
        blocks: [
          {
            p: 'The Service is containerized (Docker) and intended to be hosted with a provider to be designated for a public deployment. In an academic setting, it runs locally.',
          },
        ],
      },
      {
        title: 'Intellectual property',
        blocks: [
          {
            p: 'The “Breezy” brand, logo, visual identity, structure and editorial content of the Service are protected by intellectual property law. Any reproduction or representation, in whole or in part, without prior authorization is prohibited.',
          },
          {
            p: 'Content published by users (posts, comments, messages) remains the property of its authors, under the conditions set out in the Terms of Use.',
          },
        ],
      },
      {
        title: 'Liability',
        blocks: [
          {
            p: 'The publisher strives to ensure the accuracy of the information provided but cannot be held liable for errors, unavailability, or content published by users. As an academic project, the Service is provided “as is”, with no guarantee of continuity.',
          },
        ],
      },
      {
        title: 'Contact',
        blocks: [
          {
            p: 'For any question regarding this legal notice, you may write to: contact@breezy.example.',
          },
        ],
      },
    ],
  },

  cgu: {
    title: 'Terms of Use',
    updatedAt: 'June 8, 2026',
    intro:
      'These Terms of Use (“Terms”) govern access to and use of the Breezy social network (the “Service”). By creating an account or using the Service, you fully accept these Terms.',
    sections: [
      {
        title: 'Article 1 — Purpose',
        blocks: [
          {
            p: 'Breezy is a social network that lets you publish short posts, comment, like, repost or quote posts, follow other users, exchange end-to-end encrypted private messages, and take part in groups and communities.',
          },
        ],
      },
      {
        title: 'Article 2 — Registration and account',
        blocks: [
          {
            list: [
              'Registration requires a valid e-mail address and the choice of a username (unique handle).',
              'The username is the account’s identity; changing it may be rate-limited over time.',
              'You are responsible for keeping your credentials confidential and for all activity from your account.',
              'An account is strictly personal. You agree to provide accurate information.',
            ],
          },
        ],
      },
      {
        title: 'Article 3 — Access to the Service',
        blocks: [
          {
            p: 'The Service is free of charge. The publisher may change, suspend or discontinue all or part of the Service, in particular for maintenance, without incurring liability.',
          },
        ],
      },
      {
        title: 'Article 4 — Posts and interactions',
        blocks: [
          {
            p: 'You may publish short posts, reply with comments (including emojis), like, repost or quote posts, and pin one of your posts to your profile. Automatic translation of posts and comments may be offered for guidance only.',
          },
          {
            p: 'You retain ownership of your content and grant the Service a non-exclusive, free licence to host and display it as part of its operation. You are solely responsible for what you publish.',
          },
        ],
      },
      {
        title: 'Article 5 — Private messaging, groups and communities',
        blocks: [
          {
            list: [
              'Private messages and groups are end-to-end encrypted: only participants can read their content. The publisher has no access to the content of these exchanges.',
              'Communities are semi-public: their name is public and listed in a directory, and their content may be accessible to an administrator. You post there knowing this difference.',
              'Your encryption identity key is stored on your device; a new device cannot decrypt prior history.',
              'The conduct rules (Article 6) apply to all exchanges, including private ones.',
            ],
          },
        ],
      },
      {
        title: 'Article 6 — Conduct rules',
        blocks: [
          { p: 'When using the Service, you agree not to publish content that is:' },
          {
            list: [
              'unlawful, defamatory, abusive, hateful or discriminatory;',
              'harmful to the privacy or rights of third parties;',
              'violent, pornographic or inciting hatred;',
              'harassment, spam or fraud.',
            ],
          },
        ],
      },
      {
        title: 'Article 7 — Moderation and roles',
        blocks: [
          {
            p: 'The Service has three roles: User, Moderator and Administrator. Moderators and administrators may hide or remove public content that breaches these Terms and, where appropriate, suspend the accounts concerned. End-to-end encryption of private messages and groups limits moderation of that content to what is technically accessible.',
          },
        ],
      },
      {
        title: 'Article 8 — Liability and availability',
        blocks: [
          {
            p: 'The Service is provided “as is”, as an academic project. The publisher does not guarantee the absence of interruptions or errors and cannot be held liable for damages resulting from the use or unavailability of the Service.',
          },
        ],
      },
      {
        title: 'Article 9 — Suspension and termination',
        blocks: [
          {
            p: 'You may delete your account at any time. The publisher may suspend or terminate an account in the event of a breach of these Terms, in particular the conduct rules (Article 6).',
          },
        ],
      },
      {
        title: 'Article 10 — Personal data',
        blocks: [
          {
            p: 'The processing of your personal data is described in our Privacy Policy (accessible from the links at the bottom of the page).',
          },
        ],
      },
      {
        title: 'Article 11 — Changes to the Terms and governing law',
        blocks: [
          {
            p: 'The publisher may amend these Terms at any time; the applicable version is the one in force when you use the Service. These Terms are governed by French law; failing an amicable resolution, the competent courts shall be those of the publisher’s registered office.',
          },
        ],
      },
    ],
  },

  confidentialite: {
    title: 'Privacy Policy',
    updatedAt: 'June 8, 2026',
    intro:
      'This policy describes how Breezy (the “Service”) collects and processes your personal data, in accordance with the General Data Protection Regulation (GDPR).',
    sections: [
      {
        title: 'Data controller',
        blocks: [
          {
            p: 'The data controller is the publisher of the Service (the Breezy project team). For any question: privacy@breezy.example.',
          },
        ],
      },
      {
        title: 'Data collected',
        blocks: [
          {
            list: [
              'Account data: e-mail address, username, password (stored hashed, never in clear text).',
              'Profile data: display name, biography, avatar, banner, website, location, date of birth, gender and nationality (optional).',
              'Public content and interactions: posts, comments, likes, reposts/quotes, follows (social graph).',
              'Messaging: for private messages and groups, the server stores only encrypted data (ciphertext + per-recipient wrapped key) and has no access to their content; for communities (semi-public), the content key is held by the server.',
              'Technical data: authentication tokens (JWT and refresh token) and logs required for operation.',
            ],
          },
        ],
      },
      {
        title: 'Purposes and legal bases',
        blocks: [
          {
            list: [
              'Provide the Service and manage your account (performance of the Terms).',
              'Ensure security, prevent fraud and moderate public content (legitimate interest).',
              'Improve the Service (legitimate interest).',
            ],
          },
        ],
      },
      {
        title: 'Retention periods',
        blocks: [
          {
            p: 'Your data is kept for as long as your account is active. If the account is deleted, it is removed or anonymized within a reasonable time, subject to legal retention obligations.',
          },
        ],
      },
      {
        title: 'Recipients',
        blocks: [
          {
            p: 'Your data is processed by the application’s internal services (authentication, users, profiles, posts, messaging) and is not sold to third parties. The content of private messages and groups, being end-to-end encrypted, is technically inaccessible to the publisher. Your data may be disclosed to the competent authorities where required by law.',
          },
        ],
      },
      {
        title: 'Cookies and local storage',
        blocks: [
          {
            list: [
              'A strictly necessary, secure (httpOnly) cookie maintains your session (refresh token).',
              'A short-lived access token may be kept in your browser’s local storage to authenticate you.',
              'Your messaging private key is stored locally on your device (IndexedDB) and never leaves the browser.',
              'No advertising or third-party tracking cookies are used.',
            ],
          },
        ],
      },
      {
        title: 'Security',
        blocks: [
          {
            p: 'Passwords are hashed (bcrypt), authentication relies on signed and revocable tokens, and private messaging uses client-side end-to-end encryption (X25519 + XChaCha20-Poly1305). As no system is infallible, absolute security cannot be guaranteed.',
          },
        ],
      },
      {
        title: 'Your rights',
        blocks: [
          { p: 'Under the GDPR, you have the following rights:' },
          {
            list: [
              'right of access to your data;',
              'right to rectification;',
              'right to erasure (“right to be forgotten”);',
              'right to restriction of and objection to processing;',
              'right to data portability.',
            ],
          },
          {
            p: 'To exercise these rights, write to privacy@breezy.example. You may also lodge a complaint with the relevant data protection authority.',
          },
        ],
      },
      {
        title: 'Changes to the policy',
        blocks: [
          {
            p: 'This policy may be updated. The applicable version is the one published on the Service. For details of your obligations, see the Terms of Use.',
          },
        ],
      },
    ],
  },
}

/** Registre du contenu légal, par locale puis par page. */
export const legalContent: Partial<Record<Locale, Record<LegalSlug, LegalDoc>>> & {
  fr: Record<LegalSlug, LegalDoc>
  en: Record<LegalSlug, LegalDoc>
} = {
  fr: FR,
  en: EN,
}
