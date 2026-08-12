# 22. Sandbox Profiles

Pipe provides a **declarative sandbox profile system** to restrict what builtins can do at runtime. This is essential when running AI agents with `ai_with_tools` or executing untrusted Pipe code.

---

## 22.1 Quick Start

```pipe
-- Use a predefined profile from the CLI
-- pipe script.pipe --sandbox-profile strict
```

```pipe
-- Or define your own profile in code
sandbox_profile "strict" {fs: "read-only", network: false, exec: false, ai: false}

set_sandbox "strict"

-- E_SANDBOX: blocked
write_file "/tmp/test.txt" "blocked"
-- allowed (read-only)
read_file "/tmp/test.txt"
-- E_SANDBOX: blocked
http_get "https://example.com"
```

---

## 22.2 Why Sandbox Profiles?

When `ai_with_tools` gives an LLM access to tools like `exec`, `http_get`, `write_file`, or `file_delete`, there is **no guarantee** the LLM will only do what you intended. A hallucinated tool call could:

- Delete files via `file_delete` / `remove_dir`
- Execute arbitrary shell commands via `exec`
- Exfiltrate data via `http_post`
- Overwrite configuration files via `write_file`

Sandbox profiles solve this by **declaring upfront** what a script or agent is allowed to do. Violations produce a runtime error instead of executing the operation.

---

## 22.3 Built-in Profiles

| Profile | FS Access | Network | Exec | AI | Use Case |
|---------|-----------|---------|------|----|----------|
| `none` | full | yes | yes | yes | Unrestricted (default) |
| `strict` | read-only | no | no | no | Read-only analysis, CI/CD |
| `noexec` | full | no | no | no | File processing without shell |
| `isolated` | none | no | no | no | Totally restricted |
| `networked` | temp-only | yes | no | yes | AI agents with network access |

### CLI Usage

```bash
pipe script.pipe --sandbox-profile strict
pipe my_agent.pipe --sandbox-profile networked
```

---

## 22.4 Custom Profiles

Define profiles with the `sandbox_profile` builtin. The config is a map with these keys:

| Key | Type | Description |
|-----|------|-------------|
| `fs` | string | `"none"`, `"read-only"`, `"temp-only"`, `"full"` |
| `network` | bool | Allow HTTP/TCP operations |
| `exec` | bool | Allow shell command execution |
| `ai` | bool | Allow AI chat/embedding calls |
| `timeout` | int | Max seconds per `exec`, `tcp_connect`/`tcp_read` or `sleep` operation (0 = no limit) |
| `env` | map | Environment variables injected into `exec` and returned by `env`. Under an active profile the real process environment is never exposed |
| `work_dir` | string | Working directory for sandbox |
| `budget` | num | Max AI spend in USD (0 = unlimited). A conservative cost estimate is checked before each call is issued, so a single call cannot blow through the budget |
| `network_whitelist` | list | List of hosts or domains (optionally `host:port` or `host/path`). When set, HTTP and TCP targets must match one entry exactly or as a subdomain. Substring matching is **not** used |
| `max_tool_calls` | int | Max number of tool executions per `ai_with_tools` session (0 = unlimited) |
| `audit_log` | bool | Record all sandbox events (http, AI calls, tool calls) into an audit trail |

### Examples

```pipe
-- Minimal: read-only log analysis
sandbox_profile "log-analyzer" {fs: "read-only", network: false, exec: false, ai: false}

-- Networked AI agent with temp storage
sandbox_profile "web-agent" {fs: "temp-only", network: true, exec: false, ai: true}

-- Full access but no network (safe for local scripts)
sandbox_profile "local-only" {fs: "full", network: false, exec: true, ai: false}

-- Totally locked down
sandbox_profile "prison" {fs: "none", network: false, exec: false, ai: false}

-- AI agent with a $0.10 budget, restricted network, and full auditing
sandbox_profile "guarded-agent"
    {fs: "read-only", network: true, network_whitelist: ["api.github.com", "api.openai.com"],
     exec: false, ai: true, budget: 0.1, max_tool_calls: 10, audit_log: true}
```

---

## 22.5 Profile Management

### `sandbox_profile` — Define a Profile

```pipe
sandbox_profile "name" {fs: "...", network: bool, exec: bool, ai: bool}
```

Registers a named profile in the global registry. Returns an error if the name already exists.

### `set_sandbox` — Activate a Profile

```pipe
-- All following operations use this profile
set_sandbox "strict"
```

Changes the active profile globally. All subsequent builtin calls are checked
against this profile.

> **Ratchet:** Once a non-`none` profile is active, the sandbox can **only
> ratchet down**. Switching to a profile that is *more* permissive than the
> active one — including back to `none` — is rejected with an `E_SANDBOX`
> error. A target profile is considered a subset (permitted) if it grants no
> more than the active profile across `fs`, `network` (incl. whitelist), `exec`
> and `ai`.
>
> **Locking:** When the sandbox was started with the `--sandbox-profile` CLI
> flag (or the `sandbox_lock` builtin was used), sandboxed code can **not**
> switch back to profile `none` at all. This prevents untrusted code from
> disabling its own sandbox.
>
> **Registration is ratcheted too:** while a restricted profile is active,
> `sandbox_profile` refuses to register a profile that is more permissive than
> the active one, so sandboxed code cannot simply mint its own escape hatch.
> Define all profiles up front, before entering a restricted profile.

### `sandbox_lock` — Freeze the Active Profile

```pipe
sandbox_lock                -- lock the active profile
sandbox_lock "strict"       -- lock a specific profile
```

Once a profile is locked, `set_sandbox` and `with_sandbox` are rejected while it
is active. This is the strongest guarantee: even the script itself can no longer
change its confinement.

### `with_sandbox` — Temporary Profile

```pipe
fn do_risky_work
    exec "dangerous command"
    http_get "https://unknown-site.com"

-- blocked by sandbox
with_sandbox "strict" do_risky_work
```

Runs a function under a specific profile, then restores the previous one.

---

## 22.6 FS Access Levels

| Level | Read | Write | Delete |
|-------|------|-------|--------|
| `none` | blocked | blocked | blocked |
| `read-only` | allowed | blocked | blocked |
| `temp-only` | temp dir only | temp dir only | temp dir only |
| `full` | allowed | allowed | allowed |

With `temp-only`, file operations on paths outside the sandbox are
automatically redirected into a `.pipe_sandbox` directory under the working
directory. Paths are canonicalized first (absolute, symlink-resolved, `..`-safe),
so nothing outside the sandbox directory can be read or written.

---

## 22.7 Affected Builtins

The following builtins check the active sandbox profile:

**File System:** `read_file`, `read_lines`, `write_file`, `append_file`, `file_delete`, `file_move`, `file_copy`, `file_size`, `file_type`, `file_exists`, `list_dir`, `make_dir`, `remove_dir`, `file_open`

**Execution:** `exec`

**Network:** `http_get`, `http_post`, `http_request`, `http_stream_open`, `tcp_listen`, `tcp_connect`

**Environment:** `env` (returns only the profile's injected environment under an active profile; secret variable names such as `*KEY*`/`*TOKEN*`/`*SECRET*` are always blocked)

**AI:** `ai_chat`, `ai_chat_json`, `ai_stream`, `ai_with_tools`, `summarize`, `translate`, `classify`, `extract`, `generate`, `ask`, `ai_parallel`, `ai_batch`, `embed`, `embed_batch`

---

## 22.8 Integration with `ai_with_tools`

When using `ai_with_tools`, the sandbox profile applies to **both** the AI API call and the tool execution:

```pipe
sandbox_profile "agent-safe" {fs: "read-only", network: true, exec: false, ai: true}

fn get_weather city
    http_get ("https://api.weather.com/" ++ city)
    -- sandbox allows network -> OK

fn delete_logs pattern
    file_delete "/var/log/" ++ pattern
    -- sandbox blocks write -> error returned to LLM

ai_tool "get_weather" "Get weather" {city: "city"} get_weather
ai_tool "delete_logs" "Delete log files" {pattern: "glob"} delete_logs

set_sandbox "agent-safe"
ai_with_tools "You are a helpful assistant" "What's the weather and clean up old logs?"
```

The LLM can call `get_weather` successfully, but `delete_logs` returns a sandbox error. The LLM sees the error and can adapt its behavior.

---

## 22.9 Budget, Network Whitelist & Audit Log

### Budget Enforcement (`budget`, `budget_spent`)

The `budget` key caps the **total AI spend** of a profile in USD. Every AI call
records its cost into the active profile. Additionally, a conservative cost
estimate (prompt buffer + completion allowance for the current model) is
checked **before** each call is issued: a single call cannot push the profile
past its budget. Once the accumulated cost reaches the budget, further AI calls
are blocked with an `E_SANDBOX: budget exceeded` error.

```pipe
sandbox_profile "agent" {fs: "full", network: false, exec: false, ai: true, budget: 0.01}
set_sandbox "agent"

print (ask "Hello")                -- first call: allowed
print (budget_spent)               -- e.g. 0.000079

-- Later, once the accumulated cost reaches $0.01:
try
    print (ask "Still working?")
catch e
    print e.message
    -- -> E_SANDBOX: budget exceeded (0.0100 USD) in profile 'agent'
```

`budget_spent` returns the total cost recorded for the active profile.

### Network Whitelist (`network_whitelist`)

`network_whitelist` restricts network targets (HTTP **and** TCP) to matching
hosts. Each entry may be:

- a bare domain (`api.github.com`) — matches that host and its subdomains,
- a `host:port` pair (`api.github.com:443`) — host and port must match,
- a `host/path-prefix` (`api.github.com/repos`) — host must match and the path
  must start with the prefix.

Matching is host-based, **not** substring-based: `api.github.com` does not match
`attacker.com?x=api.github.com`. Redirect hops of `http_get`/`http_post`/
`http_request`/`http_stream_open` are re-checked against the whitelist.

```pipe
sandbox_profile "web-agent" {fs: "full", network: true, network_whitelist: ["api.github.com", "openai.com"], exec: false, ai: true}
set_sandbox "web-agent"

http_get "https://api.github.com/repos/MachuraHarry/pipe"   -- allowed
http_get "https://api.github.com/repos/MachuraHarry/pipe" > redirect to "https://internal.example.com"  -- blocked
-- http_get "https://example.com"                           -- E_SANDBOX: not in whitelist
```

### Tool Call Limits (`max_tool_calls`)

`max_tool_calls` limits how many times an LLM may invoke tools during an
`ai_with_tools` session. When the limit is reached, further tool executions are
blocked with an `E_SANDBOX` error, which the LLM receives as a tool result.

```pipe
sandbox_profile "agent" {fs: "full", network: true, exec: false, ai: true, max_tool_calls: 3}
```

### Audit Log (`audit_log`, `audit_log()`)

With `audit_log: true`, the profile records every security-relevant event:
`http_get` / `http_post` requests, `ai_call` events (provider, model, tokens,
cost, cache status), and `tool_call` executions.

```pipe
sandbox_profile "audited" {fs: "read-only", network: true, exec: false, ai: true, audit_log: true}
set_sandbox "audited"

http_get "https://example.com"
ask "Hello" > print

for entry in audit_log
    print entry.time ++ " | " ++ entry.event ++ " | " ++ entry.detail
-- -> ... | http_get | https://example.com
-- -> ... | ai_call | provider=deepseek model=deepseek-chat tokens=50 cost=0.000045 cached=false
-- -> ... | tool_call | my_tool
```

Each audit entry is a map with the fields `time` (RFC 3339), `event`,
`detail`, and `profile`.

---

## 22.10 Backward Compatibility

The old `--sandbox` / `--allow-ai` CLI flags continue to work. When no profile is active (`"none"`), the old sandbox flags are checked as before. Custom profiles take priority over the legacy system when `ActiveProfile.Name != "none"`.

Starting a run with `--sandbox-profile <name>` also **locks** the sandbox: sandboxed code cannot switch back to the unrestricted `none` profile (see 22.5).
