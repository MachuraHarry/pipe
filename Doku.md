# Pulse — Die umfassende Dokumentation

> Version 0.5.0 | 21 Beispiele | 60 Tests | 80+ Builtins

---

## Inhaltsverzeichnis

1. [Was ist Pulse?](#1-was-ist-pulse)
2. [Sprachreferenz](#2-sprachreferenz)
3. [Standardbibliothek](#3-standardbibliothek)
4. [Tooling & Ausführung](#4-tooling--ausführung)
5. [Architektur (Wie Pulse intern funktioniert)](#5-architektur)
6. [Was kann man mit Pulse bauen?](#6-anwendungsbereiche)
7. [Vergleich mit anderen Skriptsprachen](#7-vergleich-mit-anderen-skriptsprachen)
8. [Schnellreferenz](#8-schnellreferenz)

---

## 1. Was ist Pulse?

Pulse ist eine **einrückungsbasierte, pipeline-orientierte Skriptsprache**,
implementiert in Go und ausgeliefert als einzelne, statisch gelinkte Binary (~10 MB).

**Kernphilosophie:** Daten fließen sichtbar von oben nach unten durch
eine Kette von Transformationen — nicht versteckt in verschachtelten
Klammerausdrücken.

```pulse
-- Statt: print(addiere(verdopple(42), 10))
42
    > verdopple       -- verdopple(42) = 84
    > addiere 10      -- addiere(84, 10) = 94
    > print           -- Ausgabe: 94
```

Pulse kombiniert die **Lesbarkeit von Python** (Einrückung statt Klammern) mit der **Pipeline-Philosophie der Unix-Shell** und der **Portabilität einer Go-Binary** (10 MB, null Abhängigkeiten).
mit der **Pipeline-Philosophie der Unix-Shell** und der **Portabilität einer Go-Binary**
(kompakt, keine externen Abhängigkeiten).

---

## 2. Sprachreferenz

### 2.1 Lexikalische Grundlagen

**Kommentare** beginnen mit `--` und gelten bis zum Zeilenende:

```pulse
-- Das ist ein Kommentar
x: 42   -- Auch hinter Code möglich
```

**Einrückung** definiert Code-Blöcke. Pulse verwendet 4 Leerzeichen
(oder Tabs) wie Python:

```pulse
if x > 10
    print "x ist groß"       -- Eingerückt = gehört zum if
    x: 0                     -- Eingerückt = gehört zum if
print "Fertig"               -- Nicht eingerückt = nach dem if
```

**Mehrzeilige Strings** mit Backticks:

```pulse
text: `Zeile 1
Zeile 2
Zeile 3`
```

### 2.2 Datentypen

Pulse hat 7 eingebaute Datentypen:

| Typ | Literal | Beispiel |
|-----|---------|----------|
| `nil` | `nil` | Abwesenheit / kein Wert |
| `bool` | `true`, `false` | Wahrheitswerte |
| `num` | `42`, `3.14` | 64-Bit Integer oder Float |
| `str` | `"Hallo"` | Unicode-String |
| `list` | `[1, 2, 3]` | Dynamische, geordnete Liste |
| `map` | `{name: "Anna"}` | Assoziative Map (Schlüssel-Wert) |
| `fn` | `fn x: x * 2` | First-Class Function |

```pulse
nichts: nil
wahr:   true
zahl:   42
komma:  3.14
text:   "Hallo Welt"
liste:  [10, 20, 30]
map:    {name: "Pulse", version: 1}
```

### 2.3 Variablen & Zuweisung

Variablen werden mit `name: wert` definiert und neu zugewiesen:

```pulse
zaehler: 0
zaehler: zaehler + 1       -- Neuzuweisung: jetzt 1

-- Compound Assignment (seit v0.5):
x: 10
x += 5                      -- x = 15
x -= 3                      -- x = 12
x *= 2                      -- x = 24
x /= 4                      -- x = 6
x %= 4                      -- x = 2
```

**Gültigkeitsbereich:** Variablen sind im aktuellen Block sichtbar.
Funktionen sehen ihre eigenen Parameter und Zugriff auf den
umschließenden Scope (Closures).

### 2.4 Operatoren

#### Arithmetik

| Operator | Beschreibung |
|----------|-------------|
| `+` | Addition |
| `-` | Subtraktion |
| `*` | Multiplikation |
| `/` | Division (Integer, wenn beide Integer) |
| `%` | Modulo (Rest) |
| `**` | Potenz (`2 ** 10` → `1024`) |

#### Vergleich

| Operator | Beschreibung |
|----------|-------------|
| `==` | Gleich |
| `!=` | Ungleich |
| `<` | Kleiner |
| `>` | Größer |
| `<=` | Kleiner oder gleich |
| `>=` | Größer oder gleich |

#### Logik

| Operator | Beschreibung |
|----------|-------------|
| `!` | Logisches NICHT (`!true` → `false`) |
| `&&` | Logisches UND (Short-circuit) |
| `\|\|` | Logisches ODER (Short-circuit) |

#### Strings

```pulse
name: "Pulse"
print ("Hallo " ++ name)    -- "Hallo Pulse" (String-Verkettung)
```

#### Operator-Präzedenz (höchste zu niedrigste)

`. () []` Feldzugriff / Aufruf / Index →
`**` Potenz →
`* / %` Multiplikativ →
`+ -` Additiv →
`< <= > >=` Vergleich →
`== !=` Gleichheit →
`&&` Logisches UND →
`||` Logisches ODER →
`>` Pipeline →

### 2.5 Kontrollfluss

#### if / else if / else

```pulse
punkte: 85

if punkte >= 90
    print "Sehr gut"
else if punkte >= 75
    print "Gut"
else if punkte >= 50
    print "Bestanden"
else
    print "Nicht bestanden"
```

`if` ist ein **Ausdruck** — der letzte Wert jedes Zweigs wird zurückgegeben:

```pulse
status: if punkte >= 50
    "bestanden"
else
    "durchgefallen"
```

#### match (Pattern Matching)

```pulse
fn bewertung note
    match note
        | 1      -> "Sehr gut"
        | 2      -> "Gut"
        | 3      -> "Befriedigend"
        | _      -> "Ungültig"

print (bewertung 2)    -- "Gut"
```

- `|` leitet ein Muster ein
- `->` trennt Muster vom Ergebnis
- `_` ist der Wildcard („trifft immer zu")

```pulse
-- Fibonacci mit match
fn fib n
    match n
        | 0  -> 0
        | 1  -> 1
        | _  -> fib(n - 1) + fib(n - 2)
```

#### while

```pulse
i: 0
while i < 5
    print i
    i: i + 1
```

**break** und **continue** funktionieren innerhalb von while-Schleifen.

```pulse
while true
    x: x + 1
    if x > 10
        break
    if x % 2 == 0
        continue      -- überspringt gerade Zahlen
    print x           -- gibt nur ungerade aus
```

#### for-in

```pulse
for farbe in ["rot", "grün", "blau"]
    print farbe

-- Mit range():
for n in (range 1 6)
    print n            -- 1, 2, 3, 4, 5
```

`range(n)` → `[0, 1, ..., n-1]`
`range(a, b)` → `[a, a+1, ..., b-1]`
`range(a, b, step)` → `[a, a+step, ..., <b]`

#### return

```pulse
fn betrag x
    if x < 0
        return (-x)
    x

print (betrag (-5))    -- 5
print (betrag 5)       -- 5
```

### 2.6 Funktionen

#### Definition

```pulse
fn verdopple x
    x * 2

fn addiere a b
    a + b
```

- `fn` leitet die Definition ein
- Parameter durch Leerzeichen getrennt (keine Kommas, keine Klammern)
- Der letzte Ausdruck im Körper ist der **Rückgabewert**
- Kein `return` nötig (aber optional verfügbar)

#### Aufruf

Funktionsaufrufe verwenden Leerzeichen statt Kommas:

```pulse
print (verdopple 21)       -- 42
print (addiere 3 4)        -- 7
```

Bei **einem Argument** können die Klammern entfallen:

```pulse
print "Hallo"              -- äquivalent zu print("Hallo")
```

**Wichtig:** Rechenausdrücke als Argumente brauchen Klammern:

```pulse
print (1 + 2)              -- Richtig: print(3)
-- print 1 + 2             -- Falsch: (print 1) + 2
```

#### Anonyme Funktionen

```pulse
verdreifacher: fn x
    x * 3

print (verdreifacher 7)    -- 21
```

#### Closures

Funktionen merken sich den Scope, in dem sie definiert wurden:

```pulse
fn make_adder n
    fn adder x
        x + n

add5: make_adder 5
print (add5 10)            -- 15
```

### 2.7 Pipeline

Die Pipeline ist das zentrale Sprach-Feature von Pulse.

> **Wichtig:** `>` wird im Ausdruckskontext als **Vergleichsoperator** behandelt
> (`a > b` = „a größer als b"). Nur in der **vertikalen Form mit Einrückung**
> fungiert `>` als Pipeline-Operator. Siehe Abschnitt unten.

#### Vertikale Pipeline (empfohlen)

```pulse
42
    > verdopple       -- verdopple(42)
    > addiere 10      -- addiere(verdopple(42), 10)
    > print           -- print(addiere(verdopple(42), 10))
```

Jede eingerückte `>`-Zeile nimmt das Ergebnis der vorherigen Zeile
und übergibt es als **erstes Argument** an die angegebene Funktion.

#### Pipeline mit Zusatzargumenten

```pulse
100
    > addiere 50      -- addiere(100, 50) = 150
    > print           -- 150
```

### 2.8 Datenstrukturen

#### Listen

```pulse
zahlen: [10, 20, 30, 40, 50]

-- Zugriff per Index
print (zahlen[0])           -- 10
print (zahlen[2])           -- 30

-- Slicing (Teilliste)
print (zahlen[0..3])        -- [10, 20, 30]
print (zahlen[2..4])        -- [30, 40]

-- Operationen
print (len zahlen)          -- 5
push zahlen 60              -- Element anhängen
print (pop zahlen)          -- Letztes entfernen (60)
print (at zahlen 1)         -- 20 (Element an Position)
print (sort [3, 1, 2])      -- [1, 2, 3]

-- Höhere Funktionen (Tree-Walker)
fn ist_gerade x: x % 2 == 0
print (filter [1,2,3,4] ist_gerade)  -- [2, 4]
print (map [1,2,3] verdopple)        -- [2, 4, 6]
```

#### Maps

```pulse
person: {name: "Anna", alter: 28, stadt: "Berlin"}

-- Zugriff
print (get person "name")       -- "Anna"
print (get person "alter")      -- 28

-- Ändern
set person "alter" 29

-- Schlüssel & Werte
print (keys person)             -- ["name", "alter", "stadt"]
print (values person)           -- ["Anna", 29, "Berlin"]
```

### 2.9 Fehlerbehandlung

```pulse
try
    ergebnis: 10 / 0            -- gefährlicher Code
catch fehler
    print "Fehler aufgetreten:"
    print fehler                -- "Division durch Null"

print "Programm läuft weiter!"  -- Wird ausgeführt!
```

**Stack-Traces** werden automatisch erzeugt:

```
ERROR: Division durch Null
  in fn(berechne)
  in fn(hauptprogramm)
```

Seit v0.5 kann `return` für vorzeitiges Verlassen genutzt werden:

```pulse
fn teile a b
    if b == 0
        return nil
    a / b
```

### 2.10 Module & Importe

```pulse
-- lib.pulse:
fn quadrat x
    x * x

-- main.pulse:
import "lib.pulse"
print (quadrat 7)            -- 49
```

Importe werden gecached — mehrfaches Importieren derselben Datei
parst sie nur einmal.

### 2.11 CLI-Argumente

```bash
./bin/pulse script.pulse arg1 arg2 arg3
```

```pulse
print args                  -- ["arg1", "arg2", "arg3"]
print (at args 0)           -- "arg1"
print (len args)            -- 3
```

---

## 3. Standardbibliothek

Pulse hat über 80 eingebaute Funktionen — keine externen Abhängigkeiten.

### I/O & System

| Funktion | Beschreibung |
|----------|-------------|
| `print x...` | Werte ausgeben (mit Leerzeichen getrennt, Newline) |
| `input prompt?` | Zeile von stdin lesen |
| `exec cmd` | Shell-Befehl ausführen → Map{output, error, status} |
| `env name` | Umgebungsvariable lesen |
| `sleep ms` | Millisekunden warten |

### Dateisystem

| Funktion | Beschreibung |
|----------|-------------|
| `read_file p` | Datei als String lesen |
| `write_file p t` | Text in Datei schreiben |
| `append_file p t` | Text anhängen |
| `read_lines p` | Datei als Liste von Zeilen |
| `file_exists p` | Prüft ob Datei existiert |
| `file_delete p` | Datei löschen |
| `file_move a b` | Umbenennen / Verschieben |
| `file_copy a b` | Kopieren |
| `file_size p` | Dateigröße in Bytes |
| `file_type p` | `"file"` oder `"dir"` |
| `list_dir p?` | Verzeichnisinhalt auflisten |
| `make_dir p` | Verzeichnis anlegen |
| `remove_dir p` | Verzeichnis löschen (rekursiv) |

### Pfad-Operationen

| Funktion | Beschreibung |
|----------|-------------|
| `path_join a b...` | Pfade zusammensetzen |
| `path_base p` | Dateiname (`/a/b.txt` → `"b.txt"`) |
| `path_dir p` | Verzeichnis (`/a/b.txt` → `"/a"`) |
| `path_ext p` | Endung (`/a/b.txt` → `".txt"`) |

### Strings

| Funktion | Beschreibung |
|----------|-------------|
| `upper s` | Großbuchstaben |
| `lower s` | Kleinbuchstaben |
| `trim s` | Leerzeichen an den Enden entfernen |
| `split s t` | An Trennzeichen teilen → Liste |
| `join list t` | Liste mit Trennzeichen verbinden |
| `contains s sub` | Enthält s den Teilstring sub? |

### Listen

| Funktion | Beschreibung |
|----------|-------------|
| `len list` | Anzahl Elemente |
| `push list x...` | Element(e) anhängen |
| `pop list` | Letztes Element entfernen |
| `at list i` | Element an Position i |
| `sort list` | Sortieren |
| `range n` / `range a b` / `range a b s` | Zahlenbereich |
| `map list fn` | Transformieren (Tree-Walker) |
| `filter list fn` | Filtern (Tree-Walker) |
| `reduce list fn init` | Falten (Tree-Walker) |

### Maps

| Funktion | Beschreibung |
|----------|-------------|
| `get map key` | Wert abrufen (nil wenn nicht vorhanden) |
| `set map key val` | Wert setzen |
| `keys map` | Alle Schlüssel als Liste |
| `values map` | Alle Werte als Liste |

### Mathematik

| Funktion | Beschreibung |
|----------|-------------|
| `abs n` | Absolutwert |
| `min a b...` | Minimum |
| `max a b...` | Maximum |
| `pow b e` | Potenz |
| `sqrt n` | Quadratwurzel |
| `round n` | Runden zur nächsten Ganzzahl |

### Netzwerk & HTTP

| Funktion | Beschreibung |
|----------|-------------|
| `http_get url` | HTTP GET → Map{status, body} |
| `http_post url body` | HTTP POST → Map{status, body} |
| `http_get_json url` | HTTP GET + automatisches JSON-Parsing |
| `parse_json str` | JSON-String → Map/List |
| `to_json val` | Map/List → JSON-String |
| `tcp_listen host port` | TCP-Server starten |
| `tcp_connect host port` | TCP-Client verbinden |
| `tcp_accept listener` | Verbindung annehmen |
| `tcp_read conn` | Von TCP-Verbindung lesen |
| `tcp_write conn data` | Auf TCP-Verbindung schreiben |
| `tcp_close conn` | TCP-Verbindung schließen |

### Regex, Datum, Zufall, Encoding

| Funktion | Beschreibung |
|----------|-------------|
| `regex_match pattern text` | Prüft ob Muster passt |
| `regex_replace pattern repl text` | Ersetzt Vorkommen |
| `now dummy` | Unix-Timestamp (Sekunden seit 1970) |
| `format_time ts layout` | Zeitstempel formatieren (Go-Layout) |
| `random dummy` | Zufallszahl 0..1 |
| `random_range min max` | Zufällige Ganzzahl |
| `base64_encode s` | Base64-kodieren |
| `base64_decode s` | Base64-dekodieren |

### Typ-Prüfung & Konvertierung

| Funktion | Beschreibung |
|----------|-------------|
| `type_of x` | Typ als String (`"INTEGER"`, `"STRING"`, ...) |
| `is_num x` | Ist eine Zahl? |
| `is_str x` | Ist ein String? |
| `is_list x` | Ist eine Liste? |
| `is_map x` | Ist eine Map? |
| `is_nil x` | Ist nil? |
| `to_str x` | In String umwandeln |
| `to_num x` | In Zahl umwandeln |

---

## 4. Tooling & Ausführung

### Installation & Start

```bash
git clone <pulse-repo>
cd pulse
make build
./bin/pulse examples/hello.pulse
```

### Ausführungsmodi

| Modus | Befehl | Beschreibung |
|-------|--------|-------------|
| **Tree-Walker** | `./bin/pulse datei.pulse` | Alle Features, langsamer |
| **Bytecode-VM** | `./bin/pulse -vm datei.pulse` | ~7× schneller, while/for-in/return |
| **VM (quiet)** | `./bin/pulse -vm -q datei.pulse` | Ohne Bytecode-Ausgabe |
| **AST** | `./bin/pulse -ast datei.pulse` | AST anzeigen (Entwickler) |
| **REPL** | `./bin/pulse` | Interaktiver Modus |
| **Hilfe** | `./bin/pulse -h` | Alle Optionen |

### REPL

```
$ ./bin/pulse
>>> 1 + 2
3
>>> fn verdopple x
...     x * 2
...
>>> print (verdopple 21)
42
>>> :vm           -- VM-Modus umschalten
>>> :quit         -- Beenden
```

**REPL-Befehle:**
- `:quit` / `:q` → Beenden
- `:help` / `:h` → Hilfe
- `:clear` / `:c` → Eingabe zurücksetzen
- `:vm` → VM / Tree-Walker umschalten
- Leerzeile → Mehrzeiligen Block abschließen

---

## 5. Architektur

### 5.1 Überblick

Pulse besteht aus fünf Hauptkomponenten:

```
Quelltext (.pulse)
    │
    ▼
┌─────────────┐
│   Lexer     │  → Token-Stream (INDENT/DEDENT)
└─────────────┘
    │
    ▼
┌─────────────┐
│   Parser    │  → AST (rekursiver Abstieg + Pratt)
└─────────────┘
    │
    ├──→ Tree-Walker (eval)    → Direkte Ausführung
    │
    └──→ Compiler + VM (vm)    → Bytecode → Stack-Maschine
```

### 5.2 Lexer (`pkg/lexer/`)

- ~430 Zeilen Go
- Tokenisiert Quelltext in ~50 Token-Typen
- **INDENT/DEDENT**-Tracking wie Python (Stack-basiert)
- Erkennt Einrückungsänderungen und emittet INDENT/DEDENT-Tokens
- Überspringt Leerzeilen und Kommentare
- 15 Tests

### 5.3 Parser (`pkg/parser/`)

- ~860 Zeilen Go
- **Rekursiver Abstieg** für Statements
- **Pratt-Parsing** für Ausdrücke (Operator-Präzedenz)
- Parse-Funktionen pro AST-Knoten
- Block-Strukturen über INDENT/DEDENT
- 20 Tests

### 5.4 AST (`pkg/ast/`)

- ~310 Zeilen Go
- 25+ Knotentypen (Program, ExpressionStatement, FnStatement,
  IfExpression, MatchExpression, WhileExpression, PipelineExpression, ...)
- Alle Knoten implementieren `Node`-Interface

### 5.5 Tree-Walker (`pkg/eval/`)

- ~750 Zeilen Go
- Rekursive AST-Evaluation
- Environment-basiertes Scoping
- Call-Stack für Stack-Traces
- BreakValue/ContinueValue/ReturnValue für Kontrollfluss

### 5.6 Bytecode-Compiler + VM (`pkg/compiler/`, `pkg/vm/`)

- **33 Opcodes** (OpConstant, OpAdd, OpCall, OpClosure, OpJump, ...)
- Symboltabelle mit Global/Local/Builtin-Scopes
- Stack-VM mit Operanden-Stack und Call-Frames
- ~7× schneller als Tree-Walker
- 25 Tests

### 5.7 Laufzeit-Typen (`pkg/object/`)

- ~1600 Zeilen Go (inkl. aller Builtins)
- 8 Objekt-Typen: Integer, Float, String, Boolean, Nil, Function,
  CompiledFunction, List, Map, Error
- Environment mit Scope-Chain
- Alle 80+ Builtins (Reine Go-Funktionen)

### 5.8 CLI & REPL (`cmd/pulse/`)

- ~420 Zeilen Go
- Flag-Parsing (`-vm`, `-q`, `-ast`, `-h`)
- REPL mit Multi-Line-Support
- AST-Printer für Debugging

### 5.9 Technische Daten

| Metrik | Wert |
|--------|------|
| **Go-Zeilen** | ~6.500 |
| **Go-Packages** | 8 |
| **Tests** | 60 |
| **Beispiel-Programme** | 21 |
| **Builtins** | 80+ |
| **Opcodes** | 33 |
| **AST-Knoten** | 25+ |
| **Binary-Größe** | ~10 MB |
| **Externe Abhängigkeiten** | 0 |

---

## 6. Anwendungsbereiche

### CLI-Tools & Automation

```pulse
-- Build-Skript
print (exec "go build ./...")
print (exec "go test ./...")

-- Datei-Backup mit Zeitstempel
ts: now 0
ziel: path_join "/backups" ("backup_" ++ (to_str ts))
make_dir ziel
for f in (list_dir ".")
    if contains f ".pulse"
        file_copy f (path_join ziel f)
```

### API-Clients & Web-Scraping

```pulse
-- GitHub-API
user: http_get_json "https://api.github.com/users/torvalds"
print (get user "public_repos")    -- 12

-- Web-Scraper
html: get (http_get "https://example.com") "body"
woerter: len (split (regex_replace "<[^>]+>" " " html) " ")
print ("Wörter: " ++ (to_str woerter))
```

### Daten-Pipelines & Textverarbeitung

```pulse
read_file "log.txt"
    > split "\n"
    > filter (fn l: contains l "ERROR")
    > sort
    > print
```

### TCP-Dienste

```pulse
-- Echo-Server
ln: tcp_listen "0.0.0.0" 9999
conn: tcp_accept ln
msg: tcp_read conn
tcp_write conn ("ECHO: " ++ msg)
```

### Konfiguration & System-Management

```pulse
host: env "HOST"
port: to_num (env "PORT")
print (exec ("curl -s " ++ host ++ ":" ++ (to_str port) ++ "/health"))
```

---

## 7. Vergleich mit anderen Skriptsprachen

### 7.1 Pulse vs Python

| Merkmal | Pulse | Python |
|---------|-------|--------|
| **Pipeline** | `42 > fn > print` | `42 \| fn \| print` (geht nicht nativ) |
| **Funktionsaufruf** | `print "Hallo"` (space-based) | `print("Hallo")` (Klammern) |
| **Rückgabewert** | Letzter Ausdruck | `return` erforderlich |
| **Pattern Matching** | `match` (first-class) | `match` (ab Python 3.10) |
| **Einrückung** | 4 Leerzeichen | 4 Leerzeichen |
| **OOP** | ❌ | ✅ (Klassen, Vererbung) |
| **Builtins ohne Imports** | 80+ | ~70 (plus 100+ Module) |
| **Statische Binary** | ✅ (10 MB) | ❌ (braucht Interpreter) |
| **Ökosystem** | Eigenbau | Riesig (pip, 400k+ Packages) |

**Wann Pulse statt Python?** Wenn du eine einzelne Binary ausliefern willst,
keine Abhängigkeiten brauchst und Pipeline-artige Datenverarbeitung
im Vordergrund steht.

### 7.2 Pulse vs Lua

| Merkmal | Pulse | Lua |
|---------|-------|-----|
| **Portabilität** | ✅ (10 MB Binary) | ✅ (300 KB Binary) |
| **Pipeline** | ✅ First-Class | ❌ |
| **HTTP eingebaut** | ✅ | ❌ (externes luasocket) |
| **JSON eingebaut** | ✅ | ❌ (extern) |
| **Regex eingebaut** | ✅ | ❌ (extern) |
| **Einrückung** | ✅ | ❌ (`end`-Keywords) |
| **Tabellen/Arrays** | Listen + Maps getrennt | Alles ist eine Table |
| **Metatabellen** | ❌ | ✅ (mächtig) |
| **Coroutinen** | ❌ | ✅ |
| **Größe** | 10 MB | ~300 KB |
| **Performance** | ~7× langsamer (VM) | Schnell (LuaJIT: extrem) |

**Wann Pulse statt Lua?** Wenn du eine modernere Syntax mit mehr
Builtins willst und keine C-Integration brauchst. Lua ist besser
für Embedded-Systeme und wenn es auf jedes Kilobyte ankommt.

### 7.3 Pulse vs JavaScript/Node.js

| Merkmal | Pulse | Node.js |
|---------|-------|---------|
| **Pipeline** | ✅ Sprach-Feature | ❌ (nur Array-Methoden) |
| **Async/Await** | ❌ | ✅ |
| **NPM-Ökosystem** | ❌ | ✅ (Millionen Packages) |
| **Binary-Größe** | 10 MB | ~80 MB (Runtime) |
| **Pattern Matching** | ✅ | ❌ |
| **Einrückung** | ✅ | ❌ (`{}`) |
| **Typisierung** | Dynamisch | Dynamisch |

**Wann Pulse statt Node?** Für kleine bis mittlere Tools, wo NPM-Overhead
überdimensioniert wäre. Node ist besser für Web-Server mit Async-I/O.

### 7.4 Pulse vs Bash

| Merkmal | Pulse | Bash |
|---------|-------|------|
| **Datenstrukturen** | ✅ (Listen, Maps) | ❌ (nur Strings) |
| **Funktionen** | ✅ (First-Class) | 🟡 (eingeschränkt) |
| **JSON** | ✅ | ❌ (braucht `jq`) |
| **HTTP** | ✅ | ❌ (braucht `curl`) |
| **Fehlerbehandlung** | ✅ (try/catch) | 🟡 (`set -e`, `trap`) |
| **Pipeline** | ✅ (strukturiert) | ✅ (nur Text-Streams) |
| **Regex** | ✅ | 🟡 (nur `grep`/`sed`) |

**Wann Pulse statt Bash?** Sobald die Logik komplexer wird als
„ein paar Befehle aneinanderreihen". Pulse gibt dir echte
Datenstrukturen und Fehlerbehandlung.

### 7.5 Zusammenfassung

| Qualität | Score | Notiz |
|----------|-------|-------|
| **Einfachheit** | ⭐⭐⭐⭐⭐ | Grammatik passt auf eine Seite |
| **Ausdruckskraft** | ⭐⭐⭐⭐ | Pipeline + Match + Closures |
| **Builtins** | ⭐⭐⭐⭐⭐ | 80+ ohne Imports |
| **Performance** | ⭐⭐⭐ | ~7× Lua (ausreichend für Skripte) |
| **Portabilität** | ⭐⭐⭐⭐⭐ | Eine Binary, alle Plattformen |
| **Ökosystem** | ⭐⭐ | Kein Package-Manager |
| **Tooling** | ⭐⭐⭐ | REPL, AST-Printer, VM-Flag |

---

## 8. Schnellreferenz

### Syntax-Cheatsheet

```pulse
-- Kommentar
x: 42                         -- Variable
x += 1                        -- Compound Assignment
fn name a b: ...              -- Funktion
if bed: ... else: ...         -- Bedingung
match x | 0 -> ... | _ ->     -- Pattern Matching
while bed: ...                -- Schleife
for x in liste: ...           -- for-in
try: ... catch e: ...         -- Fehler abfangen
return wert                   -- Vorzeitiges Verlassen
import "datei.pulse"          -- Modul laden

wert
    > funktion                 -- Vertikale Pipeline
    > ausgabe

list[0..3]                    -- Slicing
{key: wert}                   -- Map
[1, 2, 3]                     -- Liste
`mehrzeiliger text`            -- Backtick-String
```

### Operatoren-Cheatsheet

```
+ - * / % **     Arithmetik + Potenz
+= -= *= /= %=   Compound
== != < > <= >=  Vergleiche
! && ||          Logik (nicht, und, oder)
++               String-Verkettung
>                Pipeline (vertikal)
..               Bereich (Slicing)
```

### CLI-Cheatsheet

```bash
pulse datei.pulse              # Tree-Walker
pulse -vm datei.pulse          # Bytecode-VM
pulse -vm -q datei.pulse       # VM ohne Bytecode-Ausgabe
pulse -ast datei.pulse         # AST anzeigen
pulse                          # REPL starten
pulse datei.pulse arg1 arg2    # Mit CLI-Argumenten
```

### Datei-Endung

Pulse-Dateien: **`.pulse`**

### Projektstruktur (Empfehlung)

```
mein_projekt/
├── main.pulse
├── lib/
│   ├── helpers.pulse
│   └── config.pulse
└── README.md
```

---

*Pulse — Daten fließen lassen, nicht Klammern zählen.*
