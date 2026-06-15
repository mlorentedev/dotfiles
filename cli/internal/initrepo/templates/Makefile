.PHONY: build test lint run

build:
	go build -o bin/app ./cmd/...

test:
	go test -race -cover ./...

lint:
	golangci-lint run

run:
	go run ./cmd/...
