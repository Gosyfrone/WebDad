# CONTEXT — Distributed App Dev Project (FISA INFO A3)

> This file is the single source of truth for Claude Code.
> **Always read it at session start. Update it after each significant change.**

---

## 1. ARCHITECTURE

4-layer microservices app (see archi.png):

```
[Client] User / Moderator / Administrator
    ↓
[Web]  Frontend (React or similar)
    ↓
[Services] API Gateway → User Service | Post Service | Profil Service | Auth Service
    ↓
[Data] PostgreSQL (User) | MongoDB (Post) | MongoDB (Profil) | PostgreSQL (Auth)
```

**Key rules:**
- All inter-service communication goes through the API Gateway
- JWT auth (Auth Service) + protected routes
- Full Docker containerization (docker-compose)
- 3 user roles: User, Moderator, Administrator

---

## 1bis. STACK TECHNIQUE

**Backend**
- Langage : **Go 1.22**
- Framework HTTP : **Gin** (`github.com/gin-gonic/gin`)
- Un service = **un module Go indépendant** (chacun a son propre `go.mod`)
- Linter : **golangci-lint**
- Hot reload (dev) : **air** (`github.com/cosmtrek/air`)
- Build / lint / test par service : via `Makefile` (`make run`, `make build`, `make lint`, `make test`)

**Frontend**
- Framework : **Next.js 14** (App Router)
- Langage : **TypeScript**
- Style : **Tailwind CSS** + **shadcn/ui** (thème **slate**)
- Gestionnaire de paquets : **npm**
- Port de dev : **3000** (convention Next.js — `npm run dev` est configuré avec `next dev -p 3000`)
- Variable d'environnement principale : `NEXT_PUBLIC_API_URL` (URL de l'API Gateway, par défaut `http://localhost:8080`)

**Ports par défaut**
| Service              | Port |
| -------------------- | ---- |
| frontend (Next.js)   | 3000 |
| api-gateway          | 8080 |
| auth-service         | 8081 |
| user-service         | 8082 |
| profil-service       | 8083 |
| post-service         | 8084 |

Plus aucun conflit : le frontend (3000) et le backend (8080+) occupent des plages distinctes.

**Bases de données**
| Service          | Base       |
| ---------------- | ---------- |
| auth-service     | PostgreSQL |
| user-service     | PostgreSQL |
| profil-service   | MongoDB    |
| post-service     | MongoDB    |

---

## 2. EVALUATION CRITERIA (grading grid)

### Group grade — Deliverable (written report)
| Criterion | What's needed for A (5pts) |
|---|---|
| Needs analysis | Well understood and explained |
| Architecture diagram | Clear, readable, matches requirements |
| Prioritization | Primary + secondary features, justified order |
| Planning | Tools (Trello/Gantt), tracked, delays explained |
| Methodology | Global approach, correct technical terms |
| Interface (Wireframe) | Wireframe + final, gaps justified, intuitive UX |
| Features presentation | Clear, limitations explicit |
| Improvements & outlook | Prioritized, with effort estimates |
| Report writing | Well structured, professional layout |

### Group grade — Defense (oral)
| Criterion | What's needed for A (5pts) |
|---|---|
| Context & approach | Clear, adapted to audience |
| Choices made | Justified (architecture, features) |
| Microservices architecture | Coherent, matches diagram or changes argued |
| Security | JWT sessions, protected routes |
| Containerization | Full Docker, correct params |
| Primary features | All present and functional |
| Secondary features | Several, well chosen |
| Users (roles) | All 3 roles functional |
| Oral presentation | Pro-quality slides + scripted demo |
| Timing & energy | Balanced speaking time, dynamic, on schedule |

### Individual grade
| Criterion | What's needed for A (5pts) |
|---|---|
| Technical mastery | Full block skills demonstrated, strong Q&A |
| English | Demonstrates skills in English during defense |

**Grading scale:** A=5pts / B=4pts / C=2pts / D=1pt

**Team members:** Zaid, Perujan, Candis, Théo

---

## 3. PROJECT STATE

> **Update this section after each work session.**

### Services status
| Service | Status | DB | Notes |
|---|---|---|---|
| Auth Service | 🟡 WIP | PostgreSQL | Squelette fonctionnel (config/db/models/services/handlers/middleware/router). Routes `/auth/register`, `/auth/login`, `/auth/refresh`, `/auth/logout`, `/auth/validate`, `/health`. JWT HS256 (claims user_id/email/role), bcrypt. **Refresh tokens** : access court (`JWT_EXPIRY`=15m) + refresh long (`REFRESH_EXPIRY`=24h). Refresh = jeton opaque (crypto/rand) **stocké haché** (SHA-256) dans `refresh_tokens`, **rotation** à chaque `/refresh` (ancien révoqué), révocation à `/logout`. Login/Register/Refresh renvoient `{token, refresh_token, user}`. Middleware JWT distingue l'expiration (`code: token_expired`) du token invalide. **Service autonome** : schéma embarqué au boot (`internal/db/schema.sql`, idempotent). Seed admin optionnel (`SEED_DEFAULT_ADMIN`). TODO : tests Go, détection de réutilisation de refresh token |
| User Service | 🟢 OK (v1) | PostgreSQL | Service complet (config/db/models/repository/service/handlers/middleware/router) calqué sur auth. **Autonome** : schéma embarqué au boot (`users`+`follows`, seed admin UUID figé) → plus d'init-db monté. **CRUD users** : `GET /users` (paginé, masque les désactivés), `GET /users/:id` & `GET /users/by-username/:username` (+ compteurs followers/following), `POST /users`, `GET/PATCH /users/me`, `DELETE /users/:id` (admin). **Graphe social `follows`** : `POST/DELETE /users/:id/follow` (idempotents, anti-self-follow, 404 si cible absente), `GET /users/:id/followers|following` (paginés). **Provisioning paresseux** sur `/users/me` depuis les claims JWT (avec résolution de collision de username). Middleware JWT (validation locale, secret partagé) + contrôle de rôle admin. Validation username (regex + mots réservés). **`display_name` déplacé vers profil-service** : user-service ne porte plus que `username` (identité immuable) + l'état du compte (`is_active`) + le graphe social. **`username_changed_at`** : colonne nullable enregistrée au boot, posée à `NOW()` (en SQL) uniquement quand le handle change réellement → baseline d'un cooldown configurable (`USERNAME_CHANGE_COOLDOWN`, défaut 0 = désactivé, refus en `429`). Tests Go (middleware JWT + validation). **Testé e2e** (Postgres jetable). Compteurs calculés (COUNT) — dénormalisation = perspective. TODO : tests d'intégration repo (DB) |
| Post Service | 🟡 WIP | MongoDB | Connexion Mongo via `.env` (URI construite, plus rien en dur). **Autonome** : crée ses collections (posts/comments/likes/reports) + validateurs `$jsonSchema` + index au boot (`EnsureSchema`, idempotent) → `post-init.js` supprimé. Champs snake_case (`author_id`/`content`/`created_at`). Routes posts create/list OK (testées) ; comments/likes = stubs. `make dev` vérifié (air + Mongo) |
| Profil Service | 🟢 OK (v1) | MongoDB | Service complet calqué sur post-service (config/database/models/repository/service/handler/middleware). **Autonome** : schéma embarqué au boot (`EnsureSchema`, collection `profiles` + validateur `$jsonSchema` + index unique `user_id` + seed admin upsert). **Propriétaire des champs décoratifs/perso UNIQUEMENT** : `user_id`, `display_name`, `bio`, `avatar_url`, `banner_url`, `website`, `location`, `birth_date` (date, opt.), `gender` (enum `male`/`female`, opt.), timestamps (pas de username ni compteurs → user-service). **Routes implémentées** : `GET /profils/:userId` (public, 404 si absent, **pas** de provisioning), `GET /profils/me` (JWT, **lecture seule — ne crée RIEN**, `404` si pas de profil), `PATCH /profils/me` (update partiel via `FindOneAndUpdate`, `404` si absent), `POST /profils` (JWT, **UNIQUE voie de création**, `{display_name}` **obligatoire** → le BFF pose `display_name = username` au register ; `409` si existe), `DELETE /profils/:userId` (admin). **Règle `birth_date` set-once** : modifiable tant que vide, `409` si on tente de changer une date posée. **`display_name_changed_at`** : enregistré quand le nom change réellement → baseline du cooldown configurable (`DISPLAY_NAME_CHANGE_COOLDOWN`, défaut 0 = désactivé, refus en `429`). Middleware JWT. Tests Go (routes/health/401 + **unit pur `planUpdate`** : set-once + cooldown). build/vet/gofmt/tests OK. TODO : brancher le front (lecture agrégée user+profil+post), upload avatar/bannière |
| API Gateway | 🟡 WIP | — | Reverse proxy (`httputil.ReverseProxy`) : préfixe `/auth`,`/users`,`/profils`,`/posts` → service cible (URLs via `.env`). Middleware CORS (origines via `CORS_ALLOWED_ORIGINS`). `/auth/*` proxifié vers auth-service. Middleware JWT à ajouter pour les routes protégées (login) |
| Frontend | 🟡 WIP | — | Next.js 14 : squelette + routing + layout responsive (style X). **Auth refresh tokens** : route handlers Next `/api/auth/{login,register,refresh,logout}` (BFF). Access token (15m) en **localStorage** ; refresh token (24h) en **cookie httpOnly `breezy-refresh`** géré par le BFF (jamais exposé au JS). `lib/auth-client.ts` : `apiFetch` (Bearer + refresh auto sur 401, **single-flight** + rejeu) + `logout`. **Garde de session** : `middleware.ts` redirige vers /login si pas de cookie refresh sur les routes `(app)`. Boutons « Se déconnecter » câblés (sidebar + tiroir mobile). **Provisioning user au login/register** (serveur, best-effort). Pages feed + profil + placeholders. Mobile-first (thème clair/sombre). Feed/profil = stubs, reste de l'API à brancher via `apiFetch`. TODO : rôle réel (placeholder admin) |

### Features status
| Feature | Type | Status |
|---|---|---|
| Registration / Login | Primary | 🟢 De bout en bout : UI login/register → route handlers Next (BFF) → gateway → auth-service. **Provisioning user** : register → `POST /users {username}` (handle CHOISI persisté) ; login → `GET /users/me` (dérivé email). **Pré-vérification de disponibilité** du username avant l'inscription + validation alignée sur le back. Repli dérivé email si course (409). **Déconnexion + garde de session faites** (cf. ligne JWT). |
| JWT auth + protected routes | Primary | 🟢 Auth-service : access (15m) + refresh (24h, rotation, révocable) + `/auth/validate` + middleware (`code: token_expired`). Front : access en localStorage, refresh en cookie httpOnly (BFF), `apiFetch` rafraîchit sur 401 (single-flight), `middleware.ts` garde les routes `(app)`, logout révoque + efface. TODO : middleware JWT du gateway (protéger les préfixes en amont) |
| Role management (User/Mod/Admin) | Primary | 🟡 Rôle porté par le JWT (auth) ; user-service garde l'enregistrement public et applique un contrôle de rôle (`DELETE /users/:id` réservé admin). UI nav par rôle déjà faite. TODO : généraliser côté autres services |
| Post creation/reading | Primary | 🟡 UI faite (feed + composer 280 car. inline & popup sidebar), lecture/écriture API à brancher |
| User profile | Primary | 🟡 UI faite (consultation : bannière/avatar/bio/compteurs/onglets + édition popup nom/bio + photo/bannière via sélecteur de fichier avec aperçu local). **Back : profil-service v1** (routes lecture/édition, `birth_date` set-once, `display_name_changed_at`). **Création = POST uniquement** (GET ne crée rien, 404 si absent). Au register : BFF `POST /profils {display_name: username}`. Reste : brancher le front (lecture agrégée user+profil+post ; édition = `PATCH /profils/me`, **ou `POST` si 404**) + upload réel avatar/bannière |
| Social graph (follow/followers) | Secondary | 🟢 user-service : follow/unfollow + listes + compteurs. UI à brancher |
| Moderation (moderate posts) | Secondary | 🔴 TODO |
| Admin panel | Secondary | 🔴 TODO |
| *(add features here)* | | |

### Infrastructure
- [ ] docker-compose.yml with all services
- [ ] Persistent volumes for DBs
- [ ] .env for secrets (never commit)
- [ ] README with setup instructions

---

## 4. FILE STRUCTURE (target)

```
project/
├── CLAUDE.md              ← this file (keep up to date)
├── docker-compose.yml
├── .env.example
├── frontend/
│   └── ...
├── api-gateway/
│   └── ...
├── auth-service/
│   ├── Dockerfile
│   └── ...
├── user-service/
│   ├── Dockerfile
│   └── ...
├── post-service/
│   ├── Dockerfile
│   └── ...
└── profil-service/
    ├── Dockerfile
    └── ...
```

---

## 5. TECHNICAL DECISIONS

> Fill in as decisions are made.

| Topic | Decision | Reason |
|---|---|---|
| Gateway | *(e.g. Express / Kong / custom)* | |
| Frontend | Next.js 14 (App Router) + TS + Tailwind/shadcn | Stack imposée (§1bis) |
| Frontend routing | Route groups `(auth)` (public) et `(app)` (authentifié) | Sépare layouts publics/privés sans polluer l'URL |
| Frontend config | `lib/config.ts` (API Gateway) + `lib/routes.ts` (constantes de routes) | Source de vérité unique, pas de chaînes en dur |
| Auth front (access + refresh) | **Access token (15m) en localStorage** (le client appelle le gateway directement avec `Authorization: Bearer`) ; **refresh token (24h) en cookie httpOnly `breezy-refresh`** géré par le **BFF Next** (`/api/auth/{login,register,refresh,logout}`). Le refresh token transite BFF↔gateway dans le **corps JSON** (jamais exposé au JS). `lib/auth-client.ts` : `apiFetch` ajoute le Bearer, catche le 401, déclenche un **refresh single-flight** (une promesse partagée), rejoue la requête, et redirige /login si le refresh échoue. `logout` révoque + efface. `lib/server/auth-cookie.ts` centralise pose/effacement du cookie | Pattern « access court + refresh httpOnly » : compromis assumé (access en localStorage = exposé XSS mais court ; refresh httpOnly = non volable). **Cookie géré par le BFF (same-origin)** plutôt que par l'auth-service : évite toute la complexité CORS/SameSite d'un cookie cross-origin, cohérent avec le BFF existant. Single-flight = pas de tempête de refresh quand N requêtes prennent 401 en même temps |
| Refresh token (back) | Jeton **opaque** aléatoire (crypto/rand, 256 bits), **stocké haché** (SHA-256) dans `refresh_tokens` (jamais en clair). **Rotation** : `/auth/refresh` révoque l'ancien et émet une nouvelle paire. `/auth/logout` supprime le token (révocation). Le back **ne renouvelle jamais tout seul** : il signe avec `exp` et 401 sur expiration ; le front pilote le refresh. Middleware JWT ajoute `code: token_expired` pour distinguer expiration (→ refresh) de token invalide (→ pas de refresh) | Opaque + DB-stocké = **révocable** (≠ JWT self-contained) ; hash = une fuite DB ne livre aucun token. Rotation = un refresh token n'est rejouable qu'une fois (base pour la détection de réutilisation, en perspective). « Back ne prévoit pas l'expiration » = logique d'expiration portée par le front (cf. demande) |
| URL gateway client vs serveur | `apiUrl()` choisit la base selon le contexte : **client** = `NEXT_PUBLIC_API_URL` (`localhost:8080`, port publié) ; **serveur** (route handlers, dans le conteneur) = `API_INTERNAL_URL` (`http://api-gateway:8080`, réseau Docker). `API_INTERNAL_URL` non préfixée `NEXT_PUBLIC_` → invisible client → repli sur `API_URL` | Dans le conteneur frontend, `localhost` = le conteneur lui-même, pas la gateway → les fetch serveur doivent viser le nom de service Docker. En local sans Docker, laisser les deux sur `localhost:8080` |
| Provisioning user au login | Le BFF (`lib/provision.ts`) provisionne après l'auth, best-effort (n'échoue jamais l'auth). **Register** : `POST /users {username}` → persiste le handle CHOISI ; repli `GET /users/me` (dérivé email) si pris (409). **Login** : `GET /users/me` (l'utilisateur a déjà sa ligne). Dispo du username **pré-vérifiée** avant inscription (`/api/users/check-username` → `GET /users/by-username/:username`, 404=libre) + validation front alignée sur le back (`^[a-zA-Z0-9_]{3,50}$` + réservés) | Place le provisioning au point le plus robuste (serveur, à l'auth) : garanti, sans complexité client. Relie register→user sans coupler auth↔user. Le pré-check donne l'erreur sur le champ avant de créer le compte ; le repli couvre la course rarissime. Alternative écartée : provisioning au montage du feed (client) = moins garanti |
| Gateway : pas de redirection slash | `RedirectTrailingSlash=false` + chaque service exposé via **2 routes** : préfixe nu (`/users`) ET sous-chemin (`/users/*path`) | Sans ça, un `POST /users` (endpoint collection) déclenchait un 307 vers `/users/` que le service (route `/users`) ne connaît pas → boucle de redirections → `fetch` échoue. Bug latent (on ne passait que par des sous-chemins comme `/users/me`). Touche tous les endpoints collection (liste + création) |
| Frontend nav par rôle | `navItemsForRole()` filtre les liens (user/mod/admin) | Reflète les 3 rôles côté UI |
| Nom du produit | **Breezy** (logo `frontend/public/logo_breezy.png`) | WebDad = nom du projet/repo, Breezy = nom du réseau social |
| Layout feed | 3 colonnes style X.com : nav (gauche) / fil (centre) / suggestions (droite) | UX familière, démo lisible (critère « Interface » §2) |
| Composer post | Logique de saisie factorisée dans `PostComposer` ; `CreatePost` = version inline (tête du fil), `CreatePostDialog` = popup (bouton « Poster » sidebar, shadcn `Dialog`) | Limite 280 car. + validation en un seul endroit, pas de duplication entre inline et modale |
| Emoji picker | `EmojiPicker` (shadcn `Popover` + liste statique d'emojis), insertion à la position du curseur dans le composer | Pas de dépendance lourde ; fonctionne inline et dans la popup |
| Onglets du fil | `FeedView` (client) gère l'état « Pour toi » / « Abonnements » ; Abonnements = placeholder (état vide) en attendant l'API | Switch d'onglet réel côté UI ; la page reste un Server Component qui passe les données stub |
| Page profil | `ProfilView` (client) orchestre : `ProfilHeader` (bannière/avatar/bio/compteurs/rôle) + onglets Posts/Réponses/J'aime (réutilise `PostCard`) + `EditProfilDialog`. Type `ProfilDetails`/`ProfilEditableFields` dans `types`. Édition optimiste (état local mis à jour à l'enregistrement) ; `isOwner` distingue bouton « Éditer » vs « Suivre ». Page = Server Component avec données stub | Consultation + édition en un seul flux ; édition réutilise le pattern `Dialog` du composer ; même convention stub+TODO que le feed (`PATCH /profils/me` à brancher) |
| Édition photo/bannière | Front : composant `ImagePicker` (input file caché + aperçu en **data URL** via `FileReader`, overlay icône appareil photo) dans `EditProfilDialog` ; champs `avatarUrl`/`bannerUrl`. Back (TODO) : upload du fichier vers stockage objet/disque + persistance des URLs (`profil-service`) | L'aperçu est purement front et **ne survit pas au reload** (pas de stockage) — l'upload réel est une responsabilité backend, à brancher dans l'issue du service. Data URL choisie (pas d'`objectURL` à révoquer, transférable tel quel à l'API) |
| Responsive / mobile | Mobile-first, breakpoint pivot `lg` (1024). **< lg** (téléphones, iPad portrait) : `MobileHeader` (avatar→**tiroir latéral gauche** `Sheet` avec Fil/Profil/Modération/Admin filtrés par rôle, + Paramètres et déconnexion en bas + logo Breezy centré, masqué sur /profil qui a son propre en-tête) + `MobileTabBar` fixe en bas (Accueil=`logo_only.png`, Recherche, Notifications, Messages) + `ComposeFab` (« + », bas-**droite**, convention X). **≥ lg** : sidebar gauche. **≥ xl** (1280) : + colonne droite. Sidebars en `hidden lg:flex`/`hidden xl:flex` ; `<main>` en `pb-16` pour dégager la barre. Sur feed mobile : titre + composer inline masqués (`hidden lg:block`). Favicon = `logo_only.png` (`metadata.icons`) | X.com-like sur 3 cibles (2 tél. + iPad) sans dupliquer les pages. Tiroir gauche (`Sheet` sur Radix Dialog) = navigation par section façon X mobile, déclenché par l'avatar. `MobileHeader` se masque seul (`usePathname`) sur les pages à en-tête propre. Pivot unique `lg`, robuste |
| Swipe d'ouverture du tiroir | `MobileHeader` rend le `Sheet` **contrôlé** (`open`/`setOpen`) + écoute `touchstart`/`touchend` sur `window` : un geste depuis le bord gauche (≤24px) vers la droite (>60px, surtout horizontal) ouvre le tiroir. Hooks appelés avant le `return null` (règles des hooks) ; listeners non attachés quand l'en-tête est masqué | Radix `Dialog`/`Sheet` n'a **aucun** geste de swipe natif → ajout manuel. ⚠️ Sur iOS Safari le swipe bord-gauche déclenche aussi le « retour » du navigateur (conflit connu, acceptable en démo). Actif uniquement là où `MobileHeader` est monté (pas sur /profil) |
| Anti-débordement mobile | Colonne centrale en `overflow-x-clip` (+ `min-w-0`) | Empêche le défilement horizontal parasite sur téléphone. `clip` (et non `hidden`) : ne crée pas de conteneur de scroll → ne casse pas les en-têtes `sticky` ; les éléments `fixed` (barre d'onglets, FAB) ne sont pas rognés (leur bloc conteneur = viewport) |
| Thème / couleurs | Thème clair par défaut : fond blanc, primaire magenta `#e053ff` (texte foncé pour lisibilité). 2 dimensions : mode clair/sombre (next-themes, `.dark`) + accent (`[data-accent]` : pink défaut / blue / cyan). Registre dans `lib/themes.ts` | Identité visuelle + dark theme et thèmes custom anticipés sans dupliquer la palette |
| Sélecteur de mode (clair/sombre/système) | Composant `ThemeToggle` (`useTheme()` next-themes, persistance auto) : **interrupteur façon iOS** clair/sombre (curseur qui glisse sur Soleil/Lune) + **ligne « Mode système »** (icône écran) affichant Activé/Désactivé. Quand système activé → l'interrupteur est **grisé/désactivé** et reflète `resolvedTheme` ; le désactiver fige l'apparence courante en choix manuel (`setTheme(resolvedTheme)`). Placé dans le **tiroir mobile** (`MobileHeader`). `enableSystem` activé dans le `ThemeProvider` (défaut reste clair). Flag `mounted` anti-mismatch d'hydratation. **Slide animé** du curseur via keyframes Tailwind `theme-thumb-left/right` (état `slide` mémorise le sens) | Bascule de mode demandée, UX type réglages iOS. next-themes gère application + persistance → composant purement UI. `isDark` lit `resolvedTheme` quand système actif pour positionner le curseur correctement. `mounted` requis car le serveur ignore le thème. ⚠️ Slide en **`animation`** (pas `transition`) car `disableTransitionOnChange` de next-themes désactive les `transition` au moment du switch — les keyframes y échappent |
| Inter-service auth | JWT passed in header | Grading requirement |
| API Gateway | Reverse proxy mince (`httputil.ReverseProxy`, stdlib) : table préfixe→URL (`internal/proxy` + `internal/router`), forwarde méthode/chemin/corps/headers et renvoie la réponse intacte (`{data}`/`{error}` remontent). CORS maison (`internal/middleware`, gère le preflight OPTIONS). Routes publiques pour l'instant ; le middleware JWT (validation locale avec `JWT_SECRET` partagé) viendra protéger les préfixes au login | Mince + évolutif + « on l'a construit » (démo). Forward transparent vs handlers par endpoint (BFF) réservés à l'agrégation multi-services. Préfixe conservé (`/auth/...` → service sur `/auth/...`) |
| Containerization | Docker + docker-compose | Grading requirement |
| Dev hot-reload | `make dev` = overlay `docker-compose.dev.yml` par-dessus la base. Go : stage `dev` du Dockerfile (`air` épinglé `air-verse/air@v1.52.3`) + `.air.toml` par service + bind-mount source ; caches Go partagés (volumes `go-mod-cache`/`go-build-cache`). Frontend : stage `deps` + `command: npm run dev` + bind-mount + volume anonyme `node_modules` + `WATCHPACK_POLLING=true`. Le `frontend` voit son `depends_on` effacé via `!reset`. `make up` reste les images de prod figées | Itérer sans rebuild. Stage `dev` placé AVANT le runtime → `make up`/`build` produisent toujours l'image de prod (dernier stage). `!reset` car un `depends_on: []` ne vide pas (compose fusionne les mappings). `start_period: 90s` sur les services Go pour laisser le 1er build `air` se faire |
| Config `.env` | Racine = vars transverses (`JWT_SECRET`, `JWT_EXPIRY`, `NEXT_PUBLIC_API_URL`) ; `<service>/.env` = config propre, chargée par compose via `env_file:` ; `environment:` réservé aux overrides Docker (host = nom de conteneur) | Découplage : un service tourne seul (`make run`) avec son `.env`, et en stack via compose. ⚠️ Les vars d'un `env_file` ne sont PAS interpolables (`${...}`) dans le compose — seul le `.env` racine l'est. DB host surchargé via `DB_HOST`/`MONGO_HOST` |
| Schéma DB (post) | Le service possède son schéma : `EnsureSchema` (post-service/internal/database/init.go) crée collections + validateurs `$jsonSchema` + index au démarrage, idempotent. Aucun script monté dans `mongo-post`. Config Mongo construite depuis le `.env` (`internal/config`), zéro creds en dur. `scripts/init-db/post-init.js` et le `post-service/docker-compose.yaml` parasite supprimés | Source de vérité unique + service autonome (`make run`/`make dev` contre un Mongo vierge). Même pattern que auth. ✅ user-service et profil-service sont désormais autonomes eux aussi (plus aucun init-db monté) |
| Schéma DB (auth) | Le service possède son schéma : `auth-service/internal/db/schema.sql` (embarqué `go:embed`), appliqué au boot de façon idempotente (`EnsureSchema`). Aucun script monté dans `postgres-auth`. Seed admin via `SEED_DEFAULT_ADMIN`+`SEED_ADMIN_PASSWORD` (idempotent, UUID figé). `scripts/init-db/auth-init.sql` supprimé | Source de vérité unique + service autonome (`make run` contre un Postgres nu). Même pattern que post. ✅ profil-service harmonisé (autonome) — plus aucun service sur init-db monté |
| Schéma DB (user) | Même pattern autonome : `user-service/internal/db/schema.sql` embarqué + `EnsureSchema` au boot (tables `users`+`follows`, index, trigger `updated_at`, seed admin UUID figé). Mount `scripts/init-db/user-init.sql` retiré du compose, fichier supprimé. `users.id` = `credentials.id` (auth) | Cohérent avec auth/post (TODO §6 d'harmonisation traitée pour user). Reste profil |
| Schéma DB (profil) | Même pattern autonome : `profil-service/internal/database/init.go` (`EnsureSchema`) crée la collection `profiles` + validateur `$jsonSchema` + index **unique** `user_id` + profil admin (upsert idempotent) au boot. `scripts/init-db/profil-init.js` supprimé + mount retiré du compose. **Dernier service harmonisé** → les 4 services possèdent désormais leur schéma | Source de vérité unique + autonome (`make run`/`make dev` contre un Mongo vierge). Clôt le TODO §6 d'harmonisation |
| Propriété des données profil ↔ user | **profil-service possède UNIQUEMENT les champs décoratifs/éditables** : `display_name` (déplacé depuis user-service), `bio`, `avatar_url`, `banner_url`, `website`, `location`, `birth_date` (date, opt.), `gender` (enum `male`/`female`, opt. — validateur Mongo + binding `oneof`). **user-service garde l'identité** : `username` (immuable, unique, handle), `is_active`, le graphe social (follows + compteurs). `role` reste porté par le JWT | Une seule source de vérité par donnée (le `profil-init.js` historique dupliquait username + compteurs → resync à éviter). Surtout : **tous les champs éditables d'un profil dans un seul service** → une modif de profil = **un seul `PATCH /profils/me`**, pas d'écriture multi-service ni de transaction inter-bases |
| Agrégation `ProfilDetails` (lecture) | L'écriture est atomique (1 service, cf. ci-dessus) ; seule la **lecture** de la vue agrégée (identité + décoratif + compteurs) croise 3 sources (user + profil + post). profil-service ne renvoie que SES données ; l'agrégation se fera côté **front (`apiFetch`)** ou **BFF Next** — **tranché au moment de brancher le front** (hors squelette, ne change pas les routes du service) | Découple le service de l'agrégation. Le squelette est identique quel que soit l'agrégateur choisi → décision repoussée sans dette |
| Provisioning user (lazy) | La table `users` n'est PAS remplie par auth au register (bases séparées + règle « tout passe par la gateway »). À la place : `POST /users` exposé (CRUD/admin/tests) **+** provisioning paresseux sur `GET /users/me` (upsert depuis les claims JWT au 1er accès authentifié). Username dérivé de l'email (modifiable via PATCH) | Zéro couplage auth↔user, service autonome, robuste en démo. Laisse aussi la porte à un flow « première connexion » distinct plus tard. Alternative écartée : auth appelle user au register (couplage + incohérence transactionnelle inter-bases) |
| Layering user-service | Couche `repository` (SQL pur) sous `service` (métier/erreurs) sous `handlers`, contrairement à auth (SQL inline dans le service) | Le CRUD users+follows a beaucoup plus de requêtes → couche repo dédiée justifiée (proche de post-service) |
| Follows (user-service) | Table d'arêtes `follows` en Postgres (PK composite + index 2 sens), follow/unfollow **idempotents** (`ON CONFLICT DO NOTHING` / `DELETE`), anti-self-follow (check code + contrainte `no_self_follow`), follower auto-provisionné avant l'arête. Compteurs **calculés** (COUNT) sur les vues détail uniquement (pas sur `List`, évite le N+1) | Le graphe social est un many-to-many relationnel → Postgres est le bon moteur (≠ Mongo, qui serait un anti-pattern par embarquement). Idempotence = appels réseau rejouables sans erreur. Dénormalisation des compteurs reportée (perspective scale) |
| Username (user-service) | Provisioning : username dérivé de l'email, **résolution de collision** par candidats successifs (`base` → `base_<8id>` → `user_<8id>`). User-supplied (Create/PATCH) : regex `^[a-zA-Z0-9_]{3,50}$` + liste de mots réservés (`me`, `admin`, `users`…) | Deux emails de même partie locale ne doivent pas planter le provisioning (bug corrigé) ; les handles structurants/sensibles sont protégés. Lecture par handle via `GET /users/by-username/:username` (route statique placée avant `/:id`) |
| Profil : POST = UNIQUE création, GET ne crée rien | `GET /profils/me` est en **lecture seule** : `404` si le profil n'existe pas (aucun provisioning paresseux). La **seule** création est `POST /profils` (`display_name` **obligatoire**), appelé par le BFF au register (`display_name = username`). `PATCH /profils/me` `404` si absent. Front : `GET` → si `404`, créer via `POST` (popup) ; sinon `PATCH`. Le profil admin est créé par le seed `EnsureSchema`. profil-service n'appelle JAMAIS user-service en direct | Demande utilisateur : ni écriture sur un GET, ni display_name supposé en base. Une création = un acte explicite (`POST`). En base, `display_name` est soit le username (register), soit ce que l'utilisateur a saisi — jamais deviné. Coût assumé : le front gère le `404` (POST-si-absent) ; le compte sans profil (POST register échoué, best-effort) s'auto-soigne à la 1re édition. Une donnée = un service : profil ne connaît pas le username (user-service) et ne se couple pas à lui au runtime ; le BFF aligne `display_name = username` au register (`lib/provision.ts`) |
| Profil : `birth_date` set-once | Modifiable via `PATCH /profils/me` tant que le champ est vide ; une fois posée, rejouer la même valeur est toléré (no-op) mais toute valeur différente → `409 ErrBirthDateLocked`. Logique dans `planUpdate` (fonction PURE, testée) | « On ne peut pas modifier la date de naissance » (demande) tout en laissant l'utilisateur la renseigner après coup s'il l'a sautée. Set-once > « jamais via PATCH » (plus souple, même garantie d'immuabilité une fois posée) |
| Cooldown de changement (display_name / username) | **Architecture posée maintenant, enforcement désactivé.** Chaque champ « identitaire » porte une date de dernier changement : `profiles.display_name_changed_at` (Mongo) et `users.username_changed_at` (Postgres, nullable). Elle est **posée uniquement quand la valeur change réellement** (comparaison à l'ancienne valeur — en SQL côté user via un `CASE`, dans `planUpdate` côté profil). Le **refus** (`429`) est codé derrière un délai configurable par env (`DISPLAY_NAME_CHANGE_COOLDOWN` / `USERNAME_CHANGE_COOLDOWN`, format durée Go), **défaut 0 = désactivé**. Activer la règle = poser l'env, **zéro code, zéro migration** | On capture la baseline **dès aujourd'hui** : ajouter le champ plus tard laisserait les profils existants sans date de référence (le cooldown serait contournable au 1er changement). Le timestamp est gratuit ; seul le refus est piloté par config → on tient « prépare l'archi, je l'activerai peut-être » sans dette. Symétrie stricte entre les deux services (mêmes noms, même sémantique) |

---

## 6. KNOWN ISSUES / BLOCKERS

> Add/remove as issues arise.

- **Rappel : `make up` = images de prod FIGÉES.** L'image frontend embarque un `npm run build`
  (`output: standalone` → `node server.js`) et les images Go un binaire statique, tous figés au
  build. `docker compose up` ne rebuild PAS sur changement de source → un changement de code
  n'apparaît qu'après `make build`. **Pour développer avec hot-reload, utiliser `make dev`** (voir §5).

- **À FAIRE — rôle / username réels (frontend).** La déconnexion et la garde de session sont
  faites (logout révoque + `middleware.ts` redirige vers `/login` sans cookie refresh). Reste : le
  layout `(app)` utilise encore `PLACEHOLDER_ROLE = 'administrator'`. À faire : déduire `role`/
  `username` réels (via `apiFetch('/users/me')` côté client, ou décodage du JWT) pour la nav par rôle.

- **NOTE — access token en localStorage (compromis XSS).** Choix assumé du pattern « access court +
  refresh httpOnly » : l'access token (15m) est en localStorage (lisible par le JS → exposé en cas
  de XSS), le refresh (24h) reste en cookie httpOnly (non volable). Mitigation : access très court.
  Perspective : détection de réutilisation de refresh token (la rotation est déjà en place).

- **NOTE — CORS gateway pour `apiFetch`.** Les appels data partiront du navigateur DIRECTEMENT vers
  le gateway (`Authorization: Bearer`, cross-origin). La CORS du gateway autorise déjà `Authorization`
  + l'origine front ; vérifier `CORS_ALLOWED_ORIGINS` quand on branchera le feed/profil réels.

- **✅ RÉSOLU — harmonisation des schémas.** Les **4 services** (auth/post/user/profil) possèdent
  désormais leur schéma (embarqué + `EnsureSchema` au boot, cf. §5). Plus aucun script `init-db`
  monté ; `scripts/init-db/` est vide. `profil-init.js` supprimé.

- **✅ RÉSOLU — profil-service implémenté (repository/service).** Lecture (`GET /profils/:userId`,
  `GET /profils/me` — **lecture seule, 404 si absent**), édition (`PATCH /profils/me`, `birth_date`
  set-once), **création = `POST /profils` uniquement** (display_name obligatoire), `DELETE` admin.
  Reste (front) : **gérer le 404 (POST-si-absent)**, brancher l'agrégation lecture user+profil+post
  (cf. décision §5) et l'upload réel avatar/bannière.

- **À FAIRE — activer/calibrer les cooldowns identitaires (optionnel).** L'archi est posée
  (`display_name_changed_at` / `username_changed_at` enregistrés, refus codé). Pour activer :
  poser `DISPLAY_NAME_CHANGE_COOLDOWN` / `USERNAME_CHANGE_COOLDOWN` (ex. `168h`). Aucune migration.
  Perspective : exposer la prochaine date autorisée au front (le champ `*_changed_at` est déjà
  renvoyé dans les réponses).

- **À FAIRE — tests d'intégration repo (user-service).** Les tests Go actuels couvrent le
  middleware JWT et la validation (sans DB). Les requêtes SQL (CRUD + follows) sont validées par
  un e2e manuel contre un Postgres jetable, mais pas par des tests automatisés. À ajouter (ex.
  `dockertest` ou `testcontainers`) si on veut une couverture repo en CI.

- **NOTE — compteurs followers/following calculés (COUNT).** `user-service` calcule les compteurs
  par sous-requête à chaque lecture de profil. OK à l'échelle du projet ; à dénormaliser
  (colonnes `follower_count`/`following_count` + triggers/incréments) si la charge l'exige
  (cf. perspective §2 *Improvements & outlook*).

---

## 7. INSTRUCTIONS FOR CLAUDE CODE

1. **Read this file first** at every session start.
2. **⛔ NE PAS CODER avant validation de l'utilisateur.** Pour toute feature/tâche :
   d'abord proposer (a) l'**architecture** et (b) **comment la feature sera implémentée**,
   et **attendre l'accord explicite** avant d'écrire/modifier du code. Pas d'implémentation
   spontanée.
3. **Update sections 3, 5, 6** after any significant change.
4. **Never hardcode secrets** — use `.env` variables.
5. **Each service is independent**: its own Dockerfile, its own DB.
6. When implementing a feature, **update its status** in section 3 (🔴→🟡→🟢).
7. Before any architectural decision, **check section 2** (evaluation criteria).
8. Keep responses concise — update this file rather than re-explaining context.

---

*Last updated: 04/06/2026 — feat(profil-service) : routes lecture/édition implémentées (`birth_date` set-once, `PATCH /profils/me`). **Création = `POST /profils` UNIQUEMENT** (`display_name` obligatoire = username au register via le BFF) ; **`GET /profils/me` est en lecture seule, ne crée RIEN** (404 si absent, le front crée via POST) — pas de provisioning paresseux, jamais de display_name deviné. `DELETE` admin. **Archi cooldown identitaire posée des deux côtés** : `display_name_changed_at` (profil) + `username_changed_at` (user-service, colonne + `CASE` SQL), enregistrement systématique, refus configurable (`*_CHANGE_COOLDOWN`, défaut 0 = off). BFF provisionne aussi le profil au login/register (`lib/provision.ts`). Tests Go (routes/401 + unit pur `planUpdate`). build+vet+gofmt+tests OK (profil + user). (précédemment : feat(profil-service) squelette + migration BDD).*
<!-- prev: 03/06/2026 — feat(profil-service) : squelette + migration BDD. Service complet calqué sur post (config/database/models/repository/service/handler/middleware), schéma autonome (`EnsureSchema` : collection `profiles` + validateur + index unique `user_id` + seed admin), `profil-init.js` supprimé + mount compose retiré (les 4 services sont désormais autonomes). Routes câblées mais stubs `501` sauf `/health` (`GET /profils/:userId`, `GET/PATCH /profils/me`, `POST/DELETE /profils`). **display_name déplacé de user-service vers profil-service** (profil = propriétaire de tout l'éditable → édition = 1 seul PATCH). Tests Go (routes/health/501/401). build + vet + gofmt + tests OK. (précédemment : feat(auth) refresh tokens) -->

