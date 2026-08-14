[lang:en]# ⚡ 3 LLM Calls in 1.5s, Not 4s — Parallel by Default[/lang]
[lang:de]# ⚡ 3 LLM-Calls in 1,5s statt 4s — Parallel als Standard[/lang]

[lang:en]
**The `>>` operator starts pipeline stages in the background. Futures auto-resolve when you use them — no `asyncio.gather()` boilerplate.**

> **Part of the *Pipe in 30 Lines* series:** [RAG without a vector DB](tutorial-local-rag.html) · [Self-healing code](tutorial-self-healing.html) · [Your first MCP server](tutorial-first-mcp-server.html)

Sequential LLM calls add up: three questions, one after another, each waiting its turn. Pipe's `>>` operator runs each stage concurrently — the future resolves automatically the moment you touch the value.

```pipe
ai_provider "deepseek"

a: "Löse 7*8+4 und antworte nur mit der Zahl."
    >> ask
b: "Löse 12*12 und antworte nur mit der Zahl."
    >> ask
c: "Löse 100/4 und antworte nur mit der Zahl."
    >> ask

t: now
print ("Frage 1: " ++ a)
print ("Frage 2: " ++ b)
print ("Frage 3: " ++ c)
print ("Fertig nach " ++ (to_str ((now) - t)) ++ "s")
```

What happens here:

- **`>>`** replaces `>` and starts the stage in the background — all three `ask` calls leave immediately.
- The values `a`, `b`, `c` are *futures*; they **auto-resolve** when printed or used. No `.await()`, no thread management.
- Timing is real: on a local run all three answers land in ~2 seconds instead of three round trips.

In Python this is `asyncio.gather()` plus an async client setup, event loops, and call-site discipline. In Pipe, parallelism is the default shape of the pipeline, not an import.

For batched workloads, `ai_batch` goes further — it fans out hundreds of texts with built-in rate limiting. The full comparison lives in [`examples/parallel_ai_demo.pipe`](https://github.com/MachuraHarry/pipe/blob/master/examples/parallel_ai_demo.pipe); the minimal version above runs with `export DEEPSEEK_API_KEY=... && pipe -vm -q examples/blog_parallel.pipe`.
[/lang]

[lang:de]
**Der `>>`-Operator startet Pipeline-Stufen im Hintergrund. Futures lösen sich beim Benutzen automatisch auf — kein `asyncio.gather()`-Boilerplate.**

> **Teil der Serie *Pipe in 30 Lines*:** [RAG ohne Vektor-DB](tutorial-local-rag.html) · [Selbstheilender Code](tutorial-self-healing.html) · [Dein erster MCP-Server](tutorial-first-mcp-server.html)

Sequentielle LLM-Calls summieren sich: drei Fragen, eine nach der anderen, jede wartet auf ihren Turn. Pipes `>>`-Operator startet jede Stufe parallel — das Future löst sich automatisch auf, sobald du den Wert anfasst.

```pipe
ai_provider "deepseek"

a: "Löse 7*8+4 und antworte nur mit der Zahl."
    >> ask
b: "Löse 12*12 und antworte nur mit der Zahl."
    >> ask
c: "Löse 100/4 und antworte nur mit der Zahl."
    >> ask

t: now
print ("Frage 1: " ++ a)
print ("Frage 2: " ++ b)
print ("Frage 3: " ++ c)
print ("Fertig nach " ++ (to_str ((now) - t)) ++ "s")
```

Was hier passiert:

- **`>>`** ersetzt `>` und startet die Stufe im Hintergrund — alle drei `ask`-Calls laufen sofort los.
- Die Werte `a`, `b`, `c` sind *Futures*; sie **lösen sich automatisch auf**, sobald sie gedruckt oder benutzt werden. Kein `.await()`, kein Thread-Management.
- Das Timing ist real: lokal landen alle drei Antworten in ~2 Sekunden statt drei Roundtrips.

In Python heißt das `asyncio.gather()` plus Async-Client-Setup, Event-Loops und Disziplin an jeder Call-Stelle. In Pipe ist Parallelität die Standardform der Pipeline, kein Import.

Für Batch-Workloads geht `ai_batch` weiter: hunderte Texte mit eingebautem Rate-Limiting. Der komplette Vergleich lebt in [`examples/parallel_ai_demo.pipe`](https://github.com/MachuraHarry/pipe/blob/master/examples/parallel_ai_demo.pipe); die Minimalversion oben startest du mit `export DEEPSEEK_API_KEY=... && pipe -vm -q examples/blog_parallel.pipe`.
[/lang]
