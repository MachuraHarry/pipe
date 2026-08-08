# <img src="website/logo.svg" width="32" height="32" align="left" style="margin-right:8px"> Pipe — The runtime for AI-native infrastructure

[![CI](https://github.com/MachuraHarry/pipe/actions/workflows/ci.yml/badge.svg)](https://github.com/MachuraHarry/pipe/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-purple.svg)](LICENSE)
[![Version](https://img.shields.io/badge/version-0.9.2-blue.svg)](https://github.com/MachuraHarry/pipe/releases)
[![SPR](https://img.shields.io/badge/SPR-Semantic%20Pipeline%20Runtime-7c5cfc.svg)](#)

> **Build, sandbox, and deploy LLM pipelines with a single ~7 MB binary. No Python. No dependencies. No vendor lock-in.**

## Quick Install

```sh
curl -fsSL https://pipe-lang.com/install.sh | bash   # Linux & macOS
```

Windows (PowerShell): `irm https://pipe-lang.com/install.ps1 | iex`

The installer downloads the latest release, verifies its SHA256 checksum and installs `pipe` into `~/.local/bin` (or `/usr/local/bin` when run as root). Pin a version with `PIPE_VERSION=v0.9.2`. See the [full install docs](docs/en/01-getting-started.md).

## The Problem

Running AI in production is harder than it should be:

- **Security** — LLMs with file access, network, and `exec` are a liability. You need fine-grained sandboxing at the language level, not afterthought middleware.
- **Performance** — Sequential API calls turn a 1-second pipeline into a 10-second bottleneck. Parallelism shouldn't require `asyncio.gather()` boilerplate.
- **Vendor Lock-in** — Switching from OpenAI to DeepSeek means rewriting your Python SDK code. Provider changes should be one line.
- **Deployment** — `pip install`, `requirements.txt`, Docker images, virtual environments — just to ship 5 lines of logic.

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

### RAG Pipeline (with rag-pipe module)

```pipe
import "rag-pipe"

idx: index_create h "knowledge"
index_add idx "Pipe is a semantic pipeline runtime for AI-native infrastructure."
index_add idx "The bytecode VM runs 7x faster than the tree-walker interpreter."

-- Semantic search
results: index_search idx "fast execution" 2
for r in results
    print (get r "text")

-- AI-powered Q&A with retrieved context
answer: index_ask idx "How does Pipe execute code?"
print answer
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

### Module Pipeline: HTTP → JSON → AI → Report

```pipe
import "pipe-http"
import "pipe-tpl"

data: hget_json "https://api.example.com/stats" {}
analysis: ask ("Summarize: " ++ (to_json data))

report: render "Report: {{ analysis }}\nTop items:\n{{ for item in items }}* {{ item }}\n{{ end }}" data
print report
```

### CLI Tool (with pipe-cli)

```pipe
import "pipe-cli"

cli: app "deployer" "Deployment tool"
cmd: command cli "release" "Create a release"
flag cmd "env" "e" "Environment" "staging"
flag cmd "tag" "t" "Git tag to deploy" ""
handler cmd handler_fn
run cli args
```

## Comparison: Pipe vs Python + LangChain

|                          | Python + LangChain            | Pipe                           |
|--------------------------|-------------------------------|--------------------------------|
| **RAG pipeline**         | ~80 LOC                       | ~8 LOC                         |
| **Sandbox LLM access**   | Custom middleware              | One `sandbox_profile` block    |
| **Switch AI provider**   | Rewrite SDK calls              | `ai_provider "deepseek"`       |
| **Deploy to server**     | Docker + venv + pip            | `scp pipe binary`              |
| **Parallel LLM calls**   | `asyncio.gather()` boilerplate | `>>` operator, `ai_batch`      |
| **Binary size**          | ~500 MB (with deps)            | ~7 MB                          |

## Features

- **Ship AI pipelines 10× faster** — 36 AI builtins: no imports, no SDKs, no API wrappers
- **Lock down AI agents in one line** — Declarative sandbox profiles: restrict `exec`, `write_file`, `http_get` with a single block
- **Deploy in seconds** — One statically-linked ~7 MB binary. No venv, no pip, no Docker. Linux, macOS, Windows, Raspberry Pi, or your browser via WebAssembly
- **3 LLM calls in 1.5s, not 4s** — `>>` starts any pipeline stage in the background. Futures auto-resolve. `ai_batch` handles hundreds of texts concurrently with built-in rate limiting
- **No vendor lock-in** — OpenAI, Anthropic (Claude), DeepSeek, Ollama. Switch with one line. Same code works everywhere
- **Pipeline-native syntax** — `>` sequential, `>>` parallel. Data flows top to bottom — readable, composable, debuggable
- **Bytecode VM** — Compile to bytecode, execute ~7× faster with automatic caching
- **Module ecosystem** — 21 curated modules, registry with version pinning (`@1.0.0`). `pipe -get` installs, import by name
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
make vsix     # builds the server and packages vscode/pipe-syntax-0.1.0.vsix
```

Or run the extension in development with F5 from the `vscode/` folder. See [VSCode Extension Documentation](docs/en/15-vscode-extension.md).

## Module Ecosystem

Pipe has a [curated module library](https://github.com/MachuraHarry/pipe-modules) — **21 reusable modules** with version pinning:

| Infrastructure | Data & CLI | AI & Agents | DevTools |
|---|---|---|---|
| `pipe-http` | `sqlite` | `rag-pipe` 🆕 | `pipe-test` |
| `pipe-cli` | `jpipe` | `log-analyzer` | `pipe-validate` 🆕 |
| `pipe-orm` 🆕 | `pipe-tpl` | `sentiment` | |
| `pipe-web` 🆕 | `pipe-date` | `code-review` | |
| | `telegram-bot` | `translate-batch` | |
| | | `changelog-gen` | |
| | | `email-classifier` | |
| | | `incident-report` | |
| | | `parallel-runner` | |
| | | `date-formatter` | |

```bash
pipe -search                 # Browse modules
pipe -search sql             # Filter by keyword
pipe -get sqlite             # Install latest
pipe -get sqlite@0.8.0       # Install specific version
```

```pipe
import "sqlite"                            -- database engine
import "pipe-http"                         -- HTTP client
import "rag-pipe"                          -- RAG: embeddings + search + AI

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

## 36 AI Builtins

### Understanding
`summarize`, `translate`, `classify`, `extract`, `ask`, `generate`, `generate_json`

### Speed & Control
`ai_stream`, `ai_batch`, `ai_parallel`, `ai_rate_limit`, `ai_chat`, `ai_chat_json`

### Search & Retrieval
`web_search`, `wiki_search`, `embed`, `embed_batch`, `cosine_sim`, `dot_product`, `nearest`

### Agents & Tools
`agent`, `agent_ask`, `agent_clear`, `ai_tool`, `ai_with_tools`

### Configuration
`ai_provider`, `ai_model`, `ai_timeout`, `ai_host`, `ai_cache`, `ai_set_key`

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
                               Builtins (183 total, AI + stdlib)
```

- 67 token types, 35 AST node types, 42 opcodes
- ~19,000 LoC Go, 300+ tests, ~60 example programs

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
├── examples/                  # ~60 example programs
│   ├── ai_demo.pipe
│   ├── ai_embedding_demo.pipe
│   ├── ai_parallel_demo.pipe
│   ├── ai_stream_demo.pipe
│   ├── ai_tool_demo.pipe
│   ├── selfhost/              # Self-hosting lexer/parser
│   └── ...
├── test/integration/          # Integration tests
├── vscode/                    # VSCode extension (syntax highlighting + LSP client)
│   ├── src/                   # LSP client bootstrap (TypeScript)
│   ├── syntaxes/pipe.tmLanguage.json
│   └── package.json
├── docs/                      # Documentation (DE + EN)
│   ├── en/                    # English docs (24 chapters)
│   └── de/                    # German docs (24 chapters)
├── website/                   # Project website
├── Makefile
├── go.mod
└── LICENSE
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

MIT — see [LICENSE](LICENSE).
