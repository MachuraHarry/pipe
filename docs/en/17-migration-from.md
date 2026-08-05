# Migration from Other Languages

This guide helps developers familiar with Python, Lua, Bash, or JavaScript understand how Pipe compares and when to use each language.

## Pipe vs Python

### Syntax Comparison

| Feature | Python | Pipe |
|---------|--------|------|
| Comments | `# comment` | `-- comment` |
| Variables | `x = 42` | `x: 42` |
| Constants | Convention only | No `const` keyword |
| Functions | `def name(a, b):` | `fn name a b` |
| Anonymous functions | `lambda x: x * 2` | `double: fn x`<br>`    x * 2` |
| Blocks | Colons + indentation | Colons + indentation |
| If/else | `if x: ... else: ...` | `if x: ... else: ...` |
| Loops | `for i in range(10):` | `for i in range 10:` |
| While | `while cond:` | `while cond:` |
| Return | `return value` | `return value` |
| Lists | `[1, 2, 3]` | `[1, 2, 3]` |
| Dicts/Maps | `{"key": val}` | `{key: val}` |
| String concat | `"a" + "b"` | `"a" ++ "b"` |
| Logical and/or | `and`, `or` | `&&`, `\|\|` |
| Logical not | `not x` | `!x` |
| Equality | `==` | `==` |
| None/null | `None` | `nil` |
| Booleans | `True`, `False` | `true`, `false` |
| Import | `import module` | `import "module.pipe"` |
| Export | Convention (`__all__`) | `export fn name` |
| Error handling | `try: except e:` | `try: catch e:` |
| Pipeline | None (method chaining) | `x > f > g` |
| Pattern matching | `match x: case y:` (3.10+) | `match x \| val -> ...` |
| Classes/OOP | Yes | No |
| Decorators | `@decorator` | No |
| Comprehensions | `[x*2 for x in lst]` | `map lst double` |
| Generators | `yield` | No |
| Async/await | `async/await` | No |
| Type hints | Yes | No (planned) |
| Package manager | `pip` | None (single binary) |
| Binary size | ~30MB+ (interpreter) | ~16MB (statically linked) |

### Side-by-Side: Fibonacci

**Python**:
```python
def fib(n):
    if n <= 1:
        return n
    return fib(n - 1) + fib(n - 2)

print(fib(10))
```

**Pipe**:
```pipe
fib: fn n
  if n <= 1
    n
  else
    (fib n - 1) + (fib n - 2)

print fib 10
```

### When to Use Pipe vs Python

**Use Pipe when**:
- You need a single, self-contained binary with no runtime dependencies
- You're writing scripts for system administration or automation
- You want pipeline-centric data processing with `>`
- You need a small footprint (~16MB vs Python's ~30MB+)
- You're embedding a scripting language in a Go application
- You want instant startup time for CLI tools

**Use Python when**:
- You need a vast ecosystem of third-party packages
- You're building web applications (Django, Flask, FastAPI)
- You need data science / ML libraries (NumPy, Pandas, PyTorch)
- You require classes, inheritance, and OOP patterns
- You need async/await for concurrent I/O
- You're working in an existing Python codebase

## Pipe vs Lua

### Syntax Comparison

| Feature | Lua | Pipe |
|---------|-----|------|
| Comments | `-- comment` | `-- comment` |
| Block comments | `--[[ ... ]]` | Not supported |
| Variables | `local x = 42` | `x: 42` (block-scoped) |
| Global variables | Default scope | Top-level only |
| Functions | `function name(a, b)` | `fn name a b` |
| End blocks | `end` keyword | Indentation (DEDENT) |
| If/else | `if x then ... else ... end` | `if x: ... else: ...` |
| Loops | `for i=1,10 do ... end` | `for i in range 1 11:` |
| While | `while cond do ... end` | `while cond:` |
| Tables | `{key = val}` | `{key: val}` |
| String concat | `"a" .. "b"` | `"a" ++ "b"` |
| Length | `#t` | `len t` |
| Nil | `nil` | `nil` |
| Booleans | `true`, `false` | `true`, `false` |
| Multiple returns | `return a, b` | Single return value |
| Require | `require("module")` | `import "module.pipe"` |
| Error handling | `pcall(fn)` | `try: catch e:` |
| Metatables | Yes | No |
| Coroutines | Yes | No (planned) |
| Binary size | ~300KB | ~16MB |

### Side-by-Side: Fibonacci

**Lua**:
```lua
function fib(n)
    if n <= 1 then
        return n
    end
    return fib(n - 1) + fib(n - 2)
end

print(fib(10))
```

**Pipe**:
```pipe
fib: fn n
  if n <= 1
    n
  else
    (fib n - 1) + (fib n - 2)

print fib 10
```

### When to Use Pipe vs Lua

**Use Pipe when**:
- You want indentation-based blocks instead of `end` keywords
- You need a richer standard library (86 builtins vs Lua's minimal stdlib)
- You're doing file I/O, HTTP, JSON, TCP, or regex without external packages
- You want pipeline syntax for data transformation chains

**Use Lua when**:
- You need an extremely small interpreter (~300KB)
- You're embedding in a C/C++ application (LuaJIT's FFI is excellent)
- You need coroutines for cooperative multitasking
- You're scripting game engines (Lua is the de facto standard)
- You need metatables for custom type behavior

## Pipe vs Bash

### Syntax Comparison

| Feature | Bash | Pipe |
|---------|------|------|
| Comments | `# comment` | `-- comment` |
| Variables | `x=42` (no spaces) | `x: 42` |
| Variable access | `$x` or `"${x}"` | `x` (direct) |
| Functions | `name() { ... }` | `fn name` |
| If/else | `if [[ cond ]]; then ... fi` | `if cond: ... else: ...` |
| Loops | `for i in {1..10}; do ... done` | `for i in range 1 11:` |
| Arrays | `arr=(a b c)` | `["a", "b", "c"]` |
| String concat | `"$a$b"` | `a ++ b` |
| Command exec | `$(cmd)` or backticks | `exec "cmd"` |
| Pipe operator | `\|` (between processes) | `>` (between functions) |
| Exit codes | `$?` | `result.status` |
| Subshells | `( ... )` | Not supported |
| Arithmetic | `$(( a + b ))` | `a + b` (native) |
| Error handling | `set -e`, `trap` | `try: catch e:` |
| String manipulation | `${var#prefix}` | `split`, `regex_replace` |
| JSON | Needs `jq` | Built-in `parse_json`/`to_json` |
| HTTP | Needs `curl`/`wget` | Built-in `http_get`/`http_post` |
| Binary size | ~1MB (bash itself) | ~16MB |

### When to Use Pipe vs Bash

**Use Pipe when**:
- You need data structures (lists, maps) beyond flat strings
- You need built-in JSON, HTTP, regex, and TCP without external tools
- You want readable control flow (`if`, `while`, `for` instead of `[[ ]]`, `do...done`, `fi`)
- You're doing complex string processing or data transformation
- You need cross-platform scripts (Bash varies between macOS and Linux)
- You want proper error handling with stack traces

**Use Bash when**:
- You're gluing together command-line tools with pipes (`|`)
- You need to run on systems where only Bash is available
- Your script is short (<50 lines) and primarily calls other programs
- You're writing `.bashrc` or `.profile` configuration
- You need process substitution (`<(...)`) and here-documents

## Pipe vs JavaScript/Node.js

### Syntax Comparison

| Feature | JavaScript | Pipe |
|---------|------------|------|
| Comments | `// comment`, `/* */` | `-- comment` |
| Variables | `let x = 42`, `const x = 42` | `x: 42` |
| Functions | `function name(a, b) {}` | `fn name a b` |
| Arrow functions | `(x) => x * 2` | `double: fn x`<br>`    x * 2` |
| Blocks | `{ ... }` | Indentation |
| If/else | `if (x) { ... } else { ... }` | `if x: ... else: ...` |
| Loops | `for (let i=0; i<10; i++)` | `for i in range 10:` |
| Array methods | `arr.map(x => x*2)` | `map arr double` |
| Objects | `{key: val}` | `{key: val}` |
| Template strings | `` `hello ${name}` `` | `"hello " ++ name` |
| String concat | `"a" + "b"` | `"a" ++ "b"` |
| Equality | `===` | `==` |
| Null/undefined | `null`, `undefined` | `nil` |
| Booleans | `true`, `false` | `true`, `false` |
| Import | `import ... from "..."` | `import "file.pipe"` |
| Export | `export function` | `export fn name` |
| Try/catch | `try { ... } catch(e) { ... }` | `try: catch e:` |
| Async | `async/await`, Promises | Not supported |
| Classes | `class X {}` | No |
| Arrow pipeline | `x \|> f \|> g` (proposal) | `x > f > g` (native) |
| Runtime | Node.js (~60MB+) | ~16MB binary |
| Package manager | npm (~1M+ packages) | None |

### When to Use Pipe vs Node.js

**Use Pipe when**:
- You need a single, dependency-free binary for distribution
- You're writing CLI tools or system scripts
- You want built-in file I/O, HTTP, JSON, and regex without `require()`
- You need instant startup (no V8 warm-up)
- You prefer indentation-based syntax over braces and semicolons
- You're scripting on resource-constrained systems

**Use Node.js when**:
- You're building web servers and REST APIs (Express, Fastify, etc.)
- You need the npm ecosystem for third-party packages
- You need async I/O for high-concurrency scenarios
- You're building frontend tooling (Webpack, Vite, etc.)
- You need to share code between frontend and backend
- You need TypeScript for type safety

## Summary Comparison

| Dimension | Pipe | Python | Lua | Bash | Node.js |
|-----------|------|--------|-----|------|---------|
| **Simplicity** | ★★★★★ | ★★★★☆ | ★★★★☆ | ★★★☆☆ | ★★★☆☆ |
| **Expressiveness** | ★★★★☆ | ★★★★★ | ★★★☆☆ | ★★☆☆☆ | ★★★★★ |
| **Built-in stdlib** | ★★★★☆ | ★★★★★ | ★★☆☆☆ | ★★☆☆☆ | ★★★☆☆ |
| **Startup performance** | ★★★★☆ | ★★★☆☆ | ★★★★★ | ★★★★★ | ★★☆☆☆ |
| **Runtime performance** | ★★★☆☆ | ★★★☆☆ | ★★★★★ | ★★☆☆☆ | ★★★★★ |
| **Portability** | ★★★★★ | ★★★★☆ | ★★★★★ | ★★☆☆☆ | ★★★★☆ |
| **Ecosystem** | ★☆☆☆☆ | ★★★★★ | ★★☆☆☆ | ★★★★☆ | ★★★★★ |
| **Binary size** | ~16MB | ~30MB+ | ~300KB | ~1MB | ~60MB+ |
| **Data structures** | Lists, Maps | Lists, Dicts, Sets, Tuples | Tables only | Strings only | Arrays, Objects, Maps, Sets |
| **Pipeline operator** | Native `>` | No (method chains) | No | Process pipe `\|` | Proposal `\|>` |
| **Error handling** | try/catch + Result | try/except | pcall | trap / exit codes | try/catch |
| **Package manager** | None | pip | LuaRocks | apt/brew | npm |
| **IDE support** | VSCode (syntax) | Excellent | Good | Basic | Excellent |
