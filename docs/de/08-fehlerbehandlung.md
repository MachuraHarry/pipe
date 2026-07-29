# 8. Fehlerbehandlung

## 8.1 try / catch

Pipe bietet `try`/`catch` zum Abfangen von Laufzeitfehlern:

```pipe
try
    ergebnis: 10 / 0            -- Gefährlicher Code
catch fehler
    print "Fehler aufgetreten:"
    print fehler                -- "Division durch Null"

print "Programm läuft weiter!"  -- Wird ausgeführt!
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
        1 / 0                 -- Erzwingt einen Fehler
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

print (betrag (-5))     -- 5
print (betrag 5)        -- 5

fn teile_sicher a b
    if b == 0
        return nil      -- Vorzeitiges Verlassen bei ungültiger Eingabe
    a / b

print (teile_sicher 10 0)   -- nil
print (teile_sicher 10 2)   -- 5
```

### 8.1.2 `try_ai` — KI-gesteuerte Selbstheilung

`try_ai` erweitert `try`/`catch` um automatische KI-Fehlerkorrektur. Bei einem fixbaren Fehler
analysiert und repariert die KI den Ausdruck — ohne Eingriff des Entwicklers.

```pipe
ai_provider "deepseek"

try_ai
    "42" * 3           -- Type Error → KI wrappt mit to_num → 126
catch e
    0                   -- Fallback wenn KI nicht fixen kann
```

#### Ablauf

1. **Fehler** tritt im `try_ai`-Block auf
2. **Error-Code** wird geprüft — nur E002, E003, E006 sind KI-fixbar
3. **KI-Aufruf** mit Fehlerkontext und Quellcode
4. **3-Ring-Validierung** — Parse → Sandbox-Test → Typ-Check
5. **Fix übernommen** oder **Fallback zum catch**

#### Fixbar vs. unfixbar

| Fehler | Code | KI-Strategie |
|--------|------|-------------|
| Typ-Fehler | E002 | `to_num`, `to_str` wrapping |
| Division durch Null | E003 | Guard: `max(x, 1)` oder if-Ausdruck |
| Index auf falschem Typ | E006 | Fallback via `get` mit Default |
| Undefinierte Variable | E001 | **Nicht fixbar** → direkt catch |
| Nicht aufrufbar | E004 | **Nicht fixbar** → direkt catch |

#### Ausführungsmodi

| Modus | `try_ai` Verhalten |
|-------|-------------------|
| Tree-Walker (`./bin/pipe`) | Vollständige KI-Selbstheilung |
| Bytecode-VM (`./bin/pipe -vm`) | Fallback zu normalem `try`/`catch` |

## 8.4 Result-Typ (Ok / Err)

Pipe bietet einen **Result-Typ** für Pipeline-kompatible Fehlerbehandlung
ohne try/catch:

```pipe
-- Erfolg
r1: Ok 42
print (is_ok r1)          -- true
print (is_err r1)         -- false
print (unwrap r1)         -- 42

-- Fehler
r2: Err "Etwas ist schiefgelaufen"
print (is_ok r2)          -- false
print (is_err r2)         -- true
print (unwrap r2)         -- ERROR (bricht ab)
print (unwrap_or r2 0)    -- 0 (Default-Wert)
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
    > sichere_division 2     -- Ok(5)
    > unwrap
    > print                   -- 5

10
    > sichere_division 0     -- Err("Division durch Null")
    > (fn r: unwrap_or r 0)  -- 0
    > print                   -- 0
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
print (sicherer_zugriff person "name")      -- "Anna"
print (sicherer_zugriff person "alter")     -- "Nicht gefunden"
```

### Datei-Existenz prüfen

```pipe
if file_exists "config.txt"
    inhalt: read_file "config.txt"
    print inhalt
else
    print "Konfiguration nicht gefunden"
```
