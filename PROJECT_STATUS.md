# PROJECT STATUS — WebDad / Breezy

> Current state only. Historical progression lives in `CHANGELOG.md` / `CHANGELOG_ARCHIVE.md`.
> Update status markers (🔴 todo → 🟡 WIP → 🟢 done) after each significant change.
> **Team:** Zaid, Perujan, Candis, Théo.

---

## Services

| Service | Status | DB | Current state |
|---|---|---|---|
| Auth | 🟡 WIP | PostgreSQL | `/auth/{register,login,refresh,logout,validate}` + `/health`. JWT HS256 + bcrypt, access 15m + refresh 24h (rotation, revoke). Autonomous schema + optional admin seed. **TODO:** Go tests, refresh-reuse detection, email verification (hard-block login) + password reset (opaque tokens, `account_tokens`). |
| User | 🟢 OK (v1) | PostgreSQL | repo/service/handlers, autonomous schema (`users`+`follows`+`follow_requests`). CRUD users, follow (public edge / private `pending`→accept/reject), followers/following lists+counts, search/suggestions, lazy provisioning on `/users/me`. **TODO:** repo integration tests. |
| Post | 🟢 OK (v1) | MongoDB | Autonomous. CRUD posts with **profile-visibility-filtered reads**, reposts/quotes, profile pin (hidden in feeds), likes, 2-level threaded comments, denormalized int32 counters, **media `[]MediaRef` (cap 4)**, **bookmark collections + burst model**. **TODO:** post edit front (back ready). |
| Profil | 🟢 OK (v1) | MongoDB | Autonomous. Owns decorative fields + **`visibility`**. `GET/PATCH /profils/me`, `POST` (unique creation), search, admin delete. `birth_date` set-once, display_name cooldown baseline. Avatar/banner upload wired. **TODO:** front aggregated read. |
| Message | 🟢 OK | MongoDB | E2EE (DM/groups/communities), blind server, X25519 keys, cursor pagination, WS, **encrypted attachments** (no schema change), server-side read cursor + app-wide unread badge, per-conversation mute. Back + UX complete. |
| Notification | 🟢 OK | MongoDB | Aggregated (Instagram-style), ingest `/internal/events`, types like/comment/reply/mention/repost/quote/follow_request/message_mention/post_deleted, JWT API + WS. |
| Mail | 🟡 WIP | — | **Phase 0 OK** : module Go autonome (8089), `POST /internal/send` (hors gateway, `MAIL_INTERNAL_SECRET`, best-effort) + `/health`, transport SMTP (`net/smtp`) ou repli console en dev. Symétrique de notification. **TODO Phase 1 :** intégration auth→mail (vérif e-mail + reset). |
| Media | 🟢 OK | MinIO | Cross-cutting opaque storage, autonomous bucket. `POST /media` (sniff+caps), `POST /media/encrypted` (E2EE blob), public `GET /media/:id` (Range/seek), owner/admin delete. Wired on profils/posts/messages. |
| API Gateway | 🟡 WIP | — | stdlib reverse proxy, prefix routing, WS proxy, CORS, media streaming. `/internal/events` not routed (server-to-server). **TODO:** JWT middleware to protect prefixes. |
| Frontend | 🟡 WIP | — | Next.js 14, X-style responsive layout, refresh-token auth, business clients over `apiFetch`. Wired: feed/posts (like/comments/pin/repost/quote), bookmark collections, hydrated profile + privacy, follow pending/accept/reject, Explorer + search history, E2EE messaging + attachments, notifications (badge+WS), translation, i18n FR/EN, dark mode, legal pages, muted words, @mentions, image/video upload (lightbox, Twitter-style autoplay). **TODO:** real role (admin placeholder), post edit. |

## Features

| Feature | Type | Status |
|---|---|---|
| Registration / Login | Primary | 🟢 End-to-end (UI→BFF→gateway→auth), provisioning, username pre-check, logout + session guard. |
| JWT auth + protected routes | Primary | 🟢 access 15m + refresh 24h + `/auth/validate`, front single-flight refresh + `(app)` guard. **TODO:** gateway JWT middleware. |
| Role management (User/Mod/Admin) | Primary | 🟡 Role in JWT, user-service enforces admin delete, role-based nav. **TODO:** generalize to other services, real role front. |
| Post creation/reading | Primary | 🟢 End-to-end, infinite feed (For you / Following), visibility-filtered, likes, threaded comments, reposts/quotes, pin, emoji, images/videos. **TODO:** post edit. |
| User profile | Primary | 🟡 View + edit + by-username page, private-account locking, real avatar/banner upload. **TODO:** wire aggregated user+profil+post read. |
| Social graph (follow/followers) | Secondary | 🟢 follow + lists + counts + private requests + follower removal, front wired, e2e tested. |
| Search / Explorer (accounts) | Secondary | 🟢 user + profil search, "Who to follow", per-account local search history. **TODO:** post search. |
| Automatic post translation | Secondary | 🟢 BFF `/api/translate`, conservative client gate, posts + comments, cache, toggle. |
| Muted words in feed | Secondary | 🟢 `/parametres`, per-account local persistence, feed masks others' matching posts. |
| Private encrypted messaging (E2EE) | Secondary | 🟢 End-to-end (back + UX). DM/groups admin-proof, hybrid communities, server-side read state + app-wide badge, per-conversation mute, encrypted attachments. |
| Real-time notifications | Secondary | 🟢 End-to-end (e2e live), Instagram aggregation, + `message_mention` + `follow_request`, WS, badge, detail page. |
| Mentions (@handle) | Secondary | 🟢 End-to-end: posts/comments autocomplete + clickable render; messages (E2EE) member-ids only → `message_mention`. i18n FR/EN. |
| Bookmarks (collections) | Secondary | 🟢 End-to-end, collections (many-to-many) + non-deletable default, burst model, `/signets` page. |
| Image / video upload | Secondary | 🟢 Complete (Phases 0→3): media-service + MinIO + gateway `/media`; avatar/banner; post media; encrypted message attachments. |
| GIFs (Tenor/Giphy) | Secondary | 🔴 TODO (plan in DECISIONS.md). |
| Vérification e-mail | Secondary | 🔴 TODO (plan défini — DECISIONS.md). Blocage dur login non-vérifié, mail Gmail SMTP, tokens opaques 24h. |
| Mot de passe oublié | Secondary | 🔴 TODO (plan défini — DECISIONS.md). Anti-énumération, token reset 1h, révocation des sessions. |
| Moderation | Secondary | 🔴 TODO |
| Admin panel | Secondary | 🔴 TODO |
| Internationalization (FR/EN) | Secondary | 🟢 Home-grown, all UI translated, localized dates. **TODO:** per-account persistence. |

## Infrastructure

- [x] `docker-compose.yml` with all services
- [x] Persistent volumes for DBs
- [x] `.env` for secrets (gitignored, never commit)
- [x] README with setup
- [x] CI/CD: 3 GitHub Actions workflows (`ci-go` build+test -race + golangci-lint + govulncheck;
      `ci-frontend` lint+build; `ci-integration` docker stack + healthchecks). govulncheck = 0 vuln.
- [x] **API documentation:** Swagger/OpenAPI spec — `make swagger` génère `doc/openapi.{json,yaml}` (58 routes, 7 services annotés swaggo/swag code-first). Spec agrégé commité. Per-service `docs/` gitignorés.
- [x] **CI swagger job** (`ci-go.yml` job `swagger`) : drift check (`git diff --exit-code doc/`) + validation Swagger 2.0 (`go-swagger validate`). Bloquant sur PR.
- [x] **API docs GitHub Pages** (`pages.yml`) : Redoc UI déployée sur `https://gosyfrone.github.io/WebDad/` à chaque push develop. Source = `doc/` (openapi.json + index.html). Local : `make swagger-site` (port 8088).
- [ ] **Perspective:** gitleaks, Dependabot, CD (push images to GHCR).

## Open issues / TODO (active)

- **Gateway JWT middleware** — protect prefixes at login (validate locally with shared `JWT_SECRET`).
- **Real role / username (frontend)** — `(app)` layout still uses `PLACEHOLDER_ROLE = 'administrator'`;
  derive real `role`/`username` (via `/users/me` or JWT decode).
- **Identity cooldowns** — architecture posed; activate by setting `DISPLAY_NAME_CHANGE_COOLDOWN` /
  `USERNAME_CHANGE_COOLDOWN` (e.g. `168h`), no migration.
- **user-service repo integration tests** — current Go tests cover middleware/validation, not SQL
  (validated by manual e2e). Add `dockertest`/`testcontainers` for CI repo coverage.
- **Private follow/privacy review points:** (1) user-service `/internal/...is-following...` doesn't check
  `X-Internal-Secret` (graph leak if port reachable); (2) profile `followOverride=false` after a pending
  request can hide later real-time accept until remount; (3) `NotificationsView` accept/reject relies on WS
  to clean the item; (4) `post-service.visiblePage` may scan many filtered private posts (perf/DoS watch).

## Known assumed compromises

- **Access token in localStorage** (XSS-exposed, mitigated by 15m lifetime; refresh stays httpOnly).
- **CORS:** data calls go browser→gateway directly (`Authorization: Bearer`); verify `CORS_ALLOWED_ORIGINS`.
- **Followers/following counters calculated (COUNT)** per read — fine at project scale; denormalize if load requires.
- **Messaging:** communities are admin-readable (server key); identity key is per device (new device can't decrypt
  old history → "key unavailable" banner). Perspectives: key rotation, ownership transfer, backup passphrase.
- **Notifications:** emission best-effort (lost if notif-service down at T); in-memory badge may over-count by 1
  (self-corrects); badge increments even while `/notifications` is open; `last_actor` cosmetic blur after retract.

## Setup reminders (fresh checkout)

- `make env` (or copy each `.env.example`) before `make dev` — several services need their `.env` to exist,
  notably `message-service/.env`, and `INTERNAL_EVENT_SECRET` must be in the **root** `.env` (compose interpolates
  `${...}` from the root file only).
- `minio` and `media-service` share the same `media-service/.env` (`MINIO_ROOT_USER`/`PASSWORD`). MinIO console: `http://localhost:9001`.
- For code changes to appear, use `make dev` (hot-reload) — `make up` runs frozen prod images.
