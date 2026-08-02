# 4. Kontrollfluss

## 4.1 if / else if / else

`if` ist ein **Ausdruck** — er gibt immer einen Wert zurück.

```pipe
punkte: 85

if punkte >= 90
    print "Sehr gut"
else if punkte >= 75
    print "Gut"
else if punkte >= 50
    print "Bestanden"
else
    print "Nicht bestanden"
```

### if als Ausdruck

Der letzte Wert jedes Zweigs wird zurückgegeben:

```pipe
status: if punkte >= 50
    "bestanden"
else
    "durchgefallen"

print status    -- "bestanden"
```

### Verschachtelte if-Ausdrücke

```pipe
kategorie: if alter < 13
    "Kind"
else if alter < 18
    "Jugendlich"
else if alter < 65
    "Erwachsen"
else
    "Senior"
```

## 4.2 match — Pattern Matching

`match` vergleicht einen Wert mit mehreren Mustern. Das **erste passende Muster** gewinnt.

```pipe
fn bewertung note
    match note
        | 1      -> "Sehr gut"
        | 2      -> "Gut"
        | 3      -> "Befriedigend"
        | 4      -> "Ausreichend"
        | 5      -> "Mangelhaft"
        | _      -> "Ungültig"
```

- `|` leitet ein Muster ein
- `->` trennt Muster vom Ergebnis
- `_` ist der **Wildcard** („trifft immer zu")

### Klassisches Beispiel: Fibonacci

```pipe
fn fib n
    match n
        | 0  -> 0
        | 1  -> 1
        | _  -> fib(n - 1) + fib(n - 2)

print (fib 10)    -- 55
```

### Calculator mit match

```pipe
fn calc a op b
    match op
        | "+" -> a + b
        | "-" -> a - b
        | "*" -> a * b
        | "/" -> a / b
        | "%" -> a % b
        | _   -> 0

print (calc 10 "+" 5)     -- 15
print (calc 10 "*" 5)     -- 50
```

## 4.3 while — While-Schleife

```pipe
i: 0
while i < 5
    print i
    i: i + 1
-- Ausgabe: 0, 1, 2, 3, 4
```

### break — Schleife abbrechen

```pipe
i: 0
while true
    print i
    i: i + 1
    if i >= 3
        break
-- Ausgabe: 0, 1, 2
```

### continue — Nächster Durchlauf

```pipe
i: 0
while i < 6
    i: i + 1
    if i % 2 == 0
        continue          -- Überspringt gerade Zahlen
    print i
-- Ausgabe: 1, 3, 5
```

## 4.4 for-in — Über Listen iterieren

```pipe
for farbe in ["rot", "grün", "blau"]
    print farbe
```

### range() für Zahlenbereiche

```pipe
-- range(n) → [0, 1, ..., n-1]
for i in (range 5)
    print i              -- 0, 1, 2, 3, 4

-- range(a, b) → [a, a+1, ..., b-1]
for i in (range 2 6)
    print i              -- 2, 3, 4, 5

-- range(a, b, step) → [a, a+step, ..., <b]
for i in (range 0 10 2)
    print i              -- 0, 2, 4, 6, 8
```

### for-in mit break

```pipe
for i in [5, 4, 3, 2, 1, 0]
    print i
    if i <= 2
        break
-- Ausgabe: 5, 4, 3, 2
```

## 4.5 C-style for — Volle Kontrolle

Die C-style `for`-Schleife bietet drei durch Semikolon getrennte Klauseln: Init, Bedingung und Update.

```pipe
for i: 0; i < 5; i: i + 1
    print i
-- Ausgabe: 0, 1, 2, 3, 4
```

### Leere Klauseln

Jede Klausel darf weggelassen werden. Init weglassen beginnt mit `;`:

```pipe
j: 0
for ; j < 3; j: j + 1
    print j
-- Ausgabe: 0, 1, 2
```

Ohne Bedingung läuft die Schleife endlos (`break` zum Beenden):

```pipe
k: 0
for ; ; k: k + 1
    if k >= 5
        break
    print k
```

### Rückwärts zählen

```pipe
for n: 10; n > 0; n: n - 1
    print n
-- Ausgabe: 10, 9, 8, 7, 6, 5, 4, 3, 2, 1
```

`break` und `continue` funktionieren wie in `while` und `for-in`.

## 4.6 try / catch — Fehlerbehandlung

`try` fängt Fehler während der Ausführung des try-Blocks. Produziert der Block
einen `ERROR`-Wert, springt die Ausführung in den catch-Block.

```pipe
try
    ergebnis: 10 / 0
catch fehler
    print "Division fehlgeschlagen"
    ergebnis: 0
```

Der catch-Parameter (`fehler` im Beispiel) erhält den Fehlerwert. 
Ohne Fehler wird der catch-Block übersprungen.

### try_ai — KI-Selbstheilung

`try_ai` funktioniert wie `try`, aber vor dem Fallthrough in den catch-Block
fragt Pipe einen KI-Provider, den Fehler automatisch zu beheben.

```pipe
-- KI kann Typ-Fehler korrigieren: String "42" + Zahl → konvertiert zu Zahl
try_ai
    ergebnis: "42" + 10
    print ergebnis
catch fehler
    print "Selbst die KI konnte es nicht reparieren"
```

Der `catch`-Block ist bei `try_ai` **optional** — ohne catch wird der Fehler
bei gescheitertem KI-Fix normal propagiert:

```pipe
try_ai
    schlecht: 10 / 0
-- Wenn KI Division durch Null nicht fixen kann, propagiert der Fehler
```

Benötigt einen konfigurierten KI-Provider und API-Key.

## 4.7 return — Vorzeitiges Verlassen

```pipe
fn betrag x
    if x < 0
        return (-x)
    x

print (betrag (-5))     -- 5
print (betrag 5)        -- 5
```

Ohne `return` ist der letzte Ausdruck im Funktionskörper der Rückgabewert.
`return` signalisiert ein vorzeitiges Verlassen.

```pipe
fn teile a b
    if b == 0
        return nil      -- Vorzeitig verlassen
    a / b               -- Normaler Rückgabewert
```

## 4.8 defer — Verzögerte Ausführung

`defer` plant eine Aktion für das **Ende des aktuellen Blocks** ein —
nützlich für Ressourcen-Freigabe (wie Go's `defer`):

```pipe
fn verarbeite_datei pfad
    print ("Öffne " ++ pfad)
    defer print ("Schließe " ++ pfad)  -- Wird am Ende ausgeführt
    print ("Verarbeite pfad")
    -- ... weitere Operationen

verarbeite_datei "daten.txt"
-- Ausgabe:
--   Öffne daten.txt
--   Verarbeite daten.txt
--   Schließe daten.txt    ← defer!
```

### LIFO-Reihenfolge

Defer-Ausdrücke werden in **umgekehrter Reihenfolge (LIFO)** ausgeführt —
der zuletzt deklarierte defer wird zuerst ausgeführt:

```pipe
fn demofunktion
    defer print "Dritter"
    defer print "Zweiter"
    defer print "Erster"
    print "Hauptlogik"

demofunktion
-- Ausgabe:
--   Hauptlogik
--   Erster    ← zuletzt defer't, zuerst ausgeführt
--   Zweiter
--   Dritter
```

Defer funktioniert auch auf **Top-Level** (wird am Programm-Ende ausgeführt).

## 4.9 enum — Enumerationen

Definiert benannte Konstanten, beginnend bei 0:

```pipe
enum Farbe: Rot, Grün, Blau

print Rot       -- 0
print Grün      -- 1
print Blau      -- 2
```

```pipe
enum Status: Aktiv, Inaktiv, Gelöscht

fn status_text s
    match s
        | Aktiv     -> "Aktiv"
        | Inaktiv   -> "Inaktiv"
        | Gelöscht  -> "Gelöscht"
        | _         -> "Unbekannt"

print (status_text Aktiv)     -- "Aktiv"
```

Enum-Werte sind einfache Integer (0, 1, 2, ...) und können wie Variablen verwendet werden.
```
