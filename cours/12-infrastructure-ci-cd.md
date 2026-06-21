# 12 — Infrastructure & CI/CD

> Transverse : API Gateway · conteneurisation Docker/compose · observability (slog) · CI/CD (3 workflows) ·
> documentation API (Swagger/OpenAPI) · environnement dev (hot-reload).

## Vue d'ensemble

Le socle qui coche deux critères de notation explicites : **conteneurisation complète** et **architecture cohérente**.
Tout transite par une **gateway maison mince** (« we built it », bon pour la défense), chaque service est **autonome**
(Dockerfile, DB, schéma embarqué), et trois workflows CI gardent la qualité.

## Composants & pourquoi

| Composant | Rôle | Pourquoi ainsi |
|---|---|---|
| **api-gateway** (stdlib) | reverse proxy mince, routage par préfixe, WS proxy, CORS, JWT | maison = défendable ; mince vs BFF par endpoint |
| **Docker / docker-compose** | conteneurisation complète, volumes persistants | exigence de notation |
| **slog (tous les services)** | logging structuré + X-Request-Id | observability, corrélation par requête/utilisateur |
| **GitHub Actions** | `ci-go`, `ci-frontend`, `ci-integration` (+ `deploy`) | qualité + déploiement |
| **Swagger/OpenAPI** | doc code-first (swaggo), publiée Redoc | doc API + drift check |

## Comment c'est codé

### API Gateway
- **Reverse proxy mince (stdlib)** : table préfixe→URL, forward transparent, **préserve le préfixe**. Mince + « on l'a construit » (défense) vs handlers BFF par endpoint (réservés à l'agrégation multi-services).
- **Pas de redirect trailing-slash** : chaque service exposé via **2 routes** (préfixe nu + `/*path`) → un `POST /users` ne 307 pas vers une route inconnue du service.
- **Validation/propagation JWT** (`PropagateJWT`, cf. fiche 01) : « valider-si-présent », anti-spoof, injecte `X-User-Id`/`X-User-Role`/`X-Email-Verified`. `/internal/events` **non routé** (serveur-à-serveur).

### Conteneurisation & dev
- **`docker-compose.yml`** avec tous les services + **volumes persistants** pour les DBs ; `.env` gitignoré.
- **`make dev` = overlay `docker-compose.dev.yml`** (Go : `air` hot-reload + bind-mount + caches Go partagés ; frontend : `next dev` + `WATCHPACK_POLLING`). Le stage `dev` du Dockerfile précède le stage runtime → `make up`/`build` produisent toujours l'image **prod** (dernier stage). **`make up` = images prod gelées** (pas de rebuild sur changement source).
- **Layering `.env`** : racine = vars transverses (`JWT_SECRET`, `INTERNAL_EVENT_SECRET`, `NEXT_PUBLIC_API_URL`, seule la racine est interpolable par compose) ; `<service>/.env` = config propre via `env_file:`.

### Observability (slog)
- **`log/slog` stdlib** (aucune dépendance externe), JSON en release / texte en debug, niveau via `LOG_LEVEL`, attribut `service`. Trois middlewares : `RequestID` (UUID hex propagé), `Recovery` (panic → `slog.Error` + 500 sans stack exposée), `RequestLogger` (une ligne/requête : method, path sans query, status, latency, client_ip, request_id, user_id). Le JWT middleware pose `user_id` → corrélation utilisateur. `gin.New()` + chaîne explicite (pas `gin.Default()`).

### CI/CD
- **3 workflows par domaine** : `ci-go` (matrice par service : gofmt + vet + build + `test -race` + tidy + golangci-lint v2 + govulncheck), `ci-frontend` (lint + build/typecheck), `ci-integration` (docker compose up de la stack moins le frontend, attente des healthchecks). Fichiers séparés = triggers/path-filters indépendants ; matrice = parallélisme + isolation par module. `govulncheck = 0 vuln` depuis Go 1.25.
- **Job swagger** (`ci-go.yml`) : drift check (`git diff --exit-code doc/`) + validation Swagger 2.0 → **bloquant sur PR**.
- **Smoke-test post-déploiement** (`deploy.yml`) : runner GitHub (vue externe), vérifie `API/health` + front en 200 HTTPS (3 tentatives) — aurait attrapé le bug Cloudflare (TLS edge KO alors que les conteneurs étaient sains).

### Documentation API
- **Annotations dans les handlers** (swaggo/swag code-first), jamais en DTO séparés. **`make swagger` avant chaque commit** touchant un handler → régénère `doc/openapi.{json,yaml}` (spec agrégé commité, `docs/` par service gitignorés). **Publiée Redoc** sur `https://gosyfrone.github.io/WebDad/` (auto-déployée au push develop) ; preview local `make swagger-site` (port 8088).

## Décisions clés (et alternatives écartées)

- **Gateway maison stdlib** plutôt qu'un gateway tiers (Kong/Traefik) ou des BFF par endpoint : mince, défendable (« on l'a construit »), suffisant ; les BFF restent pour l'agrégation multi-services.
- **2 routes par service** (préfixe nu + wildcard) plutôt qu'un redirect trailing-slash : un `POST` sur collection ne se fait pas 307.
- **slog stdlib** plutôt qu'une lib de logging tierce : zéro dépendance, JSON/texte selon l'env, corrélation par request-id.
- **3 workflows séparés** plutôt qu'un mono-workflow : triggers/path-filters indépendants, matrice = parallélisme + isolation.
- **Swagger code-first dans les handlers + drift check CI** : la doc ne dérive jamais du code (bloquant sur PR).
- **`make dev` overlay** (stage dev avant runtime) : le hot-reload n'altère jamais l'image prod (toujours le dernier stage).

## Points de défense / Q&A anticipées

- *« Conteneurisation complète ? »* → oui : tous les services + DBs en docker-compose, volumes persistants, `ci-integration` matérialise la santé de la stack.
- *« Pourquoi un gateway maison ? »* → mince, on en maîtrise chaque ligne (validation JWT, anti-spoof), bon argument oral ; un gateway tiers serait une boîte noire à défendre.
- *« Comment garantir que la doc API est à jour ? »* → drift check CI bloquant + publication automatique au push.

## Limites assumées / perspectives

- **Perspective** : gitleaks, Dependabot, CD (push d'images vers GHCR).
- **Faire consommer** les en-têtes propagés par les services (alléger leur revalidation JWT).
