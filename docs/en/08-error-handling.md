# 8. Error Handling

Pipe provides structured error handling through `try`/`catch` blocks, a `Result` type for explicit error management, and stack traces for debugging.

---

## 8.1 `try`/`catch` Syntax

The `try`/`catch` construct catches runtime errors, preventing program termination and allowing graceful recovery.

```pipe
try
    -- code that may error
    risky_operation
catch err
    -- handle the error
    -- err is the error message string
    print err
```

### 8.1.1 Basic Example

```pipe
try
    result: 10 / 0
    print "This line is never reached"
catch err
    print "Caught error: " ++ err
-- Output: Caught error: division by zero

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
try
    print "Start try"
    -- no error
    x: 100 / 5
    print "End try"
catch err
    print "This never runs"
print "Continues normally"
-- Output:
-- Start try
-- End try
-- Continues normally
```

### 8.1.3 Nested `try`/`catch`

Errors in a catch block are **not** caught by that same catch block. You can nest them if needed:

```pipe
try
    try
        10 / 0
    catch err
        print "Inner catch: " ++ err
        -- Simulate another error in the catch block
        x: nil
        -- This will crash unless caught externally
        print x.field
catch err
    print "Outer catch: " ++ err
```

### 8.1.4 Function That Forces Errors

You can deliberately raise errors using division by zero, accessing nil fields, or calling built-in functions with invalid arguments.

```pipe
-- Helper function to force an error
fail: fn msg
    print msg
    -- accessing field on nil always errors
    nil.field

try
    fail "Something went wrong!"
catch err
    print "Caught: " ++ err
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
        "42" * 3
catch e
        0
```

**`catch` is optional** — if omitted, an unfixable error propagates normally:

```pipe
try_ai
    "42" * 3
-- result: 126 (AI fixed type error silently)
```

#### How It Works

1. **Error occurs** inside `try_ai` block
2. **Error code checked** — E001-E006 are all AI-fixable
3. **AI called** with error context and expression source
4. **Up to 3 retry attempts** — if the AI fix fails or produces another error, it tries again with additional context
5. **Feedback on stderr** — each attempt and its result are printed: `⚡ try_ai: attempt 1 — "42" * 3 → "(to_num "42") * 3"`
6. **3-ring validation** — parse check → isolated eval test → real evaluation
7. **Fix applied** in real environment, or **falls to catch** if unfixable

#### Fixable Error Codes

| Error | Code | AI Strategy | Example |
|-------|------|-------------|---------|
| Undefined variable | E001 | Use literal default | `x + 5` → `0 + 5` |
| Type mismatch | E002 | `to_num`, `to_str` wrapping | `"42" * 3` → `(to_num "42") * 3` |
| Division by zero | E003 | Guard: `max(x, 1)` or if-expression | `100 / 0` → `100 / (max 0 1)` |
| Not a function | E004 | Wrap in parens or use builtin | `42(x)` → `42` |
| Unsupported operator | E005 | Convert operand type | `"hi" - 1` → type fix |
| Invalid index | E006 | Guard with `len` or use `get` | `list[99]` → guarded access |

#### Execution Modes

| Mode | `try_ai` Behavior |
|------|------------------|
| Tree-Walker (`./bin/pipe`) | Full AI self-healing with retry + validation |
| Bytecode VM (`./bin/pipe -vm`) | Full AI self-healing with retry + validation (via Tree-Walker bridge) |

#### Output Example

```
⚡ try_ai: attempt 1 — "42" * 3 → "( (to_num "42") * 3 )"
✓ try_ai: fixed!
126
```

### 8.1.6 Security Analysis — Is `try_ai` Safe?

This section addresses the concern that AI-generated code modification at runtime is inherently dangerous. We analyze each risk vector and the defense mechanisms in place.

#### Threat Model

| Threat | Risk | Mitigation |
|--------|------|------------|
| AI generates malicious code | **HIGH** | 3-ring validation (parse → isolated eval → real) prevents execution of anything that doesn't parse, errors in isolation, or produces unexpected types |
| AI hallucinates a wrong fix | **MEDIUM** | Retry mechanism (up to 3 attempts); `catch` block provides deterministic fallback |
| Prompt injection via error message | **MEDIUM** | System prompt is **fixed and immutable** — the attacker's input goes into the `user` message only, separated from system instructions by the chat API boundary |
| Side effects in fixed code | **HIGH** | Variable-isolated evaluation (`env.Copy()`) and **real sandbox activation** in Ring 2: `FSNone`, `Network: false`, `Exec: false` blocks all I/O during fix validation. AI allowed for nested builtins. |
| AI fix introduces performance degradation | **LOW** | Fixes are local expression rewrites (max ~1 token change); no structural code generation |
| API latency makes program unpredictable | **LOW** | `catch` block guarantees deterministic fallback; retry limit of 3 bounds worst-case latency |
| Supply chain risk via AI provider | **LOW** | `try_ai` respects `ai_provider` configuration; can use local Ollama for zero-network self-healing |

#### Defense in Depth — The 3-Ring Validation

```
┌─────────────────────────────────────────────┐
│ Ring 1: PARSE VALIDATION                     │
│ AI output → lexer → parser → AST             │
│ If parser produces errors → fix REJECTED     │
├─────────────────────────────────────────────┤
│ Ring 2: SANDBOX EVALUATION                   │
│ AST → eval(env.Copy()) → result              │
│ If result is ERROR or nil → fix REJECTED     │
│ Sandbox active: no I/O, no exec, no network  │
│ Variable mutations isolated via env.Copy()    │
├─────────────────────────────────────────────┤
│ Ring 3: REAL EVALUATION                      │
│ Same AST → eval(real env) → final result     │
│ Only passes if rings 1+2 succeeded           │
└─────────────────────────────────────────────┘
```

This means the AI-generated code must:
1. **Parse** as valid Pipe syntax
2. **Execute without error** in a variable-isolated environment
3. **Repeat successfully** in the real environment

#### What Makes It Safe (Empirical Evidence)

1. **Limited attack surface**: `try_ai` only fixes single expressions — not arbitrary code blocks. The AI receives one expression, returns one expression. Maximum blast radius: one line.

2. **Stateless validation**: Each fix is validated independently. A malicious fix from call N cannot affect validation of call N+1.

3. **Non‑persistent fixes**: The fix is never written to disk. It exists only in memory for the duration of the expression evaluation and is discarded afterward.

4. **Deterministic escape hatch**: `catch` is always available. If the AI fails all 3 retries, control passes to developer-written fallback code. No unexpected behavior escapes the `try_ai` block.

5. **No system prompt injection possible**: The system prompt (`aiFixSystemPrompt`) is a **compile‑time constant** in the Go binary. Runtime error messages go into the `user` role of the chat request, which cannot override system instructions in any of the supported AI provider APIs (OpenAI, Anthropic, DeepSeek, Ollama).

#### What Remains Risky

1. **AI provider dependency**: If the AI provider is compromised, `try_ai` could receive malicious fixes. **Mitigation**: Use a trusted provider or self-hosted Ollama.

2. **Semantic errors**: The validation only checks that the fix executes, not that it produces the *correct* result. `"42" * 3` → `(to_num "42") * 3 = 126` is correct. But `"42" * 3` → `42 * 3 = 126` is equivalent. However, `10 / 3` → `10 / (max 3 1) = 3.33...` vs the incorrect `10 / 1 = 10`. The AI may pick a mathematically different but valid expression. **Mitigation**: Use `catch` with explicit expected result validation (`assert_eq`).

3. **Cost**: Each fix attempt consumes an AI API call. With caching (`ai_cache`), repeated identical errors only call the API once.

#### Comparison with Industry

| Tool | AI modifies code at runtime? | Validation | Fallback | Open source? |
|------|------------------------------|------------|----------|--------------|
| **Pipe `try_ai`** | Yes (expressions only) | 3-ring incl. real sandbox | `catch` block | Yes (Apache 2.0) |
| GitHub Copilot | No (suggestions only) | Human review required | Manual undo | No |
| Cursor AI | No (IDE integration) | Human review required | Manual undo | No |
| AutoGPT / AgentGPT | Yes (arbitrary code execution) | None | Manual termination | Partially |
| LLM-as-judge pipelines | Yes (unconstrained) | Ad-hoc | Manual | Varies |

Pipe is the **only tool** that combines automated AI code fixes with real sandbox validation (no I/O, no exec, no network) and a guaranteed deterministic fallback — all in a single language construct.

#### Safety Summary

`try_ai` is **safe for production use** when:
- An AI provider is configured (`ai_provider`)
- A `catch` block is present for critical code paths
- The system prompt remains unmodified (compile‑time constant)
- Trust in the AI provider is established

`try_ai` adds **zero risk** when:
- No error occurs (the AI is never called)
- Running the Bytecode VM (`pipe -vm`, try_ai works fully via Tree-Walker bridge, no feature gap)
- The expression is already valid Pipe code

---

## 8.2 Stack Traces

When an error occurs, Pipe automatically generates a **stack trace** showing the call path that led to the error.

```pipe
deep_fn: fn
    10 / 0

middle_fn: fn
    deep_fn

try
    middle_fn
catch err
    print "Error: " ++ err
-- Error includes stack trace showing:
--   deep_fn (line 2)
--   middle_fn (line 6)
--   <main> (line 10)
```

The stack trace is included in the error message string, making it easy to identify where errors originated.

---

## 8.3 `return` for Early Function Exit

The `return` keyword exits a function immediately, optionally providing a value. It does not trigger `try`/`catch` — it is normal control flow.

```pipe
find_first_even: fn nums
    each nums fn n
        if n % 2 == 0
            -- early exit from find_first_even
                        return n
    return nil

nums: [1, 3, 5, 6, 7, 8]
result: find_first_even nums
-- 6
print result
```

`return` always exits the **current function scope**, not just the innermost block:

```pipe
outer: fn
    inner: fn
        -- exits inner, not outer
        return 42
    val: inner
    -- "Got: 42"
    print "Got: " ++ val
    return 99
-- 99
print outer
```

---

## 8.4 The `Result` Type

For cases where exceptions are undesirable, Pipe provides a `Result` type — an explicit way to represent success or failure.

### 8.4.1 Creating Results

```pipe
-- Wraps a value in success
ok_result: Ok 42
-- Wraps an error message
err_result: Err "failed"
```

- `Ok(value)` — represents a successful operation
- `Err(message)` — represents a failed operation with a reason string

### 8.4.2 Checking Results

```pipe
-- true
is_ok (Ok 42)
-- false
is_ok (Err "oops")
-- false
is_err (Ok 42)
-- true
is_err (Err "oops")
```

### 8.4.3 Unwrapping Results

```pipe
-- 42
unwrap (Ok 42)
-- ERROR: called unwrap on an Err value
unwrap (Err "oops")

-- 42
unwrap_or (Ok 42) 0
-- 0     (uses default)
unwrap_or (Err "x") 0
```

- `unwrap(result)` — returns value if Ok, **panics** if Err
- `unwrap_or(result, default)` — returns value if Ok, returns `default` if Err

### 8.4.4 Using Result in Functions

```pipe
safe_divide: fn a b
    if b == 0
        return Err "division by zero"
    Ok (a / b)

r1: safe_divide 10 2
if is_ok r1
    -- "Result: 5"
        print "Result: " + (unwrap r1)

r2: safe_divide 10 0
if is_err r2
    -- print the error message
        print "Failed: " + (unwrap r2)

-- Using unwrap_or for defaults
-- 5
v1: unwrap_or (safe_divide 10 2) -1
-- -1
v2: unwrap_or (safe_divide 10 0) -1
```

### 8.4.5 Result in Pipelines

Results integrate naturally with Pipe's pipeline operator:

```pipe
process: fn data
    parsed: parse_json data
    if parsed == nil
        return Err "invalid JSON"
    Ok (parsed.value * 2)

r: process `{value: 21}`
    -- 42 (or error if parsing failed)
        > unwrap

-- Safe pipeline with default
safe_r: process `{}`
    > (fn r
        unwrap_or r 0)

-- 0  (default, since "value" key is missing)
print safe_r
```

---

## 8.5 Error Avoidance Patterns

### 8.5.1 Nil Checks

Before accessing properties on a potentially nil value, check first:

```pipe
-- Defensive
process_user: fn user
    if user == nil
        return Err "no user provided"
    if user.name == nil
        return Err "user has no name"
    Ok (upper user.name)

-- Using unwrap_or for safe defaults
name: unwrap_or (process_user nil) "Anonymous"
-- "Anonymous"
print name
```

### 8.5.2 File Existence Checks

Always check file existence before operations that might fail:

```pipe
safe_read: fn path
    if !file_exists path
        return Err "file not found: " + path
    Ok (read_file path)

content: unwrap_or (safe_read "config.json") "{}"
config: parse_json content
print config
```

### 8.5.3 Type Checking Before Operations

```pipe
safe_add: fn a b
    if !is_num a or !is_num b
        return Err "both arguments must be numbers"
    Ok (a + b)

-- Ok 30
safe_add 10 20
-- Err "both arguments must be numbers"
safe_add 10 "x"
```

### 8.5.4 Combining `try`/`catch` with `Result`

For external operations that might fail, wrap with `try`/`catch` and convert to `Result`:

```pipe
read_config: fn path
    try
        raw: read_file path
        parsed: parse_json raw
        if parsed == nil
            return Err "invalid JSON"
        Ok parsed
    catch err
        Err ("read_config failed: " ++ err)
```
