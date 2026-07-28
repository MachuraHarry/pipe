# 19. KI-Builtins

Pipe bietet **12 KI-Builtins** für die Arbeit mit Large Language Models.
Die Kommunikation läuft über REST-APIs zu OpenAI, Anthropic oder DeepSeek.

---

## 19.1 Konfiguration

### Provider wählen

```pipe
ai_provider "openai"       -- OpenAI (Standard)
ai_provider "anthropic"    -- Anthropic Claude
ai_provider "deepseek"     -- DeepSeek
```

### Modell und Timeout

```pipe
ai_model "gpt-4o"           -- Modell setzen (Standard: gpt-4o-mini)
ai_timeout 120               -- Timeout in Sekunden (Standard: 60)
```

### API-Keys

API-Keys werden über Umgebungsvariablen gesetzt:

```bash
export OPENAI_API_KEY="sk-..."
export ANTHROPIC_API_KEY="sk-ant-..."
export DEEPSEEK_API_KEY="sk-..."
```

Der Provider wird automatisch anhand des gesetzten Keys erkannt. Fehlt der Key,
schlägt die Anfrage mit einem Fehler fehl.

---

## 19.2 Übersichtstabelle

| Builtin | Beschreibung | Signatur |
|---------|-------------|----------|
| `ai_provider` | Provider setzen | `ai_provider name` |
| `ai_model` | Modell setzen | `ai_model name` |
| `ai_timeout` | Timeout setzen | `ai_timeout sekunden` |
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
| `summarize` | Text zusammenfassen | `summarize text` |
| `translate` | Text übersetzen | `translate text zielsprache` |
| `classify` | Text klassifizieren | `classify text kategorien` |
| `extract` | Daten extrahieren | `extract text schema` |
| `generate` | Freitext generieren | `generate prompt` |
| `ask` | Frage beantworten | `ask frage` |

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
-- → Pipe ist eine Skriptsprache mit Pipeline-Operator und Bytecode-VM.
--   Sie unterstützt 80+ Builtins und ist für Datenverarbeitung optimiert.
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
-- → Der schnelle braune Fuchs springt über den faulen Hund.
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
print ergebnis     -- → Tech

-- Beispiel 2: Liste
kategorien: ["dringend", "niedrig", "mittel"]
print (classify "Server ausgefallen seit 2 Stunden" kategorien)
-- → dringend
```

### extract

```pipe
-- Signatur
extract text schema

-- Beschreibung
-- Extrahiert strukturierte Daten als Map aus dem Text.
-- schema beschreibt die gewünschten Felder.

-- Beispiel
text: "Max Mustermann, geboren am 15.03.1985 in Berlin,
       arbeitet als Softwareentwickler und verdient 75.000 €."
schema: { name: "str", geburtsjahr: "num", stadt: "str", beruf: "str", gehalt: "num" }
daten: extract text schema
print daten.name         -- → Max Mustermann
print daten.geburtsjahr  -- → 1985
print daten.stadt        -- → Berlin
print daten.beruf        -- → Softwareentwickler
print daten.gehalt       -- → 75000
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
-- → Automate your code review process with AI-powered analysis
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
-- → Der Stack ist ein LIFO-Speicherbereich für lokale Variablen
--   und Funktionsaufrufe mit schneller Allokation. Der Heap ist
--   ein dynamischer Speicherbereich für langlebige Objekte mit
--   manueller oder automatischer Speicherbereinigung.
```

---

## 19.4 Low-Level Funktionen

### ai_chat

```pipe
-- Signatur
ai_chat system_prompt user_prompt

-- Beschreibung
-- Direkter Chat mit einem System-Prompt und einem User-Prompt.
-- Gibt die rohe Antwort des Modells als String zurück.

-- Beispiel
antowrt: ai_chat
    "Du bist ein Bash-Experte. Antworte nur mit dem fertigen Befehl."
    "Wie liste ich alle Dateien größer als 100 MB rekursiv auf?"

print antwort
-- → find . -type f -size +100M -exec ls -lh {} \;
```

### ai_chat_json

```pipe
-- Signatur
ai_chat_json system_prompt user_prompt

-- Beschreibung
-- Wie ai_chat, aber die Antwort wird als JSON geparst.
-- Gibt eine Map oder List zurück, je nach JSON-Struktur.

-- Beispiel 1: Map
antwort: ai_chat_json
    "Du bist ein Datenanalyst. Antworte ausschließlich mit JSON."
    "Gib die Top-3-Programmiersprachen 2025 mit Anteil in Prozent."

print antwort.erste      -- → Python
print antwort.anteil_py  -- → 34

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
antworten: ai_parallel 3 "Übersetze ins Deutsche." [
    "Hello world",
    "Good morning",
    "How are you?"
]
```

### ai_rate_limit

```
ai_rate_limit calls_per_second
```

Begrenzt die Anzahl der API-Calls pro Sekunde (globaler Token-Bucket).

```pipe
ai_rate_limit 5       -- Max 5 Calls pro Sekunde

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
print (len vec)    -- 1536
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
print sim    -- z.B. 0.87 (sehr ähnlich)
```

### dot_product

```
dot_product vector_a vector_b
```

Skalarprodukt zweier Vektoren. **Kein API-Call nötig.**

```pipe
dp: dot_product [1.0, 2.0, 3.0] [2.0, 4.0, 6.0]
print dp    -- 28.0
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

## 19.8 Pipeline mit KI

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
-- → 6 (Wörter im übersetzten Text)

-- Klassifizierung im Datenfluss
dokumente: read_lines "emails.txt"
ergebnisse: map dokumente fn(doc)
    klass: classify doc "intern, extern, spam, support"
    { doc: doc, type: klass }
```

---

## 19.9 Fehlerbehandlung

KI-Operationen können fehlschlagen — etwa bei fehlendem API-Key,
Netzwerkproblemen oder Timeout. Mit `try`/`catch` lassen sich
diese Fehler abfangen:

```pipe
-- Fehlender API-Key
try
    print (summarize "Ein langer Text...")
catch e
    print "KI-Fehler: " ++ e.message
    print "Bitte API-Key setzen!"

-- Timeout abfangen
ai_timeout 5     -- kurzer Timeout
try
    print (generate "Schreibe einen 10-seitigen Aufsatz über Philosophie")
catch e
    if contains e.message "timeout"
        print "Anfrage hat zu lange gedauert"
    else
        throw e

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

## 19.10 Provider-Details

| Provider | API-Endpunkt | Standardmodell | Umgebungsvariable |
|----------|-------------|----------------|-------------------|
| OpenAI | `https://api.openai.com/v1/chat/completions` | `gpt-4o-mini` | `OPENAI_API_KEY` |
| Anthropic | `https://api.anthropic.com/v1/messages` | `claude-3-haiku-20240307` | `ANTHROPIC_API_KEY` |
| DeepSeek | `https://api.deepseek.com/v1/chat/completions` | `deepseek-chat` | `DEEPSEEK_API_KEY` |

Die OpenAI-kompatible API-Schnittstelle ermöglicht die Anbindung weiterer
kompatibler Dienste über die Umgebungsvariable `OPENAI_API_KEY` und
`OPENAI_API_BASE` für benutzerdefinierte Endpunkte.

---

## 19.11 Beispiel: Vollständiger Workflow

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
  Ihre E-Mail ist anna@techcorp.de und sie verdient 95.000 € pro Jahr."
schema: { name: "str", firma: "str", erfahrung_jahre: "num",
           sprachen: "list", email: "str", gehalt: "num" }
daten: extract lebenslauf schema
print "Name:      " ++ daten.name
print "Firma:     " ++ daten.firma
print "Erfahrung: " ++ (to_str daten.erfahrung_jahre) ++ " Jahre"
print "Sprachen:  " ++ (join daten.sprachen ", ")
print "E-Mail:    " ++ daten.email
print "Gehalt:    " ++ (to_str daten.gehalt) ++ " €"

print "\n=== 6. Frage beantworten ==="
frage: "Welche dieser Sprachen — " ++ (join daten.sprachen ", ") ++
       " — eignet sich am besten für Systemprogrammierung und warum?"
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
for s in empfehlungen.stärken
    print "  - " ++ s

print "Verbesserungspotenzial:"
for v in empfehlungen.verbesserung
    print "  - " ++ v

print "Nächster Schritt: " ++ empfehlungen.nächster_karriereschritt
print "Gehaltsprognose:  " ++ (to_str empfehlungen.gehaltsprognose) ++ " €"

print "\n=== 9. Fehlerbehandlung im Workflow ==="
try
    ai_provider "deepseek"
    print (ask "Was ist TDD?")
catch e
    print "DeepSeek nicht verfügbar: " ++ e.message
    print "Falle zurück auf OpenAI..."
    ai_provider "openai"
    print (ask "Was ist TDD?")

print "\n=== Workflow abgeschlossen ==="
```
