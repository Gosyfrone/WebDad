.PHONY: help env up dev dev-down dev-logs down build logs ps clean reset db-only \
        logs-gateway logs-auth logs-user logs-profil logs-post logs-message logs-notification logs-front logs-db \
        sh-auth sh-user sh-profil sh-post sh-message sh-notification sh-gateway \
        psql-auth psql-user mongo-profil-cli mongo-post-cli mongo-message-cli mongo-notification-cli

# Services possédant un .env propre (chargé par compose via env_file)
SERVICES := auth-service user-service profil-service post-service message-service notification-service api-gateway

# Invocation compose en mode DEV (overlay hot-reload par-dessus la base)
DEV := docker compose -f docker-compose.yml -f docker-compose.dev.yml

# ─── Aide ────────────────────────────────────────────────────────
help:
	@echo ""
	@echo "  WebDad — Commandes disponibles"
	@echo "  ────────────────────────────────────────────────────"
	@echo "  make env       Créer les .env manquants depuis les .env.example"
	@echo "  make up        Démarrer tout l'environnement (images de prod)"
	@echo "  make dev       Démarrer en mode DEV (détaché) : hot-reload front + Go (air)"
	@echo "  make dev-logs  Suivre les logs front + Go (sans le bruit des BDD)"
	@echo "  make dev-down  Arrêter la stack de dev"
	@echo "  make down      Arrêter les conteneurs"
	@echo "  make build     Rebuild toutes les images (--no-cache)"
	@echo "  make logs      Suivre les logs (tous les services)"
	@echo "  make ps        Statut des conteneurs"
	@echo "  make clean     Supprimer conteneurs + images locales"
	@echo "  make reset     Reset complet (données supprimées)"
	@echo "  make db-only   Démarrer seulement les BDD"
	@echo ""
	@echo "  Logs d'un service : make logs-auth | logs-user | logs-profil | logs-post | logs-message | logs-notification | logs-front | logs-gateway | logs-db"
	@echo "  Shell d'un service: make sh-auth | sh-user | sh-profil | sh-post | sh-message | sh-notification | sh-gateway"
	@echo "  CLI BDD           : make psql-auth | psql-user | mongo-profil-cli | mongo-post-cli | mongo-message-cli | mongo-notification-cli"
	@echo ""

# ─── Préparation des .env ─────────────────────────────────────────
# Crée les .env manquants (racine + chaque service) à partir des .env.example.
# N'écrase jamais un .env existant.
env:
	@test -f .env || { cp .env.example .env; echo "✓ créé .env"; }
	@for s in $(SERVICES); do \
		test -f $$s/.env || { cp $$s/.env.example $$s/.env; echo "✓ créé $$s/.env"; }; \
	done
	@echo "✓ .env prêts — pense à renseigner les secrets (JWT_SECRET, mots de passe)"

# ─── Stack complète ───────────────────────────────────────────────
up:
	@test -f .env || { echo "❌ .env racine manquant — exécute : make env"; exit 1; }
	@for s in $(SERVICES); do \
		test -f $$s/.env || { echo "❌ $$s/.env manquant (requis par compose) — exécute : make env"; exit 1; }; \
	done
	docker compose up -d
	@echo ""
	@echo "  ✓ WebDad démarré"
	@echo "    Gateway  → http://localhost:8080"
	@echo "    Frontend → http://localhost:3000"
	@echo ""

down:
	docker compose down

# ─── Mode DEV (hot-reload) ────────────────────────────────────────
# Monte le code source en bind mount et lance les watchers :
#   - Go        : air (recompile à la sauvegarde) via le stage `dev`
#   - Frontend  : next dev (HMR)
# `--build` force la construction du stage `dev` (sinon compose
# réutiliserait l'image de prod déjà taggée). Lancé en DÉTACHÉ : la
# commande rend la main, on consulte les logs à la demande (dev-logs).
dev:
	@test -f .env || { echo "❌ .env racine manquant — exécute : make env"; exit 1; }
	@for s in $(SERVICES); do \
		test -f $$s/.env || { echo "❌ $$s/.env manquant (requis par compose) — exécute : make env"; exit 1; }; \
	done
	$(DEV) up --build -d
	@echo ""
	@echo "  ▶ Mode DEV démarré (hot-reload front + Go)"
	@echo "    Frontend → http://localhost:3000   (édite un fichier → reload auto)"
	@echo "    Gateway  → http://localhost:8080"
	@echo "    Logs     → make dev-logs (front + Go)  |  make logs-front | logs-post ..."
	@echo "    Arrêt    → make dev-down"
	@echo ""

# Logs des services applicatifs (front + Go), sans le bruit des BDD.
dev-logs:
	$(DEV) logs -f frontend api-gateway auth-service user-service profil-service post-service message-service notification-service

dev-down:
	$(DEV) down

build:
	docker compose build --no-cache

logs:
	docker compose logs -f

ps:
	docker compose ps

# ─── BDD uniquement (pour dev sans rebuilder les services Go) ─────
db-only:
	docker compose up -d postgres-auth postgres-user mongo-profil mongo-post mongo-message mongo-notification
	@echo "BDD démarrées. Attente des healthchecks..."
	@sleep 3
	docker compose ps

# ─── Reset ────────────────────────────────────────────────────────
clean:
	docker compose down --rmi local

reset:
	docker compose down -v --rmi local
	@echo "Reset complet effectué (données supprimées)"

# ─── Logs par service ─────────────────────────────────────────────
logs-gateway:
	docker compose logs -f api-gateway

logs-auth:
	docker compose logs -f auth-service

logs-user:
	docker compose logs -f user-service

logs-profil:
	docker compose logs -f profil-service

logs-post:
	docker compose logs -f post-service

logs-message:
	docker compose logs -f message-service

logs-notification:
	docker compose logs -f notification-service

logs-front:
	docker compose logs -f frontend

logs-db:
	docker compose logs -f postgres-auth postgres-user mongo-profil mongo-post mongo-message mongo-notification

# ─── Shell dans les conteneurs ────────────────────────────────────
sh-auth:
	docker compose exec auth-service sh

sh-user:
	docker compose exec user-service sh

sh-profil:
	docker compose exec profil-service sh

sh-post:
	docker compose exec post-service sh

sh-message:
	docker compose exec message-service sh

sh-notification:
	docker compose exec notification-service sh

sh-gateway:
	docker compose exec api-gateway sh

# ─── Psql direct ──────────────────────────────────────────────────
# Les identifiants sont lus DANS le conteneur (variables POSTGRES_* injectées
# par env_file), donc valables quelles que soient les valeurs du .env.
psql-auth:
	docker compose exec postgres-auth sh -c 'psql -U "$$POSTGRES_USER" -d "$$POSTGRES_DB"'

psql-user:
	docker compose exec postgres-user sh -c 'psql -U "$$POSTGRES_USER" -d "$$POSTGRES_DB"'

# ─── Mongosh direct ───────────────────────────────────────────────
mongo-profil-cli:
	docker compose exec mongo-profil sh -c 'mongosh -u "$$MONGO_INITDB_ROOT_USERNAME" -p "$$MONGO_INITDB_ROOT_PASSWORD" --authenticationDatabase admin "$$MONGO_INITDB_DATABASE"'

mongo-post-cli:
	docker compose exec mongo-post sh -c 'mongosh -u "$$MONGO_INITDB_ROOT_USERNAME" -p "$$MONGO_INITDB_ROOT_PASSWORD" --authenticationDatabase admin "$$MONGO_INITDB_DATABASE"'

mongo-message-cli:
	docker compose exec mongo-message sh -c 'mongosh -u "$$MONGO_INITDB_ROOT_USERNAME" -p "$$MONGO_INITDB_ROOT_PASSWORD" --authenticationDatabase admin "$$MONGO_INITDB_DATABASE"'

mongo-notification-cli:
	docker compose exec mongo-notification sh -c 'mongosh -u "$$MONGO_INITDB_ROOT_USERNAME" -p "$$MONGO_INITDB_ROOT_PASSWORD" --authenticationDatabase admin "$$MONGO_INITDB_DATABASE"'
