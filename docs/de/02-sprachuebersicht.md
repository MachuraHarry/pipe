# 2. Sprachübersicht

## 2.1 Lexikalische Grundlagen

### Kommentare

Kommentare beginnen mit `--` und gelten bis zum Zeilenende:

```pipe
-- Das ist ein einzeiliger Kommentar
-- Auch hinter Code möglich (Inline-Kommentar)
x: 42
```

Es gibt keine mehrzeiligen Kommentare (Block-Kommentare).

### Einrückung

Pipe verwendet **Einrückung** zur Definition von Code-Blöcken — wie Python.
Empfohlen sind **4 Leerzeichen**, Tabs werden ebenfalls akzeptiert.

```pipe
if x > 10
    -- Eingerückt = gehört zum if
        print "x ist groß"
    -- Eingerückt = gehört zum if
        x: 0
-- Nicht eingerückt = nach dem if
print "Fertig"
```

**Wichtig:** Die erste eingerückte Zeile legt die Einrückungstiefe fest.
Alle folgenden Zeilen mit gleicher Einrückung gehören zum selben Block.

### Whitespace

Leerzeilen werden ignoriert. Leerzeichen zwischen Tokens sind erforderlich
(nur dort, wo Tokens sonst verschmelzen würden).

## 2.2 Variablen

Variablen werden mit `name: wert` definiert und neu zugewiesen:

```pipe
zaehler: 0
-- Neuzuweisung: jetzt 1
zaehler: zaehler + 1

name: "Pipe"
version: 1
pi: 3[14159]
aktiv: true
nichts: nil
```

**Compound Assignment** (seit v0.5):

```pipe
x: 10
-- x = 15
x: x + 5
-- x = 12
x: x - 3
-- x = 24
x: x * 2
-- x = 6
x: x / 4
-- x = 2
x: x % 4
```

**Gültigkeitsbereich:** Variablen sind im aktuellen Block sichtbar.
Funktionen sehen ihre eigenen Parameter und haben Zugriff auf den
umschließenden Scope (Closures).

## 2.3 Funktionsaufrufe

Funktionsaufrufe verwenden **Leerzeichen** statt Kommas:

```pipe
-- Ein Argument, ohne Klammern
print "Hallo"
-- Rechenausdruck als Argument braucht Klammern
print (1 + 2)
-- Verschachtelter Aufruf
print (addiere 3 4)
```

**Implicit Function Call (Space-based):** Wenn nach einem Ausdruck ein
Identifier folgt, wird dies als Funktionsaufruf interpretiert:

```pipe
-- print("Wert")
print "Wert"
-- print(fib(10))
fib 10 > print
```

**Wichtig:** Rechenausdrücke als Argumente brauchen Klammern, da sonst
der Operator Vorrang hat:

```pipe
-- Richtig: print(3)
print (1 + 2)
-- print 1 + 2              -- Falsch: würde (print 1) + 2 bedeuten
```

## 2.4 Pipe-Dateiendung

Pipe-Dateien haben die Endung **`.pipe`**.

## 2.5 Alle Keywords

| Keyword | Beschreibung |
|---------|-------------|
| `fn` | Funktionsdefinition |
| `match` | Pattern Matching |
| `if` | Bedingung |
| `else` | Alternative in if/match |
| `while` | While-Schleife |
| `for` | For-in-Schleife |
| `in` | Iterator in for-in |
| `break` | Schleife abbrechen |
| `continue` | Nächster Schleifendurchlauf |
| `import` | Modul laden |
| `export` | Symbol exportieren |
| `enum` | Enumeration definieren |
| `defer` | Verzögerte Ausführung |
| `return` | Vorzeitiges Verlassen einer Funktion |
| `try` | Fehlerbehandlung (try-Block) |
| `catch` | Fehlerbehandlung (catch-Block) |
| `true` | Wahrheitswert wahr |
| `false` | Wahrheitswert falsch |
| `nil` | Nullwert / Abwesenheit |

## 2.6 Operatoren auf einen Blick

| Kategorie | Operatoren | Beispiel |
|-----------|-----------|---------|
| Arithmetik | `+`, `-`, `*`, `/`, `%`, `**` | `a + b`, `2 ** 10` |
| Compound | `+=`, `-=`, `*=`, `/=`, `%=` | `x += 5` |
| Vergleich | `==`, `!=`, `<`, `>`, `<=`, `>=` | `a == b` |
| Logik | `!`, `&&`, `\|\|` | `!flag`, `a && b` |
| String | `++` | `"Hallo " ++ "Welt"` |
| Pipeline | `>`, `>>` | Vertikal mit Einrückung |
| Bereich | `..` | `list[0..3]` |
| Zuweisung | `:` | `name: wert` |

## 2.7 Typen auf einen Blick

| Typ | Literal | Beispiel |
|-----|---------|----------|
| `nil` | `nil` | Abwesenheit / kein Wert |
| `bool` | `true`, `false` | Wahrheitswerte |
| `num` | `42`, `3.14` | 64-Bit Integer oder Float |
| `str` | `"Hallo"` | Unicode-String |
| `list` | `[1, 2, 3]` | Dynamische, geordnete Liste |
| `map` | `{name: "Anna"}` | Assoziative Map |
| `fn` | `fn x` (body indented) | First-Class Function |

### Truthiness

Nur `false` und `nil` sind falsy. Alle anderen Werte (inkl. `0`, `""`, `[]`, `{}`) sind truthy.

## 2.8 Syntax-Cheatsheet

```pipe
-- Kommentar
-- Variable
x: 42
-- Compound Assignment
x: x + 1
-- Funktion
fn name a b
    -- Code
    "ergebnis"
-- Bedingung
if bed
    -- Code
    "wahr"
else
    -- Code
    "falsch"
-- Pattern Matching
match x
    | 0 -> 0
    | _ -> -1
-- Schleife
while bed
    -- Code
    print "loop"
-- for-in
for x in liste
    -- Code
    print x
-- Fehlerbehandlung
try
    -- Code
    10 / 0
catch e
    -- Code
    print e
-- Vorzeitiges Verlassen
return wert
-- Verzoegerte Ausfuehrung
defer ausdruck
-- Modul laden
import "datei.pipe"
-- Namespace-Import
import "lib" as ns
-- Export
export fn name
    "wert"
Rot: 0
Gruen: 1
-- Enumeration: 2
Blau

wert
    -- Vertikale Pipeline
    > funktion
    > ausgabe

-- Slicing
list[0..3]
-- Map
{key: wert}
-- Liste
[1, 2, 3]
-- Backtick-String
`mehrzeiliger text`
```

## 2.9 Projektstruktur (Empfehlung)

```
mein_projekt/
├── main.pipe
├── lib/
│   ├── helpers.pipe
│   └── config.pipe
└── README.md
```
