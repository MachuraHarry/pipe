# 25. MCP (Model Context Protocol)

> **STATUS: IN DEVELOPMENT — NOT yet published to the pipe-modules registry.**
> E2E-verifiziert: MCP Server + Client mit JSON-RPC 2.0 über stdio.
> HTTP/SSE-Transport in Arbeit.

Pipe implements the **[Model Context Protocol (MCP)](https://modelcontextprotocol.io)** — both as a Server (to expose tools to external clients like Claude Desktop) and as a Client (to consume external MCP servers in `ai_with_tools`). The implementation is **zero-dependency**, using only Go's standard library.

---

## 25.1 Concepts

MCP uses **JSON-RPC 2.0** over two transports:

| Transport | Use Case |
|-----------|----------|
| **stdio** | Subprocess pipe (Claude Desktop, CLI tools). Pipe reads stdin, writes stdout. |
| **Streamable HTTP** | Network-based (POST + SSE). *Planned, not yet implemented.* |

The protocol defines three primitives:
- **Tools** — Functions the AI model can call (`tools/list`, `tools/call`)
- **Resources** — Data exposed to the model (not yet implemented in Pipe)
- **Prompts** — Reusable templates (not yet implemented in Pipe)

---

## 25.2 MCP Server — Expose Pipe Tools

Pipe can act as an MCP Server, exposing all `ai_tool`-registered functions to any MCP-compatible client.

### Builtins

| Builtin | Description |
|---------|-------------|
| `mcp_server(name, version)` | Creates an MCP server. Automatically bridges all `ai_tool` entries as MCP tools. |
| `mcp_serve_stdio` | Starts the server on stdin/stdout (blocking). For Claude Desktop, Cursor, etc. |
| `mcp_serve_sse(addr)` | *(Not yet implemented)* Starts an HTTP/SSE server. |
| `mcp_tools` | Lists all registered tools (local + remote). |

### Example: Claude Desktop Integration

```pipe
--- agent.pipe — Run as an MCP server for Claude Desktop

fn get_weather city
    match city
        | "Berlin" -> "Berlin: 22°C, sunny"
        | "London" -> "London: 15°C, rainy"
        | _ -> city ++ ": no data"

fn search_docs query
    "Found results for: " ++ query

ai_tool "get_weather" "Get current weather for a city" {city: "City name"} get_weather
ai_tool "search_docs" "Search documentation" {query: "Search term"} search_docs

mcp_server "Pipe Agent" "1.0.0"
mcp_serve_stdio
```

Then in `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "pipe-agent": {
      "command": "/tmp/pipe",
      "args": ["/path/to/agent.pipe"]
    }
  }
}
```

Claude Desktop will automatically discover `get_weather` and `search_docs` as available tools.

### E2E-Verified MCP Flow

```
Client                  Pipe (MCP Server)
  │                         │
  ├─ initialize ──────────>│   → returns protocol version, capabilities
  ├─ notifications/init'ed─>│   (no response)
  ├─ tools/list ───────────>│   → returns tool schemas (JSON Schema)
  ├─ tools/call ───────────>│   → runs Pipe function, returns result
  │                         │
```

---

## 25.3 MCP Client — Consume External Tools

Pipe can connect to external MCP servers and use their tools in `ai_with_tools`.

### Builtins

| Builtin | Description |
|---------|-------------|
| `mcp_use_stdio(command, args...)` | Spawns a subprocess and connects via stdio. Discovers tools and registers them in the tool registry with a `mcp0_`, `mcp1_`, ... prefix. |
| `mcp_use_sse(url)` | Connects to a Streamable HTTP MCP server via POST. |

Discovered tools are automatically registered in Pipe's tool registry, making them available to `ai_with_tools` alongside locally defined tools.

### Example: Using a Remote MCP Server

```pipe
ai_provider "deepseek"
ai_set_key "deepseek" (env "DEEPSEEK_API_KEY")

--- Connect to an external MCP filesystem server
mcp_use_stdio "npx" "-y" "@modelcontextprotocol/server-filesystem" "/tmp"

--- Now use its tools via ai_with_tools
result: ai_with_tools
    "You manage a filesystem. Use available tools to answer questions."
    "List all files in /tmp and tell me which is the largest."
print result
```

### Tool Naming

Remote tools get a prefix to avoid name collisions between multiple servers:

| Connection | Prefix | Example |
|------------|--------|---------|
| 1st `mcp_use_stdio` | `mcp0_` | `mcp0_read_file` |
| 2nd `mcp_use_stdio` | `mcp1_` | `mcp1_search` |
| 1st `mcp_use_sse` | `mcp2_` | `mcp2_query_db` |

Local `ai_tool` tools keep their original names (no prefix).

---

## 25.4 Combined: Server + Client

Pipe can serve as an MCP hub — both exposing its own tools AND aggregating tools from external servers:

```pipe
--- hub.pipe — Pipe as an MCP aggregation hub

ai_provider "deepseek"

--- Own tools
fn local_query q
    "Local result for: " ++ q

ai_tool "local_query" "Search local knowledge base" {q: "Query"} local_query

--- Connect to external MCP servers
mcp_use_stdio "node" "./filesystem-server.js"
mcp_use_stdio "python" "./database-server.py"

--- Expose everything (own + remote tools) to external clients
mcp_server "Pipe MCP Hub" "1.0.0"
mcp_serve_stdio
```

Claude Desktop sees: `local_query`, `mcp0_read_file`, `mcp0_write_file`, `mcp1_query`, ...

---

## 25.5 Architecture (Zero-Dependency)

All MCP functionality is implemented in pure Go standard library — no external dependencies.

| Package | Responsibility |
|---------|---------------|
| `pkg/mcp/types.go` | JSON-RPC 2.0 and MCP message types |
| `pkg/mcp/server.go` | Server dispatch: `initialize`, `tools/list`, `tools/call` |
| `pkg/mcp/stdio.go` | stdio transport (`bufio.Scanner` + `fmt.Println`) |
| `pkg/mcp/client.go` | Client: `exec.Cmd` (stdio) or `net/http` (HTTP), concurrent-safe |
| `pkg/mcp/schema.go` | JSON Schema converter |
| `pkg/object/builtins_mcp.go` | 7 Pipe builtins + bridge to `ai_tool` registry |

---

## 25.6 Error Handling

All builtins return Pipe's standard `Ok`/`Err` results or plain strings. MCP protocol errors (unknown tools, parse errors) are returned as JSON-RPC error responses to the client.

```pipe
r: mcp_use_stdio "./my-server"
if (type_of r) == "ERROR"
    print "MCP connection failed: " ++ (to_str r)
```
