[lang:en]# 🔍 pipe-docs MCP — Your Entire Codebase as a Semantic Search, Inside Your AI IDE[/lang]
[lang:de]# 🔍 pipe-docs MCP — Deine gesamte Codebase als semantische Suche, direkt in deiner KI-IDE[/lang]

[lang:en]
**A semantic search and RAG MCP server for the Pipe language — 7 tools, zero dependencies, published on the MCP Registry. Point your AI at your docs, your source code, or both, and get cited answers in seconds.**

> **Related:** [docs-pipe dashboard](docs-pipe-dashboard.html) · [RAG in ~10 lines](tutorial-local-rag.html) · [Your first MCP server](tutorial-first-mcp-server.html)

Most AI coding assistants can read files — but they read them *blindly*. They don't know that `docs/en/17-compiler.md` explains the register allocator, or that `pkg/compiler/compiler.go:738` contains the `lastInstructionIsTerminator` function. They don't know which chunk of documentation answers a question about sandbox profiles, and they certainly can't trace a claim back to a specific heading in a specific file.

`pipe-docs` fixes that. It's a Pipe-native MCP server that turns your entire documentation tree *and* your Go/Pipe source code into a semantic search engine that any MCP-aware AI can query — with cited answers, code symbol lookup, and heading-aware chunking. No vector database, no embedding SDK, no framework. One binary, seven tools.

## Seven tools, one server

```pipe
import "docs-pipe"

-- 1. index your docs
idx: doc_index "docs/en" {lang: "en", db: "docs-en.db"}

-- 2. search by meaning
results: doc_search idx "How does the bytecode VM work?" 5

-- 3. ask with cited sources
res: doc_ask idx "What is MCP?" 4
print (get res "answer")     -- answers with [1], [2], [3] citations
print (get res "sources")    -- [{path, line_start, heading}, ...]
```

That's the core API — six functions: `doc_index`, `doc_search`, `doc_ask`, `doc_index_status`, `doc_reindex`, `doc_close`. The MCP server wraps them into seven tools:

| Tool | What it does |
|---|---|
| `search_docs` | Semantic + keyword hybrid search across EN, DE, and blog docs |
| `ask_docs` | RAG answer with cited sources — the AI gets context, you get traceability |
| `search_code` | Find Go/Pipe functions, types, structs by name or keyword |
| `read_doc` | Read a specific documentation file by path |
| `list_docs` | List all files in EN, DE, and blog directories |
| `index_status` | Report stats: files, chunks, symbols, has_ai flag |
| `refresh` | Re-fetch the source repo and rebuild the index |

The server runs as a single Pipe process over stdio — the same transport Claude Desktop, Cursor, opencode, and every other MCP client already speaks.

## How it works: heading-aware chunking, hybrid search

A naive approach embeds every paragraph and calls `nearest`. That misses context. `pipe-docs` does three things differently:

**Heading-aware chunking.** Documents are split on `#`/`##`/`###` boundaries, fenced code blocks stay intact, and every chunk remembers its file, line range, and heading path — so a search for "sandbox profiles" surfaces the `Security > Sandbox Profiles > Locking` section, not just a passing mention. Indexing `docs/en` yields **28 files → 317 chunks**.

**Hybrid search.** A TF-IDF keyword score and a cosine-similarity score are fused — and the weights adapt automatically: local 128-dim embeddings lean on keywords, OpenAI's 1536-dim embeddings lean on semantics. Identifiers stay single tokens (`try_ai`, not `try` + `ai`), and a query term matching a chunk's *heading* gets a boost.

**Cited answers.** `doc_ask` doesn't just answer — it cites. Every answer comes with a `sources` array: `path:line — heading`, so you can always trace a claim back to the exact doc section. The AI gets context, you get accountability.

## The code index: Go and Pipe, together

Beyond documentation, `pipe-docs` scans your Go and Pipe source code and builds a searchable symbol index. Functions, types, structs, enums, tests — all indexed with file, kind, symbol name, and line number.

```
search_code "sandbox" →
  pkg/compiler/compiler.go:738  func  lastInstructionIsTerminator
  pkg/object/builtins_process.go:1  func  proc_start
  pkg/sandbox/sandbox.go:42   struct  Sandbox
```

This means when you ask an AI "where is the sandbox gate for proc_start?", it doesn't just guess — it finds the exact file and line, reads the surrounding code, and answers with context.

## Publishing to the MCP Registry

`pipe-docs` is published on the [MCP Registry](https://registry.modelcontextprotocol.io) as `io.github.MachuraHarry/pipe-docs`. One command installs it:

```bash
# opencode (mcp.json)
{
  "pipe-docs": {
    "type": "local",
    "command": ["pipe", "examples/pipe_docs_server.pipe"],
    "environment": { "DEEPSEEK_API_KEY": "sk-..." }
  }
}
```

The MCPB bundle ships five platform variants (linux-amd64, linux-arm64, darwin-amd64, darwin-arm64, windows-amd64), each ~3 MB, containing the pipe binary and the server script. No runtime, no npm, no pip — just one process.

The server is built in pure Pipe: ~500 lines of server logic, ~900 lines of the `docs-pipe` module, and the MCP protocol handling is three builtins: `ai_tool`, `mcp_server`, `mcp_serve_stdio`. That's the entire dependency tree.

## First startup: build once, cache forever

The first time the server starts, it clones the source repo and builds the docs index. This takes ~70 seconds and produces three SQLite databases (`docs-en.db`, `docs-de.db`, `docs-blog.db`). On subsequent starts, it loads from cache in ~5 seconds. The AI provider is configured lazily — `search_code`, `search_docs`, `read_doc`, and `list_docs` work without an API key. Only `ask_docs` needs one, and a full answer costs a fraction of a cent.

```
index_status →
  ready: true
  has_ai: true
  source_files: 173
  code_symbols: 2731
  docs_en: {files: 28, chunks: 317}
  docs_de: {files: 28, chunks: 267}
  docs_blog: {files: 15, chunks: 198}
```

## What this means for Pipe users

If you're building with Pipe, your AI assistant can now *understand* your codebase — not just read files, but reason about them. Ask "how do sandbox profiles work?" and get a cited answer pointing to the exact section in the docs. Ask "where is the register allocator?" and get the file, line, and function name. Ask "what changed in the compiler?" and get the diff context.

It's the same Pipe binary serving the MCP server that you use to write your pipelines. That's the whole point: when the language *is* the runtime, the server is just another script.
[/lang]

[lang:de]
**Ein semantischer Suche- und RAG-MCP-Server für die Pipe-Sprache — 7 Tools, null Abhängigkeiten, veröffentlicht auf dem MCP Registry. Richte deine KI auf deine Doku, deinen Quellcode oder beides und erhalte in Sekunden zitierte Antworten.**

> **Verwandt:** [docs-pipe Dashboard](docs-pipe-dashboard.html) · [RAG in ~10 Zeilen](tutorial-local-rag.html) · [Dein erster MCP-Server](tutorial-first-mcp-server.html)

Die meisten KI-Coding-Assistenten können Dateien lesen — aber sie lesen sie *blind*. Sie wissen nicht, dass `docs/en/17-compiler.md` den Register-Allocator erklärt, oder dass `pkg/compiler/compiler.go:738` die Funktion `lastInstructionIsTerminator` enthält. Sie wissen nicht, welcher Dokumentations-Chunk eine Frage über Sandbox-Profile beantwortet, und können eine Aussage sicherlich nicht auf eine bestimmte Überschrift in einer bestimmten Datei zurückführen.

`pipe-docs` löst das. Es ist ein Pipe-nativer MCP-Server, der deinen gesamten Dokumentationsbaum *und* deinen Go/Pipe-Quellcode in eine semantische Suchmaschine verwandelt, die jeder MCP-fähige AI abfragen kann — mit zitierten Antworten, Code-Symbol-Lookup und Heading-bewusstem Chunking. Keine Vektor-DB, kein Embedding-SDK, kein Framework. Eine Binary, sieben Tools.

## Sieben Tools, ein Server

```pipe
import "docs-pipe"

-- 1. Doku indexieren
idx: doc_index "docs/en" {lang: "en", db: "docs-en.db"}

-- 2. nach Bedeutung suchen
results: doc_search idx "Wie funktioniert die Bytecode-VM?" 5

-- 3. mit Quellenangaben antworten
res: doc_ask idx "Was ist MCP?" 4
print (get res "answer")     -- Antworten mit [1], [2], [3] Zitaten
print (get res "sources")    -- [{path, line_start, heading}, ...]
```

Das ist die Kern-API — sechs Funktionen: `doc_index`, `doc_search`, `doc_ask`, `doc_index_status`, `doc_reindex`, `doc_close`. Der MCP-Server wickelt sie in sieben Tools:

| Tool | Was es macht |
|---|---|
| `search_docs` | Semantische + Keyword-Hybrid-Suche über EN-, DE- und Blog-Doku |
| `ask_docs` | RAG-Antwort mit Quellenangaben — die KI bekommt Kontext, du Nachvollziehbarkeit |
| `search_code` | Finde Go/Pipe-Funktionen, Typen, Structs nach Name oder Keyword |
| `read_doc` | Lies eine bestimmte Dokumentationsdatei nach Pfad |
| `list_docs` | Liste aller Dateien in EN-, DE- und Blog-Verzeichnissen |
| `index_status` | Statistiken: Dateien, Chunks, Symbole, has_ai-Flag |
| `refresh` | Quellrepo neu laden und Index neu aufbauen |

Der Server läuft als einzelner Pipe-Prozess über stdio — dasselbe Transport-Protokoll, das Claude Desktop, Cursor, opencode und jeder andere MCP-Client bereits spricht.

## So funktioniert's: Heading-bewusstes Chunking, Hybrid-Suche

Ein naiver Ansatz bettet jeden Absatz ein und ruft `nearest`. Das verliert Kontext. `pipe-docs` macht drei Dinge anders:

**Heading-bewusstes Chunking.** Dokumente werden an `#`/`##`/`###`-Grenzen geteilt, Code-Blöcke bleiben intakt, und jeder Chunk merkt sich seine Datei, seinen Zeilenbereich und seinen Heading-Pfad — also findet eine Suche nach "Sandbox-Profile" den Abschnitt `Security > Sandbox Profiles > Locking`, nicht nur eine beiläufige Erwähnung. Das Indexieren von `docs/en` ergibt **28 Dateien → 317 Chunks**.

**Hybrid-Suche.** Ein TF-IDF-Keyword-Score und ein Kosinus-Ähnlichkeits-Score werden fusioniert — und die Gewichte passen sich automatisch an: lokale 128-dim-Embeddings stützen sich auf Keywords, OpenAIs 1536-dim-Embeddings stützen sich auf Semantik. Bezeichner bleiben einzelne Tokens (`try_ai`, nicht `try` + `ai`), und ein Suchbegriff, der auf die *Überschrift* eines Chunks trifft, bekommt einen Boost.

**Zitierte Antworten.** `doc_ask` antwortet nicht nur — es zitiert. Jede Antwort kommt mit einem `sources`-Array: `path:line — heading`, damit du eine Aussage immer bis zur exakten Dokumentations-Section zurückverfolgen kannst. Die KI bekommt Kontext, du bekommst Verantwortlichkeit.

## Der Code-Index: Go und Pipe zusammen

Neben der Dokumentation scannt `pipe-docs` deinen Go- und Pipe-Quellcode und baut einen durchsuchbaren Symbol-Index auf. Funktionen, Typen, Structs, Enums, Tests — alles indexiert mit Datei, Art, Symbolname und Zeilennummer.

```
search_code "sandbox" →
  pkg/compiler/compiler.go:738  func  lastInstructionIsTerminator
  pkg/object/builtins_process.go:1  func  proc_start
  pkg/sandbox/sandbox.go:42   struct  Sandbox
```

Das bedeutet: Wenn du eine KI fragst "wo ist der Sandbox-Gate für proc_start?", errät sie nicht — sie findet die exakte Datei und Zeile, liest den umgebenden Code und antwortet mit Kontext.

## Veröffentlichung im MCP Registry

`pipe-docs` ist im [MCP Registry](https://registry.modelcontextprotocol.io) als `io.github.MachuraHarry/pipe-docs` veröffentlicht. Ein Befehl installiert es:

```bash
# opencode (mcp.json)
{
  "pipe-docs": {
    "type": "local",
    "command": ["pipe", "examples/pipe_docs_server.pipe"],
    "environment": { "DEEPSEEK_API_KEY": "sk-..." }
  }
}
```

Das MCPB-Bundle liefert fünf Plattform-Varianten (linux-amd64, linux-arm64, darwin-amd64, darwin-arm64, windows-amd64), jede ~3 MB, bestehend aus der Pipe-Binary und dem Server-Script. Kein Runtime, kein npm, kein pip — nur ein Prozess.

Der Server ist in reinem Pipe gebaut: ~500 Zeilen Server-Logik, ~900 Zeilen des `docs-pipe`-Moduls, und das MCP-Protokoll-Handling sind drei Builtins: `ai_tool`, `mcp_server`, `mcp_serve_stdio`. Das ist der gesamte Abhängigkeitsbaum.

## Erster Start: einmal bauen, für immer cachen

Beim ersten Start des Servers wird das Quellrepo geklont und der Docs-Index aufgebaut. Das dauert ~70 Sekunden und erzeugt drei SQLite-Datenbanken (`docs-en.db`, `docs-de.db`, `docs-blog.db`). Bei folgenden Starts lädt er aus dem Cache in ~5 Sekunden. Der AI-Provider wird lazy konfiguriert — `search_code`, `search_docs`, `read_doc` und `list_docs` funktionieren ohne API-Key. Nur `ask_docs` braucht einen, und eine volle Antwort kostet einen Bruchteil eines Cents.

```
index_status →
  ready: true
  has_ai: true
  source_files: 173
  code_symbols: 2731
  docs_en: {files: 28, chunks: 317}
  docs_de: {files: 28, chunks: 267}
  docs_blog: {files: 15, chunks: 198}
```

## Was das für Pipe-Nutzer bedeutet

Wenn du mit Pipe baust, kann dein KI-Assistent jetzt deine Codebase *verstehen* — nicht nur Dateien lesen, sondern darüber nachdenken. Frage "wie funktionieren Sandbox-Profile?" und erhalte eine zitierte Antwort, die auf die exakte Section in der Doku zeigt. Frage "wo ist der Register-Allocator?" und erhalte Datei, Zeile und Funktionsname. Frage "was hat sich im Compiler geändert?" und erhalte den Diff-Kontext.

Es ist dieselbe Pipe-Binary, die den MCP-Server bedient, mit der du deine Pipelines schreibst. Das ist der ganze Punkt: Wenn die Sprache *die* Runtime ist, ist der Server nur ein weiteres Script.
[/lang]
