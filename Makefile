.PHONY: build run test

build:
	go build -o bin/server ./cmd/server

run: build
	./bin/server

test:
	go test ./...
