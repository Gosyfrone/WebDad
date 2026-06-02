SERVICE := post-service

.PHONY: run dev build lint test tidy

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
