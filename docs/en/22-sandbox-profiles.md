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
| `timeout` | int | Max seconds per tool call (0 = no limit) |
| `env` | map | Injected environment variables |
| `work_dir` | string | Working directory for sandbox |

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
-- Reset to unrestricted
set_sandbox "none"
```

Changes the active profile globally. All subsequent builtin calls are checked against this profile.

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

With `temp-only`, all file operations are automatically redirected to a `.pipe_sandbox` directory under the working directory. This prevents any access to files outside the sandbox.

---

## 22.7 Affected Builtins

The following builtins check the active sandbox profile:

**File System:** `read_file`, `write_file`, `append_file`, `file_delete`, `file_move`, `file_copy`, `make_dir`, `remove_dir`

**Execution:** `exec`

**Network:** `http_get`, `http_post`, `tcp_listen`, `tcp_connect`

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

## 22.9 Backward Compatibility

The old `--sandbox` / `--allow-ai` CLI flags continue to work. When no profile is active (`"none"`), the old sandbox flags are checked as before. Custom profiles take priority over the legacy system when `ActiveProfile.Name != "none"`.
