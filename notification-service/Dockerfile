# ─── Stage 1 : Build ─────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

COPY go.mod go.sum ./
# proxy.golang.org est parfois instable en HTTP/2 (stream error / INTERNAL_ERROR
# received from peer) : on force HTTP/1.1 et on réessaie jusqu'à 5 fois.
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux \
    go build -ldflags="-w -s" -o service .

# ─── Stage dev : hot-reload (air) ────────────────────────────────
# Placé AVANT le runtime : `docker build` sans --target construit donc
# toujours l'image de prod (dernier stage). Sélectionné en dev via
# `target: dev` dans docker-compose.dev.yml.
FROM golang:1.25-alpine AS dev
RUN apk add --no-cache git wget ca-certificates tzdata && \
    go install github.com/air-verse/air@v1.52.3
WORKDIR /app
CMD ["air", "-c", ".air.toml"]

# ─── Stage 2 : Runtime ───────────────────────────────────────────
FROM alpine:3.20

RUN apk add --no-cache wget ca-certificates tzdata && \
    addgroup -S app && adduser -S app -G app

WORKDIR /app
COPY --from=builder /app/service /app/service

USER app

EXPOSE 8086

ENTRYPOINT ["/app/service"]
