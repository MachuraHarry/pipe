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
| `-search` | Search the Pipe module registry |
| `-get` | Download a module from the registry |
| `-init` | Scaffold a new module with pipe.json |
| `-validate` | Check a module's pipe.json validity |
| `-install` | Install dependencies from pipe.json |
| `-gen-registry` | Generate registry.json from pipe.json files |

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
Pipe v0.9.3.1 — REPL
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
greeting: "Hello, world!"
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

```text
x: 10
 y: 20
print   ( x+ y )
fn add(a,b){a+b}
```

**After formatting:**

```pipe
x: 10
y: 20
print (x + y)

add: (fn a b
    a + b)
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

test "addition works"
    result: m.add 2 3
    if result != 5
        print "FAIL: expected 5, got " + result

test "multiplication works"
    result: m.multiply 4 5
    if result != 20
        print "FAIL: expected 20, got " + result

test "division by zero returns nil"
    result: m.safe_divide 10 0
    if result != nil
        print "FAIL: expected nil"
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
bench "fibonacci recursive"
    fib: fn n
        if n <= 1
            n
        else
            (fib (n - 1)) + (fib (n - 2))
    noop: fn _
        fib 20
    each range 0 20 noop

bench "list operations"
    each range 0 1000 fn _
        xs: range 0 100
        double: fn x
            x * 2
        mapped: map xs double
        above_fifty: fn x
            x > 50
        filtered: filter mapped above_fifty

bench "string processing"
    each range 0 500 fn _
        s: "the quick brown fox jumps over the lazy dog"
        parts: split s " "
        joined: join parts ","
        upper: upper joined
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

**Passing arguments to a built binary:**

Arguments given on the command line are available inside the script via the
`args` builtin — exactly like when running `pipe script.pipe ...`:

```bash
# Build a CLI tool
pipe -build greet.pipe

# Run it with arguments
./greet Alice Bob
```

```pipe
-- greet.pipe
for name in args
    print "Hello, " ++ name
-- ./greet Alice Bob
-- -> Hello, Alice
-- -> Hello, Bob
```

**Embedding extra files (`--embed-file`):**

The `--embed-file` flag embeds additional files (data, config, prompts, …) into
the binary. At runtime the binary extracts them to a fresh temporary directory
and changes its working directory to that location — so the script can simply
read them by name with `read_file`.

```bash
# Bundle a script with its data
pipe -build agent.pipe -o agent --embed-file prompts.txt --embed-file config.json
```

```pipe
-- agent.pipe — reads the embedded files by name
system_prompt: read_file "prompts.txt"
settings: parse_json (read_file "config.json")

print system_prompt
print settings.model
```

```bash
# Run the self-contained artifact from anywhere
cp agent /opt/tools/
/opt/tools/agent
```

**How it works (details):**

1. The binary is the native Pipe interpreter followed by the `PIPEBUILD` source section.
2. An optional `PIPEFILES` section stores each embedded file as `name + size + bytes`.
3. At startup the binary extracts the files to a fresh `pipe_embedded_*` directory
   under the system temp directory and changes its working directory to it.

**Binary structure:**

```
┌─────────────────────────┐
│  Pipe Interpreter (ELF) │  <- Native executable
├─────────────────────────┤
│  PIPEBUILD\n            │  <- Magic marker
├─────────────────────────┤
│  Source Code            │  <- Embedded Pipe source
├─────────────────────────┤
│  PIPEFILES\n            │  <- Optional marker (only with --embed-file)
├─────────────────────────┤
│  name + size + bytes    │  <- Embedded data files
└─────────────────────────┘
```

---
### Module Management Commands

Pipe includes a built-in module system for sharing and reusing code. Modules are published via the [pipe-modules](https://github.com/MachuraHarry/pipe-modules) registry.

### `-search` — Search Registry

```bash
pipe -search
pipe -search log
```

Lists all available modules in the registry. With an optional search term, filters by name.

```bash
$ pipe -search
Pipe Modules

  jpipe  (latest: 1.0.0)
    JSON path query — navigate, pick, flatten JSON/Map structures
    Functions: jp, jpick, jkeys, jflatten

  pipe-http  (latest: 1.0.0)
    HTTP client with custom headers, auth, all HTTP methods
    Functions: hget, hpost, hput, ... (12)
```

### `-get` — Download Module

```bash
pipe -get log-analyzer
pipe -get log-analyzer@1.0.0
pipe -get https://raw.githubusercontent.com/.../module.pipe
```

Downloads a module into `~/.pipe/modules/` so it can be imported from any script. Supports:
- **Short names:** `pipe -get jpipe` (resolves via registry)
- **Version pinning:** `pipe -get jpipe@1.0.0`
- **Direct URLs:** `pipe -get https://...`

### `-init` — Scaffold New Module

```bash
pipe -init my-module
```

Creates a new module directory with three files:

```
my-module/
├── pipe.json       ← Module manifest
├── module.pipe     ← Source code
└── README.md       ← Documentation
```

The `pipe.json` contains metadata (name, version, description, exports, dependencies).

### `-validate` — Check Module

```bash
pipe -validate
pipe -validate my-module
```

Validates a module's `pipe.json` for correctness. Checks that required fields (`name`, `version`) are present and the name uses valid characters.

```bash
$ pipe -validate my-module
OK my-module v0.1.0
  Exports: hello
```

### `-install` — Install Dependencies

```bash
pipe -install
```

Reads the `dependencies` field from `pipe.json`, resolves each dependency through the registry, downloads all modules (recursively, including transitive dependencies), and writes a `pipe.lock` lockfile.

```bash
$ pipe -install
Installing dependencies for my-app…
  ✓ pipe-http v1.0.0
  ✓ jpipe v1.0.0
Saved lockfile to pipe.lock
```

The `pipe.lock` file pins exact versions and contains SHA-256 checksums for reproducible builds:

```json
{
  "modules": {
    "jpipe": {
      "version": "1.0.0",
      "url": "https://raw.githubusercontent.com/.../jpipe/module.pipe",
      "checksum": "c8b351f760fef3..."
    }
  }
}
```

**Version constraints** in `pipe.json`:
- `"1.0.0"` — exact version
- `"^1.0.0"` — latest compatible (^1.x.x)
- `"latest"` or `"*"` — always latest

### `-gen-registry` — Generate Registry

```bash
pipe -gen-registry .
```

Scans a directory for module subdirectories with `pipe.json` files and generates `registry.json` automatically. Modules without `pipe.json` are preserved from the existing registry.

```bash
$ pipe -gen-registry pipe-modules/
Generated registry.json from pipe.json files
Scanned 21 modules (8 with pipe.json, 13 legacy):
  ✓ jpipe v1.0.0 — 4 exports
  ○ sqlite (no pipe.json, preserved)
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

```text
x: 10
print x
10
x * 2
20
greet: (fn name
    "Hello, " + name + "!")
(greet "World")
Hello, World!
type_of ([1, 2, 3])
list
```

### Multi-Line Mode

The REPL automatically enters multi-line mode when you start an unfinished statement (unclosed block, missing expression, etc.):

```pipe
add: fn a b
    sum: a + b
    sum
(add 5 3)
8
```

```text
person: {
    name: "Alice",
    age: 30
person.name
Alice
```

```text
if x > 5
    print "big"
    else
    print "small"
big
```

The `... ` prompt indicates continuation lines. You can press Enter on an empty line or close an open brace to complete the input.

### Expression Evaluation

If an input is a single expression (not a statement), the REPL evaluates it and prints the result:

```pipe
2 + 2
4
[1, 2, 3]
[1, 2, 3]
"hello" + " world"
hello world
```

---

## 11.3 Formatter Details

The `-fmt` flag rewrites Pipe source files to conform to canonical formatting rules.

### Before/After Examples

**Example 1: Spacing**

Before:
```pipe
x: 10
y: 20
print(x+y)
```

After:
```pipe
x: 10
y: 20
print (x + y)
```

**Example 2: Function Definitions**

Before:
```text
fn add(a,b){return a+b}
```

After:
```pipe
add: fn a b
    a + b
```

**Example 3: Maps and Lists**

Before:
```pipe
data: {a: 1,b: 2,c: [3,4,5]}
```

After:
```pipe
data: { a: 1, b: 2, c: [3, 4, 5] }
```

**Example 4: Pipe Operators**

Before:
```text
result: data
    > filter fn x
        x > 0
    > map fn x
        x * x
    > sort
```

After:
```pipe
above_zero: fn x
    x > 0
square: fn x
    x * x

result: data
    > filter above_zero
    > map square
    > sort
```

**Example 5: If/Else Blocks**

Before:
```pipe
if x > 0
    print "positive"
else
    print "zero or neg"
```

After:
```pipe
if x > 0
    print "positive"
else
    print "zero or neg"
```

---

## 11.4 Test Runner Details

The `-test` flag discovers test functions and runs them, reporting pass/fail status.

### Test Syntax

The `test` keyword defines a named test block with indentation-based body:

```pipe
test "description"
    assert_eq (1 + 2) 3
    assert_lt 3 5
```

### Test Discovery

The runner searches the current directory for:

1. Files matching `*_test.pipe`
2. Files matching `*.test.pipe`

### Built-in Assertions

| Function | Description |
|----------|-------------|
| `assert(cond)` | Fails if value is not truthy |
| `assert_eq(a, b)` | Fails if values are not equal |
| `assert_not_eq(a, b)` | Fails if values are equal |
| `assert_lt(a, b)` | Fails unless `a < b` |
| `assert_gt(a, b)` | Fails unless `a > b` |
| `assert_error(fn)` | Fails if `fn()` does not error |

### Comprehensive Test Example

```text
test "string operations"
    assert_eq ("hello" ++ " world") "hello world"
    assert (len "hello") == 5

test "list operations"
    my_list: [1, 2, 3]
    assert_eq (len my_list) 3
    assert_eq (push my_list 4) [1, 2, 3, 4]

test "error handling"
    failing: (fn
        read_file "nonexistent"
    assert_error failing
    )

    )```

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
bench "name of the benchmark"
    -- Code to benchmark
    -- Use each/range for multiple iterations
```

### Three Benchmark Examples

**Benchmark 1: Computational — Fibonacci**

```pipe
bench "fibonacci(30)"
    fib: fn n
        if n <= 1
            n
        else
            (fib (n - 1)) + (fib (n - 2))
    fib 30
-- Typical: ~15ms per iteration (Tree-Walker), ~2ms (VM)
```

**Benchmark 2: Data Transform — List pipeline**

```pipe
    triple: fn x
        x * 3
    is_even: fn x
        x % 2 == 0
    half: fn x
        x / 2

bench "list pipeline 10k"
    data: range 0 10000
    result: data
        > map triple
        > filter is_even
        > map half
    len result
-- Typical: ~50ms per iteration (Tree-Walker), ~7ms (VM)
```

**Benchmark 3: I/O — File read and JSON parse**

```pipe
bench "read and parse JSON"
    raw: read_file "data.json"
    parsed: parse_json raw
    get_name: fn u
        u.name
    names: map parsed.users get_name
    len names
-- Typical: ~10ms per iteration (I/O dominated)
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
