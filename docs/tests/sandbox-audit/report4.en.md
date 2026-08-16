# Red-Team Sandbox Audit — Round 4

**Date:** 2026-08-13
**Model:** DeepSeek V4 Flash (`deepseek-v4-flash`), real live calls
**Key-Handling:** No key stored in the repo. `DEEPSEEK_API_KEY` read from the
process environment; rotated by the operator after the live test.

---

## Result: `embed()` gated the wrong dimension — plus a raw SSRF in `import`

Round 4 found **two** new escapes that slipped through Rounds 1–3, both
live-proven, plus one consistency gap. The lesson the audit stresses:
*"clean audit" only means "clean against what was checked", never "secure".*

---

## Gap 1 — `embed` gated on `CanNetwork()`, not `CanAI()`

`bEmbed` (`pkg/object/builtins_ai.go`) gated on `CanNetwork()` while every
other AI model egress gates on `CanAI()`. A profile `ai: false, network: true`
— a plausible "no LLM, but normal API calls" policy — did **not** block
`embed`. `embed` makes a real HTTPS call to the provider's embeddings API.
Live: with the provider configured, `embed "hello world"` issued a real
request; only the audit container's own network policy stopped it, **not**
Pipe's sandbox.

## Gap 2 — `embed_batch` had **no** gate at all

`bEmbedBatch` called `ai.EmbedBatch()` directly — it bypassed both `CanAI`
**and** `CanNetwork`. No profile could block it.

## Gap 3 — `import "https://…"` was a raw, ungated HTTP GET (SSRF)

`ResolveImportFrom` → `fetchURLModule` → `httpGet` fetched **any** URL with no
sandbox check and no redirect re-check, then executed the fetched source. A
sandboxed script could hit metadata endpoints (`169.254.169.254`), internal
services, or attacker URLs. The registry fallback (`resolveRegistryModule`)
used the same ungated path.

## Consistency — `bAgent` had no gate

`agent` (create) had no `CanAI()` gate while `agent_ask` had one. Creating an
agent triggers no call, so lower severity, but the split was inconsistent and
future-unproof.

---

## Fixes (commit `77e8a34`)

- **`bEmbed`** — gate switched to `CanAI()` (with the `"none"`-profile
  fallback `Sandbox.Enabled && !Sandbox.AllowAI`, same as `ai_chat`).
- **`bEmbedBatch`** — same `CanAI()` gate added.
- **`bAgent`** — same `CanAI()` gate added.
- **`import`** — `fetchURLModule` now checks `ActiveProfile.CanNetworkTo(url)`
  (with the `"none"` fallback `Sandbox.Enabled && !Sandbox.AllowNet`),
  covering both explicit URL imports and the registry fallback; `httpGet`
  now uses `sandboxHTTPClient`, so redirect hops are re-checked per hop.

## Tests

- `TestEmbedBlockedByAIProfile` / `TestEmbedBatchBlockedByAIProfile` —
  `ai: false, network: true` + httptest counter → exactly **0** requests.
- `TestEmbedAllowedWithAI` — `ai: true` → request reaches the server.
- `TestImportURLBlockedByProfile` / `TestImportURLAllowedByWhitelist` —
  network:false blocks (0 requests); whitelist match allows (1 request).
- Live (DeepSeek, key-gated, skip without `DEEPSEEK_API_KEY`):
  `TestEmbedRealProviderBlockedByAIProfile`,
  `TestEmbedBatchRealProviderBlockedByAIProfile` — both pass in **0.00s**,
  i.e. the gate fires before any provider logic runs.

---

## Live verification (real DeepSeek, rotated key)

| Test | Result |
|------|--------|
| `embed` under `ai: false` with a live key | ✅ **0.00s** — `E_SANDBOX: AI calls blocked by profile` |
| `embed_batch` under `ai: false` with a live key | ✅ **0.00s** — blocked identically |

The `0.00s` runtime proves no request left the process. (DeepSeek exposes no
embeddings API, so the *allowed* direction is covered deterministically by the
httptest test rather than a live call.)

---

## Known limitations / open work (not fixed here)

1. **Global `ActiveProfile`** — a single process-wide `atomic.Pointer`, not
   request-scoped. Concurrent tool execution is serialized via `toolExecMu`
   (mitigates parallel MCP requests), but a fully request-scoped profile
   remains open work.
2. **File-I/O builtin inventory** — done in `pkg/object/sandbox_fs_inventory_test.go`: every file-read/write builtin audited against the temp-only redirect (writes land in the sandbox copy, outside files untouched; reads cannot leak outside content; isolated/fs:none profiles block all file builtins).
3. **Module install vs. runtime import** — `pkg/module/install.go`
   (`pipe install`) fetches URLs as an explicit operator action; it was
   considered out of scope for the *script* sandbox.

---

## Honest framing (as requested by the audit)

These reports do **not** claim the sandbox is absolutely secure. The correct
reading: **no known escapes as of commit `77e8a34`; actively re-red-teamed.**
Every round so far has found at least one real gap that slipped through the
previous rounds — including one (round 3) that lived in a file already
reviewed twice in the same session.
