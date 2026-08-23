# 5. Funktionen und Closures

## 5.1 Funktionsdefinition

```pipe
fn verdopple x
    x * 2

fn addiere a b
    a + b
```

- `fn` leitet die Definition ein
- Dann der **Name**, dann die **Parameter** (durch Leerzeichen getrennt — keine Kommas, keine Klammern)
- Der **Körper** wird eingerückt
- Der **letzte Ausdruck** im Körper ist der Rückgabewert
- Kein `return`-Keyword nötig (aber optional verfügbar für vorzeitiges Verlassen)

## 5.2 Funktionsaufruf

Funktionsaufrufe verwenden **Leerzeichen** statt Kommas:

```pipe
-- 42
print (verdopple 21)
-- 7
print (addiere 3 4)
-- "Hallo Welt"
print (begruesse "Welt")
```

**Bei einem Argument** können die Klammern entfallen:

```pipe
-- Äquivalent zu print("Hallo")
print "Hallo"
-- Klammern optional bei einem Arg
print (fib 10)
```

**Wichtig:** Rechenausdrücke als Argumente brauchen Klammern:

```pipe
-- Richtig: print(3)
print (1 + 2)
-- print 1 + 2             -- Falsch: (print 1) + 2
```

## 5.3 Mehrere Parameter

```pipe
fn begruesse vorname nachname
    "Hallo " ++ vorname ++ " " ++ nachname

-- "Hallo Max Mustermann"
print (begruesse "Max" "Mustermann")
```

## 5.4 Anonyme Funktionen

Funktionen ohne Namen können in Variablen gespeichert werden:

```pipe
verdreifacher: fn x
    x * 3

-- 21
print (verdreifacher 7)
```

### Inline-Lambda-Syntax (v0.8+)

Für einzeilige Funktionskörper unterstützt Pipe eine kompakte Inline-Syntax mit Doppelpunkt:

```pipe
-- Inline: fn param: ausdruck
doppelt: fn x: x * 2

-- Multi-Parameter Inline-Lambda
addiere: fn a b: a + b

-- Als Argument für filter
filter [1, 2, 3, 4, 5] (fn x: x > 2)

-- Als Argument für map
map [1, 2, 3] (fn x: x * 10)

-- In Pipeline-Ketten
[1, 2, 3, 4, 5]
    > filter (fn x: x % 2 == 0)
    > map (fn x: x * 3)
    > print
```

Die Inline-Form ist äquivalent zur mehrzeiligen Block-Form, erlaubt aber
einfache Funktionsliterale in einer Zeile. Die Parameter vor dem Doppelpunkt
werden zu Funktionsparametern; der Ausdruck nach dem Doppelpunkt ist der
Funktionskörper.

> **Leerzeichen-Regel für `[`**: direkt an einem Wert angehängt ist `[...]`
> ein Index-/Slice-Zugriff (`xs[0]`, `xs[1..3]`); durch Leerzeichen getrennt
> beginnt ein neues Listenliteral — als Statement (`xs: [1, 2]`) oder als
> Aufrufargument (`map [1, 2, 3] (fn x: x * 10)` oben). Für einen Index auf
> ein Ergebnis nutze lieber eine Variable: `r: map nums f`, dann `r[0]`.

Für mehrzeilige Funktionskörper verwende die eingerückte Block-Form
(`fn params\n    body`).

Anonyme Funktionen sind **First-Class Citizens** — sie können als Argumente
übergeben und von Funktionen zurückgegeben werden:

```pipe
fn anwenden f wert
    f wert

doppelt: fn x
    x * 2

-- 10
print (anwenden doppelt 5)
```

## 5.5 Closures

Funktionen merken sich den Scope, in dem sie definiert wurden.
Das ermöglicht **Closures** — Funktionen mit eingefangenem Kontext:

```pipe
fn make_adder n
    fn adder x
        x + n

add5: make_adder 5
add10: make_adder 10

-- 12
print (add5 7)
-- 17
print (add10 7)
```

`adder` fängt die Variable `n` aus dem äußeren Scope ein —
bei jedem `make_adder`-Aufruf mit einem eigenen `n`.

### Praxis-Beispiel: Zähler-Funktion

```pipe
fn make_counter start
    count: start
    fn counter
        count: count + 1
        count

c1: make_counter 0
c2: make_counter 100

-- 1
print (c1)
-- 2
print (c1)
-- 101
print (c2)
-- 3
print (c1)
```

Jeder Zähler hat seinen eigenen internen Zustand.

### Praxis-Beispiel: Konfigurierbare Filter

```pipe
fn make_greater_than schwelle
    fn filter_fn x
        x > schwelle

filter: make_greater_than 5

-- false
print (filter 3)
-- true
print (filter 7)
-- true
print (filter 10)
```

## 5.6 Rekursion

Funktionen können sich selbst aufrufen:

```pipe
fn fakultaet n
    if n <= 1
        1
    else
        n * (fakultaet (n - 1))

-- 120
print (fakultaet 5)
-- 3628800
print (fakultaet 10)
```

## 5.7 Tail Call Optimization (TCO)

Der Tree-Walker erkennt **rekursive End-Aufrufe** und optimiert sie
zu einer Schleife. Das erlaubt tiefe Rekursion ohne Stack-Overflow:

```pipe
fn countdown n
    if n <= 0
        print "Start!"
    else
        print n
        -- End-rekursiv -> optimiert
                countdown (n - 1)

-- Kein Stack-Overflow!
countdown 5000
```

**Bedingungen für TCO:**
1. Die Funktion ruft sich selbst auf
2. Der rekursive Aufruf ist der **letzte Ausdruck** im Funktionskörper
3. Kein nachfolgender Code (wie `+` oder `print`) nach dem Aufruf

**Kein TCO (nicht end-rekursiv):**
```pipe
fn nicht_tco n
    if n <= 1
        1
    else
        -- × folgt dem rekursiven Aufruf
                n * (nicht_tco (n - 1))
```

## 5.8 Funktionen als Rückgabewert

```pipe
fn make_greeter sprache
    if sprache == "DE"
        fn name
            "Hallo " ++ name
    else if sprache == "FR"
        fn name
            "Bonjour " ++ name
    else
        fn name
            "Hello " ++ name

deutsch: make_greeter "DE"
franz: make_greeter "FR"
englisch: make_greeter "EN"

-- "Hallo Welt"
print (deutsch "Welt")
-- "Bonjour Monde"
print (franz "Monde")
-- "Hello World"
print (englisch "World")
```

## 5.9 Scope und Variablen-Sichtbarkeit

Funktionen sehen:
1. Ihre **eigenen Parameter**
2. Variablen im **umschließenden Scope** (Closure)
3. **Globale Variablen** und **Builtins**

```pipe
global: 100

fn outer x
    local: 200
    fn inner y
        -- Sieht alles!
        x + y + local + global

    -- 100 + 200 + x + 3
    print (inner 3)

-- 313
outer 10
```

## 5.10 Funktionen und Pipeline

Funktionen sind die Bausteine der Pipeline:

```pipe
fn double x
    x * 2

fn add a b
    a + b

42
    -- double(42) = 84
        > double
    -- add(84, 10) = 94
        > add 10
    -- 94
        > print
```
