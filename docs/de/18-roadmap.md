# 18. Roadmap

## Aktuelle Version: v0.9.4.0

Pipe ist aktuell in Version **v0.9.4.0**, dem **Modulsystem-Release**. Es vervollständigt das Modul-System auf dem Weg zu v1.0: **Directory-Imports** (`import "mylib/"` lädt `mylib/init.pipe`), dedizierte **relative Imports** (`./`, `../` — nie über den Registry-Fallback), **Zyklus-Erkennung** (`E009: circular import` in Tree-Walker und Bytecode-VM, inkl. Alias-Importe), `PIPE_PATH` über `filepath.SplitList` (plattformnatives Trennzeichen), **SemVer-Auflösung** für `^X.Y.Z`-Constraints bei `pipe -install` sowie **`pipe -publish`**: veröffentlicht Module per gh-CLI-Pull-Request in der Registry (Validierung, Versions-Duplikat-Check, Modul-Dir + registry.json-Eintrag, Branch `publish/<name>-<version>`). Der Sprachumfang entspricht weiterhin dem Sandbox-Härtungs-Release v0.9.3.5: zentraler Egress-Gate für alle AI-Provider-Ausgänge, Rekursions-Tiefen-Guard (E008), zero-dependency MCP-Server, X-API-v2- und Discord-Modul sowie die CI-Benachrichtigungen mit Discord + Telegram Dual-Versand inklusive KI-Code-Reviews.

---

## Phase 1: Quick Wins (v0.5.1)

| # | Feature | Beschreibung | Status |
|---|---|---|---|
| 1 | `defer`-Statement | Aktion am Block-Ende ausführen (Go-Style LIFO) | ✅ Erledigt |
| 2 | `_` Platzhalter in Pipeline | Pipeline-Arg an beliebiger Position platzieren | ✅ Erledigt |
| 3 | Doku-Korrekturen | `>`-Ambiguität, Portabilitäts-Aussagen korrigiert | ✅ Erledigt |
| 4 | DEDENT-Nesting-Fix | break/continue in if-in-while | ✅ Erledigt |

---

## Phase 2: Struktur (v0.6) — Erledigt

| # | Feature | Beschreibung | Status |
|---|---|---|---|
| 5 | **Modul-System 2.0** | Verbessertes `import`, zirkuläre Abhängigkeiten, Modul-Pfade, `pipe -install`/`-publish`, `pipe.lock` | ✅ Erledigt |
| 6 | **Verbessertes `pipe fmt`** | Konsistentere Formatierung, Konfigurations-Optionen | 🟡 Teilweise (`--check`, Verzeichnisse, Whitespace-Fallback erledigt; Optionen offen) |
| 7 | **REPL-History-Upgrade** | Pfeiltasten-Navigation, inkrementelle Suche | 🟡 Persistenz erledigt; Tab-Completion/Farben/`:load` offen |
| 8 | **Verbesserte Fehlermeldungen** | `Datei:Zeile:Spalte` in allen Laufzeitfehlern, farbige Ausgabe | 🟡 Fehlercodes + Source-Snippets + Unused-Var-Warnings erledigt; Vorschläge offen |
| 9 | **Erweiterte Pattern-Matching** | Bereichs-Muster (`1..10`), Guard-Klauseln | 🟡 Multi-Pattern + Guards erledigt; Binding/Destructuring offen |

---

## v0.7 — Ecosystem, Selbstheilung & Tooling (Abgeschlossen)

| # | Feature | Beschreibung | Status |
|---|---|---|---|
| 1 | **Parallel-Pipeline `>>`** | Future-Autoauflösung | ✅ Erledigt |
| 2 | **`try_ai` Selbstheilung** | KI repariert Laufzeitfehler, optionales `catch` | ✅ Erledigt |
| 3 | **Modul-Versionierung** | `import "mod@1.0.0"`, `pipe -get`, `pipe -search` | ✅ Erledigt |
| 4 | **C-style `for`** | `for i: 0; i < 5; i: i + 1` in Tree-Walker + VM | ✅ Erledigt |
| 5 | **`not`-Keyword** | Synonym für `!` | ✅ Erledigt |
| 6 | **Multi-Pattern-Match** | `| 1 | 2 | 3 -> "small"` | ✅ Erledigt |
| 7 | **Test-Framework** | `test`-Blöcke + `assert`/`assert_eq`/`assert_error`/`assert_lt`/`assert_gt`/`assert_not_eq` | ✅ Erledigt |
| 8 | **Sandbox-Profile** | Deklarative Laufzeit-Sicherheit für KI-Agenten | ✅ Erledigt |
| 9 | **Fehlercodes** | E001–E004 mit Parser-Hinweisen | ✅ Erledigt |
| 10 | **Formatter `--check` + Verzeichnisse** | REPL-History-Persistenz | ✅ Erledigt |
| 11 | **GitHub Action** | `pipe-action` für CI/CD | ✅ Erledigt |
| 12 | **HTTP-API-Server** | `cmd/api-server` mit Fly.io-Deployment | ✅ Erledigt |
| 13 | **WASM-Playground v2 + Blog** | Code-Sharing, Syntax-Highlighting, Bewertungen | ✅ Erledigt |
| 14 | **Inline-Lambda-Syntax** | `fn x: ausdruck` — einzeilige anonyme Funktionen in TW + VM | ✅ Erledigt |

---

## Phase 3: Reife (v0.7+) — Vision

| # | Feature | Beschreibung |
|---|---|---|
| 10 | **Concurrency** | `>>` Parallel-Pipeline, `go`/`spawn`/`await`, Channels, Mutex und Semaphor (TW + VM, v0.9.x); `select`/Coroutines noch offen |
| 11 | **Optionale Typannotationen** | `fn add(a: num, b: num) -> a + b` |
| 12 | **Package-Registry 2.0** | `pipe -install`, `pipe -publish`, `pipe.json`-Manifest, `pipe.lock`-Reproduzierbarkeit | ✅ Erledigt |
| 13 | **Bytecode-Optimierungen** | Constant Folding (Compiler-Zeit) erledigt; Inline-Caching, Peephole offen |
| 14 | **Web-Playground** | "Try Pipe in your Browser" via WASM | ✅ Erledigt |
| 15 | **VSCode-Plugin 2.0** | Auto-Vervollständigung, Go-to-Definition, Debugger-Integration |
| 16 | **LSP-Server** | Language Server Protocol für Editor-Unterstützung | ✅ Erledigt |
| 17 | **Standard-Test-Framework** | `assert`, `test`, `assert_eq`, `assert_error`, `assert_lt`, `assert_gt`, `assert_near`, `assert_contains` — Setup/Teardown-Hooks in TV + VM | ✅ Erledigt |
| 18 | **Dokumentations-Generator** | `pipe -doc` — Markdown aus `--!`-Docstrings, `--builtins` | ✅ Erledigt (Cross-References offen) |

---

## Abgeschlossen (v0.1 – v0.5)

- ✅ Lexer + Parser + AST (17 + 25 + 32 Tests)
- ✅ Tree-Walk Interpreter (42 Tests)
- ✅ Bytecode Compiler + Stack-VM (43 Opcodes, 29 + 18 + 28 Tests)
- ✅ 168 Builtins (IO, FS, HTTP, JSON, TCP, Regex, DateTime, ...)
- ✅ while, break, continue, return
- ✅ for-in Schleifen
- ✅ try/catch mit Stack-Traces
- ✅ `!`, `&&`, `||`, `**`, Compound Assignment
- ✅ CLI-Args, input(), exec()
- ✅ Listen-Slicing, Anonyme Funktionen, Import/Export
- ✅ `enum`, `defer`, Result-Typ (Ok/Err)
- ✅ REPL mit Multi-Line und History (100 Einträge)
- ✅ Code-Formatter (`-fmt`)
- ✅ Test-Runner (`-test`)
- ✅ Benchmark-Tool (`-bench`)
- ✅ Self-Extracting Binary (`-build`)
- ✅ Bytecode-Cache (`.pipec`)
- ✅ Tail Call Optimization
- ✅ Dot-Notation für Map-Zugriff
- ✅ 23 Beispiel-Programme
- ✅ VSCode Extension (Syntax-Highlighting)
- ✅ Umfassende Dokumentation (18 Kapitel, DE + EN)

---

## Feature-Status Matrix

| Feature | v0.1-v0.5 | v0.5.1 | v0.6 | v0.7+ |
|---------|:---------:|:------:|:----:|:-----:|
| Lexer + Parser | ✅ | | | |
| Tree-Walker | ✅ | | | |
| Bytecode-VM | ✅ | | | |
| 81 Builtins | ✅ | | | |
| while/for-in/break/continue | ✅ | | | |
| try/catch/return | ✅ | | | |
| defer | ✅ | | | |
| Pipeline | ✅ | | | |
| Parallel-Pipeline (`>>`) | | | ✅ | |
| Import/Export | ✅ | | | |
| export var/enum | | | ✅ | |
| Modul-Registry | | | ✅ | |
| Modul-Versionen (`@1.0.0`) | | | ✅ | |
| Enum | ✅ | | | |
| Result-Typ | ✅ | | | |
| Slicing | ✅ | | | |
| Compound Assignment | ✅ | | | |
| Anonyme Fns/Closures | ✅ | | | |
| TCO | ✅ | | | |
| Formatter | ✅ | | | |
| Test-Runner | ✅ | | | |
| Benchmarks | ✅ | | | |
| Self-Extracting Binary | ✅ | | | |
| REPL + History | ✅ | | | |
| REPL Persistenz | | | ✅ | |
| `try_ai` Selbstheilung | | | ✅ | |
| Bytecode-Cache | ✅ | | | |
| VSCode Extension | ✅ | | | |
| Erweitertes Match (Multi-Pattern) | | | ✅ | |
| C-style `for` + `not` | | | ✅ | |
| Sandbox-Profile | | | ✅ | |
| Fehlercodes E001–E004 | | | ✅ | |
| Test-Framework (`test` + asserts) | | | ✅ | |
| GitHub Action | | | ✅ | |
| HTTP-API-Server | | | ✅ | |
| Web Playground v2 + Blog | | | ✅ | |
| Inline-Lambda (`fn x: expr`) | | | | ✅ |
| Concurrency (go) | 🟡 (nur Tree) | | | ✅ |
| Typannotationen | | | | ✅ |
| Package Registry | | | ✅ | |
| LSP Server | | | ✅ | |
| Web Playground (klassisch) | | | | ✅ |
| SQLite-Modul (reines Pipe) | | | | ✅ |
