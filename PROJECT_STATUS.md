# PROJECT STATUS — WebDad / Breezy

> Current state only. Historical progression lives in `CHANGELOG.md` / `CHANGELOG_ARCHIVE.md`.
> Update status markers (🔴 todo → 🟡 WIP → 🟢 done) after each significant change.
> **Team:** Zaid, Perujan, Candis, Théo.

---

## Services

| Service | Status | DB | Current state |
|---|---|---|---|
| Auth | 🟡 WIP | PostgreSQL | `/auth/{register,login,refresh,logout,validate,verify-email/confirm,verify-email/request,password/forgot,password/reset,password/change}` + `POST /auth/users` (admin create) + `/auth/oauth/{provider}/{url,exchange}` + `/health`. JWT HS256 + bcrypt, access 15m + refresh 24h (rotation, revoke). **Admin-created accounts** : `must_change_password=true` (mot de passe temporaire mailé) + **vérif e-mail obligatoire** (`email_verified=false` en prod ; e-mail = lien de vérif + mot de passe temporaire ; le lien vérifie ET ouvre la session → feed + modale changement de mot de passe ; `VerifyEmail` propage le drapeau). Court-circuit DEV via `ADMIN_CREATE_AUTO_VERIFY=true`. Drapeau porté par le JWT + survit au refresh, levé par `POST /auth/password/change` (vérifie le pwd actuel, révoque les autres sessions, ré-émet des tokens sans drapeau). **OIDC Login with Google** (code→tokens server-side, ID token verified via JWKS/issuer/audience; `email_verified` required; generic provider registry). **Blocage dur login non-vérifié** (`403 email_not_verified`) ; register envoie le mail de vérif (best-effort) + tokens vérif opaques `account_tokens` (24h, usage unique, anti-énum sur le renvoi). **`verify-email/confirm` = auto-login** : confirme le token *et* émet une session (access + refresh, comme login ; garde compte désactivé). **Reset password** (Phase 2) : forgot anti-énum (toujours 200) + reset token 1h usage unique → bcrypt + `email_verified=true` + **révocation de toutes les sessions** (`DELETE refresh_tokens`). Autonomous schema (idempotent ALTER: `provider`/`provider_subject`, password nullable) + optional admin seed. **TODO:** Go tests, refresh-reuse detection, rate-limiting (Phase 3). *(OAuth new-user provisioning = onboarding gate côté front, cf. Frontend.)* |
| User | 🟢 OK (v1) | PostgreSQL | repo/service/handlers, autonomous schema (`users`+`follows`+`follow_requests`). CRUD users, follow (public edge / private `pending`→accept/reject), followers/following lists+counts, search/suggestions, lazy provisioning on `/users/me`. **`POST /users/admin`** (admin) provisionne l'id créé par auth ; username suffixé `_<8 hex>` + `username_pending` si pris (exposé par `/users/me`, levé sur changement effectif). **TODO:** repo integration tests. |
| Post | 🟢 OK (v1) | MongoDB | Autonomous. CRUD posts with **profile-visibility-filtered reads**, **hashtags + filtered feed + trends**, reposts/quotes, profile pin (hidden in feeds), likes, 2-level threaded comments, denormalized int32 counters, **media `[]MediaRef` (cap 4)**, **bookmark collections + burst model**. **TODO:** post edit front (back ready). |
| Profil | 🟢 OK (v1) | MongoDB | Autonomous. Owns decorative fields + **`visibility`** (`public` by default on creation, `PATCH /profils/me` to switch public/private). `GET/PATCH /profils/me`, `POST` (unique creation), search, admin delete. `birth_date` set-once, display_name cooldown baseline. Avatar/banner upload wired. **TODO:** front aggregated read. |
| Message | 🟢 OK | MongoDB | E2EE (DM/groups/communities), blind server, X25519 keys, cursor pagination, WS, **encrypted attachments** (no schema change), owner-only message edit with encrypted original preserved, server-side read cursor + app-wide unread badge, per-conversation mute, **passphrase-protected key backup (`key_backups`, zero-knowledge, multi-device)**. Back + UX complete. **TODO:** admin passphrase reset (deferred). |
| Notification | 🟢 OK | MongoDB | Aggregated (Instagram-style), ingest `/internal/events`, types like/comment/reply/mention/repost/quote/follow/follow_request/follow_request_accepted/follow_request_accept_confirm/message_mention/post_deleted, JWT API + WS. |
| Mail | 🟡 WIP | — | **Phases 0+1 OK (code)** : module Go autonome (8089), `POST /internal/send` (hors gateway, `MAIL_INTERNAL_SECRET`, best-effort) + `/health`, transport SMTP (`net/smtp`) ou repli console en dev. **Phase 1 : intégration auth→mail câblée** (register envoie le mail de vérif, client `internal/notify` best-effort). **Phase 2 : mail de reset câblé** (`sendResetMail`, lien 1h, best-effort, même client). **Templates HTML de marque** (coquille partagée `brandedEmailHTML`, clear mode, vérif + reset) ✓. 🟢 après e2e `make dev` validé (vérif + reset). **TODO Phase 3 :** rate-limiting, emails EN. |
| Media | 🟢 OK | MinIO | Cross-cutting opaque storage, autonomous bucket. `POST /media` (sniff+caps), `POST /media/encrypted` (E2EE blob), public `GET /media/:id` (Range/seek), owner/admin delete. Wired on profils/posts/messages. |
| API Gateway | 🟡 WIP | — | stdlib reverse proxy, prefix routing, WS proxy, CORS, media streaming. `/internal/events` not routed (server-to-server). **TODO:** JWT middleware to protect prefixes. |
| Frontend | 🟡 WIP | — | Next.js 14, X-style responsive layout, refresh-token auth, business clients over `apiFetch`. Wired: feed/posts (like/comments/pin/repost/quote, clickable hashtags + hashtag composer autocomplete + hashtag results tabs + X-style hashtag/profile suggestions), dynamic Top 5 trends + searchable/zoom-safe/responsive right-column suggestions with recent history, Explorer discovery sections (Top 10 trends, non-followed profile suggestions, feed publications, submitted search filters Publications/Utilisateurs) with trend post counts aligned to the feed layout, bookmark collections, hydrated profile + privacy, **visibility switch in `/parametres`**, follow pending/accept/reject, E2EE messaging + attachments, notifications (badge+WS), translation, i18n FR/EN, dark mode, legal pages, muted words, @mentions, image/video upload (lightbox, Twitter-style autoplay), **OAuth Google (bouton login/register + BFF + callback page)**, **onboarding gate comptes OAuth (modale bloquante username + date de naissance, signal profil absent)**, **vue visiteur (fil public sans inscription : `AuthPromptProvider`/`useAuthGate`, middleware ouvre `/feed`+`/posts/:id`, actions réservées → modale connexion)**. **TODO:** real role (admin placeholder), post edit. |

## Features

| Feature | Type | Status |
|---|---|---|
| Registration / Login | Primary | 🟢 End-to-end (UI→BFF→gateway→auth), login by email or username, provisioning, username pre-check, logout + session guard. |
| Login with Google (OAuth/OIDC) | Secondary | 🟢 End-to-end. BFF `GET /api/auth/oauth/[provider]` (proxy URL) + `POST /api/auth/oauth/[provider]` (exchange code→cookie+token) + callback page `/(auth)/auth/callback/[provider]` → URL `/auth/callback/google` (Suspense, `useSearchParams`, `provisionUser`, `setRefreshCookie`). Bouton Google sur login + register (loading state, error). Google Cloud OAuth client configuré (credentials dans `auth-service/.env`). **Provisioning nouveau compte = onboarding gate** (modale bloquante au layout `(app)` : choix username + date de naissance → `PATCH /users/me` + `POST /profils`, signal `GET /profils/me` 404). *(Microsoft retiré sur demande — front + back, cf. CHANGELOG 10/06/2026.)* |
| JWT auth + protected routes | Primary | 🟢 access 15m + refresh 24h + `/auth/validate`, front single-flight refresh + `(app)` guard. **TODO:** gateway JWT middleware. |
| Role management (User/Mod/Admin) | Primary | 🟡 Role in JWT, user-service enforces admin delete, role-based nav, **admin peut créer un compte de force** (`/admin` → mot de passe temporaire mailé + username/mot de passe provisoires corrigés via modales bloquantes). **TODO:** generalize to other services, real role front. |
| Post creation/reading | Primary | 🟢 End-to-end, infinite feed (For you / Following), hashtag filter, visibility-filtered, likes, threaded comments, reposts/quotes, pin, emoji, images/videos. **TODO:** post edit. |
| User profile | Primary | 🟡 View + edit + by-username page, visibility configurable in settings, private-account locking, real avatar/banner upload. **Onglet Réponses wired** : `GET /posts/comments?author_id=` (optionalAuth, barrière de visibilité sur le post parent) + `ReplyCard` front (label « En réponse à @X » + clic → thread). **TODO:** wire aggregated user+profil+post read; onglet J'aime. |
| Social graph (follow/followers) | Secondary | 🟢 follow + lists + counts + private requests + follower removal, front wired, e2e tested. |
| Search / Explorer (accounts + hashtags) | Secondary | 🟢 Explorer discovery page with Top 10 trends, non-followed profile suggestions, regular feed publications, and submitted search filters (Publications/Utilisateurs, at least one required); search by `@username`, plain username, or display_name; direct `#hashtag` navigation; hashtag/right-column/Explorer search suggest trends then profiles; recent clicked searches can be cleared globally or one by one. **TODO:** post full-text search. |
| Automatic post translation | Secondary | 🟢 BFF `/api/translate`, conservative client gate, posts + comments, cache, toggle. |
| Muted words in feed | Secondary | 🟢 `/parametres`, per-account local persistence, feed masks others' matching posts. |
| Private encrypted messaging (E2EE) | Secondary | 🟢 End-to-end (back + UX). DM/groups admin-proof, hybrid communities, server-side read state + app-wide badge, per-conversation mute, encrypted attachments, owner-only edit with original shown subdued, **passphrase key backup (zero-knowledge, multi-device)**. |
| Real-time notifications | Secondary | 🟢 End-to-end (e2e live), Instagram aggregation, + `follow` + `message_mention` + `follow_request` + `follow_request_accepted` + `follow_request_accept_confirm`, WS, badge, detail page. |
| Mentions (@handle) | Secondary | 🟢 End-to-end: posts/comments autocomplete + clickable render; messages (E2EE) member-ids only → `message_mention`. i18n FR/EN. |
| Hashtags & trends | Secondary | 🟢 End-to-end: hashtags extracted server-side on posts, composer autocomplete + blue highlight, clickable render, `?hashtag=` results view (`À la une`/`Récent`/`Média`), and dynamic Top 5 trends from visible posts. |
| Bookmarks (collections) | Secondary | 🟢 End-to-end, collections (many-to-many) + non-deletable default, burst model, `/signets` page. |
| Image / video upload | Secondary | 🟢 Complete (Phases 0→3): media-service + MinIO + gateway `/media`; avatar/banner; post media; encrypted message attachments. |
| GIFs (Tenor/Giphy) | Secondary | 🔴 TODO (plan in DECISIONS.md). |
| Vérification e-mail | Secondary | 🟡 Code complet (Phase 1) — 🟢 après e2e `make dev`. Register → mail de vérif + page « consulte ta boîte mail » (aucune session) ; blocage dur login non-vérifié (`403`, bannière + renvoi anti-énum) ; page `/verify-email` (succès → **auto-login + accès direct au feed** / invalide-expiré) ; tokens opaques 24h usage unique. Mails de marque (clear mode). |
| Mot de passe oublié | Secondary | 🟡 Code complet (Phase 2) — 🟢 après e2e `make dev`. Page `/forgot-password` (anti-énum, écran générique) → mail lien reset 1h ; page `/reset-password` (lit `?token`, nouveau mdp + confirmation) → bcrypt + `email_verified=true` + révocation de toutes les sessions ; rejouer le lien → 400 invalid_token (usage unique). Lien depuis le login. |
| Vue visiteur (fil public) | Secondary | 🟢 Front-only, zéro backend (lecture publique déjà ouverte côté post-service). Non-connecté : fil « Pour toi » + détail post en lecture seule, hashtags cliquables (feature autre branche) ; pas de recherche/profils/messages/notifications/signets. `AuthPromptProvider`/`useAuthGate` (`isVisitor`/`requireAuth`/`promptLogin`), middleware ouvre `/feed`+`/posts/:id`, `/`→`/feed`, sidebar/tabbar/header adaptés (boutons Se connecter/S'inscrire), actions réservées → modale connexion. Appels `/me` gardés derrière `getAccessToken()` (anti-redirection forcée). |
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
- **Messaging:** communities are admin-readable (server key). The identity key can now follow the user across devices via
  the **passphrase backup** (zero-knowledge); a forgotten passphrase = unrecoverable backup, and a device with neither a
  local key nor a backup still generates a fresh key (prior history unreadable there). Perspectives: key rotation, ownership
  transfer, **admin passphrase reset** (deferred — "wipe backup → new identity" semantics, no escrow to keep DM admin-proof).
- **Notifications:** emission best-effort (lost if notif-service down at T); in-memory badge may over-count by 1
  (self-corrects); badge increments even while `/notifications` is open; `last_actor` cosmetic blur after retract.

## Setup reminders (fresh checkout)

- `make env` (or copy each `.env.example`) before `make dev` — several services need their `.env` to exist,
  notably `message-service/.env`, and `INTERNAL_EVENT_SECRET` + `MAIL_INTERNAL_SECRET` must be in the **root**
  `.env` (compose interpolates `${...}` from the root file only).
- `minio` and `media-service` share the same `media-service/.env` (`MINIO_ROOT_USER`/`PASSWORD`). MinIO console: `http://localhost:9001`.
- For code changes to appear, use `make dev` (hot-reload) — `make up` runs frozen prod images.
