# 🧱 The MCP Cell — an AI That Can Only Talk to Pipe's Own MCP Server

**The smallest demo of the strongest idea: a sandboxed AI agent whose entire
universe is Pipe's own MCP server. No free network, no free filesystem, no
shell — just five tools behind a whitelist.**

---

Everyone asks the same question about sandboxes: *"But if the agent has tools,
aren't the tools the attack surface?"*

Fair. So this time I flipped it. Instead of giving the agent a toolbox and
hoping the sandbox holds, I built a profile where the **only** things the
agent can do are the MCP tools that *I* chose to expose — and nothing else.
The AI's entire world is one MCP server, and that server lives inside the
sandbox too.

The example lives in [`examples/mcp_sandbox_agent.pipe`](https://github.com/MachuraHarry/pipe/blob/master/examples/mcp_sandbox_agent.pipe).
It has two modes: an in-process agent mode (`agent`) and a real MCP server on
stdio (`serve`) that external clients like Claude Desktop or Cursor can plug
into. Both modes give the exact same five tools.

---

## The Cell 🏗️

The profile is the wall:

```pipe
sandbox_profile "mcp-cell" {
    fs:                "temp-only",       # every write lands in ./.pipe_sandbox
    network:           true,
    network_whitelist: [provider_host],   # the ONLY network target is the AI provider
    exec:              false,             # no shell, period
    ai:                true,
    budget:            0.5,
    max_tool_calls:    25,
    audit_log:         true,
    timeout:           30
}
```

Inside that wall the agent can call exactly five tools, defined with ordinary
Pipe functions and registered with `ai_tool`:

| Tool | What it does |
|------|--------------|
| `sb_write` | Writes a file — redirected into `.pipe_sandbox` |
| `sb_read` | Reads a file from the sandbox |
| `sb_list` | Lists the sandbox directory |
| `sb_note` | Stores an in-memory note (a plain Pipe map) |
| `sb_ping` | Liveness check |

```pipe
fn sb_note key value
    set __cell_notes key value
    "Notiz gesetzt: " ++ key ++ " = " ++ value

ai_tool "sb_note" "Legt eine In-Memory-Notiz an" {key: "Schluessel", value: "Wert"} sb_note
```

Then the sandbox is locked in with `set_sandbox "mcp-cell"`, and the server is
started. Because `mcp_server` bridges whatever `ai_tool` registered, the
client-facing tool list is *the same five functions* — and every single
`tools/call` executes under the active profile.

---

## What the Agent Actually Did 🤖

In `agent` mode the task was simple: create three notes, write a summary file,
list the sandbox. DeepSeek did it in six tool calls:

```
tool_call | sb_note
tool_call | sb_note
tool_call | sb_note
tool_call | sb_write
tool_call | sb_list
```

That's the whole audit log. No `exec`, no foreign `http_get`, no writes
outside the cell. The agent *couldn't* have done more even if it had tried —
the profile blocks it at the builtin level, not at the prompt level.

---

## The Real Proof: Trying to Break Out 💥

A demo that only shows success proves nothing. So I tried to escape, using the
same builtins an attacker would reach for inside the cell:

| Attempt | Result |
|---------|--------|
| `exec "id"` | `E_SANDBOX: exec blocked by profile 'mcp-cell'` |
| `http_get "https://www.google.com"` | `E_SANDBOX: network target not in whitelist` |
| `write_file "/tmp/escape.txt"` | silently redirected to `.pipe_sandbox/escape.txt` |

The third one is the interesting case: temp-only **doesn't fail**, it
*redirects*. The agent writes wherever it wants and believes it wrote to
`/tmp/escape.txt` — the file actually lands inside the cell. The attacker gets
a consistent illusion, and the host stays clean.

---

## One Real Bug Found Along the Way 🐛

Building this demo surfaced an actual bug in Pipe. With `fs: "temp-only"` and
the default `workingDir: "."`, listing the sandbox directory broke:

```
list_dir: open /tmp/.pipe_sandbox/tmp: no such file or directory
```

The `filepath.Rel` call in the redirect logic can't relate an absolute path
to a *relative* base — so `list_dir "."` resolved to the wrong place. The fix
was to make the profile's working directory absolute everywhere
([`sandbox.go`](https://github.com/MachuraHarry/pipe/blob/master/pkg/object/sandbox.go)):

```go
func currentDir() string {
    d, err := os.Getwd()
    if err != nil || d == "" {
        return "."
    }
    return d
}
```

Now every profile starts with an absolute working directory, and temp-only
redirects behave correctly — including `list_dir "."`.

---

## Try It Yourself 🚀

```bash
# DeepSeek
DEEPSEEK_API_KEY=sk-... pipe examples/mcp_sandbox_agent.pipe agent

# local, no key needed
pipe examples/mcp_sandbox_agent.pipe serve ollama
```

Then point Claude Desktop or Cursor at the stdio server:

```json
{
  "mcpServers": {
    "pipe-cell": {
      "command": "pipe",
      "args": ["examples/mcp_sandbox_agent.pipe", "serve"]
    }
  }
}
```

The same `mcp-cell` profile wraps both the in-process agent and every external
client call. One sandbox, two entry points, same guarantee.
