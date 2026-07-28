# Pipe für Dummies

Eine lockere Einführung in die Skriptsprache Pipe —
von Null auf den ersten Erfolg in 30 Minuten.

---

## Vorwort: Was ist Pipe überhaupt?

Pipe ist eine **Skriptsprache**. Das heißt: Du schreibst Text-Dateien mit
Endung `.pipe`, und ein Programm namens `pipe` führt sie aus. So ähnlich
wie Python oder Lua — aber mit einer eigenen, einfachen Syntax.

Pipe wurde in Go programmiert und läuft auf **Linux, macOS, Windows**
und sogar dem Raspberry Pi.

---

## Kapitel 1: Installation (2 Minuten)

### Schritt 1: Bauen

```bash
cd pipe
make build
```

Das erzeugt die Datei `bin/pipe`. Mehr brauchst du nicht.

### Schritt 2: Testen

```bash
./bin/pipe examples/hello.pipe
```

Wenn `Hallo Welt` erscheint: **Alles funktioniert!** 🎉

---

## Kapitel 2: Deine erste Zeile Code (1 Minute)

Erstelle eine Datei `test.pipe`:

```pipe
print "Hallo Welt"
```

Ausführen:
```bash
./bin/pipe test.pipe
```

Ausgabe:
```
Hallo Welt
```

Glückwunsch! Du hast dein erstes Pipe-Programm geschrieben.

---

## Kapitel 3: Einfache Dinge tun (5 Minuten)

### Texte und Zahlen ausgeben

```pipe
print 42
print 3.14
print "Hallo"
print true
```

### Rechnen

```pipe
print (10 + 5)    -- Ergibt 15
print (20 - 7)    -- Ergibt 13
print (4 * 3)     -- Ergibt 12
print (15 / 3)    -- Ergibt 5
```

**Wichtig:** Rechnungen in Klammern schreiben, sonst versteht Pipe sie nicht:

```pipe
-- Richtig:
print (10 + 5)

-- Falsch:
print 10 + 5       -- Pipe denkt: "print 10, dann + 5 allein?"
```

### Texte zusammenkleben

```pipe
print ("Hallo " ++ "Welt")    -- Ergibt "Hallo Welt"
```

---

## Kapitel 4: Werte merken (Variablen) (3 Minuten)

```pipe
name: "Anna"
alter: 28
groesse: 1.75

print name          -- Anna
print alter         -- 28
print groesse       -- 1.75
```

**Merkregel:** `name: wert` — immer Doppelpunkt nach dem Namen, dann der Wert.

Du kannst den Wert später ändern:

```pipe
zaehler: 0
zaehler: zaehler + 1       -- jetzt 1
zaehler: zaehler + 1       -- jetzt 2
print zaehler               -- 2
```

---

## Kapitel 5: Entscheidungen treffen (if) (5 Minuten)

```pipe
punkte: 85

if punkte >= 90
    print "Sehr gut"
else if punkte >= 75
    print "Gut"
else if punkte >= 50
    print "Bestanden"
else
    print "Leider nicht bestanden"
```

**Wichtig:** Der eingerückte Teil (4 Leerzeichen) gehört zum `if`.
Du *musst* einrücken, sonst klappt es nicht.

---

## Kapitel 6: Wiederholungen (Schleifen) (5 Minuten)

### while: „Solange..."

```pipe
i: 0
while i < 5
    print i
    i: i + 1
```

Das gibt aus:
```
0
1
2
3
4
```

**So liest du es:** „Solange i kleiner als 5 ist, gib i aus und erhöhe i um 1."

### for-in: „Für jedes Element..."

```pipe
for farbe in ["rot", "grün", "blau"]
    print farbe
```

Das gibt jede Farbe einzeln aus.

### Vorzeitig aufhören (break)

```pipe
i: 0
while true
    print i
    i: i + 1
    if i >= 3
        break       -- Springt aus der Schleife
```

---

## Kapitel 7: Eigene Befehle (Funktionen) (10 Minuten)

### Eine einfache Funktion

```pipe
fn verdopple x
    x * 2

print (verdopple 5)     -- 10
print (verdopple 21)    -- 42
```

**So liest du es:**
- `fn` = „Definiere Funktion"
- `verdopple` = Name der Funktion
- `x` = Parameter (Eingabewert)
- `x * 2` = Körper (was die Funktion tut)

### Funktion mit mehreren Parametern

```pipe
fn addiere a b
    a + b

print (addiere 3 4)     -- 7
print (addiere 10 20)   -- 30
```

### Funktion, die Text zurückgibt

```pipe
fn begruessung name
    "Hallo " ++ name

print (begruessung "Welt"))     -- "Hallo Welt"
```

**Merkregel:** Der letzte Wert im Funktionskörper wird automatisch
zurückgegeben. Kein `return` nötig!

---

## Kapitel 8: Die Pipeline (5 Minuten)

Die Pipeline ist das Besondere an Pipe. Statt Funktionen ineinander
zu verschachteln, schreibst du sie untereinander:

```pipe
-- Statt: print(addiere(verdopple(10), 5))
-- Schreibe:

10
    > verdopple     -- verdopple(10) → 20
    > addiere 5     -- addiere(20, 5) → 25
    > print         -- gibt 25 aus
```

Das `>` bedeutet: „Nimm das Ergebnis von oben und gib es der nächsten
Funktion als erstes Argument."

**Ohne Pipeline wäre das:**
```pipe
print (addiere (verdopple 10) 5)
```

Schwer zu lesen, oder? Deshalb: **Pipeline!**

---

## Kapitel 9: Listen (10 Minuten)

Listen sind geordnete Sammlungen von Werten:

```pipe
zahlen: [10, 20, 30, 40, 50]
namen: ["Anna", "Bob", "Clara"]
gemischt: [1, "zwei", true]
```

### Auf Elemente zugreifen

```pipe
print (zahlen[0])       -- 10 (das erste Element)
print (zahlen[2])       -- 30 (das dritte Element)
```

### Einen Teil der Liste nehmen (Slicing)

```pipe
print (zahlen[1..3])    -- [20, 30] (Position 1 bis vor 3)
print (zahlen[0..2])    -- [10, 20]
```

### Nützliche Funktionen für Listen

```pipe
print (len zahlen)              -- 5 (wie viele Elemente?)
push zahlen 60                  -- Fügt 60 am Ende an
print (pop zahlen)              -- 60 (entfernt letztes Element)
print (at zahlen 1)             -- 20 (Element an Position 1)
print (sort [3, 1, 2])          -- [1, 2, 3]
print (range 5)                 -- [0, 1, 2, 3, 4]
```

### Durch eine Liste gehen

```pipe
for zahl in zahlen
    print zahl
```

---

## Kapitel 10: Maps (Schlüssel-Wert-Paare) (5 Minuten)

```pipe
person: {name: "Max", alter: 30, stadt: "Berlin"}

print (get person "name")       -- "Max"
print (get person "alter")      -- 30

-- Wert ändern
set person "alter" 31
print (get person "alter")      -- 31

-- Alle Schlüssel und Werte
print (keys person)             -- ["name", "alter", "stadt"]
print (values person)           -- ["Max", 31, "Berlin"]
```

---

## Kapitel 11: Fehler abfangen (try/catch) (5 Minuten)

Manchmal geht etwas schief. Statt dass das Programm abstürzt,
kannst du Fehler mit `try`/`catch` abfangen:

```pipe
try
    -- Hier steht gefährlicher Code
    ergebnis: 10 / 0
catch fehler
    -- Hier landest du, wenn etwas schiefging
    print "Ups, ein Fehler!"
    print fehler

print "Programm läuft weiter!"
```

---

## Kapitel 12: Dateien lesen und schreiben (5 Minuten)

```pipe
-- Datei schreiben
write_file "notiz.txt" "Hallo Welt"

-- Datei lesen
inhalt: read_file "notiz.txt"
print inhalt                    -- "Hallo Welt"

-- Prüfen ob Datei existiert
print (file_exists "notiz.txt")  -- true

-- Löschen
file_delete "notiz.txt"
```

---

## Kapitel 13: Das Internet anzapfen (HTTP) (5 Minuten)

```pipe
-- Eine Webseite abrufen
antwort: http_get "https://httpbin.org/get"
print (get antwort "status")        -- 200 (OK)

-- JSON von einer API lesen
daten: http_get_json "https://api.github.com/users/torvalds"
print (get daten "name")            -- "Linus Torvalds"
```

---

## Kapitel 14: Häufige Fehler (und wie du sie vermeidest)

### Fehler 1: Einrückung vergessen

```pipe
-- Falsch:
if x > 10
print "x ist groß"    -- Nicht eingerückt!

-- Richtig:
if x > 10
    print "x ist groß"
```

### Fehler 2: Klammern bei Rechnungen vergessen

```pipe
-- Falsch:
print 1 + 2

-- Richtig:
print (1 + 2)
```

### Fehler 3: `=` statt `==` beim Vergleichen

```pipe
-- Falsch:
if x = 10

-- Richtig:
if x == 10
```

### Fehler 4: Pipeline `>` mit Vergleich `>` verwechselt

```pipe
-- Das hier ist ein VERGLEICH, keine Pipeline:
if x > 10           -- „x ist größer als 10?"

-- Pipeline nur in vertikaler Form mit Einrückung:
42
    > verdopple     -- Pipeline: verdopple(42)
    > print
```

---

## Kapitel 15: Die REPL — Dein Spielplatz (5 Minuten)

Starte `./bin/pipe` ohne Dateinamen. Du landest in der REPL
(Read-Eval-Print-Loop). Hier kannst du Code Zeile für Zeile
ausprobieren:

```
>>> 1 + 2
3
>>> name: "Pipe"
>>> print name
Pipe
>>> fn quadrat x
...     x * x
...
>>> print (quadrat 5)
25
>>> :quit
```

**Tipps:**
- `>>>` bedeutet: Gib eine neue Zeile ein
- `...` bedeutet: Du bist in einem mehrzeiligen Block
- Leerzeile = Block abschließen und ausführen
- `:quit` oder Strg+D zum Beenden

---

## Kapitel 16: Wie geht's weiter?

### Alle 18 Beispielprogramme ausprobieren

```bash
for f in examples/*.pipe; do
    echo "=== $f ==="
    ./bin/pipe "$f"
done
```

### Das vollständige Handbuch lesen

Die Datei `GUIDE.md` enthält das **komplette Handbuch** mit allen
Built-in-Funktionen, fortgeschrittenen Themen und Kochrezepten.

### Eigene Projekte starten

```bash
mkdir mein_projekt
cd mein_projekt
echo 'print "Mein erstes Projekt"' > main.pipe
../bin/pipe main.pipe
```

---

## Cheatsheet (Spickzettel)

```
-- Kommentar
name: wert                   -- Variable
fn name arg: körper          -- Funktion
if bed: ... else: ...        -- Bedingung
match x | 0 -> ... | _ ->    -- Pattern Matching
while bed: ...               -- Schleife
for x in liste: ...          -- for-in-Schleife
try: ... catch e: ...        -- Fehler abfangen
import "datei.pipe"         -- Andere Datei laden

wert > f > g > print         -- Vertikale Pipeline
list[0..3]                   -- Teil-Liste

-- Nützliche Builtins:
print x                      -- Ausgeben
len list                     -- Länge
range 5                      -- [0,1,2,3,4]
get map "key"                -- Wert aus Map lesen
push list x                  -- Element anhängen
read_file "pfad"             -- Datei lesen
write_file "pfad" "inhalt"   -- Datei schreiben
http_get "url"               -- HTTP-Anfrage
parse_json "..."             -- JSON parsen
```

---

**Und jetzt: Viel Spaß beim Programmieren mit Pipe!**
