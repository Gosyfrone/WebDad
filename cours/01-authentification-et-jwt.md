# 01 — Authentification & JWT

> Features : inscription/connexion classiques · connexion sociale (OAuth Google/GitHub) · JWT + routes protégées ·
> validation/propagation JWT au gateway · gestion des rôles (User/Mod/Admin) · comptes créés par l'administrateur.
> (MFA, vérif e-mail et reset sont traités dans la **fiche 02**.)

## Vue d'ensemble

L'authentification est le critère de notation le plus explicite (« sécurité : sessions JWT, routes protégées »).
Breezy combine **sessions JWT à double jeton** (access court + refresh long révocable), **connexion sociale OAuth**,
et un **gating par rôle**. La règle directrice : auth-service ne possède **que** les credentials et la signature des
jetons — il ne joint jamais de données utilisateur (« une donnée = un service »).

## Services concernés & pourquoi

| Service | Rôle | Pourquoi ici |
|---|---|---|
| **auth** (PostgreSQL) | credentials, bcrypt, émission/rotation/révocation des JWT, OAuth, enum `provider` | propriétaire unique du secret de signature et des mots de passe |
| **user** (PostgreSQL) | résout `username → user_id`, porte le rôle au provisioning | `username` appartient à user, pas à auth |
| **Frontend / BFF Next** | orchestration login par username, pose du cookie refresh httpOnly, single-flight refresh | le cookie httpOnly doit être posé same-origin (le navigateur ne le donne jamais au JS) |
| **api-gateway** | premier filtre JWT « valider-si-présent », anti-spoof, propagation des claims | centralise la validation et décharge les services |

## Comment c'est codé

### Sessions JWT (double jeton)
- **Access token** : JWT HS256, **5 min**, stocké en `localStorage`, envoyé en `Authorization: Bearer` à la gateway.
- **Refresh token** : aléatoire **opaque**, **24h**, stocké **haché SHA-256** en base, posé en **cookie httpOnly** via le BFF Next. Roté à chaque `/refresh`, révoqué au `/logout`.
- Le **back ne renouvelle jamais tout seul** : il signe `exp` et répond `401 code:token_expired` (distinct de `invalid` → pas de refresh) ; **le front pilote** le refresh, en **single-flight** (un seul refresh concurrent pour éviter les tempêtes de refresh sur 401 simultanés).
- Routes : `/auth/{register,login,refresh,logout,validate}` + `/health`.

### Connexion par username
- `username` reste possédé par user-service. Le **BFF Next** résout `username → user_id` via la gateway, **puis** appelle auth avec `user_id + password`. Auth ne joint jamais de données user.

### Connexion sociale OAuth
- **Registry générique de providers, deux familles** (`auth-service/internal/oauth`, map `defs`). Ajouter un provider = une entrée de map + des variables d'env ; un provider sans `CLIENT_ID` est ignoré (→ 404 inoffensif).
  - **`kindOIDC`** (Google) : Authorization Code, échange `code → tokens` **côté serveur**, vérification de l'`id_token` (signature via JWKS, `issuer`, `audience=client_id`). `email_verified` **requis**. Discovery paresseuse (au 1ᵉʳ usage, pas au boot → résilient offline).
  - **`kindOAuth2`** (GitHub) : pas d'`id_token` → `code → access_token` puis appel **userinfo** mappé en `Identity{Subject,Email}`. Spécifique GitHub : repli `/user/emails` pour l'e-mail **primaire vérifié**, `id` numérique, `User-Agent` requis.
- **Anti-CSRF `state`** généré serveur, stocké par le front, re-vérifié au callback. Le front n'obtient **que** l'URL d'autorisation et relaie le `code` — il ne fournit jamais le token.
- **Réconciliation par e-mail** : compte existant → on connecte et on remplit `provider_subject` (seulement si vide, pas de hijack) ; absent → création avec `password NULL`. Un login classique sur un compte sans mot de passe → `409 ErrNoLocalPassword` qui oriente vers le provider.
- **Nouveau compte OAuth = onboarding bloquant sans JWT** : l'`exchange` n'écrit qu'une ligne `oauth_signup_tokens` (pending 15 min) et renvoie `onboarding_required`. Aucun credential / user / profil / JWT avant que l'utilisateur ait saisi username + date de naissance + accepté les CGU sur la page publique `/auth/oauth/terms`, puis `/complete` (transaction).

### Validation & propagation JWT au gateway
- Middleware global `PropagateJWT` (`api-gateway/internal/middleware/jwt.go`), **avant** le reverse proxy, politique **« valider-si-présent »** :
  - aucun token → passe-plat (login, refresh par cookie, vue visiteur publique) ;
  - token présent mais invalide/expiré → **401 au plus tôt**, n'atteint jamais le service ;
  - token valide → injecte `X-User-Id` / `X-User-Role` / `X-Email-Verified` vers le service.
- **Anti-spoof non négociable** : les en-têtes d'identité entrants sont **toujours strippés en premier** (même sans token) — seul le gateway peut les poser.
- **Défense en profondeur** : chaque service **revalide** le JWT lui-même (même `JWT_SECRET` partagé) ; les en-têtes propagés sont posés pour usage futur, pas comme unique source de confiance.

### Rôles (User / Modérateur / Administrateur) & comptes créés par admin
- Le **rôle vit dans le JWT** (claim). Les services gardent leurs routes sensibles par middleware (`AdminOnly`, `ModeratorOnly`) ; ex. user-service applique la suppression admin, report-service les actions de modération.
- **Admin crée un compte de force** (`POST /auth/users`, AdminOnly) : credentials `must_change_password=true` (mot de passe temporaire **mailé**) + **vérif e-mail obligatoire**. Le mail porte le lien de vérif **et** le mot de passe temporaire ; cliquer le lien vérifie l'adresse **et** ouvre la session → atterrissage feed + modale de changement de mot de passe. Le username demandé est provisionné dans user-service (`POST /users/admin`) ; s'il est pris, suffixé `_<8 hex>` + `username_pending=true` (levé au vrai `PATCH /users/me`). Court-circuit dev : `ADMIN_CREATE_AUTO_VERIFY=true`.
- Les drapeaux (`must_change_password`, `username_pending`) **chevauchent le JWT / `/users/me`** → le front affiche une modale bloquante **sans appel supplémentaire** ; `POST /auth/password/change` lève le drapeau, révoque les autres sessions et ré-émet un token propre.

## Décisions clés (et alternatives écartées)

- **Access en `localStorage` + refresh httpOnly** : compromis assumé — l'access est exposé au XSS mais **vit 5 min** ; le refresh est non-volable. Cookie géré par le BFF same-origin → évite la complexité CORS/SameSite cross-origin.
- **Refresh opaque haché en base** plutôt que JWT auto-portant : **révocable** (logout, reset), une fuite de table ne livre aucun token utilisable, base d'une future détection de réutilisation.
- **Registry OAuth générique** plutôt que du code par provider : un provider = de la **config**, pas du code. *Écarté* : LinkedIn (exigeait une Page entreprise) → remplacé par GitHub ; Microsoft puis Facebook/Spotify retirés sur demande (l'enum PG garde les valeurs inertes, Postgres n'autorisant pas `DROP VALUE`).
- **Vérification de l'`id_token` côté serveur** : ne jamais faire confiance à un token fourni par le front (sinon usurpation de compte par e-mail concordant).
- **Onboarding OAuth pending sans JWT** : un nouveau compte social n'existe (credential + JWT) **qu'après** acceptation des CGU → conforme à l'exigence légale, et impossible de créer un compte sans consentement.
- **Gateway « valider-si-présent »** plutôt qu'une allowlist publique ou une table de politique par route : zéro config par route, ne casse jamais les routes publiques / la vue visiteur, n'oblige pas à dupliquer le routage de chaque service.

## Points de défense / Q&A anticipées

- *« Pourquoi 5 min d'access ? »* → fenêtre XSS minimale, refresh silencieux côté front.
- *« Le gateway protège déjà, pourquoi revalider dans les services ? »* → défense en profondeur : si un service est joignable directement (réseau interne), il ne fait pas confiance aveugle aux en-têtes.
- *« Un client peut-il forger `X-User-Id` ? »* → non : strip systématique en entrée du gateway, seul le gateway les pose.

## Limites assumées / perspectives

- Détection de **réutilisation de refresh token** (rotation déjà en place, reuse-detection à brancher).
- **Faire consommer les en-têtes propagés** par les services (alléger leur revalidation) — perspective.
- **Rôle/username réels côté front** : le layout `(app)` utilise encore un `PLACEHOLDER_ROLE`.
