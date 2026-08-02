# Pipe v0.7.0 — Parallel, Self-Healing, and Tested by Default

**Today we're shipping Pipe v0.7.0** — the biggest release yet. This version turns Pipe from a promising prototype into a language you can take seriously: parallel pipelines that just work, AI that fixes your code at runtime, a zero-setup test framework, module versioning, and production tooling for CI and servers.

**Quick stats:** 230+ Go tests · 8 integration suites · 42 example programs · 114 builtins (23 AI) · single ~10 MB binary · still zero dependencies.

---

## Parallel by Design: `>>`

No `async/await`. No `Promise.all`. In Pipe, parallelism is a language primitive.

The `>>` operator starts any pipeline stage in the background and returns a **Future** that auto-resolves when you read it:

```pipe
fn ask_about topic
    ask ("What is " ++ topic ++ "?")

a: "What is Paris?"
    >> ask_about
b: "What is Berlin?"
    >> ask_about
c: "What is Rome?"
    >> ask_about

print a ++ " | " ++ b ++ " | " ++ c    -- futures auto-resolve
```

Three AI calls in ~1.5 seconds instead of ~4.5. For bigger workloads, `ai_batch` processes hundreds of texts concurrently with built-in rate limiting.

## Self-Healing Code: `try_ai`

`try_ai` catches runtime errors and uses AI to **repair the broken expression automatically** — type mismatches, division by zero, index errors. No other language has this.

```pipe
ai_provider "deepseek"

result: try_ai
    "42" * 3            -- E002: STRING * INTEGER
catch e
    0                   -- only reached if the AI can't fix it
```

The AI receives the error code, the expression source, and the runtime state. The suggested fix goes through a **3-ring validation** — parse check, sandbox test, type check — before it's applied to your real environment. If it can't be fixed safely, execution falls through to `catch`.

## Testing Without Setup

No test runner to install. No config file. No framework to learn.

```pipe
test "math"
    fn add a b
        a + b
    assert_eq (add 2 3) 5
    assert_lt 1 2

test "error handling"
    fn boom
        "str" * 3
    assert_error boom
```

Run it with `pipe -test`. The runner auto-discovers `*_test.pipe` files — that's the entire story.

## Modern Control Flow

Small language, everyday comfort:

**C-style `for` loops** — now in both the tree-walker and the bytecode VM:

```pipe
for i: 0; i < 5; i: i + 1
    print i
```

**Multi-pattern `match`** — collapse several cases into one body instead of repeating code:

```pipe
match status
    | 200 | 201 -> print "ok"
    | 404 | 410 -> print "gone"
    | _         -> print "other"
```

**`not`** instead of `!` — reads naturally in conditions:

```pipe
if not (2 > 3)
    print "clearly true"
```

## Module Versioning

The module ecosystem gets real versioning. Pin exactly what you run:

```bash
pipe -get parallel-runner@1.0.0
```

```pipe
import "parallel-runner@1.0.0"
```

`pipe -search` discovers modules, `pipe -get` installs them, `@x.y.z` pins versions. No URLs, no lockfile hell.

## Sandbox Profiles for AI Agents

Declarative runtime security. Lock down what AI agents and untrusted scripts can do:

```pipe
sandbox_profile "strict" {fs: "read-only", network: false, exec: false, ai: false}

set_sandbox "strict"

write_file "/tmp/test.txt" "blocked"   -- E_SANDBOX: blocked
read_file "/tmp/test.txt"              -- allowed (read-only)
```

Profiles control filesystem (`none`/`read-only`/`temp-only`/`full`), network, `exec`, and AI access. Essential when you hand tools to an LLM via `ai_with_tools`.

## Better Errors: E001–E004

Errors now carry codes and contextual hints:

```
E002: type mismatch: cannot apply '*' between STRING and INTEGER
      (at line 3, column 9)
      hint: use to_num to convert before multiplying
```

Parser errors point at the offending token and suggest what's missing. AI-fixable errors are tagged so `try_ai` knows what it can repair.

## CI/CD, Servers, and the Web

- **Official GitHub Action** — run Pipe scripts and tests in CI with one step:

  ```yaml
  - uses: MachuraHarry/pipe/pipe-action@master
    with:
      sandbox: true
      script: |
        test "ci"
            assert_eq (1 + 1) 2
        print "CI passed"
  ```

- **HTTP API server** (`cmd/api-server`) — expose Pipe pipelines as JSON endpoints, deployable to Fly.io in minutes.
- **WASM playground v2** — code sharing, syntax highlighting, community rating — running the real Pipe runtime in your browser at [pipe-lang.com/playground.html](https://pipe-lang.com/playground.html).
- **Formatter** gained `--check` mode and directory processing; the **REPL** now persists history across sessions.

---

## The Road Ahead

v0.7 was about making Pipe *complete and trustworthy*. Next on the roadmap: a richer standard library, more providers, deeper local-model support (Ollama), and growing the community module registry.

## Try It

```bash
git clone https://github.com/MachuraHarry/pipe && cd pipe && make build
./bin/pipe script.pipe
```

…or run Pipe in your browser right now — no install, no signup:

👉 **[pipe-lang.com/playground.html](https://pipe-lang.com/playground.html)**

*MIT licensed · built with Go · Linux, macOS, Windows, and WebAssembly*
