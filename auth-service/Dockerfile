# ─── Stage 1 : Build ─────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copier les fichiers de dépendances en premier (cache Docker optimal)
COPY go.mod go.sum ./
RUN go mod download

# Copier le code source
COPY . .

# Build : binaire statique, sans debug symbols
RUN CGO_ENABLED=0 GOOS=linux \
    go build -ldflags="-w -s" -o service .

# ─── Stage 2 : Runtime ───────────────────────────────────────────
FROM alpine:3.20

# wget pour les healthchecks Docker, ca-certificates pour TLS sortant
RUN apk add --no-cache wget ca-certificates tzdata && \
    addgroup -S app && adduser -S app -G app

WORKDIR /app
COPY --from=builder /app/service /app/service

USER app

EXPOSE 8081

ENTRYPOINT ["/app/service"]
