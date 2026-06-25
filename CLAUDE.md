# CONTEXT — Breezy (WebDad) · Distributed App Dev Project (FISA INFO A3)

> Entry point for Claude Code. Read this first at every session start.
> Detail lives in dedicated files (keep this one lean):
> - **[ARCHITECTURE.md](ARCHITECTURE.md)** — system architecture, service boundaries, flows.
> - **[DECISIONS.md](DECISIONS.md)** — architectural decisions + rationale (defense prep).
> - **[PROJECT_STATUS.md](PROJECT_STATUS.md)** — current state, open TODOs, compromises.
> - **[CHANGELOG.md](CHANGELOG.md)** — session work log, handoffs, suggested commit messages · **[CHANGELOG_ARCHIVE.md](CHANGELOG_ARCHIVE.md)** — resolved/debugging history.
> - **[PROMPTING.md](PROMPTING.md)** — token-efficient workflow + prompt templates (read on demand, not auto-loaded).
> - Knowledge graph in `graphify-out/` — query it before reading source (Operating Rule 11).
>
> **Latest session work and suggested commit messages live in `CHANGELOG.md`, not here.**
> **Session freshness:** current work log updated in `CHANGELOG.md` on 25/06/2026 (feat(communities) — **inviter des membres dans une communauté + lien d'invitation auto-join, avec retour-après-login**. Nouvelle route back `POST /messages/conversations/:id/invite` (`InviteToCommunity`, ajout en **viewer sans envelope** car le serveur détient la clé ; tout membre peut inviter ; 409 si déjà membre ; `AddMember` groupe inchangé). Front : `inviteToCommunity()`, zone « Inviter » (UserSearch) affichée pour les communautés, **bouton « Copier le lien »** → `${origin}/messages?join=<id>`, `messages-view` gère `?join=` (joinCommunity + ouverture). Retour-après-login : `middleware.ts` pose `?next=`, `login/page.tsx` `postLoginTarget()` (validé anti-open-redirect) y revient (mot de passe + MFA ; OAuth non couvert). i18n 12 locales. Swagger régénéré. Tests : message-service `go test ./...` vert + invite testé live ; front tsc 0, 150/150 vitest. Commit suggéré : `feat(communities): invite members + shareable join link with post-login return`). Entrée précédente 25/06/2026 (feat(visitor/ui) — interrupteur clair/sombre désormais accessible aux **visiteurs** sur `/feed` et `/posts/:id`. Front-only, aucune nouvelle clé i18n. Extraction de la pilule Soleil/Lune en composant présentationnel réutilisable `ThemeSwitch({ className })` dans `components/theme-toggle.tsx` (apparence seule, positionnement fourni par l'appelant ; `FloatingThemeToggle` devient un wrapper → zéro régression login/register/légales). Mobile : l'interrupteur remplace la cloche notifications dans `mobile-header.tsx` pour le visiteur. Desktop : flottant bas-gauche comme login/register via le nouveau `visitor-theme-toggle.tsx` monté dans `app/(app)/layout.tsx`, `hidden lg:inline-flex` (la tab bar mobile occupe le bas). `tsc --noEmit` propre, 36/36 tests verts. Commit suggéré : `feat(visitor): expose dark-mode toggle on visitor pages`). Entrée précédente 25/06/2026 (fix(visitor/auth) — le mode visiteur n'éjecte plus vers `/login` sur le feed. Cause : un access token **périmé resté en `localStorage`** rendait `getAccessToken()` non-`null` → app « connectée » → salve d'appels auth-only au montage de `/feed` → 401 → refresh KO → `apiFetch` redirigeait en dur vers `/login`. Fix front-only `lib/auth-client.ts` : `redirectToLogin()` devient **conscient du chemin** (`isVisitorPath` → `/feed`, `/posts/:id`) : pas de redirection sur les pages publiques (dégradation visiteur via `AuthPromptProvider`), redirection **conservée** sur les routes protégées. 13/13 tests `auth-client-fetch.test.ts` verts. Commit suggéré : `fix(auth): keep visitor mode on public pages instead of redirecting to /login`). Entrée précédente 24/06/2026 (fix(ui/register) — case CGU **directement cochable** sur le register : suppression du gate de lecture (state `termsRead`, `useEffect` focus/storage, `disabled`), de la **card** et de la **phrase d'aide** sous « J'accepte » ; consentement toujours requis pour soumettre. i18n finalement **inchangée** (clés `terms_locked`/`terms_unlocked`/`err.terms_read_required` conservées car encore utilisées par la page OAuth `auth/oauth/terms`, qui garde son gate). Front-only, aucun back/route/Swagger. tsc propre sur fichiers touchés, 16/16 tests i18n verts. Commit suggéré : `fix(register): make terms checkbox directly checkable, drop read-gate UI`). Entrée précédente 23/06/2026 (ci(message-service) — Codecov montrait 41,34 % car les tests d'intégration Mongo étaient tous `t.Skip` en CI (Mongo + `MONGO_TEST_URI` réservés à report-service). Fix `ci-go.yml` : étape Mongo **dédiée message-service avec `--auth`** (les tests `ReadOnlyDB` exigent l'auth pour que les rôles soient appliqués), `MONGO_TEST_URI` avec creds root éphémères ; report-service inchangé. Validé local `go test -race ./...` vert, couverture filtrée 97,3 %. À POUSSER pour recalcul Codecov. Aucun changement de prod/test, CI only).

---

## ⚠️ CLAUDE OPERATING RULES (HIGH PRIORITY — do not weaken)

These govern *how* Claude works on this repo. They override default behavior.

1. **Read this file first** at every session start.
1b. **User preference:** before every assistant response, re-read this file; after each response/work unit, keep the relevant project docs updated.
2. **⛔ NE PAS CODER avant validation de l'utilisateur.** Pour toute feature/tâche : d'abord proposer (a) l'**architecture** et (b) **comment la feature sera implémentée**, puis **attendre l'accord explicite** avant d'écrire/modifier du code. Pas d'implémentation spontanée.
3. **Propose, then decide:** explain what you'll do before doing it; present viable approaches when several exist; confirm before any major architectural change. Analysis before code.
4. **Update the docs after any significant change** (source of truth, not re-explanation): `PROJECT_STATUS.md` (status 🔴→🟡→🟢, TODOs), `DECISIONS.md` (new decision + rationale), `ARCHITECTURE.md` (only if boundaries/flows change), `CHANGELOG.md` (one session entry; push older entries to `CHANGELOG_ARCHIVE.md`).
5. **Never hardcode secrets** — use `.env` variables.
5b. **⚠️ DB EN PROD — migrations rétrocompatibles obligatoires.** Une base peuplée existe. Toute feature ajoutant un champ contraint (enum `$jsonSchema`, `required`, index unique) DOIT fournir une **migration idempotente au boot** (dans `EnsureSchema`) qui backfill les docs existants vers une valeur valide — sinon Mongo `strict` revalide le doc complet et **rejette toute mise à jour** (vécu 2× : `visibility`, `likes_visibility`). Corollaire : champ à enum/défaut → poser un défaut explicite à la création **et** `omitempty`. Réf : `backfillVisibility` dans `profil-service/internal/database/init.go`.
6. **Each service is independent**: its own Dockerfile, DB, embedded schema.
7. When implementing a feature, **update its status** in `PROJECT_STATUS.md`.
8. Before any architectural decision, **check the evaluation criteria** (§7).
9. **Keep responses concise** — update the docs rather than re-explaining context.
10. **No self-commit / no self-push** — only propose the commit message; the user commits.
11. **graphify-first for codebase questions:** when `graphify-out/graph.json` exists, run `graphify query "<question>"` (scoped subgraph, far smaller than grep/GRAPH_REPORT.md). Use `graphify path "<A>" "<B>"`, `graphify explain "<concept>"`, `graphify-out/wiki/index.md` for navigation, `GRAPH_REPORT.md` only for broad architecture review. Read raw source only to modify/debug or when the graph lacks detail. **After modifying code, run `graphify update .`** (AST-only, no API cost).
12. **Session workflow (17/06/2026):** before each assistant reply, re-read `CLAUDE.md`; after each reply, keep it up to date if a new working rule or constraint was introduced during the exchange.
13. **Swagger discipline reinforced (17/06/2026):** if a task adds or changes routes, add/update swaggo annotations in the handlers and run `make swagger` before handing off.

---

## 1. Project purpose

**Breezy** is a 4-layer microservices social network (X/Twitter-like) for the FISA INFO A3 "Distributed App Dev" project. *WebDad* = repo name, *Breezy* = product (logo `frontend/public/logo_breezy.png`). 3 roles: User, Moderator, Administrator.

## 2. Technology stack

- **Backend** — Go 1.25 + Gin, one Go module per service (own `go.mod`), golangci-lint, `air` hot-reload, per-service `Makefile`.
- **Frontend** — Next.js 14 (App Router) + TypeScript + Tailwind + shadcn/ui (slate), npm, dev port 3000, env `NEXT_PUBLIC_API_URL` (default `http://localhost:8080`).
- **Databases** — PostgreSQL (auth, user) · MongoDB (profil, post, message, notification) · MinIO (media).
- **Ports** — frontend 3000 · api-gateway 8080 · auth 8081 · user 8082 · profil 8083 · post 8084 · message 8085 · notification 8086 · media 8087 · mail 8089 (interne, hors gateway) · MinIO API 9000 / console 9001.

## 3. Architecture summary

```
[Client] → [Frontend (Next.js + BFF)] → [API Gateway] → 7 Go services → DBs (PG / Mongo / MinIO)
```

- **All inter-service traffic goes through the API Gateway** (sole exception: server-to-server notification emission to `/internal/events`, off the gateway, secret-authenticated).
- **JWT auth:** access 15m (localStorage, Bearer to gateway) + refresh 24h (httpOnly cookie via Next BFF).
- **One datum = one service** (`username`→user, `display_name`/`visibility`→profil); aggregated views composed by the caller.
- **Each Go service owns its schema** (`EnsureSchema` at boot, idempotent).
- **Full Docker containerization** (docker-compose).

→ Full detail in **ARCHITECTURE.md**; the *why* in **DECISIONS.md**.

## 4. Coding conventions

- One service = one Go module (`internal/{config,database,models,repository,service,handlers,middleware}`), embedded schema at boot, no mounted init-db scripts.
- Mongo counters denormalized as `int32` (`$jsonSchema` declares `bsonType:"int"`).
- Frontend: business clients (`lib/{api,posts,bookmarks,messages,notifications,media}.ts`) layer over `apiFetch` (Bearer + single-flight refresh); no hardcoded URLs (`lib/config.ts`/`lib/routes.ts`).
- i18n: every UI string via `useT()`, keys `namespace.key`; FR is reference and any new user-facing key must be added to all 12 available locales (`fr/en/zh/es/pt/ru/ja/ko/ar/hi/de/it`).
- Identity (avatar/name) is always clickable → the person's profile, with a web hover preview.
- Pure, testable functions for non-trivial logic — Go tests + vitest.
- Secrets only via `.env` (root = cross-cutting + compose-interpolable; `<service>/.env` = own config).

## 5. Frequently used commands

```bash
make env            # create .env files from .example (run before first make dev)
make dev            # full stack with hot-reload (air + next dev)   ← develop with this
make up             # frozen prod images   ·   make build  # rebuild prod images
make logs-<svc> / make sh-<svc>
# per service:  make run | make build | make lint | make test
# frontend:     npm run dev | npm run lint | npx tsc --noEmit | npm test
# graph:        graphify query "<q>" | graphify path "<A>" "<B>" | graphify explain "<c>" | graphify update .
```

## 6. Critical architectural rules (non-negotiable)

- Everything through the gateway (incl. media download, streamed — **never** presigned MinIO URLs).
- Security never depends on the front: post-service applies the visibility barrier server-side.
- Messaging server is **blind** for DM/groups (E2EE, admin-proof); communities are admin-readable (server key) by design. Read/unread state is server-side metadata only (never the `ciphertext`).
- Media-service is content-agnostic (opaque bytes); encrypted attachments stay E2EE end-to-end.
- Notification emission is best-effort fire-and-forget (off the gateway, `INTERNAL_EVENT_SECRET`).

## 7. Reference — evaluation criteria (grading grid)

Check before architectural decisions. Scale: **A=5 / B=4 / C=2 / D=1 pts**. Team: Zaid, Perujan, Candis, Théo.

- **Group — Deliverable (report):** needs analysis · architecture diagram · prioritization (primary+secondary, justified) · planning (Trello/Gantt, delays explained) · methodology · interface (wireframe + final, UX) · features presentation (limitations explicit) · improvements & outlook (prioritized, effort estimates) · writing.
- **Group — Defense (oral):** context & approach · justified choices · coherent microservices architecture · **security (JWT sessions, protected routes)** · **full containerization** · all primary features functional · several well-chosen secondary features · **all 3 roles functional** · pro slides + scripted demo · timing & energy.
- **Individual:** technical mastery (full block skills, strong Q&A) · **English** during the defense.

## 8. API documentation rules

- Annotations live **in each service's handlers** (swaggo/swag code-first). Never in separate DTO files.
- **`make swagger` before every commit** that touches a handler or route — regenerates `doc/openapi.{json,yaml}`.
- CI drift check (`ci-go.yml` job `swagger`) blocks PRs where `doc/` is stale.
- **Published doc:** `https://gosyfrone.github.io/WebDad/` (Redoc, auto-deployed on `push develop`). Local preview: `make swagger-site` → `http://localhost:8088`.

---

*Lean entry point. Status, decisions, architecture detail and session history live in the linked files. Keep them updated per the Operating Rules; do not let this file regrow past ~150 lines.*
