# 8. Error Handling

Pipe provides structured error handling through `try`/`catch` blocks, a `Result` type for explicit error management, and stack traces for debugging.

---

## 8.1 `try`/`catch` Syntax

The `try`/`catch` construct catches runtime errors, preventing program termination and allowing graceful recovery.

```pipe
try {
    # code that may error
} catch err {
    # handle the error
    # err is the error message string
}
```

### 8.1.1 Basic Example

```pipe
try {
    let result = 10 / 0
    print "This line is never reached"
} catch err {
    print "Caught error: " + err
}
# Output: Caught error: division by zero

print "Program continues..."
```

### 8.1.2 Error Flow Semantics

When an error occurs inside a `try` block:

1. **The try block aborts immediately** — no further statements in the try block execute.
2. **The error is bound to the catch parameter** — the variable after `catch` (commonly `err`) receives the error message string.
3. **The catch block executes** — your error-handling code runs.
4. **Program continues after the catch block** — execution resumes normally.

If **no error** occurs in the try block, the catch block is **skipped entirely**.

```pipe
try {
    print "Start try"
    let x = 100 / 5         # no error
    print "End try"
} catch err {
    print "This never runs"
}
print "Continues normally"
# Output:
# Start try
# End try
# Continues normally
```

### 8.1.3 Nested `try`/`catch`

Errors in a catch block are **not** caught by that same catch block. You can nest them if needed:

```pipe
try {
    try {
        10 / 0
    } catch err {
        print "Inner catch: " + err
        # Simulate another error in the catch block
        let x = nil
        print x.field      # This will crash unless caught externally
    }
} catch err {
    print "Outer catch: " + err
}
```

### 8.1.4 Function That Forces Errors

You can deliberately raise errors using division by zero, accessing nil fields, or calling built-in functions with invalid arguments.

```pipe
# Helper function to force an error
let fail = fn(msg) {
    print msg
    nil.field     # accessing field on nil always errors
}

try {
    fail "Something went wrong!"
} catch err {
    print "Caught: " + err
}
```

Common operations that produce errors:

| Operation | Error Description |
|-----------|-------------------|
| Division by zero | `division by zero` |
| Accessing field on nil | Index or field access on nil value |
| Invalid type for operation | Type mismatch errors |
| File operations on missing files | Filesystem errors |

### 8.1.5 `try_ai` — AI-Powered Self-Healing

`try_ai` extends `try`/`catch` with automatic AI-driven error recovery. When a fixable error occurs inside `try_ai`, Pipe calls the configured AI provider to analyze and fix the expression — without developer intervention.

```pipe
ai_provider "deepseek"

try_ai
    "42" * 3           -- Type Error: STRING * INTEGER
catch e
    0                   -- fallback if AI cannot fix
```

#### How It Works

1. **Error occurs** inside `try_ai` block
2. **Error code checked** — only E002 (type), E003 (div/0), E006 (index) are AI-fixable
3. **AI called** with error context and expression source
4. **3-ring validation** — parse check → sandbox test → type check
5. **Fix applied** in real environment, or **falls to catch** if unfixable

#### Fixable vs Unfixable

| Error | Code | AI Strategy | Example |
|-------|------|-------------|---------|
| Type mismatch | E002 | `to_num`, `to_str` wrapping | `"42" * 3` → `(to_num "42") * 3` |
| Division by zero | E003 | Guard: `max(x, 1)` or if-expression | `100 / 0` → `100 / (max 0 1)` |
| Index on wrong type | E006 | Fallback via `get` with default | `map.key` when not a map |
| Undefined variable | E001 | **Not fixable** — skips AI | Falls directly to catch |
| Not callable | E004 | **Not fixable** — skips AI | Falls directly to catch |

#### Execution Modes

| Mode | `try_ai` Behavior |
|------|------------------|
| Tree-Walker (`./bin/pipe`) | Full AI self-healing with 3-ring validation |
| Bytecode VM (`./bin/pipe -vm`) | Falls back to basic `try`/`catch` (no AI) |

---

## 8.2 Stack Traces

When an error occurs, Pipe automatically generates a **stack trace** showing the call path that led to the error.

```pipe
let deep_fn = fn() {
    10 / 0
}

let middle_fn = fn() {
    deep_fn()
}

try {
    middle_fn()
} catch err {
    print "Error: " + err
}
# Error includes stack trace showing:
#   deep_fn (line 2)
#   middle_fn (line 6)
#   <main> (line 10)
```

The stack trace is included in the error message string, making it easy to identify where errors originated.

---

## 8.3 `return` for Early Function Exit

The `return` keyword exits a function immediately, optionally providing a value. It does not trigger `try`/`catch` — it is normal control flow.

```pipe
let find_first_even = fn(nums) {
    each nums fn(n) {
        if n % 2 == 0 {
            return n       # early exit from find_first_even
        }
    }
    return nil
}

let nums = [1, 3, 5, 6, 7, 8]
let result = find_first_even nums
print result     # 6
```

`return` always exits the **current function scope**, not just the innermost block:

```pipe
let outer = fn() {
    let inner = fn() {
        return 42       # exits inner, not outer
    }
    let val = inner()
    print "Got: " + val  # "Got: 42"
    return 99
}
print outer()            # 99
```

---

## 8.4 The `Result` Type

For cases where exceptions are undesirable, Pipe provides a `Result` type — an explicit way to represent success or failure.

### 8.4.1 Creating Results

```pipe
let ok_result = Ok 42           # Wraps a value in success
let err_result = Err "failed"   # Wraps an error message
```

- `Ok(value)` — represents a successful operation
- `Err(message)` — represents a failed operation with a reason string

### 8.4.2 Checking Results

```pipe
is_ok (Ok 42)       # true
is_ok (Err "oops")  # false
is_err (Ok 42)      # false
is_err (Err "oops") # true
```

### 8.4.3 Unwrapping Results

```pipe
unwrap (Ok 42)          # 42
unwrap (Err "oops")     # ERROR: called unwrap on an Err value

unwrap_or (Ok 42) 0     # 42
unwrap_or (Err "x") 0   # 0     (uses default)
```

- `unwrap(result)` — returns value if Ok, **panics** if Err
- `unwrap_or(result, default)` — returns value if Ok, returns `default` if Err

### 8.4.4 Using Result in Functions

```pipe
let safe_divide = fn(a, b) {
    if b == 0 {
        return Err "division by zero"
    }
    Ok (a / b)
}

let r1 = safe_divide 10 2
if is_ok r1 {
    print "Result: " + (unwrap r1)     # "Result: 5"
}

let r2 = safe_divide 10 0
if is_err r2 {
    print "Failed: " + (unwrap r2)     # print the error message
}

# Using unwrap_or for defaults
let v1 = unwrap_or (safe_divide 10 2) -1    # 5
let v2 = unwrap_or (safe_divide 10 0) -1    # -1
```

### 8.4.5 Result in Pipelines

Results integrate naturally with Pipe's pipeline operator:

```pipe
let process = fn(data) {
    let parsed = parse_json data
    if parsed == nil {
        return Err "invalid JSON"
    }
    Ok (parsed.value * 2)
}

let r = process `{"value": 21}`
    | unwrap      # 42 (or error if parsing failed)

# Safe pipeline with default
let safe_r = process `{}`
    | fn(r) { unwrap_or r 0 }

print safe_r     # 0  (default, since "value" key is missing)
```

---

## 8.5 Error Avoidance Patterns

### 8.5.1 Nil Checks

Before accessing properties on a potentially nil value, check first:

```pipe
# Defensive
let process_user = fn(user) {
    if user == nil {
        return Err "no user provided"
    }
    if user.name == nil {
        return Err "user has no name"
    }
    Ok (upper user.name)
}

# Using unwrap_or for safe defaults
let name = unwrap_or (process_user nil) "Anonymous"
print name    # "Anonymous"
```

### 8.5.2 File Existence Checks

Always check file existence before operations that might fail:

```pipe
let safe_read = fn(path) {
    if !file_exists path {
        return Err "file not found: " + path
    }
    Ok (read_file path)
}

let content = unwrap_or (safe_read "config.json") "{}"
let config = parse_json content
print config
```

### 8.5.3 Type Checking Before Operations

```pipe
let safe_add = fn(a, b) {
    if !is_num a or !is_num b {
        return Err "both arguments must be numbers"
    }
    Ok (a + b)
}

safe_add 10 20      # Ok 30
safe_add 10 "x"     # Err "both arguments must be numbers"
```

### 8.5.4 Combining `try`/`catch` with `Result`

For external operations that might fail, wrap with `try`/`catch` and convert to `Result`:

```pipe
let read_config = fn(path) {
    try {
        let raw = read_file path
        let parsed = parse_json raw
        if parsed == nil {
            return Err "invalid JSON"
        }
        Ok parsed
    } catch err {
        Err ("read_config failed: " + err)
    }
}
```
