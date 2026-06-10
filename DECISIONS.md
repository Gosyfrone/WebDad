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
- **Username login orchestration stays in the BFF.** `username` remains owned by user-service:
  the Next BFF resolves `username -> user_id` through the gateway, then calls auth-service with
  `user_id + password`. Auth-service still owns only credentials/JWT and never joins user data.
- **Login with Google/Microsoft = OIDC Authorization Code, code→tokens exchanged server-side in auth-service**
  (`golang.org/x/oauth2` + `go-oidc/v3`). The front only obtains the authorization URL and relays the callback
  `code`; the ID token (signature via JWKS, issuer, audience=client_id) is verified **server-side** — never trusting
  a front-supplied token. Anti-CSRF `state` is server-generated, stored by the front, re-checked at callback.
  Unverified `email_verified` is rejected (blocks account takeover by email matching).
- **Generic provider registry** (`internal/oauth`, map `{google, microsoft}` = issuer + scopes): adding Microsoft is
  one map entry + env vars (`MICROSOFT_CLIENT_ID/SECRET/TENANT`); a provider with no `CLIENT_ID` is skipped (→ 404).
  Providers are **lazily** built (OIDC discovery at first use, not at boot) so the service starts offline-resilient.
  *Caveat:* Microsoft multi-tenant (`common`) emits per-tenant issuers — single-tenant works out of the box; broad
  multi-tenant would need relaxed issuer verification (not enabled). Google is the delivered provider.
- **Account reconciliation by email:** existing account → connect + fill `provider_subject` (only if unset, no
  hijack); absent → create with `password NULL`. Classic login on a password-less account is refused with a clear
  409 (`ErrNoLocalPassword`) steering the user to the external provider. Schema migrated via idempotent `ALTER`
  (`provider` enum default `'local'`, `provider_subject`, `password` nullable) — `'local'` keeps prior behavior intact.

## Email (vérification & reset)

- **`mail-service` dédié pour le transport SMTP ; la logique-token reste dans auth-service (PG).**
  `POST /internal/send` hors gateway, authentifié par `MAIL_INTERNAL_SECRET` (`X-Internal-Secret`),
  appelé en best-effort par auth — symétrique de notification-service (2nd consommateur d'événements
  internes). Secrets SMTP isolés dans `mail-service/.env`, mailer réutilisable.
- **Tokens vérif/reset = opaques aléatoires, stockés SHA-256 hachés, usage unique (`used_at`), TTL
  vérif 24h / reset 1h.** Réutilise le pattern refresh-token : révocables sans denylist (vs JWT
  auto-portant), une fuite de table ne livre aucun token. Table unique `account_tokens(purpose enum
  'verify'|'reset', …)`. Toute nouvelle demande invalide les précédents du même `(user_id, purpose)` ;
  un reset réussi **révoque toutes les sessions** (`DELETE refresh_tokens`) — un changement de
  mot de passe doit déconnecter partout. **Un reset réussi pose aussi `email_verified=true` :**
  cliquer le lien (envoyé à l'adresse du compte, TTL 1h) prouve la possession de la boîte, donc
  débloque un compte non vérifié sans vérification séparée — le reset est un second chemin de
  preuve d'adresse, équivalent au lien de vérification.
- **`forgot-password` = anti-énumération stricte** : `ForgotPassword(email)` renvoie TOUJOURS `nil`
  et le handler répond TOUJOURS `200` générique, que le compte existe, soit actif, ou non ; le mail
  n'est envoyé (best-effort) que pour un compte existant ET actif. La page front affiche le même
  écran de confirmation dans tous les cas.
- **Login non-vérifié = blocage dur** (`403 email_not_verified`, aucun token émis), vérifié
  *après* le bcrypt pour ne pas révéler l'existence du compte.
- **Vérification d'e-mail = auto-login (révise la position « pas de session avant login »).**
  `POST /auth/verify-email/confirm` consomme le token, pose `email_verified=true` *et* émet une
  session (access + refresh, comme `/login`) ; le BFF Next pose le cookie refresh httpOnly et renvoie
  l'access token, la page `verify-email` stocke ce dernier et redirige vers le **feed**. Justifié par
  la même logique que le lien de reset : cliquer le lien (token usage unique, TTL 24h, envoyé à
  l'adresse du compte) **prouve la possession de la boîte** → facteur d'authentification suffisant.
  Garde : un compte désactivé (`is_active=false`) reste bloqué (`403`, aucune session). Le no-session
  ne vaut donc plus que pour le **register** (provisioning serveur, ci-dessous), pas pour la vérif.
- **Provisioning préservé au register, sans session client (décision Phase 1).** `auth/register`
  continue d'émettre les tokens, mais le BFF Next s'en sert **uniquement côté serveur** pour
  provisionner l'identité (`POST /users` + `POST /profils`, avec `username`/`birth_date`/`gender`
  du formulaire) puis les **jette** : il ne pose PAS le cookie refresh et ne renvoie PAS l'access
  token. Le client n'obtient donc **aucune session** et atterrit sur une page publique « consulte ta
  boîte mail » ; le blocage réel est appliqué au login. *Choisi plutôt que « register sans token +
  provisioning au 1er login » : cette variante imposerait de charrier `birth_date`/`gender` (qui
  n'existent que dans le formulaire register) jusqu'au login via un stockage temporaire — plus lourd
  et fragile.* **Admin seedé forcé `email_verified=true`** (pas de vraie boîte) pour garder un compte
  démo. Le renvoi de mail de vérif est accessible depuis la page login (les users existants passent
  `email_verified=false` et doivent se vérifier).
- **Liens dans le mail → pages front** (`APP_BASE_URL/verify-email|reset-password?token=…`), pas
  l'API directement : maîtrise de l'UX (succès/expiré/erreur), API qui reste JSON-only.
- **Corps HTML des e-mails = coquille de marque partagée** (`services/mail_template.go`,
  `brandedEmailHTML`, pure & testée) : layout table + styles inline (compat Outlook/Gmail/Apple Mail),
  direction graphique clear mode (dégradé `#8D3DFF→#5B6CFF→#47D9FF`, logo via `APP_BASE_URL`), bouton
  « bulletproof » (repli couleur solide). Vérif et reset la réutilisent → cohérence visuelle, un seul
  point de maintenance. La version **texte** reste sobre (délivrabilité).
- **Dev sans SMTP configuré = transport console** : le mailer logge le mail + le lien sur stdout au
  lieu d'envoyer (zéro dépendance Gmail en dev, on clique le lien depuis les logs).

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
- **Schema materializes "no plaintext at rest":** `messages` have only `ciphertext`+`nonce` and, after edit,
  `original_ciphertext`+`original_nonce`; `user_keys` stores only the public key; `conversations.content_key` only for communities.
- **Message edit stays E2EE:** owner-only `PATCH /messages/conversations/:id/messages/:messageId` replaces the
  ciphertext, preserves the first encrypted version for subdued display, and broadcasts `message_updated` over WS.
- **Pin/clear are per-user, on the `members` collection** (not the shared conversation), no WS broadcast (personal).
  `cleared_at` = reversible cutoff (reappears on next message, WhatsApp-style) vs destructive delete.
- **"Read" state is server data** (`members.last_read_at`), multi-device; `GET /unread-count` computes from metadata only
  (never the `ciphertext`) → E2EE intact. **Mute** (`members.muted_at`) excludes from the badge but stays unread in the list.
  Chosen over per-device localStorage (tranché with the user).
- **Passphrase key backup (multi-device), zero-knowledge** (11/06/2026, validated with user). The X25519 private key was
  per-device only → unreadable on a 2nd device. Now the private key is wrapped client-side (`XChaCha20-Poly1305`) under a
  KEK derived from a **user passphrase** (Argon2id, **19 MiB / t=2 = OWASP minimum**, chosen for mobile: the pure-JS KDF is
  synchronous and 64 MiB/t=3 froze the main thread ~15-25 s on phones → "infinite spinner"; an **auto-upgrade** re-wraps any
  heavier legacy backup with the light params on the next unlock). The KDF runs in a **Web Worker** (`key-backup.worker.ts`,
  driven by `key-backup-async.ts`, sync fallback) so the UI stays responsive during derivation. Stored server-side in
  `key_backups` as an opaque blob
  (`salt`, `nonce`, `wrapped_private_key`, `kdf_params`, `public_key`). Server never sees the passphrase nor the private key
  → **admin-proof preserved**. The IndexedDB identity is now **scoped per `userId`** (was a single global `self` record
  shared by every account on the browser → wrong key reused across accounts); a one-shot migration adopts the legacy `self`
  key for an account **only if** its public key matches the one that account already published. State machine: `ready`
  (local key + backup) / `unlock` (backup but no local key → other device: download blob, Argon2id-derive, unwrap; wrong
  passphrase ⇒ Poly1305 auth fails) / **`setup`** (no backup → define passphrase; reuses the local key if present so
  **existing users back up their current key**, else generates one). UI: `PassphraseGate` blurs the messages view until the
  identity is available. Routes declared **before**
  `/keys/:userId` (else `backup` is captured as a userId). **Assumed limits:** forgotten passphrase = unrecoverable backup
  (the point of zero-knowledge); **admin reset deferred** — the only crypto-honest semantics is "wipe backup → new identity"
  (escrow would break DM admin-proof, ruled out); a device with no local key and no backup generates a fresh key (prior
  history stays unreadable there, same as before).

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
