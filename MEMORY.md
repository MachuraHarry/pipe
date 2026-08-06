# SQLite Module - Bug Memory

## STATUS: VM Compiler has fundamental bug with get/set builtins in large module closures.

### What works
- TV mode: Tokenizer, parser, CREATE TABLE (simple SQL)
- VM mode small modules: `get registry key` works

### What does NOT work
- **VM mode large module (2400 lines, 140+ functions): `get`/`set` builtins return CLOSURE instead of executing the call**
  - Even `get local_var "key"` inside a module function returns ERROR for nested indirection
  - `at registry h` (list index) might work but other get/set calls in deeper code fail

### Root cause
VM compiler emits wrong bytecode for builtin function calls (`get`/`set`) inside module closures when the module has many globals. The `OpCall` instruction is either not emitted or `OpGetBuiltin` pushes wrong value.

### Fixes applied in Go
1. vm.go compareOp: nil fallback (pointer equality)
2. object.go bSet: list support (int index)

### Fixes applied in sqlite.pipe
3. Case: INSERT→insert, SELECT→select etc
4. sql_lex_make_token: removed `if typ=="kw"` no-body (TV root cause)
5. sql_lex_peek: intermediate vars for get st "pos"/get st "sql_src"
6. sql_lex_next: intermediate p var
7. parse_advance: intermediate p var
8. sql_lex_emit: intermediate tkl var
9. new_table: separate lists for cols/colnames/rows (not shared)
10. add_column: intermediate vars
11. get_col_index: intermediate vars (but calls get which still fails)
12. get_table/set_table: explicit return + intermediate tbls var
13. get_db/set_db/drop_db: changed from map+to_str to list+index (uses `at`/`set` but set/get broken too)
14. db_exec: now uses get_db (list index)
15. exec_insert: intermediate empty_aliases var
