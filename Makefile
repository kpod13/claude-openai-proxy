.PHONY: build run test lint ci

build:
	go build -o bin/server ./cmd/server

run: build
	./bin/server

test:
	go test ./...

lint:
	golangci-lint run ./...

ci:
	act push -P ubuntu-latest=catthehacker/ubuntu:act-latest
