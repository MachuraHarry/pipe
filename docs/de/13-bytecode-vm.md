# 13. Bytecode-VM

Die Pipe-VM ist eine **Stack-basierte virtuelle Maschine** mit 47 Opcodes.
Sie kompiliert den AST zu Bytecode und führt ihn effizient aus.

## 13.1 Architektur

```
┌──────────┐     ┌──────────────┐     ┌──────────┐
│   AST    │ ──► │   Compiler   │ ──► │ Bytecode │
└──────────┘     └──────────────┘     └──────────┘
                                              │
                    ┌──────────┐               │
                    │   .pipec │ ◄─────────────┘
                    └──────────┘     (Cache)
                                              │
                                              ▼
                                       ┌──────────┐
                                       │    VM    │
                                       └──────────┘
```

## 13.2 Komponenten

### Compiler

- Übersetzt jeden AST-Knoten in Bytecode-Instruktionen
- Verwaltet eine **Symbol-Tabelle** mit 4 Scopes:
  - `GlobalScope` — Top-Level Variablen
  - `LocalScope` — Funktions-lokale Variablen
  - `FreeScope` — Closure-Variablen (aus äußerem Scope)
  - `BuiltinScope` — Eingebaute Funktionen
- **Constant Pool**: Alle Literal-Werte werden zentral gespeichert
- **Loop Patching**: Sprungadressen für break/continue werden beim Loop-Ende aufgelöst

### VM (Virtual Machine)

- **Stack-basiert**: Alle Operationen arbeiten auf einem Operanden-Stack
- **Call-Frames**: Jeder Funktionsaufruf erzeugt einen neuen Frame mit:
  - `fn` — Die CompiledFunction/Closure
  - `ip` — Instruction Pointer (aktuelle Position im Bytecode)
  - `basePointer` — Zeiger auf den Stack-Anfang des Frames
- **Stack-Größe**: 2048 Werte
- **Maximale Frames**: 1024

## 13.3 Alle Opcodes (47)

### Konstanten & Literale

| Opcode | Operanden | Beschreibung |
|--------|-----------|-------------|
| `OpConstant` | const_idx (2 Bytes) | Konstante aus dem Pool auf den Stack pushen |
| `OpTrue` | — | `true` auf den Stack |
| `OpFalse` | — | `false` auf den Stack |
| `OpNil` | — | `nil` auf den Stack |

### Stack-Operationen

| Opcode | Beschreibung |
|--------|-------------|
| `OpPop` | Oberstes Stack-Element entfernen |
| `OpDup` | Oberstes Stack-Element duplizieren |

### Arithmetik (binär)

| Opcode | Operation |
|--------|-----------|
| `OpAdd` | Addition |
| `OpSub` | Subtraktion |
| `OpMul` | Multiplikation |
| `OpDiv` | Division |
| `OpMod` | Modulo |
| `OpPow` | Potenz |

### Vergleiche (binär)

| Opcode | Operation |
|--------|-----------|
| `OpEqual` | `==` |
| `OpNotEqual` | `!=` |
| `OpGreater` | `>` |
| `OpLess` | `<` |
| `OpGte` | `>=` |
| `OpLte` | `<=` |

### Unäre Operatoren

| Opcode | Operation |
|--------|-----------|
| `OpMinus` | Negation (-) |
| `OpNot` | Logisches NICHT (!) |
| `OpConcat` | String-Verkettung (++) |

### Variablen-Zugriff

| Opcode | Operanden | Beschreibung |
|--------|-----------|-------------|
| `OpGetGlobal` | idx (2 Bytes) | Globale Variable auf Stack |
| `OpSetGlobal` | idx (2 Bytes) | Globalen Wert setzen (peek) |
| `OpGetLocal` | idx (2 Bytes) | Lokale Variable auf Stack |
| `OpSetLocal` | idx (2 Bytes) | Lokalen Wert setzen (peek) |
| `OpGetBuiltin` | idx (2 Bytes) | Builtin-Funktion auf Stack |
| `OpGetFree` | idx (2 Bytes) | Free-Variable (Closure) auf Stack |

### Kontrollfluss

| Opcode | Operanden | Beschreibung |
|--------|-----------|-------------|
| `OpJump` | target (2 Bytes) | Unbedingter Sprung |
| `OpJumpNotTruthy` | target (2 Bytes) | Sprung wenn oberstes Stack-Element falsy |
| `OpJumpBackward` | target (2 Bytes) | Rückwärts-Sprung (für while-Schleifen) |
| `OpCheckError` | — | Prüft ob oberstes Element ein Error ist |

### Funktionsaufrufe

| Opcode | Operanden | Beschreibung |
|--------|-----------|-------------|
| `OpCall` | num_args (2 Bytes) | Funktion mit N Argumenten aufrufen |
| `OpReturn` | — | Zurückkehren (nil) |
| `OpReturnValue` | — | Zurückkehren (obersten Wert vom Stack) |
| `OpClosure` | const_idx, num_free (2+2 Bytes) | Closure aus CompiledFunction + Free-Vars erstellen |
| `OpHalt` | — | Programm beenden |

### Datenstrukturen

| Opcode | Operanden | Beschreibung |
|--------|-----------|-------------|
| `OpList` | num_elems (2 Bytes) | Liste aus N Stack-Werten erstellen |
| `OpMap` | num_pairs (2 Bytes) + key_idxs | Map aus N×2 Stack-Werten erstellen |
| `OpDot` | key_idx (2 Bytes) | Feldzugriff auf Map (Dot-Notation) |

## 13.4 Wie Kompilierung funktioniert

### Beispiel: `1 + 2`

```
Quelltext: 1 + 2

AST:
  InfixExpression
    Left:  IntegerLiteral(1)
    Right: IntegerLiteral(2)

Bytecode:
  0000: OpConstant 0    -- Push 1 (Konstante #0)
  0003: OpConstant 1    -- Push 2 (Konstante #1)
  0006: OpAdd           -- Addiere
  0007: OpHalt          -- Ende

Constant Pool: [1, 2]
```

### Beispiel: `fn double x: x * 2`

```
Bytecode der main-Funktion:
  0000: OpClosure 0 0   -- Erstelle Closure für double
  0003: OpSetGlobal 0   -- Speichere als Globale Variable #0
  0005: OpHalt

Bytecode von double:
  0000: OpGetLocal 0    -- Push x (Local #0)
  0002: OpConstant 0    -- Push 2 (Konstante #0)
  0005: OpMul           -- Multipliziere
  0006: OpReturnValue   -- Return

Symbol-Tabelle (main):
  #0 (Global): double
```

### Beispiel: Closure

```pipe
fn make_adder n
    fn adder x
        x + n

add5: make_adder 5
```

```
adder erfasst n als Free-Variable:
  OpGetFree 0    -- Push n (Free #0)
  OpGetLocal 0   -- Push x (Local #0)
  OpAdd
  OpReturnValue

Closure-Erstellung:
  OpClosure const_idx num_free
  → pop N Werte vom Stack als Free-Variablen
  → erzeuge Closure{fn, free}
```

## 13.5 Instruction-Encoding

Jeder Opcode ist 1 Byte, gefolgt von 0-4 Operanden-Bytes:

```
┌────────┬──────────┬──────────┐
│ Opcode │ Operand1 │ Operand2 │
│ 1 Byte │  2 Bytes │  2 Bytes │
└────────┴──────────┴──────────┘
```

Operanden sind 16-Bit unsigned Integer (Big-Endian).

```go
// Encoding: Make(OpConstant, 42)
// → [OpConstant, 0x00, 0x2A]    (3 Bytes)
```

## 13.6 VM Execute-Loop (vereinfacht)

```
while ip < len(instructions):
    op = instructions[ip]
    switch op:
        case OpConstant:
            idx = readUint16(ip+1)
            push(constants[idx])
            ip += 3
        case OpAdd:
            right = pop()
            left = pop()
            push(left + right)
            ip += 1
        case OpCall:
            numArgs = readUint16(ip+1)
            fn = stack[sp - 1 - numArgs]
            newFrame(fn, numArgs)
            ip += 3
        ...
```

## 13.7 Symbol-Tabelle

```go
type SymbolScope string

const (
    GlobalScope  SymbolScope = "GLOBAL"
    LocalScope   SymbolScope = "LOCAL"
    FreeScope    SymbolScope = "FREE"
    BuiltinScope SymbolScope = "BUILTIN"
)
```

Die Symbol-Tabelle wird während der Kompilierung aufgebaut:
- Bei `fn`, `match`, `if`, `while`, `for`, `try` wird ein neuer Scope betreten
- `define(name)` — Definiert ein neues Symbol im aktuellen Scope
- `resolve(name)` — Sucht rekursiv in allen äußeren Scopes
- Wenn ein Symbol im äußeren Scope gefunden wird → `FreeScope` (Closure)

## 13.8 Stack-Größen

| Konstante | Wert |
|-----------|------|
| `StackSize` | 2048 |
| `MaxFrames` | 1024 |
| `GlobalsSize` | 65536 |

Stack-Overflow führt zu einem Laufzeitfehler.
