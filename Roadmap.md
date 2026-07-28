# Pipe — Roadmap

## Phase 1: Quick Wins (v0.5.1)

| # | Feature | Beschreibung | Aufwand |
|---|---|---|---|
| 1 | **`defer`-Statement** | `defer file_close f` — Aktion am Block-Ende ausführen (wie Go) | 1h |
| 2 | **`_` Platzhalter in Pipeline** | `42 > addiere 10 _` → `addiere(10, 42)`. Pipeline-Arg an anderer Position | 30min |
| 3 | **Doku-Fixes** | `>`-Ambiguität klarstellen, „Embedded" → „Portabel" korrigieren | 10min |
| 4 | **DEDENT-Nesting-Fix** | break/continue in if-in-while reparieren | 2h |

**Ziel:** Die gröbsten Kanten glätten. Pipeline flexibler, Fehler in Blöcken beheben.

---

## Phase 2: Struktur (v0.6)

| # | Feature | Beschreibung |
|---|---|---|
| 5 | **Modul-System** | `export fn name` — explizite Exports. `import "lib" as name` — Namespace |
| 6 | **Result-Typ** | `Ok(value)` / `Err(message)` für Pipeline-kompatible Fehler |
| 7 | **`pipe fmt`** | Code-Formatter (Einrückung, Spacing) |
| 8 | **REPL-History** | Pfeiltasten, `Ctrl+R` Suche |
| 9 | **Bessere Fehlermeldungen** | `Datei:Zeile:Spalte` in allen Laufzeitfehlern |
| 10 | **`enum` / benutzerdefinierte Typen** | `enum Color: Rot, Grün, Blau` |

**Ziel:** Pipe wird teamtauglich. Module, Formatierung, bessere Fehler.

---

## Phase 3: Reife (v0.7+)

| # | Feature | Beschreibung |
|---|---|---|
| 11 | **Concurrency: `go fn()`** | Goroutine-artige parallele Ausführung |
| 12 | **Optionale Typannotationen** | `fn add(a: num, b: num) -> num: a + b` |
| 13 | **Package-Registry** | `import "github.com/user/lib"` mit Caching |
| 14 | **Bytecode-Optimierungen** | Peephole-Optimizer, Inline-Caching |
| 15 | **Web-Playground** | „Try Pipe in your Browser" via WASM |
| 16 | **VSCode-Plugin** | Syntax-Highlighting, Autocomplete |

**Ziel:** Pipe als ernsthafte Alternative zu Python/Lua für DevOps-Tools.

---

## Abgeschlossen (v0.1 – v0.5)

- ✅ Lexer + Parser + AST
- ✅ Tree-Walker + Bytecode-VM
- ✅ 80+ Builtins (HTTP, JSON, TCP, Regex, FS, DateTime, ...)
- ✅ while, for-in, break, continue, return
- ✅ try/catch mit Stack-Traces
- ✅ `!`, `&&`, `||`, `**`, `+=`, compound-assign
- ✅ CLI-Args, input(), exec()
- ✅ Listen-Slicing, Anonyme Fns, Import
- ✅ REPL mit Multi-Line
- ✅ 60 Tests, 21 Beispiele
- ✅ Umfassende Doku (Doku.md, GUIDE.md, PIPE_FUER_DUMMIES.md)
