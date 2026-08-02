# 9. Modules and Imports

Pipe supports a module system that allows code organization across multiple files. Modules promote reusability, encapsulation, and maintainability.

---

## 9.1 Import Syntax

The `import` statement loads and evaluates another Pipe file. The path is **relative to the file containing the import statement**.

```pipe
-- File: /home/user/project/main.pipe

-- imports from /home/user/project/lib/math.pipe
import "lib/math.pipe"
-- imports from /home/user/project/helpers.pipe
import "./helpers.pipe"
-- imports from /home/user/shared/common.pipe
import "../shared/common.pipe"
```

### 9.1.1 Direct Import

When you import a file without an alias, all exported symbols become available in the current scope:

```pipe
-- File: math.pipe
export PI: 3[14159]

export fn square x
    x * x

export fn cube x
    x * x * x
```

```pipe
-- File: main.pipe
import "math.pipe"

-- 3[14159]
print PI
-- 25
print (square 5)
-- 27
print (cube 3)
```

### 9.1.2 Namespace Import with `as`

To avoid name collisions and keep code organized, use the `as` keyword to create a namespace:

```pipe
-- File: main.pipe
import "math.pipe" as math
import "strings.pipe" as str

-- 3[14159]
print math.PI
-- 25
print (math.square 5)
-- "Hello"
print (str.capitalize "hello")
```

Using `as` is considered a **best practice** for non-trivial projects.

---

## 9.2 Import Caching

Pipe **caches module imports**. Each unique file path is parsed and evaluated only **once** per program execution. Subsequent imports of the same file return the cached namespace immediately.

```pipe
-- File: main.pipe
import "math.pipe" as math1
import "math.pipe" as math2

-- math1 and math2 reference the same module instance
-- The file math.pipe is only parsed/evaluated once
```

This caching behavior means:

- **Side effects run only once.** If a module has top-level `print` calls or state mutations, they execute only on the first import.
- **Shared state.** If a module exports a mutable value (like a list), all importers share the same reference.
- **Faster imports.** Repeated imports incur no additional parsing or evaluation cost.

```pipe
-- File: counter.pipe
count: 0

export fn increment
    count: count + 1
    count

export fn get_count
    count
```

```pipe
-- File: main.pipe
import "counter.pipe" as c1
import "counter.pipe" as c2

-- returns 1
c1.increment
-- returns 2  (same count!)
c2.increment
-- 2
c2.get_count
```

---

## 9.3 The `export` Keyword

By default, **all top-level definitions** in a module are visible to importers. However, if any `export` keyword is used in the file, then **only explicitly exported symbols** are visible.

```pipe
-- File: lib.pipe
-- not exported if any export exists
public_val: 42
-- exported
export visible: 100
-- not exported
hidden: "secret"

export fn greet name
    "Hello, " ++ name ++ "!"

-- not exported, private
internal_helper: fn x
    x * 2
```

```pipe
-- File: main.pipe
import "lib.pipe"

-- 100
print visible
-- "Hello, World!"
print (greet "World")

-- ERROR: not exported
print public_val
-- ERROR: not exported
print hidden
-- ERROR: not exported
print internal_helper
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
-- With PIPE_PATH set, you can import from search directories:
import "json-tools.pipe" as json
import "http-utils.pipe" as http
```

### 9.4.1 Resolution Order

When resolving an import path:

1. **Relative to the importing file's directory** — always checked first.
2. **Current working directory** — then the directory where pipe was invoked.
3. **Module cache** — `~/.pipe/modules/` for installed modules.
4. **Each directory in PIPE_PATH** — searched in order, first match wins.
5. **Module registry** — if no local file found, queries the Pipe module registry.

---

## 9.5 Remote Modules (URL Imports)

Pipe can import modules directly from URLs:

```pipe
import "https://raw.githubusercontent.com/user/repo/main/module.pipe"
import "https://example.com/lib/utils.pipe" as utils
```

URL imports are downloaded on first use and cached in `~/.pipe/modules/`. Subsequent imports use the cached version.

---

## 9.6 Module Registry and Versions

### 9.6.1 The Registry

Pipe has a central module registry at `github.com/MachuraHarry/pipe-modules`. Modules are discoverable via `pipe -search`:

```bash
pipe -search log        # find log-related modules
pipe -search ai         # find AI modules
```

### 9.6.2 Installing Modules

```bash
pipe -get log-analyzer              # install latest version
pipe -get log-analyzer@1.0.0        # install specific version
pipe -get https://.../module.pipe   # install from URL
```

Installed modules are stored in `~/.pipe/modules/` and available for import by name.

### 9.6.3 Versioned Imports

Use `@version` to pin a module to a specific release:

```pipe
-- exact version
import "log-analyzer@1.0[0]"
-- latest (implicit @latest)
import "log-analyzer"
-- version with alias
import "sentiment@0.9[0]" as s
```

The versioned import first checks the local module cache for the exact `name@version` file. If not found, it queries the registry for the version URL, downloads the module, and caches it.

### 9.6.4 How Versions Work

Each registry entry can define multiple versions:

```json
{
  "log-analyzer": {
    "description": "Log classification",
    "functions": ["log_analyze", "log_summarize"],
    "latest": "1.0.0",
    "versions": {
      "1.0.0": "https://raw.../v1.0.0/module.pipe",
      "0.9.0": "https://raw.../v0.9.0/module.pipe"
    }
  }
}
```

- `latest` points to the recommended version
- `versions` maps version tags to module URLs
- If no `@version` is specified, the `latest` version is used
- If the registry uses the legacy `url` field (no `versions`), that URL is used as default

---

## 9.7 Recommended Module Structure

### 9.7.1 Small Project

```
my-project/
  main.pipe              # entry point
  utils.pipe             # utility functions
  models.pipe            # data structures
```

### 9.7.2 Medium Project

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

### 9.7.3 Large Project

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

## 9.8 Best Practices

### Export Only What's Needed

Minimize the public API surface of each module. Internal helpers should not be exported.

```pipe
-- Good: explicit exports
export fn public_api x
    internal x * 2

-- Not exported, private to module
internal: fn x
    x + 10
```

### Use Namespaces to Prevent Collisions

Always use `as` for non-trivial programs:

```pipe
-- Good
import "math.pipe" as math
import "stats.pipe" as stats

result: stats.mean (math.square 5)

-- Avoid (unless small script)
-- All symbols dumped into global scope
import "math.pipe"
```

### Group Imports at the Top

Place all imports at the top of the file for clarity:

```pipe
import "config.pipe" as config
import "lib/math.pipe" as math
import "lib/strings.pipe" as str
import "models/user.pipe" as user_model

-- ... rest of the file
```

### Use Relative Paths, Avoid Absolute Paths

Relative paths keep your project portable across environments:

```pipe
-- Good: portable
import "./lib/helpers.pipe"
import "../shared/utils.pipe"

-- Avoid: not portable
-- Windows breaks
import "/home/user/project/lib/helpers.pipe"
-- Linux breaks
import "C:\\Users\\user\\project\\lib\\helpers.pipe"
```

### Handle Circular Imports Carefully

Circular imports (A imports B, B imports A) are technically possible due to caching but should be avoided. They can cause confusing initialization order issues:

```pipe
-- File: a.pipe
import "b.pipe" as b
-- b.y might be nil if b hasn't finished
export x: b.y + 1
-- safer: defer evaluation
export fn get_x
    b.y + 1
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
