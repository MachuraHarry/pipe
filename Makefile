.PHONY: build run test clean repl

build:
	go build -o bin/pulse ./cmd/pulse

run: build
	./bin/pulse

test:
	go test ./pkg/...

repl: build
	./bin/pulse

clean:
	rm -rf bin/
	go clean -cache

fmt:
	go fmt ./...
