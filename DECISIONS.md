# DECISIONS — WebDad / Breezy

> Architectural decisions and their **rationale**. Implementation minutiae live in the code and
> are queryable via `graphify`. This file answers *"why is it built this way?"* (defense prep).
> Was section §5 of CLAUDE.md.

---

## Observability

- **Structured logging — slog + X-Request-Id (user-service pilote, 13/06/2026).** `log/slog` stdlib (Go 1.21+, aucune dépendance externe). Format JSON en `release`, texte en `debug/test` (lisible humain). Niveau depuis `LOG_LEVEL`. Trois middlewares Gin dédiés : `RequestID` (lit ou génère un UUID hex 16 B, propagé dans la réponse), `Recovery` (panic → `slog.Error` + 500 sans stack exposée), `RequestLogger` (une ligne/requête avec method, path sans query, status, latency_ms, client_ip, request_id, user_id). `JWTAuth` pose `user_id` dans le contexte Gin pour que `RequestLogger` corrèle l'utilisateur. `gin.Default()` remplacé par `gin.New()` + chaîne explicite. Ce motif sera répliqué à l'identique sur les autres services.

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
- **Admin-created accounts = temporary password + provisional username (forced fixups via JWT/`/users/me`).**
  An admin creates an account from `/admin` (auth `POST /auth/users`, AdminOnly): credentials are written
  `must_change_password=true` (a temporary password, emailed best-effort) and **email verification is
  mandatory** (`email_verified=false`): the welcome email carries the verify link + the temp password, and
  clicking it verifies the address **and** opens the session (`VerifyEmail` now propagates the flag) → the
  user lands on the feed with the change-password modal. A **dev/local shortcut** `ADMIN_CREATE_AUTO_VERIFY=true`
  marks the account verified up-front (real mail goes online) so login works directly with the temp password.
  The flag rides in the **JWT claims** (and is re-read on
  `/refresh`) so the front shows a blocking change-password modal on any page **without an extra call**;
  `POST /auth/password/change` verifies the current password, clears the flag, revokes other sessions and
  re-issues a flagless token (modal disappears, no reload). The requested **username** is provisioned in
  user-service (`POST /users/admin`); if taken it is suffixed `_<8 hex>` with `username_pending=true`
  (exposed by `/users/me`, cleared on an effective `PATCH /users/me`), driving a second blocking modal.
  Orchestration lives client-side in `lib/admin.createAccount` (admin bearer, same pattern as the RGPD
  eraser) — one datum, one service; steps 1-2 critical, profil creation best-effort.
- **Refresh token = opaque random, stored SHA-256 hashed, rotated on `/refresh`, revoked on `/logout`.**
  Opaque + DB-stored = revocable (unlike self-contained JWT); hashing means a DB leak yields no usable token;
  rotation is the basis for future reuse-detection. The **back never auto-renews** — it signs `exp` and
  401s on expiry (`code: token_expired` distinguishes expiry→refresh from invalid→no-refresh); the **front drives** refresh.
- **Inter-service auth:** JWT in header (grading requirement).
- **Username login orchestration stays in the BFF.** `username` remains owned by user-service:
  the Next BFF resolves `username -> user_id` through the gateway, then calls auth-service with
  `user_id + password`. Auth-service still owns only credentials/JWT and never joins user data.
- **Login with Google = OIDC Authorization Code, code→tokens exchanged server-side in auth-service**
  (`golang.org/x/oauth2` + `go-oidc/v3`). The front only obtains the authorization URL and relays the callback
  `code`; the ID token (signature via JWKS, issuer, audience=client_id) is verified **server-side** — never trusting
  a front-supplied token. Anti-CSRF `state` is server-generated, stored by the front, re-checked at callback.
  `email_verified` is required (blocks account takeover by email matching): the claim is decoded as `*bool` and an
  absent or explicit-`false` value is rejected (Google always sends it).
  *(Microsoft/Entra support — multi-tenant `common` issuer with manual `tid` verification — was implemented then
  removed on owner's request; only Google remains. See CHANGELOG 10/06/2026.)*
- **Generic provider registry** (`internal/oauth`, map `{google}` = issuer + scopes): adding a provider is
  one map entry + env vars; a provider with no `CLIENT_ID` is skipped (→ 404). Providers are **lazily** built
  (OIDC discovery at first use, not at boot) so the service starts offline-resilient.
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
- **OAuth first-login onboarding gate (front).** A Google sign-up creates `credentials` (auth) + lazily a
  `users` row (derived handle via `GET /users/me`) but **no profil** → invisible in Explorer (`/profils/search`)
  though mentionable via `@` (`/users/search`). Rather than auto-deriving a profil silently, the **profil-absent
  signal** (`GET /profils/me` 404) drives a **blocking modal** mounted in the `(app)` layout (`OnboardingGate`):
  present on every authenticated page, non-dismissible (ESC/outside/close disabled), re-checked each load. The
  user picks a username (pre-filled with the derived handle, availability-checked, kept = treated as available)
  and a birth date (≥13 age parity with register), then **`PATCH /users/me`** (rename only if it differs from the
  derived handle — first rename never hits the cooldown, `username_changed_at` nil) **+ `POST /profils`**
  (`display_name`=username, `birth_date`). Frontend-only, **zero backend change** (reuses existing endpoints).
  Register users always have a profil → never gated. *Assumed limit:* it's a UX gate, not server-enforced (a
  client could call APIs directly); a gateway-level barrier would be a separate effort.

## Data ownership

- **profil-service owns decorative/editable fields incl. `visibility`; user-service owns identity +
  social graph; `role` lives in the JWT.** One source of truth per datum; any profil write = one `PATCH /profils/me`,
  no inter-DB transaction.
- **Profile visibility defaults to public, then is user-configurable in settings.** `POST /profils` always stores
  `visibility=public`; `/parametres` calls `PATCH /profils/me` with `public|private`. This keeps signup friction low
  and leaves the privacy barrier enforceable server-side by post-service.
- **Nationality is stored as an ISO 3166-1 alpha-2 code in profil-service, never as a translated label.** The
  searchable edit combobox loads the public FIRST country catalogue through cached BFF route `/api/countries`;
  `Intl.DisplayNames` localizes names in FR/EN. The external API is only a catalogue source: persisted profiles
  and profile rendering remain functional if it is unavailable.
- **`ProfilDetails` aggregation (read) is composed by the caller** (front/BFF), not the service — decouples
  service from aggregation, no routing change.
- **Each service owns its schema** (`EnsureSchema` at boot, idempotent, Mongo `collMod` resync). Single source
  of truth + autonomy (`make run` against a blank DB). Resolved a validator-drift class of bug.
- **Legacy data normalization at boot (not just the validator).** A service owning its schema must also own the
  *conformity of its existing data*, because Mongo (`validationLevel: strict`) re-validates the **whole document**
  on every write: a single legacy field that violates the current `$jsonSchema` bricks *all* future edits of that
  doc (`DocumentValidationFailure` → 500). Concrete case: profiles created before `visibility` (PR #204) had
  `visibility: ""`, outside the `{public, private}` enum → every `PATCH /profils/me` failed in prod (clean local
  data hid it). Fix = an idempotent migration step in `EnsureSchema` (`normalizeLegacyProfiles`: empty/absent
  `visibility` → `public`, `bypassDocumentValidation`), same family as `dropLegacyIndexes`/`collMod` — prod
  self-heals on deploy. Chosen over `validationLevel: moderate`, which would merely *tolerate* dirty data instead
  of cleaning it. Companion: the handler's default 500 branch now logs the **raw** error so a future validation
  failure is diagnosable in seconds instead of being an opaque 500.

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
- **Hashtags live on `posts` as normalized denormalized fields** (`hashtags: []string`, lowercase, no `#`) extracted
  on create/update. This keeps `GET /posts?hashtag=...` indexable in Mongo without a separate service or cross-DB
  search. Trends are counted in post-service after the same profile-visibility checks as the feed, so private profiles
  cannot leak through hashtag counters. The hashtag results view exposes `sort=top` (likes, reposts, comments, then
  recency) and `sort=recent`; the media tab reuses the same visibility-filtered post results and renders only their
  media attachments. Search routing is shared on the front: `#tag` opens hashtag results, while plain text remains
  account/profile search in Explorer. `/posts/trends?q=...` supports prefix suggestions for the hashtag search
  dropdown; the same visibility filtering applies before counting, so suggestions do not leak private authors.
  The right-column search stores only clicked suggestion metadata in localStorage per user (`breezy-suggestion-history`)
  because it is cosmetic UX state, not a domain datum worth a backend service.
  Composer hashtag autocomplete also reuses `/posts/trends?q=...` instead of adding a separate endpoint: suggestions
  are derived from already-visible trend data, while post-service remains the single extractor/normalizer on submit.
  Explorer's "posts with hashtags" uses `GET /posts?hashtag_any=true` rather than client-side filtering, preserving
  the server-side visibility barrier and avoiding downloading arbitrary global-feed pages just to discard non-hashtag posts.
- **Bookmarks = collections (many-to-many) + non-deletable default collection + burst model.** Bookmarks reference a
  post → same service as likes/reposts. Many-to-many because a post can be filed in several collections. **Burst model**
  (Instagram "save"): a short click within `BOOKMARK_SESSION_WINDOW` auto-files into the last collection (`filed`),
  otherwise opens a chooser (`needs_choice`, files nothing — server never guesses). Window state is server-side
  (`last_bookmark_at`, per account, multi-device, no clock cheat).
- **Polls are post-owned metadata + separate `poll_votes`.** A poll is part of the post document because it is authored,
  displayed, deleted and visibility-filtered with the post. Votes live in `poll_votes` with a unique `post_id+user_id`
  index so "one vote per account" is enforced by Mongo, not by the UI. Choice counters stay denormalized in the embedded
  poll as `int32` for fast feed rendering and validator consistency. The post-service owns duration expiry and the
  `everyone|followers` audience rule; followers-only polls reuse the existing follow client, so private/follower logic
  remains server-side and cannot be bypassed by a custom front. Results visibility is also server-side: before voting,
  ordinary viewers receive zeroed counters; after their unique vote, `voted_choice_id` makes `can_view_results=true` so
  they can see the current percentages without being able to vote again. The author always receives live counters, and
  everyone receives them after `ends_at` or manual `closed_at`. Manual close is author-only (`can_close`) and stores
  `closed_at` instead of rewriting the planned end date, preserving the original duration while making the poll final.

- **Reply audience is a post-owned setting enforced server-side (15/06/2026).** Like poll voting, *who can reply* is a
  per-post choice (`reply_audience` ∈ `everyone|followers`) made at creation and **not editable afterwards** (parity with
  `Poll.Audience`; `UpdatePostRequest` only carries `content`). The field is stored on the post document with
  `omitempty`, so an empty/absent value is never persisted and old posts (created before the field) read back as
  `everyone` via `ReplyAudienceOf` — the validator declares it as an **optional** enum property, so **no backfill
  migration** is required (règle 5b). The barrier is enforced in the single comment entry point `CreateComment` (covers
  both root comments and replies) by `canReplyTo`: a `followers` post lets the author, the author's followers, and
  moderators/admins reply, everyone else gets `ErrReplyNotAllowed` (403). Follower status reuses the existing follow
  client, exactly like followers-only polls, so the rule cannot be bypassed by a custom front. **Mods/admins bypass** the
  restriction because replying is not a moderation-blocked action and they already hold elevated rights. For UX the
  post-service hydrates a transient `can_reply` per viewer, but **only for `followers` posts** (the follow check is
  skipped entirely for the `everyone` majority → near-zero added cost on feeds); the front consults it solely to disable
  the composer and hide "Reply" buttons, never as the authority. A missing/`true` `can_reply` is treated as allowed so
  read paths that don't compute it (mutations) never wrongly lock the composer.

## Vue visiteur (fil public)

- **Mode visiteur = front-only, zéro backend.** Le post-service expose déjà la lecture publique (`GET /posts` et
  `GET /posts/:id` en `OptionalJWTAuth`, barrière de visibilité appliquée serveur → un appelant sans token ne voit
  que les posts publics). Décision : ne RIEN ajouter côté serveur, tout le mode visiteur vit dans le front.
- **Réutiliser le shell `(app)`, pas un groupe `(public)` séparé.** Le visiteur voit le MÊME fil avec la même
  sidebar, seulement amputée (carte user → boutons Se connecter/S'inscrire, nav réduite à Accueil). Un groupe de
  routes parallèle aurait dupliqué layout + feed pour un gain nul.
- **Le vrai risque n'était pas la garde mais la redirection forcée.** `apiFetch` redirige vers `/login` sur tout
  401 dont le refresh échoue. Or le fil enrichit chaque post via `/posts/me/{liked,reposted,bookmarked}-ids`
  (routes `auth`), et `useFollow`/`getMyProfil`/`getMe` tapent des routes `/me`. Sans token → 401 → redirection →
  le visiteur ne voit jamais le fil. **Parade : garder chaque appel `/me` derrière `getAccessToken()`** (early-return
  vide), même pattern que les providers Notifications/Messages. C'est la pièce centrale, pas le retrait du middleware.
- **`isVisitor` rendu « membre » par défaut au 1er paint.** Pas d'accès `localStorage` au SSR/hydratation → on suppose
  connecté puis on bascule au montage (même compromis que le thème / la session) pour éviter un flash de l'UI visiteur
  chez les membres.
- **Actions réservées → modale « Connecte-toi » (`requireAuth`/`promptLogin`), pas redirection ni masquage.** Choix
  produit : like/repost/citer/signet/commentaire invitent à s'inscrire sans quitter le fil. La LECTURE des commentaires
  reste libre (GET public) ; seul l'ENVOI est gaté. L'aperçu profil au survol est désactivé pour le visiteur (il
  chargeait le graphe social authentifié) et les profils restent gardés par le middleware.
- **Racine `/` → `/feed`** : `/feed` étant désormais public, c'est l'entrée naturelle du mode visiteur (membre = UI
  complète, visiteur = UI amputée).
- **Limite assumée** : garde UX côté front (le backend protège déjà les écritures par JWT ; un client pourrait lire
  les routes publiques directement, ce qui est précisément leur contrat).

## Translation

- **Translation via Next BFF route `/api/translate`** (LibreTranslate + Google fallback), keys server-side; post-service
  stores no derived translation. Conservative client gate (`shouldAttemptTranslation`): script detection + Latin markers
  avoid false positives (a lone foreign word, a typo, mixed text must not auto-translate a French post).
- **Coverage across the 12 locales:** upstream auto-detection is authoritative. The small local marker dictionaries are not
  an allow-list: any meaningful text not clearly identified as already being in the target locale reaches the BFF. The
  result is discarded when `detectedSourceLanguage === targetLanguage`. Local logic remains a cheap same-language veto;
  Han (Chinese) and Kana (Japanese) are distinct so ZH↔JA works. Target language always comes from the account locale.

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
- **Message read/delivery receipts ("Remis / Ouvert", 15/06/2026, validated with user).** Sender-side acknowledgements
  using **server metadata only** (timestamps, never the `ciphertext` → E2EE intact). Two per-member cursors: existing
  `last_read_at` (= *Ouvert*/opened) + new `last_delivered_at` (= *Remis*/delivered). **Delivery is server-driven (no client
  ack):** a message is *Remis* when the server actually delivers it — pushed to a member's open WS socket at send time
  (`hub.OnlineFrom`) or fetched via `ListMessages` (`TouchDelivered`). Offline recipient ⇒ not yet delivered (correct).
  Reading implies receiving, so `SetMemberRead` also advances delivery (remis ≥ ouvert). A new WS event `receipt`
  `{conversation_id, user_id, delivered_at?, read_at?}` is broadcast to the **other** members (the senders) on read/delivery
  so the checks update live; `ConversationView.member_receipts` exposes the other members' cursors for the initial render
  (**DM + groups only**, not communities — too many members, not meaningful). **UI convention (chosen by user, refined after
  iPhone testing):** a **single marker, always under my LAST message** (the check follows the latest sent message): 1 grey
  check = "Envoyé" (sent, not yet read) ; 2 brand-coloured checks = "Ouvert" (read by peer / by all in a group) ; group
  partially read = 1 coloured check + reader count. The label "Envoyé"/"Ouvert" is revealed by tapping the check (PC + mobile).
  (An earlier two-marker DM design — last-read vs last-delivered on different messages — was dropped: in practice the check
  appeared "stuck" on an older message.) Placement logic is a pure tested function (`planReceipts`); live updates via pure
  `applyReceipt` (monotonic). New field `last_delivered_at` is **nullable/optional** (no enum, no `required`) ⇒ no boot
  migration needed (règle 5b). The `last_delivered_at` cursor is still maintained server-side (harmless) though the simplified
  UI keys off the read cursor only.
- **Message deletion = tombstone "for everyone" (15/06/2026, validated with user).** `DELETE …/messages/:messageId`
  soft-deletes: sets `deleted_at` and **wipes** `ciphertext`/`nonce` (content really gone — the server was already blind, so
  this is honest "delete for everyone"). The client renders "Message supprimé" in place. **Authorization** (pure tested
  `canDeleteMessage`): the **author** always; in a **group/community**, the **owner/admin** can also delete others' messages
  (moderation); **no moderation in DM** (no hierarchy). Broadcast reuses the existing **`message_updated`** WS event (the
  tombstone replaces the bubble) — no new event type. `deleted_at` is nullable/optional ⇒ no migration. Encrypted media blobs
  in media-service are left orphaned (acceptable; cleanup is a TODO).
- **Typing indicator "en train d'écrire" (15/06/2026, validated with user).** Ephemeral, **not persisted**, E2EE-safe
  (metadata only: `user_id` + `conversation_id`, never content). Transport = **REST** `POST …/typing` (chosen over making the
  WS bidirectional): the client pings **throttled ~3 s** while typing; the server broadcasts a `typing` WS event to the other
  members. No "stop" signal — the receiver sets a **+6 s expiry** per member, pruned client-side, so the indicator fades on
  its own. Shows "écrit…" (DM) / "X écrit…" / "Plusieurs personnes écrivent…" (group).
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

- **i18n = home-grown, zero dependency** (`lib/i18n.ts` + dictionnaires statiques + `LanguageProvider` + `useT()`). 12 langues
  sont disponibles : FR/EN/ZH/ES/PT/RU/JA/KO/AR/HI/DE/IT ; FR reste la référence et le repli final. Les pages légales restent
  officiellement rédigées en FR/EN et retombent sur EN pour les autres locales afin de ne pas présenter une traduction
  automatique comme juridiquement fiable.
- **Préférence de langue = donnée de compte dans user-service**, pas une clé navigateur globale. `users.preferred_locale` est
  nullable et contraint aux locales supportées ; NULL signifie « utiliser `navigator.language` ». `GET/PATCH /users/me`
  transporte la préférence sans nouvelle route. Le logout ne l'efface pas : une reconnexion au même compte la restaure ; un
  autre compte charge sa propre préférence ou la langue du navigateur. Le localStorage ne sert qu'aux visiteurs anonymes.
- **Theme:** light/dark (next-themes) + brand accent. Brand gradient violet→indigo→cyan (logo "B"), glassmorphism.
  **Tokenized surfaces** (CSS vars in `globals.css`, light values = exact current state, dark declension) → dark lives in one
  place, light unchanged. ⚠️ a `bg-*`/`border-*` utility overrides `@layer components` classes.
- **Custom theme (user-chosen colors)** = a 3rd dimension over light/dark + accent. 3 targets, each independent:
  **background** repaints the whole reading space — `--background` + `--bg-page` (solid) + glow layers→`none` **and** the glass
  surfaces `--column`/`--glass`/`--glass-strong` (background-color → hex) + `--panel`/`--panel-y` (background-**image** → must be a
  `linear-gradient(hex,hex)`, a raw hex is invalid there); **text** → `--foreground`/`--card-foreground`/`--popover-foreground`;
  **primary** → `--primary`/`--ring` + auto-contrasted `--primary-foreground` (WCAG luminance) **and the brand-gradient action
  buttons**. The "Breeze"/"Follow"/FAB/badges were a hardcoded `from-[#8D3DFF] via-[#5B6CFF] to-[#47D9FF]` (≠ `--primary`, hence
  invisible to the picker); they now read `from-[var(--brand-from)] via-[var(--brand-via)] to-[var(--brand-to)]` (defaults in
  `:root`), and the primary target collapses the 3 stops onto the chosen solid color. Overrides set as inline styles on
  `<html>`, so they win over the stylesheet regardless of mode. **Zero dependency**: home-grown HSV color wheel (`ColorWheel`,
  conic-gradient hue + radius saturation + value slider + hex/RGB) — `node_modules` is root-owned so `npm i react-colorful` is
  out, and an in-repo wheel is defense-friendlier. Color math is pure/tested (`lib/color.ts`); apply/persist logic pure where it
  matters (`buildCustomThemeVars` in `lib/custom-theme.ts`). **Persist both** source hexes (to reopen the editor) **and the
  pre-computed CSS-var map** in localStorage → a tiny inline `<head>` script applies the map before first paint (no FOUC, no color
  math shipped in the blocking script). Palette button opens a `Dialog` held as a **sibling** of the dropdown/Sheet (not inside),
  opened on a deferred `setTimeout(0)` to dodge the Radix close↔open focus race.
- **Responsive mobile-first, pivot `lg` (1024).** <lg: `MobileHeader` (left drawer Sheet) + bottom `MobileTabBar` + `ComposeFab`;
  ≥lg sidebar; ≥xl right column. Manual edge-swipe to open the drawer (Radix Sheet has no native swipe).
- **Identity is clickable → profile everywhere** (`UserListItem` stretched link; the Follow button is raised `z-10`).
  Convention posed now so DM/notifications respect it (profile photo = minimal guaranteed anchor).

## CI/CD

- **Smoke-test externe post-déploiement (`deploy.yml`, job `smoke-test`).** Tourne sur un runner GitHub (vue externe, pas depuis le VPS) après le job `deploy`. Vérifie que `API_PUBLIC_URL/health` et `FRONT_PUBLIC_URL` répondent 200 en HTTPS — ce qui aurait attrapé le bug Cloudflare (handshake TLS KO côté edge alors que les conteneurs étaient sains). 3 tentatives espacées de 10 s pour absorber le redémarrage des conteneurs ; URLs dans des variables GitHub Actions (`vars.API_PUBLIC_URL` / `vars.FRONT_PUBLIC_URL`, non-sensibles → `vars.*` et non `secrets.*`).

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

## Visibilité des likes (`likes_visibility`)

- **`likes_visibility` vit dans profil-service** aux côtés de `visibility`, sur le même modèle (`public|private`, défaut `public`). Posé sur le `Profil` Mongo, exposé via `GET /profils/:id/likes-visibility` et modifiable par `PATCH /profils/me`.
- **Application côté serveur dans post-service** : `GET /posts/liked?author_id=<id>` vérifie `LikesVisibility` avant de renvoyer la liste ; si privé et caller ≠ owner → 403. Le front affiche un état vide explicite (`LikesPrivateTab`) sur 403.
- **Barrière de visibilité secondaire** : les posts hiddens (`is_hidden=true`) sont exclus directement en base (`$ne: true` dans la query Mongo). La visibilité par auteur (profil privé + non-abonné) n'est pas recheckée post par post pour éviter N appels à profil-service (acceptable à l'échelle du projet).
- **Pas de changement sur les routes existantes** (`/posts/me/liked-ids`, `POST /like`, etc.) — la préférence ne change que la visibilité de la liste publique de likes.
