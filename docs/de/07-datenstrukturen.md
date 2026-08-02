# 7. Datenstrukturen

## 7.1 Listen

Listen sind **geordnete, dynamische Sammlungen** von Werten. Sie können
Elemente beliebigen Typs enthalten.

### Erstellung

```pipe
leer: []
zahlen: [10, 20, 30, 40, 50]
worte: ["rot", "grün", "blau"]
gemischt: [1, "zwei", true, nil, [1, 2, 3]]
```

### Index-Zugriff

Der Zugriff erfolgt über `liste[index]` — 0-basiert:

```pipe
-- 10 (erstes Element)
print (zahlen[0])
-- 30 (drittes Element)
print (zahlen[2])
-- 50 (letztes Element)
print (zahlen[4])
```

### Slicing (Teillisten)

```pipe
list: [10, 20, 30, 40, 50]

-- [10, 20, 30]       (Index 0 bis vor 3)
print (list[0..3])
-- [30, 40]            (Index 2 bis vor 4)
print (list[2..4])
-- [20, 30, 40, 50]    (Index 1 bis vor 5)
print (list[1..5])
-- [10]                (nur erstes Element)
print (list[0..1])
```

Die Syntax `start..ende` erzeugt eine Teilliste von `start` (inklusive) bis `ende` (exklusive).

### Einfacher Index

```pipe
-- 40 (einzelnes Element, kein Slice)
print (zahlen[3])
```

### Listen-Operationen

| Funktion | Beschreibung | Beispiel |
|----------|-------------|---------|
| `len list` | Anzahl Elemente | `len [1,2,3]` → `3` |
| `push list x...` | Element(e) anhängen | `push [1,2] 3` → `[1,2,3]` |
| `pop list` | Letztes Element entfernen und zurückgeben | `pop [1,2,3]` → `3`, Liste wird `[1,2]` |
| `at list i` | Element an Position i | `at [10,20,30] 1` → `20` |
| `sort list` | Liste sortieren | `sort [3,1,2]` → `[1,2,3]` |
| `range n` | Zahlenbereich 0..n-1 | `range 5` → `[0,1,2,3,4]` |
| `range a b` | Bereich a..b-1 | `range 2 5` → `[2,3,4]` |
| `range a b s` | Bereich mit Schrittweite | `range 0 10 2` → `[0,2,4,6,8]` |

### Höhere Funktionen (Tree-Walker only)

| Funktion | Beschreibung |
|----------|-------------|
| `map list fn` | Transformiert jedes Element |
| `filter list fn` | Behält Elemente, für die fn true ergibt |
| `reduce list fn init` | Faltet die Liste auf einen Wert |
| `each list fn` | Führt fn für jedes Element aus |

```pipe
fn verdopple x
    x * 2

fn ist_gerade x
    x % 2 == 0

fn summe a b
    a + b

-- [2, 4, 6]
print (map ([1, 2, 3]) verdopple)
-- [2, 4]
print (filter ([1, 2, 3, 4]) ist_gerade)
-- 10
print (reduce ([1, 2, 3, 4]) summe 0)
```

**Wichtig:** `map`, `filter`, `reduce`, `each` mit benutzerdefinierten Funktionen
funktionieren nur im Tree-Walker, nicht in der Bytecode-VM.

## 7.2 Maps

Maps sind **assoziative Sammlungen** von Schlüssel-Wert-Paaren. Keys sind Strings.

### Erstellung

```pipe
leer: {}
person: {name: "Anna", alter: 28, stadt: "Berlin"}
```

### Zugriff

```pipe
-- "Anna"
print (get person "name")
-- 28
print (get person "alter")
```

### Ändern

```pipe
set person "alter" 29
-- 29
print (get person "alter")
```

### Schlüssel und Werte

```pipe
-- ["name", "alter", "stadt"]
print (keys person)
-- ["Anna", 29, "Berlin"]
print (values person)
```

### Map-Operationen

| Funktion | Beschreibung | Beispiel |
|----------|-------------|---------|
| `get map key` | Wert abrufen (nil wenn nicht vorhanden) | `get {a:1} "a"` → `1` |
| `set map key val` | Wert setzen | `set m "a" 2` |
| `keys map` | Alle Schlüssel als Liste | `keys {a:1,b:2}` → `["a","b"]` |
| `values map` | Alle Werte als Liste | `values {a:1,b:2}` → `[1,2]` |

### Dot-Access

Pipe unterstützt auch Punkt-Notation für Map-Zugriffe:

```pipe
person: {name: "Anna", alter: 28}

-- Beide Varianten sind äquivalent:
-- "Anna"
print (get person "name")
-- "Anna"
print person.name
```

## 7.3 Strings als Index-Zugriff

Mit `at` kann man auf einzelne Zeichen eines Strings zugreifen:

```pipe
text: "Hallo"
-- "H"
print (at text 0)
-- "a"
print (at text 1)
-- "o"
print (at text 4)
```

## 7.4 contains für Strings und Listen

```pipe
-- true
print (contains "Hallo Welt" "Welt")
-- false
print (contains "Hallo Welt" "Mars")
-- true
print (contains ([1, 2, 3]) 2)
-- false
print (contains ([1, 2, 3]) 5)
```

## 7.5 Praxis-Beispiele

### Summe einer Liste

```pipe
fn summe liste
    total: 0
    for n in liste
        total: total + n
    total

-- 15
print (summe ([1, 2, 3, 4, 5]))
```

### Elementweise Verarbeitung

```pipe
fn verdopple_alle liste
    for n in liste
        n
            > (fn x
                x * 2)
            > print

verdopple_alle ([1, 2, 3])
-- Ausgabe: 2, 4, 6
```

### JSON-Daten parsen

```pipe
antwort: parse_json "{\"name\": \"Pipe\", \"version\": 1}"
-- "Pipe"
print (get antwort "name")
-- 1
print (get antwort "version")
```
