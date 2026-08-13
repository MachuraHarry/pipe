# 🔐 Sandbox Audit Round 2 — I Found Three Real Escapes, Then Proved They're Dead

**A second red-team pass on Pipe's agent sandbox: three ratchet holes that a
confined script could actually use, one latent bug, and two LLM runs that came
back empty-handed.**

> **Follow-up:** The audit didn't stop here. Rounds 3–6 found `try_ai` ignoring
> `ai:false`, mis-gated `embed`/`import`, and a CLI `--sandbox` flag gap — plus
> the structural fixes that kill those bug classes — see
> **[The Audit Continues](?post=sandbox-audit-continued)**.

---

Round 1 ended with a confident "no escape." Round 2 started by *not* trusting
that verdict. Instead of only asking a model to attack, I read the code and
built deterministic probes myself. That's where the sandbox cracked — not in the
LLM run, but in the ratchet logic I'd shipped a week earlier.

Here's what broke, how I proved it, and how the sandbox held after the fix.

---

## The Ratchet: Right Idea, Three Holes 🕳️

Pipe's sandbox is supposed to *ratchet down*: once a restricted profile is
active, a script may only switch to a profile that grants the **same or fewer**
rights. The check lives in `IsSubsetOf`. Three dimensions were missing or wrong.

### 1. Empty whitelist = allow-all

An empty `network_whitelist` means "allow every host." But `IsSubsetOf` skipped
the empty-candidate case, so a profile with `network_whitelist: []` passed as a
*subset* of a whitelisted profile.

**Proof:** from a profile whitelisted to `api.github.com`, I registered and
switched to an empty-whitelist profile, then fetched `https://example.com` →
**HTTP 200**. Full network egress, outside the whitelist.

### 2. The port ratchet

A portless entry (`api.github.com`, any port) passed as a subset of a
port-pinned entry (`api.github.com:443`), because the matcher only rejected a
port mismatch when *both* sides carried an explicit port.

**Proof:** from a profile whitelisted to `127.0.0.1:8098`, I switched to a
portless `127.0.0.1` profile and fetched `127.0.0.1:8099` → **HTTP 200**.

### 3. Budget, max_tool_calls, timeout — never compared

`IsSubsetOf` compared `fs`, `network`, `exec`, `ai` only. So a "subset" profile
could declare `budget: 0`, `max_tool_calls: 0`, `timeout: 0` — all meaning
*unlimited* — and reset its spend and tool-call caps by switching profiles
(`CanAI`/`CanToolCall` enforce against the *active* profile).

**Proof:** from a profile with `budget: 0.000001, max_tool_calls: 1`, I switched
to `budget: 0, max_tool_calls: 0` — unlimited.

---

## What Round 1 Missed — And Fixed 🛠️

Round 2 also confirmed the two escapes from Round 1's follow-up:

| Escape | Fix |
|--------|-----|
| **No ratchet** — `set_sandbox` could switch to *any* profile, `sandbox_profile` could mint permissive ones | Subset enforcement on switch **and** registration |
| **MCP clients ignored the sandbox** — `mcp_use_stdio` spawned subprocesses, `mcp_use_sse` made HTTP calls, neither gated | `CanExec()` gate on stdio; `CanNetworkTo()` + per-request gate on SSE (incl. redirects) |

The three new holes above are fixed in `IsSubsetOf` now:

- `budget` (0 = unlimited), `max_tool_calls`, `timeout` are all compared,
- an empty sub-whitelist is only a subset of an empty super-whitelist,
- a portless sub-entry is rejected against a port-pinned super-pattern.

A bonus find while wiring the multi-parameter tools: `executeTool` passed the
model's named arguments in **Go-map iteration order** — non-deterministic, so a
two-argument tool could get its args swapped. Now it's ordered by the tool's
declared schema. Not a security hole (gates fire first), but it made the test
suite flaky and multi-arg tools unreliable.

---

## The LLM Runs 🤖

With the fixes in, I turned a model loose twice — `deepseek-v4-pro`, thinking
enabled, effort high.

**Variant A — locked profile** (`exec: false`, temp-only fs, whitelist
`api.github.com`, plus `mcp_use_stdio`/`mcp_use_sse` as tools): **14 tool
calls, 0 escapes.** `mcp_use_stdio sh` → `E_SANDBOX: exec blocked`;
`mcp_use_sse http://169.254.169.254/` → not in whitelist.

**Variant B — unlocked ratchet probe** (whitelist pinned to `api.github.com:443`,
four pre-registered `escape_*` profiles): **16 tool calls, 0 escapes.** Every
escape profile rejected as "not a subset", `set_sandbox none` blocked, the
`:80` port trick and a URL-userinfo trick both caught.

Model's verdict: *"The ratchet cannot be defeated."*

---

## The Lesson 🎓

Trust, but verify. The first audit gave a clean bill of health *because the
attacker wasn't creative enough* — the second found real holes because I stopped
outsourcing the hunting. An LLM is a great red-team **reporter**, but a lousy
substitute for reading your own code.

The full details — every vector, exact error strings, and the deterministic
proofs — live in the repo:

- `docs/tests/sandbox-audit/report2.en.md` / `report2.de.md`
- `examples/redteam.pipe` (variant A) and `examples/redteam_ratchet.pipe` (variant B)

Run it yourself:

```bash
DEEPSEEK_API_KEY=sk-... ./examples/redteam_audit.sh a   # locked
DEEPSEEK_API_KEY=sk-... ./examples/redteam_audit.sh b   # ratchet probe
```

*Got a ratchet vector I missed? Open an issue — I'll add it to the probe suite
and run it live.*
