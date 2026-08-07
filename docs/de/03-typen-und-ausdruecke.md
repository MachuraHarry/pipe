# 3. Typen und Ausdrücke

## 3.1 Datentypen im Detail

Pipe hat 7 eingebaute Datentypen, plus benutzerdefinierte Structs:

| Typ | Beschreibung | Literal | Interne Repräsentation |
|-----|-------------|---------|------------------------|
| `nil` | Nichts / Abwesenheit | `nil` | `NilObject` |
| `bool` | Wahrheitswert | `true`, `false` | `Boolean` |
| `num` | 64-Bit Zahl | `42`, `3.14` | `Integer` (int64) / `Float` (float64) |
| `str` | Unicode-String | `"Hallo"`, `` `mehrzeilig` `` | `String` |
| `list` | Dynamische Liste | `[1, 2, 3]` | `List` |
| `map` | Assoziative Map | `{a: 1, b: 2}` | `Map` |
| `struct` | Benutzerdef. Datensatz | `struct P: x, y` | `StructDef` / `StructInstance` |
| `fn` | First-Class Funktion | `fn x` (block form)<br>`fn x: expr` (inline) | `Function` / `CompiledFunction` / `Closure` |

### nil

`nil` repräsentiert die Abwesenheit eines Werts. Es ist der Default-Wert
für nicht gesetzte Map-Keys und fehlgeschlagene Lookups.

```pipe
x: nil
-- nil
print x
```

### bool

Wahrheitswerte: `true` und `false`.

Nur `false` und `nil` sind **falsy** — alle anderen Werte sind truthy
(inklusive `0`, `""`, `[]`, `{}`).

### num

Zahlen werden als 64-Bit Integer (`int64`) oder Float (`float64`)
dargestellt. Pipe konvertiert automatisch:

```pipe
-- Integer
ganzzahl: 42
-- Float
kommazahl: 3[14]
-- Integer
negativ: -100
```

**Integer + Integer → Integer**  
**Integer + Float → Float**  
**Float + Float → Float**

### str

Unicode-Strings in **doppelten Anführungszeichen** mit Escape-Sequenzen:

```pipe
text: "Hallo Welt"
escapet: "Zeile 1\nZeile 2\tTab"
quote: "Er sagte: \"Hallo\""
null_byte: "Null\0Byte"

-- Mehrzeilige Strings mit Backticks:
mehrzeilig: `Zeile 1
Zeile 2
Zeile 3`
```

**Unterstützte Escape-Sequenzen:** `\n`, `\t`, `\r`, `\\`, `\"`, `\0`

### list

Geordnete, dynamische Listen mit wahlfreiem Zugriff:

```pipe
leer: []
zahlen: [10, 20, 30, 40, 50]
gemischt: [1, "zwei", true, nil, [1, 2]]
```

### map

Assoziative Maps (Schlüssel-Wert-Paare), nur String-Keys:

```pipe
leer: {}
person: {name: "Anna", alter: 28, stadt: "Berlin"}
komplex: {name: "Pipe", version: 1, tags: ["sprache", "skript"]}
```

### fn

Funktionen sind First-Class Citizens — sie können in Variablen gespeichert,
als Argumente übergeben und von Funktionen zurückgegeben werden.

```pipe
-- Anonyme Funktion in Variable
verdoppler: fn x
    x * 2

-- Funktion als Argument
fn anwenden f wert
    f wert

-- 42
print (anwenden verdoppler 21)
```

### struct

Structs sind benutzerdefinierte Datensätze mit benannten Feldern. Sie erlauben es, zusammenhängende Daten in einem Wert zu gruppieren.

```pipe
-- Block-Form mit optionalen Default-Werten
struct Person
    name: "Unbekannt"
    alter: 0
    aktiv: true

-- Inline-Form (kompakt, keine Defaults)
struct Punkt: x, y

-- Instanzen erstellen (positionale Argumente)
alice: Person "Alice" 30 true
ursprung: Punkt 0 0
alice.name    -- "Alice"
ursprung.x    -- 0
```

## 3.2 Operatoren — Vollständige Referenz

### Arithmetische Operatoren

| Operator | Beschreibung | Beispiel | Ergebnis |
|----------|-------------|---------|----------|
| `+` | Addition | `10 + 3` | `13` |
| `-` | Subtraktion | `10 - 3` | `7` |
| `*` | Multiplikation | `10 * 3` | `30` |
| `/` | Division | `10 / 3` | `3` (Integer) / `3.333` (Float) |
| `%` | Modulo (Rest) | `10 % 3` | `1` |
| `**` | Potenz | `2 ** 10` | `1024` |

**Divisionsverhalten:**
- `Integer / Integer` → Integer (Ganzzahldivision)
- Sobald ein Float beteiligt ist → Float-Division

### Vergleichsoperatoren

| Operator | Beschreibung | Beispiel |
|----------|-------------|---------|
| `==` | Gleich | `a == b` |
| `!=` | Ungleich | `a != b` |
| `<` | Kleiner als | `a < b` |
| `>` | Größer als | `a > b` |
| `<=` | Kleiner oder gleich | `a <= b` |
| `>=` | Größer oder gleich | `a >= b` |

Vergleiche funktionieren für **Zahlen, Strings, Booleans** und geben einen Boolean zurück.

### Logische Operatoren

| Operator | Beschreibung | Beispiel |
|----------|-------------|---------|
| `!` | Logisches NICHT | `!true` → `false` |
| `&&` | Logisches UND (Short-circuit) | `a && b` |
| `\|\|` | Logisches ODER (Short-circuit) | `a \|\| b` |

**Short-circuit Semantik:**
- `a && b` — wenn `a` falsy ist, wird `b` nicht ausgewertet
- `a \|\| b` — wenn `a` truthy ist, wird `b` nicht ausgewertet

### String-Operator

| Operator | Beschreibung | Beispiel | Ergebnis |
|----------|-------------|---------|----------|
| `++` | String-Verkettung | `"Hallo " ++ "Welt"` | `"Hallo Welt"` |
| `++` | String + Zahl | `"Wert: " ++ (to_str 42)` | `"Wert: 42"` |

### Compound Assignment

| Operator | Langform | Kurzform |
|----------|----------|----------|
| `+=` | `x: x + n` | `x += n` |
| `-=` | `x: x - n` | `x -= n` |
| `*=` | `x: x * n` | `x *= n` |
| `/=` | `x: x / n` | `x /= n` |
| `%=` | `x: x % n` | `x %= n` |

### Unäre Operatoren

| Operator | Beschreibung | Beispiel |
|----------|-------------|---------|
| `-` | Negation | `-42` → `-42` |
| `!` | Logisches NICHT | `!true` → `false` |

## 3.3 Operator-Präzedenz

Von höchster zu niedrigster Priorität:

| Ebene | Operatoren | Beschreibung |
|-------|-----------|-------------|
| 13 | `.` | Dot-Access (Feldzugriff) |
| 12 | `()` `[]` | Funktionsaufruf, Index-Zugriff |
| 11 | `!`, `-` (unär) | Logisches NICHT, Negation |
| 10 | `++` | String-Verkettung |
| 9 | `**` | Potenz |
| 8 | `*`, `/`, `%` | Multiplikativ |
| 7 | `+`, `-` | Additiv |
| 6 | `<`, `>`, `<=`, `>=` | Vergleich |
| 5 | `==`, `!=` | Gleichheit |
| 4 | `>`, `>>` | Pipeline |
| 3 | `&&` | Logisches UND |
| 2 | `\|\|` | Logisches ODER |
| 1 | `:` | Zuweisung |

**Höhere Priorität = stärkere Bindung.** Operatoren auf derselben Ebene
sind links-assoziativ.

### Präzedenz-Beispiele

```pipe
-- 1 + (2 * 3) = 7, nicht (1 + 2) * 3
1 + 2 * 3
-- -(x ** 2), nicht (-x) ** 2
-x ** 2
-- (a && b) || c
a && b || c
```

## 3.4 Typ-Prüfung

Pipe bietet eingebaute Funktionen zur Typ-Prüfung:

```pipe
-- "INTEGER"
print (type_of 42)
-- "FLOAT"
print (type_of 3[14])
-- "STRING"
print (type_of "hallo")
-- "BOOLEAN"
print (type_of true)
-- "NIL"
print (type_of nil)
-- "LIST"
print (type_of ([1, 2]))
-- "MAP"
print (type_of {a: 1})
-- "FUNCTION"
print (type_of (fn x
    x))
-- "STRUCT"
struct P: x, y
print (type_of P)

-- true
print (is_num 42)
-- true
print (is_str "hallo")
-- true
print (is_list ([1, 2]))
-- true
print (is_map {a: 1})
-- true
print (is_nil nil)
```

## 3.5 Typ-Konvertierung

```pipe
-- "42"
print (to_str 42)
-- "true"
print (to_str true)
-- 42
print (to_num "42")
-- 3[14]
print (to_num "3[14]")
-- 1
print (to_num true)
-- 0
print (to_num false)
```

## 3.6 Mehrzeilige Strings (Backticks)

Backtick-Strings können Zeilenumbrüche enthalten, ohne Escape-Sequenzen:

```pipe
html: `<html>
    <body>
        <h1>Hallo</h1>
    </body>
</html>`

print html
```
