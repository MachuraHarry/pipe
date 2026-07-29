# 12. Execution Models

Pipe features two distinct execution engines: the **Tree-Walker Interpreter** and the **Bytecode Virtual Machine**. Each offers a different balance of features, speed, and memory usage.

---

## 12.1 Two Execution Engines

### Tree-Walker Interpreter

The Tree-Walker is Pipe's default execution engine. It evaluates the AST (Abstract Syntax Tree) directly by recursively walking each node. This approach is straightforward and supports all language features.

**Internal flow:**

```
Source Code → Lexer → Parser → AST → Tree-Walker Evaluator → Output
```

1. **Lexer:** Tokenizes source text into tokens (identifiers, literals, operators, keywords).
2. **Parser:** Builds an AST from tokens using recursive descent parsing.
3. **Evaluator:** Walks the AST tree recursively, evaluating each node in real time.

The Tree-Walker is always available and requires no pre-processing. It interprets code one line at a time as it walks the syntax tree.

### Bytecode Virtual Machine

The Bytecode VM is a faster alternative that compiles the AST to bytecode before execution. It trades some language features for significantly improved performance.

**Internal flow:**

```
Source Code → Lexer → Parser → AST → Compiler → Bytecode → Stack-Based VM → Output
```

1. **Lexer/Parser:** Same as Tree-Walker, producing an AST.
2. **Compiler:** Translates the AST into a compact bytecode instruction stream and constant pool.
3. **VM:** A stack-based virtual machine executes the bytecode instructions directly.

The VM compiles the entire program before execution begins, enabling optimizations and faster runtime.

### Comparison Table

| Aspect | Tree-Walker | Bytecode VM |
|--------|-------------|-------------|
| **Command** | `pipe` (default) | `pipe -vm` |
| **Relative Speed** | 1x (baseline) | ~7x faster |
| **All Features** | Yes | No (subset) |
| **Bytecode Cache** | No | Yes (`.pipec`) |
| **Memory Usage** | Lower | Higher (bytecode + constants) |
| **Compile Time** | None | Negligible |
| **Startup** | Immediate | Compile then run |
| **Best For** | Development, debugging, full features | Production, CPU-bound tasks |

---

## 12.2 Tree-Walker: Detailed

### Advantages

1. **Full Feature Support:** Every Pipe feature works in the Tree-Walker — `map`/`filter`/`reduce` with user-defined functions, `for-in` loops, `go` goroutines, tail-call optimization, and more.

2. **Tail-Call Optimization (TCO):** Recursive functions in tail position are optimized to avoid stack overflow. The Tree-Walker detects when a function call is the last operation and reuses the current stack frame.

```pipe
let countdown = fn(n) {
    if n <= 0 {
        print "Done!"
    } {
        print n
        countdown (n - 1)     # tail call — optimized, no stack growth
    }
}

countdown 10000    # Works! No stack overflow thanks to TCO
```

3. **Higher-Order Functions with User Functions:** `map`, `filter`, and `reduce` accept user-defined functions:

```pipe
let double = fn(x) { x * 2 }
let result = map [1, 2, 3] double      # [2, 4, 6]

let is_even = fn(x) { x % 2 == 0 }
let evens = filter [1, 2, 3, 4] is_even  # [2, 4]
```

4. **`for-in` Loop Support:**

```pipe
for x in [1, 2, 3, 4, 5] {
    print x * x
}
```

5. **Concurrency (`go` and `>>`):** Launch parallel goroutines or parallel pipelines:
```pipe
go fn() {
    http_get "https://api.example.com/process"
}

data >> slow_ai_call    -- parallel pipeline via Future
```

6. **Better Error Messages:** Since the Tree-Walker operates directly on the AST, error messages include the exact source location and context.

### Disadvantages

- **Slower Execution:** Walking the AST recursively on every expression evaluation adds overhead. For compute-intensive programs, this overhead is noticeable — typically 5-10x slower than a compiled approach.
- **No Bytecode Cache:** Every run requires parsing from source. No persistent cache exists between invocations.
- **All-or-Nothing:** The entire AST must be kept in memory. Very large programs may consume significant memory during parsing.

### When to Use Tree-Walker

- Daily development and debugging
- Learning Pipe
- Running short scripts
- Using features not available in the VM
- Programs dominated by I/O (HTTP requests, file operations)

---

## 12.3 Bytecode VM: Detailed

### Advantages

1. **~7x Faster:** The VM executes pre-compiled bytecode instructions directly, without repeatedly walking the AST. Compilation to bytecode is a one-time cost per invocation, after which execution is fast.

2. **Bytecode Cache (`.pipec`):** On first run with `-vm`, Pipe compiles to bytecode and saves it as a `.pipec` file. Subsequent runs skip compilation entirely if the source hasn't changed:

```bash
pipe -vm large_app.pipe
# First run: compiles + runs (~200ms)
pipe -vm large_app.pipe
# Second run: loads cache + runs (~30ms)
```

3. **Lower Runtime Overhead:** The VM uses a simple instruction dispatch loop with a stack for operands, avoiding the recursive tree-traversal overhead of the Tree-Walker.

4. **Compact Representation:** Bytecode is more compact than an AST, reducing memory usage for the program structure. Large programs with many function definitions see significant memory savings.

### Disadvantages

1. **Feature Subset:** The following features are **not available** in the VM:

| Feature | VM Support |
|---------|-----------|
| `map` with user functions | No (built-in only) |
| `filter` with user functions | No (built-in only) |
| `reduce` with user functions | No (built-in only) |
| `for-in` loops | No |
| `go` goroutines | No |
| `>>` parallel pipeline | Yes (builtins only) |
| Tail-Call Optimization (TCO) | No |

2. **No Goroutines:** Concurrency via `go` is not implemented in the VM. However, the `>>` parallel pipeline operator works for builtins (AI calls, I/O) — user-defined closures fall back to synchronous execution.

3. **Limited Tail Calls:** Deep recursion without TCO can cause stack overflow in the VM.

### When to Use Bytecode VM

- Production deployments
- CPU-bound data processing
- Web servers and long-running services
- Programs that run frequently (benefits from cache)
- Any code that doesn't require VM-incompatible features

---

## 12.4 Decision Guide

| Use Case | Recommended Mode |
|----------|-----------------|
| Development/debugging | Tree-Walker |
| Quick one-off scripts | Either (Tree-Walker has no compile step) |
| CPU-heavy data processing | Bytecode VM |
| Web server / API | Bytecode VM |
| Scripts using goroutines | Tree-Walker |
| Scripts using `for-in` loops | Tree-Walker |
| Scripts using `map`/`filter`/`reduce` with user fns | Tree-Walker |
| Large codebases, frequent runs | Bytecode VM (caching benefits) |
| Learning Pipe | Tree-Walker |
| CI/CD pipelines | Bytecode VM (speed) |

---

## 12.5 Performance Comparison

Typical benchmark results comparing Tree-Walker vs Bytecode VM on standard workloads:

| Benchmark | Tree-Walker | Bytecode VM | Speedup |
|-----------|-------------|-------------|---------|
| Fibonacci(30) | 245ms | 35ms | 7.0x |
| Loop 1M iterations | 520ms | 75ms | 6.9x |
| List pipeline (10k elements) | 180ms | 25ms | 7.2x |
| String processing (100k ops) | 95ms | 13ms | 7.3x |
| JSON parse + iterate | 42ms | 8ms | 5.3x |
| Math operations (1M) | 310ms | 43ms | 7.2x |
| Recursive tree walk | 150ms | N/A | — (VM has no TCO) |
| HTTP GET (I/O bound) | 120ms | 115ms | ~1.0x (I/O dominated) |

**Key takeaway:** VM provides ~7x speedup for CPU-bound code. I/O-bound code (HTTP, file operations) sees minimal benefit since the I/O dominates execution time.

---

## 12.6 Enabling VM Mode

### Command Line

```bash
# Run a file with VM
pipe -vm my_program.pipe

# Run with both VM and quiet mode
pipe -vm -q my_program.pipe

# Build a self-extracting binary (always uses VM internally)
pipe -build my_program.pipe
```

### REPL

Switch modes in the REPL with the `:vm` command:

```pipe
# Start REPL in Tree-Walker mode (default)
pipe

>>> :vm
Switched to Bytecode VM mode.

>>> 2 + 2
4

>>> :vm
Switched to Tree-Walker mode.

>>> 2 + 2
4
```

Each `:vm` toggles between the two modes. The prompt indicates the current mode.

### With Bytecode Cache

When running with `-vm`, Pipe automatically creates a `.pipec` cache file:

```bash
pipe -vm large_app.pipe     # Compiles and creates large_app.pipec
pipe -vm large_app.pipe     # Uses cached bytecode (faster startup)
```

Delete the `.pipec` file to force a fresh compilation:

```bash
rm large_app.pipec
pipe -vm large_app.pipe     # Recompiles from source
```

---

## 12.7 Complete Feature Matrix

| Feature | Tree-Walker | Bytecode VM | Notes |
|---------|:-----------:|:-----------:|-------|
| Variables (`let`) | Yes | Yes | |
| Assignment (`=`) | Yes | Yes | |
| Arithmetic (`+`, `-`, `*`, `/`, `%`) | Yes | Yes | |
| Comparison (`==`, `!=`, `<`, `>`, `<=`, `>=`) | Yes | Yes | |
| Logical (`and`, `or`, `not`) | Yes | Yes | |
| String concatenation (`+`) | Yes | Yes | |
| If/else expressions | Yes | Yes | |
| Blocks `{ }` | Yes | Yes | |
| Functions (first-class) | Yes | Yes | |
| Closures | Yes | Yes | |
| Recursion | Yes | Yes | VM limited depth without TCO |
| Tail-Call Optimization (TCO) | Yes | No | VM may stack overflow on deep recursion |
| Pipeline operator (`>`) | Yes | Yes | |
| Parallel pipeline (`>>`) | Yes | Yes (builtins) | User closures: sync in VM |
| `try`/`catch` | Yes | Yes | |
| `return` | Yes | Yes | |
| Lists `[ ]` | Yes | Yes | |
| Maps `{ }` | Yes | Yes | |
| Dot-notation access | Yes | Yes | |
| `len` | Yes | Yes | |
| `push`, `pop`, `at` | Yes | Yes | |
| `slice_list` (`start..end`) | Yes | Yes | |
| `sort` | Yes | Yes | |
| `range` | Yes | Yes | |
| `map` with **built-in** functions | Yes | Yes | |
| `map` with **user** functions | Yes | **No** | VM limitation |
| `filter` with **built-in** functions | Yes | Yes | |
| `filter` with **user** functions | Yes | **No** | VM limitation |
| `reduce` with **built-in** functions | Yes | Yes | |
| `reduce` with **user** functions | Yes | **No** | VM limitation |
| `each` (all function types) | Yes | Yes | |
| `for-in` loop | Yes | **No** | Use `each` with range as alternative |
| `go` (goroutines) | Yes | **No** | Concurrency only in Tree-Walker |
| Modules (`import`/`export`) | Yes | Yes | |
| `PIPE_PATH` search | Yes | Yes | |
| Import caching | Yes | Yes | |
| Result type (`Ok`/`Err`/`unwrap`) | Yes | Yes | |
| All 81+ built-in functions | Yes | Yes | |
| `http_get`/`http_post`/`http_get_json` | Yes | Yes | |
| `parse_json`/`to_json` | Yes | Yes | |
| TCP functions | Yes | Yes | |
| Regex functions | Yes | Yes | |
| File system functions | Yes | Yes | |
| Bytecode cache (`.pipec`) | No | Yes | |
| Self-extracting binary (`-build`) | No | Yes (uses VM internally) | |

---

## 12.8 Migration Tips

### From Tree-Walker to VM

If you want to run an existing Tree-Walker program in VM mode, check for these incompatibilities:

1. **Replace `for-in` with `each` + `range`:**

```pipe
# Tree-Walker only:
for item in items {
    print item
}

# Compatible with both:
each items fn(item) { print item }

# Or with index:
each range 0 (len items) fn(i) {
    print at items i
}
```

2. **Replace user-function `map`/`filter`/`reduce` with inline loops:**

```pipe
# Tree-Walker only:
let doubled = map nums fn(x) { x * 2 }

# VM-compatible (manual loop approach):
let doubled = []
each nums fn(x) {
    push doubled (x * 2)
}
```

3. **Remove `go` calls** or guard them with a mode check:

```pipe
# Not available in VM — remove or use conditional execution
```

4. **Guard deep recursion** — if your Tree-Walker code uses deep recursion that relies on TCO, refactor to use iteration or `each` in VM mode.

### From VM to Tree-Walker

VM programs generally run without changes in Tree-Walker mode. All VM-compatible code is valid Tree-Walker code.
