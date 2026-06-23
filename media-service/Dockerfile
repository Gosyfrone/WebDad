# ─── Stage 1 : Build ─────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata nodejs npm vips-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN for i in 1 2 3; do go mod download && exit 0; sleep 5; done; go mod download

COPY package.json package-lock.json ./
RUN npm ci --omit=dev

COPY . .

RUN CGO_ENABLED=0 GOOS=linux \
    go build -ldflags="-w -s" -o service .

# ─── Stage dev : hot-reload (air) ────────────────────────────────
# Placé AVANT le runtime : `docker build` sans --target construit donc
# toujours l'image de prod (dernier stage). Sélectionné en dev via
# `target: dev` dans docker-compose.dev.yml. Le code source est monté
# en bind mount par compose ; air recompile à chaud à chaque save.
FROM golang:1.25-alpine AS dev
RUN apk add --no-cache git wget ca-certificates tzdata nodejs npm vips-dev && \
    go install github.com/air-verse/air@v1.52.3
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci --omit=dev
CMD ["air", "-c", ".air.toml"]

# ─── Stage 2 : Runtime ───────────────────────────────────────────
FROM alpine:3.20

RUN apk add --no-cache wget ca-certificates tzdata nodejs vips && \
    addgroup -S app && adduser -S app -G app

WORKDIR /app
COPY --from=builder /app/service /app/service
COPY --from=builder /app/node_modules /app/node_modules
COPY --from=builder /app/package.json /app/package.json
COPY --from=builder /app/scripts /app/scripts

USER app

EXPOSE 8087

ENTRYPOINT ["/app/service"]
