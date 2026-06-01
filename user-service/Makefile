SERVICE := user-service

.PHONY: run build lint test tidy

run:
	go run main.go

build:
	go build -o bin/$(SERVICE)

lint:
	golangci-lint run

test:
	go test ./...

tidy:
	go mod tidy
