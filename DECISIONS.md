# DECISIONS — WebDad / Breezy

> Architectural decisions and their **rationale**. Implementation minutiae live in the code and
> are queryable via `graphify`. This file answers *"why is it built this way?"* (defense prep).
> Was section §5 of CLAUDE.md.

---

## Platform & frontend

- **Frontend = Next.js 14 (App Router) + TS + Tailwind/shadcn (slate).** Imposed stack.
- **Route groups** `(auth)` / `(app)` / `(legal)`: separate public/private layouts without
  polluting URLs; `(legal)` stays outside the session guard so it's reachable while logged out.
- **Config single source of truth:** `lib/config.ts` (gateway URL) + `lib/routes.ts` (route constants); no hardcoded strings.
- **`apiUrl()` client vs server base:** client = `NEXT_PUBLIC_API_URL` (published port);
  server (in-container route handlers) = `API_INTERNAL_URL` (Docker service name). In a container
  `localhost` is the container itself, so server fetches must target the Docker service name.

## Auth

- **Access (15m) localStorage + refresh (24h) httpOnly cookie via BFF.** Assumed trade-off:
  access is XSS-exposed but short-lived; refresh is non-stealable. BFF-managed cookie (same-origin)
  avoids cross-origin CORS/SameSite complexity. Refresh is **single-flight** to avoid refresh storms on simultaneous 401s.
- **Refresh token = opaque random, stored SHA-256 hashed, rotated on `/refresh`, revoked on `/logout`.**
  Opaque + DB-stored = revocable (unlike self-contained JWT); hashing means a DB leak yields no usable token;
  rotation is the basis for future reuse-detection. The **back never auto-renews** — it signs `exp` and
  401s on expiry (`code: token_expired` distinguishes expiry→refresh from invalid→no-refresh); the **front drives** refresh.
- **Inter-service auth:** JWT in header (grading requirement).

## Gateway

- **Thin reverse proxy (stdlib).** Prefix→URL table, transparent forward, preserves prefix. Mince + "we built it"
  (good for defense) vs per-endpoint BFF handlers (reserved for multi-service aggregation).
- **No trailing-slash redirect:** each service exposed via 2 routes (bare prefix + `/*path`) so collection
  endpoints (`POST /users`) don't 307 into a route the service doesn't know.

## Provisioning

- **Lazy user provisioning:** `users` is filled by `POST /users` (BFF at register, persists chosen handle)
  + upsert on `GET /users/me`, **not** by auth at register. Zero auth↔user coupling, autonomous service.
  Username pre-checked before signup; collision resolved by successive candidates.
- **Profil: `POST` = the ONLY creation (`display_name` required); `GET /profils/me` is read-only (404 if absent).**
  No write-on-GET, no guessed display_name. Front handles the 404 (POST-if-absent). profil-service never calls
  user-service at runtime; the BFF aligns `display_name = username` at register.

## Data ownership

- **profil-service owns decorative/editable fields incl. `visibility`; user-service owns identity +
  social graph; `role` lives in the JWT.** One source of truth per datum; any profil write = one `PATCH /profils/me`,
  no inter-DB transaction.
- **`ProfilDetails` aggregation (read) is composed by the caller** (front/BFF), not the service — decouples
  service from aggregation, no routing change.
- **Each service owns its schema** (`EnsureSchema` at boot, idempotent, Mongo `collMod` resync). Single source
  of truth + autonomy (`make run` against a blank DB). Resolved a validator-drift class of bug.

## Social graph

- **Edge table `follows` + separate `follow_requests`.** A pending request to a private profile is NOT a relation
  (no edge until accepted). Accept = atomic transaction (delete request + insert edge) avoiding "accepted but not following".
  Counters are **calculated (COUNT)** on detail reads; denormalization deferred until load requires it.
- **Identity cooldown architecture posed now, enforcement off by default.** `display_name_changed_at` /
  `username_changed_at` recorded only on real change; refusal (429) gated by env (`*_CHANGE_COOLDOWN`, default 0 = off).
  Capturing the baseline today avoids a contournable cooldown later; the timestamp is free, only refusal is config-driven.

## Posts

- **Cross-service visibility barrier on ALL post reads** (not just the front filtering by followed ids): public →
  visible; private → owner or accepted follower only. Security must not depend on the front.
- **Denormalized `likes_count`/`comments_count` as `int32`** (`$inc`), idempotent likes via unique index.
  `int32` because the `$jsonSchema` validator declares `bsonType:"int"`.
- **Pin:** `pinned_at` on the post, owner-only, one pin per profile; profile read sorts by it, but feeds return a
  copy **without** `pinned_at` so another's pin never personalizes the global feed (front exception `canPin` keeps
  instant visual feedback for the author).
- **Bookmarks = collections (many-to-many) + non-deletable default collection + burst model.** Bookmarks reference a
  post → same service as likes/reposts. Many-to-many because a post can be filed in several collections. **Burst model**
  (Instagram "save"): a short click within `BOOKMARK_SESSION_WINDOW` auto-files into the last collection (`filed`),
  otherwise opens a chooser (`needs_choice`, files nothing — server never guesses). Window state is server-side
  (`last_bookmark_at`, per account, multi-device, no clock cheat).

## Translation

- **Translation via Next BFF route `/api/translate`** (LibreTranslate + Google fallback), keys server-side; post-service
  stores no derived translation. Conservative client gate (`shouldAttemptTranslation`): script detection + Latin markers
  avoid false positives (a lone foreign word, a typo, mixed text must not auto-translate a French post).

## Messaging (E2EE)

- **Hybrid model.** Browser-side encryption. Each user has an X25519 keypair (private in IndexedDB, **per device**).
  Each conversation has a symmetric content key; for DM/groups it's wrapped per member (anonymous sealed box) → server
  sees only envelopes + `ciphertext`/`nonce` and can decrypt nothing = **admin-proof**. **Communities** keep the key
  **server-side** (handed to each joiner for unlimited auto-join) → semi-public, admin-readable. Strict E2EE with
  unlimited unknown viewers is impossible (needs an online key-holder) → "Telegram channel" style.
- **Groups:** owner + members, any member invites (holds the key), owner-only manage, cap 32, encrypted name, single key
  no rotation (new member reads full history; forward secrecy = future). **Communities:** viewer/talker, cap 32 talkers,
  unlimited viewers, **clear name** + public directory (name needed for discovery before joining).
- **Cursor pagination** (`before=<messageId>` on `_id`, not offset): stable on a live chat (new messages arrive via WS at
  the bottom without disturbing backward paging).
- **Schema materializes "no plaintext at rest":** `messages` have only `ciphertext`+`nonce`; `user_keys` stores only the
  public key; `conversations.content_key` only for communities.
- **Pin/clear are per-user, on the `members` collection** (not the shared conversation), no WS broadcast (personal).
  `cleared_at` = reversible cutoff (reappears on next message, WhatsApp-style) vs destructive delete.
- **"Read" state is server data** (`members.last_read_at`), multi-device; `GET /unread-count` computes from metadata only
  (never the `ciphertext`) → E2EE intact. **Mute** (`members.muted_at`) excludes from the badge but stays unread in the list.
  Chosen over per-device localStorage (tranché with the user).

## Notifications

- **Emission = synchronous best-effort fire-and-forget** to `/internal/events` (off the gateway, `INTERNAL_EVENT_SECRET`).
  Temporal coupling neutralized by best-effort; an event bus (NATS/Redis) is oversized for now (report perspective).
- **Aggregation (anti-spam):** one notification = one `group_key` per recipient (`like:<post>`, `comment:<post>`,
  `reply:<rootComment>`, `mention:<source>`, `repost:<post>`, `quote:<...>`, `follow`,
  `follow_request:<actor>`, `follow_request_accepted:<actor>`,
  `follow_request_accept_confirm:<actor>`,
  `message_mention:<conv>`), `$inc count`, `retract` decrements/deletes. O(1), no actor array (slight cosmetic
  `last_actor` blur after retract, assumed).
- **Rules by type:** like/comment/repost/quote → post (or quoted) author; reply → ROOT author; mention → each mentioned;
  follow_request → private-profile owner (actionable); accepted request → persisted notification + WS decision
  to the requester, plus persisted confirmation in the owner's notification list; rejected request → no
  requester notification. Never to self.
- **Front badge:** app-wide unread counter held in memory (dedup by id), single WS, seeded by `unread-count`. Avoids a
  server request per event during spikes; self-corrects on page open / `notification_refresh`.

## Mentions (@handle)

- **Shared pure bricks** (`lib/mentions.ts`, regex aligned with the back) reused across posts/comments/messages = one
  source of truth for regex/insertion/rendering.
- **Posts/comments:** back already emits `mention`; front adds autocomplete + clickable render — no back change.
- **Messages (E2EE):** server can't read `ciphertext`, so the **client** computes `mentioned_member_ids` (ids only, never
  text) → message-service emits `message_mention` (aggregated per conversation). Render: member → profile,
  non-member → preview card → `/explorer`. "X mentioned you" preview is 100% client (last message already decrypted) →
  no server flag, E2EE preserved.

## Media (MinIO via gateway)

- **6th microservice `media-service` + MinIO**, content-agnostic (opaque bytes under a random id, `owner_id` in MinIO
  user-metadata → no side DB). **Download through the gateway, NOT presigned MinIO URLs** — the cardinal rule is "everything
  through the gateway"; presigned URLs would expose MinIO (public port, CORS, host unresolvable by the browser) and violate
  the principle defended at the oral. The stdlib proxy already streams → video is not a memory concern. MinIO = canonical
  microservices answer (ticks "containerization"), far better at defense than "blob in DB"/disk.
- **Clear upload `POST /media`** (JWT, magic-byte MIME sniff, caps 5MB image / 50MB video) vs **`POST /media/encrypted`**
  (opaque E2EE blob, no sniff). `GET /media/:id` is **public** (unguessable id), streamed with Range/seek + immutable cache + ETag.
- **Posts (Phase 2):** post-service carries `media []MediaRef`, stays agnostic (relative `/media/<id>` refs); `content`
  optional if ≥1 media. **Messages (Phase 3):** file encrypted client-side → opaque blob uploaded → id/nonce/mime in the
  **encrypted envelope** (`{v:1,text,media[]}`, else plain text → backward-compatible) → server blind, zero schema change.

## Media viewer

- **Two behaviors.** Private message → `MediaLightbox` (the `src` is the already-decrypted objectURL → download stays E2EE).
  Post → `PostPhotoModal` two-pane X-style (media + comments); `PostActions` extracted from `PostCard` so card and modal
  share one optimistic logic. **Both overlays rendered via `createPortal(document.body)`** because a `backdrop-blur`/`transform`
  ancestor creates a containing block for `position:fixed`, confining the overlay. Assumed: card and modal hold distinct state
  (a like in the modal isn't live-reflected on the card behind; resync on next load).

## Feed video (Twitter-style autoplay)

- **`FeedVideo`:** muted+loop+playsInline autoplay when ≥60% visible (IntersectionObserver), pause off-screen.
  **Lazy-mount** via a second observer (`rootMargin:400px`) so a video far down an infinite feed is neither downloaded nor
  looped — the key to not loading N videos at once. Front-only, zero server cost. Autoplay mandates muted (browser policy).

## GIFs (planned)

- **BFF Next route, NOT a microservice** (mirrors translation): `/api/gifs/*` with the provider key server-side
  (`GIF_API_URL`/`GIF_API_KEY`, Tenor/Giphy). A GIF = third-party API + key to hide = same case as translation, no new
  container/DB. **Split by context:** posts (public) → external URL as `{url,type:'image'}`, zero backend; messages (E2EE) →
  bytes fetched via the BFF (proxy hides key + bypasses CORS), then encrypted through the Phase-3 attachment pipeline (never
  a third-party URL the recipient would fetch in clear).

## i18n / theming / responsive

- **i18n = home-grown, zero dependency** (`lib/i18n.ts` registry + `LanguageProvider` + `useT()` + globe dropdown). FR is the
  reference; cascade locale→FR→raw key. 2 languages → a home dico beats `next-intl` (routing-by-locale overhaul). Adding a
  language = 1 `LOCALES` entry + 1 `messages` block.
- **Theme:** light/dark (next-themes) + brand accent. Brand gradient violet→indigo→cyan (logo "B"), glassmorphism.
  **Tokenized surfaces** (CSS vars in `globals.css`, light values = exact current state, dark declension) → dark lives in one
  place, light unchanged. ⚠️ a `bg-*`/`border-*` utility overrides `@layer components` classes.
- **Responsive mobile-first, pivot `lg` (1024).** <lg: `MobileHeader` (left drawer Sheet) + bottom `MobileTabBar` + `ComposeFab`;
  ≥lg sidebar; ≥xl right column. Manual edge-swipe to open the drawer (Radix Sheet has no native swipe).
- **Identity is clickable → profile everywhere** (`UserListItem` stretched link; the Follow button is raised `z-10`).
  Convention posed now so DM/notifications respect it (profile photo = minimal guaranteed anchor).

## CI/CD

- **3 GitHub Actions workflows by domain:** `ci-go` (per-service matrix: gofmt + vet + build + `test -race` + tidy +
  golangci-lint v2 + govulncheck report-only), `ci-frontend` (lint + build/typecheck), `ci-integration` (docker compose up the
  stack minus the frontend, wait for all healthchecks). Separate files = independent triggers/path-filters; matrix = parallelism +
  per-module isolation; `-race` is free and catches data races; the docker smoke test materializes "DB health" + "full Docker".
  `.env` gitignored → regenerated from `.example` in CI. govulncheck = 0 vuln since Go 1.25 bump.

## Dev / infra

- **`make dev` = `docker-compose.dev.yml` overlay** (Go: `air` hot-reload + bind-mount + shared Go caches; frontend: `next dev`
  + bind-mount + `WATCHPACK_POLLING`). The `dev` Dockerfile stage sits before the runtime stage so `make up`/`build` always
  produce the prod image (last stage). `make up` runs **frozen prod images** (no rebuild on source change).
- **`.env` layering:** root = cross-cutting vars (`JWT_SECRET`, `INTERNAL_EVENT_SECRET`, `NEXT_PUBLIC_API_URL`, only the root
  `.env` is interpolable by compose); `<service>/.env` = own config via `env_file:`; `environment:` only for Docker overrides.
- **Containerization = Docker + docker-compose** (grading requirement).
