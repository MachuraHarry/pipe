# Pipe — Eine minimalistische, pipeline-basierte Skriptsprache

## Entwicklungsplan

---

## 1. Sprachvision

**Kernprinzip:** Alles ist Datenfluss. Programme werden nicht als verschachtelte Funktionsaufrufe geschrieben, sondern als gerichteter Fluss von Daten durch Transformationen.

**Designziele:**
- Die Grammatik passt auf eine Seite
- Ein kompletter Parser in <500 LOC
- Keine überraschenden Verhaltensweisen (Principle of Least Astonishment)
- Einrückungsbasiert, keine Klammern für Code-Blöcke
- Inspiriert von Lua's Minimalismus, aber mit moderner Pipeline-Syntax

---

## 2. Sprachspezifikation

### 2.1 Lexikalische Struktur

```
Tokens:
  KEYWORD       fn, match, if, else, let, in, true, false, nil
  IDENTIFIER    [a-zA-Z_][a-zA-Z0-9_]*
  NUMBER        [0-9]+(\.[0-9]+)?
  STRING        "..."  mit Escape-Sequenzen
  OPERATOR      + - * / % == != < > <= >= ++ | >
  PUNCTUATION   ( ) [ ] { } , . -> :
  COMMENT       -- bis Zeilenende
```

### 2.2 Grammatik (EBNF)

```ebnf
program        = statement*

statement      = function_def
               | variable_def
               | expression

function_def   = "fn" IDENTIFIER param* ":" NEWLINE INDENT statement* DEDENT

variable_def   = IDENTIFIER ":" NEWLINE INDENT expression DEDENT

expression     = pipeline

pipeline       = primary ("\n" INDENT ">" primary)*   -- vertikale Pipeline
               | primary ("|" primary)*               -- horizontale Pipeline (match cases)

primary        = literal
               | IDENTIFIER
               | function_call
               | match_expr
               | if_expr
               | list_literal
               | map_literal
               | group

function_call  = IDENTIFIER argument*
argument       = expression

match_expr     = "match" expression NEWLINE INDENT case+ DEDENT
case           = "|" pattern "->" expression
pattern        = literal | IDENTIFIER | "_"

if_expr        = "if" expression NEWLINE INDENT expression DEDENT
                 ("else" NEWLINE INDENT expression DEDENT)?

list_literal   = "[" (expression ("," expression)*)? "]"
map_literal    = "{" (IDENTIFIER ":" expression ("," IDENTIFIER ":" expression)*)? "}"

literal        = NUMBER | STRING | "true" | "false" | "nil"
group          = "(" expression ")"
```

### 2.3 Datentypen

| Typ | Beschreibung | Literal |
|-----|-------------|---------|
| `nil` | Nichts / Abwesenheit | `nil` |
| `bool` | Wahrheitswert | `true`, `false` |
| `num` | 64-Bit Gleitkommazahl | `42`, `3.14` |
| `str` | Unicode-String | `"Hallo"` |
| `list` | Dynamische Liste | `[1, 2, 3]` |
| `map` | Assoziative Map (wie Lua's Table) | `{a: 1, b: 2}` |
| `fn` | First-Class Funktionen | `fn greet ...` |

### 2.4 Semantik

#### Pipeline (`>`)
```
wert
    > f
    > g
    > h
```
Bedeutung: `h(g(f(wert)))`. Auf jeder Stufe wird das Ergebnis der linken Seite als **erstes Argument** an die rechte Funktion übergeben. Die Einrückung macht den Datenfluss sichtbar.

#### Match
```
match wert
    | 0      -> "null"
    | 1..10  -> "klein"
    | _      -> "sonst"
```
Erstes passendes Pattern gewinnt. `_` ist Wildcard.

#### Variablen
```
name: "Welt"
zahlen: [1, 2, 3]
```
Variablen sind standardmäßig **lokal zum aktuellen Scope**. Kein `let`-Keyword — die Zuweisung mit `:` reicht.

#### Funktionen
```
fn add a b
    a + b

fn greet name
    "Hallo " ++ name
```
Rückgabewert ist der letzte Ausdruck im Funktionskörper. Kein `return`-Keyword nötig.

---

## 3. Implementierungsarchitektur

### 3.1 Implementierungssprache: **Go**

**Begründung:**
- Einfache Syntax, schnelle Kompilierung
- Eingebaute UTF-8-Unterstützung
- Garbage Collector (muss keinen eigenen bauen)
- Statisch gelinkte Binaries (perfekt für Distribution)
- Exzellentes Tooling (`go fmt`, `go test`)
- Leichtgewichtige Goroutinen für spätere Parallelität

### 3.2 Architektur-Diagramm

```
Quelltext (.pipe)
    │
    ▼
┌─────────────┐
│   Lexer     │  → Token-Stream
└─────────────┘
    │
    ▼
┌─────────────┐
│   Parser    │  → AST
└─────────────┘
    │
    ▼
┌─────────────┐
│  Compiler   │  → Bytecode
└─────────────┘
    │
    ▼
┌─────────────┐
│     VM      │  → Ausführung
└─────────────┘
```

### 3.3 Komponenten im Detail

#### Lexer (`pkg/lexer`)
- Zustandsautomat mit Peek-Funktion
- Verarbeitet Einrückung als `INDENT`/`DEDENT`-Tokens (wie Python)
- Ausgabe: Kanal von `Token`-Structs

#### Parser (`pkg/parser`)
- Rekursiver Abstieg (Recursive Descent)
- Pratt-Parsing für Operatoren
- Baut AST-Knoten
- Fehler mit Zeilennummern und Kontext

#### AST (`pkg/ast`)
```go
type Node interface {
    TokenLiteral() string
}

type Program struct {
    Statements []Statement
}

type ExpressionStatement struct {
    Expression Expression
}

type FnLiteral struct {
    Parameters []*Identifier
    Body       *BlockStatement
}

type MatchExpression struct {
    Value Expression
    Cases []Case
}

type Case struct {
    Pattern Expression
    Body    Expression
}

type PipelineExpression struct {
    Stages []Expression
}
```

#### Bytecode-Compiler (`pkg/compiler`)
- Übersetzt AST in eine flache Bytecode-Sequenz
- Einfache Stack-basierte VM-Instruktionen
- Opcodes: `PUSH`, `POP`, `ADD`, `CALL`, `JUMP`, `MATCH`, etc.

#### VM (`pkg/vm`)
- Stack-Maschine mit Operanden-Stack
- Call-Stack für Funktionsaufrufe
- Alle Werte sind `Value`-Interface:
```go
type Value interface {
    Type() ValueType
    String() string
}
```

#### Standardbibliothek (`pkg/stdlib`)
- `print`, `len`, `push`, `pop`
- `map`, `filter`, `reduce`
- `sort`, `unique`
- `http.get`, `parse_json`
- `read_file`, `write_file`

---

## 4. Umsetzungsplan (10 Meilensteine)

### Meilenstein 0: Projekt-Setup
- [ ] Go-Modul initialisieren: `go mod init github.com/user/pipe`
- [ ] Verzeichnisstruktur anlegen
- [ ] `main.go` mit CLI-Eingang (`pipe datei.pipe` und REPL-Modus)
- [ ] Makefile mit `build`, `test`, `run` Targets

### Meilenstein 1: Lexer
- [ ] Token-Typen definieren
- [ ] Lexer implementiert Einrückungs-Tracking
- [ ] String-Literale mit Escapes
- [ ] Zahlen (Integer + Float)
- [ ] Kommentare (`--`)
- [ ] **Test:** Lexer gibt korrekte Token für alle Beispiele aus

### Meilenstein 2: Parser (Ausdrücke)
- [ ] Pratt-Parser für Literale und Operatoren
- [ ] Funktionsaufrufe
- [ ] Pipeline-Ausdrücke (`>`, `|`)
- [ ] **Test:** Parse validiert `1 + 2 * 3`, `foo > bar > baz`

### Meilenstein 3: Parser (Statements)
- [ ] Variablendefinitionen (`name: wert`)
- [ ] Funktionsdefinitionen (`fn name args: body`)
- [ ] `match`-Ausdrücke
- [ ] `if`/`else`-Ausdrücke
- [ ] Block-Strukturen mit Einrückung
- [ ] **Test:** Vollständige Programme parsen

### Meilenstein 4: AST + Pretty-Printer
- [ ] AST-Knoten definieren
- [ ] AST-Walker zur Validierung
- [ ] Pretty-Printer (AST → formatierter Pipe-Code)
- [ ] **Test:** Roundtrip: Pipe → AST → Pipe ist äquivalent

### Meilenstein 5: Tree-Walk Interpreter
- [ ] Evaluator mit Umgebungen (Scopes)
- [ ] Primitive Werte (nil, bool, num, str)
- [ ] Arithmetik, Vergleichsoperatoren
- [ ] Variablen-Definition und -Zugriff
- [ ] Funktionsdefinition und -Aufruf
- [ ] **Test:** Fibonacci, FizzBuzz laufen korrekt

### Meilenstein 6: Datenstrukturen
- [ ] Listen (`[1, 2, 3]`)
- [ ] Maps (`{a: 1, b: 2}`)
- [ ] Index-Zugriff (`liste[0]`, `map["key"]`)
- [ ] `match` mit Listen-Patterns
- [ ] **Test:** Listen-Operationen, Map-Lookups

### Meilenstein 7: Pipeline-Semantik
- [ ] `wert > f > g > h` als Threading-Makro
- [ ] `wert > match ...` funktioniert
- [ ] Pipeline mit eingebauten Funktionen (`> filter`, `> map`)
- [ ] **Test:** Komplexe Pipeline-Ketten

### Meilenstein 8: Bytecode-Compiler + VM
- [ ] Opcodes definieren
- [ ] Compiler: AST → Bytecode
- [ ] VM: Stack-Maschine mit Execute-Loop
- [ ] Funktionsaufrufe über Call-Stack
- [ ] **Benchmark:** VM mindestens 10x schneller als Tree-Walker

### Meilenstein 9: Standardbibliothek
- [ ] `print`, `input`
- [ ] Listen-Funktionen (`len`, `push`, `pop`, `at`)
- [ ] Map-Funktionen (`get`, `set`, `keys`, `values`)
- [ ] String-Funktionen (`upper`, `lower`, `trim`, `split`)
- [ ] `map`, `filter`, `reduce`, `each`
- [ ] `sort`, `unique`
- [ ] `range`, `to`

### Meilenstein 10: IO + Tooling
- [ ] Datei lesen/schreiben
- [ ] HTTP-Requests
- [ ] JSON-Parsing
- [ ] REPL mit History (Readline)
- [ ] Fehlerformatierung (wie Rust's Compiler-Fehler)
- [ ] Dokumentation und Beispiele

---

## 5. Projektstruktur

```
pipe/
├── Plan.md                  # Dieser Plan
├── README.md                # Spracheinführung
├── Makefile                 # Build-Automatisierung
├── go.mod
├── go.sum
├── cmd/
│   └── pipe/
│       └── main.go          # CLI-Einstiegspunkt
├── pkg/
│   ├── lexer/
│   │   ├── lexer.go         # Lexer-Implementierung
│   │   ├── lexer_test.go
│   │   └── token.go         # Token-Typen
│   ├── parser/
│   │   ├── parser.go        # Parser (rekursiver Abstieg)
│   │   ├── parser_test.go
│   │   └── error.go         # Parse-Fehler
│   ├── ast/
│   │   ├── ast.go           # AST-Knoten-Definitionen
│   │   ├── ast_test.go
│   │   └── printer.go       # Pretty-Printer
│   ├── object/
│   │   ├── object.go        # Laufzeit-Werte
│   │   └── environment.go   # Scopes/Variablen
│   ├── eval/
│   │   ├── eval.go          # Tree-Walk Evaluator
│   │   ├── eval_test.go
│   │   └── builtins.go      # Eingebaute Funktionen
│   ├── compiler/
│   │   ├── compiler.go      # AST → Bytecode
│   │   ├── compiler_test.go
│   │   └── opcode.go        # Opcode-Definitionen
│   ├── vm/
│   │   ├── vm.go            # Stack-VM
│   │   ├── vm_test.go
│   │   └── frame.go         # Call-Frames
│   └── stdlib/
│       ├── stdlib.go        # Standardbibliothek
│       ├── io.go
│       ├── net.go
│       └── math.go
├── examples/
│   ├── hello.pipe
│   ├── fib.pipe
│   ├── fizzbuzz.pipe
│   ├── pipeline.pipe
│   └── api.pipe
└── test/
    └── integration/
        └── integration_test.go
```

---

## 6. Geschätzter Zeitaufwand

| Meilenstein | Aufwand | Kumulativ |
|-------------|---------|-----------|
| M0: Setup | 1 Tag | 1 Tag |
| M1: Lexer | 2–3 Tage | 4 Tage |
| M2: Parser (Ausdrücke) | 3–4 Tage | 8 Tage |
| M3: Parser (Statements) | 3–4 Tage | 12 Tage |
| M4: AST + Printer | 2 Tage | 14 Tage |
| M5: Tree-Walk Interpreter | 4–5 Tage | 19 Tage |
| M6: Datenstrukturen | 2–3 Tage | 22 Tage |
| M7: Pipeline-Semantik | 2 Tage | 24 Tage |
| M8: Bytecode VM | 5–7 Tage | 31 Tage |
| M9: Standardbibliothek | 3–4 Tage | 35 Tage |
| M10: IO + Tooling | 2–3 Tage | 38 Tage |

**Gesamt: ~38 Arbeitstage** (≈ 2 Monate nebenbei)

---

## 7. Risiken

| Risiko | Wahrscheinlichkeit | Mitigation |
|--------|-------------------|------------|
| Einrückungs-Parsing wird komplex | Mittel | Früh isoliert testen. Python's `tokenize` als Referenz. |
| Bytecode-VM-Performance enttäuscht | Niedrig | Erst Tree-Walker bauen, VM nur bei Bedarf. |
| Pipeline-Semantik wird unübersichtlich | Niedrig | Klare Spezifikation vor Implementierung. |
| Scope creep (immer mehr Features) | Hoch | Meilensteine strikt einhalten. "No" ist eine valide Antwort. |

---

## 8. Design-Entscheidungen (bewusst getroffen)

1. **Kein `return`-Keyword** — Letzter Ausdruck ist Rückgabewert. Reduziert Keywords.
2. **Kein `let`/`var`** — Einfache `name: wert`-Syntax. Konsistent mit Maps `{a: 1}`.
3. **`match` statt `switch`/`if-elsif`** — Ein Konstrukt für alle Verzweigungen.
4. **`|` und `>` als Strukturzeichen** — Pfeile zeigen Flussrichtung an.
5. **Einrückung, nicht `{}` oder `end`** — Weniger visuelles Rauschen.
6. **Go als Implementierungssprache** — Gute Balance aus Geschwindigkeit und Einfachheit.
