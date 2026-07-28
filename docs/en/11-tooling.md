# 11. Tooling

Pipe ships with a comprehensive set of command-line tools for development: a REPL, formatter, test runner, benchmark harness, AST dumper, self-extracting binary builder, and bytecode caching.

---

## 11.1 CLI Flags

The Pipe interpreter supports the following command-line flags:

| Flag | Description |
|------|-------------|
| `-h` | Show help and exit |
| `-vm` | Run in Bytecode VM mode (faster, limited features) |
| `-q` | Quiet mode (suppress banner in REPL) |
| `-ast` | Print AST (Abstract Syntax Tree) and exit |
| `-fmt` | Format a Pipe source file in-place |
| `-test` | Run tests from `.pipe` files |
| `-bench` | Run benchmarks |
| `-build` | Create a self-extracting binary |

### `-h` — Help

```bash
pipe -h
```

Displays usage information, available flags, and version.

### `-vm` — Bytecode VM Mode

```bash
pipe -vm my_script.pipe
pipe -vm
```

Runs the program or REPL using the Bytecode VM instead of the Tree-Walker interpreter. Programs run approximately 7x faster, but some features are unavailable (see Chapter 12).

### `-q` — Quiet Mode

```bash
pipe -q
```

Starts the REPL without printing the welcome banner and version information.

```bash
# Normal REPL:
$ pipe
Pipe Language v1.0.0
>>>

# Quiet REPL:
$ pipe -q
>>>
```

### `-ast` — AST Output

```bash
pipe -ast hello.pipe
```

Parses the source file and prints its Abstract Syntax Tree to stdout. Useful for debugging the parser and understanding how Pipe sees your code.

**Example input (`hello.pipe`):**

```pipe
let greeting = "Hello, world!"
print greeting
```

**Example AST output:**

```
(program
  (let
    (ident greeting)
    (string "Hello, world!"))
  (expr
    (call
      (ident print)
      (ident greeting))))
```

Each node in the tree shows the expression type and its children, reflecting the parsed structure before evaluation.

### `-fmt` — Formatter

```bash
pipe -fmt my_script.pipe
pipe -fmt *.pipe
```

Formats Pipe source code according to the canonical style. The formatter operates **in-place**, overwriting the file with the formatted version.

**Before formatting:**

```pipe
let x=10
 let y = 20
print   ( x+ y )
fn add(a,b){a+b}
```

**After formatting:**

```pipe
let x = 10
let y = 20
print (x + y)

let add = fn(a, b) { a + b }
```

The formatter enforces:

- Consistent spacing around operators (`+`, `-`, `=`, etc.)
- Consistent spacing after commas
- Consistent indentation (no trailing whitespace)
- Newline separation between top-level statements
- Standardized function and block formatting

### `-test` — Test Runner

```bash
pipe -test
pipe -test tests/
pipe -test test_math.pipe
```

Discovers and runs test functions defined in Pipe files. By default, it looks for files matching `*_test.pipe` or files in a `test/` directory.

**Test file example (`math_test.pipe`):**

```pipe
import "math.pipe" as m

test "addition works" {
    let result = m.add 2 3
    if result != 5 {
        print "FAIL: expected 5, got " + result
    }
}

test "multiplication works" {
    let result = m.multiply 4 5
    if result != 20 {
        print "FAIL: expected 20, got " + result
    }
}

test "division by zero returns nil" {
    let result = m.safe_divide 10 0
    if result != nil {
        print "FAIL: expected nil"
    }
}
```

When test functions throw or fail, the runner reports them. A test that completes without errors is considered passing.

**Sample output:**

```
Running tests in math_test.pipe...
  PASS  addition works
  PASS  multiplication works
  PASS  division by zero returns nil

3/3 tests passed.
```

### `-bench` — Benchmark Runner

```bash
pipe -bench
pipe -bench benchmarks/
```

Runs benchmark functions defined with the `bench` keyword and reports execution time.

**Benchmark file example (`calc_bench.pipe`):**

```pipe
bench "fibonacci recursive" {
    let fib = fn(n) {
        if n <= 1 { n } { (fib (n - 1)) + (fib (n - 2)) }
    }
    each range 0 20 fn(_) { fib 20 }
}

bench "list operations" {
    each range 0 1000 fn(_) {
        let xs = range 0 100
        let mapped = map xs fn(x) { x * 2 }
        let filtered = filter mapped fn(x) { x > 50 }
    }
}

bench "string processing" {
    each range 0 500 fn(_) {
        let s = "the quick brown fox jumps over the lazy dog"
        let parts = split s " "
        let joined = join parts ","
        let upper = upper joined
    }
}
```

**Sample output:**

```
Running benchmarks...
  fibonacci recursive ... 245ms (20 iterations)
  list operations ...... 180ms (1000 iterations)
  string processing .... 95ms (500 iterations)
```

The benchmark runner:

1. **Pre-executes the benchmark** to determine the optimal number of iterations.
2. **Re-executes** for the determined number of runs.
3. **Reports total and per-iteration time** in milliseconds.

### `-build` — Self-Extracting Binary

```bash
pipe -build my_script.pipe
```

Creates a self-contained, self-extracting executable binary that bundles the Pipe source with the interpreter.

**How it works:**

The resulting binary contains:

1. The compiled Pipe interpreter
2. The source file contents
3. A special **PIPEBUILD** marker that the binary uses to find the embedded source at runtime

When executed, the binary reads everything after the `PIPEBUILD` marker, compiles it with the Bytecode VM, and runs the program.

```bash
# Build the binary
pipe -build server.pipe

# Run it directly
./server
```

**Binary structure:**

```
┌─────────────────────────┐
│  Pipe Interpreter (ELF) │  <- Native executable
├─────────────────────────┤
│  PIPEBUILD\n            │  <- Magic marker
├─────────────────────────┤
│  Source Code            │  <- Embedded Pipe source
└─────────────────────────┘
```

---

## 11.2 REPL (Read-Eval-Print Loop)

Start the REPL with no arguments:

```bash
pipe
```

### REPL Commands

| Command | Description |
|---------|-------------|
| `:help` | Show help |
| `:quit` or `:q` | Exit the REPL |
| `:vm` | Toggle Bytecode VM mode |
| `:clear` | Clear the current environment (reset variables) |
| `:verbose` | Toggle verbose output |

### Basic Usage

```pipe
>>> let x = 10
>>> print x
10
>>> x * 2
20
>>> let greet = fn(name) { "Hello, " + name + "!" }
>>> (greet "World")
Hello, World!
>>> type_of [1, 2, 3]
list
```

### Multi-Line Mode

The REPL automatically enters multi-line mode when you start an unfinished statement (unclosed block, missing expression, etc.):

```pipe
>>> let add = fn(a, b) {
...     let sum = a + b
...     sum
... }
>>> (add 5 3)
8
```

```pipe
>>> let person = {
...     "name" = "Alice",
...     "age" = 30
... }
>>> person.name
Alice
```

```pipe
>>> if x > 5 {
...     print "big"
... } {
...     print "small"
... }
big
```

The `... ` prompt indicates continuation lines. You can press Enter on an empty line or close an open brace to complete the input.

### Expression Evaluation

If an input is a single expression (not a statement), the REPL evaluates it and prints the result:

```pipe
>>> 2 + 2
4
>>> [1, 2, 3]
[1, 2, 3]
>>> "hello" + " world"
hello world
```

---

## 11.3 Formatter Details

The `-fmt` flag rewrites Pipe source files to conform to canonical formatting rules.

### Before/After Examples

**Example 1: Spacing**

Before:
```pipe
let x=10
let y=20
print(x+y)
```

After:
```pipe
let x = 10
let y = 20
print (x + y)
```

**Example 2: Function Definitions**

Before:
```pipe
fn add(a,b){return a+b}
```

After:
```pipe
let add = fn(a, b) { return a + b }
```

**Example 3: Maps and Lists**

Before:
```pipe
let data={"a"=1,"b"=2,"c"=[3,4,5]}
```

After:
```pipe
let data = { "a" = 1, "b" = 2, "c" = [3, 4, 5] }
```

**Example 4: Pipe Operators**

Before:
```pipe
let result=data|filter fn(x){x>0}|map fn(x){x*x}|sort
```

After:
```pipe
let result = data
    | filter fn(x) { x > 0 }
    | map fn(x) { x * x }
    | sort
```

**Example 5: If/Else Blocks**

Before:
```pipe
if x>0{print"positive"}else{print"zero or neg"}
```

After:
```pipe
if x > 0 {
    print "positive"
} {
    print "zero or neg"
}
```

---

## 11.4 Test Runner Details

The `-test` flag discovers test functions and runs them, reporting pass/fail status.

### Test Function Syntax

The `test` keyword defines a test case:

```pipe
test "description of the test" {
    # test body
    # assertion failures should use print or error
}
```

### Test Discovery

The runner searches for:

1. Files with names matching `*_test.pipe`
2. Files in a `test/` directory
3. Explicitly passed file paths

### Comprehensive Test Example

```pipe
# File: string_utils_test.pipe
import "string_utils.pipe" as su

test "capitalize single word" {
    let result = su.capitalize "hello"
    if result != "Hello" {
        print "FAIL: capitalize 'hello' => '" + result + "' expected 'Hello'"
    }
}

test "capitalize empty string" {
    let result = su.capitalize ""
    if result != "" {
        print "FAIL: capitalize '' => '" + result + "'"
    }
}

test "reverse preserves length" {
    let original = "hello"
    let reversed = su.reverse original
    if len original != len reversed {
        print "FAIL: lengths differ: " + (len original) + " vs " + (len reversed)
    }
}

test "reverse palindrome" {
    let result = su.reverse "racecar"
    if result != "racecar" {
        print "FAIL: reverse 'racecar' => '" + result + "'"
    }
}
```

---

## 11.5 Benchmark Details

The `-bench` flag measures execution time of benchmark functions.

### How Benchmarks Work

1. **Warmup phase:** The benchmark function is executed once to determine baseline timing.
2. **Iteration calculation:** Based on the warmup, the runner calculates how many iterations fit within a target duration.
3. **Measurement phase:** The function is executed repeatedly for the determined iterations.
4. **Reporting:** Total time and per-iteration average are displayed.

### Benchmark Structure

```pipe
bench "name of the benchmark" {
    # Code to benchmark
    # Use each/range for multiple iterations
}
```

### Three Benchmark Examples

**Benchmark 1: Computational — Fibonacci**

```pipe
bench "fibonacci(30)" {
    let fib = fn(n) {
        if n <= 1 { n } { (fib (n - 1)) + (fib (n - 2)) }
    }
    fib 30
}
# Typical: ~15ms per iteration (Tree-Walker), ~2ms (VM)
```

**Benchmark 2: Data Transform — List pipeline**

```pipe
bench "list pipeline 10k" {
    let data = range 0 10000
    let result = data
        | map fn(x) { x * 3 }
        | filter fn(x) { x % 2 == 0 }
        | map fn(x) { x / 2 }
    len result
}
# Typical: ~50ms per iteration (Tree-Walker), ~7ms (VM)
```

**Benchmark 3: I/O — File read and JSON parse**

```pipe
bench "read and parse JSON" {
    let raw = read_file "data.json"
    let parsed = parse_json raw
    let names = map parsed.users fn(u) { u.name }
    len names
}
# Typical: ~10ms per iteration (I/O dominated)
```

---

## 11.6 AST Debugging

The `-ast` flag is primarily for debugging the parser and understanding code structure.

```bash
pipe -ast examples/hello.pipe
```

### AST Node Types

Common AST node types:
- `program` — root node, contains all statements
- `let` — variable binding
- `ident` — identifier reference
- `string` / `number` — literals
- `call` — function invocation
- `fn` — function literal
- `if` — conditional expression
- `infix` — binary operator expression
- `prefix` — unary operator expression
- `block` — block expression `{ ... }`
- `map` / `list` — collection literals

---

## 11.7 Bytecode Cache (`.pipec`)

When running in Bytecode VM mode (`-vm`), Pipe automatically creates a bytecode cache file with the `.pipec` extension alongside the source file.

```
program.pipe      # source file
program.pipec     # cached bytecode (generated automatically)
```

### Cache Format

The `.pipec` file contains:

```
┌─────────────────────────┐
│  Magic: PIPEV1\n        │  8-byte version identifier
├─────────────────────────┤
│  Source Hash            │  SHA-256 hash of source file
├─────────────────────────┤
│  Constant Pool          │  String/number constants
│  Count: N               │
│  Constant[0]            │
│  Constant[1]            │
│  ...                    │
├─────────────────────────┤
│  Instructions           │  Opcode stream
│  Count: M               │
│  Bytecode[0]            │
│  Bytecode[1]            │
│  ...                    │
└─────────────────────────┘
```

### Cache Invalidation

The cache is automatically invalidated when the source file is modified (detected via SHA-256 hash comparison). If the source hash in the cache doesn't match the current source file, the bytecode is recompiled.

### Cache Location

The `.pipec` file is always created in the same directory as the source file. For example:

```
lib/math.pipe      →  lib/math.pipec
test/test_app.pipe →  test/test_app.pipec
```

### Disabling Cache

To disable bytecode caching, delete the `.pipec` file manually, or add `.pipec` to your `.gitignore`:

```bash
echo "*.pipec" >> .gitignore
```

---

## 11.8 Running Examples

Pipe ships with example programs in the `examples/` directory.

```bash
# Run a single example
pipe examples/hello.pipe

# Run with VM for speed
pipe -vm examples/fib.pipe

# Format all examples
pipe -fmt examples/*.pipe

# Run tests
pipe -test

# Run benchmarks
pipe -bench
```

---

## 11.9 Running Go Tests

Pipe's implementation is written in Go. To run the Go-level tests (interpreter, parser, lexer, VM):

```bash
# Run all Go tests
go test ./...

# Run with verbose output
go test -v ./...

# Run tests for a specific package
go test -v ./pkg/parser/
go test -v ./pkg/evaluator/
go test -v ./pkg/vm/

# Run with race detection
go test -race ./...

# Run with coverage
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run benchmarks
go test -bench=. ./...
go test -bench=. -benchmem ./...
```

### Test Structure

Go tests follow the standard `go test` conventions:

```
pkg/
  lexer/
    lexer.go
    lexer_test.go
  parser/
    parser.go
    parser_test.go
  evaluator/
    evaluator.go
    evaluator_test.go
  vm/
    vm.go
    vm_test.go
  compiler/
    compiler.go
    compiler_test.go
```
