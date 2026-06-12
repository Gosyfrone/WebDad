# CONTEXT — Breezy (WebDad) · Distributed App Dev Project (FISA INFO A3)

> Entry point for Claude Code. Read this first at every session start.
> Detail lives in dedicated files (keep this one lean):
> - **[ARCHITECTURE.md](ARCHITECTURE.md)** — system architecture, service boundaries, flows, critical components.
> - **[DECISIONS.md](DECISIONS.md)** — architectural decisions + rationale (defense prep).
> - **[PROJECT_STATUS.md](PROJECT_STATUS.md)** — current state, open TODOs, assumed compromises.
> - **[CHANGELOG.md](CHANGELOG.md)** — session work log · **[CHANGELOG_ARCHIVE.md](CHANGELOG_ARCHIVE.md)** — resolved issues / debugging history.
> - **[PROMPTING.md](PROMPTING.md)** — token-efficient workflow + prompt templates (read on demand, NOT auto-loaded; user may say *"from PROMPTING.md, write me the prompt for: …"*).
> - Knowledge graph in `graphify-out/` — query it before reading source (see Operating Rules).
> - Latest session note: 12/06/2026 — feed right-sidebar `Qui suivre` now shows full display names in compact mode; details in `CHANGELOG.md`.

*Last Codex sync: 12/06/2026 — full display-name visibility in feed right-sidebar Who-to-follow implemented and documented.*

---

## ⚠️ CLAUDE OPERATING RULES (HIGH PRIORITY — do not weaken)

These govern *how* Claude works on this repo. They override default behavior.

1. **Read this file first** at every session start.
2. **⛔ NE PAS CODER avant validation de l'utilisateur.** Pour toute feature/tâche :
   d'abord proposer (a) l'**architecture** et (b) **comment la feature sera implémentée**,
   et **attendre l'accord explicite** avant d'écrire/modifier du code. Pas d'implémentation
   spontanée.
3. **Propose, then decide:** explain what you are going to do before doing it; when there are
   several viable approaches, present the implementation choices; ask for confirmation before
   any major architectural change. Prefer analysis before coding.
4. **Update the docs after any significant change** — keep them the source of truth instead of
   re-explaining context:
   - `PROJECT_STATUS.md` (status markers 🔴→🟡→🟢, TODOs),
   - `DECISIONS.md` (new architectural decision + rationale),
   - `ARCHITECTURE.md` (only if boundaries/flows change),
   - `CHANGELOG.md` (one session entry; push older entries to `CHANGELOG_ARCHIVE.md`).
5. **Never hardcode secrets** — use `.env` variables.
6. **Each service is independent**: its own Dockerfile, its own DB, its own embedded schema.
7. When implementing a feature, **update its status** in `PROJECT_STATUS.md`.
8. Before any architectural decision, **check the evaluation criteria** (Reference section below).
9. **Keep responses concise** — update the docs rather than re-explaining context.
10. **No self-commit / no self-push** — only propose the commit message; the user commits.
11. **graphify-first for codebase questions:** when `graphify-out/graph.json` exists, run
    `graphify query "<question>"` (scoped subgraph, far smaller than raw grep / GRAPH_REPORT.md).
    Use `graphify path "<A>" "<B>"` for relationships, `graphify explain "<concept>"` for a concept,
    `graphify-out/wiki/index.md` for broad navigation, and `GRAPH_REPORT.md` only for broad
    architecture review. Read raw source only to modify/debug or when the graph lacks detail.
    **After modifying code, run `graphify update .`** to keep the graph current (AST-only, no API cost).
12. **Session note — follow notifications:** implemented 10/06/2026 after user validation. Public
    direct follows emit aggregated `follow` notifications; private follow accept/reject keeps the
    existing `follow_request` flow and does not emit `follow`.
13. **Session note — private follow accept UX:** implemented 10/06/2026. Rejecting a private follow
    request only retracts the owner's `follow_request`; accepting retracts it, creates a persisted
    `follow_request_accept_confirm` notification for the owner, and sends a persisted
    `follow_request_accepted` notification to the requester.

---

## 1. Project purpose

**Breezy** is a 4-layer microservices social network (X/Twitter-like) built for the FISA INFO A3
"Distributed App Dev" project. *WebDad* = repo/project name, *Breezy* = product name
(logo `frontend/public/logo_breezy.png`). 3 roles: User, Moderator, Administrator.

## 2. Technology stack

**Backend** — Go 1.25 + Gin, one independent Go module per service (own `go.mod`), golangci-lint,
`air` hot-reload, per-service `Makefile` (`make run/build/lint/test`).

**Frontend** — Next.js 14 (App Router) + TypeScript + Tailwind + shadcn/ui (slate theme), npm,
dev port 3000, env `NEXT_PUBLIC_API_URL` (default `http://localhost:8080`).

**Databases** — PostgreSQL (auth, user) · MongoDB (profil, post, message, notification) · MinIO (media).

**Ports** — frontend 3000 · api-gateway 8080 · auth 8081 · user 8082 · profil 8083 · post 8084 ·
message 8085 · notification 8086 · media 8087 · mail 8089 (interne, hors gateway) · MinIO API 9000 / console 9001.

## 3. Architecture summary

```
[Client] → [Frontend (Next.js + BFF)] → [API Gateway] → 7 Go services → DBs (PG / Mongo / MinIO)
```

- **All inter-service traffic goes through the API Gateway** (sole exception: server-to-server
  notification emission to `/internal/events`, off the gateway, secret-authenticated).
- **JWT auth:** access 15m (localStorage, Bearer to gateway) + refresh 24h (httpOnly cookie via Next BFF).
- **One datum = one service** (e.g. `username`→user-service, `display_name`/`visibility`→profil-service);
  aggregated views composed by the caller.
- **Each Go service owns its schema** (`EnsureSchema` at boot, idempotent).
- **Full Docker containerization** (docker-compose).

→ Full detail in **ARCHITECTURE.md**; the *why* behind each choice in **DECISIONS.md**.

## 4. Coding conventions

- One service = one Go module (`internal/{config,database,models,repository,service,handlers,middleware}`),
  embedded schema applied at boot, no mounted init-db scripts.
- Mongo counters denormalized as `int32` (`$jsonSchema` declares `bsonType:"int"`).
- Frontend: business clients (`lib/{api,posts,bookmarks,messages,notifications,media}.ts`) layer over
  `apiFetch` (Bearer + single-flight refresh inherited); no hardcoded URLs (`lib/config.ts`/`lib/routes.ts`).
- i18n: every UI string via `useT()`, keys `namespace.key`, FR is the reference, EN parity required.
- Identity (avatar/name) is always clickable → the person's profile, with a web hover preview.
- Pure, testable functions for non-trivial logic (e.g. `planUpdate`, `convLess`, `withinSessionWindow`,
  `computeDivider`, mention/translation helpers) — Go tests + vitest.
- Secrets only via `.env` (root = cross-cutting + interpolable by compose; `<service>/.env` = own config).

## 5. Frequently used commands

```bash
make env            # create .env files from .example (run before first make dev)
make dev            # full stack with hot-reload (air + next dev, bind-mounts)   ← use this to develop
make up             # frozen prod images (no rebuild on source change)
make build          # rebuild prod images
make logs-<svc> / make sh-<svc>
# per service:
make run | make build | make lint | make test
# frontend:
npm run dev | npm run lint | npx tsc --noEmit | npm test
# graph:
graphify query "<question>" | graphify path "<A>" "<B>" | graphify explain "<concept>" | graphify update .
```

## 6. Critical architectural rules (non-negotiable)

- Everything through the gateway (incl. media download, streamed — **never** presigned MinIO URLs).
- Security never depends on the front: post-service applies the visibility barrier server-side.
- Messaging server is **blind** for DM/groups (E2EE, admin-proof); communities are admin-readable
  (server key) by design. Read/unread state is server-side metadata only (never the `ciphertext`).
- Media-service is content-agnostic (opaque bytes); encrypted attachments stay E2EE end-to-end.
- Notification emission is best-effort fire-and-forget (off the gateway, `INTERNAL_EVENT_SECRET`).

## 7. Reference — evaluation criteria (grading grid)

Check before architectural decisions. Scale: **A=5 / B=4 / C=2 / D=1 pts**. Team: Zaid, Perujan, Candis, Théo.

**Group — Deliverable (report):** needs analysis · architecture diagram · prioritization (primary+secondary,
justified) · planning (Trello/Gantt, delays explained) · methodology · interface (wireframe + final, UX) ·
features presentation (limitations explicit) · improvements & outlook (prioritized, effort estimates) · writing.

**Group — Defense (oral):** context & approach · justified choices · coherent microservices architecture ·
**security (JWT sessions, protected routes)** · **full containerization** · all primary features functional ·
several well-chosen secondary features · **all 3 roles functional** · pro slides + scripted demo · timing & energy.

**Individual:** technical mastery (full block skills, strong Q&A) · **English** during the defense.

## 8. API documentation rules

- Annotations live **in each service's handlers** (swaggo/swag code-first). Never in separate DTO files.
- **`make swagger` before every commit** that touches a handler or route — regenerates `doc/openapi.{json,yaml}`.
- The CI drift check (`ci-go.yml` job `swagger`) blocks PRs where `doc/` is stale.
- **Published doc:** `https://gosyfrone.github.io/WebDad/` (Redoc, auto-deployed on `push develop` via `pages.yml`).
- Local preview: `make swagger-site` → `http://localhost:8088`.

## 9. Current handoff

- 2026-06-12: **vue visiteur** (fil public sans inscription). Front-only, zéro backend (post-service expose déjà la lecture publique via `OptionalJWTAuth`). Nouveau `AuthPromptProvider`/`useAuthGate` (`isVisitor`/`requireAuth`/`promptLogin`, monté dans le layout `(app)`) ; `middleware.ts` ouvre `/feed`+`/posts/:id` ; `/`→`/feed`. Pièce centrale = garder tous les appels `/me` derrière `getAccessToken()` (sinon 401→refresh raté→redirection forcée /login). Actions réservées (like/repost/citer/signet/commentaire) → modale connexion ; aperçu profil au survol désactivé pour le visiteur. i18n `visitor.*`. `tsc --noEmit` propre. Aucun changement backend → Swagger non régénéré. Détails CHANGELOG + DECISIONS « Vue visiteur ».
- 2026-06-12: suggested commit message: `feat(visitor): vue visiteur du fil public sans inscription`.
- 2026-06-10: login page now uses one "Adresse e-mail ou Username" field; username login is resolved by the Next BFF through user-service, then auth-service authenticates by `user_id`.
- 2026-06-10: Swagger annotation/docs updated for the new login contract; local `make swagger` still fails on Windows bash, so the aggregate spec was regenerated in a temporary Go Linux container with `GOBIN` aligned to the script.
- 2026-06-10: suggested commit message for this change: `feat(auth): allow login with email or username`.
- 2026-06-10: register birth date input now clamps future dates to today's local date across browsers/mobile pickers, while keeping the 13+ validation at submit time.
- 2026-06-10: suggested commit message for this change: `fix(auth): cap register birth date to today`.
- 2026-06-10: quote/repost media rendering implemented on the frontend; quoted posts now show image/video previews, and Docker exposes the app on port 3000.
- 2026-06-10: suggested commit message for this change: `fix(posts): show quoted media in quotes`.
- 2026-06-10: comment media support implemented end-to-end; comments/replies accept up to 4 uploaded images/videos/GIFs via media-service and render them in `CommentSection`.
- 2026-06-10: desktop quote dialogs now scroll when quoted media plus newly attached media exceed the viewport height.
- 2026-06-10: suggested French commit message for the latest changes: `feat(commentaires): ajouter les medias et le scroll des citations`.
- 2026-06-10: Swagger annotation/docs updated for comment media; `doc/openapi.{json,yaml}` regenerated through the aggregation script in a temporary Go Linux container because local Windows `make swagger` cannot launch WSL bash.
- 2026-06-10: suggested commit message for the Swagger/doc update: `docs(swagger): mettre a jour les annotations et la spec OpenAPI des commentaires`.
- 2026-06-11: hashtag/trends feature implemented. post-service extracts normalized hashtags on create/update, exposes `GET /posts?hashtag=...` and `GET /posts/trends` with visibility-aware trend counts; frontend renders clickable hashtags, filters the feed through `/feed?hashtag=...`, and loads dynamic right-column trends. Swagger regenerated via Linux Go container because local `make swagger` still fails to launch Windows bash. Vérifs: post-service `go test ./...`, frontend `npm run build`.
- 2026-06-11: local stack left running on `localhost:3000` (`webdad-frontend-1`) with `post-service` restarted healthy; `/login` returns 200, `/feed` redirects 307 to login when unauthenticated, and gateway `/posts/trends` returns 200. `graphify update .` could not run because `graphify` is not installed on this Windows host.
- 2026-06-11: suggested commit message: `feat(posts): add hashtag filtering and trends`.
- 2026-06-11: suggested French commit message: `feat(posts): ajouter le filtrage par hashtag et les tendances`.
- 2026-06-11: hashtag UX follow-up implemented after validation. Trends remain Top 5 and link to `tab=top`; `/feed?hashtag=...` now shows a search-style hashtag header with back button plus `À la une` / `Récent` / `Média` tabs. Backend `GET /posts` accepts `sort=top|recent`; `top` orders by engagement counters then recency. Media tab renders only media attachments from hashtag posts. Swagger regenerated through Linux Go container. Vérifs: frontend `npm run build`, post-service `go test . ./internal/...` OK; full `go test ./...` still blocked by generated `docs` package missing `github.com/swaggo/swag` in `go.mod`.
- 2026-06-11: runtime check after hashtag UX follow-up: restarted `webdad-post-service-1` and `webdad-frontend-1`; `http://localhost:3000/login` returns 200, `/feed?hashtag=test&tab=media` redirects 307 to login when unauthenticated, and gateway `GET /posts?hashtag=test&sort=top&limit=1` returns 200. `graphify update .` still unavailable because `graphify` is not installed.
- 2026-06-11: suggested French commit message: `feat(feed): ajouter les onglets de resultats hashtag`.
- 2026-06-11: hashtag/search UX follow-up implemented after validation. The hashtag results header now puts the back arrow left of the search field; shared search routing sends `#tag` to `/feed?hashtag=...&tab=top` and plain text to Explorer profile search. Explorer also handles `#hashtag` queries with a direct hashtag result. `À la une` sorts by likes, reposts, comments, then recency; the right Trends block spacing was tightened so the Top 5 stays visible. Swagger regenerated via Linux Go container. Vérifs: post-service `go test . ./internal/...`, frontend `npm run build`.
- 2026-06-11: runtime check after hashtag/search UX follow-up: restarted `webdad-post-service-1` and `webdad-frontend-1`; `http://localhost:3000/login` returns 200, `/feed?hashtag=test&tab=top` redirects 307 to `/login` when unauthenticated, gateway `GET /posts?hashtag=test&sort=top&limit=1` returns 200, and Docker exposes `webdad-frontend-1` on `0.0.0.0:3000->3000/tcp`. `graphify update .` still unavailable because `graphify` is not installed.
- 2026-06-11: hashtag results search dropdown implemented after validation. On `/feed?hashtag=...`, typing in the top search bar opens an X-style dark dropdown: first 3 matching hashtag trends from `GET /posts/trends?q=...&limit=3`, then profile suggestions. Clicking a hashtag opens `/feed?hashtag=...&tab=top`; clicking a user opens `/profil/<username>` directly. Explorer/search now finds users by `@username`, plain username, or display_name. Swagger regenerated for the new `q` query param on `/posts/trends`. Vérifs: post-service `go test . ./internal/...`, frontend `npm run build`.
- 2026-06-11: runtime check after hashtag suggestions dropdown: restarted `webdad-post-service-1` and `webdad-frontend-1`; `http://localhost:3000/login` returns 200, `/feed?hashtag=test&tab=top` redirects 307 to `/login` when unauthenticated, gateway `GET /posts/trends?q=te&limit=3` returns 200 with filtered data, and Docker exposes `webdad-frontend-1` on `0.0.0.0:3000->3000/tcp`. `graphify update .` still unavailable because `graphify` is not installed.
- 2026-06-11: right-column Trends search follow-up implemented after validation. The sidebar search now uses the same X-style dropdown as hashtag pages: while typing it shows up to 3 matching hashtag suggestions, then profiles; clicking a suggestion navigates directly and stores it in local per-user history. With an empty focused field, recent searches appear with "clear all" and per-item removal. No backend/Swagger change required. Vérif: frontend `npm run build`.
- 2026-06-11: runtime check after right-column search history follow-up: restarted `webdad-frontend-1`; `http://localhost:3000/login` returns 200 and Docker exposes `webdad-frontend-1` on `0.0.0.0:3000->3000/tcp`. `graphify update .` still unavailable because `graphify` is not installed.
- 2026-06-12: composer hashtag autocomplete implemented after validation. `PostComposer` detects `#...` under the caret, queries existing hashtag trends through `/posts/trends?q=...`, shows a hashtag suggestion popup, inserts `#tag ` with trailing space on click/Enter/Tab, and mirrors textarea content to color/underline hashtags in blue while composing. Added pure `hashtags.ts` utilities + Vitest coverage. No backend/Swagger change required. Vérifs: `npm test -- --run src/lib/hashtags.test.ts`, frontend `npm run build`.
- 2026-06-12: runtime check after composer hashtag autocomplete: restarted `webdad-frontend-1`; `http://localhost:3000/login` returns 200 and Docker exposes `webdad-frontend-1` on `0.0.0.0:3000->3000/tcp`. `graphify update .` still unavailable because `graphify` is not installed.
- 2026-06-12: Explorer discovery redesign implemented after validation. Default Explorer now shows Top 10 clickable trends, profile suggestion cards with hover profile, follow button, and best-effort "followed by ..." context, initially backed by hashtagged posts before the follow-up below switched the final block to regular feed publications. Search Explorer now shows two sections while typing: Top 3 trends then matching people. Backend post-service added `GET /posts?hashtag_any=true` for visibility-filtered hashtagged posts; Swagger regenerated. Vérifs: post-service `go test . ./internal/...`, frontend `npm run build`.
- 2026-06-12: runtime check after Explorer discovery redesign: restarted `webdad-post-service-1` and `webdad-frontend-1`; `http://localhost:3000/login` returns 200, `/explorer` redirects 307 to `/login` when unauthenticated, gateway `GET /posts?hashtag_any=true&limit=1` returns 200, and Docker exposes `webdad-frontend-1` on `0.0.0.0:3000->3000/tcp`. `graphify update .` still unavailable because `graphify` is not installed.
- 2026-06-12: Explorer follow-up implemented after validation. Suggestions now exclude profiles already followed by the current user, and the final Explorer block is renamed "Publications" and displays regular feed posts via `listFeed(10, 0)` instead of hashtag-only posts. No backend/Swagger change required. Vérif: frontend `npm run build`; `graphify update .` still unavailable because `graphify` is not installed.
- 2026-06-12: frontend dev cache repair after React Client Manifest / broken CSS issue. Removed generated `frontend/.next` after verifying the resolved path stayed inside the workspace, restarted `webdad-frontend-1`, and confirmed `http://localhost:3000/login` plus all current `_next/static` CSS/JS assets return 200. Recent logs no longer show the manifest error or asset 404s; only non-blocking webpack dev cache warnings remain.
- 2026-06-12: Explorer submitted search results implemented after validation. Pressing Enter now opens a results mode with a filter block above results, checkboxes for Publications (default on) and Utilisateurs, at least one required, showing matching post sections and/or matching users. Live search suggestions now always offer "Rechercher ..." and "Aller à @..." actions, and show "Aucun résultat pour ..." when no trend/profile matches. No backend/Swagger change required. Vérifs: frontend `npm run build`; cleaned generated `frontend/.next` after build, restarted `webdad-frontend-1`, `/explorer` redirects 307 to `/login` unauthenticated, and current `_next/static` CSS/JS assets return 200. `graphify update .` still unavailable because `graphify` is not installed.
- 2026-06-12: Explorer search polish implemented after validation. Submitted-search filters are now compact pills aligned top-right in the results header, the live search dropdown uses an opaque background with stronger shadow, and fallback actions "Rechercher ..." / "Aller à @..." only appear when no trend or person suggestion exists. No backend/Swagger change required. Vérifs: frontend `npm run build`; cleaned generated `frontend/.next` after build, restarted `webdad-frontend-1`, `/explorer` redirects 307 to `/login` unauthenticated, and current `_next/static` CSS/JS assets return 200. `graphify update .` still unavailable because `graphify` is not installed.
- 2026-06-12: Explorer filter placement follow-up implemented after validation. The Publications/Utilisateurs filter row now appears right-aligned directly under the search bar when a submitted search is active, above the results/trends content; the results header only shows the active query title. No backend/Swagger change required. Vérifs: frontend `npm run build`; cleaned generated `frontend/.next` after build, restarted `webdad-frontend-1`, `/explorer` redirects 307 to `/login` unauthenticated, and current `_next/static` CSS/JS assets return 200. `graphify update .` still unavailable because `graphify` is not installed.
- 2026-06-12: pending validation for Explorer filter placement correction: desktop should render the whole Publications/Utilisateurs filter block in the right sidebar above the Top 5 trends block; mobile should hide it behind a three-dots control with a "Filtrer" menu/dialog exposing the same two checkbox filters.

- 2026-06-12: Explorer right-sidebar/mobile filter correction implemented after validation. Desktop filter block moved to the right sidebar above the Top 5 trends via a shared Explorer filter context; mobile now uses a three-dots dropdown with "Filtrer" and the same Publications/Utilisateurs checkboxes. No backend/Swagger change required. Vérif: frontend `npm run build` OK. Post-build cleanup of generated `frontend/.next`, frontend restart, runtime asset checks, and `graphify update .` could not run on this pass because the approval layer rejected the escalated command due to usage limits.
- 2026-06-12: user rule reinforced: reread `CLAUDE.md` before each response and keep it updated after responses/work. Right-sidebar feed correction implemented after validation: sidebar now uses dynamic viewport height with internal scroll and safe bottom padding; Trends Top 5 items are compact grid rows; Who-to-follow uses compact rows so 3 suggestions fit at zoom and the initial trio stays stable after clicking Follow, with the button label `Suivi`. Front-only; no backend route/handler change, so Swagger not required. Vérif: frontend `npm run build` OK. Suggested commit message: `fix(feed): stabiliser la sidebar droite au zoom`.
- 2026-06-12: regression follow-up on the previous sidebar change: compact styling is now scoped back to the right sidebar only, restoring the default shared `UserListItem` layout outside `Qui suivre`; sidebar keeps vertical scroll and the stable `Suivi` button. Front-only; no backend route/handler change, so Swagger not required. Vérif: frontend `npm run build` OK. Suggested commit message: `fix(feed): corriger la regression CSS de la sidebar`.
- 2026-06-12: pending validation for the next UI follow-up: make the two right-sidebar cards truly responsive in their internal layout and align Explorer trend rows with feed trend rows by moving the post count to the far right on the main hashtag line. Planned scope is frontend-only; no backend route/handler change, so no Swagger expected.
- 2026-06-12: responsive right-sidebar cards and Explorer trend alignment implemented after validation. Feed right sidebar now uses a more adaptive width, trend rows keep the post count on the far right, and `Qui suivre` stabilizes the follow button width in compact mode. Explorer trend rows now match the feed layout with rank above, hashtag left, and post count right. Front-only; no backend route/handler change, so no Swagger required. Vérif: frontend `npm run build` OK. Suggested commit message: `fix(explorer): aligner les tendances et rendre la sidebar responsive`.
- 2026-06-12: pending validation for the next UI follow-up: in feed right-sidebar `Qui suivre`, show the full `displayName` instead of truncating it to one line, while keeping the follow button fixed on the right and avoiding layout regressions. Planned scope is frontend-only; no backend route/handler change, so no Swagger expected.
- 2026-06-12: full display-name visibility in feed right-sidebar `Qui suivre` implemented after validation. Compact mode no longer truncates the `displayName` to one line; it can wrap on up to 2 lines while keeping the follow button fixed on the right. Front-only; no backend route/handler change, so no Swagger required. Vérif: frontend `npm run build` OK. Suggested commit message: `fix(feed): afficher le nom complet dans qui suivre`.

---

*This file is the lean entry point. Current status, decisions, architecture detail and history live in the
linked files. Keep them updated per the Operating Rules; do not let this file regrow past ~250 lines.*
