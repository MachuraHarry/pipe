.PHONY: build run test clean repl api lsp vsix fmt

build:
	go build -o bin/pipe ./cmd/pipe

api:
	go build -o bin/pipe-api ./cmd/api-server

lsp:
	go build -o bin/pipe-lsp ./cmd/pipe-lsp

vsix:
	cd vscode && npx --yes @vscode/vsce package

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
