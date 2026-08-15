# 14. Architektur

## 14.1 Überblick

Pipe besteht aus **sechs Hauptkomponenten**, die den Quelltext Schritt für Schritt
verarbeiten:

```
Quelltext (.pipe)
    │
    ▼
┌─────────────┐
│   Lexer     │  → Token-Stream (67 Token-Typen, INDENT/DEDENT)
└─────────────┘
    │
    ▼
┌─────────────┐
│   Parser    │  → AST (Rekursiver Abstieg + Pratt-Parsing)
└─────────────┘
    │
    │    ┌──────────────────────────────────────┐
    ├──► │ Tree-Walker (eval)                    │ → Direkte Ausführung
    │    └──────────────────────────────────────┘
    │
    │    ┌──────────────────────────────────────┐
    └──► │ Compiler + VM                         │ → Bytecode → Stack-Maschine
         └──────────────────────────────────────┘
```

## 14.2 Lexer (`pkg/lexer/`)

**Größe:** ~204 Zeilen Go (token.go) + ~485 Zeilen (lexer.go)

### Token-Typen (52)

- **Spezial:** `ILLEGAL`, `EOF`
- **Literale:** `IDENT`, `INT`, `FLOAT`, `STRING`
- **Operatoren:** `ASSIGN`, `PLUS`, `MINUS`, `STAR`, `SLASH`, `PERCENT`, `EQ`, `NOT_EQ`, `LT`, `GT`, `LTE`, `GTE`, `CONCAT`, `BANG`, `AND`, `OR`, `PLUSEQ`, `MINUSEQ`, `STAREQ`, `SLASHEQ`, `PERCENTEQ`, `POWER`, `DOTDOT`
- **Pipeline/Match:** `PIPE` (|), `ARROW` (>), `MATCH` (->)
- **Interpunktion:** `LPAREN`, `RPAREN`, `LBRACKET`, `RBRACKET`, `LBRACE`, `RBRACE`, `COMMA`, `DOT`, `COLON`
- **Struktur:** `NEWLINE`, `INDENT`, `DEDENT`
- **Keywords:** `FN`, `MATCHKW`, `IF`, `ELSE`, `WHILE`, `FOR`, `BREAK`, `CONTINUE`, `IMPORT`, `EXPORT`, `ENUM`, `DEFER`, `RETURN`, `TRY`, `CATCH`, `TRUE`, `FALSE`, `NIL`

### INDENT/DEDENT-Mechanismus

Der Lexer trackt Einrückung mit einem **Stack** (`indentStack []int`):

```
def add(a, b):          ← Level 0
    result = a + b      ← INDENT (Level 4)
    return result       ← Level 4
                        ← DEDENT (zurück zu Level 0)
```

- Jede Zeile wird auf Einrückungs-Änderungen geprüft
- Mehr Einrückung als aktuell → `INDENT` Token emittieren, Level auf Stack pushen
- Weniger Einrückung → `DEDENT` Token(s) emittieren, Level(s) von Stack poppen

### Besonderheiten

- **Klammern-Inhalt:** Innerhalb von `()`, `[]`, `{}` wird Einrückung ignoriert
  (ermöglicht mehrzeilige Literale)
- **Backtick-Strings:** `` `...` `` Strings sind mehrzeilig ohne Escapes
- **Kommentare:** `--` bis Zeilenende; werden beim Einrückungs-Tracking berücksichtigt
- **Zeilen/Spalten-Tracking:** Jeder Token trägt `Line` und `Col` für Fehlermeldungen

## 14.3 Parser (`pkg/parser/`)

**Größe:** ~1245 Zeilen Go

### Parser-Typen

- **Rekursiver Abstieg** für Statements (fn, while, for, break, continue, etc.)
- **Pratt-Parsing** für Ausdrücke mit Operator-Präzedenz

### Operator-Präzedenz

| Ebene | Operatoren |
|-------|-----------|
| 1 | `\|\|` |
| 2 | `&&` |
| 3 | `>` (Pipeline) |
| 4 | `==`, `!=` |
| 5 | `<`, `>`, `<=`, `>=` |
| 6 | `+`, `-` |
| 7 | `*`, `/`, `%` |
| 8 | `**` |
| 9 | `++` |
| 10 | `!`, `-` (unär) |
| 11 | `()` `[]` |
| 12 | `.` |

### Spezielle Parsing-Funktionen

- **Implicit Function Call:** `print "Hallo"` wird als `CallExpression(print, ["Hallo"])` geparst
- **Vertical Pipeline:** Eingerückte `>` Zeilen nach einem Ausdruck → `PipelineExpression`
- **Placeholder `_`:** In Pipeline-Aufrufen wird `_` durch den Pipeline-Wert ersetzt
- **Slice vs Index:** Ein Ausdruck in Klammern → InfixExpression `[]`; Ausdruck `..` Ausdruck → SliceExpression
- **Fn Literal vs Statement:** `fn name params` → FnStatement; `fn params` → FnLiteral

## 14.4 AST (`pkg/ast/`)

 **Größe:** ~417 Zeilen Go  
 **Knotentypen:** 34 (12 Statements, 20 Expressions, Programm, MatchCase)

### Statements (12 Typen)

`ExpressionStatement`, `FnStatement`, `VarStatement`, `BlockStatement`,
`BreakStatement`, `ContinueStatement`, `ReturnStatement`, `ImportStatement`,
`DeferStatement`, `ExportStatement`, `EnumStatement`, `TestStatement`

### Expressions (20 Typen)

`Identifier`, `IntegerLiteral`, `FloatLiteral`, `StringLiteral`, `BooleanLiteral`,
`NilLiteral`, `PrefixExpression`, `InfixExpression`, `PipelineExpression`,
`CallExpression`, `ListLiteral`, `MapLiteral`, `DotExpression`, `WhileExpression`,
`ForExpression`, `FnLiteral`, `SliceExpression`, `TryExpression`, `IfExpression`,
`MatchExpression`

## 14.5 Tree-Walker (`pkg/eval/`)

**Größe:** ~1207 Zeilen Go (eval.go) + ~152 Zeilen (builtins.go)

### Evaluator

Rekursive AST-Evaluation über eine große Switch-Anweisung. Jeder der 36 AST-Knoten
wird einzeln behandelt.

### Environment (Scope-Chain)

```go
type Environment struct {
    store map[string]Object
    outer *Environment    // Äußerer Scope (Closure)
}
```

Variablen-Lookup durchläuft die Environment-Chain:
1. Aktueller Scope
2. Äußerer Scope (falls vorhanden)
3. Builtins (globale Fallback-Ebene)

### Spezielle Laufzeit-Werte

| Typ | Verwendung |
|-----|-----------|
| `ReturnValue{Value}` | Wrappt return-Wert für vorzeitiges Verlassen |
| `BreakValue{}` | Signalisiert break in Schleifen |
| `ContinueValue{}` | Signalisiert continue in Schleifen |
| `DeferredExpr{Expr, Env}` | Für LIFO-Defer-Ausführung |

### Tail Call Optimization

Erkennt rekursive End-Aufrufe und ersetzt sie durch eine Schleife —
ermöglicht tiefe Rekursion (getestet bis 5000).

### Call-Stack & Error-Tracking

Jeder Funktionsaufruf wird auf einem Call-Stack registriert.
Bei Fehlern wird der Stack-Trace automatisch an die Fehlermeldung angehängt.

### Import-System

1. Pfad relativ zur Quelldatei auflösen (oder via `PIPE_PATH`)
2. Datei parsen
3. AST evaluieren
4. Exportierte Symbole in den aktuellen Scope injizieren
5. Caching: Mehrfachimporte parsen nur einmal

## 14.6 Bytecode-Compiler (`pkg/compiler/`)

**Größe:** ~1184 Zeilen Go (compiler.go) + ~117 Zeilen (opcode.go)

### Compiler-Phasen

1. **Symbol-Table aufbauen** — Definiere alle Variablen/Funktionen mit ihren Scopes
2. **AST traversieren** — Jeder Knoten erzeugt Bytecode
3. **Konstanten sammeln** — Literale in den Constant Pool
4. **Jump-Patching** — Sprungadressen für Kontrollfluss auflösen

### Short-Circuit Kompilierung

```
a && b:
    compile a
    OpDup
    OpJumpNotTruthy <after>
    OpPop
    compile b
    <after>:

a || b:
    compile a
    OpDup
    OpJumpNotTruthy <skip>
    OpPop
    OpJump <end>
    <skip>:
    OpPop
    compile b
    <end>:
```

## 14.7 Laufzeit-Typen (`pkg/object/`)

**Größe:** ~3021 Zeilen Go

### Objekt-Typen (12)

`*Integer`, `*Float`, `*String`, `*Boolean`, `*NilObject`, `*Function`,
`*CompiledFunction`, `*Closure`, `*List`, `*Map`, `*Error`, `*Result`

### Zusätzliche Typen

`*BuiltinInfo` (VM-Builtin-Wrapper), `*TcpConn`, `*TcpListener`

### TCP Connection Management

Globale Mutex-geschützte Maps für Connections und Listeners.
Monoton steigende Handle-IDs.

## 14.8 Cache-System (`pkg/cache/`)

**Größe:** ~191 Zeilen Go

### Cache-Format (`.pipec`)

```
Header:
  Magic:    "PIPEBC" (6 Bytes)
  Version:  1 (1 Byte)
  Src-Hash: SHA-256 (16 Bytes)
Body:
  numConstants (4 Bytes uint32)
  Constants (Integer/Float/String/CompiledFunction)
  instructions (length + data)
```

### API

- `LoadOrCompile(path)` — Lädt aus Cache oder kompiliert neu
- Hash-Validierung: Bei Quelltext-Änderung automatische Neukompilierung

## 14.9 Formatter (`pkg/formatter/`)

**Größe:** ~477 Zeilen Go

- AST-basierte Formatierung (liest AST, schreibt formatiert zurück)
- Fallback: Whitespace-Normalisierung wenn Parse fehlschlägt
- Einrückung: Normalisiert auf 4er-Blöcke

## 14.10 Build-System (`pkg/build/`)

**Größe:** ~94 Zeilen Go

- `Build(input, output)` — Kopiert Binary + hängt Quelltext an
- `LoadEmbedded(path)` — Extrahiert eingebetteten Quelltext
- Marker: `\nPIPEBUILD\n<size>\n<source>`

## 14.11 Technische Daten

| Metrik | Wert |
|--------|------|
| Go-Zeilen | ~23.600 (ohne Tests ~19.000) |
| Go-Packages | 13 |
| Externe Abhängigkeiten | 0 |
| Binary-Größe | ~8 MB (dependency-frei) |
| Tests | 290 (in 12 Paketen) |
| Beispiel-Programme | 52 |
| Builtins | 168 |
| Opcodes | 40 |
| AST-Knotentypen | 34 |
| Token-Typen | 66 |
| Stack-Größe (VM) | 2048 |
| Max Call-Frames | 1024 |
| Globals-Array | 65536 |
