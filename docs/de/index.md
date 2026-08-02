# Pipe Dokumentation — Deutsch

Willkommen zur vollständigen Dokumentation der Skriptsprache **Pipe**.

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
10. [Builtin-Referenz](10-builtin-referenz.md) — Alle 115 eingebaute Funktionen
11. [Tooling](11-tooling.md) — CLI-Flags, REPL, Formatter, Test-Runner, Build
12. [Ausführungsmodelle](12-ausfuehrungsmodelle.md) — Tree-Walker vs Bytecode-VM
13. [Bytecode-VM](13-bytecode-vm.md) — 47 Opcodes, Stack-Maschine, Symbol-Tabelle
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

```pipe
-- Kommentar
x: 42                         -- Variable
x += 1                        -- Compound Assignment
fn name a b: ...              -- Funktion
if bed: ... else: ...         -- Bedingung
match x | 0 -> ... | _ ->     -- Pattern Matching
while bed: ...                -- Schleife
for x in liste: ...           -- for-in
try: ... catch e: ...         -- Fehlerbehandlung
return wert                   -- Vorzeitiges Verlassen
defer ausdruck                -- Verzögerte Ausführung
import "datei.pipe"           -- Modul laden
export fn name                -- Symbol exportieren
enum Name: A, B, C            -- Enumeration

wert
    > funktion                 -- Vertikale Pipeline
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
