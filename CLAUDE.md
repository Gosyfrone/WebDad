# CONTEXT — Breezy (WebDad) · Distributed App Dev Project (FISA INFO A3)

> Entry point for Claude Code. Read this first at every session start.
> Detail lives in dedicated files (keep this one lean):
> - **[ARCHITECTURE.md](ARCHITECTURE.md)** — system architecture, service boundaries, flows, critical components.
> - **[DECISIONS.md](DECISIONS.md)** — architectural decisions + rationale (defense prep).
> - **[PROJECT_STATUS.md](PROJECT_STATUS.md)** — current state, open TODOs, assumed compromises.
> - **[CHANGELOG.md](CHANGELOG.md)** — session work log · **[CHANGELOG_ARCHIVE.md](CHANGELOG_ARCHIVE.md)** — resolved issues / debugging history.
> - **[PROMPTING.md](PROMPTING.md)** — token-efficient workflow + prompt templates (read on demand, NOT auto-loaded; user may say *"from PROMPTING.md, write me the prompt for: …"*).
> - Knowledge graph in `graphify-out/` — query it before reading source (see Operating Rules).
> - Latest session note: 15/06/2026 — **feat(posts) « qui peut répondre »** : champ post `reply_audience` ∈ `{everyone (défaut), followers}` choisi à la création (pilule façon X dans le composer), non éditable ensuite. Barrière autoritaire dans `CreateComment` (`canReplyTo` : followers ⇒ auteur/abonnés/mods-admins ; sinon 403 `ErrReplyNotAllowed`). `bson:omitempty` + enum optionnelle (absence tolérée) + **migration idempotente au boot** `backfillReplyAudience` (pose `everyone` sur les posts legacy, règle 5b prod). Transient `can_reply` hydraté par viewer (seulement si `followers`, **fail-closed** si user-service indispo) → composer désactivé/boutons « Répondre » masqués côté front. i18n FR/EN (10 autres locales retombent sur FR). Swagger régénéré. Détails `CHANGELOG.md` / `DECISIONS.md` (« Reply audience »).
> - Session note: 15/06/2026 — **branche #215 complète** (messages) : (1) accusés « Envoyé / Ouvert » (DM + groupes) via curseur serveur `last_delivered_at` + WS `receipt` + `member_receipts` (UI : 1 coche grise = envoyé, 2 coches couleur = ouvert, groupe = nombre de lecteurs ; coches cliquables → libellé) ; (2) suppression « pour tout le monde » (tombstone `deleted_at`, contenu vidé ; auteur OU owner/admin de groupe ; diffusion via `message_updated`) ; (3) « en train d'écrire » (éphémère `POST /typing` → WS `typing`, ping throttlé ~3 s, expiry +6 s). Champs nullables → pas de migration. Métadonnée serveur, E2EE intact. Détails `CHANGELOG.md` / `DECISIONS.md`.
> - Session note: 12/06/2026 — fix(profil) : édition de profil impossible en prod (500). Cause : 16 profils legacy avec `visibility: ""` (hors enum) → le validateur Mongo strict rebloquait toute écriture. Correctif : migration idempotente `normalizeLegacyProfiles` au boot (vide/absent → `public`) + log de l'erreur brute dans le 500 par défaut ; prod réparée à chaud (`updateMany`). Détails `CHANGELOG.md` / `DECISIONS.md`.

*Last Codex sync: 12/06/2026 — Admin "create account" feature (3 services + BFF + 2 blocking gates) and full display-name visibility in feed right-sidebar Who-to-follow implemented and documented.*

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
5b. **⚠️ BASE DE DONNÉES EN PROD — migrations rétrocompatibles obligatoires.** Une base de
    données peuplée existe. Toute feature qui ajoute un champ contraint (enum `$jsonSchema`,
    `required`, index unique) DOIT fournir une **migration idempotente au boot** (dans
    `EnsureSchema`) qui backfill les documents existants vers une valeur valide. Sinon les vieux
    docs violent le nouveau schéma et **toute mise à jour est rejetée** (Mongo `strict` revalide le
    doc complet) — cas vécu 2× : `visibility` puis `likes_visibility` (un `""` persisté ne respecte
    pas l'enum). Corollaire : tag bson d'un champ à enum/défaut → jamais de valeur « vide »
    persistée (poser un défaut explicite à la création **et** `omitempty`). Réf : `backfillVisibility`
    dans `profil-service/internal/database/init.go`.
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

- 2026-06-15: **Compteurs dynamiques (likes/commentaires/reposts) — polling batch, façon X** (validé avant code : polling batch plutôt que push WS, justifié par l'échelle). post-service : `GET /posts/stats?ids=…` (`optionalAuth`, route STATIQUE avant `/:id`) → `models.PostStat{id,likes_count,comments_count,reposts_count}` ; repo `StatsByIDs` (`$in` projeté sur compteurs+`author_id`, `is_hidden $ne true`) ; service `PostStats` réapplique `canReadAuthor` (mémoïsé), borne `MaxStatsIDs=100`. Front : `getPostsStats`, helpers PURS `applyStatsToPost`/`applyStatsToPosts` (ne touchent QUE les 3 compteurs, jamais l'état « moi » ; préservent la référence si rien ne change), hook `usePostStatsPolling(getIds,onStats,intervalMs?)` (`STATS_POLL_INTERVAL_MS=7000`, sonde si `!document.hidden`, refetch au refocus, garde `inFlight`). Branché dans `FeedView`, `PostDetail`, `ProfilView`. **Suivi (même session)** : (a) **bug** « seul le repost bougeait » → `PostCard`/`PostActions` affichent les compteurs en état local ; ajout des `useEffect` de resync `likeCount`←`post.likesCount` et `commentCount`←`post.commentsCount` (reposts en avaient déjà un) — PAS un problème de transport ; (b) **animation** façon X : composant partagé `components/feed/animated-count.tsx` (`AnimatedCount`+`formatCount` dédupliqué) avec « roll » vertical (`count-up`/`count-down` dans `globals.css`) ; (c) **détail plus vif** : `STATS_POLL_INTERVAL_DETAIL_MS=3500`. Doctrine transport (défense) : polling selon la criticité (compteurs non urgents → polling ; messages/notifs critiques → WS), pas « tout en WS ». Vérifs : post-service `go build`/`go vet`/`go test ./internal/...` OK, `gofmt` propre ; front `tsc`/`eslint` OK, `vitest` 110 OK (`post-stats.test.ts`) ; `make swagger` régénéré (`/posts/stats`) ; `graphify update .` OK. Détails CHANGELOG/DECISIONS « Compteurs dynamiques (polling batch) ».
- 2026-06-15: suggested commit message: `feat(feed): compteurs dynamiques (likes/commentaires/reposts) animés par polling`.
- 2026-06-15: **Fil temps réel — bandeau « a posté » (style X)** (validé avant code : périmètre bandeau seul, firehose public + filtre front, bandeau toujours affiché). post-service : hub de DIFFUSION `internal/realtime/hub.go` (set de connexions, pas indexé par user) + `internal/handler/ws_handler.go` → `GET /posts/ws?access_token=<jwt>` ; `middleware.ParseToken` exporté. `CreatePost` → `broadcastNewPost(authorID, postID)` fire-and-forget, **public-account-gated** (`profilClient.Visibility`), ping minimal `{type, post_id, author_id}`. Hub via `WithFeedBroadcaster` (no-op défaut). `CORS_ALLOWED_ORIGINS` ajouté (config + `.env`/`.env.example`). Gateway inchangée (proxy WS natif sous `/posts`). Front : `connectFeedRealtime`/`getPostAuthor` exportés de `posts.ts` (reconnexion backoff) ; `FeedView` accumule les pings (dédup + exclut mes posts/hashtag/déjà présents ; onglet Abonnements filtré via `getFollowingIds`), bandeau flottant (avatar + `feed.new_posts`) → `revealPending` (refetch page 0 + prepend dédup + scroll top). i18n FR « a posté » / EN « posted ». Vérifs : post-service `go build`/`go vet`/`go test ./internal/...` OK, `gofmt` propre ; front `tsc`/`eslint` OK, `vitest` 105 OK ; `make swagger` régénéré (`/posts/ws` présent). Détails CHANGELOG/DECISIONS « Fil temps réel (WebSocket) ».
- 2026-06-15: suggested commit message: `feat(feed): bandeau temps réel « a posté » via WebSocket`.
- 2026-06-12: **UI Réponses façon feed**. `CommentRow` extrait/exporté de `comment-section.tsx` (onDelete/footer optionnels, liens profil `stopPropagation`). `ReplyCard` rend désormais une carte encadrée cliquable (`role="link"`+`router.push`) avec avatar/identité/contenu via `CommentRow` (commentaire parent relié à la réponse par un fil), au lieu de blocs texte bruts. `tsc`/`eslint` OK. Suggested commit: `ui(profil): commentaires des reponses encadres facon feed`.
- 2026-06-12: **fix Réponses/deep-link (3 points)**. (1) Notif `comment` racine émet maintenant `CommentID` (post-service `emitCommentEvents`) → deep-link fonctionne ; notifs créées AVANT le fix restent sans ancre (re-tester avec un nouveau commentaire). (2) Prop `embedded` sur `PostCard` (retire glass/ombre/marges/hover) utilisée dans l'onglet Réponses → plus d'ombre sur le post parent. (3) Backend hydrate le commentaire parent (`CommentWithPost.ParentComment`) → `ReplyCard` affiche post → commentaire parent → réponse. `go build/test` OK, `tsc`/`eslint` OK, `make swagger` régénéré (OK sur cet hôte Linux). Détails CHANGELOG.
- 2026-06-12: suggested commit message: `fix(profil): commentaire parent dans les reponses + deep-link notif + post sans ombre`.
- 2026-06-12: **Réponses en vue conversation + deep-link notif commentaire**. Front-only, zéro backend. (1) `ReplyContext.parentPost` (post parent complet, mappé depuis `parent_post` déjà renvoyé par l'API) → `ReplyCard` rend le `PostCard` parent puis la réponse en dessous (fil `ml-7 border-l-2`), cliquable vers `/posts/:id?comment=:id`. (2) `notificationHref` : `comment`/`reply` → `/posts/:postId?comment=:commentId` ; `post-detail` lit `?comment` → `PostCard.focusCommentId` (auto-ouvre commentaires) → `CommentSection` ancre `id="comment-<id>"`, scroll+surbrillance avec retry, et `CommentThread` auto-déplie le thread contenant la réponse. i18n `profil.replies_deleted_parent`. `tsc`/`eslint`/`vitest` OK ; `npm run build` bloqué par `.next` root (conteneur dev) → `sudo rm -rf frontend/.next` pour un build prod. Détails CHANGELOG.
- 2026-06-12: suggested commit message: `feat(profil): vue conversation des reponses + deep-link notif commentaire`.
- 2026-06-12: **création de compte par un admin** (validée avant code). Bouton « Créer un compte » sur la page `/admin` (réservé admin). Back : auth `POST /auth/users` (AdminOnly) → credentials `email_verified=true` + `must_change_password=true`, mot de passe temporaire mailé (best-effort) ; drapeau `must_change_password` dans les claims JWT (lu par `useSession`, re-scanné au `/refresh`) ; `POST /auth/password/change` (JWT) vérifie le pwd actuel, lève le drapeau, révoque les autres sessions, ré-émet des tokens. user `POST /users/admin` (AdminOnly) → ligne `users` (id imposé), username suffixé `_<8 hex>` + `username_pending` si pris, colonne exposée par `/users/me`, levée sur `PATCH /users/me` effectif. profil `POST /profils/admin` (admin) → profil display_name=username effectif (neutralise l'onboarding gate). Front : `lib/admin.createAccount` orchestre les 3 services (bearer admin) ; BFF `POST /api/auth/password/change` (relaie le bearer, rotation cookie refresh) ; 2 modales bloquantes dans le layout `(app)` — `PasswordChangeGate` (drapeau JWT, prioritaire) puis `UsernamePendingGate`. i18n FR/EN `admin.create.*` / `account.password_change.*` / `account.username_pending.*`. Vérifs : `go build`/`go vet`/`go test ./internal/...` OK (auth/user/profil), `gofmt` propre, Swagger régénéré via `make swagger` (swag local OK sur cet hôte Linux) → `/auth/users`, `/auth/password/change`, `/users/admin`, `/profils/admin` présents. tsc/lint/build frontend non lançables (node_modules root-owned). Détails CHANGELOG + DECISIONS « Admin-created accounts ».
- 2026-06-12: **suivi** — compte créé par un admin = **vérification d'e-mail obligatoire** (`email_verified=false` en prod) : l'e-mail porte le lien de vérif + le mot de passe temporaire ; le lien vérifie ET ouvre la session (`VerifyEmail` propage `must_change_password`) → feed + modale changement de mot de passe. Court-circuit DEV/LOCAL via `ADMIN_CREATE_AUTO_VERIFY` (mail réel en ligne) ; mis à `true` dans `auth-service/.env`, documenté `false` dans `.env.example`. Aucun nouvel endpoint → Swagger inchangé. `services.New` prend un paramètre `adminCreateAutoVerify`. Vérifs auth : `go build`/`go vet`/`go test ./internal/...` OK, `gofmt` propre.
- 2026-06-12: suggested commit message: `feat(admin): création de compte par un admin avec mot de passe temporaire`.
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
- 2026-06-12: **fix(profil) édition impossible en prod (issue #222)** implémenté après validation. Diagnostic confirmé par logs prod (`PATCH /profils/me` → 500 réel ~11ms, pas le gateway) + scan Mongo : 16 profils legacy avec `visibility: ""` (hors enum `{public, private}`) ; le validateur `$jsonSchema` strict revalide le document ENTIER à chaque écriture → toute édition de ces comptes échouait (`DocumentValidationFailure` 121 → 500). OK en local (données propres). Correctif profil-service : (1) `normalizeLegacyProfiles` ajouté à `EnsureSchema` — au boot, `visibility` vide/absent → `public` (`bypassDocumentValidation`), idempotent ; (2) `respondProfilError` logue l'erreur brute dans la branche 500 par défaut. Prod réparée à chaud (`updateMany`, 16 profils) en attendant le redéploiement de l'image. Aucun changement route/handler/annotation → Swagger non requis. Vérifs : `go build`/`go vet`/`go test ./internal/...` OK, `gofmt` propre ; migration testée sur Mongo local (normalise, idempotente, doc réparé éditable). Suggested commit message: `fix(profil): réparer l'édition de profil bloquée par une visibility legacy vide`.
- 2026-06-15: pending user validation for multilingual settings and reliable post translation. Proposed scope: add `zh/es/pt/ru/ja/ko/ar/hi/de/it` UI dictionaries; persist authenticated preference as `users.preferred_locale` through existing `GET/PATCH /users/me` (browser locale only when the account has no saved preference); isolate anonymous/local fallback from account preferences; make translated post content react to locale changes; replace the current permissive heuristic with conservative script/marker detection plus upstream detected-language confirmation so French slang/typos are not translated. Existing handler annotations must be updated and root `make swagger` run because the `PATCH /users/me` contract changes, even though no new route is expected.
- 2026-06-15: multilingual implementation in progress after validation. Account persistence, nullable/idempotent PostgreSQL schema, existing `GET/PATCH /users/me` contract, session-aware frontend provider, RTL Arabic direction, locale-reactive post translation, conservative French/slang detection tests, and Swagger regeneration are implemented. Blocking consent requested before sending the 677 generic English UI labels (no secrets/user data) to Google Translate once to generate and commit 10 complete static dictionaries; external upload was rejected until the user explicitly acknowledges it.
- 2026-06-15: multilingual feature complete after explicit translation consent. UI now supports FR/EN/ZH/ES/PT/RU/JA/KO/AR/HI/DE/IT (677-key parity per locale). `users.preferred_locale` persists per account across logout/login; account without preference uses browser locale. Post/comment translation reacts to locale changes and conservatively ignores French slang/typos. Legal content remains FR/EN with EN fallback. Swagger regenerated; user-service tests, live PostgreSQL migration, frontend tsc/lint/107 tests/build all pass. Suggested commit: `feat(i18n): ajouter 12 langues et persister la langue par compte`.
- 2026-06-15: pending validation for post-translation coverage follow-up. Runtime BFF checks confirm provider pairs PT→KO, ES→EN and FR→JA work; remaining failures come from the client marker gate rejecting short/unlisted Latin-language wording before `/api/translate`. Proposed fix: make upstream auto-detection the source of truth for all sufficiently meaningful posts, keep only a conservative local same-language veto (scripts + strong target markers), and discard results when upstream detects the target language. Add a 12×12 source/target matrix of representative tests plus slang/same-language regressions. Front-only/BFF behavior change, no Go route or Swagger change.
- 2026-06-15: post-translation coverage follow-up complete after validation. Marker lists no longer gate foreign-language eligibility; meaningful text reaches upstream unless clearly already in target, and upstream same-language detection discards no-op translations. Han/Kana split enables Chinese↔Japanese. Unit matrix covers 132 cross-language pairs + 12 same-language cases; live BFF cycle across all 12 targets passed. Frontend tsc/lint/109 tests/build OK; no route/Swagger change. Suggested commit: `fix(translation): traduire tous les posts vers la langue du compte`.
- 2026-06-15: **feat(posts) reply audience** implémenté après validation (UX pilule façon X + bypass mods/admins choisis par l'utilisateur). Backend post-service : `Post.ReplyAudience` (`omitempty`, enum optionnelle au validateur), **migration idempotente boot `backfillReplyAudience`** (posts legacy → `everyone`, règle 5b ; testée local = 8 posts), `CreatePostRequest.ReplyAudience`, barrière `canReplyTo` dans `CreateComment` (rôle JWT propagé, `ErrReplyNotAllowed` 403), transient `can_reply` hydraté par viewer uniquement sur les posts `followers` (coût nul sinon, **fail-closed** si user-service indispo) sur tous les chemins de lecture. Front : type `ReplyAudience`, `replyAudience`/`canReply` sur `FeedPost`, pilule `ReplyAudiencePill` dans `PostComposer`, `CommentSection`/`CommentThread` désactivent le composer + masquent « Répondre » via `canReply` (propagé par `post-card`/`post-photo-modal`). i18n FR/EN `composer.reply_*` + `comment.restricted_followers` (10 autres locales → repli FR, passe de traduction consentie à prévoir). Vérifs : `go build`/`go vet`/`gofmt`/`go test ./internal/...` OK (+ `TestReplyAudienceOf`, `TestCanReplyTo`), `make swagger` régénéré ; frontend `tsc --noEmit`, ESLint, 124 tests OK. **Non éditable après création** (parité sondage). Suggested commit message: `feat(posts): audience des réponses paramétrable à la création`.
- 2026-06-15: **feat(settings/auth) account controls** implemented after validation. `/parametres` now has « Paramètres généraux » (existing language/visibility/muted words) then « Paramètres utilisateur » with current/new/confirm password (reuses `/auth/password/change`) and verified email change. Auth adds `POST /auth/email/change/request` + `/confirm`, idempotent `pending_email` migration/partial unique index, `email_change` one-use 24h token, transactional promotion + invalidation of old verify/reset tokens + session revocation/reissue; old email remains active until proof. Public `/verify-email-change` + BFF rotates cookie/token. Swagger annotations added and aggregate regenerated through Linux container after local `make swagger` hit Windows bash Error 1312. Checks: auth gofmt/test/vet, live PG migration + Gateway invalid-token response, frontend tsc/lint/124 tests/build. `graphify update .` unavailable because `graphify` is not installed on this Windows host. Suggested commit: `feat(settings): modifier le mot de passe et l'adresse e-mail`.
- 2026-06-15: **fix(i18n/settings)** implemented after validation. The user-account block already rerendered through `useT()` but its 31 new settings keys existed only in FR/EN, causing French fallback for the other locales. Added native ZH/ES/PT/RU/JA/KO/AR/HI/DE/IT strings for general/user headings, password form/errors, email form/errors, and confirmation page. Automated audit confirms every key appears in 2 main + 10 additional dictionaries. Frontend tsc/lint/124 tests/build OK. Front-only; Swagger not required. Suggested commit: `fix(i18n): traduire les paramètres utilisateur dans les 12 langues`.
- 2026-06-15: **fix(mail) logo des e-mails + spam** implémenté après validation (plans 1 + 3). Cause commune : logo et liens dérivés d'`APP_BASE_URL` (= localhost en dev) → logo injoignable + liens localhost/http = signal spam. **Plan 1** : nouveau `MAIL_LOGO_URL` (auth-service) — URL absolue PUBLIQUE du logo découplée d'`APP_BASE_URL` (charge même en local) ; `brandedEmailHTML(logoURL, …)` prend l'URL complète (au lieu de la dériver) ; 5 appels → `s.mailLogoURL` ; `New(...)` +param `mailLogoURL` ; défaut rétro-compat `APP_BASE_URL/logo_breezy.png` ; `.env`/`.env.example`/2 compose posent `MAIL_LOGO_URL=https://breezy.philippeluu.fr/logo_breezy.png`. **Plan 3** : en-tête `Reply-To` (= From) dans `buildMIME` (mail-service). Le vrai levier anti-spam reste `APP_BASE_URL=https://breezy.philippeluu.fr` en prod (déjà posé côté serveur). SMTP/DKIM déjà alignés (`From = SMTP_USER`), non touché. Handlers/routes inchangés → Swagger non requis. Vérifs : auth `gofmt`/`build`/`vet`/`test ./internal/services/...` OK, mail `gofmt`/`build`/`vet`/`test` OK. Suggested commit: `fix(mail): logo public dans les e-mails et en-tête Reply-To`.
- 2026-06-15: **feat(settings) retour auto après vérification d'e-mail** implémenté après validation. La fonctionnalité de changement d'e-mail existait déjà entièrement (commit `3323c09`) ; seul le flux post-vérif a été ajusté. `/verify-email-change` redirige désormais automatiquement vers `/parametres?email_changed=1` au succès (après `notifySessionChanged` ; le JWT renouvelé porte déjà la nouvelle adresse) au lieu d'un bouton manuel. `UserAccountSettings` lit `?email_changed=1` au montage → bandeau succès « Adresse e-mail modifiée » (nouvelle clé i18n `settings.email.changed`, 12/12 langues), URL nettoyée via `history.replaceState`, nouvelle adresse visible via `session.email`. Front-only, pas de route/contrat backend → Swagger non requis. Vérifs : `tsc --noEmit` (0 erreur de type ; échec d'écriture `tsbuildinfo` = artefact root-owned), ESLint OK, parité i18n 12/12. Suggested commit: `feat(settings): retour auto sur les paramètres après changement d'e-mail`.
- 2026-06-15: **fix(settings/mail) email-change delivery** implemented after validation. Console mail transport now returns `ErrDeliveryUnavailable`; mail-service maps it to 503 instead of false 202. `RequestEmailChange` is strict: token creation or delivery failure clears matching `pending_email`; delivery failure also transactionally removes the exact token, then returns 503 `email_delivery_failed`. Registration/admin/reset remain best-effort. Dedicated UI error translated in all 12 locales. Tests lock console error plus handler 503/502. Auth/mail go test+vet, frontend tsc+lint and 124 Vitest tests pass. Root `make swagger` attempted (Windows Error 1312), aggregate regenerated successfully in Linux container. `graphify update .` unavailable because graphify is not installed. Real delivery still requires valid `SMTP_*` credentials in `mail-service/.env`. Suggested commit: `fix(settings): ne valider le changement d'e-mail qu'après envoi réel`.

---

*This file is the lean entry point. Current status, decisions, architecture detail and history live in the
linked files. Keep them updated per the Operating Rules; do not let this file regrow past ~250 lines.*
