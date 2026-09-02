# Changelog

All notable changes to Pipe are documented here. This file follows [Keep a Changelog](https://keepachangelog.com/) format.

## [Unreleased]

### Added
- **`ai_swarm_stream`: optional round-check callback (5th argument)** — a Pipe closure invoked with no arguments at the start of every swarm round, letting a caller (e.g. a Telegram bot) intervene in an in-flight run without any new concurrency (it runs synchronously, same goroutine/VM, at an existing checkpoint). It may return a map with any of `abort` (bool), `abort_reason` (string), `inject` (string) — every field optional; `nil`/a non-map/an empty map is a fully inert round. A truthy `abort` stops the run immediately (result then has `aborted`=true, `abort_reason` set, `content` reflecting whatever partial progress was made); a non-empty `inject` is appended to the live conversation as a new instruction before the round proceeds, letting a caller steer an in-flight run with a fresh message. Go-level API: `ai.ChatSwarm`'s trailing parameter is now `SwarmRoundCheck func() SwarmRoundAction`.
- **`ai_swarm_stream`: `"reasoning"` progress event** — the 4th-argument progress callback now also fires with `event="reasoning"` and the model's raw chain-of-thought in `detail`, whenever the provider returns one (e.g. DeepSeek reasoner models) — including on a final, non-tool-calling round, which previously discarded `ReasoningContent` entirely without surfacing it anywhere.

---

## [1.2.0] — 2026-08-30

### Added
- **`ai_swarm` / `ai_swarm_trace` / `swarm_agent`** — handoff multi-agent builtin: named agents, each with their own system prompt and tools, transfer a conversation to one another via a reserved tool call while the full message history carries forward (the pattern popularized by OpenAI's original "Swarm"). `ai_swarm` returns the final agent's answer as a plain string; `ai_swarm_trace` also returns `{content, path, rounds}` for observability. Gated exactly like `ai_chat` (profile `ai: false` / CLI `--sandbox` + `--allow-ai`). OpenAI-compatible providers only (same constraint as `ai_with_tools`). See [AI Builtins §19.14](docs/en/19-ai-builtins.md).
- **`ai_vision`** — image understanding: answers a question about an image (an http(s) URL, a local file path, or raw bytes — JPEG/PNG/GIF/WebP, content-sniffed) via an OpenAI-compatible vision model (e.g. DeepSeek's `deepseek-v4-flash-vision-exp`). Built as a self-contained request path rather than widening the shared `Message` type, so it needed zero changes to the 6 existing provider implementations. See [AI Builtins §19.15](docs/en/19-ai-builtins.md).
- **AI provider: OpenCode Zen (`opencode`)** — 6th provider, gateway at `https://opencode.ai/zen/v1/chat/completions` with free-tier models (`big-pickle`, `deepseek-v4-flash-free`, `mimo-v2.5-free`, …) usable **without any API key**: pipe sends the public bearer token plus the `x-opencode-*` client headers the Zen API requires (unique per-request session/request IDs). Setting `OPENCODE_API_KEY` (env or `ai_set_key "opencode" <key>`) switches to authenticated requests for paid models and higher limits. Free models estimate to $0 in cost tracking (mirrors OpenRouter `:free`). Reasoning-model quirks are handled: generous default `max_tokens: 2048` and a clear error when a response contains only reasoning tokens without content. Default model is `big-pickle`. Note: free-tier prompts may be used for model training; the keyless mode relies on an unofficial API contract and can break upstream.
- **Built-in self-updater**: `pipe --update` checks the latest GitHub release and, if newer, downloads `pipe-<os>-<arch>.tar.gz`, verifies its SHA256 checksum and replaces the running binary atomically (Windows-safe rename dance). `pipe --update-check` only reports whether a new release exists; `pipe --version` prints the current version. The updater compares base semvers, so `git describe` dev builds (`v1.1.1-3-gabc1234`, `-dirty`) do not false-positive against their own release tag. Self-extracting binaries (`pipe -build`) refuse to update unless `PIPE_UPDATE_EMBEDDED=1` is set, since updating would discard the embedded program. Local `make build` now injects the version via `git describe --tags --always --dirty`.
- **Installers** (`install.sh` / `install.ps1`) now print a "Keep current: pipe --update" hint after a successful install.
- **CI**: `go vet ./...` added as a gate in the test job (format check + vet + tests + build + integration tests).

### Fixed
- **Sandbox (round 7 audit)**: the CLI `--sandbox` flag (no profile) didn't gate 6 of 8 filesystem-write builtins — `write_file`, `append_file`, `file_move`, `file_copy`, `make_dir`, and `file_open` in write-capable modes let real writes through; only `file_delete`/`remove_dir` were already correctly gated. Fixed via a centralized `checkFSWriteAccess` helper, mirroring the round-6 network-gate fix. Read-only builtins are unaffected by design.
- **Sandbox (round 8 audit)**: `wiki_search` never reached the central AI egress gate at all (unlike its sibling `web_search`) — under `pipe --sandbox`, a query still reached the real Wikipedia API. Fixed by routing it through the same `gateEgress` call every other AI/search egress path already uses.
- **`.pipec` bytecode cache**: the compiler bakes each builtin's *position* in `object.Builtins` into the bytecode as an integer index; inserting a builtin anywhere but the end silently shifted every later builtin's index, and a stale on-disk cache compiled against the old table would still look "valid" and resolve a builtin call to the *wrong function*. The cache's dependency hash now also covers the ordered builtin-name table, so any future change to it self-invalidates every `.pipec` on disk automatically — no longer dependent on remembering to bump `compiler.CacheVersion` for a change that isn't a bytecode-format change.
- **`pipe --update`**: previously failed *open* if the release's `.sha256` checksum file couldn't be downloaded, installing an unverified binary. Now refuses the update and reports an error instead.
- **Compiler**: a bare `catch` without a bound parameter is now accepted.
- **`input`**: now reads full lines instead of truncating.
- Removed a dead, unused `source` variable in `WebSearch` (no behavior change).

---

## [1.1.1] — 2026-08-23

### Added
- **repo-rag MCP server**: new `file_symbols` tool — outline of one source file (every indexed declaration with kind, name, line and declaration text, in file order). Answers purely from the in-memory index, so it works under the locked read-only serve profile.
- **repo-rag MCP server**: OpenRouter support — `OPENROUTER_API_KEY` now enables `ask_docs` with any OpenRouter model (`REPO_RAG_MODEL` overrides; default is a `:free` model, e.g. `nvidia/nemotron-3-super-120b-a12b:free`). `openrouter.ai` was added to both sandbox network whitelists. Since OpenRouter offers no embeddings endpoint, `ask_docs` now falls back to the keyword chunk index when semantic retrieval yields nothing, so answers stay grounded with cited sources instead of degrading to general knowledge.

### Fixed
- **repo-rag MCP server: `REPO_RAG_REF` is now validated** — the ref went unvalidated into the same double-quoted shell argument as the URL (`git clone --branch "<ref>"`), allowing shell injection via a quote or git flag injection via a leading `-` (`--upload-pack=...`). Refs are now held to an allowlist (letters, digits, `.` `_` `-` `/`), must not start with `-`, and may not contain `..`. Invalid refs fail startup with a clean error instead of reaching the shell.
- **repo-rag / pipe-docs lib: Windows absolute paths rejected** — `safe_resolve_path` only checked `..` and a leading `/`, so Windows drive paths (`C:\Windows\system32`) passed the gate on windows-amd64 builds. Backslashes are now normalized before the checks and drive-letter patterns are rejected; covered by new cases in `scripts/pipe-docs-security-test.pipe`.
- **AI cost estimation: OpenRouter free models cost 0** — `nvidia/nemotron-3-super-120b-a12b:free` (and any other `:free` slug) matched no substring in the price table and fell through to the ~GPT-4 default, so $0 calls accumulated phantom costs that could exhaust Budget sandbox limits. Free slugs now estimate to exactly 0 (`TestEstimateCostFreeModels`).
- **Parser: `[` after an identifier in call position** — `map [1, 2, 3] (fn x: x * 10)` was mis-parsed as an index expression `map[...]` and failed at runtime. A `[` separated by whitespace from the previous token now starts a list-literal call argument; an adjacent `[` (`xs[0]`, `xs[1..3]`) remains index/slice access. Documented in both language guides.
- **Compiler: double emission of index operands** — `xs[0]` lowered to `at(xs, 0)` but the generic infix path pre-emitted the operands first, so they were compiled twice: broken stack under `-vm` ("not a function: INTEGER") and duplicated side effects. Index access is now lowered before the generic path; covered by `TestCompileIndexNoDoubleEmission`.
- **VM**: `executeFrame` (the reduced interpreter used when builtins such as `map`/`filter`/`reduce` call back into user functions) was missing `OpMap`, `OpStruct`, `OpSelect` and `OpHalt`; any map literal inside such a callback failed under `-vm` with "unknown opcode in user fn: 35". The opcodes now mirror the main loop's semantics.
- **Compiler: module symbol isolation** (`959a05c`) — unaliased imports no longer flatten into one global symbol space; modules compile in an isolated scope (`moduleMode`), see only their exports plus builtins, and importers bind aliases via fresh global slots. Fixes builtin shadowing under `-vm` (e.g. sqlite's internal `exec` hiding builtin `exec`). Bytecode cache bumped to v5 (old `.pipec` files are invalidated).
- **Compiler: while-body terminator detection** (`e6f42b7`) — statements after an inner loop at the tail of an outer loop body were dropped during bytecode generation (caused an infinite INSERT loop in the sqlite module under `-vm`).
- **Compiler: module resolve walks the full outer chain** (`38446d7`) — a same-named global in an importing file (e.g. `repo_rag_lib.pipe`'s `index_of`) masked the root builtin for nested modules and failed compilation with "undefined variable".
- **Disassembler**: `Instructions.String()` now handles `OpTestAbortIfError`, `OpTestResult` and `OpSelect` (2-byte operands) and bounds-checks all operand reads — no more panics when disassembling programs containing these opcodes.
- **pkg/parity** no longer hangs: stale untracked `.pipec` artifacts were executed because the cache format did not track compiler semantics (fixed by the v5 cache version); all deterministic examples now produce byte-identical output under tree-walker and VM.
- **pipe-docs MCP server**: `search_code` splits query tokens on `_` and `-` so snake_case queries match Go/Pipe symbols; `ask_docs` requires ≥ 2 significant-token hits plus score ≥ 0.1 before a chunk enters LLM context (suppresses generic-word noise hits on German queries)
- **sqlite module 0.8.3 / 0.8.4** (pipe-modules): UTF-8-safe SQL lexer and LIKE matcher, support for `CREATE TABLE IF NOT EXISTS` and `CREATE INDEX IF NOT EXISTS`.

### Changed
- pipe-docs noai end-to-end test unblocked (`pkg/mcp/pipe_docs_noai_test.go`)

### Removed
- Temporary `PIPE_VM_TICK` instruction sampler from the VM (debugging aid for the terminator investigation).

---

## [1.1.0] — 2026-08-21

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
