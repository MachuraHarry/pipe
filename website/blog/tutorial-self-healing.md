[lang:en]# 🩹 Self-Healing Code — `try_ai` Fixes Runtime Errors with AI[/lang]
[lang:de]# 🩹 Selbstheilender Code — `try_ai` repariert Laufzeitfehler mit KI[/lang]

[lang:en]
**A type error that repairs itself: `try_ai` catches the crash, asks the LLM to fix the expression, and re-runs it.**

> **Part of the *Pipe in 30 Lines* series:** [RAG without a vector DB](tutorial-local-rag.html) · [Parallel LLM calls](tutorial-parallel.html) · [Your first MCP server](tutorial-first-mcp-server.html)

`"42" * 3` is a type error — a string multiplied by a number. Most runtimes crash. Pipe's `try_ai` catches the error, sends the broken expression to the LLM, and re-runs the repaired code. The `catch` block only fires if even the AI can't fix it.

```pipe
ai_provider "deepseek"

result: try_ai
    "42" * 3          -- E002 type error → AI repairs it
catch e
    0                 -- only reached if the fix fails

print result          -- 126

fast: try_ai
    6 * 7             -- healthy code: zero API calls
catch e
    -1

print fast            -- 42
```

What happens here:

- **`try_ai`** runs the block. On success it returns the value directly — **no API call, no latency**. The safety net is free.
- On a runtime error it asks the LLM to repair the broken expression: `"42" * 3` becomes `(to_num "42") * 3` and re-runs.
- The **`catch`** block is the final fallback — your pipeline never dies silently.

This is where a language boundary pays off: `try_ai` isn't a library that wraps your calls, it's syntax that can *rewrite and re-execute a fragment of your own program*. That's self-healing infrastructure — not an SDK callback.

```text
[try_ai] E002 | attempt 1 | ""42" * 3" → "(to_num "42") * 3" | ✓ FIXED
126
42
```

Run it: `export DEEPSEEK_API_KEY=... && pipe -vm -q examples/blog_try_ai.pipe`. The deeper mechanics (including the sandboxed re-run profile) are covered in the [MCP Cell deep dive](mcp-cell.html).
[/lang]

[lang:de]
**Ein Typfehler, der sich selbst repariert: `try_ai` fängt den Crash, lässt das LLM den Ausdruck fixen und führt ihn erneut aus.**

> **Teil der Serie *Pipe in 30 Lines*:** [RAG ohne Vektor-DB](tutorial-local-rag.html) · [Parallele LLM-Calls](tutorial-parallel.html) · [Dein erster MCP-Server](tutorial-first-mcp-server.html)

`"42" * 3` ist ein Typfehler — ein String mal eine Zahl. Die meisten Runtimes crashen. Pipes `try_ai` fängt den Fehler, schickt den kaputten Ausdruck an das LLM und führt den reparierten Code erneut aus. Der `catch`-Block feuert nur, wenn selbst die KI es nicht hinbekommt.

```pipe
ai_provider "deepseek"

result: try_ai
    "42" * 3          -- E002 Typfehler → KI repariert ihn
catch e
    0                 -- wird nur erreicht, wenn der Fix scheitert

print result          -- 126

fast: try_ai
    6 * 7             -- gesunder Code: null API-Calls
catch e
    -1

print fast            -- 42
```

Was hier passiert:

- **`try_ai`** führt den Block aus. Bei Erfolg liefert es den Wert direkt — **kein API-Call, keine Latenz**. Das Sicherheitsnetz ist kostenlos.
- Bei einem Laufzeitfehler bittet es das LLM, den Ausdruck zu reparieren: aus `"42" * 3` wird `(to_num "42") * 3`, dann wird erneut ausgeführt.
- Der **`catch`**-Block ist die letzte Absicherung — deine Pipeline stirbt nie still.

Hier zahlt sich die Sprachgrenze aus: `try_ai` ist keine Bibliothek, die deine Calls umwickelt, sondern Syntax, die *ein Fragment deines eigenen Programms umschreiben und neu ausführen* kann. Selbstheilende Infrastruktur — kein SDK-Callback.

```text
[try_ai] E002 | attempt 1 | ""42" * 3" → "(to_num "42") * 3" | ✓ FIXED
126
42
```

Starten: `export DEEPSEEK_API_KEY=... && pipe -vm -q examples/blog_try_ai.pipe`. Die tieferen Mechanismen (inkl. des sandboxed Re-Run-Profils) findest du im [MCP-Cell-Deep-Dive](mcp-cell.html).
[/lang]
