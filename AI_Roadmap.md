# Pipe AI — Roadmap

> Vision: Pipe als die natürliche Sprache für KI-Workflows.
> Jeder LLM-Call ist eine Pipeline-Stufe, jede Transformation ein `>`.

---

## Phase 1: Streaming-Responses ⏳

### Motivation
Aktuell blockiert jeder KI-Call bis die komplette Antwort da ist. Bei langen
Antworten (Summaries, Code-Generierung) will der Nutzer die Antwort *live* sehen —
wie bei ChatGPT.

### Neue Builtins

| Builtin | Signatur | Beschreibung |
|---------|----------|-------------|
| `ai_stream` | `ai_stream system user` | Streamt die Antwort Zeichen für Zeichen via Callback |
| `ai_stream_json` | `ai_stream_json system user` | Streamt und sammelt, parsed final als JSON |

### Syntax

```pipe
-- Echtzeit-Ausgabe
ai_stream "Du bist ein Übersetzer." "Übersetze: Hello world" > print

-- Mit Zwischenverarbeitung
ai_stream "Du bist ein Dichter." "Schreibe ein kurzes Gedicht"
    > upper
    > print
```

### Callback-Mechanismus

Da Pipe kein natives Streaming kennt, wird der Callback über eine spezielle
Builtin-Funktion realisiert:

```pipe
fn on_token token
    print token    -- Wird für jedes neue Token aufgerufen

ai_stream "System" "User" on_token
```

### Implementierung
- `ai.Stream()` — neue Methode im Provider-Interface
- SSE-Parsing für OpenAI/DeepSeek (`data: {"choices":[{"delta":{"content":"..."}}]}`)
- Anthropic Streaming-Format (`data: {"type":"content_block_delta","delta":{"text":"..."}}`)
- Token-weise Ausgabe statt kompletter Antwort

### Provider-Support
| Provider | Streaming |
|----------|:---------:|
| OpenAI | ✅ `stream: true` |
| DeepSeek | ✅ OpenAI-kompatibel |
| Anthropic | ✅ SSE streaming |

---

## Phase 2: Parallele Calls ⏳

### Motivation
Eine KI-Pipeline über 100 Texte nacheinander auszuführen dauert Minuten.
Parallele Verarbeitung reduziert das auf Sekunden.

### Neue Builtins

| Builtin | Signatur | Beschreibung |
|---------|----------|-------------|
| `ai_parallel` | `ai_parallel n system_prompt items fn` | N parallele KI-Calls |
| `ai_batch` | `ai_batch system_prompt items fn` | Auto-parallel (CPU * 2) |
| `ai_rate_limit` | `ai_rate_limit calls_per_second` | Rate-Limiting setzen |

### Syntax

```pipe
-- 10 Texte parallel zusammenfassen
read_lines "articles.txt"
    > ai_batch "Fasse jeden Text in einem Satz zusammen." (fn text: summarize text)
    > save "summaries.txt"

-- Mit Rate-Limiting (max 5 Calls/Sekunde)
ai_rate_limit 5
read_lines "emails.txt"
    > ai_parallel 10 "Klassifiziere als spam/important/personal" classify
    > save "classified.txt"
```

### Implementierung
- `ai.ChatParallel()` — Goroutine-Pool mit `semaphore.Weighted`
- Konfigurierbare Parallelität (Default: `runtime.NumCPU() * 2`)
- Rate-Limiting via Token-Bucket (`golang.org/x/time/rate`)
- Fehlersammlung: partieller Erfolg statt Totalabbruch
- Ergebnis-Reihenfolge originalgetreu trotz Parallelität

### Pipeline-Integration

```pipe
["text1", "text2", ..., "text100"]
    > ai_map (fn t: summarize t)    -- Parallel map über KI
    > filter (fn s: len s > 10)     -- Kurze rausfiltern
    > print
```

---

## Phase 3: Embeddings + Vektorsuche ⏳

### Motivation
Embeddings sind der Schlüssel für semantische Suche, RAG (Retrieval Augmented
Generation) und Clustering. Ohne Embeddings bleibt KI oberflächlich — mit ihnen
wird sie kontextbewusst.

### Neue Builtins

| Builtin | Signatur | Beschreibung |
|---------|----------|-------------|
| `embed` | `embed text` | Embedding-Vektor für einen Text |
| `embed_batch` | `embed_batch texts` | Embeddings für Liste von Texten |
| `cosine_sim` | `cosine_sim a b` | Kosinus-Ähnlichkeit zweier Vektoren |
| `dot_product` | `dot_product a b` | Skalarprodukt |
| `nearest` | `nearest query docs k` | Top-K ähnlichste Dokumente |
| `vector_store` | `vector_store path` | Vektordatenbank laden/speichern |

### Syntax — RAG mit Pipe

```pipe
-- 1. Dokumente einbetten
docs: read_lines "wissensdatenbank.txt"
embeddings: embed_batch docs

-- 2. Query einbetten
query_vec: embed "Wie funktioniert der Bytecode-Compiler?"

-- 3. Relevanteste Dokumente finden
top_docs: nearest query_vec embeddings 5

-- 4. Mit Kontext antworten
context: join top_docs "\n---\n"
prompt: "Beantworte basierend auf folgendem Kontext:\n" ++ context ++ "\n\nFrage: Wie funktioniert der Bytecode-Compiler?"
ask prompt > print
```

### Implementierung
- OpenAI `/v1/embeddings` API (auch von DeepSeek unterstützt)
- Vektor als `[]float64` in Pipe-Liste
- `cosine_sim`: `dot(a,b) / (norm(a) * norm(b))`
- `nearest`: Brute-Force zunächst, später FAISS/Annoy-Integration
- `vector_store`: JSON-Serialisierung, optional LanceDB

### Anwendungsfälle
- **RAG**: Dokumente + Embeddings + KI = kontextbewusste Antworten
- **Semantische Suche**: Statt Keyword-Matching echte Bedeutungssuche
- **Clustering**: `nearest` als Baustein für Dokumenten-Clustering
- **Deduplizierung**: Ähnliche Texte via cosine_sim erkennen

---

## Phase 4: Tool-Calling / Function-Calling ⏳

### Motivation
KI-Modelle können nicht nur antworten, sondern auch *handeln* — APIs aufrufen,
Berechnungen ausführen, Dateien manipulieren. Tool-Calling macht die KI zum
aktiven Teilnehmer in der Pipeline.

### Neue Builtins

| Builtin | Signatur | Beschreibung |
|---------|----------|-------------|
| `ai_tool` | `ai_tool name description params fn` | Ein Tool registrieren |
| `ai_with_tools` | `ai_with_tools system user tools` | Chat mit Tool-Zugriff |
| `tool_def` | `tool_def name description schema` | Tool-Schema definieren |

### Syntax

```pipe
-- 1. Tools definieren
fn get_weather city
    url: "https://api.weather.com/" ++ city
    http_get_json url

fn calculate expression
    -- Führt mathematischen Ausdruck aus
    exec ("python3 -c 'print(" ++ expression ++ ")'")

-- 2. Tool-Schemas registrieren
weather_tool: tool_def "get_weather" "Holt Wetterdaten" {
    city: {type: "string", description: "Stadtname"}
}

calc_tool: tool_def "calculate" "Mathematische Berechnung" {
    expression: {type: "string", description: "Math expression"}
}

-- 3. Agent mit Tools
ai_with_tools "Du bist ein hilfreicher Assistent."
    "Wie ist das Wetter in Berlin? Und was ist 15 * 23?"
    [weather_tool, calc_tool]
    > print
```

### Ablauf (intern)
```
1. User: "Wie ist das Wetter in Berlin?"
2. LLM → Function Call: get_weather("Berlin")
3. Pipe führt get_weather aus → 22°C, sonnig
4. LLM → Antwort: "In Berlin sind es 22°C und sonnig."
```

### Implementierung
- OpenAI Function-Calling API (`tools` Parameter)
- Anthropic Tool-Use API (`tool_use` content blocks)
- Automatische Tool-Execution-Loop (bis zu N Runden)
- Tool-Ergebnisse als JSON in den Chat-Kontext einweben
- Sicherheit: Tool-Sandbox (kein exec/dateisystem ohne explizite Freigabe)

---

## Zusammenfassung

| Phase | Feature | Status | Builtins |
|-------|---------|:------:|----------|
| ✅ Done | Chat, Summarize, Translate, Classify, Extract, Generate, Ask | ✅ | 12 |
| **1** | **Streaming-Responses** | ⏳ | `ai_stream`, `ai_stream_json` |
| 2 | Parallele Calls | ⏳ | `ai_parallel`, `ai_batch`, `ai_rate_limit` |
| 3 | Embeddings + Vektorsuche | ⏳ | `embed`, `embed_batch`, `cosine_sim`, `nearest` |
| 4 | Tool-Calling | ⏳ | `ai_tool`, `ai_with_tools`, `tool_def` |

## Vision: Pipe als KI-Agenten-Plattform

Langfristig soll Pipe nicht nur KI-Calls machen, sondern vollständige
**KI-Agenten-Workflows** orchestrieren:

```pipe
-- Autonomer Research-Agent
"Was ist der Stand der Fusionsenergie-Forschung 2026?"
    > web_search          -- Sucht im Web (Tavily/Brave API)
    > fetch_pages         -- Lädt relevante Seiten
    > extract_key_facts   -- Extrahiert Fakten per KI
    > synthesize_report   -- Schreibt zusammenfassenden Bericht
    > translate "de"      -- Übersetzt
    > save "fusionsenergie_report.md"
```

Jede Stufe ist ein KI-Call oder ein Tool — orchestriert durch die Pipeline.
Kein Python, kein LangChain, kein CrewAI. Nur Pipe.
