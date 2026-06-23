/**
 * Contenu des pages légales (mentions légales, CGU, confidentialité), traduit
 * dans les 12 locales (fr/en/zh/es/pt/ru/ja/ko/ar/hi/de/it),
 * **adapté aux fonctionnalités réelles de Breezy** : publications courtes,
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
            p: 'Le site et l’application Breezy (ci-après « le Service ») sont édités par l’équipe projet Breezy (Philippe, Alexandre, Maxime, Romain) dans le cadre d’un projet étudiant. Contact : contact@breezy.example.',
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
            p: 'The Breezy website and application (the “Service”) are published by the Breezy project team (Philippe, Alexandre, Maxime, Romain) as a student project. Contact: contact@breezy.example.',
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

// --- Deutsch -----------------------------------------------------------------

const DE: Record<LegalSlug, LegalDoc> = {
  'mentions-legales': {
    title: 'Rechtlicher Hinweis',
    updatedAt: '8. Juni 2026',
    intro:
      'Breezy ist ein soziales Netzwerk, das im Rahmen eines Studienprojekts (Projekt FISA INFO A3) entwickelt wurde. Die nachstehenden Angaben dienen der Transparenz; einige Angaben sind vom Herausgeber bei einer öffentlichen Bereitstellung zu ergänzen.',
    sections: [
      {
        title: 'Herausgeber des Dienstes',
        blocks: [
          {
            p: 'Die Website und die Anwendung Breezy (nachfolgend „der Dienst“) werden vom Breezy-Projektteam (Philippe, Alexandre, Maxime, Romain) im Rahmen eines studentischen Projekts herausgegeben. Kontakt: contact@breezy.example.',
          },
        ],
      },
      {
        title: 'Verantwortlich für die Veröffentlichung',
        blocks: [
          { p: 'Verantwortlich für die Veröffentlichung ist die Projektleitung des Breezy-Teams.' },
        ],
      },
      {
        title: 'Hosting',
        blocks: [
          {
            p: 'Der Dienst ist containerisiert (Docker) und soll bei einer öffentlichen Bereitstellung bei einem noch zu benennenden Hoster gehostet werden. In einer Lernumgebung läuft er lokal.',
          },
        ],
      },
      {
        title: 'Geistiges Eigentum',
        blocks: [
          {
            p: 'Die Marke „Breezy“, das Logo, das Erscheinungsbild sowie die Struktur und der redaktionelle Inhalt des Dienstes sind durch das Recht des geistigen Eigentums geschützt. Jede vollständige oder teilweise Vervielfältigung oder Wiedergabe ohne vorherige Genehmigung ist untersagt.',
          },
          {
            p: 'Die von den Nutzern veröffentlichten Inhalte (Beiträge, Kommentare, Nachrichten) bleiben Eigentum ihrer Urheber, unter den in den Nutzungsbedingungen festgelegten Bedingungen.',
          },
        ],
      },
      {
        title: 'Haftung',
        blocks: [
          {
            p: 'Der Herausgeber bemüht sich um die Richtigkeit der bereitgestellten Informationen, kann jedoch nicht für Fehler, eine fehlende Verfügbarkeit oder von Nutzern veröffentlichte Inhalte haftbar gemacht werden. Da es sich um ein Studienprojekt handelt, wird der Dienst „wie besehen“ und ohne Gewähr für Kontinuität bereitgestellt.',
          },
        ],
      },
      {
        title: 'Kontakt',
        blocks: [
          {
            p: 'Bei Fragen zu diesem rechtlichen Hinweis können Sie schreiben an: contact@breezy.example.',
          },
        ],
      },
    ],
  },

  cgu: {
    title: 'Nutzungsbedingungen',
    updatedAt: '8. Juni 2026',
    intro:
      'Diese Nutzungsbedingungen („Bedingungen“) regeln den Zugang zum und die Nutzung des sozialen Netzwerks Breezy („der Dienst“). Mit der Erstellung eines Kontos oder der Nutzung des Dienstes akzeptieren Sie diese Bedingungen vorbehaltlos.',
    sections: [
      {
        title: 'Artikel 1 — Gegenstand',
        blocks: [
          {
            p: 'Breezy ist ein soziales Netzwerk, mit dem Sie kurze Beiträge veröffentlichen, kommentieren, liken, Beiträge teilen (Repost) oder zitieren, anderen Nutzern folgen, Ende-zu-Ende-verschlüsselte private Nachrichten austauschen und an Gruppen und Communitys teilnehmen können.',
          },
        ],
      },
      {
        title: 'Artikel 2 — Registrierung und Konto',
        blocks: [
          {
            list: [
              'Für die Registrierung sind eine gültige E-Mail-Adresse und die Wahl eines Benutzernamens (eindeutige Kennung) erforderlich.',
              'Der Benutzername bildet die Identität des Kontos; seine Änderung kann zeitlich begrenzt sein.',
              'Sie sind für die Vertraulichkeit Ihrer Zugangsdaten und für jede Aktivität über Ihr Konto verantwortlich.',
              'Ein Konto ist streng persönlich. Sie verpflichten sich, korrekte Angaben zu machen.',
            ],
          },
        ],
      },
      {
        title: 'Artikel 3 — Zugang zum Dienst',
        blocks: [
          {
            p: 'Der Dienst ist kostenlos zugänglich. Der Herausgeber kann den Dienst ganz oder teilweise weiterentwickeln, aussetzen oder einstellen, insbesondere aus Wartungsgründen, ohne dass dies eine Haftung begründet.',
          },
        ],
      },
      {
        title: 'Artikel 4 — Beiträge und Interaktionen',
        blocks: [
          {
            p: 'Sie können kurze Beiträge veröffentlichen, mit Kommentaren (auch mit Emojis) darauf antworten, Beiträge liken, teilen oder zitieren und einen Ihrer Beiträge an Ihr Profil anheften. Eine automatische Übersetzung von Beiträgen und Kommentaren kann zu Informationszwecken angeboten werden.',
          },
          {
            p: 'Sie behalten das Eigentum an Ihren Inhalten und gewähren dem Dienst eine nicht-exklusive, kostenlose Lizenz, um sie im Rahmen seines Betriebs zu hosten und anzuzeigen. Sie sind allein für das verantwortlich, was Sie veröffentlichen.',
          },
        ],
      },
      {
        title: 'Artikel 5 — Private Nachrichten, Gruppen und Communitys',
        blocks: [
          {
            list: [
              'Private Nachrichten und Gruppen sind Ende-zu-Ende-verschlüsselt: Nur die Teilnehmer können deren Inhalt lesen. Der Herausgeber hat keinen Zugriff auf den Inhalt dieser Austausche.',
              'Communitys sind halböffentlich: Ihr Name ist öffentlich und in einem Verzeichnis aufgeführt, und ihr Inhalt kann für einen Administrator zugänglich sein. Sie veröffentlichen dort in Kenntnis dieses Unterschieds.',
              'Ihr Verschlüsselungs-Identitätsschlüssel wird auf Ihrem Gerät gespeichert; ein neues Gerät kann den früheren Verlauf nicht entschlüsseln.',
              'Die Verhaltensregeln (Artikel 6) gelten für alle Austausche, auch für private.',
            ],
          },
        ],
      },
      {
        title: 'Artikel 6 — Verhaltensregeln',
        blocks: [
          { p: 'Bei der Nutzung des Dienstes verpflichten Sie sich, keine Inhalte zu veröffentlichen, die:' },
          {
            list: [
              'rechtswidrig, verleumderisch, beleidigend, hasserfüllt oder diskriminierend sind;',
              'die Privatsphäre oder die Rechte Dritter verletzen;',
              'gewalttätigen, pornografischen oder zu Hass aufrufenden Charakter haben;',
              'Belästigung, Spam oder Betrug darstellen.',
            ],
          },
        ],
      },
      {
        title: 'Artikel 7 — Moderation und Rollen',
        blocks: [
          {
            p: 'Der Dienst unterscheidet drei Rollen: Nutzer, Moderator und Administrator. Moderatoren und Administratoren können öffentliche Inhalte, die gegen diese Bedingungen verstoßen, ausblenden oder entfernen und gegebenenfalls die betreffenden Konten sperren. Die Ende-zu-Ende-Verschlüsselung privater Nachrichten und Gruppen beschränkt die Moderation dieser Inhalte auf das technisch Zugängliche.',
          },
        ],
      },
      {
        title: 'Artikel 8 — Haftung und Verfügbarkeit',
        blocks: [
          {
            p: 'Der Dienst wird „wie besehen“ im Rahmen eines Studienprojekts bereitgestellt. Der Herausgeber garantiert weder das Ausbleiben von Unterbrechungen oder Fehlern noch kann er für Schäden haftbar gemacht werden, die aus der Nutzung oder Nichtverfügbarkeit des Dienstes entstehen.',
          },
        ],
      },
      {
        title: 'Artikel 9 — Sperrung und Kündigung',
        blocks: [
          {
            p: 'Sie können Ihr Konto jederzeit löschen. Der Herausgeber kann ein Konto bei Verstoß gegen diese Bedingungen sperren oder kündigen, insbesondere gegen die Verhaltensregeln (Artikel 6).',
          },
        ],
      },
      {
        title: 'Artikel 10 — Personenbezogene Daten',
        blocks: [
          {
            p: 'Die Verarbeitung Ihrer personenbezogenen Daten wird in unserer Datenschutzrichtlinie beschrieben (zugänglich über die Links am Seitenende).',
          },
        ],
      },
      {
        title: 'Artikel 11 — Änderung der Bedingungen und anwendbares Recht',
        blocks: [
          {
            p: 'Der Herausgeber kann diese Bedingungen jederzeit ändern; maßgeblich ist die zum Zeitpunkt Ihrer Nutzung geltende Fassung. Diese Bedingungen unterliegen französischem Recht; mangels gütlicher Einigung sind die Gerichte am Sitz des Herausgebers zuständig.',
          },
        ],
      },
    ],
  },

  confidentialite: {
    title: 'Datenschutzrichtlinie',
    updatedAt: '8. Juni 2026',
    intro:
      'Diese Richtlinie beschreibt, wie Breezy („der Dienst“) Ihre personenbezogenen Daten gemäß der Datenschutz-Grundverordnung (DSGVO) erhebt und verarbeitet.',
    sections: [
      {
        title: 'Verantwortlicher für die Verarbeitung',
        blocks: [
          {
            p: 'Verantwortlicher für die Verarbeitung ist der Herausgeber des Dienstes (Breezy-Projektteam). Bei Fragen: privacy@breezy.example.',
          },
        ],
      },
      {
        title: 'Erhobene Daten',
        blocks: [
          {
            list: [
              'Kontodaten: E-Mail-Adresse, Benutzername, Passwort (gehasht gespeichert, niemals im Klartext).',
              'Profildaten: Anzeigename, Biografie, Avatar, Banner, Website, Standort, Geburtsdatum, Geschlecht und Nationalität (optional).',
              'Öffentliche Inhalte und Interaktionen: Beiträge, Kommentare, Likes, Reposts/Zitate, Abonnements (soziales Netz).',
              'Nachrichten: Bei privaten Nachrichten und Gruppen speichert der Server nur verschlüsselte Daten (verschlüsselter Text + pro Empfänger verpackter Schlüssel) und hat keinen Zugriff auf deren Inhalt; bei Communitys (halböffentlich) liegt der Inhaltsschlüssel beim Server.',
              'Technische Daten: Authentifizierungstoken (JWT und Refresh-Token) und für den Betrieb erforderliche Protokolle.',
            ],
          },
        ],
      },
      {
        title: 'Zwecke und Rechtsgrundlagen',
        blocks: [
          {
            list: [
              'Bereitstellung des Dienstes und Verwaltung Ihres Kontos (Erfüllung der Bedingungen).',
              'Gewährleistung der Sicherheit, Betrugsprävention und Moderation öffentlicher Inhalte (berechtigtes Interesse).',
              'Verbesserung des Dienstes (berechtigtes Interesse).',
            ],
          },
        ],
      },
      {
        title: 'Speicherdauer',
        blocks: [
          {
            p: 'Ihre Daten werden gespeichert, solange Ihr Konto aktiv ist. Bei Löschung des Kontos werden sie innerhalb einer angemessenen Frist gelöscht oder anonymisiert, vorbehaltlich gesetzlicher Aufbewahrungspflichten.',
          },
        ],
      },
      {
        title: 'Empfänger',
        blocks: [
          {
            p: 'Ihre Daten werden von den internen Diensten der Anwendung (Authentifizierung, Nutzer, Profile, Beiträge, Nachrichten) verarbeitet und nicht an Dritte verkauft. Der Inhalt privater Nachrichten und Gruppen ist Ende-zu-Ende-verschlüsselt und für den Herausgeber technisch nicht zugänglich. Ihre Daten können den zuständigen Behörden mitgeteilt werden, wenn das Gesetz dies verlangt.',
          },
        ],
      },
      {
        title: 'Cookies und lokale Speicherung',
        blocks: [
          {
            list: [
              'Ein unbedingt erforderliches, sicheres (httpOnly) Cookie hält Ihre Sitzung aufrecht (Refresh-Token).',
              'Ein kurzlebiges Zugriffstoken kann im lokalen Speicher Ihres Browsers aufbewahrt werden, um Sie zu authentifizieren.',
              'Ihr privater Nachrichtenschlüssel wird lokal auf Ihrem Gerät gespeichert (IndexedDB) und verlässt niemals den Browser.',
              'Es werden keine Werbe- oder Tracking-Cookies von Drittanbietern verwendet.',
            ],
          },
        ],
      },
      {
        title: 'Sicherheit',
        blocks: [
          {
            p: 'Passwörter werden gehasht (bcrypt), die Authentifizierung beruht auf signierten und widerrufbaren Token, und private Nachrichten nutzen eine clientseitige Ende-zu-Ende-Verschlüsselung (X25519 + XChaCha20-Poly1305). Da kein System unfehlbar ist, kann keine absolute Sicherheit garantiert werden.',
          },
        ],
      },
      {
        title: 'Ihre Rechte',
        blocks: [
          { p: 'Gemäß der DSGVO haben Sie folgende Rechte:' },
          {
            list: [
              'Recht auf Auskunft über Ihre Daten;',
              'Recht auf Berichtigung;',
              'Recht auf Löschung („Recht auf Vergessenwerden“);',
              'Recht auf Einschränkung der und Widerspruch gegen die Verarbeitung;',
              'Recht auf Datenübertragbarkeit.',
            ],
          },
          {
            p: 'Um diese Rechte auszuüben, schreiben Sie an privacy@breezy.example. Sie können auch eine Beschwerde bei der zuständigen Datenschutzbehörde einreichen.',
          },
        ],
      },
      {
        title: 'Änderung der Richtlinie',
        blocks: [
          {
            p: 'Diese Richtlinie kann aktualisiert werden. Maßgeblich ist die auf dem Dienst veröffentlichte Fassung. Einzelheiten zu Ihren Pflichten finden Sie in den Nutzungsbedingungen.',
          },
        ],
      },
    ],
  },
}

// --- Español -----------------------------------------------------------------

const ES: Record<LegalSlug, LegalDoc> = {
  'mentions-legales': {
    title: 'Aviso legal',
    updatedAt: '8 de junio de 2026',
    intro:
      'Breezy es una red social desarrollada en un contexto académico (proyecto FISA INFO A3). La información que figura a continuación se facilita por motivos de transparencia; algunas menciones deberá completarlas el editor en caso de despliegue público.',
    sections: [
      {
        title: 'Editor del Servicio',
        blocks: [
          {
            p: 'El sitio web y la aplicación Breezy (en adelante, «el Servicio») son editados por el equipo del proyecto Breezy (Philippe, Alexandre, Maxime, Romain) en el marco de un proyecto estudiantil. Contacto: contact@breezy.example.',
          },
        ],
      },
      {
        title: 'Director de la publicación',
        blocks: [
          { p: 'El director de la publicación es el responsable del proyecto por parte del equipo Breezy.' },
        ],
      },
      {
        title: 'Alojamiento',
        blocks: [
          {
            p: 'El Servicio está contenedorizado (Docker) y está destinado a ser alojado en un proveedor que se designará en caso de despliegue público. En un entorno académico, se ejecuta de forma local.',
          },
        ],
      },
      {
        title: 'Propiedad intelectual',
        blocks: [
          {
            p: 'La marca «Breezy», el logotipo, la identidad gráfica, así como la estructura y el contenido editorial del Servicio están protegidos por el derecho de propiedad intelectual. Queda prohibida toda reproducción o representación, total o parcial, sin autorización previa.',
          },
          {
            p: 'Los contenidos publicados por los usuarios (publicaciones, comentarios, mensajes) siguen siendo propiedad de sus autores, en las condiciones previstas por las Condiciones de Uso.',
          },
        ],
      },
      {
        title: 'Responsabilidad',
        blocks: [
          {
            p: 'El editor se esfuerza por garantizar la exactitud de la información difundida, pero no puede ser considerado responsable de los errores, de una falta de disponibilidad o de los contenidos publicados por los usuarios. Al tratarse de un proyecto académico, el Servicio se presta «tal cual», sin garantía de continuidad.',
          },
        ],
      },
      {
        title: 'Contacto',
        blocks: [
          {
            p: 'Para cualquier consulta relativa a este aviso legal, puede escribir a: contact@breezy.example.',
          },
        ],
      },
    ],
  },

  cgu: {
    title: 'Condiciones de Uso',
    updatedAt: '8 de junio de 2026',
    intro:
      'Las presentes Condiciones de Uso («Condiciones») rigen el acceso y el uso de la red social Breezy («el Servicio»). Al crear una cuenta o utilizar el Servicio, acepta estas Condiciones sin reservas.',
    sections: [
      {
        title: 'Artículo 1 — Objeto',
        blocks: [
          {
            p: 'Breezy es una red social que permite publicar mensajes cortos, comentar, dar «me gusta», compartir (repost) o citar publicaciones, seguir a otros usuarios, intercambiar mensajes privados cifrados y participar en grupos y comunidades.',
          },
        ],
      },
      {
        title: 'Artículo 2 — Registro y cuenta',
        blocks: [
          {
            list: [
              'El registro requiere una dirección de correo electrónico válida y la elección de un nombre de usuario (identificador único).',
              'El nombre de usuario constituye la identidad de la cuenta; su modificación puede estar limitada en el tiempo.',
              'Usted es responsable de la confidencialidad de sus credenciales y de toda actividad realizada desde su cuenta.',
              'Una cuenta es estrictamente personal. Se compromete a facilitar información exacta.',
            ],
          },
        ],
      },
      {
        title: 'Artículo 3 — Acceso al Servicio',
        blocks: [
          {
            p: 'El Servicio es de acceso gratuito. El editor puede modificar, suspender o interrumpir la totalidad o parte del Servicio, en particular por motivos de mantenimiento, sin que ello genere responsabilidad alguna.',
          },
        ],
      },
      {
        title: 'Artículo 4 — Publicaciones e interacciones',
        blocks: [
          {
            p: 'Puede publicar mensajes cortos, responder con comentarios (incluso con emojis), dar «me gusta», compartir o citar publicaciones y fijar una de sus publicaciones en su perfil. Puede ofrecerse una traducción automática de las publicaciones y los comentarios a título orientativo.',
          },
          {
            p: 'Usted conserva la propiedad de sus contenidos y concede al Servicio una licencia no exclusiva y gratuita para alojarlos y mostrarlos en el marco de su funcionamiento. Usted es el único responsable de lo que publica.',
          },
        ],
      },
      {
        title: 'Artículo 5 — Mensajería privada, grupos y comunidades',
        blocks: [
          {
            list: [
              'Los mensajes privados y los grupos están cifrados de extremo a extremo: solo los participantes pueden leer su contenido. El editor no tiene acceso al contenido de estos intercambios.',
              'Las comunidades son semipúblicas: su nombre es público y figura en un directorio, y su contenido puede ser accesible para un administrador. Usted publica en ellas conociendo esta diferencia.',
              'La clave de identidad de cifrado se conserva en su dispositivo; un nuevo dispositivo no puede descifrar el historial anterior.',
              'Las reglas de conducta (Artículo 6) se aplican a todos los intercambios, incluidos los privados.',
            ],
          },
        ],
      },
      {
        title: 'Artículo 6 — Reglas de conducta',
        blocks: [
          { p: 'Al utilizar el Servicio, se compromete a no publicar contenido:' },
          {
            list: [
              'ilícito, difamatorio, injurioso, que incite al odio o discriminatorio;',
              'que atente contra la vida privada o los derechos de terceros;',
              'de carácter violento, pornográfico o que incite al odio;',
              'que constituya acoso, spam o fraude.',
            ],
          },
        ],
      },
      {
        title: 'Artículo 7 — Moderación y roles',
        blocks: [
          {
            p: 'El Servicio distingue tres roles: Usuario, Moderador y Administrador. Los moderadores y administradores pueden ocultar o eliminar contenidos públicos contrarios a las presentes Condiciones y, en su caso, suspender las cuentas implicadas. El cifrado de extremo a extremo de los mensajes privados y los grupos limita la moderación de dichos contenidos a lo que sea técnicamente accesible.',
          },
        ],
      },
      {
        title: 'Artículo 8 — Responsabilidad y disponibilidad',
        blocks: [
          {
            p: 'El Servicio se presta «tal cual», en el marco de un proyecto académico. El editor no garantiza la ausencia de interrupciones o errores y no puede ser considerado responsable de los daños resultantes del uso o de la indisponibilidad del Servicio.',
          },
        ],
      },
      {
        title: 'Artículo 9 — Suspensión y resolución',
        blocks: [
          {
            p: 'Puede eliminar su cuenta en cualquier momento. El editor puede suspender o cancelar una cuenta en caso de incumplimiento de las presentes Condiciones, en particular de las reglas de conducta (Artículo 6).',
          },
        ],
      },
      {
        title: 'Artículo 10 — Datos personales',
        blocks: [
          {
            p: 'El tratamiento de sus datos personales se describe en nuestra Política de privacidad (accesible a través de los enlaces al pie de página).',
          },
        ],
      },
      {
        title: 'Artículo 11 — Modificación de las Condiciones y legislación aplicable',
        blocks: [
          {
            p: 'El editor puede modificar las presentes Condiciones en cualquier momento; la versión aplicable es la vigente en el momento de su uso. Las presentes Condiciones se rigen por el derecho francés; a falta de resolución amistosa, los tribunales competentes serán los del domicilio social del editor.',
          },
        ],
      },
    ],
  },

  confidentialite: {
    title: 'Política de privacidad',
    updatedAt: '8 de junio de 2026',
    intro:
      'La presente política describe la manera en que Breezy («el Servicio») recopila y trata sus datos personales, de conformidad con el Reglamento General de Protección de Datos (RGPD).',
    sections: [
      {
        title: 'Responsable del tratamiento',
        blocks: [
          {
            p: 'El responsable del tratamiento es el editor del Servicio (equipo del proyecto Breezy). Para cualquier consulta: privacy@breezy.example.',
          },
        ],
      },
      {
        title: 'Datos recopilados',
        blocks: [
          {
            list: [
              'Datos de cuenta: dirección de correo electrónico, nombre de usuario, contraseña (almacenada cifrada mediante hash, nunca en texto claro).',
              'Datos de perfil: nombre mostrado, biografía, avatar, banner, sitio web, ubicación, fecha de nacimiento, género y nacionalidad (opcionales).',
              'Contenidos e interacciones públicos: publicaciones, comentarios, «me gusta», reposts/citas, seguimientos (grafo social).',
              'Mensajería: para los mensajes privados y los grupos, el servidor solo almacena datos cifrados (texto cifrado + clave envuelta por destinatario) y no tiene acceso a su contenido; para las comunidades (semipúblicas), la clave de contenido está en poder del servidor.',
              'Datos técnicos: tokens de autenticación (JWT y token de actualización) y registros necesarios para el funcionamiento.',
            ],
          },
        ],
      },
      {
        title: 'Finalidades y bases legales',
        blocks: [
          {
            list: [
              'Prestar el Servicio y gestionar su cuenta (ejecución de las Condiciones).',
              'Garantizar la seguridad, prevenir el fraude y moderar los contenidos públicos (interés legítimo).',
              'Mejorar el Servicio (interés legítimo).',
            ],
          },
        ],
      },
      {
        title: 'Plazos de conservación',
        blocks: [
          {
            p: 'Sus datos se conservan mientras su cuenta esté activa. En caso de eliminación de la cuenta, se suprimen o anonimizan en un plazo razonable, sin perjuicio de las obligaciones legales de conservación.',
          },
        ],
      },
      {
        title: 'Destinatarios',
        blocks: [
          {
            p: 'Sus datos son tratados por los servicios internos de la aplicación (autenticación, usuarios, perfiles, publicaciones, mensajería) y no se venden a terceros. El contenido de los mensajes privados y los grupos, cifrado de extremo a extremo, no es técnicamente accesible para el editor. Sus datos pueden ser comunicados a las autoridades competentes cuando la ley lo exija.',
          },
        ],
      },
      {
        title: 'Cookies y almacenamiento local',
        blocks: [
          {
            list: [
              'Una cookie estrictamente necesaria y segura (httpOnly) mantiene su sesión (token de actualización).',
              'Un token de acceso de corta duración puede conservarse en el almacenamiento local de su navegador para autenticarle.',
              'Su clave privada de mensajería se almacena localmente en su dispositivo (IndexedDB) y nunca abandona el navegador.',
              'No se utiliza ninguna cookie publicitaria ni de seguimiento de terceros.',
            ],
          },
        ],
      },
      {
        title: 'Seguridad',
        blocks: [
          {
            p: 'Las contraseñas se cifran mediante hash (bcrypt), la autenticación se basa en tokens firmados y revocables, y la mensajería privada utiliza un cifrado de extremo a extremo (X25519 + XChaCha20-Poly1305) del lado del cliente. Dado que ningún sistema es infalible, no puede garantizarse una seguridad absoluta.',
          },
        ],
      },
      {
        title: 'Sus derechos',
        blocks: [
          { p: 'De conformidad con el RGPD, dispone de los siguientes derechos:' },
          {
            list: [
              'derecho de acceso a sus datos;',
              'derecho de rectificación;',
              'derecho de supresión («derecho al olvido»);',
              'derecho a la limitación del tratamiento y a oponerse a él;',
              'derecho a la portabilidad de sus datos.',
            ],
          },
          {
            p: 'Para ejercer estos derechos, escriba a privacy@breezy.example. También puede presentar una reclamación ante la autoridad de protección de datos competente.',
          },
        ],
      },
      {
        title: 'Modificación de la política',
        blocks: [
          {
            p: 'La presente política puede actualizarse. La versión aplicable es la publicada en el Servicio. Para conocer el detalle de sus obligaciones, consulte las Condiciones de Uso.',
          },
        ],
      },
    ],
  },
}

// --- Português ---------------------------------------------------------------

const PT: Record<LegalSlug, LegalDoc> = {
  'mentions-legales': {
    title: 'Aviso legal',
    updatedAt: '8 de junho de 2026',
    intro:
      'O Breezy é uma rede social desenvolvida num contexto académico (projeto FISA INFO A3). As informações abaixo são fornecidas a título de transparência; algumas menções deverão ser completadas pelo editor em caso de implementação pública.',
    sections: [
      {
        title: 'Editor do Serviço',
        blocks: [
          {
            p: 'O site e a aplicação Breezy (doravante «o Serviço») são editados pela equipa do projeto Breezy (Philippe, Alexandre, Maxime, Romain) no âmbito de um projeto estudantil. Contacto: contact@breezy.example.',
          },
        ],
      },
      {
        title: 'Diretor da publicação',
        blocks: [
          { p: 'O diretor da publicação é o responsável do projeto pela equipa Breezy.' },
        ],
      },
      {
        title: 'Alojamento',
        blocks: [
          {
            p: 'O Serviço está em contentores (Docker) e destina-se a ser alojado num fornecedor a designar em caso de implementação pública. Em ambiente académico, é executado localmente.',
          },
        ],
      },
      {
        title: 'Propriedade intelectual',
        blocks: [
          {
            p: 'A marca «Breezy», o logótipo, a identidade gráfica, bem como a estrutura e o conteúdo editorial do Serviço estão protegidos pelo direito de propriedade intelectual. É proibida qualquer reprodução ou representação, total ou parcial, sem autorização prévia.',
          },
          {
            p: 'Os conteúdos publicados pelos utilizadores (publicações, comentários, mensagens) continuam a ser propriedade dos seus autores, nas condições previstas pelos Termos de Uso.',
          },
        ],
      },
      {
        title: 'Responsabilidade',
        blocks: [
          {
            p: 'O editor esforça-se por assegurar a exatidão das informações divulgadas, mas não pode ser responsabilizado por erros, pela indisponibilidade ou pelos conteúdos publicados pelos utilizadores. Sendo um projeto académico, o Serviço é fornecido «tal como está», sem garantia de continuidade.',
          },
        ],
      },
      {
        title: 'Contacto',
        blocks: [
          {
            p: 'Para qualquer questão relativa a este aviso legal, pode escrever para: contact@breezy.example.',
          },
        ],
      },
    ],
  },

  cgu: {
    title: 'Termos de Uso',
    updatedAt: '8 de junho de 2026',
    intro:
      'Os presentes Termos de Uso («Termos») regem o acesso e a utilização da rede social Breezy («o Serviço»). Ao criar uma conta ou utilizar o Serviço, aceita estes Termos sem reservas.',
    sections: [
      {
        title: 'Artigo 1 — Objeto',
        blocks: [
          {
            p: 'O Breezy é uma rede social que permite publicar mensagens curtas, comentar, gostar, partilhar (repost) ou citar publicações, seguir outros utilizadores, trocar mensagens privadas cifradas e participar em grupos e comunidades.',
          },
        ],
      },
      {
        title: 'Artigo 2 — Registo e conta',
        blocks: [
          {
            list: [
              'O registo requer um endereço de e-mail válido e a escolha de um nome de utilizador (identificador único).',
              'O nome de utilizador constitui a identidade da conta; a sua alteração pode estar limitada no tempo.',
              'É responsável pela confidencialidade das suas credenciais e por toda a atividade realizada a partir da sua conta.',
              'Uma conta é estritamente pessoal. Compromete-se a fornecer informações exatas.',
            ],
          },
        ],
      },
      {
        title: 'Artigo 3 — Acesso ao Serviço',
        blocks: [
          {
            p: 'O Serviço é de acesso gratuito. O editor pode alterar, suspender ou interromper a totalidade ou parte do Serviço, nomeadamente por motivos de manutenção, sem que tal implique qualquer responsabilidade.',
          },
        ],
      },
      {
        title: 'Artigo 4 — Publicações e interações',
        blocks: [
          {
            p: 'Pode publicar mensagens curtas, responder com comentários (incluindo com emojis), gostar, partilhar ou citar publicações e fixar uma das suas publicações no seu perfil. Pode ser oferecida uma tradução automática das publicações e comentários a título indicativo.',
          },
          {
            p: 'Mantém a propriedade dos seus conteúdos e concede ao Serviço uma licença não exclusiva e gratuita para os alojar e exibir no âmbito do seu funcionamento. É o único responsável por aquilo que publica.',
          },
        ],
      },
      {
        title: 'Artigo 5 — Mensagens privadas, grupos e comunidades',
        blocks: [
          {
            list: [
              'As mensagens privadas e os grupos são cifrados de ponta a ponta: apenas os participantes podem ler o seu conteúdo. O editor não tem acesso ao conteúdo destas trocas.',
              'As comunidades são semipúblicas: o seu nome é público e consta de um diretório, e o seu conteúdo pode ser acessível a um administrador. Publica nelas tendo conhecimento desta diferença.',
              'A chave de identidade de cifragem é conservada no seu dispositivo; um novo dispositivo não consegue decifrar o histórico anterior.',
              'As regras de conduta (Artigo 6) aplicam-se a todas as trocas, incluindo as privadas.',
            ],
          },
        ],
      },
      {
        title: 'Artigo 6 — Regras de conduta',
        blocks: [
          { p: 'Ao utilizar o Serviço, compromete-se a não publicar conteúdo:' },
          {
            list: [
              'ilícito, difamatório, injurioso, de incitação ao ódio ou discriminatório;',
              'que viole a vida privada ou os direitos de terceiros;',
              'de caráter violento, pornográfico ou que incite ao ódio;',
              'que constitua assédio, spam ou fraude.',
            ],
          },
        ],
      },
      {
        title: 'Artigo 7 — Moderação e papéis',
        blocks: [
          {
            p: 'O Serviço distingue três papéis: Utilizador, Moderador e Administrador. Os moderadores e administradores podem ocultar ou remover conteúdos públicos contrários aos presentes Termos e, se for caso disso, suspender as contas em causa. A cifragem de ponta a ponta das mensagens privadas e dos grupos limita a moderação desses conteúdos ao que é tecnicamente acessível.',
          },
        ],
      },
      {
        title: 'Artigo 8 — Responsabilidade e disponibilidade',
        blocks: [
          {
            p: 'O Serviço é fornecido «tal como está», no âmbito de um projeto académico. O editor não garante a ausência de interrupções ou erros e não pode ser responsabilizado por danos resultantes da utilização ou da indisponibilidade do Serviço.',
          },
        ],
      },
      {
        title: 'Artigo 9 — Suspensão e cessação',
        blocks: [
          {
            p: 'Pode eliminar a sua conta a qualquer momento. O editor pode suspender ou cancelar uma conta em caso de incumprimento dos presentes Termos, nomeadamente das regras de conduta (Artigo 6).',
          },
        ],
      },
      {
        title: 'Artigo 10 — Dados pessoais',
        blocks: [
          {
            p: 'O tratamento dos seus dados pessoais é descrito na nossa Política de privacidade (acessível através das ligações no rodapé da página).',
          },
        ],
      },
      {
        title: 'Artigo 11 — Alteração dos Termos e lei aplicável',
        blocks: [
          {
            p: 'O editor pode alterar os presentes Termos a qualquer momento; a versão aplicável é a que estiver em vigor no momento da sua utilização. Os presentes Termos regem-se pela lei francesa; na falta de resolução amigável, os tribunais competentes serão os da sede do editor.',
          },
        ],
      },
    ],
  },

  confidentialite: {
    title: 'Política de privacidade',
    updatedAt: '8 de junho de 2026',
    intro:
      'A presente política descreve a forma como o Breezy («o Serviço») recolhe e trata os seus dados pessoais, em conformidade com o Regulamento Geral sobre a Proteção de Dados (RGPD).',
    sections: [
      {
        title: 'Responsável pelo tratamento',
        blocks: [
          {
            p: 'O responsável pelo tratamento é o editor do Serviço (equipa do projeto Breezy). Para qualquer questão: privacy@breezy.example.',
          },
        ],
      },
      {
        title: 'Dados recolhidos',
        blocks: [
          {
            list: [
              'Dados de conta: endereço de e-mail, nome de utilizador, palavra-passe (armazenada com hash, nunca em texto simples).',
              'Dados de perfil: nome apresentado, biografia, avatar, banner, sítio web, localização, data de nascimento, género e nacionalidade (opcionais).',
              'Conteúdos e interações públicos: publicações, comentários, gostos, reposts/citações, seguimentos (grafo social).',
              'Mensagens: para as mensagens privadas e os grupos, o servidor armazena apenas dados cifrados (texto cifrado + chave embrulhada por destinatário) e não tem acesso ao seu conteúdo; para as comunidades (semipúblicas), a chave de conteúdo é detida pelo servidor.',
              'Dados técnicos: tokens de autenticação (JWT e token de atualização) e registos necessários ao funcionamento.',
            ],
          },
        ],
      },
      {
        title: 'Finalidades e bases legais',
        blocks: [
          {
            list: [
              'Fornecer o Serviço e gerir a sua conta (execução dos Termos).',
              'Garantir a segurança, prevenir a fraude e moderar os conteúdos públicos (interesse legítimo).',
              'Melhorar o Serviço (interesse legítimo).',
            ],
          },
        ],
      },
      {
        title: 'Períodos de conservação',
        blocks: [
          {
            p: 'Os seus dados são conservados enquanto a sua conta estiver ativa. Em caso de eliminação da conta, são apagados ou anonimizados num prazo razoável, sem prejuízo das obrigações legais de conservação.',
          },
        ],
      },
      {
        title: 'Destinatários',
        blocks: [
          {
            p: 'Os seus dados são tratados pelos serviços internos da aplicação (autenticação, utilizadores, perfis, publicações, mensagens) e não são vendidos a terceiros. O conteúdo das mensagens privadas e dos grupos, cifrado de ponta a ponta, não é tecnicamente acessível ao editor. Os seus dados podem ser comunicados às autoridades competentes quando a lei o exigir.',
          },
        ],
      },
      {
        title: 'Cookies e armazenamento local',
        blocks: [
          {
            list: [
              'Um cookie estritamente necessário e seguro (httpOnly) mantém a sua sessão (token de atualização).',
              'Um token de acesso de curta duração pode ser conservado no armazenamento local do seu navegador para o autenticar.',
              'A sua chave privada de mensagens é armazenada localmente no seu dispositivo (IndexedDB) e nunca sai do navegador.',
              'Não é utilizado qualquer cookie publicitário ou de rastreio de terceiros.',
            ],
          },
        ],
      },
      {
        title: 'Segurança',
        blocks: [
          {
            p: 'As palavras-passe são protegidas com hash (bcrypt), a autenticação assenta em tokens assinados e revogáveis, e as mensagens privadas utilizam uma cifragem de ponta a ponta (X25519 + XChaCha20-Poly1305) do lado do cliente. Como nenhum sistema é infalível, não é possível garantir uma segurança absoluta.',
          },
        ],
      },
      {
        title: 'Os seus direitos',
        blocks: [
          { p: 'Nos termos do RGPD, dispõe dos seguintes direitos:' },
          {
            list: [
              'direito de acesso aos seus dados;',
              'direito de retificação;',
              'direito ao apagamento («direito a ser esquecido»);',
              'direito à limitação e à oposição ao tratamento;',
              'direito à portabilidade dos seus dados.',
            ],
          },
          {
            p: 'Para exercer estes direitos, escreva para privacy@breezy.example. Pode igualmente apresentar uma reclamação junto da autoridade de proteção de dados competente.',
          },
        ],
      },
      {
        title: 'Alteração da política',
        blocks: [
          {
            p: 'A presente política pode ser atualizada. A versão aplicável é a publicada no Serviço. Para o detalhe das suas obrigações, consulte os Termos de Uso.',
          },
        ],
      },
    ],
  },
}

// --- Italiano ----------------------------------------------------------------

const IT: Record<LegalSlug, LegalDoc> = {
  'mentions-legales': {
    title: 'Avviso legale',
    updatedAt: '8 giugno 2026',
    intro:
      'Breezy è un social network sviluppato in un contesto accademico (progetto FISA INFO A3). Le informazioni riportate di seguito sono fornite a titolo di trasparenza; alcune indicazioni dovranno essere completate dall’editore in caso di distribuzione pubblica.',
    sections: [
      {
        title: 'Editore del Servizio',
        blocks: [
          {
            p: 'Il sito e l’applicazione Breezy (di seguito «il Servizio») sono editi dal team del progetto Breezy (Philippe, Alexandre, Maxime, Romain) nell’ambito di un progetto studentesco. Contatto: contact@breezy.example.',
          },
        ],
      },
      {
        title: 'Direttore della pubblicazione',
        blocks: [
          { p: 'Il direttore della pubblicazione è il responsabile del progetto per il team Breezy.' },
        ],
      },
      {
        title: 'Hosting',
        blocks: [
          {
            p: 'Il Servizio è containerizzato (Docker) ed è destinato a essere ospitato presso un provider da designare in caso di distribuzione pubblica. In un contesto accademico, viene eseguito in locale.',
          },
        ],
      },
      {
        title: 'Proprietà intellettuale',
        blocks: [
          {
            p: 'Il marchio «Breezy», il logo, l’identità grafica, nonché la struttura e il contenuto editoriale del Servizio sono protetti dal diritto di proprietà intellettuale. È vietata qualsiasi riproduzione o rappresentazione, totale o parziale, senza previa autorizzazione.',
          },
          {
            p: 'I contenuti pubblicati dagli utenti (post, commenti, messaggi) restano di proprietà dei rispettivi autori, alle condizioni previste dalle Condizioni d’uso.',
          },
        ],
      },
      {
        title: 'Responsabilità',
        blocks: [
          {
            p: 'L’editore si adopera per garantire l’esattezza delle informazioni diffuse, ma non può essere ritenuto responsabile di errori, di un’eventuale indisponibilità o dei contenuti pubblicati dagli utenti. Trattandosi di un progetto accademico, il Servizio è fornito «così com’è», senza garanzia di continuità.',
          },
        ],
      },
      {
        title: 'Contatto',
        blocks: [
          {
            p: 'Per qualsiasi domanda relativa al presente avviso legale, è possibile scrivere a: contact@breezy.example.',
          },
        ],
      },
    ],
  },

  cgu: {
    title: 'Condizioni d’uso',
    updatedAt: '8 giugno 2026',
    intro:
      'Le presenti Condizioni d’uso («Condizioni») disciplinano l’accesso e l’utilizzo del social network Breezy («il Servizio»). Creando un account o utilizzando il Servizio, l’utente accetta integralmente le presenti Condizioni.',
    sections: [
      {
        title: 'Articolo 1 — Oggetto',
        blocks: [
          {
            p: 'Breezy è un social network che consente di pubblicare brevi messaggi, commentare, mettere «mi piace», condividere (repost) o citare pubblicazioni, seguire altri utenti, scambiare messaggi privati cifrati e partecipare a gruppi e community.',
          },
        ],
      },
      {
        title: 'Articolo 2 — Registrazione e account',
        blocks: [
          {
            list: [
              'La registrazione richiede un indirizzo e-mail valido e la scelta di un nome utente (identificativo univoco).',
              'Il nome utente costituisce l’identità dell’account; la sua modifica può essere limitata nel tempo.',
              'L’utente è responsabile della riservatezza delle proprie credenziali e di ogni attività svolta dal proprio account.',
              'Un account è strettamente personale. L’utente si impegna a fornire informazioni esatte.',
            ],
          },
        ],
      },
      {
        title: 'Articolo 3 — Accesso al Servizio',
        blocks: [
          {
            p: 'Il Servizio è accessibile gratuitamente. L’editore può modificare, sospendere o interrompere in tutto o in parte il Servizio, in particolare per motivi di manutenzione, senza che ciò comporti alcuna responsabilità.',
          },
        ],
      },
      {
        title: 'Articolo 4 — Pubblicazioni e interazioni',
        blocks: [
          {
            p: 'È possibile pubblicare brevi messaggi, rispondere con commenti (anche con emoji), mettere «mi piace», condividere o citare pubblicazioni e fissare una delle proprie pubblicazioni sul proprio profilo. Può essere offerta una traduzione automatica delle pubblicazioni e dei commenti a titolo puramente indicativo.',
          },
          {
            p: 'L’utente conserva la proprietà dei propri contenuti e concede al Servizio una licenza non esclusiva e gratuita per ospitarli e visualizzarli nell’ambito del suo funzionamento. L’utente è l’unico responsabile di ciò che pubblica.',
          },
        ],
      },
      {
        title: 'Articolo 5 — Messaggistica privata, gruppi e community',
        blocks: [
          {
            list: [
              'I messaggi privati e i gruppi sono cifrati end-to-end: solo i partecipanti possono leggerne il contenuto. L’editore non ha accesso al contenuto di questi scambi.',
              'Le community sono semipubbliche: il loro nome è pubblico e figura in una directory, e il loro contenuto può essere accessibile a un amministratore. L’utente vi pubblica essendo a conoscenza di questa differenza.',
              'La chiave di identità di cifratura è conservata sul dispositivo dell’utente; un nuovo dispositivo non può decifrare la cronologia precedente.',
              'Le regole di condotta (Articolo 6) si applicano a tutti gli scambi, compresi quelli privati.',
            ],
          },
        ],
      },
      {
        title: 'Articolo 6 — Regole di condotta',
        blocks: [
          { p: 'Utilizzando il Servizio, l’utente si impegna a non pubblicare contenuti:' },
          {
            list: [
              'illeciti, diffamatori, ingiuriosi, che incitino all’odio o discriminatori;',
              'lesivi della vita privata o dei diritti di terzi;',
              'di carattere violento, pornografico o che incitino all’odio;',
              'che costituiscano molestie, spam o frode.',
            ],
          },
        ],
      },
      {
        title: 'Articolo 7 — Moderazione e ruoli',
        blocks: [
          {
            p: 'Il Servizio distingue tre ruoli: Utente, Moderatore e Amministratore. I moderatori e gli amministratori possono nascondere o rimuovere contenuti pubblici contrari alle presenti Condizioni e, se del caso, sospendere gli account interessati. La cifratura end-to-end dei messaggi privati e dei gruppi limita la moderazione di tali contenuti a ciò che è tecnicamente accessibile.',
          },
        ],
      },
      {
        title: 'Articolo 8 — Responsabilità e disponibilità',
        blocks: [
          {
            p: 'Il Servizio è fornito «così com’è», nell’ambito di un progetto accademico. L’editore non garantisce l’assenza di interruzioni o errori e non può essere ritenuto responsabile dei danni derivanti dall’uso o dall’indisponibilità del Servizio.',
          },
        ],
      },
      {
        title: 'Articolo 9 — Sospensione e risoluzione',
        blocks: [
          {
            p: 'L’utente può eliminare il proprio account in qualsiasi momento. L’editore può sospendere o chiudere un account in caso di violazione delle presenti Condizioni, in particolare delle regole di condotta (Articolo 6).',
          },
        ],
      },
      {
        title: 'Articolo 10 — Dati personali',
        blocks: [
          {
            p: 'Il trattamento dei dati personali dell’utente è descritto nella nostra Informativa sulla privacy (accessibile tramite i link in fondo alla pagina).',
          },
        ],
      },
      {
        title: 'Articolo 11 — Modifica delle Condizioni e legge applicabile',
        blocks: [
          {
            p: 'L’editore può modificare le presenti Condizioni in qualsiasi momento; la versione applicabile è quella in vigore al momento dell’utilizzo. Le presenti Condizioni sono soggette al diritto francese; in mancanza di una risoluzione amichevole, i tribunali competenti saranno quelli della sede dell’editore.',
          },
        ],
      },
    ],
  },

  confidentialite: {
    title: 'Informativa sulla privacy',
    updatedAt: '8 giugno 2026',
    intro:
      'La presente informativa descrive il modo in cui Breezy («il Servizio») raccoglie e tratta i dati personali dell’utente, in conformità al Regolamento generale sulla protezione dei dati (GDPR).',
    sections: [
      {
        title: 'Titolare del trattamento',
        blocks: [
          {
            p: 'Il titolare del trattamento è l’editore del Servizio (team del progetto Breezy). Per qualsiasi domanda: privacy@breezy.example.',
          },
        ],
      },
      {
        title: 'Dati raccolti',
        blocks: [
          {
            list: [
              'Dati dell’account: indirizzo e-mail, nome utente, password (memorizzata sotto forma di hash, mai in chiaro).',
              'Dati del profilo: nome visualizzato, biografia, avatar, banner, sito web, posizione, data di nascita, genere e nazionalità (facoltativi).',
              'Contenuti e interazioni pubblici: pubblicazioni, commenti, «mi piace», repost/citazioni, follow (grafo sociale).',
              'Messaggistica: per i messaggi privati e i gruppi, il server memorizza solo dati cifrati (testo cifrato + chiave incapsulata per destinatario) e non ha accesso al loro contenuto; per le community (semipubbliche), la chiave di contenuto è detenuta dal server.',
              'Dati tecnici: token di autenticazione (JWT e token di aggiornamento) e log necessari al funzionamento.',
            ],
          },
        ],
      },
      {
        title: 'Finalità e basi giuridiche',
        blocks: [
          {
            list: [
              'Fornire il Servizio e gestire l’account (esecuzione delle Condizioni).',
              'Garantire la sicurezza, prevenire le frodi e moderare i contenuti pubblici (legittimo interesse).',
              'Migliorare il Servizio (legittimo interesse).',
            ],
          },
        ],
      },
      {
        title: 'Periodi di conservazione',
        blocks: [
          {
            p: 'I dati sono conservati per tutto il tempo in cui l’account è attivo. In caso di eliminazione dell’account, vengono cancellati o anonimizzati entro un termine ragionevole, fatti salvi gli obblighi legali di conservazione.',
          },
        ],
      },
      {
        title: 'Destinatari',
        blocks: [
          {
            p: 'I dati sono trattati dai servizi interni dell’applicazione (autenticazione, utenti, profili, pubblicazioni, messaggistica) e non sono venduti a terzi. Il contenuto dei messaggi privati e dei gruppi, cifrato end-to-end, non è tecnicamente accessibile all’editore. I dati possono essere comunicati alle autorità competenti quando la legge lo richiede.',
          },
        ],
      },
      {
        title: 'Cookie e archiviazione locale',
        blocks: [
          {
            list: [
              'Un cookie strettamente necessario e sicuro (httpOnly) mantiene la sessione (token di aggiornamento).',
              'Un token di accesso di breve durata può essere conservato nell’archiviazione locale del browser per autenticare l’utente.',
              'La chiave privata di messaggistica è archiviata localmente sul dispositivo dell’utente (IndexedDB) e non lascia mai il browser.',
              'Non viene utilizzato alcun cookie pubblicitario o di tracciamento di terze parti.',
            ],
          },
        ],
      },
      {
        title: 'Sicurezza',
        blocks: [
          {
            p: 'Le password sono protette tramite hash (bcrypt), l’autenticazione si basa su token firmati e revocabili e la messaggistica privata utilizza una cifratura end-to-end (X25519 + XChaCha20-Poly1305) lato client. Poiché nessun sistema è infallibile, non è possibile garantire una sicurezza assoluta.',
          },
        ],
      },
      {
        title: 'I tuoi diritti',
        blocks: [
          { p: 'Ai sensi del GDPR, l’utente dispone dei seguenti diritti:' },
          {
            list: [
              'diritto di accesso ai propri dati;',
              'diritto di rettifica;',
              'diritto alla cancellazione («diritto all’oblio»);',
              'diritto di limitazione e di opposizione al trattamento;',
              'diritto alla portabilità dei propri dati.',
            ],
          },
          {
            p: 'Per esercitare questi diritti, scrivere a privacy@breezy.example. È inoltre possibile presentare un reclamo all’autorità di controllo competente.',
          },
        ],
      },
      {
        title: 'Modifica dell’informativa',
        blocks: [
          {
            p: 'La presente informativa può essere aggiornata. La versione applicabile è quella pubblicata sul Servizio. Per il dettaglio degli obblighi, consultare le Condizioni d’uso.',
          },
        ],
      },
    ],
  },
}

// --- Русский -----------------------------------------------------------------

const RU: Record<LegalSlug, LegalDoc> = {
  'mentions-legales': {
    title: 'Официальное уведомление',
    updatedAt: '8 июня 2026 г.',
    intro:
      'Breezy — это социальная сеть, разработанная в учебных целях (проект FISA INFO A3). Приведённые ниже сведения предоставляются в целях прозрачности; некоторые пункты должны быть дополнены издателем при публичном развёртывании.',
    sections: [
      {
        title: 'Издатель Сервиса',
        blocks: [
          {
            p: 'Сайт и приложение Breezy (далее — «Сервис») издаются командой проекта Breezy (Philippe, Alexandre, Maxime, Romain) в рамках студенческого проекта. Контакт: contact@breezy.example.',
          },
        ],
      },
      {
        title: 'Ответственный за публикацию',
        blocks: [
          { p: 'Ответственным за публикацию является руководитель проекта от команды Breezy.' },
        ],
      },
      {
        title: 'Хостинг',
        blocks: [
          {
            p: 'Сервис контейнеризирован (Docker) и предназначен для размещения у хостинг-провайдера, который будет определён при публичном развёртывании. В учебной среде он запускается локально.',
          },
        ],
      },
      {
        title: 'Интеллектуальная собственность',
        blocks: [
          {
            p: 'Бренд «Breezy», логотип, фирменный стиль, а также структура и редакционное содержание Сервиса защищены правом интеллектуальной собственности. Любое полное или частичное воспроизведение или представление без предварительного разрешения запрещено.',
          },
          {
            p: 'Контент, публикуемый пользователями (публикации, комментарии, сообщения), остаётся собственностью их авторов на условиях, предусмотренных Условиями использования.',
          },
        ],
      },
      {
        title: 'Ответственность',
        blocks: [
          {
            p: 'Издатель стремится обеспечить точность распространяемой информации, но не несёт ответственности за ошибки, недоступность или контент, публикуемый пользователями. Поскольку Сервис является учебным проектом, он предоставляется «как есть», без гарантии непрерывности работы.',
          },
        ],
      },
      {
        title: 'Контакт',
        blocks: [
          {
            p: 'По любым вопросам, касающимся настоящего уведомления, вы можете написать по адресу: contact@breezy.example.',
          },
        ],
      },
    ],
  },

  cgu: {
    title: 'Условия использования',
    updatedAt: '8 июня 2026 г.',
    intro:
      'Настоящие Условия использования («Условия») регулируют доступ к социальной сети Breezy («Сервис») и её использование. Создавая учётную запись или используя Сервис, вы безоговорочно принимаете настоящие Условия.',
    sections: [
      {
        title: 'Статья 1 — Предмет',
        blocks: [
          {
            p: 'Breezy — это социальная сеть, позволяющая публиковать короткие сообщения, комментировать, ставить отметки «нравится», репостить или цитировать публикации, подписываться на других пользователей, обмениваться зашифрованными личными сообщениями и участвовать в группах и сообществах.',
          },
        ],
      },
      {
        title: 'Статья 2 — Регистрация и учётная запись',
        blocks: [
          {
            list: [
              'Для регистрации требуется действующий адрес электронной почты и выбор имени пользователя (уникального идентификатора).',
              'Имя пользователя является идентификатором учётной записи; его изменение может быть ограничено по времени.',
              'Вы несёте ответственность за конфиденциальность своих учётных данных и за любые действия, совершённые с вашей учётной записи.',
              'Учётная запись является строго персональной. Вы обязуетесь предоставлять точные сведения.',
            ],
          },
        ],
      },
      {
        title: 'Статья 3 — Доступ к Сервису',
        blocks: [
          {
            p: 'Доступ к Сервису бесплатный. Издатель может изменять, приостанавливать или прекращать работу Сервиса полностью или частично, в частности по причинам технического обслуживания, без наступления ответственности.',
          },
        ],
      },
      {
        title: 'Статья 4 — Публикации и взаимодействия',
        blocks: [
          {
            p: 'Вы можете публиковать короткие сообщения, отвечать на них комментариями (в том числе с эмодзи), ставить отметки «нравится», репостить или цитировать публикации и закреплять одну из своих публикаций в профиле. Автоматический перевод публикаций и комментариев может предлагаться исключительно в справочных целях.',
          },
          {
            p: 'Вы сохраняете право собственности на свой контент и предоставляете Сервису неисключительную безвозмездную лицензию на его размещение и отображение в рамках работы Сервиса. Вы несёте единоличную ответственность за то, что публикуете.',
          },
        ],
      },
      {
        title: 'Статья 5 — Личные сообщения, группы и сообщества',
        blocks: [
          {
            list: [
              'Личные сообщения и группы зашифрованы сквозным шифрованием: только участники могут читать их содержимое. Издатель не имеет доступа к содержимому этих переписок.',
              'Сообщества являются полупубличными: их название является публичным и указывается в каталоге, а их содержимое может быть доступно администратору. Вы публикуете в них, осознавая это различие.',
              'Ключ идентификации шифрования хранится на вашем устройстве; новое устройство не может расшифровать предыдущую историю.',
              'Правила поведения (Статья 6) применяются ко всем переписками, включая личные.',
            ],
          },
        ],
      },
      {
        title: 'Статья 6 — Правила поведения',
        blocks: [
          { p: 'Используя Сервис, вы обязуетесь не публиковать контент:' },
          {
            list: [
              'незаконный, клеветнический, оскорбительный, разжигающий ненависть или дискриминационный;',
              'нарушающий частную жизнь или права третьих лиц;',
              'насильственного, порнографического характера или подстрекающий к ненависти;',
              'представляющий собой домогательства, спам или мошенничество.',
            ],
          },
        ],
      },
      {
        title: 'Статья 7 — Модерация и роли',
        blocks: [
          {
            p: 'Сервис различает три роли: Пользователь, Модератор и Администратор. Модераторы и администраторы могут скрывать или удалять публичный контент, нарушающий настоящие Условия, и при необходимости приостанавливать соответствующие учётные записи. Сквозное шифрование личных сообщений и групп ограничивает модерацию такого контента тем, что технически доступно.',
          },
        ],
      },
      {
        title: 'Статья 8 — Ответственность и доступность',
        blocks: [
          {
            p: 'Сервис предоставляется «как есть» в рамках учебного проекта. Издатель не гарантирует отсутствие перерывов или ошибок и не несёт ответственности за ущерб, возникший в результате использования или недоступности Сервиса.',
          },
        ],
      },
      {
        title: 'Статья 9 — Приостановление и расторжение',
        blocks: [
          {
            p: 'Вы можете удалить свою учётную запись в любое время. Издатель может приостановить или закрыть учётную запись в случае нарушения настоящих Условий, в частности правил поведения (Статья 6).',
          },
        ],
      },
      {
        title: 'Статья 10 — Персональные данные',
        blocks: [
          {
            p: 'Обработка ваших персональных данных описана в нашей Политике конфиденциальности (доступна по ссылкам внизу страницы).',
          },
        ],
      },
      {
        title: 'Статья 11 — Изменение Условий и применимое право',
        blocks: [
          {
            p: 'Издатель может изменять настоящие Условия в любое время; применимой является версия, действующая на момент использования вами Сервиса. Настоящие Условия регулируются французским правом; при отсутствии мирного урегулирования компетентными являются суды по месту нахождения издателя.',
          },
        ],
      },
    ],
  },

  confidentialite: {
    title: 'Политика конфиденциальности',
    updatedAt: '8 июня 2026 г.',
    intro:
      'Настоящая политика описывает, как Breezy («Сервис») собирает и обрабатывает ваши персональные данные в соответствии с Общим регламентом по защите данных (GDPR).',
    sections: [
      {
        title: 'Ответственный за обработку',
        blocks: [
          {
            p: 'Ответственным за обработку является издатель Сервиса (команда проекта Breezy). По любым вопросам: privacy@breezy.example.',
          },
        ],
      },
      {
        title: 'Собираемые данные',
        blocks: [
          {
            list: [
              'Данные учётной записи: адрес электронной почты, имя пользователя, пароль (хранится в виде хеша, никогда в открытом виде).',
              'Данные профиля: отображаемое имя, биография, аватар, баннер, веб-сайт, местоположение, дата рождения, пол и гражданство (необязательно).',
              'Публичный контент и взаимодействия: публикации, комментарии, отметки «нравится», репосты/цитаты, подписки (социальный граф).',
              'Обмен сообщениями: для личных сообщений и групп сервер хранит только зашифрованные данные (зашифрованный текст + ключ, упакованный для каждого получателя) и не имеет доступа к их содержимому; для сообществ (полупубличных) ключ содержимого хранится на сервере.',
              'Технические данные: токены аутентификации (JWT и токен обновления) и журналы, необходимые для работы.',
            ],
          },
        ],
      },
      {
        title: 'Цели и правовые основания',
        blocks: [
          {
            list: [
              'Предоставление Сервиса и управление вашей учётной записью (исполнение Условий).',
              'Обеспечение безопасности, предотвращение мошенничества и модерация публичного контента (законный интерес).',
              'Улучшение Сервиса (законный интерес).',
            ],
          },
        ],
      },
      {
        title: 'Сроки хранения',
        blocks: [
          {
            p: 'Ваши данные хранятся, пока ваша учётная запись активна. В случае удаления учётной записи они удаляются или анонимизируются в разумный срок с учётом установленных законом обязательств по хранению.',
          },
        ],
      },
      {
        title: 'Получатели',
        blocks: [
          {
            p: 'Ваши данные обрабатываются внутренними сервисами приложения (аутентификация, пользователи, профили, публикации, обмен сообщениями) и не продаются третьим лицам. Содержимое личных сообщений и групп, защищённое сквозным шифрованием, технически недоступно издателю. Ваши данные могут быть переданы компетентным органам, когда этого требует закон.',
          },
        ],
      },
      {
        title: 'Файлы cookie и локальное хранилище',
        blocks: [
          {
            list: [
              'Строго необходимый защищённый файл cookie (httpOnly) поддерживает вашу сессию (токен обновления).',
              'Краткосрочный токен доступа может храниться в локальном хранилище вашего браузера для вашей аутентификации.',
              'Ваш закрытый ключ для обмена сообщениями хранится локально на вашем устройстве (IndexedDB) и никогда не покидает браузер.',
              'Никакие рекламные или сторонние отслеживающие файлы cookie не используются.',
            ],
          },
        ],
      },
      {
        title: 'Безопасность',
        blocks: [
          {
            p: 'Пароли хешируются (bcrypt), аутентификация основана на подписанных и отзываемых токенах, а личный обмен сообщениями использует сквозное шифрование (X25519 + XChaCha20-Poly1305) на стороне клиента. Поскольку ни одна система не является безупречной, абсолютная безопасность не может быть гарантирована.',
          },
        ],
      },
      {
        title: 'Ваши права',
        blocks: [
          { p: 'В соответствии с GDPR вы обладаете следующими правами:' },
          {
            list: [
              'право на доступ к своим данным;',
              'право на исправление;',
              'право на удаление («право быть забытым»);',
              'право на ограничение обработки и возражение против неё;',
              'право на переносимость своих данных.',
            ],
          },
          {
            p: 'Чтобы воспользоваться этими правами, напишите по адресу privacy@breezy.example. Вы также можете подать жалобу в компетентный орган по защите данных.',
          },
        ],
      },
      {
        title: 'Изменение политики',
        blocks: [
          {
            p: 'Настоящая политика может обновляться. Применимой является версия, опубликованная в Сервисе. Подробнее о ваших обязанностях см. Условия использования.',
          },
        ],
      },
    ],
  },
}

// --- 中文 --------------------------------------------------------------------

const ZH: Record<LegalSlug, LegalDoc> = {
  'mentions-legales': {
    title: '法律声明',
    updatedAt: '2026 年 6 月 8 日',
    intro:
      'Breezy 是在教学背景下（FISA INFO A3 项目）开发的社交网络。以下信息出于透明的目的提供；部分内容须由发布者在公开部署时补充完善。',
    sections: [
      {
        title: '服务的发布者',
        blocks: [
          {
            p: 'Breezy 网站及应用程序（以下简称“本服务”）由 Breezy 项目团队（Philippe、Alexandre、Maxime、Romain）作为学生项目发布。联系方式：contact@breezy.example。',
          },
        ],
      },
      {
        title: '出版负责人',
        blocks: [{ p: '出版负责人为 Breezy 团队的项目负责人。' }],
      },
      {
        title: '托管',
        blocks: [
          {
            p: '本服务采用容器化（Docker），在公开部署时拟托管于待指定的托管服务商。在教学环境中，本服务在本地运行。',
          },
        ],
      },
      {
        title: '知识产权',
        blocks: [
          {
            p: '“Breezy”商标、徽标、视觉形象以及本服务的结构和编辑内容均受知识产权法保护。未经事先授权，禁止全部或部分复制或展示。',
          },
          {
            p: '用户发布的内容（帖子、评论、消息）仍归其作者所有，受《使用条款》规定的条件约束。',
          },
        ],
      },
      {
        title: '责任',
        blocks: [
          {
            p: '发布者努力确保所发布信息的准确性，但对错误、不可用或用户发布的内容概不负责。由于本服务为教学项目，故按“现状”提供，不保证持续可用。',
          },
        ],
      },
      {
        title: '联系方式',
        blocks: [
          {
            p: '如对本法律声明有任何疑问，您可发送邮件至：contact@breezy.example。',
          },
        ],
      },
    ],
  },

  cgu: {
    title: '使用条款',
    updatedAt: '2026 年 6 月 8 日',
    intro:
      '本使用条款（“条款”）规范对 Breezy 社交网络（“本服务”）的访问和使用。创建账户或使用本服务即表示您无保留地接受本条款。',
    sections: [
      {
        title: '第 1 条 — 目的',
        blocks: [
          {
            p: 'Breezy 是一个社交网络，您可以发布短消息、发表评论、点赞、转发或引用帖子、关注其他用户、交换加密的私信，并参与群组和社区。',
          },
        ],
      },
      {
        title: '第 2 条 — 注册与账户',
        blocks: [
          {
            list: [
              '注册需要有效的电子邮件地址并选择一个用户名（唯一标识符）。',
              '用户名构成账户的身份标识；其修改可能受到时间限制。',
              '您应对凭据的保密性以及通过您账户进行的一切活动负责。',
              '账户严格限于个人使用。您承诺提供准确的信息。',
            ],
          },
        ],
      },
      {
        title: '第 3 条 — 访问本服务',
        blocks: [
          {
            p: '本服务免费提供。发布者可对本服务的全部或部分进行变更、暂停或中止，尤其出于维护原因，且不因此承担任何责任。',
          },
        ],
      },
      {
        title: '第 4 条 — 发布与互动',
        blocks: [
          {
            p: '您可以发布短消息、以评论方式回复（包括使用表情符号）、点赞、转发或引用帖子，并将您的某条帖子置顶到个人资料。帖子和评论可能提供仅供参考的自动翻译。',
          },
          {
            p: '您保留对自身内容的所有权，并授予本服务一项非排他、免费的许可，以在其运营过程中托管和展示这些内容。您对所发布的内容承担全部责任。',
          },
        ],
      },
      {
        title: '第 5 条 — 私信、群组与社区',
        blocks: [
          {
            list: [
              '私信和群组采用端到端加密：只有参与者才能阅读其内容。发布者无法访问这些交流的内容。',
              '社区为半公开：其名称是公开的并列入目录，其内容可能可供管理员访问。您在了解这一差异的情况下于其中发布内容。',
              '加密身份密钥保存在您的设备上；新设备无法解密此前的历史记录。',
              '行为规则（第 6 条）适用于所有交流，包括私下交流。',
            ],
          },
        ],
      },
      {
        title: '第 6 条 — 行为规则',
        blocks: [
          { p: '使用本服务时，您承诺不发布以下内容：' },
          {
            list: [
              '违法、诽谤、侮辱、仇恨或歧视性的内容；',
              '侵犯隐私或第三方权利的内容；',
              '具有暴力、色情性质或煽动仇恨的内容；',
              '构成骚扰、垃圾信息或欺诈的内容。',
            ],
          },
        ],
      },
      {
        title: '第 7 条 — 审核与角色',
        blocks: [
          {
            p: '本服务区分三种角色：用户、版主和管理员。版主和管理员可以隐藏或删除违反本条款的公开内容，并在必要时暂停相关账户。私信和群组的端到端加密将此类内容的审核限制在技术上可访问的范围内。',
          },
        ],
      },
      {
        title: '第 8 条 — 责任与可用性',
        blocks: [
          {
            p: '本服务在教学项目框架内按“现状”提供。发布者不保证不会出现中断或错误，且对因使用或无法使用本服务而造成的损害概不负责。',
          },
        ],
      },
      {
        title: '第 9 条 — 暂停与终止',
        blocks: [
          {
            p: '您可随时删除您的账户。如违反本条款，尤其是行为规则（第 6 条），发布者可暂停或终止账户。',
          },
        ],
      },
      {
        title: '第 10 条 — 个人数据',
        blocks: [
          {
            p: '我们对您个人数据的处理在《隐私政策》中说明（可通过页面底部的链接访问）。',
          },
        ],
      },
      {
        title: '第 11 条 — 条款的修改与适用法律',
        blocks: [
          {
            p: '发布者可随时修改本条款；适用的版本为您使用时有效的版本。本条款适用法国法律；如未能友好解决，管辖法院为发布者注册地所在法院。',
          },
        ],
      },
    ],
  },

  confidentialite: {
    title: '隐私政策',
    updatedAt: '2026 年 6 月 8 日',
    intro:
      '本政策描述 Breezy（“本服务”）依据《通用数据保护条例》（GDPR）收集和处理您个人数据的方式。',
    sections: [
      {
        title: '数据控制者',
        blocks: [
          {
            p: '数据控制者为本服务的发布者（Breezy 项目团队）。如有任何疑问：privacy@breezy.example。',
          },
        ],
      },
      {
        title: '收集的数据',
        blocks: [
          {
            list: [
              '账户数据：电子邮件地址、用户名、密码（以哈希形式存储，绝不以明文保存）。',
              '资料数据：显示名称、个人简介、头像、横幅、网站、所在地、出生日期、性别和国籍（可选）。',
              '公开内容与互动：帖子、评论、点赞、转发/引用、关注（社交图谱）。',
              '消息：对于私信和群组，服务器仅存储加密数据（密文 + 按收件人封装的密钥），无法访问其内容；对于社区（半公开），内容密钥由服务器持有。',
              '技术数据：身份验证令牌（JWT 和刷新令牌）以及运行所需的日志。',
            ],
          },
        ],
      },
      {
        title: '目的与法律依据',
        blocks: [
          {
            list: [
              '提供本服务并管理您的账户（履行条款）。',
              '保障安全、防范欺诈并审核公开内容（合法利益）。',
              '改进本服务（合法利益）。',
            ],
          },
        ],
      },
      {
        title: '保留期限',
        blocks: [
          {
            p: '只要您的账户处于活动状态，您的数据即予以保留。账户删除后，将在合理期限内删除或匿名化，但须遵守法律规定的保留义务。',
          },
        ],
      },
      {
        title: '接收方',
        blocks: [
          {
            p: '您的数据由应用程序的内部服务（身份验证、用户、资料、帖子、消息）处理，不会出售给第三方。私信和群组的内容经端到端加密，发布者在技术上无法访问。当法律要求时，您的数据可能会提供给主管机关。',
          },
        ],
      },
      {
        title: 'Cookie 与本地存储',
        blocks: [
          {
            list: [
              '一个严格必要且安全的（httpOnly）Cookie 用于维持您的会话（刷新令牌）。',
              '短时效的访问令牌可能保存在您浏览器的本地存储中，以对您进行身份验证。',
              '您的消息私钥本地存储在您的设备上（IndexedDB），绝不会离开浏览器。',
              '不使用任何广告或第三方跟踪 Cookie。',
            ],
          },
        ],
      },
      {
        title: '安全',
        blocks: [
          {
            p: '密码经过哈希处理（bcrypt），身份验证依赖于已签名且可撤销的令牌，私信在客户端使用端到端加密（X25519 + XChaCha20-Poly1305）。由于没有任何系统是万无一失的，故无法保证绝对的安全。',
          },
        ],
      },
      {
        title: '您的权利',
        blocks: [
          { p: '根据 GDPR，您享有以下权利：' },
          {
            list: [
              '访问您数据的权利；',
              '更正权；',
              '删除权（“被遗忘权”）；',
              '限制处理权和反对处理权；',
              '数据可携权。',
            ],
          },
          {
            p: '如需行使这些权利，请发送邮件至 privacy@breezy.example。您也可以向主管的数据保护机关提出投诉。',
          },
        ],
      },
      {
        title: '政策的修改',
        blocks: [
          {
            p: '本政策可能会更新。适用的版本为发布在本服务上的版本。有关您义务的详细信息，请参阅《使用条款》。',
          },
        ],
      },
    ],
  },
}

// --- 日本語 ------------------------------------------------------------------

const JA: Record<LegalSlug, LegalDoc> = {
  'mentions-legales': {
    title: '法的通知',
    updatedAt: '2026年6月8日',
    intro:
      'Breezy は教育目的（FISA INFO A3 プロジェクト）で開発されたソーシャルネットワークです。以下の情報は透明性のために提供されるものであり、一部の記載は公開時に発行者が補完する必要があります。',
    sections: [
      {
        title: 'サービスの発行者',
        blocks: [
          {
            p: 'Breezy のウェブサイトおよびアプリケーション（以下「本サービス」）は、学生プロジェクトとして Breezy プロジェクトチーム（Philippe、Alexandre、Maxime、Romain）が発行しています。連絡先：contact@breezy.example。',
          },
        ],
      },
      {
        title: '発行責任者',
        blocks: [{ p: '発行責任者は Breezy チームのプロジェクト責任者です。' }],
      },
      {
        title: 'ホスティング',
        blocks: [
          {
            p: '本サービスはコンテナ化（Docker）されており、公開時には別途指定するホスティング事業者でホストされる予定です。教育環境ではローカルで実行されます。',
          },
        ],
      },
      {
        title: '知的財産',
        blocks: [
          {
            p: '「Breezy」ブランド、ロゴ、ビジュアルアイデンティティ、ならびに本サービスの構成および編集コンテンツは、知的財産権により保護されています。事前の許可なく、全部または一部を複製または表示することは禁止されています。',
          },
          {
            p: 'ユーザーが投稿したコンテンツ（投稿、コメント、メッセージ）は、利用規約に定める条件のもとで、その作成者の財産であり続けます。',
          },
        ],
      },
      {
        title: '責任',
        blocks: [
          {
            p: '発行者は提供される情報の正確性の確保に努めますが、誤り、利用不能、またはユーザーが投稿したコンテンツについて責任を負いません。本サービスは教育プロジェクトであるため、継続性の保証なく「現状有姿」で提供されます。',
          },
        ],
      },
      {
        title: 'お問い合わせ',
        blocks: [
          {
            p: '本法的通知に関するお問い合わせは、次の宛先までお願いします：contact@breezy.example。',
          },
        ],
      },
    ],
  },

  cgu: {
    title: '利用規約',
    updatedAt: '2026年6月8日',
    intro:
      '本利用規約（「本規約」）は、ソーシャルネットワーク Breezy（「本サービス」）へのアクセスおよび利用を規定するものです。アカウントを作成し、または本サービスを利用することにより、あなたは本規約を無条件で承諾するものとします。',
    sections: [
      {
        title: '第1条 — 目的',
        blocks: [
          {
            p: 'Breezy は、短い投稿の公開、コメント、いいね、リポストまたは引用、他のユーザーのフォロー、暗号化されたダイレクトメッセージの交換、グループおよびコミュニティへの参加ができるソーシャルネットワークです。',
          },
        ],
      },
      {
        title: '第2条 — 登録とアカウント',
        blocks: [
          {
            list: [
              '登録には有効なメールアドレスと、ユーザー名（一意の識別子）の選択が必要です。',
              'ユーザー名はアカウントの識別情報を構成します。その変更は期間によって制限される場合があります。',
              'あなたは認証情報の機密保持、およびご自身のアカウントから行われるすべての活動について責任を負います。',
              'アカウントは厳密に個人のものです。あなたは正確な情報を提供することに同意します。',
            ],
          },
        ],
      },
      {
        title: '第3条 — 本サービスへのアクセス',
        blocks: [
          {
            p: '本サービスは無料でアクセスできます。発行者は、特にメンテナンスのために、本サービスの全部または一部を変更、停止または終了することができ、それについて責任を負いません。',
          },
        ],
      },
      {
        title: '第4条 — 投稿とインタラクション',
        blocks: [
          {
            p: '短い投稿の公開、コメント（絵文字を含む）による返信、いいね、リポストまたは引用、ならびにご自身の投稿の一つをプロフィールに固定することができます。投稿およびコメントの自動翻訳が参考として提供される場合があります。',
          },
          {
            p: 'あなたはご自身のコンテンツの所有権を保持し、本サービスがその運営の範囲内でこれをホストおよび表示するために、非独占的かつ無償のライセンスを本サービスに付与します。あなたは投稿する内容について単独で責任を負います。',
          },
        ],
      },
      {
        title: '第5条 — ダイレクトメッセージ、グループおよびコミュニティ',
        blocks: [
          {
            list: [
              'ダイレクトメッセージおよびグループはエンドツーエンドで暗号化されています。参加者のみがその内容を読むことができます。発行者はこれらのやり取りの内容にアクセスできません。',
              'コミュニティは準公開です。その名称は公開されディレクトリに掲載され、その内容は管理者がアクセスできる場合があります。あなたはこの違いを認識した上で投稿します。',
              '暗号化の識別鍵はあなたのデバイスに保存されます。新しいデバイスでは以前の履歴を復号できません。',
              '行動規則（第6条）は、プライベートなものを含むすべてのやり取りに適用されます。',
            ],
          },
        ],
      },
      {
        title: '第6条 — 行動規則',
        blocks: [
          { p: '本サービスを利用するにあたり、あなたは次のコンテンツを投稿しないことに同意します。' },
          {
            list: [
              '違法、名誉毀損、侮辱的、憎悪を煽る、または差別的なコンテンツ；',
              'プライバシーまたは第三者の権利を侵害するコンテンツ；',
              '暴力的、ポルノ的、または憎悪を扇動するコンテンツ；',
              '嫌がらせ、スパム、または詐欺を構成するコンテンツ。',
            ],
          },
        ],
      },
      {
        title: '第7条 — モデレーションと役割',
        blocks: [
          {
            p: '本サービスには、ユーザー、モデレーター、管理者の3つの役割があります。モデレーターおよび管理者は、本規約に反する公開コンテンツを非表示または削除し、必要に応じて該当アカウントを停止することができます。ダイレクトメッセージおよびグループのエンドツーエンド暗号化により、これらのコンテンツのモデレーションは技術的にアクセス可能な範囲に限られます。',
          },
        ],
      },
      {
        title: '第8条 — 責任と可用性',
        blocks: [
          {
            p: '本サービスは教育プロジェクトの一環として「現状有姿」で提供されます。発行者は中断や誤りがないことを保証せず、本サービスの利用または利用不能から生じる損害について責任を負いません。',
          },
        ],
      },
      {
        title: '第9条 — 停止と解除',
        blocks: [
          {
            p: 'あなたはいつでもアカウントを削除できます。発行者は、本規約、特に行動規則（第6条）への違反があった場合、アカウントを停止または解除することができます。',
          },
        ],
      },
      {
        title: '第10条 — 個人データ',
        blocks: [
          {
            p: 'あなたの個人データの取扱いは、当社のプライバシーポリシー（ページ下部のリンクからアクセス可能）に記載されています。',
          },
        ],
      },
      {
        title: '第11条 — 規約の変更および準拠法',
        blocks: [
          {
            p: '発行者はいつでも本規約を変更できます。適用される版は、あなたが利用する時点で有効な版です。本規約はフランス法に準拠します。友好的に解決できない場合、管轄裁判所は発行者の所在地を管轄する裁判所とします。',
          },
        ],
      },
    ],
  },

  confidentialite: {
    title: 'プライバシーポリシー',
    updatedAt: '2026年6月8日',
    intro:
      '本ポリシーは、Breezy（「本サービス」）が一般データ保護規則（GDPR）に従ってあなたの個人データを収集および取り扱う方法について説明します。',
    sections: [
      {
        title: '管理者',
        blocks: [
          {
            p: '管理者は本サービスの発行者（Breezy プロジェクトチーム）です。お問い合わせ：privacy@breezy.example。',
          },
        ],
      },
      {
        title: '収集するデータ',
        blocks: [
          {
            list: [
              'アカウントデータ：メールアドレス、ユーザー名、パスワード（ハッシュ化して保存し、平文では保存しません）。',
              'プロフィールデータ：表示名、自己紹介、アバター、バナー、ウェブサイト、所在地、生年月日、性別、国籍（任意）。',
              '公開コンテンツとインタラクション：投稿、コメント、いいね、リポスト／引用、フォロー（ソーシャルグラフ）。',
              'メッセージ：ダイレクトメッセージおよびグループについて、サーバーは暗号化データ（暗号文＋受信者ごとにラップされた鍵）のみを保存し、その内容にはアクセスできません。コミュニティ（準公開）については、コンテンツ鍵はサーバーが保持します。',
              '技術データ：認証トークン（JWT およびリフレッシュトークン）と運用に必要なログ。',
            ],
          },
        ],
      },
      {
        title: '目的と法的根拠',
        blocks: [
          {
            list: [
              '本サービスの提供およびアカウントの管理（規約の履行）。',
              'セキュリティの確保、不正の防止、公開コンテンツのモデレーション（正当な利益）。',
              '本サービスの改善（正当な利益）。',
            ],
          },
        ],
      },
      {
        title: '保存期間',
        blocks: [
          {
            p: 'あなたのデータはアカウントが有効である限り保存されます。アカウントが削除された場合、法的な保存義務に従うことを条件として、合理的な期間内に削除または匿名化されます。',
          },
        ],
      },
      {
        title: '取得者',
        blocks: [
          {
            p: 'あなたのデータはアプリケーションの内部サービス（認証、ユーザー、プロフィール、投稿、メッセージ）により処理され、第三者に販売されることはありません。ダイレクトメッセージおよびグループの内容はエンドツーエンドで暗号化されており、発行者は技術的にアクセスできません。あなたのデータは、法律が要求する場合、所轄当局に提供されることがあります。',
          },
        ],
      },
      {
        title: 'Cookie とローカルストレージ',
        blocks: [
          {
            list: [
              '厳密に必要かつ安全な（httpOnly）Cookie がセッション（リフレッシュトークン）を維持します。',
              '短期間の有効なアクセストークンが、あなたを認証するためにブラウザのローカルストレージに保存される場合があります。',
              'あなたのメッセージ用秘密鍵はデバイス上にローカルで保存され（IndexedDB）、ブラウザを離れることはありません。',
              '広告用または第三者のトラッキング Cookie は一切使用しません。',
            ],
          },
        ],
      },
      {
        title: 'セキュリティ',
        blocks: [
          {
            p: 'パスワードはハッシュ化され（bcrypt）、認証は署名済みかつ失効可能なトークンに基づき、ダイレクトメッセージはクライアント側でエンドツーエンド暗号化（X25519 + XChaCha20-Poly1305）を使用します。いかなるシステムも完全ではないため、絶対的な安全性を保証することはできません。',
          },
        ],
      },
      {
        title: 'あなたの権利',
        blocks: [
          { p: 'GDPR に基づき、あなたは次の権利を有します。' },
          {
            list: [
              'ご自身のデータへのアクセス権；',
              '訂正の権利；',
              '消去の権利（「忘れられる権利」）；',
              '処理の制限および処理への異議の権利；',
              'データポータビリティの権利。',
            ],
          },
          {
            p: 'これらの権利を行使するには、privacy@breezy.example までご連絡ください。所轄のデータ保護当局に苦情を申し立てることもできます。',
          },
        ],
      },
      {
        title: 'ポリシーの変更',
        blocks: [
          {
            p: '本ポリシーは更新されることがあります。適用される版は本サービス上で公開されている版です。あなたの義務の詳細については、利用規約をご参照ください。',
          },
        ],
      },
    ],
  },
}

// --- 한국어 ------------------------------------------------------------------

const KO: Record<LegalSlug, LegalDoc> = {
  'mentions-legales': {
    title: '법적 고지',
    updatedAt: '2026년 6월 8일',
    intro:
      'Breezy는 교육 목적(FISA INFO A3 프로젝트)으로 개발된 소셜 네트워크입니다. 아래 정보는 투명성을 위해 제공되며, 일부 항목은 공개 배포 시 발행자가 보완해야 합니다.',
    sections: [
      {
        title: '서비스 발행자',
        blocks: [
          {
            p: 'Breezy 웹사이트 및 애플리케이션(이하 “본 서비스”)은 학생 프로젝트의 일환으로 Breezy 프로젝트 팀(Philippe, Alexandre, Maxime, Romain)이 발행합니다. 연락처: contact@breezy.example.',
          },
        ],
      },
      {
        title: '발행 책임자',
        blocks: [{ p: '발행 책임자는 Breezy 팀의 프로젝트 책임자입니다.' }],
      },
      {
        title: '호스팅',
        blocks: [
          {
            p: '본 서비스는 컨테이너화(Docker)되어 있으며, 공개 배포 시 별도로 지정될 호스팅 제공업체에서 호스팅될 예정입니다. 교육 환경에서는 로컬에서 실행됩니다.',
          },
        ],
      },
      {
        title: '지식재산권',
        blocks: [
          {
            p: '“Breezy” 브랜드, 로고, 비주얼 아이덴티티, 그리고 본 서비스의 구조와 편집 콘텐츠는 지식재산권법에 의해 보호됩니다. 사전 허가 없이 전부 또는 일부를 복제하거나 표시하는 것은 금지됩니다.',
          },
          {
            p: '사용자가 게시한 콘텐츠(게시물, 댓글, 메시지)는 이용약관에 정한 조건에 따라 해당 작성자의 소유로 남습니다.',
          },
        ],
      },
      {
        title: '책임',
        blocks: [
          {
            p: '발행자는 제공되는 정보의 정확성을 보장하기 위해 노력하지만, 오류, 이용 불가 또는 사용자가 게시한 콘텐츠에 대해서는 책임을 지지 않습니다. 본 서비스는 교육 프로젝트이므로 연속성에 대한 보증 없이 “있는 그대로” 제공됩니다.',
          },
        ],
      },
      {
        title: '문의',
        blocks: [
          {
            p: '본 법적 고지에 관한 문의는 다음 주소로 연락해 주십시오: contact@breezy.example.',
          },
        ],
      },
    ],
  },

  cgu: {
    title: '이용약관',
    updatedAt: '2026년 6월 8일',
    intro:
      '본 이용약관(“약관”)은 소셜 네트워크 Breezy(“본 서비스”)에 대한 접근 및 이용을 규율합니다. 계정을 생성하거나 본 서비스를 이용함으로써 귀하는 본 약관에 조건 없이 동의하게 됩니다.',
    sections: [
      {
        title: '제1조 — 목적',
        blocks: [
          {
            p: 'Breezy는 짧은 게시물을 게시하고, 댓글을 달고, 좋아요를 누르고, 게시물을 리포스트하거나 인용하고, 다른 사용자를 팔로우하고, 암호화된 개인 메시지를 주고받으며, 그룹 및 커뮤니티에 참여할 수 있는 소셜 네트워크입니다.',
          },
        ],
      },
      {
        title: '제2조 — 가입 및 계정',
        blocks: [
          {
            list: [
              '가입에는 유효한 이메일 주소와 사용자 이름(고유 식별자)의 선택이 필요합니다.',
              '사용자 이름은 계정의 신원을 구성하며, 그 변경은 기간에 따라 제한될 수 있습니다.',
              '귀하는 자격 증명의 기밀 유지와 귀하의 계정에서 이루어지는 모든 활동에 대해 책임을 집니다.',
              '계정은 엄격히 개인용입니다. 귀하는 정확한 정보를 제공할 것에 동의합니다.',
            ],
          },
        ],
      },
      {
        title: '제3조 — 본 서비스 접근',
        blocks: [
          {
            p: '본 서비스는 무료로 이용할 수 있습니다. 발행자는 특히 유지보수를 위해 본 서비스의 전부 또는 일부를 변경, 중단 또는 종료할 수 있으며, 이에 대해 책임을 지지 않습니다.',
          },
        ],
      },
      {
        title: '제4조 — 게시물 및 상호작용',
        blocks: [
          {
            p: '짧은 게시물을 게시하고, 댓글(이모지 포함)로 답글을 달고, 좋아요를 누르고, 게시물을 리포스트하거나 인용하며, 자신의 게시물 중 하나를 프로필에 고정할 수 있습니다. 게시물과 댓글에 대한 자동 번역이 참고용으로 제공될 수 있습니다.',
          },
          {
            p: '귀하는 자신의 콘텐츠에 대한 소유권을 보유하며, 본 서비스가 그 운영의 범위 내에서 이를 호스팅하고 표시할 수 있도록 비독점적이고 무상인 라이선스를 본 서비스에 부여합니다. 귀하는 게시하는 내용에 대해 단독으로 책임을 집니다.',
          },
        ],
      },
      {
        title: '제5조 — 개인 메시지, 그룹 및 커뮤니티',
        blocks: [
          {
            list: [
              '개인 메시지와 그룹은 종단 간 암호화됩니다. 참여자만 그 내용을 읽을 수 있습니다. 발행자는 이러한 대화의 내용에 접근할 수 없습니다.',
              '커뮤니티는 준공개입니다. 그 이름은 공개되어 디렉터리에 등재되며, 그 내용은 관리자가 접근할 수 있습니다. 귀하는 이러한 차이를 인지한 상태에서 게시합니다.',
              '암호화 신원 키는 귀하의 기기에 보관됩니다. 새 기기에서는 이전 기록을 복호화할 수 없습니다.',
              '행동 규칙(제6조)은 개인 대화를 포함한 모든 대화에 적용됩니다.',
            ],
          },
        ],
      },
      {
        title: '제6조 — 행동 규칙',
        blocks: [
          { p: '본 서비스를 이용함에 있어 귀하는 다음과 같은 콘텐츠를 게시하지 않을 것에 동의합니다:' },
          {
            list: [
              '불법적, 명예훼손적, 모욕적, 증오적 또는 차별적인 콘텐츠;',
              '사생활 또는 제3자의 권리를 침해하는 콘텐츠;',
              '폭력적, 음란한 성격을 띠거나 증오를 선동하는 콘텐츠;',
              '괴롭힘, 스팸 또는 사기를 구성하는 콘텐츠.',
            ],
          },
        ],
      },
      {
        title: '제7조 — 모더레이션 및 역할',
        blocks: [
          {
            p: '본 서비스는 사용자, 모더레이터, 관리자의 세 가지 역할을 구분합니다. 모더레이터와 관리자는 본 약관에 반하는 공개 콘텐츠를 숨기거나 삭제할 수 있으며, 필요한 경우 해당 계정을 정지할 수 있습니다. 개인 메시지와 그룹의 종단 간 암호화로 인해 해당 콘텐츠의 모더레이션은 기술적으로 접근 가능한 범위로 제한됩니다.',
          },
        ],
      },
      {
        title: '제8조 — 책임 및 가용성',
        blocks: [
          {
            p: '본 서비스는 교육 프로젝트의 일환으로 “있는 그대로” 제공됩니다. 발행자는 중단이나 오류가 없음을 보장하지 않으며, 본 서비스의 이용 또는 이용 불가로 인해 발생하는 손해에 대해 책임을 지지 않습니다.',
          },
        ],
      },
      {
        title: '제9조 — 정지 및 해지',
        blocks: [
          {
            p: '귀하는 언제든지 계정을 삭제할 수 있습니다. 발행자는 본 약관, 특히 행동 규칙(제6조) 위반 시 계정을 정지하거나 해지할 수 있습니다.',
          },
        ],
      },
      {
        title: '제10조 — 개인정보',
        blocks: [
          {
            p: '귀하의 개인정보 처리에 관한 사항은 당사의 개인정보 보호정책(페이지 하단의 링크를 통해 접근 가능)에 설명되어 있습니다.',
          },
        ],
      },
      {
        title: '제11조 — 약관의 변경 및 준거법',
        blocks: [
          {
            p: '발행자는 언제든지 본 약관을 변경할 수 있으며, 적용되는 버전은 귀하가 이용하는 시점에 유효한 버전입니다. 본 약관은 프랑스법의 적용을 받습니다. 원만한 해결이 이루어지지 않을 경우 관할 법원은 발행자의 소재지를 관할하는 법원으로 합니다.',
          },
        ],
      },
    ],
  },

  confidentialite: {
    title: '개인정보 보호정책',
    updatedAt: '2026년 6월 8일',
    intro:
      '본 정책은 Breezy(“본 서비스”)가 일반 개인정보 보호법(GDPR)에 따라 귀하의 개인정보를 수집하고 처리하는 방식을 설명합니다.',
    sections: [
      {
        title: '개인정보 처리자',
        blocks: [
          {
            p: '개인정보 처리자는 본 서비스의 발행자(Breezy 프로젝트 팀)입니다. 문의: privacy@breezy.example.',
          },
        ],
      },
      {
        title: '수집하는 정보',
        blocks: [
          {
            list: [
              '계정 정보: 이메일 주소, 사용자 이름, 비밀번호(해시 처리하여 저장하며, 평문으로 저장하지 않음).',
              '프로필 정보: 표시 이름, 자기소개, 아바타, 배너, 웹사이트, 위치, 생년월일, 성별 및 국적(선택 사항).',
              '공개 콘텐츠 및 상호작용: 게시물, 댓글, 좋아요, 리포스트/인용, 팔로우(소셜 그래프).',
              '메시지: 개인 메시지와 그룹의 경우 서버는 암호화된 데이터(암호문 + 수신자별로 래핑된 키)만 저장하며 그 내용에 접근하지 않습니다. 커뮤니티(준공개)의 경우 콘텐츠 키는 서버가 보유합니다.',
              '기술 정보: 인증 토큰(JWT 및 리프레시 토큰)과 운영에 필요한 로그.',
            ],
          },
        ],
      },
      {
        title: '목적 및 법적 근거',
        blocks: [
          {
            list: [
              '본 서비스 제공 및 귀하의 계정 관리(약관의 이행).',
              '보안 확보, 사기 방지 및 공개 콘텐츠 모더레이션(정당한 이익).',
              '본 서비스 개선(정당한 이익).',
            ],
          },
        ],
      },
      {
        title: '보관 기간',
        blocks: [
          {
            p: '귀하의 정보는 계정이 활성 상태인 동안 보관됩니다. 계정이 삭제되는 경우, 법적 보관 의무를 조건으로 합리적인 기간 내에 삭제되거나 익명화됩니다.',
          },
        ],
      },
      {
        title: '수령인',
        blocks: [
          {
            p: '귀하의 정보는 애플리케이션의 내부 서비스(인증, 사용자, 프로필, 게시물, 메시지)에 의해 처리되며 제3자에게 판매되지 않습니다. 개인 메시지와 그룹의 내용은 종단 간 암호화되어 있어 발행자가 기술적으로 접근할 수 없습니다. 귀하의 정보는 법률이 요구하는 경우 관할 당국에 제공될 수 있습니다.',
          },
        ],
      },
      {
        title: '쿠키 및 로컬 저장소',
        blocks: [
          {
            list: [
              '엄격히 필요하고 안전한(httpOnly) 쿠키가 귀하의 세션(리프레시 토큰)을 유지합니다.',
              '귀하를 인증하기 위해 단기 액세스 토큰이 브라우저의 로컬 저장소에 보관될 수 있습니다.',
              '귀하의 메시지 개인 키는 귀하의 기기에 로컬로 저장되며(IndexedDB) 브라우저를 벗어나지 않습니다.',
              '광고용 또는 제3자 추적 쿠키는 일절 사용하지 않습니다.',
            ],
          },
        ],
      },
      {
        title: '보안',
        blocks: [
          {
            p: '비밀번호는 해시 처리되며(bcrypt), 인증은 서명되고 폐기 가능한 토큰에 기반하며, 개인 메시지는 클라이언트 측에서 종단 간 암호화(X25519 + XChaCha20-Poly1305)를 사용합니다. 완벽한 시스템은 없으므로 절대적인 보안을 보장할 수는 없습니다.',
          },
        ],
      },
      {
        title: '귀하의 권리',
        blocks: [
          { p: 'GDPR에 따라 귀하는 다음과 같은 권리를 가집니다:' },
          {
            list: [
              '귀하의 정보에 대한 접근권;',
              '정정권;',
              '삭제권(“잊힐 권리”);',
              '처리의 제한권 및 처리에 대한 반대권;',
              '귀하의 정보에 대한 이동권.',
            ],
          },
          {
            p: '이러한 권리를 행사하려면 privacy@breezy.example로 연락해 주십시오. 또한 관할 개인정보 보호 당국에 민원을 제기할 수 있습니다.',
          },
        ],
      },
      {
        title: '정책의 변경',
        blocks: [
          {
            p: '본 정책은 업데이트될 수 있습니다. 적용되는 버전은 본 서비스에 게시된 버전입니다. 귀하의 의무에 대한 자세한 내용은 이용약관을 참조하십시오.',
          },
        ],
      },
    ],
  },
}

// --- العربية -----------------------------------------------------------------

const AR: Record<LegalSlug, LegalDoc> = {
  'mentions-legales': {
    title: 'إشعار قانوني',
    updatedAt: '8 يونيو 2026',
    intro:
      'Breezy شبكة اجتماعية طُوّرت في إطار تعليمي (مشروع FISA INFO A3). تُقدَّم المعلومات أدناه بدافع الشفافية؛ ويتعين على الناشر استكمال بعض البيانات عند النشر العلني.',
    sections: [
      {
        title: 'ناشر الخدمة',
        blocks: [
          {
            p: 'يتولى فريق مشروع Breezy (Philippe وAlexandre وMaxime وRomain) نشر موقع وتطبيق Breezy (المشار إليهما فيما يلي بـ«الخدمة») في إطار مشروع طلابي. للتواصل: contact@breezy.example.',
          },
        ],
      },
      {
        title: 'مدير النشر',
        blocks: [{ p: 'مدير النشر هو المسؤول عن المشروع نيابةً عن فريق Breezy.' }],
      },
      {
        title: 'الاستضافة',
        blocks: [
          {
            p: 'الخدمة معبَّأة في حاويات (Docker) ومخصصة للاستضافة لدى مزوّد يُحدَّد عند النشر العلني. وفي البيئة التعليمية، تعمل محليًا.',
          },
        ],
      },
      {
        title: 'الملكية الفكرية',
        blocks: [
          {
            p: 'إنّ علامة «Breezy» والشعار والهوية البصرية، وكذلك بنية الخدمة ومحتواها التحريري، محمية بموجب قانون الملكية الفكرية. ويُحظر أي استنساخ أو عرض، كليًّا أو جزئيًّا، دون إذن مسبق.',
          },
          {
            p: 'تظل المحتويات التي ينشرها المستخدمون (المنشورات والتعليقات والرسائل) ملكًا لأصحابها، وفقًا للشروط المنصوص عليها في شروط الاستخدام.',
          },
        ],
      },
      {
        title: 'المسؤولية',
        blocks: [
          {
            p: 'يسعى الناشر إلى ضمان دقة المعلومات المنشورة، لكنه لا يتحمّل المسؤولية عن الأخطاء أو عدم التوفّر أو المحتويات التي ينشرها المستخدمون. ولمّا كانت الخدمة مشروعًا تعليميًا، فإنها تُقدَّم «كما هي»، دون ضمان للاستمرارية.',
          },
        ],
      },
      {
        title: 'التواصل',
        blocks: [
          {
            p: 'لأي استفسار يتعلق بهذا الإشعار القانوني، يمكنك المراسلة على: contact@breezy.example.',
          },
        ],
      },
    ],
  },

  cgu: {
    title: 'شروط الاستخدام',
    updatedAt: '8 يونيو 2026',
    intro:
      'تحكم شروط الاستخدام هذه («الشروط») الوصول إلى شبكة Breezy الاجتماعية («الخدمة») واستخدامها. وبإنشائك حسابًا أو باستخدامك الخدمة، فإنك تقبل هذه الشروط دون تحفّظ.',
    sections: [
      {
        title: 'المادة 1 — الموضوع',
        blocks: [
          {
            p: 'Breezy شبكة اجتماعية تتيح نشر رسائل قصيرة، والتعليق، والإعجاب، وإعادة النشر أو الاقتباس، ومتابعة مستخدمين آخرين، وتبادل رسائل خاصة مشفّرة، والمشاركة في المجموعات والمجتمعات.',
          },
        ],
      },
      {
        title: 'المادة 2 — التسجيل والحساب',
        blocks: [
          {
            list: [
              'يتطلب التسجيل عنوان بريد إلكتروني صالحًا واختيار اسم مستخدم (مُعرّف فريد).',
              'يشكّل اسم المستخدم هوية الحساب؛ وقد يكون تعديله محدودًا زمنيًا.',
              'أنت مسؤول عن سرية بيانات اعتمادك وعن كل نشاط يجري من حسابك.',
              'الحساب شخصي بحت. وتتعهّد بتقديم معلومات دقيقة.',
            ],
          },
        ],
      },
      {
        title: 'المادة 3 — الوصول إلى الخدمة',
        blocks: [
          {
            p: 'الوصول إلى الخدمة مجاني. ويجوز للناشر تطوير الخدمة كليًّا أو جزئيًّا أو تعليقها أو إيقافها، لا سيما لأسباب الصيانة، دون أن تترتب على ذلك أي مسؤولية.',
          },
        ],
      },
      {
        title: 'المادة 4 — المنشورات والتفاعلات',
        blocks: [
          {
            p: 'يمكنك نشر رسائل قصيرة، والرد عليها بتعليقات (بما في ذلك الرموز التعبيرية)، والإعجاب، وإعادة النشر أو الاقتباس، وتثبيت أحد منشوراتك في ملفك الشخصي. وقد تُتاح ترجمة آلية للمنشورات والتعليقات على سبيل الاستئناس فقط.',
          },
          {
            p: 'تحتفظ بملكية محتوياتك وتمنح الخدمة ترخيصًا غير حصري ومجاني لاستضافتها وعرضها في إطار تشغيلها. وأنت وحدك المسؤول عمّا تنشره.',
          },
        ],
      },
      {
        title: 'المادة 5 — الرسائل الخاصة والمجموعات والمجتمعات',
        blocks: [
          {
            list: [
              'الرسائل الخاصة والمجموعات مشفّرة من طرف إلى طرف: لا يستطيع قراءة محتواها سوى المشاركين. ولا يستطيع الناشر الوصول إلى محتوى هذه المراسلات.',
              'المجتمعات شبه عامة: اسمها عام ويظهر في دليل، وقد يكون محتواها متاحًا لمسؤول. وأنت تنشر فيها مع علمك بهذا الفرق.',
              'يُحفظ مفتاح هوية التشفير على جهازك؛ ولا يمكن لجهاز جديد فك تشفير السجل السابق.',
              'تنطبق قواعد السلوك (المادة 6) على جميع المراسلات، بما فيها الخاصة.',
            ],
          },
        ],
      },
      {
        title: 'المادة 6 — قواعد السلوك',
        blocks: [
          { p: 'باستخدامك الخدمة، تتعهّد بعدم نشر أي محتوى:' },
          {
            list: [
              'غير قانوني أو تشهيري أو مهين أو يحضّ على الكراهية أو تمييزي؛',
              'ينتهك الخصوصية أو حقوق الغير؛',
              'ذي طابع عنيف أو إباحي أو يحرّض على الكراهية؛',
              'يشكّل تحرّشًا أو رسائل مزعجة أو احتيالًا.',
            ],
          },
        ],
      },
      {
        title: 'المادة 7 — الإشراف والأدوار',
        blocks: [
          {
            p: 'تميّز الخدمة بين ثلاثة أدوار: المستخدم والمشرف والمسؤول. ويمكن للمشرفين والمسؤولين إخفاء المحتويات العامة المخالفة لهذه الشروط أو حذفها، وعند الاقتضاء تعليق الحسابات المعنية. ويحدّ التشفير من طرف إلى طرف للرسائل الخاصة والمجموعات من الإشراف على هذه المحتويات بما هو متاح تقنيًا.',
          },
        ],
      },
      {
        title: 'المادة 8 — المسؤولية والتوفّر',
        blocks: [
          {
            p: 'تُقدَّم الخدمة «كما هي» في إطار مشروع تعليمي. ولا يضمن الناشر خلوّها من الانقطاع أو الأخطاء، ولا يتحمّل المسؤولية عن الأضرار الناجمة عن استخدام الخدمة أو عدم توفّرها.',
          },
        ],
      },
      {
        title: 'المادة 9 — التعليق والإنهاء',
        blocks: [
          {
            p: 'يمكنك حذف حسابك في أي وقت. ويجوز للناشر تعليق حساب أو إنهاؤه في حال الإخلال بهذه الشروط، لا سيما قواعد السلوك (المادة 6).',
          },
        ],
      },
      {
        title: 'المادة 10 — البيانات الشخصية',
        blocks: [
          {
            p: 'تُوصف معالجة بياناتك الشخصية في سياسة الخصوصية الخاصة بنا (المتاحة عبر الروابط في أسفل الصفحة).',
          },
        ],
      },
      {
        title: 'المادة 11 — تعديل الشروط والقانون الواجب التطبيق',
        blocks: [
          {
            p: 'يجوز للناشر تعديل هذه الشروط في أي وقت؛ والنسخة الواجبة التطبيق هي السارية عند استخدامك. وتخضع هذه الشروط للقانون الفرنسي؛ وفي حال تعذّر التسوية الودية، تكون المحاكم المختصة هي محاكم مقر الناشر.',
          },
        ],
      },
    ],
  },

  confidentialite: {
    title: 'سياسة الخصوصية',
    updatedAt: '8 يونيو 2026',
    intro:
      'تصف هذه السياسة الطريقة التي تجمع بها Breezy («الخدمة») بياناتك الشخصية وتعالجها، وفقًا للائحة العامة لحماية البيانات (GDPR).',
    sections: [
      {
        title: 'المسؤول عن المعالجة',
        blocks: [
          {
            p: 'المسؤول عن المعالجة هو ناشر الخدمة (فريق مشروع Breezy). لأي استفسار: privacy@breezy.example.',
          },
        ],
      },
      {
        title: 'البيانات المُجمَّعة',
        blocks: [
          {
            list: [
              'بيانات الحساب: عنوان البريد الإلكتروني، اسم المستخدم، كلمة المرور (مخزّنة مُجزّأة، وليست بنص واضح أبدًا).',
              'بيانات الملف الشخصي: الاسم المعروض، النبذة التعريفية، الصورة الرمزية، اللافتة، الموقع الإلكتروني، الموقع الجغرافي، تاريخ الميلاد، الجنس والجنسية (اختيارية).',
              'المحتويات والتفاعلات العامة: المنشورات، التعليقات، الإعجابات، إعادة النشر/الاقتباسات، المتابعات (الرسم البياني الاجتماعي).',
              'المراسلة: بالنسبة للرسائل الخاصة والمجموعات، لا يخزّن الخادم سوى بيانات مشفّرة (نص مشفّر + مفتاح مغلّف لكل مستلِم) ولا يصل إلى محتواها؛ أما المجتمعات (شبه العامة) فيحتفظ الخادم بمفتاح المحتوى.',
              'البيانات التقنية: رموز المصادقة (JWT ورمز التحديث) والسجلات اللازمة للتشغيل.',
            ],
          },
        ],
      },
      {
        title: 'الأغراض والأسس القانونية',
        blocks: [
          {
            list: [
              'تقديم الخدمة وإدارة حسابك (تنفيذ الشروط).',
              'ضمان الأمن ومنع الاحتيال والإشراف على المحتويات العامة (المصلحة المشروعة).',
              'تحسين الخدمة (المصلحة المشروعة).',
            ],
          },
        ],
      },
      {
        title: 'مدد الاحتفاظ',
        blocks: [
          {
            p: 'يُحتفظ ببياناتك ما دام حسابك نشطًا. وفي حال حذف الحساب، تُحذف أو تُجهَّل هويتها خلال مدة معقولة، مع مراعاة الالتزامات القانونية بالاحتفاظ.',
          },
        ],
      },
      {
        title: 'المستلِمون',
        blocks: [
          {
            p: 'تُعالَج بياناتك من قِبل الخدمات الداخلية للتطبيق (المصادقة، المستخدمون، الملفات الشخصية، المنشورات، المراسلة) ولا تُباع لأطراف ثالثة. ومحتوى الرسائل الخاصة والمجموعات، المشفّر من طرف إلى طرف، غير متاح تقنيًا للناشر. وقد تُكشَف بياناتك للسلطات المختصة عندما يقتضي القانون ذلك.',
          },
        ],
      },
      {
        title: 'ملفات تعريف الارتباط والتخزين المحلي',
        blocks: [
          {
            list: [
              'يحافظ ملف تعريف ارتباط ضروري للغاية وآمن (httpOnly) على جلستك (رمز التحديث).',
              'قد يُحتفظ برمز وصول قصير الأمد في التخزين المحلي لمتصفحك لمصادقتك.',
              'يُخزَّن مفتاح المراسلة الخاص بك محليًا على جهازك (IndexedDB) ولا يغادر المتصفح أبدًا.',
              'لا تُستخدم أي ملفات تعريف ارتباط إعلانية أو تتبّع تابعة لأطراف ثالثة.',
            ],
          },
        ],
      },
      {
        title: 'الأمن',
        blocks: [
          {
            p: 'تُجزّأ كلمات المرور (bcrypt)، وتعتمد المصادقة على رموز موقّعة وقابلة للإبطال، وتستخدم المراسلة الخاصة تشفيرًا من طرف إلى طرف (X25519 + XChaCha20-Poly1305) من جهة العميل. ولمّا كان لا يوجد نظام معصوم من الخطأ، فلا يمكن ضمان أمن مطلق.',
          },
        ],
      },
      {
        title: 'حقوقك',
        blocks: [
          { p: 'وفقًا للائحة العامة لحماية البيانات (GDPR)، تتمتع بالحقوق التالية:' },
          {
            list: [
              'حق الوصول إلى بياناتك؛',
              'حق التصحيح؛',
              'حق المحو («الحق في النسيان»)؛',
              'حق تقييد المعالجة والاعتراض عليها؛',
              'حق نقل بياناتك.',
            ],
          },
          {
            p: 'لممارسة هذه الحقوق، راسِل privacy@breezy.example. ويمكنك أيضًا تقديم شكوى إلى الهيئة المختصة بحماية البيانات.',
          },
        ],
      },
      {
        title: 'تعديل السياسة',
        blocks: [
          {
            p: 'قد تُحدَّث هذه السياسة. والنسخة الواجبة التطبيق هي المنشورة على الخدمة. وللاطلاع على تفاصيل التزاماتك، راجِع شروط الاستخدام.',
          },
        ],
      },
    ],
  },
}

// --- हिन्दी -------------------------------------------------------------------

const HI: Record<LegalSlug, LegalDoc> = {
  'mentions-legales': {
    title: 'कानूनी नोटिस',
    updatedAt: '8 जून 2026',
    intro:
      'Breezy एक सोशल नेटवर्क है जिसे शैक्षिक संदर्भ (FISA INFO A3 परियोजना) में विकसित किया गया है। नीचे दी गई जानकारी पारदर्शिता के लिए प्रदान की गई है; कुछ विवरण सार्वजनिक परिनियोजन के समय प्रकाशक द्वारा पूरे किए जाने हैं।',
    sections: [
      {
        title: 'सेवा का प्रकाशक',
        blocks: [
          {
            p: 'Breezy वेबसाइट और एप्लिकेशन (आगे "सेवा") को Breezy परियोजना टीम (Philippe, Alexandre, Maxime, Romain) द्वारा एक छात्र परियोजना के रूप में प्रकाशित किया जाता है। संपर्क: contact@breezy.example।',
          },
        ],
      },
      {
        title: 'प्रकाशन निदेशक',
        blocks: [{ p: 'प्रकाशन निदेशक Breezy टीम की ओर से परियोजना प्रमुख हैं।' }],
      },
      {
        title: 'होस्टिंग',
        blocks: [
          {
            p: 'सेवा कंटेनरीकृत (Docker) है और सार्वजनिक परिनियोजन के समय निर्दिष्ट किए जाने वाले होस्टिंग प्रदाता के यहाँ होस्ट की जानी है। शैक्षिक परिवेश में, यह स्थानीय रूप से चलती है।',
          },
        ],
      },
      {
        title: 'बौद्धिक संपदा',
        blocks: [
          {
            p: '"Breezy" ब्रांड, लोगो, दृश्य पहचान, साथ ही सेवा की संरचना और संपादकीय सामग्री बौद्धिक संपदा कानून द्वारा संरक्षित हैं। पूर्व अनुमति के बिना किसी भी रूप में, पूर्ण या आंशिक, पुनरुत्पादन या प्रदर्शन निषिद्ध है।',
          },
          {
            p: 'उपयोगकर्ताओं द्वारा प्रकाशित सामग्री (पोस्ट, टिप्पणियाँ, संदेश) उपयोग की शर्तों में निर्धारित शर्तों के अधीन उनके लेखकों की संपत्ति बनी रहती है।',
          },
        ],
      },
      {
        title: 'दायित्व',
        blocks: [
          {
            p: 'प्रकाशक प्रसारित जानकारी की सटीकता सुनिश्चित करने का प्रयास करता है, परंतु त्रुटियों, अनुपलब्धता या उपयोगकर्ताओं द्वारा प्रकाशित सामग्री के लिए उत्तरदायी नहीं ठहराया जा सकता। चूँकि यह एक शैक्षिक परियोजना है, सेवा निरंतरता की किसी गारंटी के बिना "जैसी है" प्रदान की जाती है।',
          },
        ],
      },
      {
        title: 'संपर्क',
        blocks: [
          {
            p: 'इस कानूनी नोटिस से संबंधित किसी भी प्रश्न के लिए, आप इस पते पर लिख सकते हैं: contact@breezy.example।',
          },
        ],
      },
    ],
  },

  cgu: {
    title: 'उपयोग की शर्तें',
    updatedAt: '8 जून 2026',
    intro:
      'ये उपयोग की शर्तें ("शर्तें") Breezy सोशल नेटवर्क ("सेवा") तक पहुँच और उसके उपयोग को नियंत्रित करती हैं। खाता बनाकर या सेवा का उपयोग करके, आप इन शर्तों को बिना किसी आरक्षण के स्वीकार करते हैं।',
    sections: [
      {
        title: 'अनुच्छेद 1 — उद्देश्य',
        blocks: [
          {
            p: 'Breezy एक सोशल नेटवर्क है जो छोटे संदेश प्रकाशित करने, टिप्पणी करने, पसंद करने, पोस्ट को रिपोस्ट या उद्धृत करने, अन्य उपयोगकर्ताओं को फ़ॉलो करने, एन्क्रिप्टेड निजी संदेशों का आदान-प्रदान करने और समूहों व समुदायों में भाग लेने की अनुमति देता है।',
          },
        ],
      },
      {
        title: 'अनुच्छेद 2 — पंजीकरण और खाता',
        blocks: [
          {
            list: [
              'पंजीकरण के लिए एक वैध ई-मेल पता और एक उपयोगकर्ता नाम (अद्वितीय पहचानकर्ता) का चयन आवश्यक है।',
              'उपयोगकर्ता नाम खाते की पहचान बनाता है; इसका संशोधन समय के साथ सीमित हो सकता है।',
              'आप अपने क्रेडेंशियल की गोपनीयता और अपने खाते से की गई सभी गतिविधियों के लिए ज़िम्मेदार हैं।',
              'खाता पूर्णतः व्यक्तिगत है। आप सटीक जानकारी प्रदान करने का वचन देते हैं।',
            ],
          },
        ],
      },
      {
        title: 'अनुच्छेद 3 — सेवा तक पहुँच',
        blocks: [
          {
            p: 'सेवा निःशुल्क सुलभ है। प्रकाशक सेवा के संपूर्ण या किसी भाग को, विशेष रूप से रखरखाव के कारणों से, परिवर्तित, निलंबित या बंद कर सकता है, बिना इसके लिए कोई दायित्व वहन किए।',
          },
        ],
      },
      {
        title: 'अनुच्छेद 4 — प्रकाशन और अंतःक्रियाएँ',
        blocks: [
          {
            p: 'आप छोटे संदेश प्रकाशित कर सकते हैं, टिप्पणियों (इमोजी सहित) के साथ उत्तर दे सकते हैं, पसंद कर सकते हैं, पोस्ट को रिपोस्ट या उद्धृत कर सकते हैं, और अपनी किसी एक पोस्ट को अपनी प्रोफ़ाइल पर पिन कर सकते हैं। पोस्ट और टिप्पणियों का स्वचालित अनुवाद केवल संकेत के तौर पर प्रस्तुत किया जा सकता है।',
          },
          {
            p: 'आप अपनी सामग्री का स्वामित्व बनाए रखते हैं और सेवा को उसके संचालन के दायरे में उसे होस्ट और प्रदर्शित करने के लिए एक गैर-विशिष्ट, निःशुल्क लाइसेंस प्रदान करते हैं। आप जो प्रकाशित करते हैं उसके लिए आप अकेले ज़िम्मेदार हैं।',
          },
        ],
      },
      {
        title: 'अनुच्छेद 5 — निजी संदेश, समूह और समुदाय',
        blocks: [
          {
            list: [
              'निजी संदेश और समूह एंड-टू-एंड एन्क्रिप्टेड हैं: केवल प्रतिभागी ही उनकी सामग्री पढ़ सकते हैं। प्रकाशक के पास इन आदान-प्रदानों की सामग्री तक पहुँच नहीं है।',
              'समुदाय अर्ध-सार्वजनिक हैं: उनका नाम सार्वजनिक है और एक निर्देशिका में सूचीबद्ध है, और उनकी सामग्री एक व्यवस्थापक के लिए सुलभ हो सकती है। आप इस अंतर को जानते हुए उनमें प्रकाशित करते हैं।',
              'एन्क्रिप्शन पहचान कुंजी आपके डिवाइस पर संग्रहीत रहती है; एक नया डिवाइस पुराने इतिहास को डिक्रिप्ट नहीं कर सकता।',
              'आचरण नियम (अनुच्छेद 6) निजी सहित सभी आदान-प्रदानों पर लागू होते हैं।',
            ],
          },
        ],
      },
      {
        title: 'अनुच्छेद 6 — आचरण नियम',
        blocks: [
          { p: 'सेवा का उपयोग करते समय, आप ऐसी सामग्री प्रकाशित न करने का वचन देते हैं जो:' },
          {
            list: [
              'अवैध, मानहानिकारक, अपमानजनक, घृणास्पद या भेदभावपूर्ण हो;',
              'निजता या तीसरे पक्ष के अधिकारों का उल्लंघन करती हो;',
              'हिंसक, अश्लील प्रकृति की हो या घृणा भड़काती हो;',
              'उत्पीड़न, स्पैम या धोखाधड़ी का गठन करती हो।',
            ],
          },
        ],
      },
      {
        title: 'अनुच्छेद 7 — मॉडरेशन और भूमिकाएँ',
        blocks: [
          {
            p: 'सेवा तीन भूमिकाओं में अंतर करती है: उपयोगकर्ता, मॉडरेटर और व्यवस्थापक। मॉडरेटर और व्यवस्थापक इन शर्तों के विरुद्ध सार्वजनिक सामग्री को छिपा या हटा सकते हैं और, यथास्थिति, संबंधित खातों को निलंबित कर सकते हैं। निजी संदेशों और समूहों के एंड-टू-एंड एन्क्रिप्शन के कारण इन सामग्रियों का मॉडरेशन तकनीकी रूप से सुलभ सीमा तक ही सीमित रहता है।',
          },
        ],
      },
      {
        title: 'अनुच्छेद 8 — दायित्व और उपलब्धता',
        blocks: [
          {
            p: 'सेवा एक शैक्षिक परियोजना के दायरे में "जैसी है" प्रदान की जाती है। प्रकाशक रुकावट या त्रुटि के अभाव की गारंटी नहीं देता और सेवा के उपयोग या अनुपलब्धता से होने वाले नुकसान के लिए उत्तरदायी नहीं ठहराया जा सकता।',
          },
        ],
      },
      {
        title: 'अनुच्छेद 9 — निलंबन और समाप्ति',
        blocks: [
          {
            p: 'आप किसी भी समय अपना खाता हटा सकते हैं। इन शर्तों, विशेष रूप से आचरण नियमों (अनुच्छेद 6) के उल्लंघन की स्थिति में प्रकाशक किसी खाते को निलंबित या समाप्त कर सकता है।',
          },
        ],
      },
      {
        title: 'अनुच्छेद 10 — व्यक्तिगत डेटा',
        blocks: [
          {
            p: 'आपके व्यक्तिगत डेटा का प्रसंस्करण हमारी गोपनीयता नीति में वर्णित है (पृष्ठ के नीचे दिए गए लिंक के माध्यम से सुलभ)।',
          },
        ],
      },
      {
        title: 'अनुच्छेद 11 — शर्तों में संशोधन और लागू कानून',
        blocks: [
          {
            p: 'प्रकाशक किसी भी समय इन शर्तों में संशोधन कर सकता है; लागू संस्करण वह है जो आपके उपयोग के समय प्रभावी हो। ये शर्तें फ्रांसीसी कानून के अधीन हैं; सौहार्दपूर्ण समाधान न होने पर, सक्षम न्यायालय प्रकाशक के पंजीकृत कार्यालय के क्षेत्राधिकार वाले होंगे।',
          },
        ],
      },
    ],
  },

  confidentialite: {
    title: 'गोपनीयता नीति',
    updatedAt: '8 जून 2026',
    intro:
      'यह नीति वर्णन करती है कि Breezy ("सेवा") सामान्य डेटा संरक्षण विनियमन (GDPR) के अनुसार आपके व्यक्तिगत डेटा को किस प्रकार एकत्र और प्रसंस्कृत करती है।',
    sections: [
      {
        title: 'प्रसंस्करण नियंत्रक',
        blocks: [
          {
            p: 'प्रसंस्करण नियंत्रक सेवा का प्रकाशक है (Breezy परियोजना टीम)। किसी भी प्रश्न के लिए: privacy@breezy.example।',
          },
        ],
      },
      {
        title: 'एकत्र किया गया डेटा',
        blocks: [
          {
            list: [
              'खाता डेटा: ई-मेल पता, उपयोगकर्ता नाम, पासवर्ड (हैश के रूप में संग्रहीत, कभी स्पष्ट पाठ में नहीं)।',
              'प्रोफ़ाइल डेटा: प्रदर्शित नाम, परिचय, अवतार, बैनर, वेबसाइट, स्थान, जन्मतिथि, लिंग और राष्ट्रीयता (वैकल्पिक)।',
              'सार्वजनिक सामग्री और अंतःक्रियाएँ: पोस्ट, टिप्पणियाँ, पसंद, रिपोस्ट/उद्धरण, फ़ॉलो (सामाजिक ग्राफ़)।',
              'संदेश: निजी संदेशों और समूहों के लिए, सर्वर केवल एन्क्रिप्टेड डेटा (एन्क्रिप्टेड पाठ + प्रति प्राप्तकर्ता रैप की गई कुंजी) संग्रहीत करता है और उनकी सामग्री तक उसकी पहुँच नहीं है; समुदायों (अर्ध-सार्वजनिक) के लिए, सामग्री कुंजी सर्वर के पास होती है।',
              'तकनीकी डेटा: प्रमाणीकरण टोकन (JWT और रिफ़्रेश टोकन) और संचालन के लिए आवश्यक लॉग।',
            ],
          },
        ],
      },
      {
        title: 'उद्देश्य और कानूनी आधार',
        blocks: [
          {
            list: [
              'सेवा प्रदान करना और आपके खाते का प्रबंधन करना (शर्तों का निष्पादन)।',
              'सुरक्षा सुनिश्चित करना, धोखाधड़ी रोकना और सार्वजनिक सामग्री का मॉडरेशन करना (वैध हित)।',
              'सेवा में सुधार करना (वैध हित)।',
            ],
          },
        ],
      },
      {
        title: 'प्रतिधारण अवधि',
        blocks: [
          {
            p: 'जब तक आपका खाता सक्रिय रहता है, आपका डेटा संरक्षित रहता है। खाता हटाए जाने की स्थिति में, कानूनी प्रतिधारण दायित्वों के अधीन, इसे उचित समय के भीतर हटा या अनाम कर दिया जाता है।',
          },
        ],
      },
      {
        title: 'प्राप्तकर्ता',
        blocks: [
          {
            p: 'आपका डेटा एप्लिकेशन की आंतरिक सेवाओं (प्रमाणीकरण, उपयोगकर्ता, प्रोफ़ाइल, पोस्ट, संदेश) द्वारा प्रसंस्कृत किया जाता है और तीसरे पक्ष को नहीं बेचा जाता। निजी संदेशों और समूहों की सामग्री, एंड-टू-एंड एन्क्रिप्टेड होने के कारण, तकनीकी रूप से प्रकाशक के लिए सुलभ नहीं है। कानून द्वारा अपेक्षित होने पर आपका डेटा सक्षम प्राधिकरणों को बताया जा सकता है।',
          },
        ],
      },
      {
        title: 'कुकीज़ और स्थानीय भंडारण',
        blocks: [
          {
            list: [
              'एक नितांत आवश्यक और सुरक्षित (httpOnly) कुकी आपके सत्र (रिफ़्रेश टोकन) को बनाए रखती है।',
              'आपको प्रमाणित करने के लिए एक अल्पकालिक एक्सेस टोकन आपके ब्राउज़र के स्थानीय भंडारण में रखा जा सकता है।',
              'आपकी संदेश निजी कुंजी आपके डिवाइस पर स्थानीय रूप से संग्रहीत होती है (IndexedDB) और कभी ब्राउज़र से बाहर नहीं जाती।',
              'कोई विज्ञापन या तीसरे पक्ष की ट्रैकिंग कुकी उपयोग नहीं की जाती।',
            ],
          },
        ],
      },
      {
        title: 'सुरक्षा',
        blocks: [
          {
            p: 'पासवर्ड हैश किए जाते हैं (bcrypt), प्रमाणीकरण हस्ताक्षरित और प्रतिसंहरणीय टोकन पर आधारित है, और निजी संदेश क्लाइंट-साइड एंड-टू-एंड एन्क्रिप्शन (X25519 + XChaCha20-Poly1305) का उपयोग करते हैं। चूँकि कोई भी प्रणाली अचूक नहीं है, पूर्ण सुरक्षा की गारंटी नहीं दी जा सकती।',
          },
        ],
      },
      {
        title: 'आपके अधिकार',
        blocks: [
          { p: 'GDPR के अनुसार, आपके पास निम्नलिखित अधिकार हैं:' },
          {
            list: [
              'अपने डेटा तक पहुँच का अधिकार;',
              'सुधार का अधिकार;',
              'मिटाने का अधिकार ("भुला दिए जाने का अधिकार");',
              'प्रसंस्करण को सीमित करने और उसका विरोध करने का अधिकार;',
              'अपने डेटा की सुवाह्यता का अधिकार।',
            ],
          },
          {
            p: 'इन अधिकारों का प्रयोग करने के लिए, privacy@breezy.example पर लिखें। आप सक्षम डेटा संरक्षण प्राधिकरण के समक्ष शिकायत भी दर्ज कर सकते हैं।',
          },
        ],
      },
      {
        title: 'नीति में संशोधन',
        blocks: [
          {
            p: 'यह नीति अद्यतन की जा सकती है। लागू संस्करण वह है जो सेवा पर प्रकाशित है। अपने दायित्वों के विवरण के लिए, उपयोग की शर्तें देखें।',
          },
        ],
      },
    ],
  },
}

/** Registre du contenu légal, par locale puis par page (les 12 locales). */
export const legalContent: Record<Locale, Record<LegalSlug, LegalDoc>> = {
  fr: FR,
  en: EN,
  zh: ZH,
  es: ES,
  pt: PT,
  ru: RU,
  ja: JA,
  ko: KO,
  ar: AR,
  hi: HI,
  de: DE,
  it: IT,
}
