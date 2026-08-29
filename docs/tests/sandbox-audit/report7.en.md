# Red-Team Sandbox Audit — Round 7

**Date:** 2026-08-29
**Method:** static egress inventory of every fs-write builtin + verification against the CLI flag set, following the exact bug class found in round 6
**Commit baseline:** after `995bfd8`

---

## Finding: `--sandbox` without a profile let 6 of 8 filesystem-write builtins through — now gated

Round 7 re-ran the round-6 methodology (compare every dangerous-capability
builtin's profile-path check against its CLI-flag-path check) against the
**filesystem-write** dimension instead of network. Result: the same bug class
reappeared. `pipe --sandbox <script>` (without `--sandbox-profile`) claims to
restrict "dangerous builtins (exec, tcp, http, ai, fs-write)" — but most
fs-write builtins **ignored the flag** and performed real writes to the host
filesystem.

Scope note: the **registered-profile path** (`strict`/`noexec`/`isolated`/custom
with `fs: "none"` or similar, exercised by
`pkg/object/sandbox_fs_inventory_test.go`) was not affected — it correctly
redirects into the profile's temp jail. The **exec** domain (`exec`,
`proc_start`) was also checked and found already correctly gated on both
paths. The gap was limited to the "`none` profile + sandbox flags" path for
filesystem writes, exactly like round 6 was for network.

---

## The gap — the `if Name != "none"` path forgot the CLI flags, again

The CLI `--sandbox` flag sets `Sandbox.Enabled = true` and leaves
`ActiveProfile` at `"none"` (the "allow everything" profile). The correctly
gated builtins (`file_delete`, `remove_dir`) followed this pattern:

```go
if ActiveProfile.Load().Name != "none" {
    // canonicalWrite via profile
} else if Sandbox.Enabled && !Sandbox.AllowFS {
    return sandboxBlock("... (filesystem write)")
}
```

The second branch (`else if Sandbox …`) was **missing entirely** in six
fs-write builtins — they only checked the profile and let every write through
in the `"none"` path:

- `write_file` (`pkg/object/builtins_io.go`) — the primary write primitive
- `append_file` (`pkg/object/builtins_fs.go`)
- `file_move` (`pkg/object/builtins_fs.go`) — destination path only; source read was already correct
- `file_copy` (`pkg/object/builtins_fs.go`) — destination path only; source read was already correct
- `make_dir` (`pkg/object/builtins_fs.go`)
- `file_open` in write-capable modes `w`/`a`/`rw`/`rw+` (`pkg/object/file_io.go`) — read mode (`r`) was unaffected by design

Only `file_delete` and `remove_dir` had the CLI-flag branch already in place.
A red-teamed script could therefore write, append to, move, copy, or create
files/directories anywhere the OS permissions allowed — and open a file
handle for arbitrary writes — under `pipe --sandbox`, exactly the write
surface the sandbox mode promises to stop.

---

## Fix (commit `c928178`)

- **Central helper** `checkFSWriteAccess(feature, path)` in
  `pkg/object/builtins_sandbox.go`: wraps both branches (profile
  `canonicalWrite` **and** `Sandbox.Enabled && !Sandbox.AllowFS`) in one
  place, mirroring `checkNetworkAccess` from round 6.
- All eight fs-write builtins now use it: `write_file`, `append_file`,
  `file_delete`, `file_move` (destination), `file_copy` (destination),
  `make_dir`, `remove_dir`, `file_open` (write-capable modes). Read-only
  builtins (`read_file`, `read_lines`, `file_exists`, `file_size`,
  `file_type`, `list_dir`, `file_open` mode `r`) are unchanged — by design
  the CLI flag only restricts writes, not reads.
- The "forgotten `AllowFS` branch" bug class is now structurally
  unreproducible for filesystem writes: a future builtin doing fs writes
  only has to call `checkFSWriteAccess`.

### Regression tests (`pkg/object/fs_write_gate_test.go`)

| Test | covers |
|------|--------|
| `TestFSWriteBuiltinsBlockedBySandboxFlag` | all 9 write-mode call variants (8 builtins, `file_open` counted twice for `w`/`a`) blocked under `Sandbox.Enabled=true, AllowFS=false`, and the temp dir stays empty afterward (no fs side effects from a blocked call) |
| `TestFSWriteBuiltinsAllowedBySandboxFlag` | `write_file` succeeds and produces the expected content under `AllowFS=true` |
| `TestFileOpenReadModeUnaffectedBySandboxFlag` | `file_open` in `"r"` mode stays ungated by the CLI flag, matching `read_file`/`read_lines`/`list_dir` |

---

## Live verification (CLI, no privileged filesystem needed)

```text
$ printf 'write_file "escape.txt" "exfil"' > wf.pipe

$ timeout 20 pipe --sandbox wf.pipe
Runtime error: wf.pipe: SANDBOX: write_file (filesystem write) is disabled in sandbox mode
  in fn(write_file)
exit=1
$ ls escape.txt
ls: cannot access 'escape.txt': No such file or directory      # blocked before any fs I/O ✓

$ timeout 20 pipe wf.pipe                # control without --sandbox:
exit=0
$ ls -la escape.txt
-rw-r--r-- 1 root root 5 ... escape.txt                         # write reaches disk → block is sandbox-specific ✓
```

The block happens **before** any filesystem I/O (no `os.WriteFile` call), as
shown by the control group actually creating the file.

---

## Assessment (honest)

- No known escape in **AI egress** (central gate, round 5), the **network
  gate** (central helper, round 6), or the **profile path** (ratchet, round 2;
  fs-write redirect inventory, `sandbox_fs_inventory_test.go`).
- **exec domain checked and clean**: both `exec` and `proc_start` already had
  the CLI-flag branch on both code paths — no fix needed there.
- Found and fixed: the `--sandbox` flag path let 6 of 8 fs-write builtins
  through, most notably `write_file`. Both directions (blocked and allowed)
  and the read/write boundary are covered by tests.
- Remaining known limitations: unchanged from rounds 4/5/6 (global
  `ActiveProfile`, layer-2 without redirect re-check — n/a here since this
  round is fs-only).
- Methodology note: this finding was located by systematically re-applying
  the round-6 audit method to a different capability dimension, not by
  red-team LLM interaction. It suggests future rounds should keep sweeping
  each remaining dangerous-capability dimension (e.g. `ai` builtins beyond
  the central egress gate) the same way rather than assuming a fix in one
  dimension implies the pattern was fixed everywhere.
