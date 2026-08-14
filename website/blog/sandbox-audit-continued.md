# 🔬 The Audit Continues: try_ai Lied, embed Slipped, and We Made the Gates Structural

**Rounds 3–6 of Pipe's sandbox audit: one flat-out lie in an error-repair
feature, two builtins that forgot their own gate, and the structural fixes that
make both bug classes unreproducible.**

> **Follow-up:** This post covers rounds 3–6. For the earlier rounds, see
> **[Sandbox Audit Round 1](sandbox-audit.html)** and
> **[Round 2](sandbox-audit-2.html)**.

---

Round 2 taught me to stop outsourcing the hunting: the real escapes came from
reading my own code, not from an LLM run. So after the ratchet fix, I kept
probing. Four more rounds, four more findings — and the last two weren't patched
with another `if` branch. They were *made structurally impossible*.

Here's the whole arc, honest about every slip.

---

## Round 3 — `try_ai`: "ai: false" Was a Lie 🕳️

Pipe has a self-healing feature: when a script hits a runtime error (undefined
variable, type mismatch, division by zero), `try_ai` asks the model to fix the
expression and re-runs it. Up to **three attempts per error** — each one a real,
paid `ai.Chat()` call.

And none of them checked the sandbox. A profile with `ai: false` was still
calling the provider. A profile with `budget: 0.01` could spend whatever it
wanted before the budget was ever consulted. Every script that triggered an
error — even by accident — made live provider calls, regardless of policy.

**The fix:** every repair attempt is now gated exactly like the AI builtins
(`CanAI()` per attempt, so costs recorded between attempts count), and the
`try_ai_ring2` fallback profile inherits `budget`, `max_tool_calls`, `timeout`
and `audit_log` from the caller instead of being hardcoded unlimited.

**Proof:** the blocked-path test runs in **0.00s** — zero provider requests ever
left the process. A regression of the gate would turn that test back into a
real, paid call and it would fail.

---

## Round 4 — `embed`, `embed_batch`, `import`: Gates Pointed the Wrong Way 🔎

Three more gaps in the same family:

- `embed` gated the **wrong dimension** (it checked the AI flag against the
  wrong profile field),
- `embed_batch` had **no gate at all**,
- `import` could be pointed at a network URL — an **SSRF** hole in the module
  loader.

All three fixed and covered by tests. Nothing dramatic — but it confirmed a
pattern that was about to become the story of this entire audit: **the gates
were per-builtin hand-rolled code, and per-builtin hand-rolled code forgets
things.**

---

## Round 5 — The Central Egress Gate: Killing the Bug Class 🏗️

If every builtin must remember its own `CanAI()` check, then "a builtin forgot
its check" is a bug you will keep re-discovering. So I stopped patching builtins
and moved the checkpoint to where it can't be missed.

`pkg/ai` now has a **central egress gate** (`SetEgressGate`). Every provider
egress — `Chat`, `Stream`, `ChatParallel`, `ChatWithTools`, `Embed`,
`EmbedBatch`, `WebSearch` — must pass it **before any network call is made**,
including **cache hits** (a cached response is still a policy decision, and it
still has to be gated).

The gate is wired to the active profile once, at init. From that commit on, the
round-3/4 bug class — *a builtin that forgot its own gate* — is structurally
impossible. The per-builtin checks remain as defense-in-depth.

---

## Round 6 — The CLI Flag Path Was the Blind Spot 🪟

The central gate closed the **AI-provider** egress. But the audit kept going,
and found a different blind spot entirely: the enforcement lived on **two
paths** — registered profiles (`network: false`) and the CLI flags
(`pipe --sandbox`, which keeps the default `"none"` profile and governs network
via `Sandbox.AllowNet`).

Three network builtins — `http_post`, `tcp_connect`, `tcp_listen` — only checked
the **profile** path. So `pipe --sandbox script.pipe` — which promises to
restrict "dangerous builtins (exec, tcp, http, ai, fs-write)" — still let a
script POST data to arbitrary hosts and open TCP connections.

**The fix:** a single central helper, `checkNetworkAccess`, that wraps *both*
paths (profile `CanNetwork` **and** the `--sandbox` flag check). All **seven**
network-capable builtins now use it. Same medicine as round 5: one checkpoint,
not seven hand-rolled `if`s.

**Proof:** under `--sandbox`, `http_post` is now blocked *before any network
I/O* — the block test never dials. The control run without `--sandbox` reaches
the dial and gets `connection refused`, proving the block is sandbox-specific.

---

## The Meta-Lesson 🎓

Every round of this audit is the same story with a different face:

| Round | The bug | The pattern |
|-------|---------|-------------|
| 3 | `try_ai` ignored `ai:false` + budget | a feature outside the builtin gates |
| 4 | `embed`/`embed_batch`/`import` un- or mis-gated | hand-rolled per-builtin gates |
| 5 | gate could be forgotten *anywhere* | per-builtin checks, by construction |
| 6 | CLI flag path drifted from profile path | **two** enforcement paths to forget |

The fix pattern is the same too: **don't add more `if`s — move the checkpoint
where it can't be missed.** A central gate and a central helper can't be
"forgotten" by a future builtin, because there's nothing to remember. That's the
difference between a security feature and a security architecture.

---

## Where It Stands (honest) 🏁

- **No known escapes.** AI-provider egress, the profile ratchet, and the CLI
  `--sandbox` flag path are all covered — by construction, not by convention.
- **Remaining known limitations**, still on the list: the global
  `ActiveProfile` is process-wide (tool execution is serialized, but true
  concurrency is untested), the file-I/O builtins haven't had a full inventory
  pass against the temp-only redirect, and provider/MCP HTTP doesn't run through
  the redirect-rechecking client.

The full bilingual reports, every vector and exact error string, live in the
repo under `docs/tests/sandbox-audit/` (`report3`–`report6`, EN/DE), and the
tests are in `pkg/object/egress_gate_test.go` and `network_gate_test.go`.

Run the live red-team yourself:

```bash
DEEPSEEK_API_KEY=sk-... ./examples/redteam_audit.sh a   # locked profile
DEEPSEEK_API_KEY=sk-... ./examples/redteam_audit.sh b   # ratchet probe
```

*Found a vector we missed? Open an issue — we'll add it to the probe suite and
run it live.*
