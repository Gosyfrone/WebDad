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
| Auth Service | 🔴 TODO | PostgreSQL | JWT, login/register |
| User Service | 🔴 TODO | PostgreSQL | CRUD users |
| Post Service | 🔴 TODO | MongoDB | CRUD posts |
| Profil Service | 🔴 TODO | MongoDB | User profiles |
| API Gateway | 🔴 TODO | — | Route dispatch, auth middleware |
| Frontend | 🟡 WIP | — | Next.js 14 : squelette + routing + layout feed 3 colonnes (style X). Données feed = stubs, API à brancher |

### Features status
| Feature | Type | Status |
|---|---|---|
| Registration / Login | Primary | 🔴 TODO |
| JWT auth + protected routes | Primary | 🔴 TODO |
| Role management (User/Mod/Admin) | Primary | 🔴 TODO |
| Post creation/reading | Primary | 🟡 UI faite (feed + composer 280 car. inline & popup sidebar), lecture/écriture API à brancher |
| User profile | Primary | 🔴 TODO |
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
| Thème / couleurs | Thème clair par défaut : fond blanc, primaire magenta `#e053ff` (texte foncé pour lisibilité). 2 dimensions : mode clair/sombre (next-themes, `.dark`) + accent (`[data-accent]` : pink défaut / blue / cyan). Registre dans `lib/themes.ts` | Identité visuelle + dark theme et thèmes custom anticipés sans dupliquer la palette |
| Inter-service auth | JWT passed in header | Grading requirement |
| Containerization | Docker + docker-compose | Grading requirement |
| Dev hot-reload | `make dev` = overlay `docker-compose.dev.yml` par-dessus la base. Go : stage `dev` du Dockerfile (`air` épinglé `air-verse/air@v1.52.3`) + `.air.toml` par service + bind-mount source ; caches Go partagés (volumes `go-mod-cache`/`go-build-cache`). Frontend : stage `deps` + `command: npm run dev` + bind-mount + volume anonyme `node_modules` + `WATCHPACK_POLLING=true`. Le `frontend` voit son `depends_on` effacé via `!reset`. `make up` reste les images de prod figées | Itérer sans rebuild. Stage `dev` placé AVANT le runtime → `make up`/`build` produisent toujours l'image de prod (dernier stage). `!reset` car un `depends_on: []` ne vide pas (compose fusionne les mappings). `start_period: 90s` sur les services Go pour laisser le 1er build `air` se faire |
| Config `.env` | Racine = vars transverses (`JWT_SECRET`, `JWT_EXPIRY`, `NEXT_PUBLIC_API_URL`) ; `<service>/.env` = config propre, chargée par compose via `env_file:` ; `environment:` réservé aux overrides Docker (host = nom de conteneur) | Découplage : un service tourne seul (`make run`) avec son `.env`, et en stack via compose. ⚠️ Les vars d'un `env_file` ne sont PAS interpolables (`${...}`) dans le compose — seul le `.env` racine l'est. DB host surchargé via `DB_HOST`/`MONGO_HOST` |

---

## 6. KNOWN ISSUES / BLOCKERS

> Add/remove as issues arise.

- **Rappel : `make up` = images de prod FIGÉES.** L'image frontend embarque un `npm run build`
  (`output: standalone` → `node server.js`) et les images Go un binaire statique, tous figés au
  build. `docker compose up` ne rebuild PAS sur changement de source → un changement de code
  n'apparaît qu'après `make build`. **Pour développer avec hot-reload, utiliser `make dev`** (voir §5).

---

## 7. INSTRUCTIONS FOR CLAUDE CODE

1. **Read this file first** at every session start.
2. **Update sections 3, 5, 6** after any significant change.
3. **Never hardcode secrets** — use `.env` variables.
4. **Each service is independent**: its own Dockerfile, its own DB.
5. When implementing a feature, **update its status** in section 3 (🔴→🟡→🟢).
6. Before any architectural decision, **check section 2** (evaluation criteria).
7. Keep responses concise — update this file rather than re-explaining context.

---

*Last updated: 02/06/2026 — feat(frontend) : popup de publication + emoji picker + onglet Abonnements (placeholder)*
