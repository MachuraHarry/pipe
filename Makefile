.PHONY: build run test test-parity test-integration clean repl api lsp vsix fmt stats stats-check docs-dashboard

# Regenerate stats.json + sync README/website numbers (commit the result so CI can check for drift).
stats:
	go run ./scripts/stats

# Verify the committed stats.json + README/website numbers still match the live numbers.
stats-check:
	go run ./scripts/stats && git diff --exit-code -- stats.json README.md website/index.html website/docs.html

build:
	go build -ldflags="-s -w" -trimpath -o bin/pipe ./cmd/pipe

build-upx: build
	upx -q bin/pipe -o bin/pipe-upx 2>/dev/null || (echo "Install UPX: apt install upx-ucl" && false)

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

test-parity:
	go test -count=1 -v ./pkg/parity/

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
