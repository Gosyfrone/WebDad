# ─── Stage 1 : Build ─────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copier les fichiers de dépendances en premier (cache Docker optimal)
COPY go.mod go.sum ./
RUN for i in 1 2 3; do go mod download && exit 0; sleep 5; done; go mod download

# Copier le code source
COPY . .

# Build : binaire statique, sans debug symbols
RUN CGO_ENABLED=0 GOOS=linux \
    go build -ldflags="-w -s" -o service .

# ─── Stage dev : hot-reload (air) ────────────────────────────────
# Placé AVANT le runtime : `docker build` sans --target construit donc
# toujours l'image de prod (dernier stage). Sélectionné en dev via
# `target: dev` dans docker-compose.dev.yml. Le code source est monté
# en bind mount par compose ; air recompile à chaud à chaque save.
FROM golang:1.25-alpine AS dev
RUN apk add --no-cache git wget ca-certificates tzdata && \
    go install github.com/air-verse/air@v1.52.3
WORKDIR /app
CMD ["air", "-c", ".air.toml"]

# ─── Stage 2 : Runtime ───────────────────────────────────────────
FROM alpine:3.20

# wget pour les healthchecks Docker, ca-certificates pour TLS sortant (SMTP)
RUN apk add --no-cache wget ca-certificates tzdata && \
    addgroup -S app && adduser -S app -G app

WORKDIR /app
COPY --from=builder /app/service /app/service

USER app

EXPOSE 8089

ENTRYPOINT ["/app/service"]
