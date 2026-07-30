.PHONY: build run test clean repl api

build:
	go build -o bin/pipe ./cmd/pipe

api:
	go build -o bin/pipe-api ./cmd/api-server

run: build
	./bin/pipe

test:
	go test ./pkg/...

repl: build
	./bin/pipe

clean:
	rm -rf bin/
	go clean -cache

fmt:
	go fmt ./...
