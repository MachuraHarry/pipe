[lang:en]# 🔌 Your First MCP Server — in a Language, Not an SDK[/lang]
[lang:de]# 🔌 Dein erster MCP-Server — in einer Sprache, nicht in einem SDK[/lang]

[lang:en]
**Expose a plain function to Claude, Cursor, or any MCP client — in four lines, with zero dependencies.**

> **Part of the *Pipe in 30 Lines* series:** [RAG without a vector DB](tutorial-local-rag.html) · [Self-healing code](tutorial-self-healing.html) · [Parallel LLM calls](tutorial-parallel.html)

Building an MCP server usually means picking an SDK, wiring up JSON-RPC plumbing, and shipping a runtime with dependencies. Pipe is the first language with **built-in MCP** — `ai_tool` registers a function, `mcp_server` + `mcp_serve_stdio` start the server.

```pipe
fn greet name
    "Hello, " ++ name ++ "! Pipe speaks MCP natively."

ai_tool "greet" "Greet a person by name" {name: "Person's name"} greet

mcp_server "Pipe Greeting Server" "1.0.0"
mcp_serve_stdio
```

Point Claude Desktop (or Cursor, or any MCP client) at it:

```json
{
  "mcpServers": {
    "pipe-greet": {
      "command": "/path/to/pipe",
      "args": ["examples/blog_mcp_server.pipe"]
    }
  }
}
```

What happens here:

- **`ai_tool`** turns any Pipe function into a schema'd tool — name, description, and an argument map are all it needs.
- **`mcp_server` + `mcp_serve_stdio`** start the server and serve the tool over stdio.
- The same binary that *serves* MCP can also *consume* it: `mcp_use_stdio` connects to any MCP server off npm/uvx, so your LLM gets GitHub, filesystem, or database tools in the same pipeline.

There's no SDK, no `package.json`, no build step. A single ~8 MB binary is both server and client — the tool you expose to Claude is the same language you write your pipeline in. That's the whole idea behind the [MCP Cell](mcp-cell.html): when MCP is a language primitive, the sandbox can wrap the server, the client, and the tools.

Run it yourself: `pipe examples/blog_mcp_server.pipe` (it starts a stdio server — hit it with `initialize`, `tools/list`, then `tools/call` with `{"name":"greet","arguments":{"name":"Harry"}}`).
[/lang]

[lang:de]
**Eine normale Funktion für Claude, Cursor oder jeden MCP-Client bereitstellen — in vier Zeilen, mit null Abhängigkeiten.**

> **Teil der Serie *Pipe in 30 Lines*:** [RAG ohne Vektor-DB](tutorial-local-rag.html) · [Selbstheilender Code](tutorial-self-healing.html) · [Parallele LLM-Calls](tutorial-parallel.html)

Einen MCP-Server zu bauen heißt sonst: SDK auswählen, JSON-RPC-Verkabelung, und eine Runtime mit Dependencies ausliefern. Pipe ist die erste Sprache mit **eingebautem MCP** — `ai_tool` registriert eine Funktion, `mcp_server` + `mcp_serve_stdio` starten den Server.

```pipe
fn greet name
    "Hello, " ++ name ++ "! Pipe speaks MCP natively."

ai_tool "greet" "Greet a person by name" {name: "Person's name"} greet

mcp_server "Pipe Greeting Server" "1.0.0"
mcp_serve_stdio
```

Claude Desktop (oder Cursor, oder jeder MCP-Client) zeigt darauf:

```json
{
  "mcpServers": {
    "pipe-greet": {
      "command": "/path/to/pipe",
      "args": ["examples/blog_mcp_server.pipe"]
    }
  }
}
```

Was hier passiert:

- **`ai_tool`** macht aus jeder Pipe-Funktion ein Tool mit Schema — Name, Beschreibung und Argument-Map reichen.
- **`mcp_server` + `mcp_serve_stdio`** starten den Server und servieren das Tool über stdio.
- Dieselbe Binary, die MCP *serviert*, kann es auch *konsumieren*: `mcp_use_stdio` verbindet sich mit jedem MCP-Server von npm/uvx — dein LLM bekommt GitHub-, Dateisystem- oder Datenbank-Tools in derselben Pipeline.

Kein SDK, kein `package.json`, kein Build-Schritt. Eine einzelne ~8-MB-Binary ist Server *und* Client — das Tool, das du Claude gibst, ist dieselbe Sprache, in der du deine Pipeline schreibst. Genau das ist die Idee hinter der [MCP-Zelle](mcp-cell.html): Wenn MCP ein Sprach-Primitiv ist, kann die Sandbox Server, Client und Tools gemeinsam umschließen.

Selbst ausprobieren: `pipe examples/blog_mcp_server.pipe` (startet einen stdio-Server — teste mit `initialize`, `tools/list`, dann `tools/call` mit `{"name":"greet","arguments":{"name":"Harry"}}`).
[/lang]
