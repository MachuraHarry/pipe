# SQLite Module - State Memory

## TV Mode - FULLY WORKING (all demo queries pass)
- ✅ CREATE TABLE
- ✅ INSERT (4 rows, multiple-row insert)
- ✅ SELECT * (correct row data)
- ✅ WHERE (filtered queries: `priority = 'high'` → 2 matching rows)
- ✅ GROUP BY (aggregate: `priority, COUNT(*)` → 3 groups with counts)
- ✅ UPDATE (with WHERE)

## VM Mode - BROKEN (deep compiler bug)
- `db_open` works, `db_open`+`get_db` works, `parse_sql` works
- Individual functions work when called directly from main: exec_create_table, new_table, get_table
- `exec_insert` (and deeper dispatch) fails — error: `. only on map: INTEGER`
- Root cause: VM compiler bug in `compileImport` for large modules (140+ functions)
  - Module global variable access + nested function calls in compiled closures
  - Likely slot/index mismatch between compiled function bodies and main scope
- 500-simple-function module works; complex nested functions with globals fail
- Workaround: dot notation bypasses `get` builtin but can't fix `exec_insert` dispatch

## Go-level Fixes Applied
1. `vm.go:compareOp`: nil comparison fallback (pointer equality)
2. `object.go:bSet`: list support with int index
3. `eval.go:extendFunctionEnv`: bound check
4. `eval.go:evalInfixExpression`: && / || short-circuit

## SQLite Module Fixes Applied
5. `sql_lex_make_token`: removed dead `if typ=="kw"` (ROOT CAUSE: token corruption)
6. 85× `get(map, "key")` → `map.key` dot notation
7. Registry: `map`+`to_str` → `list`+`at` 
8. `parse_select_item`: star (*) wrapped in `selitem` node
9. **FULL keyword case normalization**: lexer lowers kw text, all parser/evaluator comparisons lowercase
10. `new_table`: separate lists for cols/colnames/rows
11. `eval_where` argument order fix (Z.2004: row was in wrong position)
12. `gen/builtins.go`: removed `db_*` entries (builtins deleted from runtime)

## Builtins Status
- Runtime: 165 total (~140 object.go + 19 bytes.go + 6 file_io.go)
- `db_*` builtins: deleted from Go runtime (db_builtins.go now empty stubs)
- SQLite module: pure Pipe implementation (2444 lines)

## Binary
- ~10 MB (not ~7 MB)
