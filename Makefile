.PHONY: build run test clean repl

build:
	go build -o bin/pipe ./cmd/pipe

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
