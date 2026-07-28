# Contributing to Pipe

## Setup

```bash
git clone https://github.com/harry/pipe
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
pkg/object/    — Runtime values + 81 stdlib + 25 AI builtins
pkg/eval/      — Tree-walk interpreter
pkg/compiler/  — AST → Bytecode compiler
pkg/vm/        — Stack-based virtual machine
pkg/ai/        — AI provider abstraction (OpenAI, Anthropic, DeepSeek)
cmd/pipe/      — CLI entry point, REPL
docs/          — Full documentation (DE + EN, 41 chapters)
```

## Building

```bash
make build        # Build binary → bin/pipe
make test         # Run all tests
make fmt          # Format Go code
```

## Running Tests

```bash
go test ./pkg/...                           # All unit tests
go test -v -race -cover ./pkg/...           # With race detection + coverage
./bin/pipe -test                            # Pipe integration tests
```

## Before Submitting

- Format your code: `gofmt -w .`
- All tests must pass: `go test ./pkg/...`
- Add tests for new functionality
- Update documentation in `docs/` for user-facing changes

## Commit Convention

- Language features: `Feature: short description`
- Bugfixes: `Fix: short description`  
- Documentation: `Docs: short description`
- Examples/Tests: `Test: short description`

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
