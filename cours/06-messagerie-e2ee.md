# 06 — Messagerie chiffrée (E2EE)

> Features : messagerie privée chiffrée de bout en bout (DM / groupes / communautés) · serveur aveugle ·
> pièces jointes chiffrées · édition · état lu/non-lu + badge · mute · **reçus « Envoyé/Ouvert »** ·
> **« en train d'écrire »** · **suppression pour tous** · **sauvegarde de clé par passphrase (multi-appareils)**.

## Vue d'ensemble

La feature secondaire la plus ambitieuse, et le meilleur argument « sécurité » de la soutenance : **le serveur de
messagerie est aveugle** (admin-proof) pour les DM et groupes. Tout le chiffrement se fait dans le navigateur ;
**message-service (MongoDB)** ne voit que des **enveloppes** (`ciphertext`/`nonce`) et des **métadonnées** (qui, quand,
lu/non-lu) — jamais de clair. Les **communautés** sont un compromis assumé (clé côté serveur, admin-readable).

## Services concernés & pourquoi

| Service | Rôle | Pourquoi ici |
|---|---|---|
| **message** (MongoDB) | conversations, enveloppes chiffrées, clés publiques, curseurs lu/remis, sauvegardes de clé | propriétaire du domaine messagerie, **aveugle** par conception |
| **media** (MinIO) | blobs de pièces jointes **chiffrés** (opaques) | content-agnostic → le chiffrement reste E2EE de bout en bout |
| **notification** | notifie un nouveau message (agrégé par expéditeur) | DM/groupes privés seulement, jamais les communautés |
| **Frontend** | tout le chiffrement (X25519, scellés), IndexedDB, Web Worker KDF | l'E2EE impose que la crypto vive côté client |

## Comment c'est codé

### Modèle hybride (le compromis clé)
- Chaque utilisateur a une **paire X25519** (privée en **IndexedDB**, **par appareil**). Chaque conversation a une **clé de contenu symétrique** ; pour les DM/groupes elle est **scellée par membre** (sealed box anonyme) → le serveur ne voit que des enveloppes et **ne déchiffre rien = admin-proof**.
- **Communautés** : la clé reste **côté serveur** (remise à chaque membre rejoignant, auto-join illimité) → semi-public, **admin-readable** par conception. *Raison* : l'E2EE strict avec des viewers inconnus illimités est impossible (il faut un détenteur de clé en ligne) → modèle « canal Telegram ».
- **Groupes** : owner + membres, tout membre invite (il détient la clé), gestion owner-only, cap 32, nom chiffré, **clé unique sans rotation** (un nouveau membre lit tout l'historique ; forward secrecy = futur). **Communautés** : viewer/talker, cap 32 talkers, viewers illimités, **nom en clair** + annuaire public.

### Schéma « pas de clair au repos »
- `messages` n'ont que `ciphertext`+`nonce` (et après édition `original_ciphertext`+`original_nonce`) ; `user_keys` ne stocke que la clé **publique** ; `conversations.content_key` **seulement** pour les communautés.
- **Pagination par curseur** (`before=<messageId>` sur `_id`, pas offset) : stable sur un chat live (les nouveaux messages arrivent par WS en bas sans perturber le défilement arrière).

### Édition, suppression, état lu, mute
- **Édition** owner-only : remplace le ciphertext, conserve la 1ʳᵉ version chiffrée (affichage atténué), broadcast `message_updated` par WS.
- **Suppression « pour tous »** = tombstone : `deleted_at` + **effacement** de `ciphertext`/`nonce` (contenu réellement parti — le serveur était déjà aveugle, donc honnête). Autorisation pure testée `canDeleteMessage` : l'auteur toujours ; en groupe/communauté, l'owner/admin aussi (modération) ; **pas de modération en DM**. Réutilise l'événement `message_updated` (pas de nouveau type).
- **« Lu » = donnée serveur** (`members.last_read_at`), multi-appareils ; `GET /unread-count` calcule depuis les métadonnées **seulement** (jamais le ciphertext) → E2EE intact. **Mute** (`members.muted_at`) exclut du badge mais reste non-lu dans la liste. Pin/clear sont **par utilisateur** (sur `members`), sans broadcast.

### Reçus « Envoyé / Ouvert »
- Accusés **côté expéditeur, métadonnées serveur seulement** (timestamps, jamais le ciphertext). Deux curseurs par membre : `last_read_at` (= Ouvert) + `last_delivered_at` (= Remis). **Remise pilotée serveur** (pas d'ack client) : un message est *Remis* quand le serveur le pousse réellement (socket WS ouverte au send, ou fetch via `ListMessages`). Lire implique recevoir (remis ≥ ouvert).
- Événement WS `receipt` diffusé aux **autres** membres ; UI : **un seul marqueur sous mon dernier message** — 1 coche grise = Envoyé, 2 coches couleur = Ouvert ; groupe partiel = 1 coche + nombre de lecteurs. Logique de placement = fonction pure testée (`planReceipts`), maj live pure (`applyReceipt`, monotone). **DM + groupes seulement** (pas les communautés).

### « En train d'écrire »
- Éphémère, **non persisté**, E2EE-safe (métadonnées : `user_id`+`conversation_id`). Transport **REST** `POST .../typing` (plutôt que rendre le WS bidirectionnel) : ping throttlé ~3 s ; le serveur diffuse `typing` ; **pas de signal « stop »** → le receveur pose une **expiration +6 s** qui s'efface seule.

### Sauvegarde de clé par passphrase (multi-appareils), zero-knowledge
- La clé privée X25519 était **par appareil** → illisible sur un 2ᵉ appareil. Désormais elle est **scellée côté client** (`XChaCha20-Poly1305`) sous une KEK dérivée d'une **passphrase utilisateur** (**Argon2id, 19 MiB / t=2 = minimum OWASP**, choisi pour mobile : 64 MiB/t=3 figeait le thread ~15-25 s → « spinner infini » ; **auto-upgrade** des anciens backups lourds au prochain unlock).
- KDF dans un **Web Worker** (`key-backup.worker.ts`) → UI réactive. Stocké serveur dans `key_backups` comme **blob opaque** (`salt`, `nonce`, `wrapped_private_key`, `kdf_params`, `public_key`). **Le serveur ne voit ni la passphrase ni la clé privée → admin-proof préservé.**
- Identité IndexedDB **scopée par `userId`** (était un `self` global partagé par tous les comptes → mauvaise clé réutilisée). États : `ready` / `unlock` (backup mais pas de clé locale → autre appareil) / `setup` (pas de backup). `PassphraseGate` floute la vue messages tant que l'identité n'est pas disponible.

### Pièces jointes (E2EE)
- Fichier chiffré côté client → blob opaque uploadé sur media-service → id/nonce/mime dans l'**enveloppe chiffrée** (`{v:1,text,media[]}`) → **serveur aveugle, zéro changement de schéma**.

## Décisions clés (et alternatives écartées)

- **Modèle hybride** : DM/groupes admin-proof (scellé par membre) ; communautés admin-readable (clé serveur). *Écarté* : E2EE strict pour les communautés (impossible avec viewers inconnus illimités sans détenteur de clé en ligne).
- **État lu = donnée serveur** plutôt que localStorage par appareil (tranché avec l'utilisateur) : cohérent multi-appareils, calculé depuis les métadonnées → E2EE intact.
- **Reçus = 1 seul marqueur sous le dernier message** (raffiné après test iPhone) : un design à 2 marqueurs faisait apparaître la coche « coincée » sur un vieux message.
- **Typing en REST, pas WS bidirectionnel** : plus simple, expiration auto sans signal stop.
- **Argon2id 19 MiB / t=2** plutôt que 64 MiB / t=3 : compromis mobile (KDF JS synchrone), reste au minimum OWASP, auto-upgrade des anciens.
- **Suppression = tombstone honnête** : le serveur étant aveugle, effacer le ciphertext est une vraie suppression « pour tous ».
- **QR/crypto côté client obligatoire** : `node_modules` root-owned interdit d'ajouter des dépendances ; crypto maison défense-friendly.

## Points de défense / Q&A anticipées

- *« Un admin peut-il lire un DM ? »* → non : clé scellée par membre, le serveur n'a que des enveloppes. (Communautés = oui par conception, annoncé.)
- *« Comment l'état lu reste-t-il E2EE ? »* → ce sont des **timestamps** (métadonnées), jamais le contenu.
- *« Passphrase oubliée ? »* → backup irrécupérable (le prix du zero-knowledge) ; un appareil sans clé ni backup régénère une clé (historique antérieur illisible là).

## Limites assumées / perspectives

- **Reset admin de passphrase** différé (seule sémantique crypto-honnête = « wipe backup → nouvelle identité » ; pas d'escrow pour garder l'admin-proof).
- Rotation de clé / forward secrecy (groupes) — futur.
- Nettoyage des blobs média orphelins à la suppression — TODO.
