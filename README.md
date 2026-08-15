# <img src="website/logo.svg" width="32" height="32" align="left" style="margin-right:8px"> Pipe — MCP-native runtime for AI infrastructure

[![CI](https://github.com/MachuraHarry/pipe/actions/workflows/ci.yml/badge.svg)](https://github.com/MachuraHarry/pipe/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-purple.svg)](LICENSE)
[![Version](https://img.shields.io/badge/version-v0.9.4.0-blue.svg)](https://github.com/MachuraHarry/pipe/releases)
[![SPR](https://img.shields.io/badge/SPR-Semantic%20Pipeline%20Runtime-7c5cfc.svg)](#)
[![MCP](https://img.shields.io/badge/MCP-Server%20%2B%20Client-3ce096.svg)](#model-context-protocol)
[![GitHub MCP Registry](https://img.shields.io/badge/GitHub_MCP_Registry-Onboarding-6e5494.svg)](https://github.com/github/github-mcp-server/discussions/3057)
[![MCP Registry](https://img.shields.io/badge/MCP_Registry-Listed-4a90d9.svg)](https://registry.modelcontextprotocol.io/?q=MachuraHarry)

> **The first language with built-in MCP — server and client. 226 builtins, single ~8 MB binary. Zero dependencies.**
> **Officially listed in the [official MCP Registry](https://registry.modelcontextprotocol.io/?q=MachuraHarry)** (v0.9.4.0, active). GitHub MCP Registry [onboarding requested](https://github.com/github/github-mcp-server/discussions/3057) for one-click install in GitHub Copilot & VS Code.

## Quick Install

```sh
curl -fsSL https://pipe-lang.com/install.sh | bash   # Linux & macOS
```

Windows (PowerShell): `irm https://pipe-lang.com/install.ps1 | iex`

The installer downloads the latest release, verifies its SHA256 checksum and installs `pipe` into `~/.local/bin` (or `/usr/local/bin` when run as root). Pin a version with `PIPE_VERSION=v0.9.4.0`. See the [full install docs](docs/en/01-getting-started.md).

## Privacy & DSGVO

Pipe is **DSGVO-konform / GDPR-compliant by design**:

- **Zero telemetry & analytics** — the binary never phones home, nothing leaves your machine
- **Self-hosted single binary** — runs entirely on your infrastructure
- **No cloud** — no vendor server processes your data
- **Open source (MIT)** — fully auditable
- **Local AI** — with Ollama, not a single byte leaves your network; cloud providers are used only if you configure one

## The Problem

Running AI in production is harder than it should be:

- **Security** — LLMs with file access, network, and `exec` are a liability. You need fine-grained sandboxing at the language level, not afterthought middleware.
- **Performance** — Sequential API calls turn a 1-second pipeline into a 10-second bottleneck. Parallelism shouldn't require `asyncio.gather()` boilerplate.
- **Vendor Lock-in** — Switching from OpenAI to DeepSeek means rewriting your Python SDK code. Provider changes should be one line.
- **Tool Integration** — Connecting LLMs to external tools (GitHub, databases, filesystems) is a maze of SDKs and API wrappers. MCP should be a language primitive, not a library.

**Pipe fixes this at the language level.**

## What is Pipe?

Pipe is a **Semantic Pipeline Runtime (SPR)** — a pipeline-native language where `summarize`, `translate`, and `classify` sit on the same syntax level as `+`, `sort`, and `len`. Data flows top to bottom through composable transformations. One binary. Zero dependencies.

**Python + LangChain (~80 lines):**

```python
import openai
client = openai.OpenAI()
def summarize(text):
    r = client.chat.completions.create(model="gpt-4o", messages=[{"role":"user","content":text}])
    return r.choices[0].message.content
def translate(text, lang):
    r = client.chat.completions.create(model="gpt-4o",
        messages=[{"role":"system","content":f"Translate to {lang}"},{"role":"user","content":text}])
    return r.choices[0].message.content
text = open("news.txt").read()
print(translate(summarize(text), "de"))
```

**Pipe (5 lines):**

```pipe
read_file "news.txt"
    > summarize       -- LLM call
    > translate "de"  -- LLM call
    > print
```

## Model Context Protocol

Pipe has **built-in MCP** — both as a server and client. No SDKs, no npm packages, no Python. Pure Go stdlib.

### MCP Server — Expose your tools

```pipe
fn get_weather city
    match city
        | "Berlin" -> "22°C, sunny"
        | "London" -> "15°C, rainy"
        | _ -> city ++ ": no data"

ai_tool "get_weather" "Get weather for a city" {city: "City name"} get_weather
mcp_server "Weather Agent" "1.0.0"
mcp_serve_stdio
```

Configure in Claude Desktop (`claude_desktop_config.json`):

```json
{ "mcpServers": { "pipe": { "command": "/tmp/pipe", "args": ["agent.pipe"] } } }
```

### MCP Client — Use external tools

```pipe
ai_provider "deepseek"
ai_set_key "deepseek" (env "DEEPSEEK_API_KEY")

-- Connect to GitHub + Filesystem MCP servers
mcp_use_stdio "npx" "-y" "@modelcontextprotocol/server-github" {GITHUB_TOKEN: (env "GITHUB_TOKEN")}
mcp_use_stdio "npx" "-y" "@modelcontextprotocol/server-filesystem" "/tmp"

-- AI discovers and uses all tools automatically
result: ai_with_tools "You are a DevOps assistant." "Search pipe's open issues and list files in /tmp." 10
print result
```

**Any stdio MCP server** works immediately: Filesystem, GitHub, Git, Postgres, SQLite, Slack, Brave Search, Memory, Sequential Thinking — anything on npm/uvx.

## Use Cases

### Log Analysis → Incident Report

```pipe
is_critical: fn line
    contains line "critical"

read_file "/var/log/app/errors.log"
    > split "\n"
    > filter is_critical
    > summarize
    > translate "de"
    > save "incident_report.txt"
```

### RAG Pipeline

```pipe
ai_provider "deepseek"

docs: read_lines "knowledge_base.txt"
vectors: embed_batch docs

question: "How does the bytecode VM work?"
q_vec: embed question
top: nearest q_vec vectors 3

context: ""
for idx in top
    context: context ++ (at docs idx) ++ "\n---\n"

ask ("Context:\n" ++ context ++ "\nQuestion: " ++ question)
    > print
```

### AI Agent with Tool Calling

```pipe
fn get_weather city
    match city
        | "Berlin" -> "22°C, sunny"
        | "London" -> "15°C, rainy"
        | _ -> city ++ ": no data"

ai_tool "get_weather" "Get current weather for a city" {city: "Name of the city"} get_weather

ai_with_tools "You are a weather assistant." "What's the weather in Berlin and London?"
    > print
```

### Discord CI/CD Notifications

```pipe
import "discord.pipe" as d
ai_provider "deepseek"

-- AI code review per commit, sent as Discord embed
review: ai_chat "Review this code change" diff 800

d.d_webhook_embed (env "DISCORD_WEBHOOK") {
    title: "🔧 CI: Push to master",
    color: 3447003,
    fields: [
        {name: "Changed Files", value: stat},
        {name: "AI Review", value: review}
    ]
}
```

## Comparison: Pipe vs Python + LangChain

|                          | Python + LangChain            | Pipe                           |
|--------------------------|-------------------------------|--------------------------------|
| **RAG pipeline**         | ~80 LOC                       | ~8 LOC                         |
| **Sandbox LLM access**   | Custom middleware              | One `sandbox_profile` block    |
| **Switch AI provider**   | Rewrite SDK calls              | `ai_provider "deepseek"`       |
| **Deploy to server**     | Docker + venv + pip            | `scp pipe binary`              |
| **Parallel LLM calls**   | `asyncio.gather()` boilerplate | `>>` operator, `ai_batch`      |
| **MCP Server + Client**  | Library-dependent              | 13 builtins, zero deps, 100+ servers |
| **Binary size**          | ~500 MB (with deps)            | ~8 MB                          |

## Features

- **MCP-native** — 6 builtins for MCP Server + Client. Pure Go stdlib. Connect to any stdio MCP server
- **Ship AI pipelines 10× faster** — 36 AI + 13 MCP builtins: no imports, no SDKs, no API wrappers
- **Lock down AI agents in one line** — Declarative sandbox profiles: restrict `exec`, `write_file`, `http_get` with a single block
- **Deploy in seconds** — One statically-linked ~8 MB binary. No venv, no pip, no Docker. Linux, macOS, Windows, Raspberry Pi, or your browser via WebAssembly
- **3 LLM calls in 1.5s, not 4s** — `>>` starts any pipeline stage in the background. Futures auto-resolve. `ai_batch` handles hundreds of texts concurrently with built-in rate limiting
- **No vendor lock-in** — OpenAI, Anthropic (Claude), DeepSeek, Ollama. Switch with one line. Same code works everywhere
- **Pipeline-native syntax** — `>` sequential, `>>` parallel. Data flows top to bottom — readable, composable, debuggable
- **Social platforms built in** — Discord webhooks and Telegram bots as Pipe modules. AI code reviews, notifications, chat — zero API costs for sending
- **Bytecode VM** — Compile to bytecode, execute ~7× faster with automatic caching
- **Module ecosystem** — 23 curated modules, registry with version pinning (`@1.0.0`). `pipe -get` installs, import by name
- **Built-in testing** — `test` blocks with `assert_eq`, `assert_error`. Run with `pipe -test`. Zero setup
- **GitHub Action** — Run Pipe directly in CI/CD. No installation needed
- **VSCode Extension** — Syntax highlighting, IntelliSense, LSP-powered diagnostics and completions
- **Self-extracting binary** — Ship your pipeline as a standalone executable (`pipe -build`)

## Quick Start

```bash
git clone https://github.com/MachuraHarry/pipe && cd pipe && make build
export DEEPSEEK_API_KEY="sk-..."
./bin/pipe -vm -q -c 'ai_provider "deepseek"; ask "What makes Pipe different?" > print'
```

## Try it in your browser

No install needed — Pipe runs fully in your browser via WebAssembly:

<p align="center">
  <a href="https://pipe-lang.com/playground.html">
    <img src="website/logo.svg" width="64" height="64" alt="Pipe"><br>
    <b>Open the Pipe Playground →</b>
  </a>
</p>

```pipe
-- Paste this into the playground and hit Run
levels: ["error","warn","info"]
read_file "server.log"
    > classify levels
    > summarize
    > print
```

## GitHub Action

Run Pipe directly in CI/CD — no installation needed:

```yaml
- uses: MachuraHarry/pipe/.github/actions/pipe-action@master
  with:
    script: |
      print "Hello from CI/CD!"
      log: exec "git log --oneline -20"
      print (get log "output")
```

[→ GitHub Action Documentation](docs/en/20-github-action.md)

## VSCode Extension

Syntax highlighting and full IntelliSense for `.pipe` files, powered by a Language Server Protocol client (`vscode/`) and the `pipe-lsp` server (`cmd/pipe-lsp`):

- Completion, hover docs, signature help, go-to-definition, references, rename
- Diagnostics (parse errors, undefined/unused variables) and semantic highlighting
- Format document, auto-completion of brackets, auto-indent and code folding

```sh
make vsix     # builds the server and packages vscode/pipe-syntax-0.9.5.vsix
```

Or run the extension in development with F5 from the `vscode/` folder. See [VSCode Extension Documentation](docs/en/15-vscode-extension.md).

## Module Ecosystem

Pipe has a [curated module library](https://github.com/MachuraHarry/pipe-modules) — **21 reusable modules** (2 more in development) with version pinning:

| Infrastructure | Data & CLI | AI & Agents | DevTools | Social |
|---|---|---|---|---|
| `pipe-http` | `sqlite` | `rag-pipe` 🆕 | `pipe-test` | `discord` 🆕 |
| `pipe-cli` | `jpipe` | `log-analyzer` | `pipe-validate` 🆕 | `x` 🆕 (in dev) |
| `pipe-orm` 🆕 | `pipe-tpl` | `sentiment` | | `telegram-bot` |
| `pipe-web` 🆕 | `pipe-date` | `code-review` | | `discord` (in dev) |
| | | `translate-batch` | | `x` (in dev) |
| | | `changelog-gen` | | |
| | | `email-classifier` | | |
| | | `incident-report` | | |
| | | `parallel-runner` | | |
| | | `date-formatter` | | |

```bash
pipe -search                 # Browse modules
pipe -search sql             # Filter by keyword
pipe -get sqlite             # Install latest
pipe -get sqlite@0.8.0       # Install specific version
```

```pipe
import "sqlite"                            -- database engine
import "pipe-http"                         -- HTTP client
import "discord.pipe" as d                 -- Discord webhooks + bot
import "x.pipe" as x                       -- X (Twitter) API v2

idx: index_create h "knowledge"
index_add idx "Pipe is an AI-native language."
index_search idx "language" 3 > each print
```

[→ Ecosystem Documentation](docs/en/21-ecosystem.md) | [→ Contribute a Module](https://github.com/MachuraHarry/pipe-modules/blob/master/CONTRIBUTING.md)

## Execution Modes

| Mode | Command | Speed |
|------|---------|-------|
| Tree-Walker | `./bin/pipe script.pipe` | Baseline |
| Bytecode VM | `./bin/pipe -vm -q script.pipe` | ~7× faster |

## 49 AI + MCP Builtins (36 AI + 13 MCP)

### Understanding
`summarize`, `translate`, `classify`, `extract`, `ask`, `generate`, `generate_json`

### Speed & Control
`ai_stream`, `ai_batch`, `ai_parallel`, `ai_rate_limit`, `ai_chat`, `ai_chat_json`

### Search & Retrieval
`web_search`, `wiki_search`, `embed`, `embed_batch`, `cosine_sim`, `dot_product`, `nearest`

### Agents & Tools
`agent`, `agent_ask`, `agent_clear`, `ai_tool`, `ai_with_tools`

### Config & Cost
`ai_provider`, `ai_model`, `ai_host`, `ai_set_key`, `ai_timeout`, `ai_cache`, `ai_cost`, `ai_tokens`, `ai_cache_hits`, `ai_cache_misses`

### MCP — Model Context Protocol
`mcp_server`, `mcp_serve_stdio`, `mcp_serve_sse`, `mcp_tools`, `mcp_resource`, `mcp_resource_template`, `mcp_prompt`, `mcp_resources`, `mcp_read_resource`, `mcp_prompts`, `mcp_prompt_get`, `mcp_use_stdio`, `mcp_use_sse`

### Self-Healing
`try_ai`, `try_ai_log`

## Advanced Features

### Self-Healing Code (`try_ai`)
```pipe
ai_provider "deepseek"

result: try_ai
    "42" * 3           -- E002 Type Error → AI wraps with to_num → 126
catch e
    0                   -- only reached if AI fix fails

print result           -- 126
```

### Parallel Pipeline (`>>`)
```pipe
a: "Frage A"
    >> ask
b: "Frage B"
    >> ask
c: "Frage C"
    >> ask

print a ++ b ++ c   -- Future auto-resolution
```

### Sandbox Profiles
```pipe
sandbox_profile "safe" {fs: "read-only", network: false, exec: false, ai: true}
sandbox_profile "agent" {fs: "temp-only", network: true, exec: false, ai: true}

set_sandbox "safe"
read_file "/etc/config"     -- ✅ reading allowed
write_file "/etc/config"    -- ❌ E_SANDBOX blocked
```

## Architecture

```
Source (.pipe) → Lexer → Parser → AST → [ Tree-Walker | Compiler + VM ]
                                            ↓
                                  Builtins (226 total: 36 AI + 13 MCP + 177 standard)
                                            ↓
                              MCP Server ↔ MCP Clients (stdio + HTTP)
```

- 67 token types, 36 AST node types, 43 opcodes
- ~37,000 LoC Go, 570 tests, 87 example programs
- Zero dependencies — pure Go stdlib

## Documentation

[→ Full documentation (English)](/docs/en/index.md)
[→ Vollständige Dokumentation (Deutsch)](/docs/de/index.md)

## Project Structure

```
pipe/
├── cmd/
│   ├── pipe/main.go           # Entry point
│   └── pipe-lsp/              # Language Server Protocol server (IntelliSense)
├── pkg/
│   ├── ai/                    # AI provider integrations
│   │   ├── ai.go
│   │   ├── ai_test.go
│   │   ├── embeddings.go
│   │   ├── providers.go
│   │   └── tools.go
│   ├── analysis/              # IntelliSense library (builtins, diagnostics, completion…)
│   ├── ast/                   # AST node definitions
│   │   └── ast.go
│   ├── build/                 # Self-extracting binary builder
│   │   └── build.go
│   ├── cache/                 # Bytecode cache
│   │   ├── cache.go
│   │   └── cache_test.go
│   ├── compiler/              # Compiler to bytecode
│   │   ├── compiler.go
│   │   ├── compiler_test.go
│   │   └── opcode.go
│   ├── eval/                  # Tree-walk interpreter
│   │   ├── builtins.go
│   │   ├── eval.go
│   │   └── eval_test.go
│   ├── formatter/             # Code formatter
│   │   ├── formatter.go
│   │   └── formatter_test.go
│   ├── lexer/                 # Lexer and tokens
│   │   ├── lexer.go
│   │   ├── lexer_test.go
│   │   └── token.go
│   ├── mcp/                   # MCP server + client (zero-dependency)
│   │   ├── types.go
│   │   ├── server.go
│   │   ├── client.go
│   │   ├── stdio.go
│   │   └── schema.go
│   ├── object/                # Runtime objects
│   │   ├── ai_builtins_test.go
│   │   ├── environment.go
│   │   └── object.go
│   ├── parser/                # Parser
│   │   ├── parser.go
│   │   └── parser_test.go
│   ├── stdlib/                # Standard library helpers
│   └── vm/                    # Bytecode VM
│       ├── vm.go
│       └── vm_test.go
├── examples/                  # 87 example programs
│   ├── mcp_server.pipe        # MCP server with weather/docs/shell tools
│   ├── mcp_filesystem.pipe    # MCP client using filesystem server
│   ├── mcp_github.pipe        # MCP client using GitHub server
│   ├── mcp_combined.pipe      # MCP hub: own tools + external servers
│   ├── ai_tool_demo.pipe
│   ├── selfhost/              # Self-hosting lexer/parser
│   └── ...
├── test/integration/          # Integration tests
├── vscode/                    # VSCode extension (syntax highlighting + LSP client)
│   ├── src/                   # LSP client bootstrap (TypeScript)
│   ├── syntaxes/pipe.tmLanguage.json
│   └── package.json
├── docs/                      # Documentation (DE + EN)
│   ├── en/                    # English docs (27 chapters)
│   └── de/                    # German docs (27 chapters)
├── website/                   # Project website
├── Makefile
├── go.mod
└── LICENSE
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

MIT — see [LICENSE](LICENSE).
