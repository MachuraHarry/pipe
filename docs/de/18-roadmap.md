# 18. Roadmap

## Aktuelle Version: v0.6.0

---

## Phase 1: Quick Wins (v0.5.1)

| # | Feature | Beschreibung | Status |
|---|---|---|---|
| 1 | `defer`-Statement | Aktion am Block-Ende ausführen (Go-Style LIFO) | ✅ Erledigt |
| 2 | `_` Platzhalter in Pipeline | Pipeline-Arg an beliebiger Position platzieren | ✅ Erledigt |
| 3 | Doku-Korrekturen | `>`-Ambiguität, Portabilitäts-Aussagen korrigiert | ✅ Erledigt |
| 4 | DEDENT-Nesting-Fix | break/continue in if-in-while | ✅ Erledigt |

---

## Phase 2: Struktur (v0.6) — Geplant

| # | Feature | Beschreibung |
|---|---|---|
| 5 | **Modul-System 2.0** | Verbessertes `import`, zirkuläre Abhängigkeiten, Modul-Pfade |
| 6 | **Verbessertes `pipe fmt`** | Konsistentere Formatierung, Konfigurations-Optionen |
| 7 | **REPL-History-Upgrade** | Pfeiltasten-Navigation, inkrementelle Suche |
| 8 | **Verbesserte Fehlermeldungen** | `Datei:Zeile:Spalte` in allen Laufzeitfehlern, farbige Ausgabe |
| 9 | **Erweiterte Pattern-Matching** | Bereichs-Muster (`1..10`), Guard-Klauseln |

---

## Phase 3: Reife (v0.7+) — Vision

| # | Feature | Beschreibung |
|---|---|---|
| 10 | **Concurrency** | `>>` Parallel-Pipeline existiert (v0.6); `go fn()` für volles VM-Concurrency geplant |
| 11 | **Optionale Typannotationen** | `fn add(a: num, b: num) -> a + b` |
| 12 | **Package-Registry 2.0** | `pipe publish`, `pipe install`; Basis (`-search`, `-get`, `@version`) existiert seit v0.6 |
| 13 | **Bytecode-Optimierungen** | Peephole-Optimizer, Inline-Caching, Constant Folding |
| 14 | **Web-Playground** | "Try Pipe in your Browser" via WASM |
| 15 | **VSCode-Plugin 2.0** | Auto-Vervollständigung, Go-to-Definition, Debugger-Integration |
| 16 | **LSP-Server** | Language Server Protocol für Editor-Unterstützung |
| 17 | **Standard-Test-Framework** | `assert`, `test`, `suite` Keywords |
| 18 | **Dokumentations-Generator** | `pipe doc` — Dokumentation aus Quelltext-Kommentaren |

---

## Abgeschlossen (v0.1 – v0.5)

- ✅ Lexer + Parser + AST (15 + 20 + 29 Tests)
- ✅ Tree-Walk Interpreter (42 Tests)
- ✅ Bytecode Compiler + Stack-VM (47 Opcodes, 29 + 18 + 28 Tests)
- ✅ 80+ Builtins (IO, FS, HTTP, JSON, TCP, Regex, DateTime, ...)
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
| Erweitertes Match | | | ✅ | |
| Concurrency (go) | 🟡 (nur Tree) | | | ✅ |
| Typannotationen | | | | ✅ |
| Package Registry | | | ✅ | |
| LSP Server | | | | ✅ |
| Web Playground | | | | ✅ |
