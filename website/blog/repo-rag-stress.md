[lang:en]# 🧪 Stress-Testing repo-rag Against Real GitHub Repositories — From 99 Files to a 12k-File Monster[/lang]
[lang:de]# 🧪 repo-rag unter Dauerbeschuss mit echten GitHub-Repos — von 99 Files bis zum 12k-Files-Monster[/lang]

[lang:en]
**We pointed [`repo_rag_server`](repo-rag-mcp.html) at four real repositories — gin, fiber, beego and microsoft/vscode — drove it with raw JSON-RPC over stdio, measured every phase with byte-level I/O accounting, and found one real bug, three architectural limits and a surprisingly capable keyless AI mode. This is the full lab report.**

> **Related reading:** [repo-rag MCP](repo-rag-mcp.html) — the server under test · [pipe-docs MCP](pipe-docs-mcp.html) — the architecture it generalizes

Everything you read below was measured against the actual MCP server process — not unit fixtures, not mocks. The question was simple: **at what repository size does this design stop working, and why?**

## 🔬 The setup: a measuring harness, not a vibe check

We wrote a small Python harness (`harness.py`) that:

- spawns the real `pipe examples/repo_rag_server.pipe` as a subprocess,
- speaks newline-delimited JSON-RPC over stdio exactly like an MCP host,
- polls `/proc/<pid>/status` (peak RSS) and `/proc/<pid>/io` (bytes read/written) while the server works,
- timestamps everything: spawn → `initialize` response = full clone+prune+index build time.

No API key was configured for most runs, so the server operated in its **keyword-only mode** — the baseline every user gets out of the box. Repos lived in isolated cache directories so we could measure four distinct phases independently:

| Phase | How |
|-------|-----|
| Cold start | fresh cache → clone + junk-prune + both index builds |
| Warm start | untouched cache → SHA-256 hash checks only |
| Reindex-only | `.db` files deleted, checkout kept |
| Incremental | DBs restored, a handful of files touched |

## 📊 The ladder: gin → fiber → beego → vscode

| | gin | fiber | beego | vscode (aborted) |
|---|---|---|---|---|
| Source files | 99 | 303 | 365 | **12,614** |
| Code symbols | 1,617 | 6,017 | 4,528 | ~39,000+ (est.) |
| Markdown chunks | 124 | 776 | 35 | – |
| **Cold start** | 37.6 s | **245.8 s** | **90.5 s** | **>85 min, unfinished** |
| Warm start | 7.3 s | 36.2 s | 38.5 s | – |
| Reindex-only | – | 159.9 s | 93.1 s | – |
| Incremental (+5 files) | – | 51.6 s | 35.8 s | – |
| `refresh_index` (warm) | 2.2 s | 22.0 s | 18.0 s | – |
| Peak RSS | 33 MB | 104 MB | 60 MB | **>1 GB** |
| `search_code` latency | 30–60 ms | 90–140 ms | 70–95 ms | – |

Correctness held up everywhere: symbol lookups verified against `grep` hit the exact file and line (`newViewsLockStore` → `app.go:148`, `RunWithMiddleWares` → `server/web/beego.go:59`), path traversal (`../../etc/passwd`) was cleanly rejected, unknown tools returned proper JSON-RPC errors.

## 🐛 Finding 1: The bug that appeared every other run

The first smoke test produced something stranger than slowness: `read_source` crashed with `replace_all: first argument must be a string` — but only sometimes. Four identical restarts: OK, crash, OK, crash.

Root cause: when registering tools via `ai_tool`, the parameter order used for **positional argument binding** was built by iterating a Go map — whose iteration order is deliberately randomized. With two parameters (`offset`, `path`), roughly half of all processes bound the arguments swapped, feeding the number `1` into a string function.

The fix made the schema's `required` list use the same element type JSON unmarshalling produces (`[]interface{}` instead of `[]string`), so every consumer can type-assert uniformly — binding order is now deterministically alphabetical, guarded by a regression test that registers six deliberately unsorted parameters and asserts their arrival order.

## 🐌 Finding 2: O(n²) hash checks — where cold starts go to die

Per file, the sync does `SELECT hash FROM files WHERE path = ?`. Sounds innocent — except the pure-Pipe SQLite engine answers queries with a **full table scan**, and the `files` table grows with every indexed file. File #12,000 scans up to 12,000 interpreted rows before its own insert.

The signature of this is unmistakable in the process telemetry: long phases of hot CPU with near-zero read I/O and flat memory. At fiber's scale (303 files ≈ ~46k row comparisons) it costs seconds; extrapolating to vscode's 12.6k files (~80M comparisons) it dominates everything. The irony: reading and parsing all of gin took ~25 s — the *checking whether we need to* part scales worse than the work itself. A plain in-memory map lookup would turn this O(n²) into O(n).

## 💾 Finding 3: Memory grows with the whole repo — and crashes lose everything

Peak RSS scaled at roughly **15–18 KB per symbol** across all repos: 33 MB (gin, 1.6k symbols) → 104 MB (fiber, 6k) → 60 MB (beego, 4.5k). That's linear and predictable — until you multiply it by a monorepo.

vscode: after ~85 minutes the build was still running, memory had blown past **1 GB** and was climbing at 200 MB per 30 seconds. Then the test machine (a phone-class ARM device) collapsed. And here is the harsh part: **everything was lost**. The pure-Pipe SQLite module persists only on `db_close`, which happens once, after the entire index is built. No checkpoint, no journal, no partial state — 85 minutes of scanning evaporated with the process.

## ⏱️ Finding 4: Warm starts are racing your MCP host's timeout

MCP hosts typically give a stdio server 30–60 s to answer `initialize`. repo-rag builds **all indexes before serving** — by design, so it never serves stale data. The consequence shows in the ladder: already at ~400 files, a warm start takes 36–39 s. Not because anything is slow per se, but because every file gets hashed and every symbol loaded before the first response. Combined with Finding 2, cold starts at vscode scale are simply beyond what any host will wait for.

## 🤖 Keyless AI: surprisingly capable (coming in the next release)

> **Note:** The **opencode provider described here is brand new and will ship with the next Pipe release** — it is merged on `master` but not part of v1.1.1.

Pipe gained a fourth AI provider: **OpenCode Zen** (`opencode.ai/zen`). Its party trick: a free public tier that works **without any API key** — requests authenticate via CLI-mimicking headers, and free models (`big-pickle`, `-free` suffixes) cost exactly $0.00 in the sandbox budget accounting.

We wired it into repo-rag's provider chain (opt-in via `REPO_RAG_AI=opencode`, model override via `REPO_RAG_MODEL`) and re-ran gin **with AI enabled**:

```bash
# keyless public tier, free model
export REPO_RAG_URL="https://github.com/gin-gonic/gin"
export REPO_RAG_AI="opencode"
export REPO_RAG_MODEL="mimo-v2.5-free"   # big-pickle was 503 during our window
pipe examples/repo_rag_server.pipe
```


- Embeddings run locally for this provider, so building the semantic index added **nothing measurable** to startup (7.41 s, same as keyword-only).
- Semantic `search_docs` got **faster** than keyword mode (10–39 ms vs 65 ms) because embedding lookup avoids the chunk-table scan entirely.
- `ask_docs` answered five real questions about gin in 9–20 s each:

| Question | Verdict |
|----------|---------|
| Create a router with default middleware | ✅ correct, complete, cited |
| Recovery middleware internals | ⚠️ **honest refusal** — "not in the provided context" |
| JSON binding + validation errors | ✅ correct method, tags, error handling |
| Route groups with group-scoped middleware | ✅ correct pattern |
| JSON libraries via build tags | ✅ exactly the documented trio (jsoniter/go_json/sonic) |

Zero hallucinations. The one miss is telling: gin's docs *do* contain a `CustomRecovery` example — the retrieval layer just didn't surface it (local embeddings + only six candidates). The model correctly refused instead of inventing. Better recall (more candidates, heading boosts) would have turned that refusal into a hit.

One operational note: the free tier rotates models — `big-pickle` answered with a 503 upstream error during our window, while `mimo-v2.5-free` worked flawlessly. Set `REPO_RAG_MODEL=mimo-v2.5-free` and you're productive again.

## 🔧 What we'd fix next

1. **Replace the per-file SQL hash check with an in-memory map** — turns the dominant O(n²) cost into O(n) and makes vscode-class repos feasible.
2. **Persist incrementally** (checkpoint every N files) — converts a crash from total loss into a resume.
3. **Support optional tool arguments properly** (schema `required` lists per tool) so the argument-shift normalizations become reachable.
4. **Raise `ask_docs` candidate count or add heading boosts** to close retrieval gaps like the `CustomRecovery` case.

## ✅ Verdict

For its sweet spot — repositories up to a few hundred source files — repo-rag delivers exactly what it promises: correct symbol search, clean path safety, graceful degradation without keys, and (with the upcoming Zen provider) cited AI answers for free. Past ~10k files, the current architecture hits a wall that is architectural, not incidental — and now there are numbers that show precisely where and why.

That is what stress tests are for.
[/lang]

[lang:de]
**Wir haben [`repo_rag_server`](repo-rag-mcp.html) auf vier echte Repositorys losgelassen — gin, fiber, beego und microsoft/vscode —, ihn mit rohem JSON-RPC über stdio getrieben, jede Phase mit Byte-Level-I/O-Accounting gemessen und dabei einen echten Bug, drei architektonische Grenzen und einen überraschend fähigen schlüssellosen KI-Modus gefunden. Das ist der komplette Laborbericht.**

> **Verwandte Lektüre:** [repo-rag MCP](repo-rag-mcp.html) — der getestete Server · [pipe-docs MCP](pipe-docs-mcp.html) — die Architektur, die er verallgemeinert

Alles unten wurde gegen den echten MCP-Server-Prozess gemessen — keine Unit-Fixtures, keine Mocks. Die Frage war simpel: **Bei welcher Repository-Größe hört dieses Design auf zu funktionieren, und warum?**

## 🔬 Das Setup: Mess-Harness statt Bauchgefühl

Ein kleiner Python-Harness:

- startet den echten `pipe examples/repo_rag_server.pipe` als Subprozess,
- spricht newline-delimited JSON-RPC über stdio wie ein echter MCP-Host,
- pollt `/proc/<pid>/status` (Peak-RSS) und `/proc/<pid>/io` (gelesene/geschriebene Bytes),
- stempelt alles: Spawn → `initialize`-Antwort = komplette Clone+Prune+Index-Build-Zeit.

Kein API-Key war konfiguriert, der Server lief also im **Keyword-only-Modus** — die Basis, die jeder Nutzer out-of-the-box bekommt. Isolierte Cache-Verzeichnisse machten vier Phasen unabhängig messbar: Kaltstart, Warmstart, Reindex-only (`.db`s gelöscht) und inkrementell (wenige Files angefasst).

## 📊 Die Treppe: gin → fiber → beego → vscode

| | gin | fiber | beego | vscode (abgebrochen) |
|---|---|---|---|---|
| Source-Files | 99 | 303 | 365 | **12.614** |
| Code-Symbole | 1.617 | 6.017 | 4.528 | ~39.000+ (geschätzt) |
| Markdown-Chunks | 124 | 776 | 35 | – |
| **Kaltstart** | 37,6 s | **245,8 s** | **90,5 s** | **>85 min, nicht fertig** |
| Warmstart | 7,3 s | 36,2 s | 38,5 s | – |
| Reindex-only | – | 159,9 s | 93,1 s | – |
| Inkrementell (+5 Files) | – | 51,6 s | 35,8 s | – |
| Peak-RSS | 33 MB | 104 MB | 60 MB | **>1 GB** |

Die Korrektheit hielt überall stand: Symbol-Lookups gegen `grep` verifiziert trafen exakt Datei und Zeile, Path-Traversal wurde sauber abgewiesen, unbekannte Tools lieferten korrekte JSON-RPC-Fehler.

## 🐛 Fund 1: Der Bug, der nur bei jedem zweiten Start erschien

Der erste Smoke-Test produzierte etwas Seltsameres als Langsamkeit: `read_source` crashte mit einem Typfehler — aber nur manchmal. Vier identische Restarts: OK, Crash, OK, Crash.

Die Ursache: Beim Tool-Registering über `ai_tool` wurde die Parameterreihenfolge für das **positionale Argument-Binding** aus einer Go-Map-Iteration gebaut — deren Reihenfolge ist absichtlich zufällig. Bei zwei Parametern (`offset`, `path`) vertauschte etwa die Hälfte aller Prozesse die Argumente und fütterte eine Zahl in eine Stringfunktion.

Der Fix bringt das Schema-`required` auf denselben Elementtyp, den JSON-Unmarshalling erzeugt (`[]interface{}` statt `[]string`), sodass alle Konsumenten einheitlich type-asserten können — die Binding-Order ist jetzt deterministisch alphabetisch, abgesichert durch einen Regressionstest mit sechs bewusst unsortierten Parametern.

## 🐌 Fund 2: O(n²)-Hash-Checks — wo Kaltstarts sterben

Pro File macht der Sync `SELECT hash FROM files WHERE path = ?`. Klingt harmlos — nur antwortet die pure-Pipe-SQLite-Engine auf Queries mit einem **Full-Table-Scan**, und die `files`-Tabelle wächst mit jedem indizierten File. File #12.000 scannt bis zu 12.000 interpretierte Rows vor seinem eigenen Insert.

Das Muster ist in der Telemetrie unverkennbar: lange Phasen heißer CPU bei nahezu null Lese-I/O und flachem RAM. Bei fibers Größe (303 Files ≈ ~46k Row-Vergleiche) kostet das Sekunden; hochgerechnet auf vsodes 12.6k Files (~80 Mio. Vergleiche) dominiert es alles. Die Ironie: Alle gin-Files zu lesen und parsen dauerte ~25 s — das *Prüfen, ob wir müssen*, skaliert schlechter als die Arbeit selbst. Ein simpler In-Memory-Map-Lookup würde aus O(n²) ein O(n) machen.

## 💾 Fund 3: RAM wächst mit dem ganzen Repo — und Crashes verlieren alles

Der Peak-RSS skalierte mit grob **15–18 KB pro Symbol**: 33 MB (gin) → 104 MB (fiber) → 60 MB (beego). Linear und berechenbar — bis man ihn mit einem Monorepo multipliziert.

vscode: Nach ~85 Minuten lief der Build noch, der Speicher war über **1 GB** und stieg um 200 MB pro 30 Sekunden. Dann kollabierte die Testmaschine (ein Handy-Klasse-ARM-Gerät). Und das Harte daran: **Alles war verloren.** Das pure-Pipe-SQLite persistiert ausschließlich bei `db_close` — einmal, nach dem kompletten Index-Build. Kein Checkpoint, kein Journal, kein Teilststand: 85 Minuten Scannen verdampften mit dem Prozess.

## ⏱️ Fund 4: Warmstarts rasen gegen den Startup-Timeout deines MCP-Hosts

MCP-Hosts geben einem Stdio-Server typischerweise 30–60 s Zeit für die `initialize`-Antwort. repo-rag baut **alle Indizes vor dem Serven** — absichtlich, um nie veraltete Daten zu liefern. Die Folge zeigt die Treppe: Schon bei ~400 Files dauert ein Warmstart 36–39 s. Kombiniert mit Fund 2 sind Kaltstarts in vscode-Größe schlicht jenseits dessen, worauf ein Host wartet.

## 🤖 Schlüssellose KI: überraschend fähig (kommt mit dem nächsten Release)

> **Hinweis:** Der hier beschriebene **opencode-Provider ist brandneu und erscheint erst mit dem nächsten Pipe-Release** — er ist auf `master` gemerged, aber nicht Teil von v1.1.1.

Pipe bekommt einen vierten AI-Provider: **OpenCode Zen** (`opencode.ai/zen`). Sein Party-Trick: ein kostenloser Public-Tier, der **ohne jeden API-Key** funktioniert — Requests authentifizieren sich über CLI-nachempfundene Header, und Free-Modelle (`big-pickle`, `-free`-Suffixe) kosten exakt $0,00 im Sandbox-Budget.

In repo-rags Provider-Chain eingebunden (Opt-in über `REPO_RAG_AI=opencode`, Modell-Override über `REPO_RAG_MODEL`) haben wir gin **mit KI** neu gefahren:

```bash
# schlüsselloser Public-Tier, freies Modell
export REPO_RAG_URL="https://github.com/gin-gonic/gin"
export REPO_RAG_AI="opencode"
export REPO_RAG_MODEL="mimo-v2.5-free"   # big-pickle lief in unserem Fenster auf 503
pipe examples/repo_rag_server.pipe
```


- Embeddings laufen bei diesem Provider lokal — der semantische Index kostete **nichts Messbares** an Startzeit (7,41 s, gleichauf mit Keyword-only).
- Semantische `search_docs` wurden **schneller** als der Keyword-Modus (10–39 ms vs 65 ms), weil Embedding-Lookup den Chunk-Table-Scan komplett umgeht.
- `ask_docs` beantwortete fünf echte gin-Fragen in je 9–20 s:

| Frage | Urteil |
|----------|---------|
| Router mit Default-Middleware erstellen | ✅ korrekt, vollständig, zitiert |
| Recovery-Middleware intern | ⚠️ **ehrliche Verweigerung** — „nicht im Kontext enthalten" |
| JSON-Binding + Validierungsfehler | ✅ korrekte Methode, Tags, Fehlerbehandlung |
| Route-Groups mit gruppenweisem Middleware-Scope | ✅ korrektes Muster |
| JSON-Bibliotheken via Build-Tags | ✅ exakt das dokumentierte Trio (jsoniter/go_json/sonic) |

Null Halluzinationen. Der eine Fehltreffer ist aufschlussreich: gins Docs *enthalten* ein `CustomRecovery`-Beispiel — nur die Retrieval-Schicht hob es nicht hervor (lokale Embeddings + nur sechs Kandidaten). Das Modell hat korrekt verweigert statt zu erfinden. Besseres Recall (mehr Kandidaten, Heading-Boosts) hätte aus der Verweigerung einen Treffer gemacht.

Eine operative Anmerkung: Der Free-Tier rotiert Modelle — `big-pickle` antwortete in unserem Zeitfenster mit einem 503-Upstream-Fehler, während `mimo-v2.5-free` einwandfrei lief. Ein `REPO_RAG_MODEL=mimo-v2.5-free` macht dich wieder produktiv.

## 🔧 Was wir als Nächstes fixen würden

1. **Den per-File-SQL-Hash-Check durch eine In-Memory-Map ersetzen** — macht aus der dominierenden O(n²)-Kostenstelle ein O(n) und machen vscode-Klasse-mäßige Repos machbar.
2. **Inkrementell persistieren** (Checkpoint alle N Files) — verwandelt einen Crash von Totalverlust in Wiederaufnahme.
3. **Optionale Tool-Argumente sauber unterstützen** (pro-Tool-`required`-Listen), damit die Argument-Shift-Normalisierungen überhaupt erreichbar werden.
4. **`ask_docs`-Kandidatenzahl erhöhen oder Heading-Boosts ergänzen**, um Retrieval-Lücken wie den `CustomRecovery`-Fall zu schließen.

## ✅ Fazit

In ihrer Sweet-Spot-Zone — Repositorys bis einige hundert Source-Files — liefert repo-rag genau, was sie verspricht: korrekte Symbolsuche, saubere Path-Safety, graceful Degradation ohne Keys und (mit dem kommenden Zen-Provider) zitierte KI-Antworten gratis. Jenseits von ~10k Files stößt die aktuelle Architektur an eine Wand, die architektonisch, nicht zufällig ist — und jetzt gibt es Zahlen, die exakt zeigen, wo und warum.

Dafür sind Stresstests da.
[/lang]
