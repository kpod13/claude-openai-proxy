.PHONY: build run test lint ci

build:
	go build -o bin/claude-openai-proxy ./cmd/claude-openai-proxy

run: build
	./bin/claude-openai-proxy

test:
	go test ./...

lint:
	golangci-lint run ./...

ci:
	act push -P ubuntu-latest=catthehacker/ubuntu:act-latest
