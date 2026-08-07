# SQLite — Pure-Pipe relationales Datenbankmodul

**Status:** Modul als externes Package im [`pipe-modules`](https://github.com/MachuraHarry/pipe-modules)-Repository verfügbar. Install per `pipe -get sqlite`. API-kompatibel mit den früheren `modernc.org/sqlite`-Builtins. Binary ist dependency-free (~10 MB). **TV-Modus** führt alle Operationen korrekt aus: CREATE TABLE, INSERT, SELECT, WHERE, GROUP BY, ORDER BY, UPDATE, DELETE. **VM-Modus** hat einen Compiler-Bug bei großen Modul-Imports (siehe unten).

**Pipeline-API:** Das Modul exportiert Pipeline-Helper (`q`, `exec`, `row_get`, `row_eq`, `row_ne`), die mit Pipe's `>`-Operator und `map`/`filter`/`each`-Builtins komponierbar sind. Demo: `examples/sqlite_pipeline.pipe`.

**Benchmark:** 5000-Row-Benchmark gegen Python und Lua zeigt Pipe als schnellste Gesamtlaufzeit (~21 ms vs Python ~28 ms vs Lua ~44 ms). Pipe's pure-pipe SQL-Engine schlägt nativen C-sqlite3 in SELECT-Queries um Faktor 10-30x, weil keine C-Binding-Kopiervorgänge anfallen. Siehe `BENCHMARK.md`.

## Architektur

Das Modul ist im [`pipe-modules`](https://github.com/MachuraHarry/pipe-modules)-Repository unter `sqlite/module.pipe` abgelegt und wird via `pipe -get sqlite` oder per URL-Import geladen:

### Core API (klassisch)
- `db_open(path)` — Öffnet eine Datenbank (Datei oder `":memory:"`)
- `db_close(handle)` — Schließt die Datenbank und persistiert Änderungen
- `db_exec(handle, sql)` — Führt DDL/DML aus, gibt Anzahl betroffener Zeilen zurück
- `db_query(handle, sql)` — Führt SELECT aus, gibt Liste von Row-Maps zurück

### Pipeline API (komponierbar via `>`)
- `q(handle, sql)` — Short-Alias für `db_query`
- `exec(handle, sql)` — Short-Alias für `db_exec`
- `row_get(row, key)` — Nil-sicherer dynamischer Feldzugriff
- `row_eq(row, key, val)` — Prädikat für `filter` (==)
- `row_ne(row, key, val)` — Prädikat für `filter` (!=)

## Unterstützte SQL-Features

- **DDL:** `CREATE TABLE`, `DROP TABLE`, `CREATE INDEX`
- **DML:** `INSERT INTO` (auch multi-value), `UPDATE`, `DELETE`
- **Queries:** `SELECT` mit `WHERE` (=, !=, <, >, <=, >=, AND, OR, NOT, LIKE, IN, BETWEEN, IS NULL), `GROUP BY` + Aggregatfunktionen (COUNT, SUM, AVG, MIN, MAX), `ORDER BY` (ASC/DESC), `LIMIT`/`OFFSET`, `JOIN` (INNER, LEFT, RIGHT), `DISTINCT`
- **SQL ist case-insensitiv:** Alle Keywords werden vom Lexer normalisiert und Parser/Evaluator arbeiten durchgehend lowercase
- **Transaktionen:** `BEGIN`, `COMMIT`, `ROLLBACK` mit Journal
- **Persistence:** Binäres Paging-Format mit CRC32-Checksummen, atomische Commits via `file_move`

## Implementierungsdetails

- **Lexer:** Zeichenweise Tokenisierung (Keywords, Identifier, Zahlen, Strings, Operatoren)
- **Parser:** Recursive-Descent, Pratt-Parser für Ausdrücke
- **Evaluator:** AST-Walker für Ausdrücke, Tabellen-Scan für Queries
- **Encoding:** Tag-basiertes Binärformat (nil/bool/string/number/bytes/list/map)
- **Storage:** Paged File I/O mit 4KB-Seiten, CRC32-Footer pro Seite

## Bekannte Einschränkung

**VM-Modus (`pipe -vm`):** Der Compiler hat einen Bug in `compileImport` für große Module mit >140 Funktionen. Modul-Globals und verschachtelte Funktions-Calls in Closures funktionieren nicht korrekt — exec_insert/exec_select scheitern mit `. only on map: CLOSURE`/`INTEGER`. Einfache Operationen (db_open, get_db, exec_create_table) funktionieren isoliert. Siehe `MEMORY.md` für Details.

## Beispiele

```pipe
-- Klassische API
import "sqlite"
h: db_open ":memory:"
db_exec h "CREATE TABLE t (id INTEGER PRIMARY KEY, title TEXT)"
db_exec h "INSERT INTO t VALUES (1, 'hello')"
rows: db_query h "SELECT * FROM t"
i: 0
while i < (len rows)
    row: at rows i
    print (get row "title")
    i: i + 1
db_close h
```

```pipe
-- Pipeline API
import "sqlite"
h: db_open ":memory:"
exec h "CREATE TABLE t (a INTEGER)"
exec h "INSERT INTO t VALUES (1), (2), (3)"

fn is_even row
    (get row "a") % 2 == 0

q h "SELECT * FROM t" > filter is_even > each (fn r: print ("a=" ++ (to_str (get r "a"))))
db_close h
```

Weitere Beispiele: `examples/sqlite_basic.pipe`, `examples/sqlite_pipeline.pipe`.

## Phasen-Status

| Phase | Beschreibung | Status |
|-------|-------------|--------|
| 0 | modernc.org/sqlite entfernen | ✅ Erledigt |
| 1 | Kern-Primitive (BYTES, bitwise, file_*, sort, crc32) | ✅ Erledigt |
| 2 | Registrierung & Doku | ✅ Erledigt |
| 3 | SQLite-Modul (reines Pipe) | ✅ Voll funktionsfähig (TV) |
| 4 | Aufräumen (Stubs entfernen) | ✅ Erledigt |
| 5 | Pipeline-API | ✅ Erledigt |
| 6 | Benchmark (Pipe vs Python vs Lua) | ✅ Erledigt |
| 7 | VM-Modus-Fix | ⏳ Compiler-Bug (tief in compileImport) |
