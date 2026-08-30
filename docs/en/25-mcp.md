# 25. MCP (Model Context Protocol)

> **STATUS: Production-ready — built into Pipe v1.0.0.**
> E2E-verified: MCP Server + Client with JSON-RPC 2.0 over stdio AND Streamable HTTP (SSE).

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
| `mcp_use_stdio(command, args..., env?, alias?)` | Spawns a subprocess and connects via stdio. Discovers tools and registers them in the tool registry with a `mcp0_`, `mcp1_`, ... prefix, or a custom prefix when `alias` is given. |
| `mcp_use_sse(url, alias?)` | Connects to a Streamable HTTP MCP server via POST + SSE (session-managed), optionally with a custom `alias` prefix. |

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

Instead of the default registration-order prefix, you can give a connection an explicit alias so its tools get a stable, meaningful name:

```pipe
mcp_use_stdio "npx" "-y" "@modelcontextprotocol/server-github" {GITHUB_TOKEN: (env "GITHUB_TOKEN")} "github"
mcp_use_sse "http://localhost:9090/mcp" "postgres"

--- github_list_issues, postgres_query, ...
```

The alias must be a valid identifier fragment and must not collide with another client's prefix.

### Deterministic invocation without an LLM (`tool_call`)

Discovered MCP tools land in the same tool registry as local `ai_tool`
functions, so they can also be called **without** `ai_with_tools` — directly
by name, via `tool_call(name, args?)`. This is useful when your own
deterministic control flow (e.g. a task executor with a fixed plan library)
needs an MCP tool as a validated action, without querying an LLM on every
step:

```pipe
mcp_use_stdio "npx" "-y" "@modelcontextprotocol/server-filesystem" "/tmp" {} "fs"

-- no LLM call, no ai_with_tools needed:
content: tool_call "fs_read_text_file" {path: "/tmp/note.txt"}
print content
```

`tool_call` is subject to the same sandbox gates (`max_tool_calls`,
`audit_log`) as `ai_with_tools`, since both run through the same dispatch
internally. See [Chapter 19.8](19-ai-builtins.md#198-tool-calling-function-calling)
for details.

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

---

## 25.7 The `pipe-docs` Server

The repository ships a ready-made RAG MCP server, `examples/pipe_docs_server.pipe`, published to the MCP registry as `io.github.MachuraHarry/pipe-docs`. It clones the Pipe repository on first run, indexes the documentation (EN + DE + blog) with `docs-pipe`, and builds a declaration-level symbol index over the Go and Pipe source. AI agents can then ask about Pipe without reading the repository themselves.

**Tools:**

| Tool | Needs key | Description |
|------|-----------|-------------|
| `search_docs(query)` | yes | Hybrid keyword + semantic search over docs and blog, with citations |
| `ask_docs(question)` | yes | Cited RAG answer grounded in the documentation |
| `read_doc(path)` | no | Read a documentation file (e.g. `docs/en/25-mcp.md`) |
| `list_docs()` | no | List documentation files (en, de, blog) |
| `search_code(query)` | no | Find Go/Pipe functions, types, structs, enums by name or keyword |
| `read_source(path)` | no | Read a source file with line numbers (e.g. `pkg/mcp/client.go`) |
| `list_sources()` | no | List all source files |
| `index_status()` | no | Index statistics (files, symbols, docs chunks) |
| `refresh_index()` | no | Re-fetch the repo and rebuild the indexes |

**Configuration (env):**

- `DEEPSEEK_API_KEY` / `OPENAI_API_KEY` — enables `search_docs` and `ask_docs` (embeddings + chat). Without a key, the file/source tools still work.
- `PIPE_DOCS_CACHE` — cache directory (default `~/.pipe/cache/pipe-docs`).

The first run clones the repo and embeds the docs index (a few minutes); subsequent runs load the persistent SQLite index. Register it in your MCP client with the desired API key:

```json
{
  "mcpServers": {
    "pipe-docs": {
      "command": "pipe",
      "args": ["examples/pipe_docs_server.pipe"],
      "env": { "DEEPSEEK_API_KEY": "sk-..." }
    }
  }
}
```

The server is distributed as an `.mcpb` bundle built by `release.yml` and published via `publish-mcp-docs.yml`.

## 25.8 The `repo-rag` Server

While `pipe-docs` is hard-wired to the Pipe repository, `examples/repo_rag_server.pipe` brings the same experience to **any** Git repository: point it at a URL and it clones, indexes and serves the codebase as an MCP server over stdio.

**Run it:**

```bash
export REPO_RAG_URL="https://github.com/your-user/your-repo"
pipe examples/repo_rag_server.pipe
```

**Tools:**

| Tool | Needs key | Description |
|------|-----------|-------------|
| `search_docs(query)` | optional | Markdown search — semantic hybrid with a chat/embeddings provider, keyword-only without any key |
| `ask_docs(question)` | yes | Cited RAG answer; falls back to the keyword chunk index when semantic retrieval is unavailable |
| `read_doc(path)` / `list_docs()` | no | Read or list Markdown files |
| `search_code(query)` | no | Find declarations in Go, Pipe, Python, JS/TS, Rust + generic fallback |
| `file_symbols(path)` | no | Outline of one source file: every indexed declaration with kind, name, line and declaration text |
| `read_source(path, offset)` | no | Source view with line numbers, paginated |
| `list_sources()` | no | All recognized source files |
| `repo_info()` / `index_status()` | no | Repository metadata / index statistics and last sync counts |
| `refresh_index()` | no | Incremental re-sync from the cached checkout |

**Configuration (env):**

- `REPO_RAG_URL` (required) — repository to clone and index.
- `REPO_RAG_REF` — optional branch or tag.
- `REPO_RAG_CACHE` — cache directory (default `~/.pipe/cache/repo-rag/<owner>__<repo>`).
- `DEEPSEEK_API_KEY`, `OPENAI_API_KEY` or `OPENROUTER_API_KEY` — enables `ask_docs`; with DeepSeek/OpenAI also the semantic hybrid search.
- `REPO_RAG_MODEL` — model override for OpenRouter (default `nvidia/nemotron-3-super-120b-a12b:free`). OpenRouter exposes no embeddings endpoint, so `search_docs` runs in keyword mode there.

Indexes persist across restarts (`code.db`, `docs-kw.db`, `docs.db`): files are re-hashed via SHA-256 and only rescanned on change, so warm starts are near-instant. A ref change triggers a fresh clone and index wipe automatically.

After indexing the server switches from the `rag-build` profile (clone/exec/network for Git hosts) to the locked `rag-serve` profile: read-only filesystem, no exec, network restricted to the configured AI providers. Repository URLs are validated against a character allowlist before reaching any shell command.

