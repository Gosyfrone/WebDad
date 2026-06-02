SERVICE := auth-service

.PHONY: run dev build lint test tidy migrate

run:
	go run main.go

# Hot-reload local via air (en stack Docker, utiliser `make dev` à la racine).
dev:
	air -c .air.toml

build:
	go build -o bin/$(SERVICE)

lint:
	golangci-lint run

test:
	go test ./...

tidy:
	go mod tidy

# Applique le schéma sur la base locale (optionnel : le service le fait
# aussi tout seul au démarrage). Même fichier que celui embarqué.
migrate:
	@set -a; . ./.env; set +a; \
	PGPASSWORD=$$POSTGRES_PASSWORD psql \
		-h $${DB_HOST:-localhost} -p $${DB_PORT:-5432} \
		-U $$POSTGRES_USER -d $$POSTGRES_DB \
		-f internal/db/schema.sql
