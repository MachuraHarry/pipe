# 6. Pipelines

Die Pipeline ist das **zentrale Sprach-Feature** von Pipe. Sie erlaubt es,
Werte durch eine Kette von Transformationen zu leiten — von oben nach unten,
in Leserichtung des Datenflusses.

## 6.1 Grundprinzip

Statt Funktionen ineinander zu verschachteln:

```pipe
print (addiere (verdopple 42) 10)
```

Schreibt man in Pipe die Datenfluss-Richtung aus:

```pipe
42
    > verdopple
    > addiere 10
    > print
```

Bedeutung: `print(addiere(verdopple(42), 10))`  
Ausgabe: `94`

Jede eingerückte `>`-Zeile nimmt das Ergebnis der vorherigen Zeile
und übergibt es als **erstes Argument** an die angegebene Funktion.

## 6.2 Vertikale Pipeline (empfohlen)

Die vertikale Pipeline ist die bevorzugte Schreibweise:

```pipe
fn double x
    x * 2

fn add a b
    a + b

fn is_positive x
    x > 0

42
    -- double(42) = 84
    > double
    -- add(84, 10) = 94
    > add 10
    -- 94
    > print

100
    > is_positive
    > print
```

### Syntax-Regeln

1. Der **Anfangswert** steht auf einer eigenen Zeile, nicht eingerückt
2. Jede Pipeline-Stufe beginnt mit `>` und ist **eingerückt** (4 Leerzeichen)
3. Nach dem `>` folgt die Funktion und optionale Zusatzargumente
4. Das Ergebnis jeder Stufe wird als **erstes Argument** an die nächste Stufe weitergereicht

## 6.3 Pipeline mit Zusatzargumenten

Wenn die Pipeline-Funktion mehr als ein Argument braucht, werden die
zusätzlichen Argumente nach dem Funktionsnamen angegeben:

```pipe
100
    -- add(100, 50) = 150
        > add 50
    -- 150
        > print
```

Das Pipeline-Ergebnis (100) wird als erstes Argument eingefügt,
die zusätzlichen Argumente (50) folgen.

## 6.3b Parallele Pipelines (`>>`)

Der `>>`-Operator (Doppelpfeil) startet eine Pipeline-Stufe **im Hintergrund**
und gibt sofort einen **Future** (ein Platzhalter-Ticket) zurück. Das Programm
läuft weiter, ohne auf das Ergebnis zu warten.

Sobald das Ergebnis gebraucht wird (Arithmetik, Funktionsaufruf, Ausgabe etc.),
wartet Pipe **automatisch** auf den Future.

### Syntax

```pipe
wert
    -- startet im Hintergrund, gibt Future zurueck
    >> langsame_operation
    -- wartet auf Future-Aufloesung
    > naechste_operation
    > print
```

### Warum parallele Pipelines?

KI-Abfragen und I/O-Operationen dauern oft Sekunden. Mit `>` läuft alles sequentiell:

```pipe
-- Sequentiell: ~9 Sekunden (3 × 3 Sekunden)
"Frage 1" > ask > print
"Frage 2" > ask > print
"Frage 3" > ask > print
```

Mit `>>` laufen alle drei gleichzeitig:

```pipe
-- Parallel: ~3 Sekunden (alle 3 parallel)
"Frage 1" >> ask
"Frage 2" >> ask
"Frage 3" >> ask

-- wartet automatisch
print antwort1 ++ antwort2 ++ antwort3
```

### Futures: Automatische Auflösung

Wenn eine `>>`-Stufe einen Future zurückgibt, wird dieser automatisch
aufgelöst sobald der Wert konsumiert wird:

```pipe
-- Future in Variable speichern
ergebnis: 10
    >> langsam_verdoppeln

-- Arithmetik wartet automatisch
ergebnis + 100
    -- Future wird vor der Addition aufgelöst
        > print

-- String-Konkatenation wartet auch
-- wartet auf Future
"Wert: " ++ ergebnis

-- Funktionsargumente warten
-- wartet vor dem Vergleich
max ergebnis 50
```

### Mischen von `>` und `>>`

Parallele und sequentielle Pipeline-Stufen lassen sich beliebig kombinieren:

```pipe
daten
    -- parallel: API-Call starten
    >> api_abfrage
    -- sequentiell: warten, dann parsen
    > parse_json
    -- parallel: Analyse starten
    >> analysieren
    -- sequentiell: warten, dann formatieren
    > formatieren
    > print
```

Jedes `>>` startet sofort, jedes `>` wartet auf den vorherigen Schritt.

### Ausführungsmodi

| Modus | `>>` Verhalten |
|-------|---------------|
| Tree-Walker (`./bin/pipe`) | Echte Parallelität via Goroutinen für alle Funktionen |
| Bytecode-VM (`./bin/pipe -vm`) | Echte Parallelität für Builtins (KI, I/O etc.), synchrone Ausführung für Closures |

## 6.4 `_` Platzhalter für Argument-Position

Mit `_` kann der Pipeline-Wert an einer **beliebigen Position** eingefügt werden,
nicht nur als erstes Argument:

```pipe
fn subtrahiere a b
    a - b

-- Standard: subtrahiere(10, 3) = 7
10
    > subtrahiere 3
    > print

-- Mit Platzhalter: subtrahiere(3, 10) = -7
10
    > subtrahiere 3 _
    > print
```

`_` wird durch den Pipeline-Wert ersetzt. Ohne `_` wird der Wert
automatisch als **erstes Argument** eingefügt.

## 6.5 Pipeline mit eingebauten Funktionen

```pipe
[3, 1, 4, 1, 5, 9]
    -- sort([3,1,4,1,5,9]) = [1,1,3,4,5,9]
        > sort
    -- push([1,1,3,4,5,9], 10) = [1,1,3,4,5,9,10]
        > push 10
    -- [1, 1, 3, 4, 5, 9, 10]
        > print

"hallo welt"
    -- upper("hallo welt") = "HALLO WELT"
        > upper
    -- split("HALLO WELT", " ") = ["HALLO", "WELT"]
        > split " "
    -- ["HALLO", "WELT"]
        > print
```

## 6.6 Datei-Pipeline (Praxisbeispiel)

```pipe
is_error: fn l
    contains l "ERROR"

read_file "log.txt"
    -- Zeilenweise
    > split "\n"
    -- Nur ERROR-Zeilen
    > filter is_error
    -- Sortieren
    > sort
    -- Ausgeben
    > print
```

## 6.7 Horizontale Pipeline

Die Pipeline funktioniert auch einzeilig (horizontal), ist aber weniger lesbar:

```pipe
-- 94
print (42 > double > add 10)
```

Die horizontale Form verwendet ebenfalls `>` zwischen den Stufen,
braucht aber Klammern um den gesamten Ausdruck.

**Wichtig:** `>` hat im Ausdruckskontext zwei Bedeutungen:
1. **Vergleichsoperator** bei `a > b` („a größer als b")
2. **Pipeline-Operator** in der vertikalen Form mit Einrückung

Im Zweifel die vertikale Form verwenden — sie ist eindeutig und lesbarer.

## 6.8 Pipeline vs Funktionale Komposition

Eine Pipeline ist äquivalent zur funktionalen Komposition — nur in
umgekehrter Leserichtung:

```
Pipeline:   42 > double > add 10 > print
Funktional: print(add10(double(42)))
```

Die Pipeline-Schreibweise entspricht dem **Datenfluss von oben nach unten**
und macht Programme leichter lesbar.

## 6.9 Mehrstufige Pipeline mit bedingter Logik

```pipe
fn bewertung punkte
    if punkte >= 90
        "A"
    else if punkte >= 80
        "B"
    else
        "C"

punkte_liste: [95, 82, 67, 91, 73]
punkte_liste
    > map bewertung
    > print
```

## 6.10 Design-Prinzip

> **Daten fließen lassen, nicht Klammern zählen.**

Die Pipeline macht den Datenfluss **sichtbar**. Statt verschachtelter
Klammern (`f(g(h(x)))`) liest man das Programm als gerichteten Fluss
von Daten durch Transformationen.
