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
| Auth Service | 🟡 WIP | PostgreSQL | Squelette fonctionnel (config/db/models/services/handlers/middleware/router). Routes `/auth/register`, `/auth/login`, `/auth/validate`, `/health` — testées de bout en bout. JWT HS256 (claims user_id/email/role), bcrypt. **Service autonome** : applique son propre schéma au boot (`internal/db/schema.sql` embarqué, idempotent) → plus de script init-db monté. Seed admin optionnel (`SEED_DEFAULT_ADMIN`). TODO : tests Go, refresh tokens |
| User Service | 🟡 WIP | PostgreSQL | Squelette complet (config/db/models/repository/service/handlers/middleware/router) calqué sur auth. **Autonome** : applique son schéma au boot (`internal/db/schema.sql` embarqué, idempotent — tables `users`+`follows`, seed admin UUID figé) → plus de script init-db monté. **CRUD users réel et testé e2e** : `GET /users` (paginé), `GET /users/:id`, `POST /users` (id=claims), `GET/PATCH /users/me`, `DELETE /users/:id` (admin). **Provisioning paresseux** : `/users/me` crée la ligne à la volée depuis les claims JWT (relie register→user sans coupler auth↔user). Middleware JWT (validation locale, `JWT_SECRET` partagé). Graphe `follows` = **stubs** (501). TODO : implémenter follows, tests Go |
| Post Service | 🟡 WIP | MongoDB | Connexion Mongo via `.env` (URI construite, plus rien en dur). **Autonome** : crée ses collections (posts/comments/likes/reports) + validateurs `$jsonSchema` + index au boot (`EnsureSchema`, idempotent) → `post-init.js` supprimé. Champs snake_case (`author_id`/`content`/`created_at`). Routes posts create/list OK (testées) ; comments/likes = stubs. `make dev` vérifié (air + Mongo) |
| Profil Service | 🔴 TODO | MongoDB | User profiles |
| API Gateway | 🟡 WIP | — | Reverse proxy (`httputil.ReverseProxy`) : préfixe `/auth`,`/users`,`/profils`,`/posts` → service cible (URLs via `.env`). Middleware CORS (origines via `CORS_ALLOWED_ORIGINS`). `/auth/*` proxifié vers auth-service. Middleware JWT à ajouter pour les routes protégées (login) |
| Frontend | 🟡 WIP | — | Next.js 14 : squelette + routing + layout responsive (style X). Pages feed + profil (consultation/édition) + placeholders explorer/notifications/messages. Mobile-first : en-tête mobile (avatar→menu + logo), barre d'onglets en bas, FAB « + », sélecteur de thème clair/sombre/système dans le tiroir. Données = stubs, API à brancher |

### Features status
| Feature | Type | Status |
|---|---|---|
| Registration / Login | Primary | 🟡 Auth-service fait (register/login) + gateway proxy (`/auth/*`). UI à brancher |
| JWT auth + protected routes | Primary | 🟡 Auth-service : génération + `/auth/validate` + middleware. Gateway à brancher |
| Role management (User/Mod/Admin) | Primary | 🟡 Rôle porté par le JWT (auth) ; user-service garde l'enregistrement public et applique un contrôle de rôle (`DELETE /users/:id` réservé admin). UI nav par rôle déjà faite. TODO : généraliser côté autres services |
| Post creation/reading | Primary | 🟡 UI faite (feed + composer 280 car. inline & popup sidebar), lecture/écriture API à brancher |
| User profile | Primary | 🟡 UI faite (consultation : bannière/avatar/bio/compteurs/onglets + édition popup nom/bio + photo/bannière via sélecteur de fichier avec aperçu local). Upload réel + lecture/écriture API à brancher |
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
| Schéma DB (post) | Le service possède son schéma : `EnsureSchema` (post-service/internal/database/init.go) crée collections + validateurs `$jsonSchema` + index au démarrage, idempotent. Aucun script monté dans `mongo-post`. Config Mongo construite depuis le `.env` (`internal/config`), zéro creds en dur. `scripts/init-db/post-init.js` et le `post-service/docker-compose.yaml` parasite supprimés | Source de vérité unique + service autonome (`make run`/`make dev` contre un Mongo vierge). Même pattern que auth. ⚠️ user-service/profil-service restent sur init-db monté — à harmoniser quand on les traitera |
| Schéma DB (auth) | Le service possède son schéma : `auth-service/internal/db/schema.sql` (embarqué `go:embed`), appliqué au boot de façon idempotente (`EnsureSchema`). Aucun script monté dans `postgres-auth`. Seed admin via `SEED_DEFAULT_ADMIN`+`SEED_ADMIN_PASSWORD` (idempotent, UUID figé). `scripts/init-db/auth-init.sql` supprimé | Source de vérité unique + service autonome (`make run` contre un Postgres nu). Même pattern que post. ⚠️ profil-service reste sur init-db monté — à harmoniser quand on le traitera |
| Schéma DB (user) | Même pattern autonome : `user-service/internal/db/schema.sql` embarqué + `EnsureSchema` au boot (tables `users`+`follows`, index, trigger `updated_at`, seed admin UUID figé). Mount `scripts/init-db/user-init.sql` retiré du compose, fichier supprimé. `users.id` = `credentials.id` (auth) | Cohérent avec auth/post (TODO §6 d'harmonisation traitée pour user). Reste profil |
| Provisioning user (lazy) | La table `users` n'est PAS remplie par auth au register (bases séparées + règle « tout passe par la gateway »). À la place : `POST /users` exposé (CRUD/admin/tests) **+** provisioning paresseux sur `GET /users/me` (upsert depuis les claims JWT au 1er accès authentifié). Username dérivé de l'email (modifiable via PATCH) | Zéro couplage auth↔user, service autonome, robuste en démo. Laisse aussi la porte à un flow « première connexion » distinct plus tard. Alternative écartée : auth appelle user au register (couplage + incohérence transactionnelle inter-bases) |
| Layering user-service | Couche `repository` (SQL pur) sous `service` (métier/erreurs) sous `handlers`, contrairement à auth (SQL inline dans le service) | Le CRUD users+follows a beaucoup plus de requêtes → couche repo dédiée justifiée (proche de post-service) |

---

## 6. KNOWN ISSUES / BLOCKERS

> Add/remove as issues arise.

- **Rappel : `make up` = images de prod FIGÉES.** L'image frontend embarque un `npm run build`
  (`output: standalone` → `node server.js`) et les images Go un binaire statique, tous figés au
  build. `docker compose up` ne rebuild PAS sur changement de source → un changement de code
  n'apparaît qu'après `make build`. **Pour développer avec hot-reload, utiliser `make dev`** (voir §5).

- **À FAIRE — harmoniser la gestion de schéma du dernier service.** `auth-service`,
  `post-service` et `user-service` sont passés au pattern « le service possède son schéma »
  (schéma embarqué + `EnsureSchema` au boot, cf. §5). Il ne reste que **`profil-service`** sur
  le script `scripts/init-db/profil-init.js` monté. **Quand on l'attaquera, on changera son init
  BDD** pour le même pattern autonome (et on supprimera `profil-init.js`).

- **À FAIRE — implémenter le graphe `follows` du user-service.** Routes/handlers en place mais
  en stub (501) : `POST/DELETE /users/:id/follow`, `GET /users/:id/followers|following`. La table
  `follows` existe déjà dans le schéma. À brancher (repo + service) dans une issue dédiée.

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

*Last updated: 03/06/2026 — feat(user-service) : squelette CRUD users (repository/service/handlers/middleware/router) calqué sur auth, schéma autonome embarqué (users+follows, init-db retiré), provisioning paresseux sur /users/me, contrôle de rôle admin, follows en stub. Testé e2e. (précédemment : feat(frontend) switch de thème clair/sombre/système animé)*
