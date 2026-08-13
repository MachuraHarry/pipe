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
6. **3-Ring-Validierung** — Parse → Isolierte Eval → echte Ausführung
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
| Rekursion zu tief | E008 | Rekursionstiefe reduzieren oder mit `while` umbauen |

#### Ausführungsmodi

| Modus | `try_ai` Verhalten |
|-------|-------------------|
| Tree-Walker (`./bin/pipe`) | Vollständige KI-Selbstheilung mit Retry |
| Bytecode-VM (`./bin/pipe -vm`) | Vollständige KI-Selbstheilung mit Retry (via Tree-Walker-Bridge) |

#### Ausgabe-Beispiel

```
⚡ try_ai: attempt 1 — "42" * 3 → "( (to_num "42") * 3 )"
✓ try_ai: fixed!
126
```

### 8.1.3 Sicherheitsanalyse — Ist `try_ai` sicher?

Diese Sektion adressiert die Kritik, dass KI-gesteuerte Code-Änderungen zur Laufzeit gefährlich seien. Wir analysieren jeden Risikovektor und die implementierten Schutzmechanismen.

#### Bedrohungsmodell

| Bedrohung | Risiko | Gegenmaßnahme |
|-----------|--------|---------------|
| KI generiert Schadcode | **HOCH** | 3-Ring-Validierung (Parse → Isolierte Eval → Echttest) verhindert Ausführung von ungültigem Code |
| KI halluziniert falschen Fix | **MITTEL** | Bis zu 3 Retry-Versuche; `catch`-Block als deterministischer Fallback |
| Prompt-Injection via Fehlermeldung | **MITTEL** | System-Prompt ist **fix und unveränderlich** (Compile-Zeit-Konstante); Nutzereingaben gehen nur in die `user`-Rolle |
| Seiteneffekte im fixierten Code | **HOCH** | Variablen‑isolierte Evaluation (`env.Copy()`) und **echte Sandbox-Aktivierung** in Ring 2: `FSNone`, `Network: false`, `Exec: false` blockiert alle I/O während der Fix-Validierung. AI erlaubt für Builtins. |
| API-Latenz macht Programm unberechenbar | **NIEDRIG** | `catch`-Block garantiert deterministischen Fallback; Retry-Limit von 3 begrenzt Worst-Case |

#### Defense in Depth — Die 3-Ring-Validierung

```
┌─────────────────────────────────────────────┐
│ Ring 1: PARSE-VALIDIERUNG                    │
│ KI-Ausgabe → Lexer → Parser → AST            │
│ Bei Parse-Fehlern → Fix ABGELEHNT            │
├─────────────────────────────────────────────┤
│ Ring 2: SANDBOX-EVALUATION                   │
│ AST → eval(env.Copy()) → Ergebnis            │
│ Bei ERROR oder nil → Fix ABGELEHNT            │
│ Sandbox aktiv: kein I/O, Exec, Netzwerk       │
│ Variablen in geklonter Umgebung isoliert      │
├─────────────────────────────────────────────┤
│ Ring 3: ECHTE EVALUATION                     │
│ Gleicher AST → eval(echte env) → final       │
│ Nur wenn Ringe 1+2 erfolgreich waren         │
└─────────────────────────────────────────────┘
```

#### Was `try_ai` sicher macht

1. **Begrenzte Angriffsfläche**: Nur einzelne Ausdrücke werden repariert — keine beliebigen Codeblöcke. Maximale Auswirkung: eine Zeile.

2. **Zustandslose Validierung**: Jeder Fix wird unabhängig validiert. Ein bösartiger Fix aus Aufruf N kann die Validierung von Aufruf N+1 nicht beeinflussen.

3. **Nicht‑persistente Fixes**: Der Fix wird nie auf die Festplatte geschrieben. Er existiert nur im Speicher während der Evaluierung und wird danach verworfen.

4. **Deterministische Notbremse**: `catch` ist immer verfügbar. Scheitert die KI nach 3 Versuchen, wird der Fallback-Code ausgeführt. Kein unerwartetes Verhalten verlässt den `try_ai`-Block.

5. **Keine System‑Prompt‑Injection möglich**: Der System-Prompt ist eine **Compile-Zeit-Konstante** im Go-Binary. Laufzeit-Fehlermeldungen landen in der `user`-Rolle und können die System-Anweisungen in keiner der unterstützten APIs überschreiben.

#### Vergleich mit der Industrie

| Tool | KI ändert Code zur Laufzeit? | Validierung | Fallback | Open Source? |
|------|------------------------------|-------------|----------|--------------|
| **Pipe `try_ai`** | Ja (nur Ausdrücke) | 3-Ring inkl. echter Sandbox | `catch`-Block | Ja (Apache 2.0) |
| GitHub Copilot | Nein (nur Vorschläge) | Manuelles Review | Manuelles Undo | Nein |
| Cursor AI | Nein (IDE-Integration) | Manuelles Review | Manuelles Undo | Nein |
| AutoGPT / AgentGPT | Ja (beliebiger Code) | Keine | Manueller Abbruch | Teilweise |

Pipe ist das **einzige Tool**, das automatisierte KI-Code-Reparatur mit echter Sandbox-Validierung (kein I/O, kein Exec, kein Netzwerk) und garantiertem deterministischen Fallback kombiniert — alles in einem einzigen Sprachkonstrukt.

#### Sicherheits-Fazit

`try_ai` ist **produktionstauglich** wenn:
- Ein KI-Provider konfiguriert ist (`ai_provider`)
- Ein `catch`-Block für kritische Codepfade vorhanden ist
- Der System-Prompt unverändert bleibt (Compile-Zeit-Konstante)

`try_ai` fügt **null Risiko** hinzu wenn:
- Kein Fehler auftritt (die KI wird nie aufgerufen)
- Im Bytecode-VM-Modus (`pipe -vm`, try_ai funktioniert vollständig via Tree-Walker-Bridge)
- Der Ausdruck bereits gültiger Pipe-Code ist

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
