# SQLite Module - Bug Memory (Final State)

## TV Mode - WORKING: CREATE TABLE, INSERT, SELECT *
- ✅ CREATE TABLE: works
- ✅ INSERT: 4 rows inserted correctly  
- ✅ SELECT *: returns correct data (id is nil because not provided in INSERT)
- ⏳ WHERE: returns empty (eval_compare/eval_where bug)
- ⏳ GROUP BY: parsing issue (parse_expr_primary token consumption)

## VM Mode - BROKEN: Module-closure function calls return closures
- `get`, `set`, `new_db` etc. calls return the function object instead of result
- OpCall is either not emitted or not executed for global function calls in closures
- Dot notation (`map.field`) bypasses `get` via OpDot → partial workaround
- List+index (`at registry h`) bypasses `get` for registry access

## Go-level Fixes Applied
1. `vm.go`: compareOp nil fallback (pointer equality)
2. `object.go`: bSet list support (int index)
3. `eval.go`: extendFunctionEnv bound check (prevents index out of range)
4. `eval.go`: && / || short-circuit evaluation (was always evaluating both sides!)

## Module Fixes Applied
5. sql_lex_make_token: removed `if typ=="kw"` no-body (ROOT CAUSE of token corruption)
6. 85× `get map "key"` → `map.key` dot notation
7. Registry: map+to_str → list+at
8. parse_select_item: star wrapped in selitem node
9. Case mismatch: INSERT→insert, SELECT→select
10. new_table: separate lists for cols/colnames/rows
11. Various intermediate variable fixes for nested calls
