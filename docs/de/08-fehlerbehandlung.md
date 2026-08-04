# 8. Fehlerbehandlung

## 8.1 try / catch

Pipe bietet `try`/`catch` zum Abfangen von Laufzeitfehlern:

```pipe
try
    -- Gefährlicher Code
        ergebnis: 10 / 0
catch fehler
    print "Fehler aufgetreten:"
    -- "Division durch Null"
        print fehler

-- Wird ausgeführt!
print "Programm läuft weiter!"
```

### Funktionsweise

1. Der `try`-Block wird ausgeführt
2. Tritt ein Fehler auf, wird die Ausführung im `try`-Block **sofort abgebrochen**
3. Der Fehler wird an den `catch`-Parameter gebunden (hier `fehler`)
4. Der `catch`-Block wird ausgeführt
5. Das Programm läuft **normal weiter** nach dem `try`/`catch`

### Fehler-Funktion

```pipe
fn teile a b
    if b == 0
        -- Erzwingt einen Fehler
                1 / 0
    else
        a / b

try
    print (teile 10 0)
catch e
    print "Division fehlgeschlagen:"
    print e
```

## 8.2 Stack-Traces

Bei Fehlern wird automatisch ein **Stack-Trace** erzeugt, der die
Aufrufkette zeigt:

```
ERROR: Division durch Null
  in fn(berechne)
  in fn(hauptprogramm)
```

## 8.3 return für Vorzeitiges Verlassen

```pipe
fn betrag x
    if x < 0
        return (-x)
    x

-- 5
print (betrag (-5))
-- 5
print (betrag 5)

fn teile_sicher a b
    if b == 0
        -- Vorzeitiges Verlassen bei ungültiger Eingabe
                return nil
    a / b

-- nil
print (teile_sicher 10 0)
-- 5
print (teile_sicher 10 2)
```

### 8.1.2 `try_ai` — KI-gesteuerte Selbstheilung

`try_ai` erweitert `try`/`catch` um automatische KI-Fehlerkorrektur. Bei einem fixbaren Fehler
analysiert und repariert die KI den Ausdruck — ohne Eingriff des Entwicklers.

```pipe
ai_provider "deepseek"

try_ai
        "42" * 3
catch e
        0
```

**`catch` ist optional** — ohne catch wird ein unfixbarer Fehler normal weitergegeben:

```pipe
try_ai
    "42" * 3
-- Ergebnis: 126 (KI hat Typfehler stillschweigend behoben)
```

#### Ablauf

1. **Fehler** tritt im `try_ai`-Block auf
2. **Error-Code** wird geprüft — E001-E006 sind alle KI-fixbar
3. **KI-Aufruf** mit Fehlerkontext und Quellcode
4. **Bis zu 3 Wiederholungsversuche** — schlägt der Fix fehl, wird mit mehr Kontext erneut versucht
5. **Feedback auf stderr** — jeder Versuch mit Diff: `⚡ try_ai: attempt 1 — "42" * 3 → "(to_num "42") * 3"`
6. **3-Ring-Validierung** — Parse → Sandbox-Test → echte Ausführung
7. **Fix übernommen** oder **Fallback zum catch**

#### Fixbare Error-Codes

| Fehler | Code | KI-Strategie |
|--------|------|-------------|
| Undefinierte Variable | E001 | Literale Default-Werte |
| Typ-Fehler | E002 | `to_num`, `to_str` wrapping |
| Division durch Null | E003 | Guard: `max(x, 1)` oder if-Ausdruck |
| Keine Funktion | E004 | Klammern oder Builtin |
| Operator nicht unterstützt | E005 | Typ-Konvertierung |
| Ungültiger Index | E006 | Guard mit `len` oder `get` |

#### Ausführungsmodi

| Modus | `try_ai` Verhalten |
|-------|-------------------|
| Tree-Walker (`./bin/pipe`) | Vollständige KI-Selbstheilung mit Retry |
| Bytecode-VM (`./bin/pipe -vm`) | Fallback zu normalem `try`/`catch` (kein AI — VM ist für Produktion) |

#### Ausgabe-Beispiel

```
⚡ try_ai: attempt 1 — "42" * 3 → "( (to_num "42") * 3 )"
✓ try_ai: fixed!
126
```

## 8.4 Result-Typ (Ok / Err)

Pipe bietet einen **Result-Typ** für Pipeline-kompatible Fehlerbehandlung
ohne try/catch:

```pipe
-- Erfolg
r1: Ok 42
-- true
print (is_ok r1)
-- false
print (is_err r1)
-- 42
print (unwrap r1)

-- Fehler
r2: Err "Etwas ist schiefgelaufen"
-- false
print (is_ok r2)
-- true
print (is_err r2)
-- ERROR (bricht ab)
print (unwrap r2)
-- 0 (Default-Wert)
print (unwrap_or r2 0)
```

### Result-Funktionen

| Funktion | Beschreibung |
|----------|-------------|
| `Ok wert` | Erzeugt ein erfolgreiches Result |
| `Err nachricht` | Erzeugt ein Fehler-Result |
| `is_ok r` | Prüft ob Result ein Erfolg ist |
| `is_err r` | Prüft ob Result ein Fehler ist |
| `unwrap r` | Extrahiert den Wert (bricht bei Fehler ab) |
| `unwrap_or r default` | Extrahiert den Wert oder gibt Default zurück |

### Result in Pipelines

```pipe
fn sichere_division a b
    if b == 0
        Err "Division durch Null"
    else
        Ok (a / b)

10
    > sichere_division 2
    > unwrap
    > print

10
    > sichere_division 0
    > (fn r
        unwrap_or r 0)
    > print
```

## 8.5 Fehler vermeiden

### Nil-Checks

```pipe
fn sicherer_zugriff map key
    wert: get map key
    if is_nil wert
        "Nicht gefunden"
    else
        wert

person: {name: "Anna"}
-- "Anna"
print (sicherer_zugriff person "name")
-- "Nicht gefunden"
print (sicherer_zugriff person "alter")
```

### Datei-Existenz prüfen

```pipe
if file_exists "config.txt"
    inhalt: read_file "config.txt"
    print inhalt
else
    print "Konfiguration nicht gefunden"
```
