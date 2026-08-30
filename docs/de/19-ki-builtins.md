# 19. KI-Builtins

Pipe bietet **40 KI-Builtins** für die Arbeit mit Large Language Models.
Die Kommunikation läuft über REST-APIs zu OpenAI, Anthropic oder DeepSeek.

---

## 19.1 Konfiguration

### Provider wählen

```pipe
-- OpenAI (Standard)
ai_provider "openai"
-- Anthropic Claude
ai_provider "anthropic"
-- DeepSeek
ai_provider "deepseek"
-- OpenRouter (400+ Modelle über einen Key, Modell-Slugs wie "openai/gpt-4o-mini")
ai_provider "openrouter"
-- OpenCode Zen (kostenlose Modelle ohne Key, bezahlte mit OPENCODE_API_KEY)
ai_provider "opencode"
```

### Modell und Timeout

Jeder Provider nutzt ein günstiges und schnelles Standard-Modell. Überschreiben
kannst du es mit einem Config-Block bei `ai_provider` oder jederzeit mit `ai_model`:

```pipe
-- Standard (günstigstes & schnellstes Modell pro Provider):
--   openai      → gpt-4o-mini
--   anthropic   → claude-3-5-haiku-20241022
--   deepseek    → deepseek-v4-flash
--   ollama      → llama3.1:8b
--   openrouter  → openrouter/free
--   opencode    → big-pickle

-- Provider setzen + Modell/Timeout in einem Zug überschreiben
ai_provider "deepseek" {model: "deepseek-v4-flash", timeout: 120}

-- Spätere Overrides (wirken sofort)
ai_model "deepseek-v4-flash"
ai_host "https://api.deepseek.com"
ai_timeout 60
```

### Thinking & Reasoning-Effort (DeepSeek V4)

DeepSeek V4 macht Thinking-Modus und abgestuften Reasoning-Effort über
Request-Parameter steuerbar. Pipe mappt sie auf die `ai_provider`-Config-Keys
`thinking` und `effort`:

```pipe
-- Thinking-Modus AN + hoher Effort (V4-Standard: enabled/high)
ai_provider "deepseek" {model: "deepseek-v4-flash", thinking: true, effort: "high"}

-- Thinking AUS (schnell & günstig, keine Reasoning-Spur)
ai_provider "deepseek" {model: "deepseek-v4-flash", thinking: false}

-- Maximaler Effort für schwere Agent-Aufgaben
ai_provider "deepseek" {model: "deepseek-v4-flash", effort: "max"}

-- "none" deaktiviert Thinking komplett (wie thinking: false)
ai_provider "deepseek" {model: "deepseek-v4-flash", effort: "none"}
```

Zulässige `effort`-Werte werden unverändert weitergereicht; das finale Mapping
macht DeepSeek serverseitig (`low`/`medium` → `high`, `xhigh` → `max` auf pro):
`low`, `medium`, `high`, `xhigh`, `max`, `none`.

Hinweise:

- Beide Keys sind **nur DeepSeek**; bei anderen Providern gibt es einen Fehler.
  Kombinierbar mit `model`/`host`/`timeout` in einem Block.
- Im Thinking-Modus haben `temperature`, `top_p`, `presence_penalty` und
  `frequency_penalty` keine Wirkung.
- Thinking-Tokens zählen ins Completion-Budget — `max_tokens` großzügig setzen
  (`ai_chat`/`ai_with_tools`), sonst endet der Request mit Reasoning, aber ohne
  finale Antwort.
- Tool-Calls im Thinking-Modus (`ai_with_tools`) senden `reasoning_content`
  automatisch zurück; die API verlangt das und liefert sonst einen 400-Fehler.

### API-Keys

API-Keys werden über Umgebungsvariablen gesetzt:

```bash
export OPENAI_API_KEY="sk-..."
export ANTHROPIC_API_KEY="sk-ant-..."
export DEEPSEEK_API_KEY="sk-..."
```

Der Provider wird automatisch anhand des gesetzten Keys erkannt. Fehlt der Key,
schlägt die Anfrage mit einem Fehler fehl.

**Ausnahme — `opencode`:** funktioniert auf den Free-Tier-Modellen (`big-pickle`,
`*-free`) **ganz ohne Key**. `OPENCODE_API_KEY` ist nur für bezahlte Modelle oder
höhere Rate-Limits nötig. Achtung: Free-Tier-Prompts dürfen zum Modelltraining
verwendet werden — sensible Daten gehören dort nicht hinein.

---

## 19.2 Übersichtstabelle

| Builtin | Beschreibung | Signatur |
|---------|-------------|----------|
| `ai_provider` | Provider setzen (+ optionaler Modell/Host/Timeout/Thinking/Effort-Block) | `ai_provider name {model, host, timeout, thinking, effort}?` |
| `ai_model` | Modell setzen | `ai_model name` |
| `ai_timeout` | Timeout setzen | `ai_timeout sekunden` |
| `ai_host` | Host-URL setzen | `ai_host url` |
| `ai_chat` | Low-Level Chat | `ai_chat system prompt` |
| `ai_chat_json` | Chat → JSON | `ai_chat_json system prompt` |
| `ai_stream` | Live-Streaming | `ai_stream system prompt` |
| `ai_parallel` | Parallel-Calls | `ai_parallel n system items` |
| `ai_batch` | Auto-Parallel | `ai_batch system items` |
| `ai_rate_limit` | Rate-Limiting | `ai_rate_limit calls_per_sec` |
| `embed` | Text → Vektor | `embed text` |
| `embed_batch` | Batch-Embedding | `embed_batch texts` |
| `cosine_sim` | Kosinus-Ähnlichkeit | `cosine_sim a b` |
| `dot_product` | Skalarprodukt | `dot_product a b` |
| `nearest` | Top-K ähnlichste | `nearest query docs k` |
| `ai_tool` | Tool registrieren | `ai_tool name desc schema fn` |
| `ai_with_tools` | Chat mit Tools | `ai_with_tools system user` |
| `swarm_agent` | Swarm-Mitglied registrieren | `swarm_agent name {system, tools, handoff}` |
| `ai_swarm` | Handoff-Swarm ausführen | `ai_swarm task entry_agent` |
| `ai_swarm_trace` | Swarm ausführen, mit Trace | `ai_swarm_trace task entry_agent` |
| `ai_vision` | Frage zu einem Bild beantworten | `ai_vision image prompt` |
| `summarize` | Text zusammenfassen | `summarize text` |
| `translate` | Text übersetzen | `translate text zielsprache` |
| `classify` | Text klassifizieren | `classify text kategorien` |
| `extract` | Daten extrahieren | `extract text schema` |
| `generate` | Freitext generieren | `generate prompt` |
| `ask` | Frage beantworten | `ask frage` |
| `ai_cost` | Kosten-Metriken | `ai_cost` |
| `ai_tokens` | Gesamte Tokens | `ai_tokens` |
| `ai_cache_hits` | Cache-Treffer | `ai_cache_hits` |
| `ai_cache_misses` | Cache-Fehltreffer | `ai_cache_misses` |

---

## 19.3 High-Level Funktionen

### summarize

```pipe
-- Signatur
summarize text

-- Beschreibung
-- Fasst den übergebenen Text in 2–3 Sätzen zusammen.

-- Beispiel
text: "Pipe ist eine moderne Skriptsprache mit Fokus auf..."
print (summarize text)
-- -> Pipe ist eine Skriptsprache mit Pipeline-Operator und Bytecode-VM.
--   Sie unterstützt 168 Builtins und ist für Datenverarbeitung optimiert.
```

### translate

```pipe
-- Signatur
translate text zielsprache

-- Beschreibung
-- Übersetzt den Text in die angegebene Zielsprache.

-- Beispiel
text: "The quick brown fox jumps over the lazy dog."
print (translate text "German")
-- -> Der schnelle braune Fuchs springt über den faulen Hund.
```

### classify

```pipe
-- Signatur
classify text kategorien

-- Beschreibung
-- Ordnet den Text einer der angegebenen Kategorien zu.
-- kategorien kann ein String (kommagetrennt) oder eine Liste sein.

-- Beispiel 1: String
ergebnis: classify "Apple stellt neues iPhone vor" "Tech, Sport, Politik"
-- -> Tech
print ergebnis

-- Beispiel 2: Liste
kategorien: ["dringend", "niedrig", "mittel"]
print (classify "Server ausgefallen seit 2 Stunden" kategorien)
-- -> dringend
```

### extract

```pipe
-- Signatur
extract text schema

-- Beschreibung
-- Extrahiert strukturierte Daten als Map aus dem Text.
-- schema beschreibt die gewünschten Felder.

-- Beispiel
text: "Max Mustermann, geboren am 15.03[1985] in Berlin,
       arbeitet als Softwareentwickler und verdient 75[000] €."
schema: { name: "str", geburtsjahr: "num", stadt: "str", beruf: "str", gehalt: "num" }
daten: extract text schema
-- -> Max Mustermann
print daten.name
-- -> 1985
print daten.geburtsjahr
-- -> Berlin
print daten.stadt
-- -> Softwareentwickler
print daten.beruf
-- -> 75000
print daten.gehalt
```

### generate

```pipe
-- Signatur
generate prompt

-- Beschreibung
-- Generiert Freitext basierend auf dem Prompt.

-- Beispiel
prompt: "Schreibe eine kurze Produktbeschreibung für ein KI-Tool
         zur automatischen Code-Review."
print (generate prompt)
-- -> Automate your code review process with AI-powered analysis
--   that catches bugs, suggests improvements, and ensures code
--   quality — all before your PR reaches human reviewers.
```

### ask

```pipe
-- Signatur
ask frage

-- Beschreibung
-- Beantwortet eine Frage auf Basis des Modellwissens.

-- Beispiel
print (ask "Was ist der Unterschied zwischen Stack und Heap?")
-- -> Der Stack ist ein LIFO-Speicherbereich für lokale Variablen
--   und Funktionsaufrufe mit schneller Allokation. Der Heap ist
--   ein dynamischer Speicherbereich für langlebige Objekte mit
--   manueller oder automatischer Speicherbereinigung.
```

---

## 19.4 Low-Level Funktionen

### ai_chat

```pipe
-- Signatur
ai_chat system_prompt user_prompt [max_tokens]

-- Beschreibung
-- Direkter Chat mit einem System-Prompt und einem User-Prompt.
-- Gibt die rohe Antwort des Modells als String zurück.
-- Das optionale dritte Argument begrenzt die Antwort auf max_tokens Tokens.

-- Beispiel
antowrt: ai_chat
    "Du bist ein Bash-Experte. Antworte nur mit dem fertigen Befehl."
    "Wie liste ich alle Dateien größer als 100 MB rekursiv auf?"

print antwort
-- -> find . -type f -size +100M -exec ls -lh {} \;
```

### ai_chat_json

```pipe
-- Signatur
ai_chat_json system_prompt user_prompt [max_tokens]

-- Beschreibung
-- Wie ai_chat, aber die Antwort wird als JSON geparst.
-- Gibt eine Map oder List zurück, je nach JSON-Struktur.
-- Das optionale dritte Argument begrenzt die Antwort auf max_tokens Tokens.

-- Beispiel 1: Map
antwort: ai_chat_json
    "Du bist ein Datenanalyst. Antworte ausschließlich mit JSON."
    "Gib die Top-3-Programmiersprachen 2025 mit Anteil in Prozent."

-- -> Python
print antwort.erste
-- -> 34
print antwort.anteil_py

-- Beispiel 2: List
sprachen: ai_chat_json
    "Antworte als JSON-Liste von Sprach-Strings."
    "Nenne 5 funktionale Programmiersprachen."

for sprache in sprachen
    print sprache
```

---

## 19.5 Streaming

Streaming gibt KI-Antworten **Token für Token in Echtzeit** aus — der Nutzer sieht
die Antwort erscheinen wie bei ChatGPT, statt auf den kompletten Text zu warten.

### ai_stream

```
ai_stream system_prompt user_prompt
```

Streamt die Antwort live auf stdout und gibt den vollständigen Text als String zurück.

```pipe
-- Live-Gedicht — erscheint Wort für Wort
ai_stream "Du bist ein Dichter." "Schreibe ein kurzes Gedicht."

-- Live-Übersetzung mit Weiterverarbeitung
text: ai_stream "Übersetze ins Deutsche." "Hello world, this is streaming."
print ("Übersetzung: " ++ text)
```

**Verhalten:**
- Tokens werden **sofort** auf stdout ausgegeben (Live-Erlebnis)
- Der vollständige Antworttext wird als **Rückgabewert** geliefert
- Nutzbar in Pipelines: `> upper > print` verarbeitet den kompletten gesammelten Text

```pipe
-- Streaming + Pipeline (Text wird live angezeigt, dann weiterverarbeitet)
ai_stream "Du bist ein Übersetzer." "Translate: Hello world"
    > upper
    > print
```

---

## 19.6 Parallele Calls

Parallele KI-Calls verarbeiten mehrere Texte gleichzeitig und reduzieren
die Gesamtlaufzeit drastisch.

### ai_batch

```
ai_batch system_prompt items
```

Verarbeitet eine Liste von Texten parallel. Die Concurrency wird automatisch
gewählt (CPU-Kerne × 2). Gibt eine Liste mit den Antworten zurück.

```pipe
texte: ["Text 1", "Text 2", "Text 3", "Text 4", "Text 5"]
ergebnisse: ai_batch "Fasse jeden Text in einem Satz zusammen." texte

for e in ergebnisse
    print e
```

### ai_parallel

```
ai_parallel concurrency system_prompt items
```

Wie `ai_batch`, aber mit explizit steuerbarer Parallelität.

```pipe
-- Maximal 3 parallele Calls
fragen: ["Hello world", "Good morning", "How are you?"]
antworten: ai_parallel 3 "Uebersetze ins Deutsche." fragen
```

### ai_rate_limit

```
ai_rate_limit calls_per_second
```

Begrenzt die Anzahl der API-Calls pro Sekunde (globaler Token-Bucket).

```pipe
-- Max 5 Calls pro Sekunde
ai_rate_limit 5

-- 100 Texte verarbeiten, aber mit Rate-Limit
texte: read_lines "daten.txt"
ergebnisse: ai_batch "Analysiere diesen Text." texte
```

### Performance-Vergleich

```
3 Texte sequential:  14s
3 Texte ai_batch:     6s  (2.3× schneller)
5 Fragen, 5 concurrent: 1s  (massiv parallel)
```

---

## 19.7 Embeddings & Vektorsuche

Embeddings wandeln Text in Vektoren (Zahlenlisten) um, die die *Bedeutung*
des Texts im mathematischen Raum abbilden. Ähnliche Texte liegen nah beieinander.
Das ermöglicht **semantische Suche** — nicht Keyword-Matching, sondern echtes
Bedeutungs-Verständnis.

### embed

```
embed text
```

Wandelt einen Text in einen Embedding-Vektor um (Liste von ~1536 Float-Zahlen).
Nutzt die `/v1/embeddings` API (OpenAI-kompatibel, Modell: `text-embedding-3-small`).

```pipe
vec: embed "Pipe ist eine Skriptsprache."
-- 1536
print (len vec)
```

**Hinweis:** Benötigt `OPENAI_API_KEY`. DeepSeek unterstützt keine Embeddings.

### embed_batch

```
embed_batch texts
```

Berechnet Embeddings für eine Liste von Texten parallel (4 concurrent).

```pipe
dokumente: ["Text A", "Text B", "Text C"]
vektoren: embed_batch dokumente
```

### cosine_sim

```
cosine_sim vector_a vector_b
```

Berechnet die Kosinus-Ähnlichkeit zweier Vektoren. Werte von -1 (entgegengesetzt)
bis 1 (identische Richtung). **Kein API-Call nötig** — pure Mathematik.

```pipe
sim: cosine_sim vec1 vec2
-- z.B. 0[87] (sehr ähnlich)
print sim
```

### dot_product

```
dot_product vector_a vector_b
```

Skalarprodukt zweier Vektoren. **Kein API-Call nötig.**

```pipe
dp: dot_product ([1, 2, 3]) ([2, 4, 6])
-- 28
print dp
```

### nearest

```
nearest query_vector document_vectors k
```

Findet die Top-K ähnlichsten Dokument-Vektoren zu einem Query-Vektor.
Gibt die **Indizes** der ähnlichsten Dokumente zurück.

```pipe
-- Welche Dokumente sind der Frage am ähnlichsten?
frage: embed "Wie funktioniert die VM?"
top3: nearest frage vektoren 3

for idx in top3
    print (at dokumente idx)
```

### RAG mit Pipe (vollständiges Beispiel)

```pipe
-- 1. Wissensdatenbank einbetten
dokumente: read_lines "wissen.txt"
vektoren: embed_batch dokumente

-- 2. Frage einbetten
frage: "Wie funktioniert der Compiler?"
frage_vec: embed frage

-- 3. Relevante Dokumente finden
top: nearest frage_vec vektoren 5
kontext: ""
for idx in top
    kontext: kontext ++ (at dokumente idx) ++ "\n---\n"

-- 4. KI mit Kontext antworten
prompt: "Kontext:\n" ++ kontext ++ "\nFrage: " ++ frage
ask prompt > print
```

---

## 19.8 Tool-Calling (Function Calling)

Tool-Calling erlaubt dem LLM, Pipe-Funktionen als **Werkzeuge** aufzurufen.
Statt nur zu antworten, kann die KI aktiv handeln — APIs aufrufen, Berechnungen
ausführen, Daten abfragen.

### ai_tool

```
ai_tool name description parameters_schema function
```

Registriert eine Pipe-Funktion als Tool für das LLM. Das `parameters_schema`
ist eine Map, die die erwarteten Parameter beschreibt.

```pipe
-- Wetter-Tool definieren
fn get_wetter stadt
    match stadt
        | "Berlin" -> "22°C, sonnig"
        | "London" -> "15°C, regnerisch"
        | _ -> "Keine Daten"

-- Tool registrieren
schema: {stadt: "Name der Stadt"}
ai_tool "get_wetter" "Holt das Wetter für eine Stadt" schema get_wetter
```

### ai_with_tools

```
ai_with_tools system_prompt user_prompt
```

Führt einen Chat mit Zugriff auf alle registrierten Tools durch.
Der LLM entscheidet selbstständig, ob und wann er Tools aufruft.

```pipe
ergebnis: ai_with_tools
    "Du bist ein Wetter-Assistent. Nutze get_wetter."
    "Wie ist das Wetter in Berlin und London?"

print ergebnis
```

### Ablauf (intern)

```
1. User: "Wetter in Berlin?"
2. LLM → Tool Call: get_wetter("Berlin")
3. Pipe führt get_wetter aus → "22°C, sonnig"
4. Ergebnis wird an LLM zurückgesendet
5. LLM → Antwort: "In Berlin sind es 22°C und sonnig."
```

Dieser Loop wiederholt sich bis zu 5 Runden, bis der LLM eine finale
Antwort ohne weitere Tool-Calls gibt.

### Wetter-Assistent (vollständig)

```pipe
fn get_wetter stadt
    match stadt
        | "Berlin" -> "22 Grad, sonnig"
        | "London" -> "15 Grad, regnerisch"
        | "Paris" -> "25 Grad, klar"
        | _ -> stadt ++ ": Keine Daten"

schema: {stadt: "Name der Stadt"}
ai_tool "get_wetter" "Wetter für eine Stadt abrufen" schema get_wetter

ergebnis: ai_with_tools "Du bist ein Wetter-Experte." "Wie ist das Wetter in Berlin, London und Paris?"
print ergebnis
```

---

## 19.9 Pipeline mit KI

KI-Operationen lassen sich nahtlos in Pipe-Pipelines integrieren:

```pipe
-- Text einlesen und zusammenfassen
read_file "protokoll.txt"
    > summarize
    > upper
    > print

-- Übersetzung mit Nachbearbeitung
"Hallo Welt, wie geht es dir?"
    > translate "English"
    > split " "
    > len
    > print
-- -> 6 (Wörter im übersetzten Text)

-- Klassifizierung im Datenfluss
dokumente: read_lines "emails.txt"
ergebnisse: map dokumente fn doc
    klass: classify doc "intern, extern, spam, support"
    { doc: doc, type: klass }
```

---

## 19.10 Fehlerbehandlung

KI-Operationen können fehlschlagen — etwa bei fehlendem API-Key,
Netzwerkproblemen oder Timeout. Mit `try`/`catch` lassen sich
diese Fehler abfangen:

```pipe
-- Fehlender API-Key
try
    print (summarize "Ein langer Text...")
catch e
    print "KI-Fehler: " ++ e
    print "Bitte API-Key setzen!"

-- Timeout abfangen
-- kurzer Timeout
ai_timeout 5
try
    print (generate "Schreibe einen 10-seitigen Aufsatz über Philosophie")
catch e
    if contains e "timeout"
        print "Anfrage hat zu lange gedauert"
    else
        raise e

-- Fallback-Strategie
try
    ai_provider "openai"
    print (ask "Was ist Monade?")
catch e
    try
        ai_provider "anthropic"
        print (ask "Was ist Monade?")
    catch e2
        print "Alle Provider fehlgeschlagen: " ++ e2.message
```

---

## 19.11 Provider-Details

| Provider | API-Endpunkt | Standardmodell | Umgebungsvariable |
|----------|-------------|----------------|-------------------|
| OpenAI | `https://api.openai.com/v1/chat/completions` | `gpt-4o-mini` | `OPENAI_API_KEY` |
| Anthropic | `https://api.anthropic.com/v1/messages` | `claude-3-5-sonnet-20241022` | `ANTHROPIC_API_KEY` |
| DeepSeek | `https://api.deepseek.com/v1/chat/completions` | `deepseek-v4-flash` | `DEEPSEEK_API_KEY` |
| **Ollama** | `http://localhost:11434/v1/chat/completions` | `llama3.1:8b` | **Kein Key nötig!** |
| OpenCode Zen | `https://opencode.ai/zen/v1/chat/completions` | `big-pickle` | optional: `OPENCODE_API_KEY` |

### Ollama — Lokale KI ohne Cloud

Ollama ist ein **lokaler LLM-Server**, der auf deinem eigenen Rechner läuft.
Pipe kommuniziert über die OpenAI-kompatible API auf `localhost:11434`.

```bash
# Ollama installieren
curl -fsSL https://ollama.com/install.sh | sh

# Modell laden
ollama pull llama3.2:3b
```

```pipe
ai_provider "ollama"
ai_model "llama3.2:3b"

ask "Was ist eine Pipeline?" > print
```

**Vorteile:**
- Keine API-Keys, keine Registrierung
- Keine Daten verlassen dein System (DSGVO/Compliance)
- Funktioniert komplett offline
- Kostenlos, unbegrenzte Nutzung
- Alle 36 KI-Builtins funktionieren mit Ollama

**Remote Ollama** (z.B. im Firmennetzwerk):
```pipe
ai_provider "ollama"
ai_host "http://192.168.1.50:11434"
ai_model "qwen2.5-coder:7b"
```

---

## 19.12 Kosten-Tracking & Token-Verbrauch

Pipe verfolgt **Kosten und Token-Verbrauch** für jeden KI-Aufruf. Das ist
wichtig für den Betrieb von Agenten im großen Stil, für Budgets oder den
Vergleich von Providern.

### ai_cost

```
ai_cost
```

Gibt eine Map mit den kumulierten Metriken des aktuellen Laufs zurück:

| Key | Typ | Beschreibung |
|-----|------|-------------|
| `cost_usd` | num | Gesamtkosten in USD |
| `calls` | int | Anzahl der KI-API-Aufrufe |
| `cache_hits` | int | Antworten aus dem Cache |
| `cache_misses` | int | Antworten vom Provider geladen |

`ai_cost "reset"` setzt alle Metriken zurück.

```pipe
print (ai_cost)
-- -> {calls: 1, cost_usd: 0.000045, cache_hits: 0, cache_misses: 1}
```

### ai_tokens

```
ai_tokens
```

Gibt die Gesamtzahl der Tokens zurück, die KI-Aufrufe im aktuellen Lauf
verbraucht haben.

### ai_cache_hits / ai_cache_misses

```
ai_cache_hits
ai_cache_misses
```

Liefern die Zahl der Cache-Treffer bzw. Fehltreffer. Wiederholte identische
Prompts werden aus dem Antwort-Cache bedient – das spart Geld und Latenz:

```pipe
ask "Was ist ein Monad?" > print
-- -> Cache-Miss
ask "Was ist ein Monad?" > print
-- -> Cache-Hit (gleicher Prompt, aus dem Cache)

print (ai_cache_hits)   -- 1
print (ai_cache_misses) -- 1
```

### Kosten-Trace (CLI)

Nach einem Lauf gibt das CLI einen **Kosten-Trace** auf stderr aus, sobald
KI-Aufrufe stattgefunden haben:

```
═══ Cost Trace ═══
Total cost:    $0.000045
Total tokens:  50
API calls:     1
Cache hits:    0 | misses: 0
  #1 deepseek/deepseek-chat | 50 Tokens | $0.000045
══════════════════════════
```

### Budget-Durchsetzung

KI-Budgets lassen sich über Sandbox-Profile durchsetzen – siehe
[22. Sandbox-Profile](22-sandbox-profile.md) (der `budget`-Key und das
`budget_spent`-Builtin).

### Preise (Schätz-Fallback)

Liefert der Provider keine Kosten, schätzt Pipe sie aus den Token-Zahlen mit
folgenden Preisen pro 1K Tokens:

| Provider / Modell | Prompt ($ / 1K Tokens) | Completion ($ / 1K Tokens) |
|------------------|------------------------|----------------------------|
| OpenAI `gpt-4o-mini` | 0.00015 | 0.0006 |
| OpenAI `gpt-4` | 0.03 | 0.06 |
| OpenAI (andere) | 0.005 | 0.015 |
| DeepSeek | 0.0009 | 0.0009 |
| Anthropic Claude Haiku | 0.0008 | 0.004 |
| Anthropic Claude Sonnet | 0.003 | 0.015 |
| Anthropic Claude Opus | 0.015 | 0.075 |
| Ollama | 0 | 0 |

---

## 19.13 Beispiel: Vollständiger Workflow

Dieser Workflow demonstriert alle KI-Builtins in einer praktischen Anwendung:

```pipe
-- ai_demo.pipe — KI-Workflow mit allen Builtins
-- Nutzung: OPENAI_API_KEY="sk-..." pipe ai_demo.pipe

ai_provider "openai"
ai_model "gpt-4o-mini"
ai_timeout 90

print "=== 1. Text generieren ==="
beschreibung: generate "Beschreibe die Programmiersprache Rust in 2 Sätzen"
print beschreibung

print "\n=== 2. Zusammenfassen ==="
zusammenfassung: summarize beschreibung
print zusammenfassung

print "\n=== 3. Übersetzen ==="
uebersetzung: translate beschreibung "German"
print uebersetzung

print "\n=== 4. Klassifizieren ==="
kategorie: classify beschreibung "Programmiersprachen, Hardware, Netzwerke, Datenbanken"
print "Kategorie: " ++ kategorie

print "\n=== 5. Daten extrahieren ==="
lebenslauf: "
  Anna Schmidt, Senior Developerin bei TechCorp in München.
  Sie hat 12 Jahre Erfahrung mit Python, Go und Rust.
  Ihre E-Mail ist anna@techcorp.de und sie verdient 95[000] € pro Jahr."
schema: {name: "str", firma: "str", erfahrung_jahre: "num", sprachen: "list", email: "str", gehalt: "num"}
daten: extract lebenslauf schema
print "Name:      " ++ daten.name
print "Firma:     " ++ daten.firma
print "Erfahrung: " ++ (to_str daten.erfahrung_jahre) ++ " Jahre"
print "Sprachen:  " ++ (join daten.sprachen ", ")
print "E-Mail:    " ++ daten.email
print "Gehalt:    " ++ (to_str daten.gehalt) ++ " €"

print "\n=== 6. Frage beantworten ==="
frage: "Welche dieser Sprachen — " ++ (join daten.sprachen ", ") ++ " — eignet sich am besten fuer Systemprogrammierung und warum?"
antwort: ask frage
print antwort

print "\n=== 7. Low-Level Chat ==="
analyse: ai_chat
    "Du bist ein Karriereberater. Analysiere das Profil und gib 3 Empfehlungen.
     Antworte mit nummerierten Punkten."
    "Profil: " ++ lebenslauf

print analyse

print "\n=== 8. Strukturierte JSON-Antwort ==="
empfehlungen: ai_chat_json
    "Antworte als JSON-Objekt mit den Feldern: stärken (Liste),
     verbesserung (Liste), nächster_karriereschritt (String),
     gehaltsprognose (Zahl)."
    "Analysiere: " ++ lebenslauf

print "Stärken:"
for s in empfehlungen.staerken
    print "  - " ++ s

print "Verbesserungspotenzial:"
for v in empfehlungen.verbesserung
    print "  - " ++ v

print "Naechster Schritt: " ++ empfehlungen.naechster_karriereschritt
print "Gehaltsprognose:  " ++ (to_str empfehlungen.gehaltsprognose) ++ " €"

print "\n=== 9. Fehlerbehandlung im Workflow ==="
try
    ai_provider "deepseek"
    print (ask "Was ist TDD?")
catch e
    print "DeepSeek nicht verfügbar: " ++ e
    print "Falle zurück auf OpenAI..."
    ai_provider "openai"
    print (ask "Was ist TDD?")

print "\n=== Workflow abgeschlossen ==="
```

---

## 19.14 KI-Swarms

Ein **Swarm** ist eine Menge benannter Agenten, jeder mit eigenem System-Prompt
und eigenen Tools, die sich eine Konversation gegenseitig zuschieben können.
Immer ein Agent ist "aktiv"; ruft er ein reserviertes Handoff-Tool auf,
übernimmt der nächste Agent — mit dem **kompletten Gesprächsverlauf**, sodass
beim Handoff kein Kontext verloren geht. Das ist dasselbe Pattern, das OpenAIs
ursprüngliche "Swarm"-Bibliothek populär gemacht hat (Poststelle → Spezialist).

Swarms bauen direkt auf `ai_tool`/`ai_with_tools` (Abschnitt 19.8) auf — die
`tools` eines Swarm-Agenten sind ganz normale, bereits per `ai_tool`
registrierte Tool-Namen.

**Einschränkung:** wie `ai_with_tools` braucht auch `ai_swarm` eine
OpenAI-kompatible Tool-Calling-API — `openai`, `deepseek`, `ollama`,
`openrouter` oder `opencode`. `anthropic` wird für `ai_swarm` nicht
unterstützt.

### swarm_agent

```
swarm_agent name config
```

Registriert ein Swarm-Mitglied. `config` ist ein Block mit:

| Schlüssel | Typ | Pflicht | Beschreibung |
|-----------|-----|---------|--------------|
| `system` | string | ja | Der System-Prompt des Agenten. |
| `tools` | Liste von Strings | nein | Namen bereits per `ai_tool` registrierter Tools. |
| `handoff` | Liste von Strings | nein | Namen anderer Swarm-Agenten, an die dieser Agent die Konversation weiterreichen darf. |

Die Registrierungsreihenfolge spielt keine Rolle — `tools`- und
`handoff`-Namen werden erst beim Aufruf von `ai_swarm` aufgelöst, nicht bei
`swarm_agent`.

```pipe
swarm_agent "triage"
    { system: "Leite Rechnungsfragen an 'billing' weiter, technische Fragen an 'tech'. Alles andere beantwortest du selbst.",
      handoff: ["billing", "tech"] }

swarm_agent "billing"
    { system: "Du beantwortest Rechnungsfragen präzise und mit Zahlen.",
      tools: ["get_invoice"],
      handoff: ["triage"] }
```

### ai_swarm

```
ai_swarm task entry_agent [max_rounds]
```

Führt den Swarm ausgehend von `entry_agent` mit `task` als Nutzer-Nachricht
aus. Gibt die Antwort des letzten Agenten als reinen String zurück —
pipeline-freundlich, genau wie `ask`/`generate`/`ai_with_tools`. `max_rounds`
ist standardmäßig 5 und begrenzt, wie viele Chat-/Tool-Call-Runden der
gesamte Swarm-Lauf maximal nutzen darf (nicht pro Agent).

```pipe
ai_provider "deepseek"

"Meine Rechnung diesen Monat stimmt nicht"
    > ai_swarm "triage"
    > print
```

### ai_swarm_trace

```
ai_swarm_trace task entry_agent [max_rounds]
```

Wie `ai_swarm`, gibt aber eine Map `{content, path, rounds}` statt eines
reinen Strings zurück — nützlich, um nachzuvollziehen, welche Agenten eine
Anfrage tatsächlich bearbeitet haben:

```pipe
result: ai_swarm_trace "Meine Rechnung diesen Monat stimmt nicht" "triage"
print result.content
print result.path
-- -> ["triage", "billing"]
print result.rounds
-- -> 2
```

### Vollständiges Beispiel: Triage-Swarm

```pipe
ai_provider "deepseek"

fn get_invoice kunde
    "Rechnung Nr. 4471, 49.90€, fällig 15.09.2026."

fn restart_service dienst
    dienst ++ " wurde neu gestartet."

ai_tool "get_invoice" "Aktuelle Rechnung eines Kunden abrufen" {kunde: "Kundenname"} get_invoice
ai_tool "restart_service" "Einen benannten Dienst neu starten" {dienst: "Dienstname"} restart_service

swarm_agent "triage"
    { system: "Du bist die Poststelle. Leite Rechnungsfragen an 'billing', technische Probleme an 'tech'. Alles andere beantwortest du selbst.",
      handoff: ["billing", "tech"] }

swarm_agent "billing"
    { system: "Du beantwortest Rechnungsfragen mit get_invoice.",
      tools: ["get_invoice"],
      handoff: ["triage"] }

swarm_agent "tech"
    { system: "Du bist technischer Support. Nutze restart_service, wenn um eine Reparatur gebeten wird.",
      tools: ["restart_service"],
      handoff: ["triage"] }

"Was steht auf meiner letzten Rechnung?"
    > ai_swarm "triage"
    > print

"Der Login-Dienst ist down, bitte neu starten"
    > ai_swarm "triage"
    > print
```

### Sandbox

`ai_swarm`/`ai_swarm_trace` werden genau wie jeder andere KI-Call gegatet:
Unter einem registrierten Profil blockiert `ai: false`; unter dem CLI-Flag
`--sandbox` (ohne Profil) ist `--allow-ai` erforderlich. Tool-Aufrufe während
eines Swarm-Laufs durchlaufen dieselbe Sandbox-Durchsetzung wie
`ai_with_tools` (`CanToolCall`, Audit-Log). Siehe
[22. Sandbox-Profile](22-sandbox-profiles.md).

---

## 19.15 KI-Vision

`ai_vision` beantwortet eine Frage zu einem Bild mit einem vision-fähigen
Modell (z. B. DeepSeeks `deepseek-v4-flash-vision-exp`).

**Einschränkung:** braucht ein OpenAI-kompatibles Modell — `openai`,
`deepseek`, `ollama`, `openrouter` oder `opencode`. `anthropic` nutzt ein
anderes Bild-Content-Block-Format und wird nicht unterstützt.

### ai_vision

```
ai_vision image prompt [max_tokens]
```

Gibt die Antwort des Modells als reinen String zurück — pipeline-freundlich,
genau wie `ask`/`generate`. `image` akzeptiert drei Formen:

| Form | Beispiel | Hinweis |
|------|----------|---------|
| http(s)-URL | `"https://example.com/foto.jpg"` | Wird unverändert an den Provider gesendet — dessen Server holt sie, nicht Pipe. |
| Lokaler Dateipfad | `"foto.jpg"` | Wird über dasselbe Sandbox-Lese-Gate wie `read_file` gelesen. |
| Rohe Bytes | `read_file "foto.jpg" "bytes"` | Nützlich, wenn die Bytes nicht direkt von der Platte kommen. |

Unterstützte Bildformate: JPEG, PNG, GIF, WebP (anhand des Inhalts erkannt,
nicht der Dateiendung). Lokale Dateien und rohe Bytes werden vor dem Senden
als `data:`-URL base64-kodiert.

```pipe
ai_provider "deepseek" {model: "deepseek-v4-flash-vision-exp"}

"foto.jpg"
    > ai_vision "Was ist auf diesem Bild zu sehen?"
    > print

"https://example.com/chart.png"
    > ai_vision "Fasse den Trend in diesem Chart zusammen."
    > print
```

### Sandbox

`ai_vision` wird genau wie `ai_chat`/`ai_swarm` gegatet: Unter einem
registrierten Profil blockiert `ai: false`; unter dem CLI-Flag `--sandbox`
(ohne Profil) ist `--allow-ai` erforderlich. Das Lesen eines lokalen
Bildpfads läuft über dasselbe Lese-Gate wie `read_file` (unberührt von
`--sandbox`, das nur Schreibzugriffe einschränkt). Siehe
[22. Sandbox-Profile](22-sandbox-profiles.md).
