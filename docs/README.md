# Pipe Documentation

Welcome to the official documentation for **Pipe** — a minimalist, pipeline-based scripting language implemented in Go. The entire grammar fits on one page, inspired by Lua's minimalism but with a modern pipeline syntax.

Pipe combines **Python-like readability** (indentation-based, no braces) with **Unix shell pipelines** and the **portability of a single Go binary** (~7 MB, zero external dependencies).

## Documentation Languages

- [Deutsch (German)](/docs/de/index.md)
- [English](/docs/en/index.md)

## Quick Links

| Topic | German | English |
|-------|--------|---------|
| Getting Started | [Erste Schritte](/docs/de/01-erste-schritte.md) | [Getting Started](/docs/en/01-getting-started.md) |
| Language Tour | [Sprachübersicht](/docs/de/02-sprachuebersicht.md) | [Language Tour](/docs/en/02-language-tour.md) |
| Builtin Reference | [Builtin-Referenz](/docs/de/10-builtin-referenz.md) | [Builtin Reference](/docs/en/10-builtin-reference.md) |
| Architecture | [Architektur](/docs/de/14-architektur.md) | [Architecture](/docs/en/14-architecture.md) |

## About Pipe

- **Version**: 0.9.3.5
- **Implementation**: Go 1.25+
- **License**: MIT
- **Binary size**: ~7 MB (dependency-free, statically linked)
- **Tests**: 416 (across 14 packages)
- **Builtins**: 198
- **Modules**: 21 (curated, installable via `pipe -get`)
- **Opcodes**: 42
- **AST node types**: 35
- **Example programs**: ~60

## Project Links

- Repository: `https://github.com/MachuraHarry/pipe`
- VSCode Extension: Included in `vscode/`
