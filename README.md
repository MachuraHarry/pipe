# Pipe (SPR) — Semantic Pipeline Runtime

[![CI](https://github.com/MachuraHarry/pipe/actions/workflows/ci.yml/badge.svg)](https://github.com/MachuraHarry/pipe/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-purple.svg)](LICENSE)
[![Version](https://img.shields.io/badge/version-0.5.0-blue.svg)](https://github.com/MachuraHarry/pipe/releases)
[![SPR](https://img.shields.io/badge/SPR-Semantic%20Pipeline%20Runtime-7c5cfc.svg)](#)

> **SPR** — a new category of runtime where AI operations are language primitives, not library calls.
> Pipe compiles to a single 10 MB binary. Zero dependencies.

## What is Pipe?

Pipe is a **Semantic Pipeline Runtime (SPR)** — a pipeline-based execution environment where `summarize`, `translate`, and `classify` live on the same syntactic level as `+`, `sort`, and `len`. 

**Python (30 lines):**

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

## Quick Start

```bash
git clone https://github.com/harry/pipe && cd pipe && make build
export DEEPSEEK_API_KEY="sk-..."
./bin/pipe -vm -q -c 'ai_provider "deepseek"; ask "What makes Pipe different?" > print'
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

## Module Ecosystem

Pipe has a [curated module library](https://github.com/MachuraHarry/pipe-modules) — 8 reusable AI pipeline modules:

```bash
pipe -search                 # Browse modules
pipe -search log             # Filter by keyword
pipe -get log-analyzer       # Install a module
```

```pipe
import "https://raw.githubusercontent.com/MachuraHarry/pipe-modules/master/log-analyzer/module.pipe"
logs > log_analyze > save "report.md"
```

[→ Ecosystem Documentation](docs/en/21-ecosystem.md) | [→ Contribute a Module](https://github.com/MachuraHarry/pipe-modules/blob/master/CONTRIBUTING.md)

## Execution Modes

| Mode | Command | Speed |
|------|---------|-------|
| Tree-Walker | `./bin/pipe script.pipe` | Baseline |
| Bytecode VM | `./bin/pipe -vm -q script.pipe` | ~7× faster |

## 25 AI Builtins

### Understanding
`summarize`, `translate`, `classify`, `extract`, `ask`, `generate`

### Speed & Control
`ai_stream`, `ai_batch`, `ai_parallel`, `ai_rate_limit`, `ai_chat`, `ai_chat_json`

### Search & Semantics
`embed`, `embed_batch`, `cosine_sim`, `dot_product`, `nearest`

### Agency & Tools
`ai_tool`, `ai_with_tools`, `ai_provider`, `ai_model`, `ai_timeout`

### Configuration
`ai_provider`, `ai_model`, `ai_timeout`

## Features

- **AI as Primitive** — 25 built-in AI operations, no libraries needed
- **Pipeline-Native** — Data flows top-to-bottom through transformations
- **Single Binary** — One ~10 MB file, statically linked, no venv/pip/npm
- **Bytecode VM** — Compile to bytecode, execute ~7× faster with automatic caching
- **Multi-Provider** — OpenAI, Anthropic (Claude), DeepSeek — switch with one line
- **Tool Calling** — Register Pipe functions as LLM tools, model decides when to call them
- **Streaming** — Real-time token output via `ai_stream`
- **Parallel AI** — `ai_batch` processes hundreds of texts concurrently with rate limiting
- **Embeddings** — Native vector operations: embed, cosine_sim, nearest, RAG-ready
- **Self-Extracting Binary** — Ship your pipeline as a standalone executable
- **81 Standard Builtins** — HTTP, JSON, TCP, Regex, File I/O, and more
- **Zero Dependencies** — No externals, pure Go standard library

## Examples

### Log Analysis → Incident Report
```pipe
read_file "server.log"
    > classify -- output: "error", "warning", "info"
    > summarize
    > translate "de"
    > write_file "incident_de.md"
```

### RAG Pipeline
```pipe
read_file "docs/"
    > embed_batch
    > store in docs_index

ask "What is the refund policy?"
    > embed
    > nearest docs_index
    > ask
    > print
```

### AI Agent with Tools
```pipe
func get_weather(city) {
    fetch "https://api.weather.com/" + city
        > json_parse
        > get "current"
}

ai_tool "get_weather" get_weather "Get current weather for a city"
ai_with_tools "What is the weather in Berlin?"
    > print
```

## Why Pipe?

| Feature | Pipe | Python | Bash |
|---------|------|--------|------|
| AI Primitives | Built-in | Libraries required | N/A |
| Single Binary | ✓ | ✗ | ✗ |
| Pipeline Syntax | Native | Manual | Pipes |
| JSON Support | Built-in | `json` module | `jq` |
| HTTP Client | Built-in | `requests` | `curl` |
| Error Handling | `?>` operator | `try/except` | `||` |
| Binary Size | ~10 MB | ~40 MB+ venv | Varies |
| Dependencies | Zero | pip + venv | System tools |

## Architecture

```
Source (.pipe) → Lexer → Parser → AST → [ Tree-Walker | Compiler + VM ]
                                          ↓
                              Builtins (81 stdlib + 25 AI)
```

- 52 token types, 27 AST node types, 47 opcodes
- ~6,500 LoC Go, 134+ tests, 23 example programs

## Documentation

[→ Full documentation (English)](/docs/en/index.md)
[→ Vollständige Dokumentation (Deutsch)](/docs/de/index.md)

## Project Structure

```
pipe/
├── cmd/pipe/main.go           # Entry point
├── pkg/
│   ├── ai/                    # AI provider integrations
│   │   ├── ai.go
│   │   ├── ai_test.go
│   │   ├── embeddings.go
│   │   ├── providers.go
│   │   └── tools.go
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
│   └── vm/                    # Bytecode VM
│       ├── vm.go
│       └── vm_test.go
├── examples/                  # 23 example programs
│   ├── ai_demo.pipe
│   ├── ai_embedding_demo.pipe
│   ├── ai_parallel_demo.pipe
│   ├── ai_stream_demo.pipe
│   ├── ai_tool_demo.pipe
│   ├── selfhost/              # Self-hosting lexer/parser
│   └── ...
├── test/integration/          # Integration tests
├── vscode/                    # VSCode extension
│   ├── package.json
│   └── syntaxes/pipe.tmLanguage.json
├── docs/                      # Documentation (DE + EN)
│   ├── en/                    # English docs (20 chapters)
│   └── de/                    # German docs (20 chapters)
├── website/                   # Project website
├── Makefile
├── go.mod
└── LICENSE
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

MIT — see [LICENSE](LICENSE).
