# SQLite Module - State Memory

## TV Mode - FULLY WORKING (both demos pass)
- ✅ CREATE TABLE
- ✅ INSERT (multi-row)
- ✅ SELECT *, WHERE, GROUP BY, ORDER BY, UPDATE
- ✅ Pipeline API: map/filter/each with > operator

## VM Mode - BROKEN (deep compiler bug)
- db_open/get_db/parse_sql work; exec_create_table works when called directly
- exec_insert and deeper dispatch fails: `. only on map: INTEGER`
- Root cause: VM compiler bug in `compileImport` for large modules (>140 functions)
  - Module global variable access + nested function calls in compiled closures
  - Likely slot/index mismatch between compiled function bodies and main scope
- 500-simple-function module works; complex nested functions with globals fail

## Pipeline API
Exports in `sqlite` module (pipe-modules):
- `q(handle, sql)` → short alias for `db_query`
- `exec(handle, sql)` → short alias for `db_exec`
- `row_get(row, key)` → nil-safe dynamic field access (uses `get` builtin)
- `row_eq(row, key, val)` → predicate for `filter` (equals check)
- `row_ne(row, key, val)` → predicate for `filter` (not-equals check)

Demo: `examples/sqlite_pipeline.pipe` — pipeline patterns
```
db_query h "SELECT *" > filter is_high > map fmt > each print
```

## Go-level Fixes
1. `vm.go:compareOp`: nil comparison fallback
2. `object.go:bSet`: list support with int index
3. `eval.go:extendFunctionEnv`: bound check
4. `eval.go:evalInfixExpression`: && / || short-circuit

## SQLite Module Fixes
5. `sql_lex_make_token`: removed dead `if typ=="kw"` (token corruption root cause)
6. 85× `get(map, "key")` → `map.key` dot notation
7. Registry: `map`+`to_str` → `list`+`at`
8. `parse_select_item`: star (*) wrapped in `selitem` node
9. FULL keyword case normalization (SQL case-insensitive)
10. `new_table`: separate lists for cols/colnames/rows
11. `eval_where` argument order fix (row was 6th instead of 3rd)
12. `gen/builtins.go`: removed `db_*` entries

## Builtins
- 165 total (~140 object.go + 19 bytes.go + 6 file_io.go)
- `db_*` builtins: deleted from Go runtime

## Binary
- ~10 MB
