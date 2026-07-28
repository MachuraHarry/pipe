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
print (verdopple 21)       -- 42
print (addiere 3 4)        -- 7
print (begrüße "Welt")     -- "Hallo Welt"
```

**Bei einem Argument** können die Klammern entfallen:

```pipe
print "Hallo"              -- Äquivalent zu print("Hallo")
print (fib 10)              -- Klammern optional bei einem Arg
```

**Wichtig:** Rechenausdrücke als Argumente brauchen Klammern:

```pipe
print (1 + 2)              -- Richtig: print(3)
-- print 1 + 2             -- Falsch: (print 1) + 2
```

## 5.3 Mehrere Parameter

```pipe
fn begrüße vorname nachname
    "Hallo " ++ vorname ++ " " ++ nachname

print (begrüße "Max" "Mustermann"))     -- "Hallo Max Mustermann"
```

## 5.4 Anonyme Funktionen

Funktionen ohne Namen können in Variablen gespeichert werden:

```pipe
verdreifacher: fn x
    x * 3

print (verdreifacher 7)    -- 21
```

Anonyme Funktionen sind **First-Class Citizens** — sie können als Argumente
übergeben und von Funktionen zurückgegeben werden:

```pipe
fn anwenden f wert
    f wert

doppelt: fn x
    x * 2

print (anwenden doppelt 5)    -- 10
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

print (add5 7)      -- 12
print (add10 7)     -- 17
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

print (c1)     -- 1
print (c1)     -- 2
print (c2)     -- 101
print (c1)     -- 3
```

Jeder Zähler hat seinen eigenen internen Zustand.

### Praxis-Beispiel: Konfigurierbare Filter

```pipe
fn make_greater_than schwelle
    fn filter_fn x
        x > schwelle

filter: make_greater_than 5

print (filter 3)      -- false
print (filter 7)      -- true
print (filter 10)     -- true
```

## 5.6 Rekursion

Funktionen können sich selbst aufrufen:

```pipe
fn fakultät n
    if n <= 1
        1
    else
        n * (fakultät (n - 1))

print (fakultät 5)      -- 120
print (fakultät 10)     -- 3628800
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
        countdown (n - 1)       -- End-rekursiv → optimiert

countdown 5000                   -- Kein Stack-Overflow!
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
        n * (nicht_tco (n - 1))     -- × folgt dem rekursiven Aufruf
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

print (deutsch "Welt"))       -- "Hallo Welt"
print (franz "Monde"))        -- "Bonjour Monde"
print (englisch "World"))     -- "Hello World"
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
        x + y + local + global     -- Sieht alles!

    print (inner 3)                 -- 100 + 200 + x + 3

outer 10     -- 313
```

## 5.10 Funktionen und Pipeline

Funktionen sind die Bausteine der Pipeline:

```pipe
fn double x
    x * 2

fn add a b
    a + b

42
    > double            -- double(42) = 84
    > add 10             -- add(84, 10) = 94
    > print              -- 94
```
