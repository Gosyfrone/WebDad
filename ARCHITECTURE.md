# ARCHITECTURE — WebDad / Breezy

> System architecture, service boundaries, communication flows, critical components.
> Derived primarily from `graphify-out/GRAPH_REPORT.md` (2018 nodes · 4544 edges · 133 communities).
> For a scoped subgraph, prefer `graphify query "<question>"` over reading source files.

---

## 1. Overview

**Breezy** (product name; *WebDad* = repo/project name) is a 4-layer microservices
social network built for the FISA INFO A3 "Distributed App Dev" project.

```
[Client]   User / Moderator / Administrator
   ↓
[Web]      Frontend (Next.js 14, App Router, TS)
   ↓
[Services] API Gateway → Auth | User | Profil | Post | Message | Notification | Media
   ↓
[Data]     PostgreSQL (Auth, User) | MongoDB (Profil, Post, Message, Notification) | MinIO (Media)
```

**Cardinal rules**
- All inter-service communication goes **through the API Gateway** (the one exception:
  server-to-server notification emission to `notification-service:8086/internal/events`,
  which is intentionally *not* routed through the gateway).
- JWT auth (Auth Service) + protected routes; 3 roles: User, Moderator, Administrator.
- One service = one independent Go module (own `go.mod`, own Dockerfile, own DB/schema).
- Each Go service **owns and maintains its own schema** at boot (`EnsureSchema`, idempotent,
  `collMod` resync for Mongo) — no mounted `init-db` scripts.

## 2. Services & ownership

| Service | Port | DB | Owns |
|---|---|---|---|
| api-gateway | 8080 | — | Thin reverse proxy (stdlib `httputil.ReverseProxy`), CORS, prefix routing, WS proxy, media streaming |
| auth-service | 8081 | PostgreSQL | Credentials, JWT (HS256), refresh tokens (opaque, SHA-256 hashed, rotated) |
| user-service | 8082 | PostgreSQL | Identity: `username` (immutable handle), `is_active`, social graph (follows + follow_requests + counts) |
| profil-service | 8083 | MongoDB | Decorative/editable fields: `display_name`, `bio`, avatar/banner, website, location, birth_date, gender, nationality (ISO alpha-2), **`visibility`** |
| post-service | 8084 | MongoDB | Posts, hashtags/trends, comments (threaded 2 levels), likes, reposts/quotes, pins, polls/votes, bookmarks (collections), post media refs |
| message-service | 8085 | MongoDB | E2EE messaging (DM/groups/communities), conversations, members, encrypted messages, WS |
| notification-service | 8086 | MongoDB | Aggregated notifications (Instagram-style), ingest `/internal/events`, WS |
| media-service | 8087 | MinIO | Opaque byte storage (avatar/banner, post media, encrypted attachments) |
| frontend | 3000 | — | Next.js UI + BFF route handlers (`/api/auth/*`, `/api/translate`, `/api/countries`, provisioning) |

**Data-ownership invariant:** one datum = one service. `username` lives in user-service,
`display_name`/`visibility` in profil-service → no backend join; aggregated views are
composed by the **caller** (front/BFF).

## 3. Communication flows

- **Client → Gateway → Service.** Gateway forwards method/path/body/headers and returns the
  response intact; prefix is preserved (`/auth/...` → auth-service `/auth/...`).
- **Auth:** access token (5m, localStorage, sent as `Authorization: Bearer` by the client
  directly to the gateway) + refresh token (24h, httpOnly cookie `breezy-refresh`, managed by
  the Next BFF same-origin). Login accepts email directly; username login is resolved by the
  BFF through `GET /users/by-username/:username`, then auth-service checks credentials by
  `user_id`. Refresh is single-flight on 401.
- **Registration gate:** local registration requires `acceptedTerms=true` in the Next BFF. OAuth
  sign-up exchanges/verifies the provider code, then returns a short `pending_token` when no
  account exists yet; the callback stores it in `sessionStorage` and redirects to the blocking
  public `/auth/oauth/terms` page. That page collects username/date/CGU and only then creates
  the auth credential, user row, profile and session.
- **Credential settings:** password changes reuse the authenticated BFF route and rotate the session.
  Email changes use `pending_email` plus a one-use 24h token sent to the new address; confirmation
  atomically promotes it, revokes prior refresh tokens, and opens a session carrying the new email.
- **Cross-service reads (privacy):** post-service calls profil-service (`visibility`) and
  user-service (follow status) to filter post visibility; clients have no-op fallbacks for autonomy.
- **Notifications (server→server):** post/message/user-service POST best-effort fire-and-forget
  events to notification-service `/internal/events` (secret `INTERNAL_EVENT_SECRET`, off the gateway).
- **Auto-moderation (server→server):** when a post crosses the admin-set report threshold, report-service
  POSTs best-effort to post-service `/internal/posts/:id/auto-hide` (and `…/auto-unhide` on approval),
  off the gateway, authenticated by `X-Internal-Secret` (same shared secret). Post-service owns the
  `auto_hidden` visibility flag (server-side barrier); report-service owns the count and the validation
  lock (terminal `approved` status). Bug tickets are exempt.
- **Realtime:** WebSocket hubs per user for messages (`/messages/ws`) and notifications
  (`/notifications/ws`), plus a **broadcast** hub on post-service (`/posts/ws`) that pings all
  connected clients on each new public root post (id + author only → "X a posté" banner, content
  refetched via the normal feed). All proxied natively by the gateway (101 upgrade).
- **Media download:** streamed through the gateway (`http.ServeContent`, Range/seek); MinIO never exposed.

## 4. Critical components (god nodes — most connected)

From GRAPH_REPORT "God Nodes":
1. `apiFetch()` (98 edges) — front HTTP client: Bearer injection, 401 catch, single-flight refresh, replay.
2. `cn()` (76) — Tailwind class-merge util.
3. `useT()` (64) / `useLanguage()` (53) — home-grown i18n (12 langues) + préférence de compte via user-service.
4. `PostService` (52) / `PostRepository` (39) — post domain core (Mongo).
5. `MessageService` (44) / `MessageRepository` (34) — E2EE messaging core (Mongo).
6. `useToast()` (38) — UI notifications.
7. `Context` (33) — shared Go `context` usage across services.

**Hyperedges (group relationships)**
- *Backend Go Microservices*: auth, user, profil, post, message, notification, media, gateway.
- *CI/CD Pipeline*: ci_go, ci_frontend, ci_integration.
- *Server-to-Server Notification Emission*: post, message, user, notification, internal_event_secret.

**No import cycles detected.**

## 5. Frontend structure

- **Route groups:** `(auth)` (public), `(app)` (authenticated, guarded by `middleware.ts`),
  `(legal)` (public, outside session guard).
- **Settings:** `/parametres` separates general preferences from user credentials; the public
  `/verify-email-change` page completes proof of the new mailbox through a same-origin BFF handler.
- **Business clients** over `apiFetch`: `lib/{api,posts,bookmarks,messages,notifications,media}.ts`.
- **App-wide providers** in `(app)/layout`: `NotificationsProvider` (badge + single WS),
  `MessagesProvider` (unread badge + single messages WS), `LanguageProvider`, `ThemeProvider`.
- **Language flow:** `LanguageProvider` écoute les changements de session, lit `preferred_locale` via `GET /users/me`, utilise
  la locale navigateur si elle est absente, et persiste les choix authentifiés via `PATCH /users/me`. L'arabe pose `dir=rtl`.
- **E2EE crypto** (`lib/crypto.ts`, `@noble/ciphers`/`@noble/curves`): X25519 sealed box +
  XChaCha20-Poly1305; identity key per device in IndexedDB.
- **Aggregation:** caller-side enrichment with memoized caches (`authorCache`, `user-cache`).

## 6. Key communities (navigation)

Notable cohesive modules from the graph (see GRAPH_REPORT for the full 133):
Post Visibility & Follow Client · Message Service Models · Bookmarks Handler · Media Service
(MinIO) · Frontend i18n & Dialogs · Notification Service Core · E2EE Crypto · BFF Provisioning
& Routes · Mentions & Translation · Post/Message/User Repository · `EnsureSchema` clusters
(one per service, high cohesion 0.7–1.0) · per-service `JWTAuth`/`Config.Load` clusters.

## 7. Target file structure

```
project/
├── CLAUDE.md            ← operating rules + project summary (entry point)
├── ARCHITECTURE.md      ← this file
├── DECISIONS.md         ← architectural decisions + rationale
├── PROJECT_STATUS.md    ← current state
├── CHANGELOG.md / CHANGELOG_ARCHIVE.md ← history
├── docker-compose.yml · docker-compose.dev.yml · Makefile · .env.example
├── frontend/
├── api-gateway/
└── {auth,user,profil,post,message,notification,media}-service/  (each: Dockerfile, go.mod, internal/)
```
