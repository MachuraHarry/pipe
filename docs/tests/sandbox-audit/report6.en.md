# Red-Team Sandbox Audit — Round 6

**Date:** 2026-08-13
**Method:** static egress inventory + verification against the CLI flag set
**Commit baseline:** after `f6879f8`

---

## Finding: `--sandbox` without a profile let `http_post`/`tcp_connect`/`tcp_listen` through — now gated

Round 6 found a gap in the **CLI flag path**: the command-line sandbox mode
`pipe --sandbox <script>` (without `--sandbox-profile`) claims to restrict
"dangerous builtins (exec, tcp, http, ai, fs-write)" — but three network-capable
builtins **ignored the network ban** and kept performing real network egress
(HTTP-POST and TCP connections).

Scope note: the **AI-provider egress** (central gate from round 5) and the
**profile path** (`strict`/`noexec`/`isolated`/custom with `network: false`) were
not affected. The gap was limited to the "`none` profile + sandbox flags" path.

---

## The gap — the `if Name != "none"` path forgot the CLI flags

The CLI `--sandbox` flag sets `Sandbox.Enabled = true` and leaves `ActiveProfile`
at `"none"` (the "allow everything" profile). Builtin gates have followed this
pattern since round 1:

```go
if ActiveProfile.Load().Name != "none" {
    if canErr := ActiveProfile.Load().CanNetwork(); canErr != nil {
        return err(canErr.Error())
    }
} else if Sandbox.Enabled && !Sandbox.AllowNet {
    return sandboxBlock("... (network)")
}
```

The second branch (`else if Sandbox …`) was missing in **three** network-capable
builtins — they only checked the profile and let everything through in the
`"none"` path:

- `http_post` (`pkg/object/builtins_net.go`)
- `tcp_connect` (`pkg/object/builtins_net.go`)
- `tcp_listen` (`pkg/object/builtins_net.go`)

`http_get`, `http_request`, `http_stream_open` and `http_server` were covered. A
red-teamed script could therefore keep exfiltrating data via HTTP-POST or opening
TCP connections under `pipe --sandbox` — exactly the egress surface the sandbox
mode promises to stop.

---

## Fix (branch `network-gate-6`)

- **Central helper** `checkNetworkAccess(feature)` in
  `pkg/object/builtins_sandbox.go`: wraps both branches (profile `CanNetwork`
  **and** `Sandbox.Enabled && !Sandbox.AllowNet`) in a single place.
- All **seven** network-capable builtins now use it:
  `http_get`, `http_post`, `http_request`, `http_stream_open`, `http_server`,
  `tcp_connect`, `tcp_listen`. The whitelist checks (`CanNetworkTo`) remain as
  defense-in-depth.
- The "forgotten `AllowNet` branch" bug class is now structurally unreproducible:
  a future builtin doing network I/O only has to call `checkNetworkAccess`.

### Regression tests (`pkg/object/network_gate_test.go`)

| Test | covers |
|------|--------|
| `TestNetworkBuiltinsBlockedBySandboxFlag` | all 7 builtins blocked under `Sandbox.Enabled=true, AllowNet=false` (block before dial/request, no network I/O) |
| `TestNetworkBuiltinsAllowedBySandboxFlag` | `http_get`/`http_post` work against an `httptest` server under `AllowNet=true` |
| `TestNetworkBuiltinsBlockedByProfile` | blocked under a `network: false` profile |
| `TestNetworkBuiltinsAllowedByProfile` | `http_post` works under a `network: true` profile |

---

## Live verification (CLI, no real remote hosts)

```text
$ printf 'http_post "http://127.0.0.1:9/" "{}"' > net.pipe

$ timeout 20 pipe --sandbox net.pipe
Runtime error: net.pipe: SANDBOX: http_post (network) is disabled in sandbox mode
exit=1                                   # blocked before any network I/O ✓

$ timeout 20 pipe net.pipe               # control without --sandbox:
Runtime error: http_post: Post ... dial tcp 127.0.0.1:9: connect: connection refused
exit=1                                   # reaches the dial → block is sandbox-specific ✓
```

The block happens **before** any network I/O (no `net.Dial`/`client.Post`), as
shown by the control group reaching `connection refused`.

---

## Assessment (honest)

- No known escape in **AI egress** (central gate, round 5) or the **profile path**
  (ratchet, round 2).
- Found and fixed: the `--sandbox` flag path let `http_post`/`tcp_connect`/
  `tcp_listen` through. Both directions (blocked and allowed) are covered by tests.
- Remaining known limitations: unchanged from rounds 4/5 (global `ActiveProfile`,
  file-I/O inventory, layer-2 without redirect re-check).
