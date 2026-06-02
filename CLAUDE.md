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
| Frontend | 🔴 TODO | — | React, all 3 roles |

### Features status
| Feature | Type | Status |
|---|---|---|
| Registration / Login | Primary | 🔴 TODO |
| JWT auth + protected routes | Primary | 🔴 TODO |
| Role management (User/Mod/Admin) | Primary | 🔴 TODO |
| Post creation/reading | Primary | 🔴 TODO |
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
| Frontend | *(e.g. React + Vite)* | |
| Inter-service auth | JWT passed in header | Grading requirement |
| Containerization | Docker + docker-compose | Grading requirement |
| Config `.env` | Racine = vars transverses (`JWT_SECRET`, `JWT_EXPIRY`, `NEXT_PUBLIC_API_URL`) ; `<service>/.env` = config propre, chargée par compose via `env_file:` ; `environment:` réservé aux overrides Docker (host = nom de conteneur) | Découplage : un service tourne seul (`make run`) avec son `.env`, et en stack via compose. ⚠️ Les vars d'un `env_file` ne sont PAS interpolables (`${...}`) dans le compose — seul le `.env` racine l'est. DB host surchargé via `DB_HOST`/`MONGO_HOST` |

---

## 6. KNOWN ISSUES / BLOCKERS

> Add/remove as issues arise.

- *(none yet)*

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

*Last updated: [DATE — update manually or ask Claude Code to update]*
