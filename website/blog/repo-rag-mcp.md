[lang:en]# 🔍 repo-rag MCP — RAG over ANY Git Repository, Not Just This One[/lang]
[lang:de]# 🔍 repo-rag MCP — RAG über ein BELIEBIGES Git-Repository, nicht nur über dieses[/lang]

[lang:en]
**One command turns any Git repository into a full RAG server for your AI IDE: keyword search that works with zero API keys, cited AI answers, code symbol lookup across five languages, and file outlines — all backed by persistent SQLite indexes and a locked-down sandbox.**

> **Related reading:** [pipe-docs MCP](pipe-docs-mcp.html) — the same architecture, but hard-wired to the Pipe language docs · [RAG in ~10 Lines](tutorial-local-rag.html) — the minimal pattern this server generalizes

Our [`pipe_docs_server`](pipe-docs-mcp.html) answers questions about *Pipe itself*. But the moment you work on a different project, you want the same experience there: point an AI agent at **any** repository and let it search, read, and reason about the code without dumping files into context windows. That is exactly what `examples/repo_rag_server.pipe` does — one Pipe file, no dependencies beyond the `docs-pipe` module (auto-fetched from the registry), published indexes, and a hardened sandbox.

## 🚀 Quickstart: 60 seconds to your own repo RAG

```bash
# 1. Install Pipe (Linux/macOS/Windows)
curl -fsSL https://raw.githubusercontent.com/MachuraHarry/pipe/master/install.sh | sh

# 2. Point it at any repository and start the MCP server (stdio)
export REPO_RAG_URL="https://github.com/your-user/your-repo"
pipe examples/repo_rag_server.pipe
```

The first run clones the repository shallowly, prunes junk directories, builds three persistent SQLite indexes, then locks itself into a read-only sandbox and serves MCP over stdio. Register it in your MCP client:

```json
{
  "mcpServers": {
    "repo-rag": {
      "command": "pipe",
      "args": ["examples/repo_rag_server.pipe"],
      "env": {
        "REPO_RAG_URL": "https://github.com/your-user/your-repo",
        "OPENROUTER_API_KEY": "sk-or-..."
      }
    }
  }
}
```

## 🧰 The tools: 11 ways into your codebase

| Tool | Needs key | What it does |
|------|-----------|--------------|
| `search_docs(query)` | optional* | Markdown search across README, docs/, wikis — semantic hybrid with a key, keyword-only without |
| `ask_docs(question)` | yes | Cited RAG answer grounded in the documentation |
| `read_doc(path)` | no | Read any Markdown file |
| `list_docs()` | no | List all discovered `.md`/`.mdx` files |
| `search_code(query)` | no | Find functions, types, classes, structs, enums, tests in Go, Pipe, Python, JS/TS, Rust + generic fallback |
| `file_symbols(path)` | no | **New:** outline of ONE file — every indexed declaration with kind, name, line and declaration text |
| `read_source(path, offset)` | no | Source view with line numbers, paginated at 500 lines |
| `list_sources()` | no | All recognized source files |
| `repo_info()` | no | URL, ref, detected languages, readiness |
| `index_status()` | no | Index statistics + last sync counts |
| `refresh_index()` | no | Incremental re-sync from the cached checkout |

\* keyword mode needs no key at all — the documentation of any public repo is searchable out of the box.

The typical agent loop looks like: `list_sources` → `file_symbols("pkg/server/handler.go")` to understand a file's structure → `read_source` for the interesting region → `search_code` when hunting for a name. Every answer stays small and targeted instead of flooding the context window.

## ⚙️ Under the hood: three SQLite indexes, zero re-indexing pain

On startup the server builds up to three persistent databases in its cache directory:

1. **`code.db`** — every declaration (function, class, struct, enum, test) with file, line range, language and source text. Synced incrementally: each file's SHA-256 decides whether it gets rescanned, so a warm start costs a few hash checks.
2. **`docs-kw.db`** — heading-aware Markdown chunks for keyword retrieval. Works with **no API key whatsoever**; scores weight heading hits 3× and normalize by query token count so single-hit chunks don't truncate to zero.
3. **`docs.db`** — semantic embedding index via `docs-pipe`, only built when a provider with an embeddings endpoint is configured.

Persistence has one subtlety: the pure-Pipe sqlite module flushes on `db_close`, while serve handles stay open for the process lifetime. The server therefore runs each index through a throwaway build handle whose close persists, then reopens a serving handle — and if the filesystem is already read-only at that point, a `try/catch` falls back to in-memory resync. A killed process loses nothing; a restart reports `unchanged: N` instead of re-indexing.

## 🔒 Security: least privilege by construction

An MCP server that clones arbitrary repositories must be paranoid. The server declares two sandbox profiles and locks the harder one before serving:

- **`rag-build`** (startup): filesystem full, exec restricted to `git` and `rm`, network limited to Git hosts and AI provider APIs. The repository URL is validated against a strict character allowlist *before* it ever touches a shell command.
- **`rag-serve`** (locked): filesystem read-only, exec completely disabled, network narrowed to the configured AI providers. Even a prompt-injected model cannot write files, run commands, or phone home elsewhere.

Path arguments pass through a resolver gate that rejects absolute paths and `..` traversal, so `read_source("/etc/passwd")` fails cleanly.

## 🤖 AI providers: including OpenRouter free models

Any of three keys enables the AI tier:

- **`DEEPSEEK_API_KEY`** / **`OPENAI_API_KEY`** — full experience: semantic hybrid `search_docs` plus grounded `ask_docs`.
- **`OPENROUTER_API_KEY`** — chat completions through OpenRouter, ideal with the free tier. Set `REPO_RAG_MODEL` to pick a model (default `nvidia/nemotron-3-super-120b-a12b:free`; browse others with the `:free` suffix).

One honest caveat: OpenRouter exposes **no embeddings endpoint**, so the semantic layer stays dormant there. Instead of letting answers degrade to general knowledge, `ask_docs` detects empty semantic retrieval and falls back to the keyword chunk index — answers remain grounded in the actual repository with numbered citations:

```json
{
  "answer": "Widgets are components or concepts described as great in the Alpha Doc [1] and explored in detail—including their lifecycle and tips—in the Beta Doc [2].",
  "sources": [
    { "path": "a.md", "score": 0.4, "line_start": 1 },
    { "path": "b.md", "score": 0.2, "line_start": 1 }
  ]
}
```

Real output from a live test against a two-file fixture repository, produced by a free-tier model. Free models share an upstream rate-limit pool, so expect occasional 429s — pick another `:free` slug via `REPO_RAG_MODEL` or retry shortly.

## ✅ Quality: parity-tested, not vibes-tested

The server ships with a 41-test suite (`scripts/repo-rag-code-index-test.pipe`) covering chunking, syncing, searching and persistence. The suite runs byte-identically under the tree-walker **and** the bytecode VM — which is not a given: getting there surfaced and fixed four real engine bugs (module symbol isolation, a while-body terminator defect, builtin masking in module scope, and double-emitted index operands). The server itself was smoke-tested end-to-end over stdio JSON-RPC: cold start, warm start, cited AI answers against a live OpenRouter key, sandbox-locked refreshes.

## 🗺️ Try it

Everything lives in the repository:

- Server: [`examples/repo_rag_server.pipe`](https://github.com/MachuraHarry/pipe/blob/master/examples/repo_rag_server.pipe)
- Library: [`examples/lib/repo_rag_lib.pipe`](https://github.com/MachuraHarry/pipe/blob/master/examples/lib/repo_rag_lib.pipe)
- Test suite: [`scripts/repo-rag-code-index-test.pipe`](https://github.com/MachuraHarry/pipe/blob/master/scripts/repo-rag-code-index-test.pipe)
- Docs: MCP chapter §25.8 in [`docs/en/25-mcp.md`](https://github.com/MachuraHarry/pipe/blob/master/docs/en/25-mcp.md)

Clone Pipe, export `REPO_RAG_URL`, run the server, and give your AI IDE eyes into any codebase. If you build something with it — or want more tools (reference search? git-log integration?) — issues and PRs are welcome.
[/lang]

[lang:de]
**Ein Befehl verwandelt ein beliebiges Git-Repository in einen vollwertigen RAG-Server für deine KI-IDE: Keyword-Suche ganz ohne API-Key, zitierte KI-Antworten, Code-Symbol-Lookup in fünf Sprachen und File-Outlines — alles auf persistenten SQLite-Indexen und in einer verriegelten Sandbox.**

> **Weiterlesen:** [pipe-docs MCP](pipe-docs-mcp.html) — dieselbe Architektur, aber fest auf die Pipe-Doku verdrahtet · [RAG in ~10 Zeilen](tutorial-local-rag.html) — das minimale Muster, das dieser Server verallgemeinert

Unser [`pipe_docs_server`](pipe-docs-mcp.html) beantwortet Fragen zu *Pipe selbst*. Aber sobald du an einem anderen Projekt arbeitest, willst du dort dasselbe Erlebnis: einen KI-Agenten auf **ein beliebiges** Repository zeigen lassen und ihn suchen, lesen und über den Code nachdenken lassen — ohne Dateien in Kontextfenster zu kippen. Genau das macht `examples/repo_rag_server.pipe`: eine Pipe-Datei, keine Abhängigkeiten außer dem `docs-pipe`-Modul (holt sich der Registry automatisch), publizierte Indexe und eine gehärtete Sandbox.

## 🚀 Quickstart: In 60 Sekunden zum eigenen Repo-RAG

```bash
# 1. Pipe installieren (Linux/macOS/Windows)
curl -fsSL https://raw.githubusercontent.com/MachuraHarry/pipe/master/install.sh | sh

# 2. Auf ein beliebiges Repo zeigen und den MCP-Server starten (stdio)
export REPO_RAG_URL="https://github.com/dein-user/dein-repo"
pipe examples/repo_rag_server.pipe
```

Der erste Lauf klont das Repository flach, entfernt Junk-Verzeichnisse, baut drei persistente SQLite-Indexe, verriegelt sich dann in eine Read-only-Sandbox und serviert MCP über stdio. Registriere den Server in deinem MCP-Client:

```json
{
  "mcpServers": {
    "repo-rag": {
      "command": "pipe",
      "args": ["examples/repo_rag_server.pipe"],
      "env": {
        "REPO_RAG_URL": "https://github.com/dein-user/dein-repo",
        "OPENROUTER_API_KEY": "sk-or-..."
      }
    }
  }
}
```

## 🧰 Die Tools: 11 Wege in deine Codebase

| Tool | Key nötig | Was es tut |
|------|-----------|------------|
| `search_docs(query)` | optional* | Markdown-Suche über README, docs/, Wikis — semantisch-hybrid mit Key, sonst rein per Keyword |
| `ask_docs(question)` | ja | Zitierte RAG-Antwort auf Basis der Doku |
| `read_doc(path)` | nein | Beliebige Markdown-Datei lesen |
| `list_docs()` | nein | Alle gefundenen `.md`/`.mdx`-Dateien |
| `search_code(query)` | nein | Functions, Types, Classes, Structs, Enums, Tests in Go, Pipe, Python, JS/TS, Rust + generischem Fallback |
| `file_symbols(path)` | nein | **Neu:** Outline einer Datei — jede indizierte Deklaration mit Art, Name, Zeile und Deklarationstext |
| `read_source(path, offset)` | nein | Quellcode-Ansicht mit Zeilennummern, paginiert à 500 Zeilen |
| `list_sources()` | nein | Alle erkannten Quelldateien |
| `repo_info()` | nein | URL, Ref, erkannte Sprachen, Bereitschaft |
| `index_status()` | nein | Index-Statistiken + letzte Sync-Counts |
| `refresh_index()` | nein | Inkrementeller Re-Sync aus dem gecachten Checkout |

\* Der Keyword-Modus braucht gar keinen Key — die Doku jedes öffentlichen Repos ist damit out-of-the-box durchsuchbar.

Die typische Agent-Schleife: `list_sources` → `file_symbols("pkg/server/handler.go")` zum Verstehen der Dateistruktur → `read_source` für die interessante Region → `search_code` bei der Jagd nach einem Namen. Jede Antwort bleibt klein und gezielt, statt das Kontextfenster zu fluten.

## ⚙️ Unter der Haube: drei SQLite-Indexe, kein Re-Indexierungs-Stress

Beim Start baut der Server bis zu drei persistente Datenbanken in seinem Cache-Verzeichnis:

1. **`code.db`** — jede Deklaration (Funktion, Klasse, Struct, Enum, Test) mit Datei, Zeilenbereich, Sprache und Quelltext. Inkrementell synchronisiert: Der SHA-256 jeder Datei entscheidet über einen Rescan — ein Warm Start kostet nur wenige Hash-Prüfungen.
2. **`docs-kw.db`** — Heading-bewusste Markdown-Chunks für Keyword-Retrieval. Funktioniert **ganz ohne API-Key**; Scores gewichten Heading-Treffer 3× und normalisieren über die Query-Token, sodass Ein-Treffer-Chunks nicht auf Score 0 abschneiden.
3. **`docs.db`** — semantischer Embedding-Index via `docs-pipe`, nur gebaut wenn ein Provider mit Embeddings-Endpunkt konfiguriert ist.

Bei der Persistenz gibt es einen Kniff: Das Pure-Pipe-sqlite-Modul flushed bei `db_close`, während Serve-Handles prozesslang offen bleiben. Deshalb läuft jeder Index durch einen Wegwerf-Build-Handle, dessen `close` persistiert, danach wird ein frisches Serve-Handle geöffnet — und wenn das Filesystem zu dem Zeitpunkt schon read-only ist, fängt ein `try/catch` das mit In-Memory-Resync ab. Ein gekillter Prozess verliert nichts; ein Neustart meldet `unchanged: N` statt neu zu indizieren.

## 🔒 Sicherheit: Least Privilege by Construction

Ein MCP-Server, der beliebige Repositories klont, muss paranoid sein. Der Server deklariert zwei Sandbox-Profile und verriegelt das härtere, bevor er serviert:

- **`rag-build`** (Startup): Filesystem voll, exec auf `git` und `rm` beschränkt, Netz auf Git-Hosts und AI-Provider-APIs begrenzt. Die Repository-URL wird *vor* jedem Shell-Kontakt gegen eine strikte Zeichen-Allowlist validiert.
- **`rag-serve`** (verriegelt): Filesystem read-only, exec komplett deaktiviert, Netz auf die konfigurierten AI-Provider eingedampft. Selbst ein prompt-injiziertes Modell kann keine Dateien schreiben, keine Kommandos ausführen und nirgendwohin telefonieren.

Pfad-Argumente laufen durch ein Resolver-Gate, das absolute Pfade und `..`-Traversal ablehnt — `read_source("/etc/passwd")` scheitert sauber.

## 🤖 KI-Provider: inklusive OpenRouter-Free-Models

Drei Keys schalten die KI-Ebene frei:

- **`DEEPSEEK_API_KEY`** / **`OPENAI_API_KEY`** — das volle Erlebnis: semantisch-hybrides `search_docs` plus grounded `ask_docs`.
- **`OPENROUTER_API_KEY`** — Chat-Completions über OpenRouter, ideal für den Free-Tier. Mit `REPO_RAG_MODEL` wählst du das Modell (Default `nvidia/nemotron-3-super-120b-a12b:free`; weitere mit `:free`-Suffix).

Ein ehrlicher Hinweis: OpenRouter bietet **keinen Embeddings-Endpunkt**, dort bleibt also die semantische Ebene außen vor. Statt Antworten aufs Allgemeinwissen abrutschen zu lassen, erkennt `ask_docs` die leere semantische Treffermenge und fällt auf den Keyword-Chunks-Index zurück — Antworten bleiben im echten Repository verankert, mit nummerierten Zitationen:

```json
{
  "answer": "Widgets are components or concepts described as great in the Alpha Doc [1] and explored in detail—including their lifecycle and tips—in the Beta Doc [2].",
  "sources": [
    { "path": "a.md", "score": 0.4, "line_start": 1 },
    { "path": "b.md", "score": 0.2, "line_start": 1 }
  ]
}
```

Echte Ausgabe eines Live-Tests gegen ein Zwei-Dateien-Fixture-Repo, produziert von einem Free-Tier-Model. Free-Models teilen sich einen Upstream-Rate-Limit-Pool — gelegentliche 429s sind normal: entweder kurz warten oder über `REPO_RAG_MODEL` einen anderen `:free`-Slug wählen.

## ✅ Qualität: Parität statt Vibes

Der Server kommt mit einer 41-Test-Suite (`scripts/repo-rag-code-index-test.pipe`): Chunking, Sync, Suche und Persistenz. Sie läuft byte-identisch unter dem Tree-Walker **und** der Bytecode-VM — was nicht selbstverständlich ist: Auf dem Weg dorthin kamen vier echte Engine-Bugs ans Licht (Modul-Symbol-Isolation, ein While-Body-Terminator-Defekt, Builtin-Masking im Modul-Scope und doppelt emittierte Index-Operanden). Der Server selbst wurde End-to-End per stdio-JSON-RPC gesmoket: Cold Start, Warm Start, zitierte KI-Antworten gegen einen echten OpenRouter-Key, Sandbox-verriegelte Refreshes.

## 🗺️ Ausprobieren

Alles liegt im Repository:

- Server: [`examples/repo_rag_server.pipe`](https://github.com/MachuraHarry/pipe/blob/master/examples/repo_rag_server.pipe)
- Bibliothek: [`examples/lib/repo_rag_lib.pipe`](https://github.com/MachuraHarry/pipe/blob/master/examples/lib/repo_rag_lib.pipe)
- Test-Suite: [`scripts/repo-rag-code-index-test.pipe`](https://github.com/MachuraHarry/pipe/blob/master/scripts/repo-rag-code-index-test.pipe)
- Doku: MCP-Kapitel §25.8 in [`docs/de/25-mcp.md`](https://github.com/MachuraHarry/pipe/blob/master/docs/de/25-mcp.md)

Pipe klonen, `REPO_RAG_URL` exportieren, Server starten — und deiner KI-IDE Augen in jede Codebase geben. Wenn du etwas damit baust — oder mehr Tools willst (Referenzsuche? git-log-Integration?) — Issues und PRs sind willkommen.
[/lang]
