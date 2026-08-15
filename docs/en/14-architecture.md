# Architecture

This document describes the internal architecture of the Pipe language implementation. Pipe is written entirely in Go with zero external dependencies, producing a single ~16MB statically-linked binary.

## Overall Architecture

```
                          ┌──────────────────────┐
                          │     Source Code       │
                          │     (.pipe file)      │
                          └──────────┬───────────┘
                                     │
                              ┌──────▼──────┐
                               │    Lexer     │   ~204+485 lines
                              │  (pkg/lexer) │   token.go + lexer.go
                              └──────┬───────┘
                                     │ Token stream
                              ┌──────▼──────┐
                               │    Parser    │   ~1245 lines
                              │  (pkg/parser)│   recursive descent + Pratt
                              └──────┬───────┘
                                     │ AST
                    ┌────────────────┼────────────────┐
                    │                │                │
           ┌────────▼──────┐  ┌──────▼──────┐  ┌─────▼────────┐
           │  Tree-Walker  │  │  Compiler   │  │   Formatter  │
           │   (pkg/eval)  │  │(pkg/compiler)│  │(pkg/formatter)│
            │  ~1207+152 ln │  │ ~1184+117 ln │  │   ~477 lines │
           └────────┬──────┘  └──────┬──────┘  └──────────────┘
                    │                │
                    │         ┌──────▼──────┐
                    │         │      VM      │
                    │         │   (pkg/vm)   │
                    │         │   ~775 lines │
                    │         └──────────────┘
                    │                │
                    ▼                ▼
              Program Output   Program Output
```

The architecture uses a dual-execution strategy: during development, code runs through the tree-walking interpreter (default mode). For production or performance-sensitive code, the bytecode VM compiles the AST to optimized bytecode first.

Supporting modules:
- **Cache** (`pkg/cache/`, ~191 lines): writes/reads compiled `.pipec` bytecode files with SHA-256 source validation
- **Build** (`pkg/build/`, ~94 lines): produces self-extracting binaries by appending source to the Pipe runtime
- **Object** (`pkg/object/`, ~3021 lines): all runtime value types

## Lexer (`pkg/lexer/`)

The lexer converts raw source text into a stream of tokens. It consists of two files:

| File | Lines | Purpose |
|------|-------|---------|
| `token.go` | 204 | Token type definitions, keyword map, string representation |
| `lexer.go` | 485 | Character-level scanning, indent tracking, tokenization |

### 67 Token Types

**Literals** (4):
`IDENT`, `INT`, `FLOAT`, `STRING`

**Operators** (18):
`ASSIGN` (`=`), `PLUS` (`+`), `MINUS` (`-`), `STAR` (`*`), `SLASH` (`/`), `PERCENT` (`%`), `EQ` (`==`), `NOT_EQ` (`!=`), `LT` (`<`), `GT` (`>`), `LTE` (`<=`), `GTE` (`>=`), `CONCAT` (`++`), `BANG` (`!`), `AND` (`&&`), `OR` (`||`), `POWER` (`**`), `DOTDOT` (`..`)

**Compound Assignment** (5):
`PLUSEQ` (`+=`), `MINUSEQ` (`-=`), `STAREQ` (`*=`), `SLASHEQ` (`/=`), `PERCENTEQ` (`%=`)

**Pipeline** (4):
`PIPE` (`|`), `ARROW` (`>`), `ARROW2` (`>>`), `FAT_ARROW` (`->`)

**Punctuation** (10):
`LPAREN`, `RPAREN`, `LBRACKET`, `RBRACKET`, `LBRACE`, `RBRACE`, `COMMA`, `SEMICOLON`, `DOT`, `COLON`

**Structure** (3):
`NEWLINE`, `INDENT`, `DEDENT`

**Keywords** (20):
`FN`, `MATCH_KW`, `IF`, `ELSE`, `WHILE`, `FOR`, `BREAK`, `CONTINUE`, `IMPORT`, `EXPORT`, `ENUM`, `DEFER`, `RETURN`, `TRY`, `CATCH`, `TRYAI`, `TEST`, `TRUE`, `FALSE`, `NIL`

**Special** (2):
`ILLEGAL`, `EOF`

### INDENT/DEDENT Mechanism

Pipe uses Python-style significant whitespace. The lexer maintains an `indentStack` (initialized with `[0]`) and tracks indentation at the start of each line:

1. **At line start** (`atLineStart = true`): The lexer counts leading space characters (tabs counted as 4 spaces).
2. **Empty lines and comments** are skipped entirely (no tokens emitted).
3. **Inside brackets** (`parenDepth > 0`): INDENT/DEDENT tracking is disabled — content inside `()`, `[]`, `{}` is treated as a single logical line.
4. **Indent increase** (`indent > top`): Push new indent level, emit `INDENT` token.
5. **Indent decrease** (`indent < top`): Pop deeper levels, buffer multiple `DEDENT` tokens. Buffered tokens are emitted one at a time by `NextToken()`.
6. **Indent mismatch**: If the indent level doesn't match any previous level, an `ILLEGAL` token is emitted.

### Backtick Strings

Backtick-delimited strings (`` `...` ``) support multi-line content without escape processing, similar to Go raw strings. Newlines within backtick strings increment the line counter.

### Comment Handling

- `--` at the start of a line (after optional whitespace) causes the entire line to be skipped.
- `--` appearing inline (not at line start) is handled by `scanToken()` as a comment that reads to end-of-line and recursively calls `NextToken()`.

### Line/Column Tracking

Each token carries `Line` and `Col` fields for error reporting. The column is reset to 0 on each newline. The position is tracked via `pos` and `readPos` pointers, with `ch` as the current character and `peekChar()` for lookahead.

## Parser (`pkg/parser/`)

The parser uses a hybrid recursive descent + Pratt parsing approach (~1245 lines).

### Recursive Descent

Statement parsing (`parseStatement`) uses a top-down recursive descent strategy. Each statement type has a dedicated method:

```
parseStatement() → parseFnStatement / parseWhileStatement / parseForStatement
                  / parseReturnStatement / parseDeferStatement / parseExportStatement
                  / parseEnumStatement / parseImportStatement / parseVarStatement
                  / parseExpressionOrVarStatement
```

### Pratt Parsing

Expression parsing uses Pratt's top-down operator precedence (TDOP) technique with `prefixParseFns` and `infixParseFns` dispatch tables.

### 13 Precedence Levels

| Level | Constant | Operators |
|-------|----------|-----------|
| 1 | `PrecedenceLowest` | (none) |
| 2 | `PrecedenceOr` | `\|\|` |
| 3 | `PrecedenceAnd` | `&&` |
| 4 | `PrecedencePipeline` | `>` |
| 5 | `PrecedenceEquals` | `==`, `!=` |
| 6 | `PrecedenceCompare` | `<`, `>`, `<=`, `>=` |
| 7 | `PrecedenceSum` | `+`, `-` |
| 8 | `PrecedenceProduct` | `*`, `/`, `%` |
| 9 | `PrecedencePower` | `**` |
| 10 | `PrecedenceConcat` | `++` |
| 11 | `PrecedencePrefix` | `-x`, `!x` |
| 12 | `PrecedenceCall` | `f()`, `x[y]` |
| 13 | `PrecedenceDot` | `x.field` |

### Special Parsing Features

**Implicit Calls**: When parsing an expression followed by a value token (identifier, literal, etc.) without an operator between them, the parser automatically wraps the left expression in a `CallExpression`. This enables calling functions without parentheses:
```pipe
-- parsed as CallExpression{print, ["hello"]}
print "hello"
-- parsed as CallExpression{print, ["a", "b", "c"]}
print "a" "b" "c"
```

**Vertical Pipeline**: After parsing an expression, if the parser encounters `NEWLINE INDENT ARROW`, it enters vertical pipeline mode, parsing each `>`-prefixed line as a pipeline stage and wrapping them in chained `PipelineExpression` nodes.

**`_` Placeholder**: In pipeline mode, the `_` identifier in a call argument position is replaced with the piped value:
```pipe
-- "_" is replaced with the value 5
5 > add 1 _
```

**Slice vs Index**: The `[ ]` infix parser distinguishes between indexing (`x[i]`) and slicing (`x[i..j]` or `x[..j]`) by checking for a `..` token. Slices produce `SliceExpression` nodes; indices produce `InfixExpression{Operator: "[]"}`.

**Fn Literal vs Statement**: When `fn` is encountered as a prefix expression (inside another expression), it produces a `FnLiteral` expression node. When encountered at statement level, the named `FnStatement` is produced.

## AST (`pkg/ast/`)

The AST defines 36 node types implemented in ~433 lines: 13 statements, 20 expressions, the root `Program`, and the `MatchCase`/`StructField` helper nodes.

### 13 Statements

| Node | Description |
|------|-------------|
| `ExpressionStatement` | Wraps an expression as a statement |
| `FnStatement` | Named function definition (name, params, body block) |
| `VarStatement` | Variable declaration: `name: value` |
| `BlockStatement` | Sequence of statements (function body, if body, etc.) |
| `ReturnStatement` | Early return from a function |
| `BreakStatement` | Exit the nearest loop |
| `ContinueStatement` | Skip to the next loop iteration |
| `ImportStatement` | Module import with optional alias |
| `DeferStatement` | Deferred expression execution |
| `ExportStatement` | Symbol export (wraps an `FnStatement`) |
| `EnumStatement` | Enumeration definition |
| `TestStatement` | Test block: `test "name": ...` |
| `StructStatement` | Struct definition |

### 20 Expressions

| Node | Description |
|------|-------------|
| `Identifier` | Variable or function name |
| `IntegerLiteral` | Integer value |
| `FloatLiteral` | Floating-point value |
| `StringLiteral` | String value |
| `BooleanLiteral` | `true` or `false` |
| `NilLiteral` | `nil` |
| `PrefixExpression` | Unary operator (`-x`, `!x`) |
| `InfixExpression` | Binary operator (`+`, `==`, `&&`, etc.) |
| `PipelineExpression` | Pipeline operator `>` (horizontal or vertical) |
| `CallExpression` | Function call with explicit or implicit arguments |
| `ListLiteral` | List literal `[a, b, c]` |
| `MapLiteral` | Map literal `{key: val}` |
| `DotExpression` | Field access `obj.field` |
| `FnLiteral` | Anonymous function expression |
| `SliceExpression` | List slice `x[1..3]` |
| `IfExpression` | Conditional expression (returns a value) |
| `MatchExpression` | Pattern matching |
| `WhileExpression` | While loop |
| `ForExpression` | For-in loop (or classic for for literal iteration) |
| `TryExpression` | Try/catch error handling |

**Note**: `IfExpression`, `MatchExpression`, `WhileExpression`, `ForExpression`, and `TryExpression` are all **expressions** in Pipe — they evaluate to a value (the last expression in their block, or `nil`).

## Tree-Walker (`pkg/eval/`)

The tree-walking interpreter evaluates the AST directly by recursively traversing nodes. It consists of `eval.go` (~1207 lines) and `builtins.go` (~152 lines).

### Recursive AST Evaluation

Each AST node type has a corresponding evaluation case in `Eval(node, env)`. For example:
- `IntegerLiteral` → wraps in `object.Integer`
- `InfixExpression` → evaluates left and right, dispatches based on operator
- `CallExpression` → evaluates the function, evaluates arguments, applies the call

### Environment Chain

Environments track variable bindings and are chained for nested scopes. Each environment holds a `map[string]Object` store and a pointer to its outer environment. Variable lookup walks the chain; variable assignment (`:`) writes to the innermost environment where the variable was found (or creates it in the current one if new).

### Special Runtime Values

| Type | Purpose |
|------|---------|
| `ReturnValue` | Sent from `return` statements, unwrapped by function calls |
| `BreakValue` | Sent from `break`, caught by loop evaluation |
| `ContinueValue` | Sent from `continue`, caught by loop evaluation |
| `DeferredExpr` | Wraps an expression to be executed at scope exit |

### Tail-Call Optimization (TCO)

Pipe's tree-walker implements tail-call optimization: when a function's last expression is a call to itself (self-recursion), the interpreter reuses the current call frame instead of creating a new one. This prevents stack overflow for recursive algorithms like factorial or fibonacci (with accumulator pattern).

### Call-Stack / Error Tracking

Errors carry stack traces. Each function call pushes a frame onto the call stack, and errors include the stack trace in their message for debugging.

### Import System

`import "file.pipe"` evaluates the imported file in a new environment, then merges exported symbols (`export fn ...`) into the importer's namespace. The `PIPE_PATH` environment variable can specify additional search directories.

## Compiler (`pkg/compiler/`)

The compiler transforms the AST into bytecode instructions. It consists of `compiler.go` (~1184 lines) and `opcode.go` (~117 lines). The compiler is detailed in [Bytecode VM](13-bytecode-vm.md).

### Compilation Phases

1. **Symbol Definition**: Walk the AST and register all top-level function names and variables in the symbol table
2. **Instruction Generation**: Traverse the AST depth-first, emitting opcodes for each node
3. **Jump Patching**: Back-patch forward jump targets after their destinations are known
4. **Constant Pooling**: Deduplicate literals into the constant pool

### Short-Circuit Compilation

**`&&` (logical AND)**:
```
compile left
OpDup                    -- duplicate left result
OpJumpNotTruthy END      -- if falsy, jump to end (short-circuit)
OpPop                    -- discard duplicate
compile right            -- push right (truthy result)
END:                     -- result is on stack
```

**`||` (logical OR)**:
```
compile left
OpDup                    -- duplicate left result
OpJumpNotTruthy EVAL_R   -- if not truthy, evaluate right
OpJump END               -- otherwise skip right entirely
EVAL_R:
OpPop                    -- discard duplicate
compile right            -- push right result
END:
```

## Object Types (`pkg/object/`)

The object system defines all runtime value types in ~3021 lines (plus `environment.go`, ~66 lines).

### 12 Main Types

| Type | Go Struct | Description |
|------|-----------|-------------|
| `INTEGER` | `Integer{Value int64}` | Whole numbers |
| `FLOAT` | `Float{Value float64}` | Floating-point numbers |
| `STRING` | `String{Value string}` | Text strings |
| `BOOLEAN` | `Boolean{Value bool}` | `true` / `false` |
| `NIL` | `NilObject{}` | Null/nil value |
| `FUNCTION` | `Function{Name, Parameters, Body, Env, EvalCtx}` | User-defined function (tree-walker) |
| `COMPILED_FUNCTION` | `CompiledFunction{Instructions, NumLocals, NumFree}` | User-defined function (compiled) |
| `CLOSURE` | `Closure{Fn *CompiledFunction, Free []Object}` | Closure over compiled function |
| `LIST` | `List{Elements []Object}` | Ordered collection |
| `MAP` | `Map{Pairs map[string]Object}` | Key-value collection (string keys) |
| `ERROR` | `Error{Message string}` | Runtime error |
| `RESULT` | `Result{Ok bool, Val Object, Err string}` | Result type for error handling |

### Additional Types

| Type | Go Struct | Description |
|------|-----------|-------------|
| `FUTURE` | `Future{...}` | Async result from `go` / concurrent calls |
| `BUILTIN` | `BuiltinInfo{Name, Fn}` | Built-in function wrapper |
| `TCP_CONN` | `TcpConn{Handle}` | TCP client connection handle |
| `TCP_LISTENER` | `TcpListener{Handle}` | TCP server listener handle |

### TCP Connection Management

TCP connections and listeners are stored in global maps (`connStore`, `listeners`) protected by a mutex. Each connection/listener gets a monotonically increasing `ConnHandle` integer ID. The VM is capable of calling user functions from within builtins (e.g., `map`, `filter`) via the `callUserFn` bridge, though this is less efficient than tree-walker mode for higher-order functions.

## Cache (`pkg/cache/`)

The cache module (~191 lines) provides transparent bytecode caching via the `.pipec` file format.

### `.pipec` Binary Format

```
+--------+---------+------+--------------------+--------------+----------------+
| Magic  | Version | Hash | NumConstants       | Constants... | Instructions   |
| PIPEBC | 1 byte  |16 B  | 4 bytes (uint32)   | (variable)   | (variable)     |
| 6 bytes|         |      |                    |              |                |
+--------+---------+------+--------------------+--------------+----------------+
```

**Constants encoding** (per constant):
- Type tag: 1 byte (`1`=Integer, `2`=Float, `3`=String, `4`=CompiledFunction)
- Integer: 8 bytes (int64 big-endian)
- Float: 8 bytes (float64 big-endian)
- String: 2 bytes length + data
- CompiledFunction: 4 bytes numLocals + 4 bytes insLen + instruction bytes

### LoadOrCompile API

```go
func LoadOrCompile(filePath string) (*Bytecode, bool, error)
```

1. Read the source file and compute its SHA-256 hash
2. If a `.pipec` file exists with matching magic, version, and source hash → load and return (`true`)
3. Otherwise → lex, parse, compile, and return the bytecode (`false`)
4. Source hash comparison prevents stale cache from being used

## Formatter (`pkg/formatter/`)

The formatter (~477 lines) provides `pipe fmt <file>` functionality.

### AST-Based Formatting

The formatter re-parses the source code, then pretty-prints the AST with consistent 4-space indentation. Function/export/enum definitions get blank line separation. If parsing fails (e.g., malformed source), the formatter falls back to whitespace normalization — normalizing indentation to multiples of 4 spaces while preserving code structure.

### Whitespace Fallback

When the AST cannot be produced (parse errors), `fallbackFormat()` operates on raw lines: trimming trailing whitespace, normalizing leading whitespace to 4-space multiples, and preserving logical structure.

## Build (`pkg/build/`)

The build module (~94 lines) creates self-extracting binaries using a `PIPEBUILD` marker.

### Self-Extracting Binary Format

```
+------------------+------------+----------+----------+
| Pipe Runtime     | \nPIPEBUILD\n | Size\n | Source   |
| (binary copy)    | (marker)     | (ASCII)  | (UTF-8)  |
+------------------+------------+----------+----------+
```

The runtime reads itself, scans backward from EOF for the `PIPEBUILD` marker, extracts the size and source code, then executes it. The output binary is set to executable mode (`0755`).

## Technical Data Table

| Metric | Value |
|--------|-------|
| **Go source lines** | ~29,200 total (~22,800 excluding tests) |
| **Go packages** | 13 (`ai`, `analysis`, `ast`, `build`, `cache`, `compiler`, `eval`, `formatter`, `lexer`, `object`, `parser`, `stdlib`, `vm`) |
| **External dependencies** | 0 (standard library only) |
| **Binary size** | ~8 MB (dependency-free, statically linked) |
| **Tests** | 290 (across 12 packages) |
| **Examples** | 52 example programs in `examples/` |
| **Built-in functions** | 168 |
| **Opcodes** | 43 |
| **AST node types** | 35 (13 statements, 20 expressions, program, match case) |
| **Token types** | 66 |
| **Operand stack size** | 2048 slots |
| **Maximum call frames** | 1024 |
| **Maximum global variables** | 65536 |
| **Precedence levels** | 13 |
| **Symbol scopes** | 4 (Global, Local, Free, Builtin) |
