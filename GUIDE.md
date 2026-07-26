# Pulse — Das Handbuch

Eine vollständige Einführung in die Programmiersprache Pulse.
Von den ersten Schritten bis zu fortgeschrittenen Konzepten.

---

## Inhaltsverzeichnis

1. [Installation und erster Start](#1-installation-und-erster-start)
2. [Hallo Welt](#2-hallo-welt)
3. [Werte und Datentypen](#3-werte-und-datentypen)
4. [Variablen](#4-variablen)
5. [Operatoren](#5-operatoren)
6. [Bedingungen (if/else)](#6-bedingungen-ifelse)
7. [Pattern Matching (match)](#7-pattern-matching-match)
8. [Schleifen (while, for-in)](#8-schleifen-while-for-in)
9. [Funktionen](#9-funktionen)
10. [Pipelines](#10-pipelines)
11. [Listen und Maps](#11-listen-und-maps)
12. [Fehlerbehandlung (try/catch)](#12-fehlerbehandlung-trycatch)
13. [Module und Importe](#13-module-und-importe)
14. [Alle Built-in-Funktionen](#14-alle-built-in-funktionen)
15. [Dateisystem](#15-dateisystem)
16. [Netzwerk & HTTP](#16-netzwerk--http)
17. [Regex, Datum, Zufall](#17-regex-datum-zufall)
18. [Die Pulse REPL](#18-die-pulse-repl)
19. [Der VM-Modus](#19-der-vm-modus)
20. [Kochrezepte](#20-kochrezepte)
21. [Schnellreferenz](#21-schnellreferenz)

---

## 1. Installation und erster Start

### Voraussetzungen

Pulse benötigt **Go 1.21+**. Keine weiteren Abhängigkeiten.

```bash
# Repository klonen und bauen
git clone <pulse-repo>
cd pulse
make build

# Ausführen
./bin/pulse examples/hello.pulse
```

### Erster Test

```bash
# Eine Datei ausführen
./bin/pulse datei.pulse

# REPL starten (interaktiv)
./bin/pulse

# Mit Bytecode-VM (schneller)
./bin/pulse -vm datei.pulse

# AST anzeigen (für Entwickler)
./bin/pulse -ast datei.pulse
```

---

## 2. Hallo Welt

```pulse
-- Dein erstes Pulse-Programm
print "Hallo Welt"
```

Ausführen:
```bash
./bin/pulse hello.pulse
```

Ausgabe:
```
Hallo Welt
```

**Kommentare** beginnen mit `--` und gelten bis zum Zeilenende.

---

## 3. Werte und Datentypen

Pulse hat 7 eingebaute Datentypen:

| Typ | Beispiel | Beschreibung |
|-----|----------|-------------|
| `nil` | `nil` | Nichts / kein Wert |
| `bool` | `true`, `false` | Wahrheitswerte |
| `num` | `42`, `3.14` | Zahlen (Ganzzahl oder Kommazahl) |
| `str` | `"Hallo"` | Text / Zeichenkette |
| `list` | `[1, 2, 3]` | Geordnete Liste |
| `map` | `{name: "Anna"}` | Schlüssel-Wert-Paare |
| `fn` | `fn x: x * 2` | Funktionen |

```pulse
-- Literale direkt ausprobieren
print 42          -- Zahl
print 3.14        -- Kommazahl
print "Hallo"     -- String
print true        -- Wahr
print false       -- Falsch
print nil         -- Nichts
print [1, 2, 3]   -- Liste
print {a: 1}      -- Map
```

---

## 4. Variablen

Variablen werden mit `name: wert` definiert:

```pulse
name: "Pulse"
version: 1
pi: 3.14159
aktiv: true

print name       -- Pulse
print version    -- 1
```

**Neuzuweisung** verwendet die gleiche Syntax:

```pulse
zaehler: 0
zaehler: zaehler + 1    -- jetzt 1
zaehler: zaehler + 1    -- jetzt 2
print zaehler            -- 2
```

Variablen sind **sichtbar im aktuellen Block** (geschweifte Klammern gibt es nicht —
Einrückung bestimmt den Gültigkeitsbereich).

---

## 5. Operatoren

### Arithmetik

```pulse
print (10 + 3)    -- 13
print (10 - 3)    -- 7
print (10 * 3)    -- 30
print (10 / 3)    -- 3
print (10 % 3)    -- 1 (Rest)
```

### Vergleiche

```pulse
print (10 == 10)   -- true
print (10 != 5)    -- true
print (5 < 10)     -- true
print (10 > 5)     -- true
print (5 <= 5)     -- true
print (10 >= 5)    -- true
```

### Strings

```pulse
print ("Hallo " ++ "Welt")    -- "Hallo Welt"
print ("abc" == "abc")        -- true
```

### Logische Operatoren

Pulse hat keine `&&`/`||`-Operatoren. Stattdessen verwendet man
verschachtelte `if`-Ausdrücke oder Hilfsfunktionen.

---

## 6. Bedingungen (if/else)

```pulse
temperatur: 25

if temperatur > 30
    print "Heiß!"
else if temperatur > 20
    print "Angenehm"
else if temperatur > 10
    print "Kühl"
else
    print "Kalt!"
```

**Wichtig:** Der Körper eines `if` wird eingerückt (4 Leerzeichen oder Tab).
`else if` und `else` stehen auf der gleichen Ebene wie das `if`.

`if` ist ein **Ausdruck** — er gibt einen Wert zurück:

```pulse
status: if temperatur > 30
    "heiß"
else
    "ok"
print status
```

---

## 7. Pattern Matching (match)

`match` vergleicht einen Wert mit mehreren Mustern. Das erste passende Muster gewinnt:

```pulse
fn bewertung note
    match note
        | 1      -> "Sehr gut"
        | 2      -> "Gut"
        | 3      -> "Befriedigend"
        | 4      -> "Ausreichend"
        | 5      -> "Mangelhaft"
        | _      -> "Ungültig"

print (bewertung 2)    -- "Gut"
print (bewertung 99)   -- "Ungültig"
```

- `|` leitet ein Muster ein
- `->` trennt Muster vom Ergebnis
- `_` ist der Wildcard („alles andere")

```pulse
-- Fibonacci mit match
fn fib n
    match n
        | 0  -> 0
        | 1  -> 1
        | _  -> fib(n - 1) + fib(n - 2)

print (fib 10)    -- 55
```

---

## 8. Schleifen (while, for-in)

### while-Schleife

```pulse
i: 0
while i < 5
    print i
    i: i + 1
-- Ausgabe: 0 1 2 3 4
```

**break** bricht die Schleife ab:

```pulse
i: 0
while true
    print i
    i: i + 1
    if i >= 3
        break
-- Ausgabe: 0 1 2
```

**continue** springt zum nächsten Durchlauf:

```pulse
i: 0
while i < 6
    i: i + 1
    if i % 2 == 0
        continue
    print i
-- Ausgabe: 1 3 5
```

### for-in-Schleife

```pulse
-- Über eine Liste iterieren
for farbe in ["rot", "grün", "blau"]
    print farbe

-- Mit range (Zahlenbereich)
for n in (range 1 5)
    print n
-- Ausgabe: 1 2 3 4
```

`range` erzeugt eine Liste von Zahlen:
- `range 5` → `[0, 1, 2, 3, 4]`
- `range 2 6` → `[2, 3, 4, 5]`
- `range 0 10 2` → `[0, 2, 4, 6, 8]`

---

## 9. Funktionen

### Definition

```pulse
fn verdopple x
    x * 2

fn addiere a b
    a + b

fn begrüße name
    "Hallo " ++ name
```

- `fn` leitet die Definition ein
- Dann der **Name**, dann die **Parameter** (durch Leerzeichen getrennt)
- Der **Körper** wird eingerückt
- Der **letzte Ausdruck** im Körper ist der Rückgabewert
- Kein `return`-Keyword nötig!

### Aufruf

```pulse
print (verdopple 21)       -- 42
print (addiere 3 4)        -- 7
print (begrüße "Welt")     -- "Hallo Welt"
```

Funktionsaufrufe verwenden Leerzeichen statt Kommas:
`funktion arg1 arg2 arg3`

### Anonyme Funktionen

```pulse
-- Funktion in Variable speichern
verdreifacher: fn x
    x * 3

print (verdreifacher 7)    -- 21
```

### Rekursion

```pulse
fn fakultät n
    if n <= 1
        1
    else
        n * (fakultät (n - 1))

print (fakultät 5)    -- 120
```

---

## 10. Pipelines

Die Pipeline (`>`) ist Pulse's Markenzeichen. Sie leitet einen Wert
durch eine Kette von Funktionen — von links nach rechts.

### Horizontale Pipeline

```pulse
-- Statt: print(verdopple(addiere(10, 5)))
-- Schreibt man:

-- Achtung: > wird als Vergleich interpretiert,
-- daher Klammern für Pipeline-Werte verwenden:
print (42 > verdopple)    -- Pipeline
```

Die vertikale Pipeline ist die empfohlene Schreibweise:

### Vertikale Pipeline

```pulse
42
    > verdopple
    > addiere 10
    > print

-- Bedeutet: print(addiere(verdopple(42), 10))
-- Ausgabe: 94
```

Jede eingerückte Zeile mit `>` wendet eine Funktion auf das
vorherige Ergebnis an. Das Pipeline-Ergebnis wird **als erstes Argument**
an die nächste Funktion übergeben.

### Pipeline mit mehreren Argumenten

```pulse
100
    > addiere 50    -- addiere(100, 50) = 150
    > verdopple     -- verdopple(150) = 300
    > print         -- 300
```

---

## 11. Listen und Maps

### Listen

```pulse
-- Erstellen
zahlen: [10, 20, 30, 40, 50]
leer: []

-- Zugriff per Index
print (zahlen[0])       -- 10 (erstes Element)
print (zahlen[2])       -- 30

-- Slicing
print (zahlen[0..3])    -- [10, 20, 30]
print (zahlen[2..4])    -- [30, 40]

-- Operationen
print (len zahlen)      -- 5
push zahlen 60          -- [10, 20, 30, 40, 50, 60]
print (pop zahlen)      -- 60 (entfernt letztes)
print (at zahlen 1)     -- 20
```

### Maps

```pulse
-- Erstellen
person: {name: "Anna", alter: 28, stadt: "Berlin"}

-- Zugriff
print (get person "name")     -- "Anna"
print (get person "alter")    -- 28

-- Ändern
set person "alter" 29
print (get person "alter")    -- 29

-- Schlüssel und Werte
print (keys person)     -- ["name", "alter", "stadt"]
print (values person)   -- ["Anna", 29, "Berlin"]
```

### Höhere Funktionen (map, filter, reduce)

Nur im Tree-Walker-Modus (nicht `-vm`) mit benutzerdefinierten Funktionen:

```pulse
fn verdopple x
    x * 2

fn ist_gerade x
    x % 2 == 0

fn summe a b
    a + b

-- map: transformiert jedes Element
print (map [1, 2, 3] verdopple)        -- [2, 4, 6]

-- filter: behält Elemente, für die die Funktion true ergibt
print (filter [1, 2, 3, 4, 5] ist_gerade)   -- [2, 4]

-- reduce: faltet die Liste auf einen Wert
print (reduce [1, 2, 3, 4] summe 0)    -- 10
```

---

## 12. Fehlerbehandlung (try/catch)

```pulse
fn teile a b
    if b == 0
        1 / 0     -- Erzwingt Fehler
    else
        a / b

-- Fehler abfangen
try
    print (teile 10 0)
catch fehler
    print "Fehler aufgetreten:"
    print fehler

print "Programm läuft weiter!"
```

Bei einem Fehler im `try`-Block springt die Ausführung in den `catch`-Block.
Die Fehlervariable (hier `fehler`) enthält eine Beschreibung des Fehlers.

**Stack-Traces** werden automatisch hinzugefügt:

```
ERROR: Division durch Null
  in fn(teile)
  in fn(berechne)
```

---

## 13. Module und Importe

Mit `import` kannst du Code aus anderen Dateien laden:

**math.pulse:**
```pulse
fn quadrat x
    x * x

fn kubik x
    x * (quadrat x)
```

**main.pulse:**
```pulse
import "math.pulse"

print (quadrat 7)    -- 49
print (kubik 3)      -- 27
```

Der Pfad ist relativ zur aufrufenden Datei.

---

## 14. Alle Built-in-Funktionen

### Ein-/Ausgabe

| Funktion | Beschreibung | Beispiel |
|----------|-------------|---------|
| `print x...` | Gibt Werte aus | `print "Hallo" 42` |
| `read_file p` | Datei als String lesen | `read_file "text.txt"` |
| `write_file p t` | Text in Datei schreiben | `write_file "x.txt" "Hi"` |
| `append_file p t` | Text anhängen | `append_file "log.txt" "Zeile"` |
| `read_lines p` | Datei als Liste von Zeilen | `read_lines "text.txt"` |

### String-Operationen

| Funktion | Beschreibung | Beispiel |
|----------|-------------|---------|
| `upper s` | Großbuchstaben | `upper "hallo"` → `"HALLO"` |
| `lower s` | Kleinbuchstaben | `lower "ALLO"` → `"allo"` |
| `trim s` | Leerzeichen entfernen | `trim " hi "` → `"hi"` |
| `split s t` | An Trennzeichen teilen | `split "a,b" ","` → `["a","b"]` |
| `join list t` | Mit Trennzeichen verbinden | `join ["x","y"] "-"` → `"x-y"` |
| `contains s sub` | Enthält? | `contains "abc" "b"` → `true` |

### Listen-Operationen

| Funktion | Beschreibung | Beispiel |
|----------|-------------|---------|
| `len list` | Länge | `len [1,2,3]` → `3` |
| `push list x` | Element anhängen | `push [1,2] 3` → `[1,2,3]` |
| `pop list` | Letztes entfernen | `pop [1,2,3]` → `3` |
| `at list i` | Element an Position i | `at [10,20,30] 1` → `20` |
| `sort list` | Sortieren | `sort [3,1,2]` → `[1,2,3]` |
| `range n` | Zahlenbereich | `range 5` → `[0,1,2,3,4]` |
| `range a b` | Bereich von a bis b-1 | `range 2 5` → `[2,3,4]` |
| `range a b s` | Mit Schrittweite | `range 0 10 3` → `[0,3,6,9]` |

### Map-Operationen

| Funktion | Beschreibung | Beispiel |
|----------|-------------|---------|
| `get map key` | Wert abrufen | `get {a:1} "a"` → `1` |
| `set map key v` | Wert setzen | `set m "a" 2` |
| `keys map` | Alle Schlüssel | `keys {a:1,b:2}` → `["a","b"]` |
| `values map` | Alle Werte | `values {a:1,b:2}` → `[1,2]` |

### Mathematik

| Funktion | Beschreibung | Beispiel |
|----------|-------------|---------|
| `abs n` | Absolutwert | `abs (-5)` → `5` |
| `min a b...` | Minimum | `min 3 1 5` → `1` |
| `max a b...` | Maximum | `max 3 1 5` → `5` |
| `pow b e` | Potenz | `pow 2 3` → `8` |
| `sqrt n` | Quadratwurzel | `sqrt 16` → `4` |
| `round n` | Runden | `round 3.7` → `4` |

### Typ-Prüfung

| Funktion | Beschreibung | Beispiel |
|----------|-------------|---------|
| `is_num x` | Ist eine Zahl? | `is_num 42` → `true` |
| `is_str x` | Ist ein String? | `is_str "hi"` → `true` |
| `is_list x` | Ist eine Liste? | `is_list [1]` → `true` |
| `is_map x` | Ist eine Map? | `is_map {a:1}` → `true` |
| `is_nil x` | Ist nil? | `is_nil nil` → `true` |

### Konvertierung

| Funktion | Beschreibung | Beispiel |
|----------|-------------|---------|
| `to_str x` | In String umwandeln | `to_str 42` → `"42"` |
| `to_num x` | In Zahl umwandeln | `to_num "42"` → `42` |

---

## 15. Dateisystem

```pulse
-- Existiert eine Datei?
print (file_exists "config.txt")     -- true/false

-- Dateigröße
print (file_size "config.txt")       -- 1234 (Bytes)

-- Ist es eine Datei oder ein Verzeichnis?
print (file_type "config.txt")       -- "file"
print (file_type "/tmp")             -- "dir"

-- Verzeichnis auflisten
print (list_dir ".")                 -- ["main.pulse", "lib.pulse", ...]
print (list_dir "/tmp")              -- Inhalt von /tmp

-- Datei kopieren / verschieben / löschen
file_copy "a.txt" "b.txt"            -- Kopie erstellen
file_move "b.txt" "c.txt"            -- Umbenennen
file_delete "c.txt"                  -- Löschen

-- Verzeichnisse
make_dir "/tmp/mein_ordner"           -- Anlegen
remove_dir "/tmp/mein_ordner"         -- Löschen (rekursiv)

-- Pfad-Operationen
print (path_join "/home" "user")     -- "/home/user"
print (path_base "/a/b/c.txt")       -- "c.txt"
print (path_dir "/a/b/c.txt")        -- "/a/b"
print (path_ext "/a/b/c.txt")        -- ".txt"
```

---

## 16. Netzwerk & HTTP

### HTTP GET

```pulse
-- Einfacher GET-Request
antwort: http_get "https://httpbin.org/get"
print (get antwort "status")         -- 200
print (get antwort "body")           -- Antwort-Body

-- JSON-API aufrufen
daten: http_get_json "https://api.github.com/users/torvalds"
print (get daten "name")             -- "Linus Torvalds"
print (get daten "public_repos")     -- 12
```

### HTTP POST

```pulse
payload: {name: "Pulse", typ: "sprache"}
json: to_json payload
antwort: http_post "https://httpbin.org/post" json
print (get antwort "status")         -- 200
```

### JSON

```pulse
-- JSON parsen
daten: parse_json "{\"name\": \"Pulse\"}"
print (get daten "name")             -- "Pulse"

-- Zu JSON konvertieren
print (to_json {a: 1, b: 2})        -- {"a":1,"b":2}
```

### TCP (Server + Client)

```pulse
-- Einfacher Echo-Server
fn starte_server
    listener: tcp_listen "0.0.0.0" 9999
    verbindung: tcp_accept listener
    nachricht: tcp_read verbindung
    tcp_write verbindung ("ECHO: " ++ nachricht)
    tcp_close verbindung
    tcp_close listener

-- Client
fn sende_nachricht text
    conn: tcp_connect "127.0.0.1" 9999
    tcp_write conn text
    antwort: tcp_read conn
    tcp_close conn
    print antwort
```

---

## 17. Regex, Datum, Zufall

### Regex

```pulse
-- Prüfen ob Muster passt
print (regex_match "[0-9]+" "abc123")       -- true
print (regex_match "^\\d{3}$" "123")        -- true

-- Ersetzen
print (regex_replace "[0-9]" "#" "Tel: 0123"))  -- "Tel: ####"
```

### Datum und Zeit

```pulse
-- Aktueller Unix-Timestamp
ts: now 0
print ts                                     -- 1785100000

-- Formatiert (Go-Zeitformat: 2006-01-02 15:04:05)
print (format_time ts "2006-01-02")          -- "2026-07-27"
print (format_time ts "15:04:05")            -- "14:30:00"
```

### Zufall

```pulse
-- Zufallszahl zwischen 0 und 1
print (random 0)                             -- 0.370874...

-- Ganzzahl im Bereich [min, max)
print (random_range 1 7)                     -- 4 (Würfel)
print (random_range 1 101)                   -- 42
```

---

## 18. Die Pulse REPL

Starte die REPL mit `./bin/pulse` (ohne Dateiname):

```
Pulse v0.4.0 — REPL
Gib Pulse-Code ein. :quit oder Strg+D zum Beenden.
Leerzeile zum Abschließen von mehrzeiligen Blöcken.

>>> 1 + 2
3
>>> print "Hallo"
Hallo
>>> fn verdopple x
...     x * 2
...
>>> print (verdopple 21)
42
>>> :vm
  VM-Modus: ein
>>> :quit
```

### REPL-Befehle

| Befehl | Wirkung |
|--------|---------|
| `:quit`, `:q` | Beenden |
| `:help`, `:h` | Hilfe anzeigen |
| `:clear`, `:c` | Eingabe zurücksetzen |
| `:vm` | VM-Modus umschalten |
| `Strg+D` | Beenden |
| Leerzeile | Mehrzeiligen Block abschließen |

**Tipp:** Nach `fn`, `if`, `while`, `match`, `for`, `try` automatisch
mehrzeilig. Mit einer Leerzeile abschließen.

---

## 19. Der VM-Modus

Pulse hat zwei Ausführungsmodi:

| Modus | Befehl | Geschwindigkeit |
|-------|--------|----------------|
| **Tree-Walker** | `./bin/pulse datei.pulse` | Langsamer, alle Features |
| **Bytecode-VM** | `./bin/pulse -vm datei.pulse` | ~3-7× schneller |

```bash
# VM-Modus (schnell)
./bin/pulse -vm fibonacci.pulse

# VM ohne Bytecode-Ausgabe
./bin/pulse -vm -q fibonacci.pulse
```

Die VM kompiliert den Code zu Bytecode (32 Opcodes) und führt ihn
auf einer Stack-Maschine aus. Die meisten Features sind in beiden
Modi verfügbar.

**Nur im Tree-Walker:**
- `map`, `filter`, `reduce`, `each` mit benutzerdefinierten Funktionen
- `for-in`-Schleifen
- `try`/`catch`

---

## 20. Kochrezepte

### Fibonacci-Zahlen

```pulse
fn fib n
    match n
        | 0  -> 0
        | 1  -> 1
        | _  -> fib(n - 1) + fib(n - 2)

print (fib 10)    -- 55
```

### FizzBuzz

```pulse
fn fizzbuzz n
    if n % 15 == 0
        "FizzBuzz"
    else if n % 3 == 0
        "Fizz"
    else if n % 5 == 0
        "Buzz"
    else
        n

for n in (range 1 16)
    print (fizzbuzz n)
```

### Primzahlen finden

```pulse
fn ist_prim n
    ist_prim_hilfe n 2

fn ist_prim_hilfe n d
    if d * d > n
        true
    else if n % d == 0
        false
    else
        ist_prim_hilfe n (d + 1)

-- Alle Primzahlen bis 30
for n in (range 2 31)
    if ist_prim n
        print n
```

### Palindrom-Prüfer

```pulse
fn reverse s
    reverse_hilfe s (len s - 1)

fn reverse_hilfe s i
    if i < 0
        ""
    else
        (at s i) ++ (reverse_hilfe s (i - 1))

fn ist_palindrom s
    s == (reverse s)

print (ist_palindrom "racecar")    -- true
print (ist_palindrom "hello")      -- false
```

### Datei-Zeilen zählen

```pulse
fn zähle_zeilen pfad
    zeilen: read_lines pfad
    print "Datei: "
    print pfad
    print "Zeilen: "
    print (len zeilen)

zähle_zeilen "main.pulse"
```

### Konfigurationsdatei parsen

```pulse
-- Liest key=value-Paare aus einer Datei
fn parse_config pfad
    zeilen: read_lines pfad
    for zeile in zeilen
        if contains zeile "="
            teile: split zeile "="
            key: at teile 0
            value: at teile 1
            print (key ++ " = " ++ value)

parse_config "config.ini"
```

### Einfacher Webservice-Check

```pulse
fn check_url url
    try
        antwort: http_get url
        status: get antwort "status"
        print (url ++ " -> " ++ (to_str status))
    catch e
        print (url ++ " -> FEHLER")

check_url "https://google.com"
check_url "https://github.com"
```

### Summe einer Liste

```pulse
fn summe liste
    total: 0
    for n in liste
        total: total + n
    total

print (summe [1, 2, 3, 4, 5])    -- 15
```

### Caesar-Verschlüsselung

```pulse
fn caesar text verschiebung
    alpha: "abcdefghijklmnopqrstuvwxyz"
    ergebnis: ""
    i: 0
    while i < (len text)
        c: at text i
        pos: finde_pos alpha c 0
        neu_pos: (pos + verschiebung) % 26
        ergebnis: ergebnis ++ (at alpha neu_pos)
        i: i + 1
    ergebnis

fn finde_pos alpha c pos
    if pos >= (len alpha)
        pos
    else if (at alpha pos) == c
        pos
    else
        finde_pos alpha c (pos + 1)

print (caesar "hallo" 3)    -- "kdoor"
```

---

## 21. Schnellreferenz

### Syntax auf einen Blick

```pulse
-- Kommentar
x: 42                         -- Variable
fn name a b: ...              -- Funktion
if bedingung: ...             -- Bedingung
match wert | 0 -> ... | _ ->  -- Pattern Matching
while bedingung: ...          -- while-Schleife
for x in liste: ...           -- for-in-Schleife
try: ... catch e: ...         -- Fehlerbehandlung
import "datei.pulse"          -- Modul laden

wert > funktion > ausgabe     -- Pipeline
list[0..3]                    -- Slicing
{key: wert}                   -- Map
[1, 2, 3]                     -- Liste
```

### Operatoren

```
+ - * / %      Arithmetik
== != < > <= >=  Vergleiche
++             String-Verkettung
>              Pipeline (vertikal mit Einrückung)
..             Bereich (in list[0..3])
```

### Datei-Endung

Pulse-Dateien haben die Endung **`.pulse`**.

### Projektstruktur

```
mein_projekt/
├── main.pulse          -- Hauptprogramm
├── lib.pulse           -- Hilfsfunktionen
└── daten/              -- Daten-Verzeichnis
```

---

**Viel Spaß mit Pulse!**
