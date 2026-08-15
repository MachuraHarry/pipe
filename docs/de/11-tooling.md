# 11. Tooling

## 11.1 CLI-Flags

Die Pipe-Binary (`bin/pipe`) akzeptiert folgende Flags:

```bash
./bin/pipe [flags] [datei.pipe] [args...]
```

| Flag | Beschreibung |
|------|-------------|
| `-h`, `--help` | Hilfe anzeigen |
| `-vm` | Bytecode-VM statt Tree-Walker verwenden |
| `-q` | VM-Modus: Bytecode-Ausgabe unterdrücken (quiet) |
| `-ast` | Nur AST ausgeben, nicht ausführen |
| `-fmt` | Datei formatieren (Einrückung, Whitespace) |
| `-test` | Alle `*_test.pipe` / `*.test.pipe` Dateien im aktuellen Verzeichnis ausführen |
| `-bench` | Benchmarks: Tree-Walker vs VM vergleichen |
| `-build` | Standalone Binary aus .pipe-Datei erzeugen |
| `-search` | Modul-Registry durchsuchen |
| `-get` | Modul aus Registry herunterladen |
| `-init` | Neues Modul-Gerüst erstellen |
| `-validate` | pipe.json eines Moduls prüfen |
| `-install` | Abhängigkeiten aus pipe.json installieren |
| `-gen-registry` | registry.json aus pipe.json-Dateien generieren |

### Modul-Verwaltung

Pipe hat ein eingebautes Modul-System. Module werden über das [pipe-modules](https://github.com/MachuraHarry/pipe-modules) Repository geteilt.

```bash
# Module suchen
pipe -search
pipe -search log

# Modul installieren (einmalig)
pipe -get jpipe
pipe -get jpipe@1.0.0

# Neues Modul erstellen
pipe -init mein-modul

# pipe.json validieren
pipe -validate mein-modul

# Abhängigkeiten installieren (rekursiv)
pipe -install

# Registry aus pipe.json generieren
pipe -gen-registry .
```

**`pipe -install`** liest `dependencies` aus `pipe.json`, löst alle Abhängigkeiten rekursiv auf und schreibt eine `pipe.lock`-Datei mit gepinnten Versionen und SHA-256-Prüfsummen.

**`pipe -init`** erstellt:
```
mein-modul/
├── pipe.json       ← Manifest
├── module.pipe     ← Quellcode
└── README.md       ← Dokumentation
```

### Verwendung

```bash
# Tree-Walker (Standard)
./bin/pipe datei.pipe

# Bytecode-VM (schneller)
./bin/pipe -vm datei.pipe

# VM ohne Bytecode-Ausgabe
./bin/pipe -vm -q datei.pipe

# AST anzeigen (Debugging)
./bin/pipe -ast datei.pipe

# Datei formatieren
./bin/pipe -fmt datei.pipe

# Tests ausführen
./bin/pipe -test

# Benchmarks
./bin/pipe -bench

# Standalone Binary bauen
./bin/pipe -build mein_skript.pipe -o mein_programm

# Mit CLI-Argumenten
./bin/pipe datei.pipe arg1 arg2 arg3
```

## 11.2 REPL (Interaktiver Modus)

Starte `./bin/pipe` ohne Dateinamen:

```
$ ./bin/pipe
Pipe v0.9.4.0 — REPL
>>> 1 + 2
3
>>> fn verdopple x
...     x * 2
...
>>> print (verdopple 21)
42
>>> :quit
```

### REPL-Befehle

| Befehl | Kurzform | Beschreibung |
|--------|----------|-------------|
| `:quit` | `:q` | REPL beenden |
| `:help` | `:h` | Hilfe anzeigen |
| `:clear` | `:c` | Aktuelle Eingabe zurücksetzen |
| `:vm` | — | VM-Modus umschalten (Tree-Walker ↔ VM) |
| `:history` | — | Letzte 100 Befehle anzeigen |
| `:!N` | — | Befehl N aus der History wiederholen (z.B. `:!3`) |
| `Strg+D` | — | Beenden (EOF) |
| Leerzeile | — | Mehrzeiligen Block abschließen und ausführen |

### Mehrzeiliger Modus

Nach Eingabe folgender Keywords erscheint `...` statt `>>>`:

`fn`, `if`, `match`, `while`, `for`, `try`, `defer`, `export`, `enum`

Eine **Leerzeile** schließt den Block ab und führt ihn aus:

```
>>> fn addiere a b
...     a + b
...
>>> print (addiere 3 4)
7
```

### VM-Modus in der REPL

`:vm` schaltet zwischen Tree-Walker und VM um:

```
>>> 1 + 2
3
>>> :vm
  VM-Modus: ein
>>> 1 + 2
3
```

## 11.3 Formatter (`-fmt`)

Der Formatter normalisiert Whitespace und Einrückung:

```bash
./bin/pipe -fmt datei.pipe
```

**Was der Formatter macht:**
- Einrückung auf 4-Leerzeichen-Blöcke normalisieren
- Leerzeichen nach `:` in Variablen-Definitionen
- Leerzeilen zwischen Funktions-/Enum-Definitionen
- Inline-Spacing für Operatoren
- Abschließende Newline garantieren

**Formatierungs-Beispiele:**

```pipe
-- Vorher (unformatiert):
fn add a b
    a+b
x:42

-- Nachher (formatiert):
fn add a b
    a + b

x: 42
```

Der Formatter parst die Datei und schreibt die AST-Struktur formatiert zurück.
Falls der Parse fehlschlägt, wird eine reine Whitespace-Normalisierung durchgeführt.

## 11.4 Test-Runner (`-test`)

Pipe hat einen eingebauten Test-Runner:

```bash
./bin/pipe -test
```

Der Test-Runner:
1. Findet alle `*_test.pipe` und `*.test.pipe` Dateien im aktuellen Verzeichnis
2. Parst jede Datei
3. Führt sie aus
4. Zeigt PASS/FAIL mit Zählung

### Eingebaute Assertions

| Funktion | Beschreibung |
|----------|-------------|
| `assert(cond)` | Schlägt fehl, wenn der Wert nicht truthy ist |
| `assert_eq(a, b)` | Schlägt fehl, wenn die Werte nicht gleich sind |
| `assert_not_eq(a, b)` | Schlägt fehl, wenn die Werte gleich sind |
| `assert_lt(a, b)` | Schlägt fehl, wenn nicht `a < b` |
| `assert_gt(a, b)` | Schlägt fehl, wenn nicht `a > b` |
| `assert_error(fn)` | Schlägt fehl, wenn `fn()` keinen Fehler wirft |

```pipe
test "addition"
    assert (1 + 2) == 3
    assert_eq (2 + 2) 4

test "multiplikation"
    assert_eq (2 * 3) 6

test "vergleich"
    assert_lt 3 5
    assert_gt 10 3
```

## 11.5 Benchmark (`-bench`)

```bash
./bin/pipe -bench
```

Führt drei vordefinierte Benchmarks aus und vergleicht Tree-Walker vs VM:

1. **Fibonacci(20)** — Rekursive Berechnung (5 Iterationen)
2. **FizzBuzz 1-100** — FizzBuzz über 100 Zahlen (5 Iterationen)
3. **List Sum 10000** — Summe einer Liste mit 10.000 Elementen (5 Iterationen)

Ausgabe zeigt Laufzeiten und Speedup-Faktor der VM.

## 11.6 AST-Ausgabe (`-ast`)

Zeigt den AST (Abstract Syntax Tree) einer Datei als formatierten Baum:

```bash
./bin/pipe -ast datei.pipe
```

Nützlich für:
- Debugging des Parsers
- Verstehen der Sprachstruktur
- Entwicklung von Tools

## 11.7 Self-Extracting Binary (`-build`)

```bash
./bin/pipe -build mein_skript.pipe
./mein_skript                  -- Direkt ausführbar!
```

`-build` erzeugt eine **eigenständige Binary**:
1. Kopiert die `pipe`-Binary
2. Hängt `PIPEBUILD`-Marker + Quellcode an
3. Setzt Ausführungsrechte

Beim Start erkennt die Binary den eingebetteten Code und führt ihn aus —
**kein separater Interpreter nötig**.

```bash
./bin/pipe -build server.pipe -o mein-server
./mein-server 8080            -- Führt server.pipe aus
```

**Argumente an die Binary übergeben:**

Argumente auf der Kommandozeile sind im Skript über das `args`-Builtin
verfügbar – genau wie bei `pipe skript.pipe ...`:

```bash
./bin/pipe -build begruessung.pipe
./begruessung Alice Bob
```

```pipe
-- begruessung.pipe
for name in args
    print "Hallo, " ++ name
-- ./begruessung Alice Bob
-- -> Hallo, Alice
-- -> Hallo, Bob
```

**Dateien einbetten (`--embed-file`):**

Mit `--embed-file` lassen sich zusätzliche Dateien (Daten, Configs, Prompts, …)
in die Binary einbetten. Zur Laufzeit entpackt die Binary sie in ein frisches
temporäres Verzeichnis und wechselt ihr Arbeitsverzeichnis dorthin — das Skript
kann sie also einfach per Namen mit `read_file` lesen.

```bash
# Skript mit seinen Daten bündeln
./bin/pipe -build agent.pipe -o agent --embed-file prompts.txt --embed-file config.json
```

```pipe
-- agent.pipe — liest die eingebetteten Dateien per Namen
system_prompt: read_file "prompts.txt"
einstellungen: parse_json (read_file "config.json")

print system_prompt
print einstellungen.model
```

```bash
# Das selbstständige Artefakt von überall ausführen
cp agent /opt/tools/
/opt/tools/agent
```

**So funktioniert es:**

1. Die Binary ist der native Pipe-Interpreter, gefolgt von der `PIPEBUILD`-Quellsektion.
2. Eine optionale `PIPEFILES`-Sektion speichert jede Datei als `Name + Größe + Bytes`.
3. Beim Start entpackt die Binary die Dateien in ein frisches `pipe_embedded_*`-Verzeichnis
   unter dem System-Temp-Verzeichnis und wechselt das Arbeitsverzeichnis dorthin.

**Binary-Struktur:**

```
┌─────────────────────────┐
│  Pipe-Interpreter (ELF) │  <- Natives ausführbares Programm
├─────────────────────────┤
│  PIPEBUILD\n            │  <- Magic-Marker
├─────────────────────────┤
│  Quellcode              │  <- Eingebetteter Pipe-Quellcode
├─────────────────────────┤
│  PIPEFILES\n            │  <- Optionaler Marker (nur mit --embed-file)
├─────────────────────────┤
│  Name + Größe + Bytes   │  <- Eingebettete Datendateien
└─────────────────────────┘
```

## 11.8 Bytecode-Cache (`.pipec`)

Im VM-Modus erzeugt Pipe automatisch einen **Bytecode-Cache**:

```
datei.pipe   →  datei.pipec  (Bytecode-Cache)
```

Der Cache enthält den kompilierten Bytecode und einen **SHA-256-Hash**
des Quelltextes. Bei Änderungen wird der Cache automatisch invalidiert.

**Cache-Format (binär):**
```
Magic:  "PIPEBC" (6 bytes)
Version: 1 byte
Source-Hash: 16 bytes (SHA-256)
Constants-Section
Instructions-Section
```

## 11.9 Alle Beispiele ausführen

```bash
for f in examples/*.pipe; do echo "=== $f ===" && ./bin/pipe "$f"; done
```

## 11.10 Tests ausführen (Go-Tests)

```bash
make test
# oder:
go test ./pkg/...
```

Führt alle 220+ Unit-Tests aus (Lexer, Parser, AST, Eval, Compiler, VM, Cache, Formatter, Object).
