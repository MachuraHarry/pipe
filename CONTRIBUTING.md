# Contributing to Pipe

## Setup

```bash
git clone https://github.com/MachuraHarry/pipe
cd pipe
make build
make test
```

## Project Structure

The project follows Go conventions. Key directories:

```
pkg/lexer/     — Tokenizer with INDENT/DEDENT tracking
pkg/parser/    — Recursive descent + Pratt parser
pkg/ast/       — AST node definitions (27 types)
pkg/object/    — Runtime values + stdlib builtins
pkg/eval/      — Tree-walk interpreter
pkg/compiler/  — AST → Bytecode compiler
pkg/vm/        — Stack-based virtual machine
pkg/mcp/       — MCP (Model Context Protocol) server + client
pkg/ai/        — AI provider abstraction (OpenAI, Anthropic, DeepSeek)
cmd/pipe/      — CLI entry point, REPL
cmd/pipe-lsp/  — Language server for the VS Code extension
cmd/wasm/      — Browser/WASM build for the website playground
docs/          — Full documentation (DE + EN)
test/integration/ — Pipe-language integration tests
vscode/        — VS Code extension (syntax highlighting + LSP)
```

## Building

```bash
make build        # Build binary → bin/pipe
make test         # Run Go unit tests + Pipe integration tests
make fmt          # Format Go code
```

## Running Tests

```bash
go test ./pkg/...                           # All Go unit tests
go test -v -race -cover ./pkg/...           # With race detection + coverage
make test-integration                       # Pipe integration tests (test/integration)
./bin/pipe -test                            # Run *._test.pipe in the current directory
```

## Before Submitting

- Format your code: `gofmt -w .`
- All tests must pass: `make test`
- Add tests for new functionality
- Update documentation in `docs/` for user-facing changes
- Update the version in `cmd/pipe/main.go`, `mcpb/manifest.json`, and `server.json` when releasing

## Commit Convention

- Language features: `Feature: short description`
- Bugfixes: `Fix: short description`
- Documentation: `Docs: short description`
- Examples/Tests: `Test: short description`

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
