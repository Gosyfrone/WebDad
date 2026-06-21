# 10 — Signets, partage & vue visiteur

> Features : signets en collections · partage de post/profil · vue visiteur (fil public sans inscription).

## Vue d'ensemble

Trois features « secondaires » qui partagent une philosophie : **en faire le minimum côté serveur**. Les signets vivent
dans post-service (ils référencent un post) ; le partage est **100 % front** ; la vue visiteur est **front-only, zéro
backend** (la lecture publique existait déjà). L'angle de défense : savoir **où ne PAS ajouter de service**.

## Services concernés & pourquoi

| Service | Rôle | Pourquoi ici |
|---|---|---|
| **post** (MongoDB) | signets (collections many-to-many), lecture publique des posts | un signet référence un post → même service que likes/reposts |
| **message** (MongoDB) | partage via DM (lien dans un message chiffré) | réutilise `startDM`/`sendMessage` |
| **Frontend / BFF** | partage (ShareDialog), mode visiteur (`AuthPromptProvider`), middleware | tout le reste est front |

## Comment c'est codé

### Signets (collections)
- **Collections (many-to-many) + collection par défaut non supprimable + burst model.** Un signet référence un post → même service que likes/reposts. Many-to-many car un post peut être classé dans plusieurs collections.
- **Burst model** (façon « save » Instagram) : un clic court dans `BOOKMARK_SESSION_WINDOW` classe automatiquement dans la dernière collection (`filed`), sinon ouvre un sélecteur (`needs_choice`, ne classe rien — **le serveur ne devine jamais**). L'état de fenêtre est **serveur** (`last_bookmark_at`, par compte, multi-appareils, pas de triche d'horloge). Page `/signets`.

### Partage de post / profil
- **Front-only, aucun backend, pas de Swagger.** Boutons Partager sur la carte de fil, la vue photo et la vue profil → `ShareDialog` réutilisable. 3 voies : **message privé** (recherche → `startDM`+`sendMessage`, le lien en clair voyage dans le message **chiffré E2EE**, format inchangé), **partage natif** (`navigator.share()`), **copier le lien**.
- **Aperçu enrichi dans le chat** : `shared-link-preview.tsx` + `extractSharedRef` reconstruisent une carte cliquable (post : avatar/nom/extrait/miniature ; profil : photo+nom) via `getPostById`/`getUserByUsername` (cache module-level) ; URLs linkifiées.
- **Destinataires récents** persistés (`localStorage`) → remontés en tête de la recherche. Helpers purs `lib/share.ts`.

### Vue visiteur (fil public)
- **Front-only, zéro backend** : le post-service expose **déjà** la lecture publique (`GET /posts` et `/posts/:id` en `OptionalJWTAuth`, barrière de visibilité serveur → sans token, on ne voit que les posts publics). Décision : **ne RIEN ajouter côté serveur**.
- **Réutiliser le shell `(app)`**, pas un groupe `(public)` séparé : le visiteur voit le **même** fil avec la sidebar **amputée** (carte user → boutons Se connecter/S'inscrire, nav réduite à Accueil).
- **Le vrai risque n'était pas la garde mais la redirection forcée** : `apiFetch` redirige vers `/login` sur tout 401 dont le refresh échoue. Or le fil enrichit chaque post via des routes `/me`. **Parade : garder chaque appel `/me` derrière `getAccessToken()`** (early-return vide) — c'est la pièce centrale, pas le retrait du middleware.
- **Actions réservées → modale « Connecte-toi »** (`requireAuth`/`promptLogin`), pas redirection ni masquage : like/repost/citer/signet/commentaire invitent à s'inscrire sans quitter le fil. La **lecture** des commentaires reste libre ; seul l'**envoi** est gaté. `AuthPromptProvider`/`useAuthGate` (`isVisitor`/`requireAuth`/`promptLogin`), middleware ouvre `/feed`+`/posts/:id`, `/` → `/feed`.

## Décisions clés (et alternatives écartées)

- **Signets dans post-service** (réfèrent un post) + **burst model serveur** : l'état de fenêtre est serveur (multi-appareils, pas de triche). Le serveur ne classe jamais à l'aveugle (`needs_choice`).
- **Partage 100 % front** : aucun backend (le lien voyage dans le message E2EE existant pour le partage par DM).
- **Vue visiteur front-only** : la lecture publique existait déjà → ne rien ajouter côté serveur. *Écarté* : un groupe de routes `(public)` parallèle (aurait dupliqué layout + feed pour rien).
- **Parade = garder les appels `/me` derrière le token**, pas retirer le middleware : le vrai bug était la redirection forcée sur 401.
- **Actions gatées par modale**, pas masquage/redirection : choix produit (inviter à s'inscrire sans quitter le fil).

## Points de défense / Q&A anticipées

- *« Le visiteur peut-il voir un post privé ? »* → non : `OptionalJWTAuth` + barrière serveur → seuls les posts publics sortent.
- *« Pourquoi le partage n'a pas de backend ? »* → un partage est un lien ; les 3 voies (DM/natif/copie) sont 100 % client, le DM réutilise la messagerie E2EE.

## Limites assumées / perspectives

- Vue visiteur : garde UX côté front (le backend protège déjà les écritures par JWT ; un client peut lire les routes publiques directement — c'est précisément leur contrat).
