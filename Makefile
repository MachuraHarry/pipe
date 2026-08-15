.PHONY: build run test clean repl api lsp vsix fmt stats docs-dashboard

stats:
	go run ./scripts/stats

build:
	go build -ldflags="-s -w" -trimpath -o bin/pipe ./cmd/pipe

build-lite:
	go build -tags pipe_lite -ldflags="-s -w" -trimpath -o bin/pipe-lite ./cmd/pipe

build-upx: build
	upx -q bin/pipe -o bin/pipe-upx 2>/dev/null || (echo "Install UPX: apt install upx-ucl" && false)

build-lite-upx: build-lite
	upx -q bin/pipe-lite -o bin/pipe-lite-upx 2>/dev/null || (echo "Install UPX: apt install upx-ucl" && false)

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
	$(MAKE) test-integration

test-integration:
	go build -o bin/pipe ./cmd/pipe
	cd test/integration && ../../bin/pipe -test

repl: build
	./bin/pipe

docs-dashboard: build
	./scripts/docs-dashboard.sh

clean:
	rm -rf bin/
	go clean -cache

fmt:
	go fmt ./...
