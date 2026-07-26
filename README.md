# Pulse — Eine minimalistische, pipeline-basierte Skriptsprache

Pulse ist eine einrückungsbasierte Skriptsprache mit Pipeline-Syntax.  
Die gesamte Grammatik passt auf eine Seite — inspiriert von Lua's Minimalismus.

## Schnellstart

```bash
# Bauen
make build

# REPL starten
make repl

# Datei ausführen (Tree-Walker)
./bin/pulse beispiele/hello.pulse

# Datei ausführen (VM, schnell)
./bin/pulse -vm beispiele/fib.pulse

# VM ohne Bytecode-Ausgabe
./bin/pulse -vm -q beispiele/pipeline.pulse

# AST ausgeben
./bin/pulse -ast beispiele/fib.pulse

# Alle Beispiele ausführen
for f in examples/*.pulse; do echo "=== $f ===" && ./bin/pulse "$f"; done
```

## Sprachüberblick

### Kommentare

```
-- Einzeilige Kommentare mit doppeltem Bindestrich
```

### Variablen

```
name: "Welt"
x: 42
aktiv: true
```

### Funktionen

```
fn greet name
    "Hallo " ++ name

fn add a b
    a + b

fn fib n
    match n
        | 0  -> 0
        | 1  -> 1
        | _  -> fib(n - 1) + fib(n - 2)
```

Der letzte Ausdruck im Funktionskörper ist der Rückgabewert. Kein `return`-Keyword.

### Pipeline

```
-- Horizontal
fib 10 > print

-- Vertikal
42
    > double
    > add 10
    > print
```

Bedeutet: `print(add(double(42), 10))`. Leserichtung = Datenfluss.

### Bedingungen

```
if x > 10
    "groß"
else if x > 5
    "mittel"
else
    "klein"
```

### Pattern Matching

```
match wert
    | 0      -> "null"
    | 1..5   -> "klein"
    | _      -> "sonst"
```

### Funktionsaufrufe

```
print "Hallo"           -- space between function and argument
print (1 + 2)           -- parens for expressions
fib (n - 1)             -- parens for complex args
```

## Datentypen

| Typ | Literal | Beispiel |
|-----|---------|----------|
| `nil` | `nil` | Abwesenheit |
| `bool` | `true`, `false` | Wahrheitswerte |
| `num` | `42`, `3.14` | 64-Bit Gleitkomma |
| `str` | `"Hallo"` | Unicode-Strings |
| `list` | `[1, 2, 3]` | Dynamische Liste |
| `map` | `{a: 1, b: 2}` | Assoziative Map |
| `fn` | `fn x: ...` | First-Class Funktion |

## Operatoren

| Operator | Beschreibung |
|----------|-------------|
| `+ - * / %` | Arithmetik |
| `== != < > <= >=` | Vergleiche |
| `++` | String-Verkettung |
| `>` | Pipeline |
| `\|` `->` | Match-Case |

## Eingebaute Funktionen (34 Builtins)

### IO
| Funktion | Beschreibung |
|----------|-------------|
| `print ...` | Werte ausgeben |
| `read_file pfad` | Datei lesen |
| `write_file pfad inhalt` | Datei schreiben |

### Strings
| Funktion | Beschreibung |
|----------|-------------|
| `upper s` | Großbuchstaben |
| `lower s` | Kleinbuchstaben |
| `trim s` | Leerzeichen entfernen |
| `split s t` | An Trennzeichen teilen |
| `join list t` | Mit Trennzeichen verbinden |
| `contains s sub` | Enthalten-Prüfung |

### Listen
| Funktion | Beschreibung |
|----------|-------------|
| `len list` | Länge |
| `push list x...` | Elemente anhängen |
| `pop list` | Letztes Element entfernen |
| `at list i` | Element an Index |
| `sort list` | Sortieren |
| `range n` / `range a b` | Zahlenbereich |
| `map list fn` | Transformieren (Tree-Walker) |
| `filter list fn` | Filtern (Tree-Walker) |
| `reduce list fn init` | Falten (Tree-Walker) |
| `each list fn` | Für jedes Element (Tree-Walker) |

### Math
| Funktion | Beschreibung |
|----------|-------------|
| `abs n` | Absolutwert |
| `min a b ...` | Minimum |
| `max a b ...` | Maximum |
| `pow b e` | Potenz |
| `sqrt n` | Quadratwurzel |
| `round n` | Runden |

### Map
| Funktion | Beschreibung |
|----------|-------------|
| `keys map` | Schlüssel als Liste |
| `values map` | Werte als Liste |

### Type-Checks
| Funktion | Beschreibung |
|----------|-------------|
| `is_num x` | Ist Zahl? |
| `is_str x` | Ist String? |
| `is_list x` | Ist Liste? |
| `is_map x` | Ist Map? |
| `is_nil x` | Ist nil? |

### Konvertierung
| Funktion | Beschreibung |
|----------|-------------|
| `to_str x` | Zu String |
| `to_num x` | Zu Zahl |

## Beispiele

```
examples/hello.pulse     — Hallo Welt
examples/fib.pulse       — Fibonacci-Zahlen  
examples/fizzbuzz.pulse  — FizzBuzz
examples/pipeline.pulse  — Pipeline-Ketten
```

## Entwicklungsstand

- [x] M0: Projekt-Setup
- [x] M1: Lexer (Einrückungs-Tracking, 15 Tests)
- [x] M2+M3: Parser (Pratt-Parsing + Block-Strukturen, 20 Tests)
- [x] M4: AST + CLI
- [x] M5: Tree-Walk Interpreter
- [x] M6: Datenstrukturen (Listen, Maps)
- [x] M7: Pipeline-Semantik (horizontal + vertikal)
- [x] M8: Bytecode-Compiler + Stack-VM (25 Tests)
- [x] M9: Standardbibliothek (60+ Builtins: FS, HTTP, JSON, TCP)
- [x] M10: REPL + CLI-Tooling
- [x] v0.4: while, break, continue, slice, import, anonyme Fns

## REPL

```bash
make repl        # oder: ./bin/pulse
```

```
>>> fn double x
...     x * 2
...
>>> double 21 > print
42
>>> :vm          # VM-Modus umschalten
>>> 1 + 2
3
>>> :quit
```

Befehle: `:quit`, `:help`, `:clear`, `:vm`

## Tests (60 total)

```bash
go test ./...
# ok  pkg/lexer    0.004s  (15 tests)
# ok  pkg/parser   0.004s  (20 tests)
# ok  pkg/vm       0.020s  (25 tests)
```

## Projektstruktur

```
pulse/
├── cmd/pulse/main.go       # CLI
├── pkg/lexer/              # Lexer (15 Tests)
├── pkg/parser/             # Parser (20 Tests)  
├── pkg/ast/                # AST-Definitionen
├── pkg/object/             # Laufzeit-Werte
├── pkg/eval/               # Tree-Walk Evaluator
├── examples/               # Beispiel-Dateien
├── Makefile
└── Plan.md                 # Vollständiger Entwicklungsplan
```
