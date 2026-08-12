# 25. MCP (Model Context Protocol)

> **STATUS: IN DEVELOPMENT — NOT yet published to the pipe-modules registry.**
> E2E-verifiziert: MCP Server + Client mit JSON-RPC 2.0 über stdio UND Streamable HTTP (SSE).

Pipe implements the **[Model Context Protocol (MCP)](https://modelcontextprotocol.io)** — both as a Server (to expose tools to external clients like Claude Desktop) and as a Client (to consume external MCP servers in `ai_with_tools`). The implementation is **zero-dependency**, using only Go's standard library.

---

## 25.1 Concepts

MCP uses **JSON-RPC 2.0** over two transports:

| Transport | Use Case |
|-----------|----------|
| **stdio** | Subprocess pipe (Claude Desktop, CLI tools). Pipe reads stdin, writes stdout. |
| **Streamable HTTP** | Network-based (POST + SSE, session-managed via `Mcp-Session-Id`). |

The protocol defines three primitives:
- **Tools** — Functions the AI model can call (`tools/list`, `tools/call`)
- **Resources** — Data exposed to the model (`resources/list`, `resources/read`, incl. URI templates)
- **Prompts** — Reusable message templates (`prompts/list`, `prompts/get`)

The server negotiates the **protocol version** during `initialize` (latest supported: `2025-11-25`, backwards compatible with `2025-06-18`, `2025-03-26`, `2024-11-05`) and answers `ping` keep-alive requests.

---

## 25.2 MCP Server — Expose Pipe Tools

Pipe can act as an MCP Server, exposing all `ai_tool`-registered functions to any MCP-compatible client.

### Builtins

| Builtin | Description |
|---------|-------------|
| `mcp_server(name, version)` | Creates an MCP server. Automatically bridges all `ai_tool` entries as MCP tools. |
| `mcp_serve_stdio` | Starts the server on stdin/stdout (blocking). For Claude Desktop, Cursor, etc. |
| `mcp_serve_sse(addr)` | Starts a Streamable HTTP server on `addr` (e.g. `:9090`, blocking). Clients connect via `POST` + SSE. |
| `mcp_tools` | Lists all registered tools (local + remote). |
| `mcp_resource(uri, name, mime, read_fn)` | Registers a static resource. `read_fn(uri)` returns the resource text. |
| `mcp_resource_template(template, name, mime, read_fn)` | Registers a URI-template resource, e.g. `file:///{path}`. `read_fn(uri)` is called with the concrete URI. |
| `mcp_prompt(name, description, args_map, build_fn)` | Registers a prompt template. `args_map` maps argument names to a description (or a Map with `description`/`required`). `build_fn(args)` returns the rendered text. |
| `mcp_resources` | Lists all resources (local + remote). |
| `mcp_read_resource(uri)` | Reads a resource from the local server or a connected client. |
| `mcp_prompts` | Lists all prompts (local + remote). |
| `mcp_prompt_get(name, args?)` | Renders a prompt from the local server or a connected client. |

> **Note:** Missing required arguments are rejected with an `isError: true` result; tool names must match the spec pattern `^[a-zA-Z0-9_-]{1,64}$` (invalid names are skipped). Tools returning a Map/List are serialized as pretty JSON so structured data survives the round trip.

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

### Example: Streamable HTTP Server

```pipe
--- http_agent.pipe — Serve tools over HTTP for network clients

fn get_weather city
    match city
        | "Berlin" -> "Berlin: 22°C, sunny"
        | _ -> city ++ ": no data"

ai_tool "get_weather" "Get current weather for a city" {city: "City name"} get_weather

mcp_server "Pipe HTTP Agent" "1.0.0"
mcp_serve_sse ":9090"
```

Clients POST JSON-RPC messages to `http://host:9090/` with
`Accept: application/json, text/event-stream`. The server manages sessions via the
`Mcp-Session-Id` header; `DELETE` terminates a session.

### Resources & Prompts

Resources and prompts are registered with the dedicated builtins below and are
automatically bridged into the server when `mcp_server` is called. The
read/render builtins also work without a running server (they fall back to the
local registries).

```pipe
--- resources.pipe — Serve resources + prompts over MCP

--- Static resource
fn read_docs uri
    "Documentation for " ++ uri ++ "\n\n# Pipe\nZero-dependency language."

--- URI-template resource: any file:///{path} matches
fn read_tmp uri
    path: replace uri "file:///" ""
    content: read_file ("/tmp/" ++ path)
    content

mcp_resource "docs://pipe" "Pipe Docs" "text/markdown" read_docs
mcp_resource_template "file:///{path}" "File" "text/plain" read_tmp

--- Prompt with a required argument
fn build_summary args
    "Please summarize: " ++ (get args "text")

mcp_prompt "summarize" "Summarize the given text" {text: "The text to summarize"} build_summary

--- Inspect without a running server (mcp_read_resource / mcp_prompt_get
--- work standalone; do NOT print() when mcp_serve_stdio is active, as
--- stdout carries the JSON-RPC protocol)
print (mcp_resources)
print (mcp_read_resource "file:///hello.txt")
print (mcp_prompt_get "summarize" {text: "A long article ..."})

mcp_server "Pipe Resources" "1.0.0"
mcp_serve_stdio
```

`args_map` entries may be plain strings (a description) or Maps with a
`description` and an optional `required` boolean (default `true`). Remote
resources/prompts discovered via `mcp_use_*` appear in `mcp_resources` /
`mcp_prompts` and can be read via `mcp_read_resource` / `mcp_prompt_get`.

> A runnable example with tools, resources and prompts is available at
> `examples/mcp_resources.pipe` (run it directly, or configure it as a stdio
> server in Claude Desktop). More combinations: `examples/mcp_server.pipe`,
> `examples/mcp_filesystem.pipe`, `examples/mcp_github.pipe`,
> `examples/mcp_combined.pipe`.

---

## 25.3 MCP Client — Consume External Tools

Pipe can connect to external MCP servers and use their tools in `ai_with_tools`.

### Builtins

| Builtin | Description |
|---------|-------------|
| `mcp_use_stdio(command, args...)` | Spawns a subprocess and connects via stdio. Discovers tools and registers them in the tool registry with a `mcp0_`, `mcp1_`, ... prefix. |
| `mcp_use_sse(url)` | Connects to a Streamable HTTP MCP server via POST + SSE (session-managed). |

> **Sandbox:** MCP client connections are subject to the active sandbox
> profile. `mcp_use_stdio` spawns a subprocess and therefore requires
> `exec: true`; `mcp_use_sse` makes HTTP requests and requires
> `network: true` — the endpoint URL is checked against the profile's
> `network_whitelist`, and so is every subsequent request (including redirect
> targets). Under the `none` profile no sandbox gate applies.

> **Note:** Client calls have a configurable timeout (default 120 s, set via
> `client.SetCallTimeout`); subprocess stderr is captured (last 64 KB) and
> surfaced in error messages. Remote tool arguments support nested Maps/Lists
> (recursively converted to JSON).

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
| `pkg/mcp/types.go` | JSON-RPC 2.0 and MCP message types, protocol versions, error codes |
| `pkg/mcp/server.go` | Server dispatch: `initialize` (version negotiation), `ping`, `tools/list`, `tools/call`, `resources/list` + `resources/read`, `prompts/list` + `prompts/get` |
| `pkg/mcp/stdio.go` | stdio transport (`bufio.Scanner` + `fmt.Println`) |
| `pkg/mcp/http_server.go` | Streamable HTTP server (POST + SSE, session management, `DELETE`) |
| `pkg/mcp/client.go` | Client: stdio (`exec.Cmd`) + HTTP (SSE read loop, session id), call timeouts, stderr capture, concurrent-safe |
| `pkg/mcp/schema.go` | JSON Schema converter |
| `pkg/object/builtins_mcp.go` | 13 Pipe builtins + bridge to the `ai_tool` registry |

---

## 25.6 Error Handling

All builtins return Pipe's standard `Ok`/`Err` results or plain strings. MCP protocol errors (unknown tools, parse errors) are returned as JSON-RPC error responses to the client. Tool-level failures (including missing required arguments or remote tool errors) are returned as `isError: true` results, so clients can distinguish a failed call from a successful one.

```pipe
r: mcp_use_stdio "./my-server"
if (type_of r) == "ERROR"
    print "MCP connection failed: " ++ (to_str r)
```
