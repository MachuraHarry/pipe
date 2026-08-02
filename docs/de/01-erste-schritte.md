# 1. Erste Schritte

## 1.1 Was ist Pipe?

Pipe ist eine **einrückungsbasierte, pipeline-orientierte Skriptsprache**,
implementiert in Go. Sie wird als einzelne, statisch gelinkte Binary
ausgeliefert (~10 MB).

**Kernphilosophie:** Daten fließen sichtbar von oben nach unten durch
eine Kette von Transformationen — nicht versteckt in verschachtelten
Klammerausdrücken.

```pipe
-- Statt: print(addiere(verdopple(42), 10))
42
    > verdopple       -- verdopple(42) = 84
    > addiere 10      -- addiere(84, 10) = 94
    > print           -- Ausgabe: 94
```

## 1.2 Voraussetzungen

- **Go 1.21+** (zum Bauen aus dem Quellcode)
- Keine weiteren Abhängigkeiten — Pipe nutzt ausschließlich die Go-Standardbibliothek

## 1.3 Installation

```bash
# Repository klonen
git clone <pipe-repo>
cd pipe

# Bauen (erzeugt bin/pipe)
make build

# Testen ob alles funktioniert
./bin/pipe examples/hello.pipe
```

Die Binary `bin/pipe` kann direkt auf andere Systeme kopiert werden — sie ist statisch gelinkt und benötigt keine Laufzeitumgebung.

## 1.4 Hallo Welt

Erstelle eine Datei `hello.pipe`:

```pipe
-- Mein erstes Pipe-Programm
print "Hallo Welt"
```

Ausführen:

```bash
./bin/pipe hello.pipe
```

Ausgabe:
```
Hallo Welt
```

## 1.5 Die REPL (Interaktiver Modus)

Starte `./bin/pipe` ohne Dateinamen, um in die REPL zu gelangen:

```
$ ./bin/pipe
Pipe v0.7.0 — REPL
>>> 1 + 2
3
>>> print "Hallo"
Hallo
>>> :quit
```

**REPL-Befehle:**

| Befehl | Wirkung |
|--------|---------|
| `:quit`, `:q` | Beenden |
| `:help`, `:h` | Hilfe anzeigen |
| `:clear`, `:c` | Eingabe zurücksetzen |
| `:vm` | VM-Modus umschalten (Tree-Walker ↔ VM) |
| `:history` | Letzte Befehle anzeigen (bis zu 100) |
| `:!N` | Befehl N aus der History wiederholen |
| `Strg+D` | Beenden |
| Leerzeile | Mehrzeiligen Block abschließen |

Nach `fn`, `if`, `match`, `while`, `for`, `try`, `defer`, `export`, `enum`
wird automatisch in den mehrzeiligen Modus geschaltet (erkennbar am `...`-Prompt).

## 1.6 Ausführungsmodi

Pipe hat zwei Ausführungs-Engines:

| Modus | Befehl | Geschwindigkeit | Features |
|-------|--------|----------------|----------|
| **Tree-Walker** | `./bin/pipe datei.pipe` | Basis | Alle Features |
| **Bytecode-VM** | `./bin/pipe -vm datei.pipe` | ~7× schneller | Meiste Features |

Weitere Details in [Kapitel 12: Ausführungsmodelle](12-ausfuehrungsmodelle.md).

## 1.7 CLI-Argumente an Skripte übergeben

```bash
./bin/pipe script.pipe arg1 arg2 arg3
```

Im Skript verfügbar als `args`-Liste:

```pipe
print args                  -- ["arg1", "arg2", "arg3"]
print (at args 0)           -- "arg1"
print (len args)            -- 3
```

## 1.8 Hilfe anzeigen

```bash
./bin/pipe -h
```

Gibt alle verfügbaren CLI-Optionen aus.

## 1.9 Erstes kleines Programm

```pipe
fn verdopple x
    x * 2

fn quadrat x
    x * x

for n in (range 1 6)
    print (n ++ " -> " ++ (to_str (verdopple n)) ++ " | " ++ (to_str (quadrat n)))
```

Ausgabe:
```
1 -> 2 | 1
2 -> 4 | 4
3 -> 6 | 9
4 -> 8 | 16
5 -> 10 | 25
```

## 1.10 Nächste Schritte

- [Sprachübersicht](02-sprachuebersicht.md) — Alle Sprachfeatures im Überblick
- [Builtin-Referenz](10-builtin-referenz.md) — Alle 115 eingebaute Funktionen
- [Kochrezepte](16-kochrezepte.md) — Praktische Code-Beispiele
