[lang:en]# 📚 RAG in ~10 Lines — No Vector Database, No `pip install`[/lang]
[lang:de]# 📚 RAG in ~10 Zeilen — ohne Vektor-DB, ohne `pip install`[/lang]

[lang:en]
**Semantic search and retrieval-augmented generation without a vector DB, a framework, or a single dependency.**

> **Part of the *Pipe in 30 Lines* series:** [Self-healing code](tutorial-self-healing.html) · [Parallel LLM calls](tutorial-parallel.html) · [Your first MCP server](tutorial-first-mcp-server.html)

RAG usually means a stack: a vector database, an embedding SDK, a retrieval library, and glue code to hold it together. Pipe collapses that into language primitives. `embed_batch` vectorizes your documents, `nearest` finds the top matches by *meaning*, and `ask` answers with the retrieved context — all in one binary, zero imports.

```pipe
ai_provider "deepseek"

docs: ["Pipe is a pipeline-native language where data flows top to bottom.", "The bytecode VM is measured 0.6x-55x faster than the tree-walker, depending on workload.", "Built-in MCP lets Pipe expose its own tools or consume any stdio MCP server.", "Sandbox profiles restrict exec, write_file, and http_get in one declarative block.", "One ~8 MB binary, zero dependencies, on Linux, macOS, Windows, or the browser."]

vectors: embed_batch docs        -- 1. vectorize once
question: "What makes the VM fast?"
q_vec: embed question            -- 2. embed the query
top: nearest q_vec vectors 2     -- 3. top-k by similarity

context: ""
for idx in top
    context: context ++ (at docs idx) ++ "\n---\n"

ask ("Context:\n" ++ context ++ "\nQuestion: " ++ question)
    > print
```

What happens here:

- **`embed_batch`** turns every document into a vector in one call — no vector DB to install, no index to configure.
- **`nearest`** returns the *indices* of the most relevant documents by cosine similarity. Meaning, not keywords.
- The top matches are stitched into a context block, and **`ask`** answers grounded in it — classic RAG.

In Python + LangChain the same pipeline is roughly 80 lines across SDKs. In Pipe it's a dozen, and it works with **every provider**: swap `"deepseek"` for `"openai"`, `"anthropic"`, or `"ollama"` and nothing else changes.

Want the full version with a web UI, SQLite persistence, and a `/search` API? See [`examples/rag_knowledge_base.pipe`](https://github.com/MachuraHarry/pipe/blob/master/examples/rag_knowledge_base.pipe) — or run the minimal version above with `export DEEPSEEK_API_KEY=... && pipe examples/blog_rag.pipe`.
[/lang]

[lang:de]
**Semantische Suche und RAG ohne Vektor-DB, ohne Framework und ohne eine einzige Abhängigkeit.**

> **Teil der Serie *Pipe in 30 Lines*:** [Selbstheilender Code](tutorial-self-healing.html) · [Parallele LLM-Calls](tutorial-parallel.html) · [Dein erster MCP-Server](tutorial-first-mcp-server.html)

RAG bedeutet sonst einen ganzen Stack: eine Vektor-DB, ein Embedding-SDK, eine Retrieval-Bibliothek und Kleber-Code. Pipe reduziert das auf Sprach-Primitives. `embed_batch` vektorisiert deine Dokumente, `nearest` findet die besten Treffer *nach Bedeutung*, und `ask` antwortet mit dem gelieferten Kontext — alles in einer Binary, null Imports.

```pipe
ai_provider "deepseek"

docs: ["Pipe ist eine pipeline-native Sprache, in der Daten von oben nach unten fließen.", "Die Bytecode-VM führt Programme etwa 7x schneller aus als der Tree-Walker.", "Eingebautes MCP: Pipe stellt eigene Tools bereit oder nutzt jeden stdio-MCP-Server.", "Sandbox-Profile sperren exec, write_file und http_get in einem deklarativen Block.", "Eine ~8-MB-Binary, null Abhängigkeiten, auf Linux, macOS, Windows oder im Browser."]

vectors: embed_batch docs        -- 1. einmal vektorisieren
question: "Was macht die VM schnell?"
q_vec: embed question            -- 2. Query einbetten
top: nearest q_vec vectors 2     -- 3. Top-K nach Ähnlichkeit

context: ""
for idx in top
    context: context ++ (at docs idx) ++ "\n---\n"

ask ("Context:\n" ++ context ++ "\nQuestion: " ++ question)
    > print
```

Was hier passiert:

- **`embed_batch`** verwandelt jedes Dokument in einen Vektor — kein `pip install`, keine Vektor-DB, keine Index-Konfiguration.
- **`nearest`** liefert die *Indizes* der relevantesten Dokumente per Kosinus-Ähnlichkeit. Bedeutung statt Keywords.
- Die Treffer werden zu einem Kontext-Block verbunden und **`ask`** antwortet darauf — klassisches RAG.

In Python + LangChain ist dieselbe Pipeline ~80 Zeilen über mehrere SDKs. In Pipe sind es ein Dutzend — und es funktioniert mit **jedem Provider**: Tausche `"deepseek"` gegen `"openai"`, `"anthropic"` oder `"ollama"`, sonst ändert sich nichts.

Die Vollversion mit Web-UI, SQLite-Persistenz und `/search`-API? Siehe [`examples/rag_knowledge_base.pipe`](https://github.com/MachuraHarry/pipe/blob/master/examples/rag_knowledge_base.pipe) — oder starte die Minimalversion oben mit `export DEEPSEEK_API_KEY=... && pipe examples/blog_rag.pipe`.
[/lang]
