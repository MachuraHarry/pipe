# Pipe Dokumentation — Deutsch

Willkommen zur vollständigen Dokumentation der Skriptsprache **Pipe**.

## Wo soll ich anfangen?

| Ich möchte… | Hier starten |
|-------------|--------------|
| 🚀 **In 5 Minuten ausprobieren** | → [Erste Schritte](01-erste-schritte.md) — Installation, Hallo Welt, erste Pipeline |
| 🐍 **Ich komme von Python/Bash/JS** | → [Sprachübersicht](02-sprachuebersicht.md) — Unterschiede in 10 Minuten |
| ⚙️ **Die VM verstehen** | → [Architektur](14-architektur.md) — Lexer → Parser → Bytecode-VM |
| 🤖 **KI-Funktionen nutzen** | → [KI-Builtins](19-ki-builtins.md) — 36 KI-Operationen als Sprach-Primitives |

## Inhaltsverzeichnis

1. [Erste Schritte](01-erste-schritte.md) — Installation, Hallo Welt, 5-Minuten-Quickstart
2. [Sprachübersicht](02-sprachuebersicht.md) — Kommentare, Einrückung, Variablen, Funktionsaufrufe, alle Keywords
3. [Typen und Ausdrücke](03-typen-und-ausdruecke.md) — Datentypen, Literale, Operatoren, Präzedenz
4. [Kontrollfluss](04-kontrollfluss.md) — if/else, match, while, break, continue, for-in, return, defer
5. [Funktionen und Closures](05-funktionen-und-closures.md) — fn-Definition, Parameter, anonyme Fns, Closures, TCO
6. [Pipelines](06-pipelines.md) — Das Kern-Feature: vertikale/horizontale Pipeline, \_-Platzhalter
7. [Datenstrukturen](07-datenstrukturen.md) — Listen, Slicing, Maps, Dot-Access, höhere Funktionen
8. [Fehlerbehandlung](08-fehlerbehandlung.md) — try/catch, Stack-Traces, Result-Typ
9. [Module und Importe](09-module-und-importe.md) — import, export, Namespaces, PIPE\_PATH
10. [Builtin-Referenz](10-builtin-referenz.md) — Alle 142 eingebaute Funktionen
11. [Tooling](11-tooling.md) — CLI-Flags, REPL, Formatter, Test-Runner, Build
12. [Ausführungsmodelle](12-ausfuehrungsmodelle.md) — Tree-Walker vs Bytecode-VM
13. [Bytecode-VM](13-bytecode-vm.md) — 40 Opcodes, Stack-Maschine, Symbol-Tabelle
14. [Architektur](14-architektur.md) — Lexer, Parser, AST, Compiler, VM intern
15. [VSCode-Erweiterung](15-vscode-erweiterung.md) — Installation, Syntax-Highlighting
16. [Kochrezepte](16-kochrezepte.md) — 20+ praktische Code-Beispiele
17. [Migration von anderen Sprachen](17-migration-von.md) — Python, Lua, Bash, JavaScript
18. [Roadmap](18-roadmap.md) — Zukunft der Sprache
19. [KI-Builtins](19-ki-builtins.md) — KI-Funktionen für LLM-Integration
20. [GitHub Action](20-github-action.md) — Pipe in CI/CD-Pipelines nutzen
21. [Modul-Ökosystem](21-ecosystem.md) — Module finden, installieren, beitragen
22. [Sandbox-Profile](22-sandbox-profile.md) — Deklarative Sicherheitsprofile für KI-Agenten und nicht vertrauenswürdigen Code

## Anhang

- [Anhang A: Formale Grammatik](anhang-a-grammatik.md) — Vollständige EBNF-Grammatik

## Schnellreferenz

```text
-- Kommentar
-- Variable
x: 42
-- Compound Assignment
x: x + 1
-- Funktion
fn name a b: ...
-- Bedingung
if bed: ... else: ...
-- Pattern Matching
match x | 0 -> ... | _ ->
-- Schleife
while bed: ...
-- for-in
for x in liste: ...
-- Fehlerbehandlung
try: ... catch e: ...
-- Vorzeitiges Verlassen
return wert
-- Verzögerte Ausführung
defer ausdruck
-- Modul laden
import "datei.pipe"
-- Symbol exportieren
export fn name
A: 0
B: 1
-- Enumeration: 2
C

wert
    -- Vertikale Pipeline
        > funktion
    > ausgabe
```

## Operatoren

```
+ - * / % **      Arithmetik + Potenz
+= -= *= /= %=     Compound Assignment
== != < > <= >=   Vergleiche
! && ||            Logik (not, and, or)
++                 String-Verkettung
>                  Pipeline (vertikal mit Einrückung)
>>                 Parallele Pipeline (Hintergrundausführung)
..                 Bereich (Slicing)
```
