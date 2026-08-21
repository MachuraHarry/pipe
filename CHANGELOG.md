# Changelog

All notable changes to Pipe are documented here. This file follows [Keep a Changelog](https://keepachangelog.com/) format.

## [Unreleased]

### Fixed
- **pipe-docs MCP server** (`examples/pipe_docs_server.pipe`):
  - `ask_docs` dropped the result when only the German docs index was available (missing return in the DE branch)
  - `refresh_index` was a silent no-op under the locked `server-secure` sandbox (`exec: false` blocks `git pull`); it is now rebuild-only from the cached checkout and its description says so — repository updates require a server restart
  - `index_status` reported `"ready": nil` / `"ai_ready": nil`; state is now initialized up front, `ai_ready` tracks provider configuration and startup errors are exposed as `err`
  - AI failures in `search_docs` / `ask_docs` crashed the tool call; they are now caught and returned as error results
- Unquoted paths in `git clone` (broke with spaces or shell metacharacters)

### Changed
- Shared parser/security helpers moved to `examples/lib/pipe_docs_lib.pipe`, consumed by the MCP server and both offline test scripts (removes three copies of the declaration scanner)
- `pipe_decl` now also indexes `export test` declarations
- Search snippets are truncated UTF-8-safe (word-boundary preferred, incomplete multi-byte tails dropped) instead of raw byte cuts that could produce mojibake
- `read_source` renders numbered lines in linear time and stops rendering at the 500-line cap instead of building the full text first
- `ask_docs` routes German-language questions to the German docs index (umlaut/token detection) instead of always querying EN first, and retrieves 6 chunks per question (was 4)

### Security
- Path gate rejects absolute paths (e.g. `/etc/passwd`) in addition to `..` traversal; covered by `scripts/pipe-docs-security-test.pipe`

---

## [1.0.0] — 2026-08-17

### Production-Ready Release

Pipe v1.0.0 consolidates the entire v0.9.x series into a stable, production-ready release.

### Added
- **Guard clauses** in match expressions (`| pattern if cond -> body`)
- **Concurrency primitives** — channels (`chan`, `send`, `recv`, `try_recv`, `try_send`, `close`), mutex (`mutex`, `lock`, `unlock`, `try_lock`), counting semaphore (`semaphore`, `acquire`, `release`, `try_acquire`)
- **MQTT 5.0 module** — pure Pipe MQTT client with input validation, CONNACK properties, DISCONNECT handling
- **docs-pipe module** — RAG for documentation-native search with heading-aware chunking and web dashboard
- **Constant folding** of literal expressions in the compiler
- **Alias import namespaces** in the bytecode VM
- **Bytecode cache** wired up in the VM path
- **Test framework improvements** — setup/teardown hooks, `assert_near`/`assert_contains`, VM test blocks with `OpTestResult`/`OpTestAbortIfError`
- **True parallelism** for user closures in `>>` and `go`

### Changed
- MCP status promoted to production-ready (was "IN DEVELOPMENT")
- Version bumped from v0.9.4.0 to v1.0.0 across all files
- Updated builtin count: 232 (was 226 in v0.9.3)
- Updated module count: 23 (was 22)
- Updated test count: 643 (was 640)

### Fixed
- Memory leak in `mqtt_publish` — `unregister_pending` now called on PUBACK/PUBREC/PUBCOMP timeout
- `server.json` and `mcpb/manifest.json` version drift (were at 0.9.3.5, now 1.0.0)

### Security
- Sandbox audit rounds 1-6 completed
- Deterministic env masking in sandbox
- Central egress gate for AI provider HTTP
- Input validation for MQTT connect/publish/subscribe

---

## [0.9.4.0]

### Added
- Directory imports (`import "mylib/"` loads `mylib/init.pipe`)
- Relative imports (`./`, `../`)
- Cycle detection (`E009: circular import`) in tree-walker and bytecode VM
- `PIPE_PATH` via `filepath.SplitList`
- SemVer resolution for `^X.Y.Z` constraints
- `pipe -publish` — publishes modules via gh-CLI pull request

---

## [0.9.3.5]

### Added
- Central egress gate for every AI-provider egress
- Recursion-depth guard (E008)
- Zero-dependency MCP server
- X (Twitter) API v2 module
- Discord module
- CI notifications with Discord + Telegram dual delivery

---

## [0.9.0]

### Added
- MCP Server + Client (stdio + SSE)
- Sandbox profiles (declarative runtime security)
- AI provider abstraction (OpenAI, Anthropic, DeepSeek, Ollama)
- Bytecode VM (43 opcodes)
- LSP server
- Formatter
- WASM playground
- Module registry
- GitHub Action
- VSCode extension
- Self-extracting binary builder
