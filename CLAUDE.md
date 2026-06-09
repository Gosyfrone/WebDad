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
- Langage : **Go 1.25** (bumpé depuis 1.22, EOL : 1.22 ne recevait plus les patchs sécu stdlib → CVE govulncheck. Dockerfiles `golang:1.25-alpine`, `go 1.25.0` dans les 5 `go.mod`, CI `setup-go: 1.25`)
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
| message-service      | 8085 |
| notification-service | 8086 |

Plus aucun conflit : le frontend (3000) et le backend (8080+) occupent des plages distinctes.

**Bases de données**
| Service              | Base       |
| -------------------- | ---------- |
| auth-service         | PostgreSQL |
| user-service         | PostgreSQL |
| profil-service       | MongoDB    |
| post-service         | MongoDB    |
| message-service      | MongoDB    |
| notification-service | MongoDB    |

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
| Auth Service | 🟡 WIP | PostgreSQL | Routes `/auth/{register,login,refresh,logout,validate}` + `/health`. JWT HS256 (claims user_id/email/role) + bcrypt. Access 15m + refresh 24h opaque haché SHA-256, rotation à `/refresh`, révoqué à `/logout` (cf. §5 « Refresh token »). Autonome (schéma embarqué au boot, seed admin opt.). TODO : tests Go, détection de réutilisation de refresh. |
| User Service | 🟢 OK (v1) | PostgreSQL | Couches repo/service/handlers, autonome (schéma `users`+`follows` au boot). CRUD users (`GET /users`, `/users/:id`, `/by-username/:username`, `POST /users`, `GET/PATCH /users/me`, `DELETE /users/:id` admin). Follows idempotents (`POST/DELETE /users/:id/follow`, `GET .../followers|following`). Recherche (`/users/search`, `/users/suggestions`). Provisioning paresseux sur `/users/me`. Possède `username`+`is_active`+graphe social (`display_name` → profil-service). `username_changed_at` = baseline cooldown (cf. §5). Tests Go + e2e. TODO : tests intégration repo. |
| Post Service | 🟢 OK (v1) | MongoDB | Autonome (collections+validateurs+index au boot). CRUD posts (`POST/GET/PATCH/DELETE /posts`, filtres `?author_id=`/`?author_ids=`, `quote_post_id`). Reposts/citations, épinglage profil (`pinned_at`, `PATCH/DELETE /posts/:id/pin`), likes (`POST/DELETE /posts/:id/like`, `/me/liked-ids`), commentaires threadés 2 niveaux (`/posts/:id/comments[/:id/replies]`). Compteurs dénormalisés int32 (`$inc`), nettoyage orphelins. Émet vers notif-service (cf. §5). **Signets en collections** : collections `bookmark_collections`/`bookmarks` (many-to-many)/`bookmark_prefs` ; collection par défaut `is_default` (non supprimable/renommable, en tête) ; routes JWT `/posts/bookmarks/collections[/:cid[/posts]]`, `/posts/:id/bookmark`, `/posts/:id/bookmark/collections`, vue « Tous » dédupliquée (`/posts/bookmarks`), `/posts/me/bookmarked-ids` ; **modèle de rafale** `BOOKMARK_SESSION_WINDOW` (clic court → `filed`/`needs_choice`) (cf. §5). Tests Go (`withinSessionWindow` + routes 401) + e2e signets 24/24. TODO : édition post côté front (back prêt). |
| Profil Service | 🟢 OK (v1) | MongoDB | Autonome (collection `profiles`, validateur resync `collMod` au boot). Possède le décoratif (`display_name`/`bio`/`avatar_url`/`banner_url`/`website`/`location`/`birth_date`/`gender`). Routes : `GET /profils/:userId`, `GET /profils/me` (lecture seule, 404 si absent), `PATCH /profils/me`, `POST /profils` (UNIQUE création, display_name requis), `DELETE` admin, `GET /profils/search`. `birth_date` set-once, `display_name_changed_at` cooldown (cf. §5). Tests Go OK. TODO : agrégation lecture front, upload avatar/bannière. |
| Message Service | 🟢 OK (DM+groupes+communautés) | MongoDB | Messagerie E2EE (cf. §5 « Messagerie »). Serveur aveugle pour DM/groupes (`ciphertext`+`nonce`+enveloppes). Clés X25519 (`/messages/keys`), conversations/messages paginés par curseur, WebSocket `/messages/ws`. Groupes (nom chiffré, cap 32, owner-only manage), communautés hybrides (nom clair, clé serveur, annuaire public `/messages/communities`, talker/viewer). Épinglage/suppression côté user (`members.pinned_at/cleared_at`). **Curseur de lecture serveur (`members.last_read_at`) : `PUT /conversations/:id/read` + `GET /messages/unread-count` (badge, cf. §5), calcul par métadonnées (jamais le `ciphertext`).** Tests Go + e2e (Phase 1 13/13, 2 21/21, 3 28/28). Back COMPLET + UX. |
| Notification Service | 🟢 OK (temps réel, agrégé) | MongoDB | Notifications agrégées façon Instagram (`group_key` unique/destinataire, `$inc count`). Ingestion `POST /internal/events` (secret `X-Internal-Secret`, hors gateway) : like/comment/reply/mention/repost/quote/post_deleted, `retract` défait. API `/notifications` (JWT : list curseur, unread-count, read) + WS `/notifications/ws`. Règles par type + modèle d'agrégation (cf. §5). Tests Go + e2e live OK. |
| API Gateway | 🟡 WIP | — | Reverse proxy stdlib : préfixes `/auth`,`/users`,`/profils`,`/posts`,`/messages`,`/notifications` → service cible (.env). WebSocket proxifié nativement (101 vérifié). CORS (`CORS_ALLOWED_ORIGINS`). ⚠️ `/internal/events` non routé (serveur-à-serveur). TODO : middleware JWT pour protéger les préfixes. |
| Frontend | 🟡 WIP | — | Next.js 14 + layout responsive style X. Auth refresh tokens (access localStorage, refresh cookie httpOnly via BFF `/api/auth/*`, `apiFetch` refresh single-flight, garde `middleware.ts`). Clients métier `lib/{api,posts,bookmarks,messages,notifications}.ts` au-dessus d'`apiFetch`. Câblé : feed/posts (like/commentaires/pin/repost/citation), **signets en collections** (bouton `PostCard` + `BookmarkDialog` + page `/signets`), profil hydraté, relations (suivre), Explorer + « Qui suivre », messagerie E2EE (DM/groupes/communautés), notifications (badge+WS), traduction auto, i18n FR/EN maison, dark mode tokenisé, pages légales `(legal)`, **mots filtrés par compte dans le feed** (`breezy-muted-words:<user_id>`, propres posts visibles). TODO : rôle réel (placeholder admin), `/profil/[username]`, édition post, upload images. |

### Features status
| Feature | Type | Status |
|---|---|---|
| Registration / Login | Primary | 🟢 De bout en bout : UI → BFF → gateway → auth-service. Provisioning user (register `POST /users`, login `GET /users/me`), pré-vérification username. Déconnexion + garde de session faites. |
| JWT auth + protected routes | Primary | 🟢 access 15m + refresh 24h (rotation, révocable) + `/auth/validate`. Front : refresh single-flight sur 401, garde `(app)`. TODO : middleware JWT gateway. |
| Role management (User/Mod/Admin) | Primary | 🟡 Rôle dans le JWT ; user-service applique le contrôle admin (`DELETE /users/:id`). UI nav par rôle faite. TODO : généraliser aux autres services, rôle réel front (placeholder admin). |
| Post creation/reading | Primary | 🟢 De bout en bout (cf. Post Service + §5). Feed défilement infini (Pour toi / Abonnements), like optimiste animé, commentaires threadés 2 niveaux, reposts/citations, épinglage, emoji. `lib/posts.ts` cache auteur. TODO : édition post, upload images. |
| User profile | Primary | 🟡 UI consultation+édition faite. Back profil-service v1 (cf. ligne service). TODO : brancher lecture agrégée user+profil+post, édition `PATCH /profils/me` (POST si 404), upload réel avatar/bannière. |
| Social graph (follow/followers) | Secondary | 🟢 user-service follow + listes + compteurs. Front câblé (`lib/api.ts`, `RelationsDialog`, bouton Suivre⇄Abonné optimiste, hook `use-follow`). Testé e2e. TODO : `/profil/[username]`, resync compteurs header après (dé)suivi. |
| Recherche / Explorer (comptes) | Secondary | 🟢 `GET /users/search` + `/profils/search`. Front `/explorer` debouncé (`@`→username sinon display_name), « Qui suivre » via `/users/suggestions`. Testé e2e. TODO : recherche de posts. |
| Traduction automatique des posts | Secondary | 🟢 Route BFF `/api/translate` (LibreTranslate + fallback Google, timeouts), `TranslatedContent` (posts+commentaires), cache, bascule original/traduction. |
| Mots filtrés dans le feed | Secondary | 🟢 `/parametres` : bloc sous la langue, ajout/suppression par chips, ascenseur après ~3 lignes. Persistance locale **par compte** (`breezy-muted-words:<user_id>`, `getMe()` si token pas encore rechargé). `FeedView` masque les posts des autres contenant l'expression filtrée (hashtag inclus), mais garde visibles les propres posts de l'utilisateur. |
| Messagerie privée chiffrée (E2EE) | Secondary | 🟢 De bout en bout (back + UX). DM/groupes admin-proof, communautés hybrides (clé serveur). UI `/messages` deux volets (cf. §5). **État lu/non-lu côté serveur** (`members.last_read_at`, multi-appareil) : pastille liste + ancre « Nouveaux messages » + **badge non-lu app-wide** (`MessagesProvider`, cf. §5). **Sourdine par conversation** (`members.muted_at`) : exclue du badge, reste non-lue dans la liste. Aperçu dernier message, épinglage/suppression côté user. Clé d'identité par appareil (IndexedDB). TODO : pièces jointes. |
| Notifications temps réel | Secondary | 🟢 De bout en bout (testé e2e live). Agrégation Instagram, types like/comment/reply/mention/repost/quote, WebSocket. Front `NotificationsProvider` (badge app-wide + WS unique), page `/notifications` + détail `/posts/[id]`. TODO : notifs de follow, rôle réel. |
| Signets / Bookmarks (collections) | Secondary | 🟢 De bout en bout (back + front, cf. Post Service + décision §5). Posts enregistrés dans des **collections** (many-to-many) + **collection par défaut** non supprimable. Front : `lib/bookmarks.ts`, bouton signet sur `PostCard` (clic court `filed`/`needs_choice`, appui long → sélecteur, dé-signer), `BookmarkDialog`, page `/signets` (`BookmarksView`), nav `nav.bookmarks`, i18n FR/EN. Vérifié : tsc/lint/vitest 31/31 + back e2e 24/24. |
| Moderation (moderate posts) | Secondary | 🔴 TODO |
| Admin panel | Secondary | 🔴 TODO |
| Internationalisation (FR/EN) | Secondary | 🟢 i18n maison zéro dépendance (`lib/i18n.ts` registre + `LanguageProvider` + `useT()` + `LanguageSelector` globe). Tous les textes UI traduits, dates localisées. Défaut FR. Ajout langue = 1 entrée `LOCALES` + 1 bloc `messages`. TODO : persistance côté compte. |
| *(add features here)* | | |

### Infrastructure
- [x] docker-compose.yml with all services
- [x] Persistent volumes for DBs
- [x] .env for secrets (never commit)
- [x] README with setup instructions
- [x] CI/CD : 3 workflows GitHub Actions (`ci-go` build+test -race **+ golangci-lint v2.12.2 / action v9 (bloquant, config par défaut, vert sur les 6 modules dont notification-service) + govulncheck (report-only)**, `ci-frontend` lint+build, `ci-integration` stack docker + healthchecks BDD, **incluant `mongo-notification`+`notification-service`**). Cf. décision §5. **govulncheck = 0 vuln** depuis le bump Go 1.25 + `x/net@v0.55.0` + `golang-jwt/jwt/v5@v5.3.1` (tooles en CI/Docker résolvent vers le dernier patch 1.25.x). TODO (perspective) : gitleaks, Dependabot, CD (push images GHCR)

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
| Frontend routing | Route groups `(auth)` (public), `(app)` (authentifié, gardé par `middleware.ts`) et `(legal)` (public : mentions légales / CGU / confidentialité) | Sépare layouts publics/privés sans polluer l'URL. `(legal)` distinct de `(app)` pour rester **hors garde de session** (le `matcher` du middleware ne couvre pas ces chemins → accessibles déconnecté depuis login/register) |
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
| i18n (FR/EN, maison) | **Mécanisme maison léger, zéro dépendance** (pas de `next-intl`/`react-i18next`). `lib/i18n.ts` = registre `LOCALES` (source de vérité du sélecteur, scalable) + `messages: Record<Locale, Record<string,string>>` (clés `namespace.key`, **FR référence**, repli en cascade locale→FR→clé brute, interpolation `{param}`). `LanguageProvider` (Context client, calqué sur next-themes) lit `localStorage breezy-locale` au montage et pose `<html lang>` ; hook `useT()` (+`useLanguage()` pour `locale`). **Hydratation** : on rend TOUJOURS `DEFAULT_LOCALE` (FR) au 1er rendu serveur+client (pas de mismatch), puis bascule en effet → bref flash assumé (même compromis que le thème). Sélecteur **dropdown globe** (pas un toggle binaire) pour accueillir ES/IT/ZH plus tard. `routes.ts`/nav arrays stockent une `labelKey` (pas le libellé). Dates via `timeAgo(iso, locale)` + `Intl.DateTimeFormat('en-US'|'fr-FR')`. Pages stub = Server Components qui passent l'**icône en JSX** (`ReactNode`) au Client `PlaceholderPage` (on ne peut pas passer une *fonction* composant à travers la frontière RSC). | Cohérent avec le style « on l'a construit » + registre `themes.ts`. 2 langues → un dico maison suffit, `next-intl` (routing par locale, refonte) serait disproportionné. Demande utilisateur : interrupteur « comme le thème » MAIS **scalable** → dropdown piloté par registre, ajout d'une langue = 1 entrée `LOCALES` + 1 bloc `messages`. Placement « juste au-dessus du clair/sombre » + variante flottante translucide sur les pages publiques (tranché avec l'utilisateur, décideur) |
| Onglets du fil | `FeedView` (client) gère l'état « Pour toi » / « Abonnements » ; Abonnements = placeholder (état vide) en attendant l'API | Switch d'onglet réel côté UI ; la page reste un Server Component qui passe les données stub |
| Page profil | `ProfilView` (client) orchestre : `ProfilHeader` (bannière/avatar/bio/compteurs/rôle) + onglets Posts/Réponses/J'aime (réutilise `PostCard`) + `EditProfilDialog`. Type `ProfilDetails`/`ProfilEditableFields` dans `types`. Édition optimiste (état local mis à jour à l'enregistrement) ; `isOwner` distingue bouton « Éditer » vs « Suivre ». Page = Server Component avec données stub | Consultation + édition en un seul flux ; édition réutilise le pattern `Dialog` du composer ; même convention stub+TODO que le feed (`PATCH /profils/me` à brancher) |
| Édition photo/bannière | Front : composant `ImagePicker` (input file caché + aperçu en **data URL** via `FileReader`, overlay icône appareil photo) dans `EditProfilDialog` ; champs `avatarUrl`/`bannerUrl`. Back (TODO) : upload du fichier vers stockage objet/disque + persistance des URLs (`profil-service`) | L'aperçu est purement front et **ne survit pas au reload** (pas de stockage) — l'upload réel est une responsabilité backend, à brancher dans l'issue du service. Data URL choisie (pas d'`objectURL` à révoquer, transférable tel quel à l'API) |
| Responsive / mobile | Mobile-first, breakpoint pivot `lg` (1024). **< lg** (téléphones, iPad portrait) : `MobileHeader` (avatar→**tiroir latéral gauche** `Sheet` avec Fil/Profil/Modération/Admin filtrés par rôle, + Paramètres et déconnexion en bas + logo Breezy centré, masqué sur /profil qui a son propre en-tête) + `MobileTabBar` fixe en bas (Accueil=`logo_only.png`, Recherche, Notifications, Messages) + `ComposeFab` (« + », bas-**droite**, convention X). **≥ lg** : sidebar gauche. **≥ xl** (1280) : + colonne droite. Sidebars en `hidden lg:flex`/`hidden xl:flex` ; `<main>` en `pb-16` pour dégager la barre. Sur feed mobile : titre + composer inline masqués (`hidden lg:block`). Favicon = `logo_only.png` (`metadata.icons`) | X.com-like sur 3 cibles (2 tél. + iPad) sans dupliquer les pages. Tiroir gauche (`Sheet` sur Radix Dialog) = navigation par section façon X mobile, déclenché par l'avatar. `MobileHeader` se masque seul (`usePathname`) sur les pages à en-tête propre. Pivot unique `lg`, robuste |
| Swipe d'ouverture du tiroir | `MobileHeader` rend le `Sheet` **contrôlé** (`open`/`setOpen`) + écoute `touchstart`/`touchend` sur `window` : un geste depuis le bord gauche (≤24px) vers la droite (>60px, surtout horizontal) ouvre le tiroir. Hooks appelés avant le `return null` (règles des hooks) ; listeners non attachés quand l'en-tête est masqué | Radix `Dialog`/`Sheet` n'a **aucun** geste de swipe natif → ajout manuel. ⚠️ Sur iOS Safari le swipe bord-gauche déclenche aussi le « retour » du navigateur (conflit connu, acceptable en démo). Actif uniquement là où `MobileHeader` est monté (pas sur /profil) |
| Anti-débordement mobile | Colonne centrale en `overflow-x-clip` (+ `min-w-0`) | Empêche le défilement horizontal parasite sur téléphone. `clip` (et non `hidden`) : ne crée pas de conteneur de scroll → ne casse pas les en-têtes `sticky` ; les éléments `fixed` (barre d'onglets, FAB) ne sont pas rognés (leur bloc conteneur = viewport) |
| Thème / couleurs | 2 dimensions : mode clair/sombre (next-themes, `.dark`) + accent (`[data-accent]` : pink défaut / blue / cyan, registre `lib/themes.ts`). **Direction graphique (refonte #79)** : glassmorphism, dégradé de marque **violet `#8D3DFF` → indigo `#5B6CFF` → cyan `#47D9FF`** (calé sur le logo "B") + fond de page dégradé + cartes « verre ». **Dark mode (refonte #81)** : décliné dans la **même direction**, fond **violet quasi-noir** (`#0b0712→#1a1033`), surfaces violet sombre, accents identiques (le dégradé du logo claque sur fond sombre). | Identité visuelle alignée sur le logo ; dark mode cohérent avec le clair sans toucher au clair |
| Surfaces tokenisées (clair+sombre) | La refonte #79 codait les couleurs **en dur** partout (≈230 occurrences : `bg-white/72`, `from-[#F8F3FF]`, `text-slate-*`, fond inline) → invisibles au `.dark`. #81 a **tokenisé** : variables CSS dans `globals.css` (`--bg-page`, `--panel`, `--glass`, `--column`, `--ink-strong`) avec **valeurs light = état exact** + déclinaison dark, consommées par des classes `@layer components` (`.bg-page`/`.bg-page-glow-1/2`, `.panel`/`.panel-y`, `.glass`/`.glass-strong`, `.glass-column`, `.brand-text`). Textes `text-slate-*` → tokens sémantiques (`text-foreground`/`text-muted-foreground`). Long tail (couleurs uniques) → variantes `dark:` ponctuelles. Accents de marque (boutons dégradés, `.brand-text`) inchangés dans les 2 modes | Une seule source de vérité par surface → le dark vit à un seul endroit, le clair reste identique au pixel. ⚠️ Priorité CSS : un utilitaire `bg-*`/`border-*` écrase ces classes (`@layer components`) → retirer l'utilitaire conflictuel en gardant la largeur de bordure (`border`, `border-b`) |
| Sélecteur de mode (clair/sombre/système) | Composant `ThemeToggle` (`useTheme()` next-themes, persistance auto) : **interrupteur façon iOS** clair/sombre (curseur qui glisse sur Soleil/Lune) + **ligne « Mode système »** (icône écran) affichant Activé/Désactivé. Quand système activé → l'interrupteur est **grisé/désactivé** et reflète `resolvedTheme` ; le désactiver fige l'apparence courante en choix manuel (`setTheme(resolvedTheme)`). Monté à **2 endroits** (même composant, pas de duplication) : **tiroir mobile** (`MobileHeader`, < lg) ET **sidebar gauche PC** (`SidebarLeft`, ≥ lg, sur un panneau `.panel` **au-dessus de la card user**, les deux dans un conteneur de bas pour garder le `justify-between` de l'`<aside>`). **Variante compacte flottante `FloatingThemeToggle`** (slider Soleil/Lune seul, sans option système, **translucide** `bg-white/20`+`backdrop-blur`, **fixe en bas à gauche**) sur les **pages publiques login/register** (montée une fois dans le layout `(auth)`, qui n'a pas de menu) — laisse voir le dégradé de fond ; réutilise les mêmes keyframes de slide. ⚠️ **Au chargement complet** (arrivée sur /login après logout = `window.location.assign`, page statique), `FloatingThemeToggle` **ne se rend qu'après montage** (`if (!mounted) return null`) et lit le thème **effectif** (`resolvedTheme`, **repli sur la classe `.dark` du `<html>`** posée par next-themes avant le paint) → corrige le bug « curseur figé côté Soleil alors qu'on est en sombre » (l'état `theme` n'était pas encore propagé au 1er rendu). `enableSystem` activé dans le `ThemeProvider` (défaut reste clair). Flag `mounted` anti-mismatch d'hydratation. **Slide animé** du curseur via keyframes Tailwind `theme-thumb-left/right` (état `slide` mémorise le sens) | Bascule de mode demandée, UX type réglages iOS. next-themes gère application + persistance → composant purement UI. `isDark` lit `resolvedTheme` quand système actif pour positionner le curseur correctement. `mounted` requis car le serveur ignore le thème. ⚠️ Slide en **`animation`** (pas `transition`) car `disableTransitionOnChange` de next-themes désactive les `transition` au moment du switch — les keyframes y échappent |
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
| CI/CD (GitHub Actions) | **3 workflows séparés par domaine** dans `.github/workflows/` : `ci-go.yml` (matrice 1 job/service sur les 5 modules Go → gofmt + `go vet` + `go build` + `go test -race` + check `go mod tidy`), `ci-frontend.yml` (`npm ci` + `npm run lint` + `npm run build`/typecheck), `ci-integration.yml` (génère les `.env` depuis les `.example`, `docker compose up -d --build` de **tout sauf le frontend** = 2 Postgres + 2 Mongo + 5 services Go, puis **attend que tous les conteneurs soient `healthy`** via les healthchecks existants `pg_isready`/`mongosh ping`/`wget /health`, timeout 240s, logs+teardown). Tous : `concurrency` (annule les runs obsolètes) + cache `setup-go`/`setup-node`. Déclencheurs : `push main` + `pull_request` avec **path-filters** par domaine | Fichiers séparés = déclencheurs/path-filters indépendants. Matrice Go = parallélisme + isolation par module (chacun son `go.mod`). `-race` gratuit et attrape les data races. Le smoke test docker matérialise « santé BDD » + « Docker complet » (§2) sans monter le frontend (lent, déjà couvert par `ci-frontend`). `.env` gitignoré → régénéré depuis les `.example` en CI |
| Cooldown de changement (display_name / username) | **Architecture posée maintenant, enforcement désactivé.** Chaque champ « identitaire » porte une date de dernier changement : `profiles.display_name_changed_at` (Mongo) et `users.username_changed_at` (Postgres, nullable). Elle est **posée uniquement quand la valeur change réellement** (comparaison à l'ancienne valeur — en SQL côté user via un `CASE`, dans `planUpdate` côté profil). Le **refus** (`429`) est codé derrière un délai configurable par env (`DISPLAY_NAME_CHANGE_COOLDOWN` / `USERNAME_CHANGE_COOLDOWN`, format durée Go), **défaut 0 = désactivé**. Activer la règle = poser l'env, **zéro code, zéro migration** | On capture la baseline **dès aujourd'hui** : ajouter le champ plus tard laisserait les profils existants sans date de référence (le cooldown serait contournable au 1er changement). Le timestamp est gratuit ; seul le refus est piloté par config → on tient « prépare l'archi, je l'activerai peut-être » sans dette. Symétrie stricte entre les deux services (mêmes noms, même sémantique) |
| Front abonnés/abonnements (refonte post-#79) | **Re-codé sur develop** (l'ancienne branche #68 bâtie sur l'ancien UI + ancien user-service `display_name` était périmée → repartie de zéro). Client métier `lib/api.ts` **mince au-dessus de `apiFetch`** (pas de gestion token maison : Bearer + refresh single-flight hérités). Modale `RelationsDialog` (shadcn `Dialog`, onglets internes maison comme `profil-view`, pas de `tabs.tsx`), ligne `UserListItem` réutilisable. Données = `[]User` (username seul) **enrichies** par `getProfil(userId)` (N+1 best-effort). État du bouton = `getFollowingIds(moi)` (pas de flag `is_following` côté API). `/profil` hydraté **côté client** dans `ProfilView` (`apiFetch` est client-only → impossible dans le Server Component `page.tsx`) | Reprend les patterns existants (Dialog d'`edit-profil`, onglets de `profil-view`, glassmorphism). `apiFetch` centralise déjà session/refresh → le client API ne fait que typer + déplier `{data}` + mapper snake→camel. Enrichissement N+1 assumé (perspective : endpoint agrégé ou dénormalisation). Hydratation client (et non serveur) car le token vit en localStorage, pas dans un cookie lisible côté serveur |
| Fil « Abonnements » (cross-service) | Le post-service **ne connaît pas le graphe social** (follows = user-service). Plutôt que de coupler post→user au runtime, le **front fournit les ids suivis** (`getFollowingIds`, seul user-service les connaît) au post-service via `GET /posts?author_ids=a,b,c`, qui fait **la sélection côté DB** (`{author_id:{$in:[...]}}`, index `author_id`). Le front **ne filtre rien** : il fournit l'ensemble, le service requête. | Même pattern que la recherche cross-service (orchestration front, requête efficace par service). Évite le couplage post→user et une jointure backend impossible (bases séparées). À l'échelle projet (≤ quelques centaines de suivis) le query string suffit ; bascule sur `POST /posts/feed` (liste en corps) si la liste grossit |
| Compteurs likes/comments (post) | Dénormalisés sur le document `posts` (`likes_count`/`comments_count`), maintenus par `$inc` à chaque like/commentaire. **Typés `int32`** (Go) car le validateur `$jsonSchema` les déclare `bsonType:"int"` — un `int64` (long) casserait la validation à l'insert/update. Likes idempotents via l'index unique `post_id+user_id` (on n'incrémente que si l'insert a créé une ligne / le delete en a supprimé une). | Lecture du fil sans agrégation (compteur lu direct sur le post). Idempotence = un like rejoué ne double pas le compteur. `int32` = contrainte du validateur, pas un choix de capacité |
| Client posts front (`lib/posts.ts`) | Mince au-dessus de `apiFetch` (Bearer + refresh hérités). Le post-service ne renvoie que `author_id` → le client **résout l'auteur** (username via user-service + displayName/avatar via profil-service) et **mémoïse** dans un `authorCache` (Map module-level) pour ne pas refetch le même auteur à chaque post d'un fil. État « liké » initialisé par `getLikedIds()` (pas de flag `is_liked` côté API, comme `getFollowingIds` pour les follows). Like/suppression **optimistes + rollback**. Post créé → event `breezy:post-created` (comme `breezy:profil-updated`) → le fil prépend sans refetch. Commentaires repliables chargés à la demande (`comment-section`). | Reprend les patterns acquis (enrichissement N+1 best-effort, état stateless via liste d'ids, broadcast par CustomEvent). Cache auteur = la seule optimisation N+1 nécessaire à l'échelle projet (vs endpoint agrégé/dénormalisation, en perspective) |
| Épinglage de post profil | Le propriétaire est le post-service : champ optionnel `pinned_at` sur le document `posts`, routes protégées `PATCH /posts/:id/pin` et `DELETE /posts/:id/pin`, autorisées uniquement à l'auteur. Lors d'un pin, le service désépingle les autres posts du même auteur, puis pose `pinned_at`. La lecture profil reste `GET /posts?author_id=` mais triée côté Mongo par `pinned_at DESC, created_at DESC`. Le front ne fait qu'afficher l'état, appeler pin/unpin, et synchroniser les posts visibles. | Le post épinglé doit être visible par tous les visiteurs du profil et survivre au refresh → donnée persistée côté service, pas état local. Un seul post épinglé par profil garde une UX claire façon X/Twitter. Le tri reste côté DB pour que `/profil` et `/profil/[username]` aient le même comportement |
| **Signets (bookmarks) — collections & rafale** | Propriétaire = **post-service** (mirroir likes/reposts), 3 collections : `bookmark_collections` (collections nommées par user), `bookmarks` (**many-to-many** : 1 doc par `user_id+post_id+collection_id`, index unique → idempotent), `bookmark_prefs` (1 doc/user : `last_collection_id` + `last_bookmark_at`). **Collection par défaut** (`is_default`, une seule par user, créée à la volée à la 1re lecture des collections, index unique partiel) : **toujours en tête, NON supprimable et NON renommable** (`ErrDefaultCollection` → 403), mais on y ajoute/retire des posts comme les autres → permet d'enregistrer **sans créer de collection**. Le front affiche un libellé localisé (`bookmarks.default_name`) pour cette collection. En plus, « Tous mes signets » reste une **vue virtuelle** distincte (union dédupliquée de TOUS les posts signés quelle que soit la collection, agrégation Mongo) — utile car un post peut être rangé dans une collection nommée sans être dans la défaut. Compteur d'items **calculé** à la lecture (`CountDocuments`, pas de dénormalisation → pas de dérive), comme les followers. **Modèle de rafale** : le **clic court** (`POST /posts/:id/bookmark` sans `collection_id`) consulte une **fenêtre glissante** (`BOOKMARK_SESSION_WINDOW`, défaut 5m) — si le dernier signet date de < fenêtre → range automatiquement dans `last_collection_id` (statut `filed`) ; sinon « ouverture de rafale » → statut `needs_choice`, **rien n'est rangé**, le front ouvre le sélecteur (choisir/créer une collection). `collection_id` fourni (sélecteur / appui long) → ajout explicite + repousse la fenêtre. **Dé-signer** = `DELETE` sans `collection_id` (retire de toutes les collections). État des boutons via `GET /posts/me/bookmarked-ids` (dédupliqué `Distinct`), cases du sélecteur via `GET /posts/:id/bookmark/collections`. Ownership stricte (404 sur collection d'autrui, pas de fuite d'existence). `withinSessionWindow` = fonction PURE testée. | Les signets ressemblent à likes/reposts (référencent un post) → même service. Many-to-many car « collection » (un post peut être rangé à plusieurs endroits), ≠ dossier unique façon X. **Modèle de rafale** (demande explicite, façon Instagram « enregistrer ») : on choisit la collection en début de session/rafale, puis on enchaîne sans friction tant qu'on reste dans la fenêtre — « 1er signet » = pas une notion d'« une fois pour toujours » mais « dernier signet ancien (> fenêtre) », porté **côté serveur** (`last_bookmark_at`, par compte, multi-appareil, pas de triche d'horloge client). `needs_choice` ne range rien (le serveur ne devine pas la collection au 1er signet). Compteur calculé = cohérent avec la philosophie du projet (followers). Fenêtre configurable comme les cooldowns identitaires |
| Traduction automatique des posts | La traduction passe par une **route BFF Next** (`POST /api/translate`) et non par le navigateur directement. Le client envoie `{text,targetLanguage}` ; la route appelle un fournisseur compatible LibreTranslate (`source:auto`, `target`, `format:text`) avec `TRANSLATION_API_URL` configurable et `TRANSLATION_API_KEY` optionnelle. Le front (`lib/post-translation.ts`) déduit la langue cible depuis `html lang` puis `navigator.language`, cache par post/langue/contenu, et `PostCard` affiche la traduction si elle diffère de l'original avec bascule original/traduit. | Les clés/API de traduction restent côté serveur. Le post-service reste indépendant et ne stocke pas de traduction dérivée. Le cache client évite de retraduire les mêmes posts au scroll. Le fallback sans erreur visible garde le fil utilisable si le fournisseur externe est lent/indisponible. |
| Recherche de comptes (cross-service, front) | Le préfixe `@` **route** la recherche : `@xxx` → `GET /users/search` (username, user-service) ; sinon → `GET /profils/search` (display_name, profil-service). Dans les deux cas l'orchestration + l'enrichissement (récupérer le champ manquant dans l'autre service) se font **côté front** (`searchUsers`) → `RelationUser[]` uniforme rendu par `UserListItem`. « Qui suivre » = `GET /users/suggestions` (tri SQL par `COUNT(followers)` DESC). État de suivi mutualisé dans le hook `use-follow` (modale/Explorer/sidebar) | username et display_name vivent dans 2 services distincts (règle « une donnée = un service ») → aucune jointure backend possible, la vue agrégée se compose chez l'appelant (pattern déjà acté pour le profil). `@` = signal explicite et familier (identifiant) vs nom affiché. Tri par followers en SQL (sous-requête `COUNT`) plutôt que dénormalisé : OK à l'échelle projet. Hook partagé pour ne pas tripler la logique optimiste de follow |
| **Convention : identité d'un utilisateur = cliquable → son profil** | Partout où un utilisateur est affiché (avatar + nom/@handle), cliquer **mène à son profil** (`/profil/<username>`, ou `/profil` pour soi). Implémenté dans `UserListItem` via un **lien « étiré »** (`<Link absolute inset-0>`) qui rend toute la zone de survol cliquable ; le bouton « Suivre » est remonté (`relative z-10` + `preventDefault`) pour rester actionnable sans naviguer. Couvre déjà Explorer, « Qui suivre » et la modale des relations. Par ailleurs l'avatar du composer (« Ça breez ? ») affiche le **vrai avatar de l'utilisateur courant** (`getMyProfil` + resync `subscribeProfilUpdated`, même source que la sidebar). **À étendre** quand elles arriveront : **messages privés** et **notifications** — au minimum la **photo de profil** (idéalement nom + @handle) doit renvoyer vers le profil de la personne | Cohérence UX façon X/Twitter : une identité est toujours un point d'entrée vers le profil. Le lien étiré garde « tout le hover » cliquable sans HTML invalide (`<button>` dans `<a>`). Convention posée maintenant pour que DM/notifications la respectent dès leur implémentation (la photo de profil = ancre minimale garantie) |
| **Messagerie E2EE — modèle (hybride)** | Chiffrement **côté navigateur** (≠ at-rest serveur). Chaque user a une paire **X25519** (privée en IndexedDB, **par appareil**, jamais envoyée). Chaque conversation a une **clé de contenu symétrique** ; pour DM/groupes elle est **emballée par membre** (sealed box anonyme) → le serveur ne voit que des enveloppes + des `ciphertext`/`nonce` (XChaCha20-Poly1305), il ne peut RIEN déchiffrer. **Hybride** : DM + groupes = vrai E2EE **admin-proof** ; **communautés** = la clé est **détenue par le serveur** (remise à chaque arrivant pour l'auto-join illimité) → semi-publiques, **lisibles par un admin**. | Tient « même les admins ne peuvent pas lire » pour le privé. « Viewers illimités + auto-join par des inconnus » est impossible en E2EE strict (il faudrait un détenteur de clé en ligne) → communautés « à la Telegram channel ». Clé par appareil = simple (backup passphrase = perspective). Transport **WebSocket** (`/messages/ws`, token en query param, hub par user) choisi pour la fluidité d'un chat |
| **Messagerie — groupes / communautés** | **Groupes** : owner + membres ; **tout membre invite** (il détient la clé → l'emballe pour l'invité) ; **owner-only** exclure/renommer/supprimer ; owner ne peut pas quitter (supprime) ; cap 32 ; nom **chiffré** ; **clé unique sans rotation** → nouvel arrivant lit tout l'historique (Option A). **Communautés** : rejoint en **viewer** (auto-join), **owner promeut** viewer↔talker (cap 32 talkers, viewers illimités), nom **en clair** + **annuaire public** (`GET /messages/communities?q=`, sans la clé), `canWrite` refuse les viewers. Endpoints généralisés group→community via `isManageable`. | Modèle « à la Insta » (groupes) validé. Pas de rotation = simple/démo-able (forward secrecy = perspective). Nom de communauté en clair = nécessaire pour la découverte (un non-membre lit le nom avant de rejoindre). Cap sur les talkers (pas les viewers) = contrainte produit. `content_key` exposée **uniquement aux membres** (via `buildView`), jamais dans l'annuaire |
| **Messagerie — pagination par curseur** | `GET .../messages?limit=&before=<messageId>` : sans `before` = la page la plus récente ; avec = les messages antérieurs (scroll haut). **Curseur sur `_id`**, pas d'offset. Front : `listMessagesPage → {messages, hasMore, oldestId}` (pur, testé) pour brancher `useInfiniteScroll`. | Sur un chat *vivant* l'offset se décale (doublons/trous) ; le curseur `before _id` est stable (les nouveaux messages arrivent par WS en bas, sans perturber la pagination arrière) |
| **Schéma DB (message)** | Pattern autonome (post/profil) : `EnsureSchema` crée collections (`user_keys`/`conversations`/`members`/`messages`) + validateurs `$jsonSchema` + index au boot, resync `collMod`. `mongo-message` (port hôte `27019`). Les `messages` n'ont **aucun champ clair** (`ciphertext`+`nonce`) ; `user_keys` ne stocke que la clé PUBLIQUE ; `conversations.content_key` (communautés only). | Source de vérité unique + service autonome. Le validateur Mongo **matérialise** la garantie « pas de clair en base » pour DM/groupes |
| **Messagerie — épinglage / suppression « côté user »** | État **par-utilisateur** porté par la collection `members` (et non `conversations`) : `pinned_at` + `cleared_at` (nullable). Endpoints `PATCH/DELETE /conversations/:id/pin` et `DELETE /conversations/:id/me`, **sans diffusion WS** (personnel). « Supprimer » = poser `cleared_at` → `ListConversations` masque la conv tant qu'aucun message n'est postérieur (`HasMessagesAfter`), `ListMessages` filtre `created_at > cleared_at` (historique coupé). Tri « épinglées d'abord » (`convLess` PURE). Le front réplique le tri (`sortConversations`) car le `bump` temps réel doit respecter l'épinglage. | Le pin/clear ne concernent QUE le membre courant → l'état vit sur SON document `members`, pas sur la conversation partagée (≠ DeleteConversation owner qui efface pour tous). `cleared_at` (cutoff) plutôt qu'une vraie suppression = réversible, non destructif pour les autres, et « réapparaît au prochain message » gratuitement (façon WhatsApp). Choix back-side (≠ localStorage de l'état lu) tranché avec l'utilisateur : par-compte, multi-appareil, cohérent « une donnée = un service » |
| **Notifications — émission (post→notif)** | Communication **synchrone best-effort fire-and-forget** : post-service (`internal/notifier`) poste l'événement à `notification-service:8086/internal/events` en **goroutine** (timeout court, erreur loggée jamais propagée → un like/commentaire ne casse JAMAIS si notif est down). Appel **direct sur le réseau Docker** (pas via la gateway), authentifié par `INTERNAL_EVENT_SECRET` (en-tête `X-Internal-Secret`). Émission désactivée si `NOTIFICATION_SERVICE_URL`/`INTERNAL_EVENT_SECRET` absents → post-service reste autonome. Interface `Notifier` (+ `Noop`) injectée via `SetNotifier` (tests inchangés). | Couplage temporel neutralisé par le best-effort (même pattern que le provisioning user). Serveur-à-serveur ≠ trafic client → la règle « tout par la gateway » ne s'applique pas (aucun gain de sécurité, juste un hop). **Perspective rapport** : un bus d'événements (NATS/Redis) découplerait totalement (post ne connaîtrait plus notif) — surdimensionné à l'échelle projet, l'agrégation amortit déjà les pics |
| **Notifications — agrégation (anti-spam)** | Une notification = un GROUPE `group_key` unique par destinataire (`{recipient_id, group_key}` index unique) : `like:<post>`, `comment:<post>`, `reply:<rootComment>`, `mention:<source>`. `Upsert` = `$inc count` + `$set last_actor_id`/`is_read:false`/`updated_at` → 300 likes = **un seul** document, `count=300`. Affichage « X et N autres » = `last_actor_id` (résolu en username/avatar côté front, cache) + `count-1`. `retract` (unlike, suppr. commentaire) = `$inc -1`, suppression à 0. PAS de tableau d'acteurs (compteur `count` simple suffit ; léger flou cosmétique possible sur `last_actor` après un retract, assumé). | Tient « 300 likes = 1 notif façon Insta » avec un coût O(1) par événement (upsert) et zéro agrégation à la lecture. `count` entier toujours juste (≠ recompter un set d'acteurs). Acteurs résolus front = cohérent avec posts (cache auteur, pas de N+1 backend) |
| **Notifications — règles par type** | **like** → auteur du post. **comment** (racine) → auteur du post. **reply** → auteur de la **RACINE** du fil (clé `reply:<rootComment>`), PAS l'auteur du post — « réponse à ton commentaire ». **mention** (`@handle`) → mentionnés, **une notif par mention** (pas d'agrégation). **repost** → auteur du post (agrégé `repost:<post>`, retract à l'unrepost). **citation** (`quote_post_id`) → auteur cité, **une par citation** (`quote:<postCitant>`, post_id = post citant → navigation vers la citation ; purge à la suppression du post citant via `post_deleted`) — repost/citation = **tag implicite** de l'auteur. Jamais à soi-même (recipient≠actor). Suppression de commentaire/post **défait** (post supprimé = purge `DeleteByPost`). | Threading à plat sous la racine : seul `parent_id`=racine est stocké → notifier/clé sur la racine garde la **symétrie création↔suppression** (la suppression n'a pas l'`parent` cliqué d'origine). Mentions = contexte distinct → non agrégées. Résolution `@handle`→id par notif-service (seul user-service connaît les handles ; parser anti-email `(?:^\|[^\w@])@\w{3,50}`) |
| **Notifications — front (badge + temps réel)** | `lib/notifications.ts` (REST+WS sur `apiFetch`). `NotificationsProvider` (Context, monté dans `(app)/layout`) tient le **compteur non-lu app-wide** + **une seule** connexion WS + la liste vivante. Badge amorcé par `GET /unread-count`, maintenu en mémoire : une notif entrante non-lue d'id inconnu incrémente (les activités répétées d'un même groupe partagent l'id → comptées une fois). Ouvrir `/notifications` = `markAllSeen` (badge→0). WS pousse `notification`/`notification_deleted`/`notification_refresh`. Clic notif → **page détail `/posts/[id]`** (réutilise `PostCard`). Acteur (avatar+nom) → profil (convention « identité cliquable »). | Provider dans le layout = badge présent partout sans recharger la liste à chaque page, WS unique partagé. Dédup par id du badge évite de recompter un même groupe sans requête serveur par événement (≠ refetch sur chaque like d'un pic de 300). Page détail nécessaire car « aller au post direct » et le fil n'a pas de vue post unique |
| **Messagerie — état « lu » côté serveur + badge non-lu** | L'état « lu » est devenu une **donnée serveur** (avant : `localStorage` par appareil) : curseur `members.last_read_at` (jumeau de `pinned_at`/`cleared_at`). **`PUT /messages/conversations/:id/read`** avance le curseur à l'ouverture ; **`GET /messages/unread-count`** renvoie `{count}` = **nb de conversations** ayant un message postérieur à `max(last_read_at, cleared_at)` **et** pas de soi (agrégation `$lookup` bornée à 1 msg/conv, **jamais le `ciphertext`**). Front : **`MessagesProvider`** (monté dans `(app)/layout`, jumeau de `NotificationsProvider`) **possède l'UNIQUE WS messages** (la vue `/messages` s'y abonne via `subscribeMessages`/`subscribeEvents` au lieu d'ouvrir la sienne, et déchiffre seule) ; il amorce le badge via `unread-count` puis **ré-interroge le serveur (coalescé 300 ms)** sur message entrant d'autrui / marquage lu → toujours exact, multi-appareil. Conv active (`setActiveConversation`) → messages entrants marqués lus à la volée. Pastille de liste + ancre « Nouveaux messages » dérivées de `lastReadAt` (`computeDivider` prend désormais un **timestamp**, plus un id). Badge nav réutilisé tel quel (sidebar + barre mobile), `messages.badge_aria`. `message-reads.ts` (localStorage) **supprimé**. **Sourdine par conversation** : `members.muted_at` (jumeau de `pinned_at`), `PATCH/DELETE /conversations/:id/mute`, `ConversationView.muted` ; `CountUnreadConversations` ajoute `muted_at: null` à son `$match` → une conv en sourdine **n'alimente pas le badge** mais **reste « non lue » dans la liste** (pastille client inchangée). Toggle dans le menu « … » de la ligne + icône cloche barrée ; `toggleMute` optimiste appelle `refresh()` (le serveur exclut, le provider ré-interroge). | Demande utilisateur : badge « rond avec compteur » comme les notifs **sans charger tous les messages** côté client. Le serveur calcule (métadonnées seules → E2EE intact). Ré-interrogation coalescée plutôt qu'un compteur en mémoire fragile : débit d'événements faible en messagerie → résultat exact + simple, et **multi-appareil** (lu sur un appareil = lu partout, livre la perspective « accusés de lecture »). WS unique (≠ 2 connexions) tranché avec l'utilisateur |

---

## 6. KNOWN ISSUES / BLOCKERS

> Add/remove as issues arise.

- **✅ RÉSOLU — drift du validateur Mongo `profiles`.** Un volume `mongo-profil` créé par une
  **ancienne** version d'`EnsureSchema` gardait un validateur `$jsonSchema` exigeant encore
  `user_id` + **`username`** (avant le déménagement de `display_name`) → le `POST /profils` du
  register (qui n'envoie plus `username`) échouait → aucun profil créé → recherche par
  `display_name` vide. **Correctif** : `ensureCollections` applique désormais `collMod` sur les
  collections existantes (resync idempotent du validateur), au lieu de les ignorer — le service
  **maintient** son schéma à chaque boot, pas seulement à la création. Vérifié e2e (validateur
  repassé à `user_id`+`created_at`, `POST`/insert sans username OK, recherche display_name OK).
  ⚠️ **Limite résiduelle** (pas un bug) : les comptes créés par un login-only (sans passer par le
  register) n'ont **pas** de profil — le profil n'est créé qu'au register (`POST /profils`) ou via
  la popup d'édition (branche WIP), **jamais** au login (décision : pas de provisioning paresseux
  du profil). Ces comptes ne sont donc pas trouvables par `display_name` tant qu'ils n'ont pas de
  profil. (Note dev : le `display_name` du seed admin a été corrigé en base sur le volume courant ;
  un volume recréé l'aura via le seed.)

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

- **MESSAGERIE — COMPLÈTE (back + UX).** message-service 🟢 (DM + groupes + communautés +
  WebSocket) **+ UI livrée** (`components/messages/`, page `/messages` deux volets). **Compromis
  assumés** : (1) communautés **lisibles par un admin** (clé serveur, ≠ DM/groupes admin-proof) ;
  (2) clé d'identité **par appareil** (nouvel appareil ne déchiffre pas l'historique d'avant → la
  conversation affiche un bandeau « clé indisponible » et désactive lecture/écriture). **État « lu »
  désormais serveur** (`members.last_read_at`, multi-appareil) + **badge non-lu app-wide** via
  `MessagesProvider` (cf. §5) ; le serveur ne lit que des métadonnées (E2EE intact). **Perspectives** :
  aperçu du dernier message dans la liste (nécessiterait de déchiffrer le dernier message par conv au
  chargement), pièces jointes, rotation de clé (forward secrecy), transfert d'ownership, backup
  passphrase (login multi-appareil — règle aussi la cause « nav privée / nouvel appareil » du non-déchiffrement,
  + à coupler avec un stockage de clé scopé **par utilisateur** dans IndexedDB, cf. clé `'self'` unique
  actuelle qui mélange deux comptes sur un même navigateur), coffre séparé pour `content_key`.

- **SETUP — `message-service/.env` doit exister pour `make dev`/`docker compose`.** Il était absent
  (seul `.env.example` présent) et **bloquait tout le compose** (`env file ... not found`). Créé via
  `cp message-service/.env.example message-service/.env` (gitignoré, comme les autres services). À
  refaire sur un checkout neuf : `make env` (ou copier l'exemple) avant `make dev`.

- **NOTIFICATIONS — compromis assumés.** (1) Émission post→notif **best-effort sans réessai** :
  si notif-service est down à l'instant T, l'événement est perdu (le like/commentaire réussit
  quand même). Mitigation : un bus d'événements (NATS/Redis) en perspective. (2) Le badge non-lu
  est maintenu **en mémoire côté front** (dédup par id) : amorcé par `unread-count`, il peut
  sur-compter de 1 si une activité arrive sur une notif déjà non-lue **antérieure à la session** et
  non encore chargée — s'auto-corrige à l'ouverture de la page (`markAllSeen`) ou sur
  `notification_refresh`. (3) Tant que la page `/notifications` est ouverte, une nouvelle notif
  **incrémente quand même** le badge (pas de « lu en direct »). (4) Après un `retract`,
  `last_actor_id` peut rester celui de l'acteur parti (flou cosmétique, jamais le `count`).
  (5) **Pas de notif de follow** (hors périmètre ; reposts + citations notifient désormais). Perspectives : exactly-once
  (broker), notifs de follow, ne pas incrémenter quand la page est active, rôle réel (placeholder).

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

## CHANGELOG

> Historique complet déplacé dans **[CHANGELOG.md](CHANGELOG.md)** (sortie du contexte auto-chargé). Ne garder ici que les 2-3 dernières entrées ; pousser les plus anciennes vers `CHANGELOG.md`.

*Last updated: 09/06/2026 — feat(settings/feed) : **mots filtrés par utilisateur** dans `/parametres` sous la langue. UI champ + bouton icône + chips supprimables, zone scrollable après ~3 lignes. Persistance locale scopée compte (`breezy-muted-words:<user_id>`, résolution via `getMe()` après reconnexion), filtrage feed côté front sur contenu/post cité/hashtags, avec exception pour les propres posts de l'utilisateur. Vérifs : lint/tsc/localhost:3000 OK.*

*Last updated: 09/06/2026 — feat(signets) : **enregistrement de posts en collections (back + front)** + **collection par défaut non supprimable**, branche `116-signets` (rebasée sur `origin/develop` = notifications #138 puis mots filtrés #140). Back post-service : 3 collections (`bookmark_collections`/`bookmarks` many-to-many/`bookmark_prefs`), collection `is_default` (en tête, non supprimable/renommable → 403), modèle de rafale `BOOKMARK_SESSION_WINDOW` (`filed`/`needs_choice`), vue « Tous » dédupliquée, nettoyage à la suppression post/collection. Front : `lib/bookmarks.ts` + `FeedPost.bookmarked`, bouton signet `PostCard` (clic court / appui long / dé-signer + toast « Ranger… »), `BookmarkDialog`, page `/signets` (`BookmarksView`), nav + i18n FR/EN (23 clés). Détails §5 « Signets — collections & rafale ». **Vérifié** : back gofmt/build/`go test` + e2e gateway 24/24 ; front tsc/lint/vitest + parité i18n.*

*Last updated: 09/06/2026 — feat(messages) : **badge de messages non lus app-wide + état « lu » côté serveur**. Back : `members.last_read_at` (+ validateur), `PUT /messages/conversations/:id/read`, `GET /messages/unread-count` (agrégation `$lookup` par métadonnées, jamais le `ciphertext` → E2EE intact). Front : `MessagesProvider` (monté dans `(app)/layout`) **possède l'unique WS messages** ; `MessagesView` s'y abonne (plus de WS propre) ; badge dégradé sur l'icône Messages (sidebar + barre mobile), `messages.badge_aria` FR/EN. `message-reads.ts` (localStorage) supprimé ; `computeDivider` prend un timestamp (`lastReadAt`). **+ Mise en sourdine par conversation** : `members.muted_at`, `PATCH/DELETE /conversations/:id/mute`, `ConversationView.muted` ; `CountUnreadConversations` exclut les sourdines (`muted_at: null`) → exclue du badge mais **toujours non-lue dans la liste** ; toggle menu « … » + icône cloche barrée, `toggleMute` optimiste + `refresh()`. go build/vet/test OK, tsc/lint OK, vitest 40/40. Détails §5 (décision « état lu serveur + badge ») / §6.*

*Last updated: 09/06/2026 — chore(docs) : **allègement de `CLAUDE.md`** (dépassait la limite de chargement de contexte ~40 KB, il faisait 113 KB). (1) Tout l'historique « Last updated » déplacé dans **`CHANGELOG.md`** (non auto-chargé) ; ne restent ici que les dernières entrées. (2) Colonnes « Status » des tableaux §3 (services + features) **condensées** (état + endpoints clés + renvoi vers §5 + TODO) — le détail d'implémentation vit dans le code et la §5. Résultat : 113 KB → ~64 KB. §5 (décisions techniques, utile pour la soutenance) et §6 (issues) conservées intégralement.*
