# Pipe – Verbesserungsplan

## 1. Aussagekräftige Fehlermeldungen

**Status:** 🔶 Teilweise erledigt — Parser/Eval/Builtin-Texte verbessert, AST-Positionsinfos fehlen noch.

### Parser (aktuell)
```
line 2 col 5: no prefix parser for NEWLINE ("\n")
line 3 col 11: no prefix parser for : (":")
```
→ kryptisch, kein Kontext, keine Vorschläge

### Evaluator (aktuell)
```
E002: Type error: STRING > BUILTIN
```
→ keine Zeile/Spalte, kein Hinweis was gemeint war

### Builtins (aktuell)
```
read_file: no such file or directory
```
→ kein Quellort, kein Kontext

### Ziel
Jeder Fehler enthält:
- Datei + Zeile + Spalte
- Das relevante Source-Snippet (optional)
- Konkrete Ursache in natürlicher Sprache
- Ggf. Vorschlag zur Korrektur

### Aufwand
- AST-Positionsinfos in alle Nodes einbauen: 3 Tage
- Parser-Fehlermeldungen umschreiben: 1 Tag
- Evaluator-Fehlermeldungen umschreiben: 2 Tage
- Builtin-Fehler umschreiben: 1 Tag

---

## 2. Builtin-Referenz-Dokumentation

Kein Builtin ist vollständig dokumentiert mit:
- Signatur (Argumente + Rückgabe)
- Beispiel-Code
- Sandbox-Verhalten
- Fehlermeldungen

Jeder Builtin in `object.go` kriegt einen Doku-Eintrag in `docs/en/10-builtin-reference.md`.

**Aufwand:** 2–3 Tage (ca. 100 Builtins)

---

## 3. Test-Assertions in der Sprache

Aktuell: `-test` existiert, aber man kann nur per `==` vergleichen und selbst `print`-Auswerten.

Gewünscht:
```pipe
test "addition"
    assert (2 + 2) == 4
    assert_eq (2 + 2) 4
    assert_lt 3 5

test "list operations"
    assert (len [1,2,3]) == 3
    assert_error { read_file "nix" }
```

**Aufwand:** 2 Tage (Parser + Evaluator + Test-Runner)

**Status:** ✅ Erledigt (2026-07-30)
- Lexer: `TEST`-Token + `"test"`-Keyword
- AST: `TestStatement` (Name + Body)
- Parser: `parseTestStatement()` — `test "name" \n <body>`
- Evaluator: `evalTestStatement()` — führt Body aus, fängt Fehler, reportet PASS/FAIL
- Builtins (6 neue): `assert`, `assert_eq`, `assert_not_eq`, `assert_lt`, `assert_gt`, `assert_error`
- Test-Runner: per-test Ausgabe (`PASS name` / `FAIL name (reason)`), alle Tests laufen (erster Fehler bricht nur den Block ab)
- Integration-Test: `test/integration/assertions_test.pipe`

```pipe
test "addition"
    assert_eq (2 + 2) 4
    assert_lt 3 5

test "assert error"
    failing: fn
        read_file "nonexistent"
    assert_error failing
```

---

## 4. Language Server (LSP)

Aktuell: VSCode-Extension ist reiner Syntax-Highlighter.
- Kein Autocomplete
- Keine Hover-Docs
- Keine Diagnostics
- Kein Go-to-Definition

Ein minimaler LSP in Go mit `textdocument` + `jsonrpc` wäre der nächste Schritt.

**Aufwand:** 5–7 Tage (Grundfunktionen)

---

## 5. Debugger

- `debug`-Keyword oder `breakpoint()`-Builtin
- Step-Through im REPL
- Variablen-Inspektion
- Call-Stack-Navigation

**Aufwand:** 3–4 Tage

---

## 6. Ökosystem und Tools

- `pipe init` – Projekt-Scaffolding
- `pipe publish` – Modul hochladen
- Zentrale Registry-Website
- `pipe upgrade` – Abhängigkeiten aktualisieren

**Aufwand:** 5–7 Tage

---

## 7. Graduelles Typ-System

Optionale Typannotationen:
```pipe
fn add(x: int, y: int) -> int
    x + y

name: string = "welt"
```
Typ-Prüfung als separater Durchlauf (optional, nicht zur Laufzeit).

**Aufwand:** 7–10 Tage

---

## Priorisierung

| Rang | Thema | Aufwand | Impact | Status |
|------|-------|---------|--------|--------|
| 1 | Fehlermeldungen | ~7 Tage | ★★★★★ | 🔶 Parser/Eval/Builtins verbessert, Positionsinfos fehlen |
| 2 | Builtin-Doku | ~3 Tage | ★★★★ | ✅ Erledigt (en+de, 105 Builtins) |
| 3 | Test-Assertions | ~2 Tage | ★★★★ | ✅ Erledigt (test, assert, assert_eq, assert_error) |
| 4 | LSP | ~6 Tage | ★★★ |
| 5 | Debugger | ~4 Tage | ★★★ |
| 6 | Ökosystem | ~6 Tage | ★★ |
| 7 | Typ-System | ~8 Tage | ★★ |
