# 2. Sprachübersicht

## 2.1 Lexikalische Grundlagen

### Kommentare

Kommentare beginnen mit `--` und gelten bis zum Zeilenende:

```pipe
-- Das ist ein einzeiliger Kommentar
x: 42   -- Auch hinter Code möglich (Inline-Kommentar)
```

Es gibt keine mehrzeiligen Kommentare (Block-Kommentare).

### Einrückung

Pipe verwendet **Einrückung** zur Definition von Code-Blöcken — wie Python.
Empfohlen sind **4 Leerzeichen**, Tabs werden ebenfalls akzeptiert.

```pipe
if x > 10
    print "x ist groß"       -- Eingerückt = gehört zum if
    x: 0                     -- Eingerückt = gehört zum if
print "Fertig"               -- Nicht eingerückt = nach dem if
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
zaehler: zaehler + 1       -- Neuzuweisung: jetzt 1

name: "Pipe"
version: 1
pi: 3.14159
aktiv: true
nichts: nil
```

**Compound Assignment** (seit v0.5):

```pipe
x: 10
x += 5                      -- x = 15
x -= 3                      -- x = 12
x *= 2                      -- x = 24
x /= 4                      -- x = 6
x %= 4                      -- x = 2
```

**Gültigkeitsbereich:** Variablen sind im aktuellen Block sichtbar.
Funktionen sehen ihre eigenen Parameter und haben Zugriff auf den
umschließenden Scope (Closures).

## 2.3 Funktionsaufrufe

Funktionsaufrufe verwenden **Leerzeichen** statt Kommas:

```pipe
print "Hallo"               -- Ein Argument, ohne Klammern
print (1 + 2)               -- Rechenausdruck als Argument braucht Klammern
print (addiere 3 4)         -- Verschachtelter Aufruf
```

**Implicit Function Call (Space-based):** Wenn nach einem Ausdruck ein
Identifier folgt, wird dies als Funktionsaufruf interpretiert:

```pipe
print "Wert"                -- print("Wert")
fib 10 > print              -- print(fib(10))
```

**Wichtig:** Rechenausdrücke als Argumente brauchen Klammern, da sonst
der Operator Vorrang hat:

```pipe
print (1 + 2)               -- Richtig: print(3)
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
| `fn` | `fn x: x * 2` | First-Class Function |

### Truthiness

Nur `false` und `nil` sind falsy. Alle anderen Werte (inkl. `0`, `""`, `[]`, `{}`) sind truthy.

## 2.8 Syntax-Cheatsheet

```pipe
-- Kommentar
x: 42                         -- Variable
x += 1                        -- Compound Assignment
fn name a b: ...              -- Funktion
if bed: ... else: ...         -- Bedingung
match x | 0 -> ... | _ ->     -- Pattern Matching
while bed: ...                -- Schleife
for x in liste: ...           -- for-in
try: ... catch e: ...         -- Fehlerbehandlung
return wert                   -- Vorzeitiges Verlassen
defer ausdruck                -- Verzögerte Ausführung
import "datei.pipe"           -- Modul laden
import "lib" as ns            -- Namespace-Import
export fn name                -- Export
enum Farbe: Rot, Grün, Blau   -- Enumeration

wert
    > funktion                 -- Vertikale Pipeline
    > ausgabe

list[0..3]                    -- Slicing
{key: wert}                   -- Map
[1, 2, 3]                     -- Liste
`mehrzeiliger text`            -- Backtick-String
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
