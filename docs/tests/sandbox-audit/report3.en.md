# Red-Team Sandbox Audit — Round 3

**Date:** 2026-08-13
**Model:** DeepSeek V4 Flash (`deepseek-v4-flash`), real live call
**Key-Handling:** No key stored in the repo. `DEEPSEEK_API_KEY` read from the
process environment; rotated by the operator right after the live test.

---

## Result: `ai: false` was a lie — `try_ai` bypassed it; now gated & live-verified

Round 3 found a gap that breaks the core sandbox promise: **`try_ai` ignores
`ai: false` and makes real, billable API calls** no matter what the active
profile says. Fixed and proven with a live DeepSeek call in both directions
(blocked with `ai: false`, working with AI enabled).

---

## The Gap — `try_ai` self-healing bypassed every AI gate

The `try_ai` error-repair feature calls the AI provider on any ordinary
runtime error (E001–E006: undefined variable, type conflict, division by
zero, …) — up to **3 attempts per error**, each a real `ai.Chat()` call.
Unlike every other AI entry point (`ai_chat`, `ai_chat_json`, `ai_with_tools`,
`mcp_use_stdio`, …), the fix loop in `tryAIFix()` (`pkg/eval/eval.go`) never
checked the active profile:

- No `CanAI()` gate → `ai: false` was ignored.
- No budget pre-check → a `budget`-capped script could spend unbounded
  (cost *is* recorded into the active profile via `ai.SetCostHook`, but the
  `CanAI()` budget check was never invoked before a call).
- `pkg/ai/` itself is completely blind to `ActiveProfile`/`CanAI`.

The trigger needs no malice: any script hitting an E001–E006 error triggers up
to three live provider calls, regardless of the `ai: false` policy.

### Bonus finding — `try_ai_ring2` profile was hardcoded unlimited

`validateAndApply()` swapped in a hardcoded `try_ai_ring2` profile
(`AI: true`, `FSNone`, no network, no exec) with **no `Budget` and no
`MaxToolCalls`** — both `0` mean *unlimited* per `budgetIsSubset`/
`limitIsSubset`. An AI-generated fix expression that itself called an AI
builtin would pass ring-2's `CanAI()` (`AI: true`, budget `0` skips the
check) and spend without any cap, recorded against the throwaway ring-2
profile instead of the caller's.

---

## Fixes (commit `e96ff44`)

### `tryAIFix()` now gates every attempt (`pkg/eval/eval.go`)

Before each `ai.Chat()` in the repair loop, the active profile is checked
with the same pattern the other AI builtins use:

- active profile `Name != "none"` → `profile.CanAI()` (blocks on `ai: false`
  **and** on an exhausted budget, re-checked per attempt so costs recorded
  between attempts count);
- `"none"` profile → `Sandbox.Enabled && !Sandbox.AllowAI` global check.

Blocked attempts are logged via `ai.LogTryAIFix` and `tryAIFix` returns `nil`,
so `try_ai` falls through to `catch`/error handling exactly as before.

### `try_ai_ring2` inherits the caller's limits

Extracted `newTryAIRing2Profile(prev)` — the ring-2 profile now inherits
`Budget`, `MaxToolCalls`, `Timeout` and `AuditLog` from the calling profile.
A nested AI call inside a fix expression can no longer outspend the caller.

---

## Live verification (real DeepSeek, rotated key)

Both tests use the real provider; they `t.Skip` when `DEEPSEEK_API_KEY` is
unset, so CI stays offline.

| Test | Result |
|------|--------|
| `TestTryAIRealProviderFixes` | ✅ **2.41s** — AI enabled: `try_ai` called DeepSeek and fixed `no_such_var + 1` → `(0 + 1)` → result `1` |
| `TestTryAIRealProviderBlockedByAIProfile` | ✅ **0.00s** — `ai: false`: blocked (`E_SANDBOX: AI calls blocked by profile 'no_ai'`), **zero** provider requests, catch fallback |

The `0.00s` runtime of the blocked test is itself the proof: no request ever
left the process. A regression of the gate would turn it into a live, billable
call and the test would fail.

## Additional tests

- `TestTryAIFixBlockedByProfile` — `httptest` server counts requests;
  with an `ai: false` profile exactly **0** requests arrive.
- `TestTryAIRing2InheritsLimits` — ring-2 copies budget / max_tool_calls /
  timeout / audit_log from the caller, keeps `FSNone`, no network, no exec,
  `AI: true`.

---

## Verification

- `gofmt`, `go build ./...`, `go test ./...` (offline, integration tests skip) — green.
- Live run with `DEEPSEEK_API_KEY=… go test ./pkg/eval/ -run TestTryAIRealProvider -v` — both tests pass as shown above.
