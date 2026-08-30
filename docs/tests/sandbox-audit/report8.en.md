# Red-Team Sandbox Audit — Round 8

**Date:** 2026-08-30
**Method:** inventory of every `pkg/ai` function that dispatches a real network request, verified against the central egress gate (round 5) — same sweep methodology as rounds 6/7, applied to the AI-provider egress dimension itself
**Commit baseline:** after `ec8e8de`

---

## Finding: `wiki_search` never reached the central egress gate — real network egress under `--sandbox`

Round 5 introduced a central egress gate (`SetEgressGate`/`gateEgress` in
`pkg/ai/ai.go`) so that no AI-provider or search egress point can silently
skip the sandbox check by forgetting a builtin-level `CanAI`/`CanNetwork`
call. Round 8 re-verified that every egress function actually calls the gate
before dispatching — and found one that doesn't: `WikiSearch`
(`pkg/ai/search.go`), reachable via the `wiki_search` builtin, made its HTTP
request to `en.wikipedia.org` **without ever calling `gateEgress`**. Under
`pipe --sandbox` (no `--sandbox-profile`, network disabled), the query text
was sent to Wikipedia's public API and real results were returned — the
build's own sandbox-mode promise ("dangerous builtins ... blocked") did not
hold for this one builtin.

Scope note: `WebSearch` (the sibling DuckDuckGo-backed builtin) already
called `gateEgress(EgressSearch, apiURL)` correctly and was covered by the
round-5 regression test `TestEgressGateBlocksAtEntry` — `WikiSearch` was
simply missing from that inventory. Every other egress path (`Chat`,
`Stream`, `ChatWithTools`, `ChatParallel`, `Embed`, `EmbedBatch`, and all six
provider-specific chat/stream implementations in `pkg/ai/providers.go`) was
re-traced call-site by call-site in this round and confirmed to be reachable
only through an already-gated entry point.

---

## The gap — one function skipped the gate its sibling already used

```go
// WebSearch (correct, existing):
func WebSearch(query string) ([]SearchResult, error) {
    apiURL := ...
    if err := gateEgress(EgressSearch, apiURL); err != nil {
        return nil, err
    }
    body, err := httpGetString(EgressSearch, apiURL)
    ...
}

// WikiSearch (before the fix — no gate call):
func WikiSearch(query string) ([]SearchResult, error) {
    apiURL := ...
    body, err := httpGetString(EgressSearch, apiURL)   // dispatched straight through
    ...
}
```

`httpGetString` → `gatedHTTPClient` only re-checks the gate on **redirect
hops** (`CheckRedirect`), not on the initial request — that's a deliberate
round-5 design (see `TestEgressGateRechecksRedirectHops`), not a bug. The
initial dispatch has to be gated by the caller, and `WikiSearch` never did
that. The `wiki_search` builtin itself (`pkg/object/builtins_ai.go`) adds no
protection either: like `bWebSearch`, it only checks `ActiveProfile.CanNetwork()`
under a registered profile and has no CLI-`--sandbox`-flag fallback — by
design, since round 5 moved enforcement to the network-dispatch point itself
rather than duplicating it in every builtin. For `wiki_search` that design
assumption silently broke: enforcement never happened anywhere on this path.

---

## Fix (branch: this round)

- Added the missing `gateEgress(EgressSearch, apiURL)` call to `WikiSearch`
  in `pkg/ai/search.go`, immediately mirroring `WebSearch`.
- Added `WikiSearch` to `TestEgressGateBlocksAtEntry` in
  `pkg/ai/ai_gate_test.go` (round 5's own entry-point inventory test), so
  the gate list can no longer silently omit a real egress function.

No other fs-write/network/exec/AI dimension changes were needed — the rest
of the `pkg/ai` egress surface was confirmed clean by this round's sweep.

---

## Live verification (CLI, real public endpoint, no API key required)

```text
$ printf 'print(wiki_search("Pipe programming language"))' > wiki.pipe

$ timeout 20 pipe --sandbox wiki.pipe        # before the fix:
[{title: Comparison of multi-paradigm programming languages, ...}, ...]
exit=0                                       # real Wikipedia data returned — sandbox bypassed

$ timeout 20 pipe --sandbox wiki.pipe        # after the fix:
Runtime error: wiki.pipe: E_SANDBOX: network access blocked by sandbox
  in fn(wiki_search)
exit=1                                       # blocked before any network I/O ✓

$ timeout 20 pipe wiki.pipe                  # control without --sandbox:
[{title: Comparison of multi-paradigm programming languages, ...}, ...]
exit=0                                       # unaffected outside sandbox mode ✓
```

---

## Assessment (honest)

- Found and fixed: `wiki_search` reached a real external API under
  `pipe --sandbox`, the only known network egress path that never touched
  the central gate at all (not even as inconsistent defense-in-depth like
  the round-6/7 builtin-level gaps — this one had zero enforcement).
- Every other AI/network egress function in `pkg/ai` was traced to a single
  gated entry point (`Chat`, `Stream`, `ChatWithTools`, `ChatParallel`,
  `Embed`, `EmbedBatch`, `WebSearch`) and confirmed clean.
- `ollamaEmbed` in `pkg/ai/providers.go` was noted as dead code (defined,
  never called — `Embed` always uses the generic embeddings path) during
  this sweep; not a security issue, left untouched as out of scope for this
  round.
- No known escape in the **fs-write gate** (round 7), **network gate**
  (round 6), or the **profile path** (ratchet, round 2).
- Methodology note: this is the third round in a row that found a gap by
  systematically re-verifying an existing gate's own inventory rather than
  through red-team LLM interaction. The pattern across rounds 6-8 is
  consistent: a **capability dimension gets a correct central gate, but the
  inventory of call sites feeding into it is not exhaustively re-checked
  when a new function is added** (round 6: 3 of 7 network builtins; round 7:
  6 of 8 fs-write builtins; round 8: 1 of 7 AI/search egress functions).
  Future work on this codebase should treat "add a new egress-capable
  function" as requiring an inventory-test update in the same commit, not
  just a passing build.
