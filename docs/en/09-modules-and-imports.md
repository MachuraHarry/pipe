# 9. Modules and Imports

Pipe supports a module system that allows code organization across multiple files. Modules promote reusability, encapsulation, and maintainability.

---

## 9.1 Import Syntax

The `import` statement loads and evaluates another Pipe file. The path is **relative to the file containing the import statement**.

```pipe
# File: /home/user/project/main.pipe

import "lib/math.pipe"        # imports from /home/user/project/lib/math.pipe
import "./helpers.pipe"       # imports from /home/user/project/helpers.pipe
import "../shared/common.pipe" # imports from /home/user/shared/common.pipe
```

### 9.1.1 Direct Import

When you import a file without an alias, all exported symbols become available in the current scope:

```pipe
# File: math.pipe
export let PI = 3.14159

export let square = fn(x) {
    x * x
}

export let cube = fn(x) {
    x * x * x
}
```

```pipe
# File: main.pipe
import "math.pipe"

print PI            # 3.14159
print (square 5)    # 25
print (cube 3)      # 27
```

### 9.1.2 Namespace Import with `as`

To avoid name collisions and keep code organized, use the `as` keyword to create a namespace:

```pipe
# File: main.pipe
import "math.pipe" as math
import "strings.pipe" as str

print math.PI               # 3.14159
print (math.square 5)       # 25
print (str.capitalize "hello")  # "Hello"
```

Using `as` is considered a **best practice** for non-trivial projects.

---

## 9.2 Import Caching

Pipe **caches module imports**. Each unique file path is parsed and evaluated only **once** per program execution. Subsequent imports of the same file return the cached namespace immediately.

```pipe
# File: main.pipe
import "math.pipe" as math1
import "math.pipe" as math2

# math1 and math2 reference the same module instance
# The file math.pipe is only parsed/evaluated once
```

This caching behavior means:

- **Side effects run only once.** If a module has top-level `print` calls or state mutations, they execute only on the first import.
- **Shared state.** If a module exports a mutable value (like a list), all importers share the same reference.
- **Faster imports.** Repeated imports incur no additional parsing or evaluation cost.

```pipe
# File: counter.pipe
let count = 0

export let increment = fn() {
    count = count + 1
    count
}

export let get_count = fn() { count }
```

```pipe
# File: main.pipe
import "counter.pipe" as c1
import "counter.pipe" as c2

c1.increment()     # returns 1
c2.increment()     # returns 2  (same count!)
c2.get_count()     # 2
```

---

## 9.3 The `export` Keyword

By default, **all top-level definitions** in a module are visible to importers. However, if any `export` keyword is used in the file, then **only explicitly exported symbols** are visible.

```pipe
# File: lib.pipe
let public_val = 42        # not exported if any export exists
export let visible = 100   # exported
let hidden = "secret"      # not exported

export let greet = fn(name) {
    "Hello, " + name + "!"
}

let internal_helper = fn(x) {   # not exported, private
    x * 2
}
```

```pipe
# File: main.pipe
import "lib.pipe"

print visible       # 100
print (greet "World")  # "Hello, World!"

print public_val    # ERROR: not exported
print hidden        # ERROR: not exported
print internal_helper  # ERROR: not exported
```

**Rule:** If there are one or more `export` declarations in a module, only those marked with `export` are accessible by importers. If there are **zero** `export` declarations, all top-level symbols are accessible (open module).

---

## 9.4 `PIPE_PATH` Environment Variable

Pipe supports a search path for modules via the `PIPE_PATH` environment variable. When an import path does not resolve relative to the importing file, Pipe searches each directory in `PIPE_PATH`.

```bash
# Unix/Linux/macOS
export PIPE_PATH="/home/user/pipe-libs:/usr/local/share/pipe"

# Windows (PowerShell)
$env:PIPE_PATH = "C:\pipe-libs;C:\Program Files\Pipe\lib"

# Windows (CMD)
set PIPE_PATH=C:\pipe-libs;C:\Program Files\Pipe\lib
```

```pipe
# With PIPE_PATH set, you can import from search directories:
import "json-tools.pipe" as json
import "http-utils.pipe" as http
```

### 9.4.1 Resolution Order

When resolving an import path:

1. **Relative to the importing file's directory** — always checked first.
2. **Each directory in PIPE_PATH** — searched in order, first match wins.
3. **Not found** — runtime error.

---

## 9.5 Recommended Module Structure

### 9.5.1 Small Project

```
my-project/
  main.pipe              # entry point
  utils.pipe             # utility functions
  models.pipe            # data structures
```

### 9.5.2 Medium Project

```
my-project/
  main.pipe
  lib/
    math.pipe            # math utilities
    strings.pipe         # string helpers
    http.pipe            # HTTP client wrappers
  models/
    user.pipe            # user-related functions
    post.pipe            # post-related functions
```

### 9.5.3 Large Project

```
my-app/
  main.pipe
  cmd/
    server.pipe          # server entry point
    worker.pipe          # worker entry point
  internal/
    config.pipe          # configuration loading
    db.pipe              # database helpers
    logging.pipe         # logging utilities
  pkg/
    api/
      handler.pipe       # HTTP handlers
      middleware.pipe    # middleware
    models/
      user.pipe
      order.pipe
  test/
    test_helpers.pipe
```

---

## 9.6 Best Practices

### Export Only What's Needed

Minimize the public API surface of each module. Internal helpers should not be exported.

```pipe
# Good: explicit exports
export let public_api = fn(x) { internal(x) * 2 }

let internal = fn(x) {      # Not exported, private to module
    x + 10
}
```

### Use Namespaces to Prevent Collisions

Always use `as` for non-trivial programs:

```pipe
# Good
import "math.pipe" as math
import "stats.pipe" as stats

let result = stats.mean (math.square 5)

# Avoid (unless small script)
import "math.pipe"     # All symbols dumped into global scope
```

### Group Imports at the Top

Place all imports at the top of the file for clarity:

```pipe
import "config.pipe" as config
import "lib/math.pipe" as math
import "lib/strings.pipe" as str
import "models/user.pipe" as user_model

# ... rest of the file
```

### Use Relative Paths, Avoid Absolute Paths

Relative paths keep your project portable across environments:

```pipe
# Good: portable
import "./lib/helpers.pipe"
import "../shared/utils.pipe"

# Avoid: not portable
import "/home/user/project/lib/helpers.pipe"     # Windows breaks
import "C:\\Users\\user\\project\\lib\\helpers.pipe"  # Linux breaks
```

### Handle Circular Imports Carefully

Circular imports (A imports B, B imports A) are technically possible due to caching but should be avoided. They can cause confusing initialization order issues:

```pipe
# File: a.pipe
import "b.pipe" as b
export let x = b.y + 1       # b.y might be nil if b hasn't finished
export let get_x = fn() { b.y + 1 }  # safer: defer evaluation
```

### Keep Modules Focused

Each module should have a single, clear responsibility:

```
# Good module names reflect purpose:
- math.pipe      (mathematical functions)
- http.pipe      (HTTP operations)
- config.pipe    (configuration)
- csv.pipe       (CSV parsing/writing)

# Avoid vague containers:
- utils.pipe     (too broad — split into focused modules)
- helpers.pipe   (what kind of helpers?)
```
