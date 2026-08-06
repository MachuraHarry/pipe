# SQLite — Pure-Pipe relationales Datenbankmodul

**Status:** Modul implementiert (`examples/sqlite.pipe`, ~2400 Zeilen), API-kompatibel mit den früheren `modernc.org/sqlite`-Builtins. Binary ist wieder dependency-free (~7 MB). TV-Modus (`pipe examples/sqlite_demo.pipe`) führt CREATE TABLE, INSERT und SELECT * korrekt aus. VM-Modus hat einen Compiler-Bug (OpCall in Modul-Closures). Siehe `MEMORY.md` für Details.

## Architektur

Das Modul ist eine einzelne `.pipe`-Datei (`examples/sqlite.pipe`), die über `import "sqlite.pipe"` (oder `import "sqlite"` nach Installation in `~/.pipe/modules/`) geladen wird. Es exportiert vier Funktionen:

- `db_open(path)` — Öffnet eine Datenbank (Datei oder `":memory:"`)
- `db_close(handle)` — Schließt die Datenbank und persistiert Änderungen
- `db_exec(handle, sql)` — Führt DDL/DML aus, gibt Anzahl betroffener Zeilen zurück
- `db_query(handle, sql)` — Führt SELECT aus, gibt Liste von Maps zurück

## Unterstützte SQL-Features

- **DDL:** `CREATE TABLE`, `DROP TABLE`, `CREATE INDEX`
- **DML:** `INSERT INTO` (auch multi-value), `UPDATE`, `DELETE`
- **Queries:** `SELECT` mit `WHERE` (=, !=, <, >, <=, >=, AND, OR, NOT, LIKE, IN, BETWEEN, IS NULL), `GROUP BY` + Aggregatfunktionen (COUNT, SUM, AVG, MIN, MAX), `ORDER BY` (ASC/DESC), `LIMIT`/`OFFSET`, `JOIN` (INNER, LEFT, RIGHT), `DISTINCT`
- **Transaktionen:** `BEGIN`, `COMMIT`, `ROLLBACK` mit Journal
- **Persistence:** Binäres Paging-Format mit CRC32-Checksummen, atomische Commits via `file_move`

## Implementierungsdetails

- **Lexer:** Zeichenweise Tokenisierung (Keywords, Identifier, Zahlen, Strings, Operatoren)
- **Parser:** Recursive-Descent, Pratt-Parser für Ausdrücke
- **Evaluator:** AST-Walker für Ausdrücke, Tabellen-Scan für Queries
- **Encoding:** Tag-basiertes Binärformat (nil/bool/string/number/bytes/list/map)
- **Storage:** Paged File I/O mit 4KB-Seiten, CRC32-Footer pro Seite
- **Alle Primitive:** Nutzt die in Phase 1-2 eingeführten Builtins (BYTES, bitwise, file_*, slice, sort, crc32, substring, index_of)

## Bekannte Einschränkungen (Pipe-Runtime-Bugs)

1. **VM-Modus (`pipe -vm`):** Der Compiler weist lokalen Variablen in Closures überlappende Slots zu. Bestimmte Variablennamen (z.B. `result`, `text`, `tok`) korrumpieren sich gegenseitig über Funktionsgrenzen hinweg. `for`-Schleifen korrumpieren äußere lokale Variablen. Verschachtelte Funktionsaufrufe (`at(get(...), get(...))`) verlieren Zwischenergebnisse. Map-Literale mit Bare-Identifier-Keys (`{key: val}`) werden falsch kompiliert (Workaround: `set`-basierte Konstruktion). Aktuell parst der Tokenizer korrekt, aber `parse_sql` liefert 0 Statements weil `parse_cur`-Rückgaben korrumpiert werden.

2. **TV-Modus (`pipe`):** Go-Panic `index out of range [4]` in `extendFunctionEnv` bei tief verschachtelten if/else-Ketten im Evaluator. Einfache Operationen funktionieren, aber sequentielle Queries lösen die Panik aus.

Diese Bugs liegen im Pipe-Compiler/VM (`pkg/compiler/compiler.go`, `pkg/vm/vm.go`, `pkg/eval/eval.go`), nicht im sqlite-Modul selbst.

```pipe
import "sqlite.pipe"

h: db_open ":memory:"
db_exec h "CREATE TABLE tasks (id INTEGER PRIMARY KEY, title TEXT)"
db_exec h "INSERT INTO tasks VALUES (1, 'Fix login bug')"
rows: db_query h "SELECT * FROM tasks"
db_close h
```

## Phasen-Status

| Phase | Beschreibung | Status |
|-------|-------------|--------|
| 0 | modernc.org/sqlite entfernen | ✅ Erledigt |
| 1 | Kern-Primitive (BYTES, bitwise, file I/O, sort, crc32) | ✅ Erledigt |
| 2 | Registrierung & Doku | ✅ Erledigt |
| 3 | SQLite-Modul (reines Pipe) | ✅ Implementiert (eingeschränkt durch Pipe-Runtime-Bugs) |
| 4 | Aufräumen (Stubs entfernen) | ✅ Erledigt |
| 5 | Tests & CI | ⏳ Offen (abhängig von Runtime-Bugfixes) |
