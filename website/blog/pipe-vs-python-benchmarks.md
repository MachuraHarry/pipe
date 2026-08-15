[lang:en]# 📊 Pipe vs Python + LangChain — Measured, Not Marketed[/lang]
[lang:de]# 📊 Pipe gegen Python + LangChain — gemessen, nicht vermarktet[/lang]

[lang:en]
**Six real task pairs, one provider, one machine. We replaced the marketing numbers with measurements — and published the exact commands so you can reproduce them.**

We built six pairs of programs that do the *same* job — once in Python with LangChain, once in Pipe — and measured two things: lines of code and wall-clock time. Same LLM provider (`deepseek-v4-flash`), same local machine, same prompt semantics. The full code lives in [`benchmarks/python-vs-pipe`](https://github.com/MachuraHarry/pipe/tree/master/benchmarks/python-vs-pipe) with the raw numbers in `results/results.json`.

## Methodology

- **Six tasks:** RAG retrieval, three parallel LLM calls, an agent with a tool, an HTTP summarize API, log analysis, and a batch of translations.
- **LOC** counts non-blank, non-comment lines.
- **Time** is wall time for the whole process — in Python that includes ~7 seconds of interpreter + `langchain`/`openai` import startup before the first call runs.
- **Python setup:** a fresh venv with `langchain` 1.3.15, `langchain-deepseek`, `langchain-ollama`, `faiss-cpu`, `fastapi` — the standard stack.
- **Pipe setup:** one 8.6 MB binary, no installs.
- One run each (LLM latency varies run to run); the parallel task also reports its internal timings.

## Results

| Task | Python+LC LOC | Pipe LOC | Python+LC | Pipe | |
|---|---|---|---|---|---|
| RAG pipeline | 26 | 14 | 9.58s | 2.08s | 4.6× |
| Parallel LLM calls | 21 | 12 | 9.59s | 1.16s | 8.3× |
| AI agent + tool | 19 | 11 | 7.08s | 2.50s | 2.8× |
| HTTP summarize API | 12 | 14 | 5.13s | 1.41s | 3.6× |
| Log analysis → report | 14 | 8 | 13.72s | 4.84s | 2.8× |
| Batch of translations | 22 | 9 | 7.16s | 7.07s | ~1× |

**Binary: 8.6 MB** for Pipe vs **345 MB** for the Python venv (40×).

## What the numbers say

**LOC: 40% less on average.** The RAG pipeline is the cleanest example — 26 lines of LangChain imports, embedding setup, vector store, retriever, prompt template, and chain wiring versus 14 lines where `embed_batch`, `nearest`, and `ask` are language primitives:

```pipe
ai_provider "deepseek"

docs: [read_file "data/docs/database.txt"]
push docs (read_file "data/docs/caching.txt")
push docs (read_file "data/docs/api.txt")
push docs (read_file "data/docs/deployment.txt")

vectors: embed_batch docs
question: "How do we rate-limit API requests?"
q_vec: embed question
top: nearest q_vec vectors 3

context: ""
for idx in top
    context: context ++ (at docs idx) ++ "\n---\n"

answer: ask ("Context:\n" ++ context ++ "\nQuestion: " ++ question)
print answer
```

**Startup is the real difference.** In the parallel task, the actual LLM work is comparable: Python sequential ~3.6s, `asyncio.gather()` ~1.3s, Pipe's `>>` ~1.0s. But the *whole Python process* took 9.59s — most of it importing `langchain` and friends before a single request left the machine. Pipe starts in milliseconds.

**The batch case is the honest one.** Both versions took ~7.1s because the work is pure network round-trips with zero per-process overhead to hide. When the bottleneck is the LLM provider, the language doesn't matter — which is exactly the result you want from a benchmark that isn't cherry-picked.

## Reproduce it

```bash
cd benchmarks/python-vs-pipe
export DEEPSEEK_API_KEY=...
python3 tools/measure.py --run
```

The script runs all six pairs, measures LOC and time, and writes `results/results.json`. The Python side needs a venv: `python3 -m venv python/.venv && python/.venv/bin/pip install -r python/requirements.txt`.

The bytecode VM comparison behind these numbers: Pipe compiles to bytecode and runs it in its own VM — the same idea that makes the runtime fast and the binary small. The sandbox, MCP, and `>>` parallelism all ship in that one 8.6 MB file.
[/lang]

[lang:de]
**Sechs echte Aufgaben-Paare, ein Provider, eine Maschine. Wir haben die Marketing-Zahlen durch Messungen ersetzt — und die exakten Befehle veröffentlicht, damit du alles nachvollziehen kannst.**

Wir haben sechs Programm-Paare gebaut, die dieselbe Aufgabe lösen — einmal in Python mit LangChain, einmal in Pipe — und zwei Dinge gemessen: Codezeilen und Wanduhr-Zeit. Gleicher LLM-Provider (`deepseek-v4-flash`), gleiche lokale Maschine, gleiche Prompt-Semantik. Der komplette Code liegt in [`benchmarks/python-vs-pipe`](https://github.com/MachuraHarry/pipe/tree/master/benchmarks/python-vs-pipe), die Rohwerte in `results/results.json`.

## Methodik

- **Sechs Aufgaben:** RAG-Retrieval, drei parallele LLM-Calls, ein Agent mit Tool, eine HTTP-Summarize-API, Log-Analyse und ein Batch von Übersetzungen.
- **LOC** zählt Zeilen ohne Leerzeilen und Kommentare.
- **Zeit** ist die Wanduhr-Zeit des gesamten Prozesses — in Python inklusive ~7 Sekunden Interpreter- und `langchain`/`openai`-Import-Startup, bevor der erste Call überhaupt startet.
- **Python-Setup:** ein frisches venv mit `langchain` 1.3.15, `langchain-deepseek`, `langchain-ollama`, `faiss-cpu`, `fastapi` — der Standard-Stack.
- **Pipe-Setup:** eine 8,6-MB-Binary, keine Installationen.
- Ein Lauf pro Paar (LLM-Latenz variiert von Lauf zu Lauf); die Parallel-Aufgabe meldet zusätzlich ihre internen Zeiten.

## Ergebnisse

| Aufgabe | Python+LC LOC | Pipe LOC | Python+LC | Pipe | |
|---|---|---|---|---|---|
| RAG-Pipeline | 26 | 14 | 9,58s | 2,08s | 4,6× |
| Parallele LLM-Calls | 21 | 12 | 9,59s | 1,16s | 8,3× |
| KI-Agent + Tool | 19 | 11 | 7,08s | 2,50s | 2,8× |
| HTTP-Summarize-API | 12 | 14 | 5,13s | 1,41s | 3,6× |
| Log-Analyse → Report | 14 | 8 | 13,72s | 4,84s | 2,8× |
| Batch Übersetzungen | 22 | 9 | 7,16s | 7,07s | ~1× |

**Binary: 8,6 MB** für Pipe gegenüber **345 MB** für das Python-venv (40×).

## Was die Zahlen sagen

**LOC: im Schnitt 40% weniger.** Am klarsten wird das an der RAG-Pipeline — 26 Zeilen LangChain-Imports, Embedding-Setup, Vector Store, Retriever, Prompt-Template und Chain-Verdrahtung gegenüber 14 Zeilen, in denen `embed_batch`, `nearest` und `ask` Sprach-Primitives sind:

```pipe
ai_provider "deepseek"

docs: [read_file "data/docs/database.txt"]
push docs (read_file "data/docs/caching.txt")
push docs (read_file "data/docs/api.txt")
push docs (read_file "data/docs/deployment.txt")

vectors: embed_batch docs
question: "How do we rate-limit API requests?"
q_vec: embed question
top: nearest q_vec vectors 3

context: ""
for idx in top
    context: context ++ (at docs idx) ++ "\n---\n"

answer: ask ("Context:\n" ++ context ++ "\nQuestion: " ++ question)
print answer
```

**Der Startup ist der eigentliche Unterschied.** Bei der Parallel-Aufgabe ist die reine LLM-Arbeit vergleichbar: Python sequenziell ~3,6s, `asyncio.gather()` ~1,3s, Pipes `>>` ~1,0s. Aber der *gesamte Python-Prozess* brauchte 9,59s — der Großteil davon entfiel auf das Importieren von `langchain` und Co., bevor eine einzige Anfrage das System verlassen hat. Pipe startet in Millisekunden.

**Der Batch-Fall ist der ehrliche.** Beide Versionen brauchten ~7,1s, weil die Arbeit reine Netzwerk-Roundtrips sind und es keinen Prozess-Overhead zu verstecken gibt. Wenn der LLM-Provider der Flaschenhals ist, spielt die Sprache keine Rolle — genau das will man von einem Benchmark sehen, der nicht schön gerechnet ist.

## Nachvollziehen

```bash
cd benchmarks/python-vs-pipe
export DEEPSEEK_API_KEY=...
python3 tools/measure.py --run
```

Das Skript führt alle sechs Paare aus, misst LOC und Zeit und schreibt `results/results.json`. Die Python-Seite braucht ein venv: `python3 -m venv python/.venv && python/.venv/bin/pip install -r python/requirements.txt`.

Der Bytecode-VM-Vergleich dahinter: Pipe kompiliert zu Bytecode und führt ihn in einer eigenen VM aus — dieselbe Idee, die die Laufzeit schnell und die Binary klein macht. Sandbox, MCP und `>>`-Parallelität stecken alle in dieser einen 8,6-MB-Datei.
[/lang]
