# Roadmap

This document outlines the past, present, and future of the Pipe language. The roadmap is organized into phases, with completed features marked accordingly.

## Current Version: v0.9.4.0

Pipe is currently at version **v0.9.4.0**, the **module-system release**. It completes the module system on the road to v1.0: **directory imports** (`import "mylib/"` loads `mylib/init.pipe`), dedicated **relative imports** (`./`, `../` — never through the registry fallback), **cycle detection** (`E009: circular import` in both the tree-walker and the bytecode VM, including alias imports), `PIPE_PATH` via `filepath.SplitList` (platform-native separators), **SemVer resolution** for `^X.Y.Z` constraints in `pipe -install`, and **`pipe -publish`**: publishes modules via a gh-CLI pull request into the registry (validation, version-duplicate check, module dir + registry.json entry, branch `publish/<name>-<version>`). The language feature set remains that of the sandbox-hardening release **v0.9.3.5**: a central egress gate for every AI-provider egress, a recursion-depth guard (E008), the zero-dependency MCP server, the X (Twitter) API v2 module and the Discord module, and CI notifications with Discord + Telegram dual delivery including AI code reviews.

---

## v0.7 — Ecosystem, Self-Healing & Tooling

> **Status: Completed**

- [x] Parallel pipeline operator (`>>`) with Future auto-resolution
- [x] `try_ai` — AI-powered self-healing error recovery (optional `catch`)
- [x] Module versioning (`import "mod@1.0.0"`), `pipe -get`, `pipe -search`
- [x] C-style `for` loops (`for i: 0; i < 5; i: i + 1`) in Tree-Walker and Bytecode VM
- [x] `not` keyword (synonym for `!`)
- [x] Multi-pattern match cases (`| 1 | 2 | 3 -> "small"`)
- [x] Test framework: `test` blocks + `assert`, `assert_eq`, `assert_error`, `assert_lt`, `assert_gt`, `assert_not_eq`, `pipe -test`
- [x] Sandbox profiles — declarative runtime security for AI agents and untrusted code
- [x] Error codes (E001–E004) with contextual parser hints
- [x] Formatter `--check` + directory processing, REPL history persistence
- [x] GitHub Action (`pipe-action`) for CI/CD
- [x] HTTP API server (`cmd/api-server`) with Fly.io deployment
- [x] WASM playground v2 (code sharing, syntax highlighting, rating) + blog
- [x] 230+ Go tests, 8 integration suites, 42 example programs
- [x] Inline lambda syntax (`fn x: expr`) — single-expression anonymous functions in both Tree‑Walker and Bytecode VM

---

## Completed Features (v0.1 – v0.5)

### v0.1 — Foundation

- [x] Lexer with 52 token types and INDENT/DEDENT mechanism
- [x] Parser with recursive descent + Pratt parsing (13 precedence levels)
- [x] AST with 27 node types (12 statements, 15 expressions)
- [x] Tree-walking interpreter
- [x] Basic data types: Integer, Float, String, Boolean, Nil
- [x] Variables, functions, closures
- [x] Control flow: if/else, while, for-in, match
- [x] Error handling: try/catch
- [x] Lists and Maps
- [x] Pipeline operator (`>`)
- [x] Comments and string literals

### v0.2 — Standard Library

- [x] I/O: print, input, read_file, write_file, append_file, read_lines
- [x] File system: file_exists, file_delete, file_move, file_copy, file_size, file_type
- [x] Directory: list_dir, make_dir, remove_dir
- [x] Path utilities: path_join, path_base, path_dir, path_ext
- [x] String functions: upper, lower, trim, split, join, contains
- [x] List functions: len, push, pop, at, sort, range
- [x] Higher-order: map, filter, reduce, each
- [x] Map functions: get, set, keys, values
- [x] Type checks: type_of, is_num, is_str, is_list, is_map, is_nil
- [x] Conversion: to_str, to_num
- [x] System: exec, env, sleep
- [x] Math: abs, min, max, pow, sqrt, round

### v0.3 — Networking & Advanced Types

- [x] HTTP client: http_get, http_post, http_get_json
- [x] JSON: parse_json, to_json
- [x] TCP: tcp_listen, tcp_connect, tcp_accept, tcp_read, tcp_write, tcp_close
- [x] Regex: regex_match, regex_replace
- [x] Date/Time: now, format_time
- [x] Random: random, random_range
- [x] Encoding: base64_encode, base64_decode
- [x] Result type: Ok, Err, is_ok, is_err, unwrap, unwrap_or
- [x] Compound assignment: +=, -=, *=, /=, %=
- [x] Enum: enum keyword
- [x] Import/export system with PIPE_PATH

### v0.4 — Bytecode VM

- [x] Compiler: AST → Bytecode transformation
- [x] 43 opcodes covering all language features
- [x] Stack-based VM with 2048-slot stack and 1024 frame depth
- [x] Symbol table with 4 scope types (Global, Local, Free, Builtin)
- [x] Closure compilation with free variable capture
- [x] Short-circuit compilation for `&&` and `||`
- [x] Loop patching for break/continue
- [x] `.pipec` bytecode cache format (SHA-256 validated)
- [x] `-vm` CLI flag

### v0.5 — Tooling & Polish

- [x] Pipe formatter: `pipe fmt`
- [x] Self-extracting binary builder: `pipe build`
- [x] VSCode extension with syntax highlighting and auto-indentation
- [x] 19 example programs in `examples/`
- [x] Comprehensive documentation (18 chapters + appendix)
- [x] REPL with history, `:vm` toggle, and multi-line input
- [x] Dual execution (tree-walker default, VM with `-vm` flag)
- [x] Tail-call optimization in tree-walker
- [x] Break/continue in loops
- [x] Dot expression for map field access

---

## Phase 1: Quick Wins (v0.5.1) ✅

> **Status: Completed**

- [x] `defer` keyword for deferred execution (LIFO order)
- [x] `_` placeholder in pipeline call arguments
- [x] Documentation fixes and improvements
- [x] DEDENT nesting fix for nested blocks (if/while inside other blocks)

---

## Phase 2: Structure (v0.6)

### Improved Module System
- [x] Directory-based imports (`import "mylib/"` loads `mylib/init.pipe`)
- [x] Relative imports (`import "./utils.pipe"`)
- [x] Circular import detection and error reporting (E009, tree-walker + VM)
- [x] `PIPE_PATH` environment variable documentation and improvements (native separator via `filepath.SplitList`)
- [x] Semantic-version dependency constraints (`^X.Y.Z` caret resolution) for `pipe -install`
- [x] `pipe -publish` — publish modules to the registry via gh-CLI pull request

### Formatter Enhancements (`pipe fmt`)
- [x] `--check` flag (exit non-zero if formatting needed)
- [x] `--write` flag (overwrite files in-place)
- [x] Directory processing (`pipe fmt ./src/` for all `.pipe` files)
- [x] Whitespace-only mode for unparseable files (fallback normalization)
- [ ] Configuration options (indent size, quote style)

### REPL Improvements
- [x] Persistent history across sessions (save to `~/.pipe_history`)
- [ ] Tab completion for builtin function names
- [ ] Multi-line editing improvements
- [ ] Colored output for errors and values
- [ ] `:load` command to load source files into REPL session

### Better Error Messages
- [x] Source code snippets in error output (show the offending line)
- [x] Error code system (e.g., `E001: undefined variable`)
- [ ] Suggestions for common mistakes (e.g., `=` instead of `:` for assignment)
- [x] Warning for unused variables

### Enhanced Pattern Matching
- [ ] Binding patterns (`| x: Some(x) -> ...`)
- [ ] Guard clauses (`| x if x > 0 -> ...`)
- [ ] List destructuring patterns (`| [a, b, ...rest] -> ...`)
- [ ] Map destructuring patterns (`| {name: n, age: a} -> ...`)

---

## Phase 3: Maturity (v0.7+)

### Concurrency
- [x] `>>` parallel pipeline with Future auto-resolution (v0.6)
- [x] `>>` true parallelism for user-defined closures in the bytecode VM (was synchronous fallback)
- [x] `spawn` builtin — launch a function and get back a Future
- [x] `await` builtin — block on a Future with an optional timeout
- [x] Channels (`chan`, `send`, `recv`, `try_recv`, `try_send`, `close`)
- [x] Mutex (`mutex`, `lock`, `unlock`, `try_lock`)
- [x] Counting semaphore (`semaphore`, `acquire`, `release`, `try_acquire`)
- [ ] Lightweight coroutines/green threads
- [ ] `select` statement (Go-style multi-channel select)
- [ ] `spawn` keyword for launching concurrent tasks
- [ ] `await` for waiting on task completion

### Type System
- [ ] Optional type annotations (`x: int = 42`)
- [ ] Basic type checking at compile time
- [ ] Union types (`int | string`)
- [ ] Type inference for local variables
- [ ] `fn` return type annotations

### Package Registry
- [x] Central module registry (`pipe -search`, `pipe -get`) (v0.6)
- [x] Semantic versioning (`@1.0.0`) (v0.6)
- [x] `pipe -install <package>` for dependency management (transitive deps, `^X.Y.Z` constraints, `pipe.lock`)
- [x] `pipe -publish` for package authors (gh-CLI pull request into the registry)
- [x] Package manifest format (`pipe.json`) — created via `pipe -init`
- [x] Lockfile consumption for reproducible installs (`pipe.lock` honored when present, SHA-256 checksums verified)

### Bytecode Optimizations
- [x] Constant folding at compile time
- [ ] Dead code elimination
- [ ] Inlining of small functions
- [ ] Peephole optimization pass
- [ ] Bytecode serialization to/from `.pipec`

### Web Playground
- [x] Browser-based Pipe editor (compiled to WASM)
- [x] Interactive examples and tutorials
- [x] Code sharing via URLs
- [x] Syntax-highlighted editor with live execution

### VSCode 2.0 Extension
- [ ] Debugger support with breakpoints
- [ ] Variable inspection and watch expressions
- [ ] REPL terminal integration
- [ ] Code snippets for common patterns
- [ ] Project-level configuration (`.vscode/pipe.json`)

### Language Server Protocol (LSP)
- [x] Go-to-definition for functions and variables
- [x] Find all references
- [x] Hover information with type and documentation
- [x] Inline diagnostics (parse errors, warnings)
- [x] Code completion suggestions
- [x] Rename symbol refactoring
- [ ] Workspace symbol search

### Test Framework
- [x] `test` keyword or `test` builtin function
- [x] Assertion helpers: `assert`, `assert_eq`, `assert_not_eq`, `assert_lt`, `assert_gt`, `assert_error`
- [x] Test runner CLI (`pipe test`)
- [x] Test file discovery (`*_test.pipe`)
- [x] Setup/teardown hooks

### Documentation Generator
- [x] Extract docstrings from source code (`--!` docstrings)
- [x] Generate API documentation in Markdown (`pipe -doc`)
- [ ] Cross-reference links between documented symbols
- [x] `pipe doc` command (`pipe -doc` and `pipe -doc --builtins`)

### Additional Features
- [ ] Set data structure (unique, unordered collection)
- [ ] File system watchers (`watch_dir`, `watch_file`)
- [ ] Process management (start, kill, wait for subprocesses)
- [ ] Regular expression capture groups in `regex_match`
- [ ] Binary data support (byte arrays)
- [ ] Cryptographic functions (hash, encrypt, decrypt)
- [ ] Environment variable file loading (`.env` support)
- [ ] Signal handling (SIGINT, SIGTERM)
- [ ] Interactive prompt library (readline-style)
- [ ] Internationalization (i18n) support
- [ ] Plugin system for extending the runtime with Go or WASM modules

---

## Feature Status Matrix

| Feature | Status | Version | Notes |
|---------|--------|---------|-------|
| Lexer (52 tokens) | ✅ Done | v0.1 | Includes INDENT/DEDENT |
| Parser (Pratt + RD) | ✅ Done | v0.1 | 13 precedence levels |
| AST (27 nodes) | ✅ Done | v0.1 | 12 stmts, 15 exprs |
| Tree-Walker | ✅ Done | v0.1 | Default execution mode |
| Variables & Functions | ✅ Done | v0.1 | Closures supported |
| Control Flow | ✅ Done | v0.1 | if/while/for/match/try |
| Lists & Maps | ✅ Done | v0.1 | Dot access, slicing |
| Pipelines | ✅ Done | v0.1 | Horizontal + vertical |
| Parallel Pipeline (`>>`) | ✅ Done | v0.6 | Future auto-resolution |
| Standard Library | ✅ Done | v0.2 | 86 builtins |
| HTTP Client | ✅ Done | v0.3 | GET, POST, JSON |
| TCP Networking | ✅ Done | v0.3 | Server + client |
| Regex | ✅ Done | v0.3 | Match + replace |
| Date/Time | ✅ Done | v0.3 | now, format_time |
| Compound Assignment | ✅ Done | v0.3 | +=, -=, *=, /=, %= |
| Enum | ✅ Done | v0.3 | Named values |
| Import/Export | ✅ Done | v0.3 | Module system |
| Module Registry | ✅ Done | v0.6 | pipe -search, pipe -get |
| Module Versions | ✅ Done | v0.6 | import \"mod@1.0.0\" |
| export let/enum | ✅ Done | v0.6 | Variable + enum exports |
| Result Type | ✅ Done | v0.3 | Ok/Err pattern |
| Bytecode VM | ✅ Done | v0.4 | 43 opcodes |
| `.pipec` Cache | ✅ Done | v0.4 | SHA-256 validated |
| Formatter | ✅ Done | v0.5 | `pipe fmt` |
| Formatter Enhance | ✅ Done | v0.6 | --check, dirs |
| Build System | ✅ Done | v0.5 | Self-extracting bins |
| VSCode Extension | ✅ Done | v0.5 | Syntax + auto-indent |
| Documentation | ✅ Done | v0.5 | 18 chapters |
| REPL | ✅ Done | v0.5 | History, :vm toggle |
| REPL History Persist | ✅ Done | v0.6 | Save/load ~/.pipe_history |
| TCO | ✅ Done | v0.5 | Tree-walker only |
| Defer | ✅ Done | v0.5.1 | LIFO cleanup |
| `_` Placeholder | ✅ Done | v0.5.1 | Pipeline args |
| `try_ai` Self-Healing | ✅ Done | v0.6 | AI auto-fix errors |
| Improved Module Sys | ✅ Done | v0.9.4.0 | Dirs, relative, circular, `PIPE_PATH`, SemVer, install, publish, lockfile |
| Better Errors | 🟡 Partial | v0.6/v0.7 | Error codes, snippets, unused-var warnings done; suggestions open |
| Pattern Matching+ | 🟡 Partial | v0.7 | Multi-pattern done; guards, destructuring open |
| Concurrency | 🔮 Future | v0.7+ | spawn/await/channels/mutex/semaphore done; coroutines, select open |
| Type Annotations | 🔮 Future | v0.7+ | Optional typing |
| Bytecode Opts | 🟡 Partial | v0.7+ | Constant folding done; inline, peephole open |
| Sets | 🔮 Future | v0.7+ | Unique collections |
| Inline Lambdas | ✅ Done | v0.8 | `fn x: expr` in TW + VM |
| Web Playground | ✅ Done | v0.7 | WASM-based + code sharing |
| VSCode 2.0 | 🔮 Future | v0.7+ | Debugger, snippets |
| LSP | ✅ Done | v0.7 | Go-to-def, references, hover, completion, diagnostics, rename |
| Test Framework | ✅ Done | v0.9.4.0+ | test blocks + asserts + CLI, assert_near/assert_contains, setup/teardown hooks, runs in TV and VM |
| Doc Generator | ✅ Done | v0.9.4.0 | `pipe -doc` + `--builtins`; cross-references open |
| Plugins | 🔮 Future | v0.8+ | Go/WASM extensions |
| SQLite Module | ✅ Done | v0.8 | Pure-Pipe SQL engine, available via `pipe -get sqlite` from pipe-modules. Replaces modernc.org/sqlite. Binary ~8 MB, zero deps. SQL: CREATE/INSERT/UPDATE/DELETE/SELECT with WHERE, GROUP BY, ORDER BY, JOINs, transactions, paged binary persistence. Pipeline API (q, exec, row_get, row_eq, row_ne). Benchmarks vs Python/Lua in the [SQLite Module](26-sqlite-module.md) chapter. VM mode now runs the full module (ORDER BY/JOIN fixed). |
