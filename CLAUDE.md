# CONTEXT — Breezy (WebDad) · Distributed App Dev Project (FISA INFO A3)

> Entry point for Claude Code. Read this first at every session start.
> Detail lives in dedicated files (keep this one lean):
> - **[ARCHITECTURE.md](ARCHITECTURE.md)** — system architecture, service boundaries, flows, critical components.
> - **[DECISIONS.md](DECISIONS.md)** — architectural decisions + rationale (defense prep).
> - **[PROJECT_STATUS.md](PROJECT_STATUS.md)** — current state, open TODOs, assumed compromises.
> - **[CHANGELOG.md](CHANGELOG.md)** — session work log · **[CHANGELOG_ARCHIVE.md](CHANGELOG_ARCHIVE.md)** — resolved issues / debugging history.
> - **[PROMPTING.md](PROMPTING.md)** — token-efficient workflow + prompt templates (read on demand, NOT auto-loaded; user may say *"from PROMPTING.md, write me the prompt for: …"*).
> - Knowledge graph in `graphify-out/` — query it before reading source (see Operating Rules).

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
message 8085 · notification 8086 · media 8087 · MinIO API 9000 / console 9001.

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
- Identity (avatar/name) is always clickable → the person's profile.
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

- 2026-06-10: quote/repost media rendering implemented on the frontend; quoted posts now show image/video previews, and Docker exposes the app on port 3000.
- 2026-06-10: suggested commit message for this change: `fix(posts): show quoted media in quotes`.
- 2026-06-10: comment media support implemented end-to-end; comments/replies accept up to 4 uploaded images/videos/GIFs via media-service and render them in `CommentSection`.
- 2026-06-10: desktop quote dialogs now scroll when quoted media plus newly attached media exceed the viewport height.
- 2026-06-10: suggested French commit message for the latest changes: `feat(commentaires): ajouter les medias et le scroll des citations`.

---

*This file is the lean entry point. Current status, decisions, architecture detail and history live in the
linked files. Keep them updated per the Operating Rules; do not let this file regrow past ~250 lines.*
