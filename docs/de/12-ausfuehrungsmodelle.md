# 12. Ausführungsmodelle

Pipe hat **zwei Ausführungs-Engines**, die denselben Code ausführen können.

## 12.1 Übersicht

| Eigenschaft | Tree-Walker | Bytecode-VM |
|------------|-------------|-------------|
| **Befehl** | `./bin/pipe datei.pipe` | `./bin/pipe -vm datei.pipe` |
| **Geschwindigkeit** | Basis (~1×) | 0,6×–55× je nach Workload (rekursionslastig bis ~55×, einfache Schleifen vergleichbar oder langsamer) |
| **Alle Features** | Ja | Nein (Subset) |
| **Bytecode-Cache** | Nein | Ja (`.pipec`) |
| **map/filter/reduce (User-Fn)** | Ja | Nein |
| **for-in** | Ja | Nein |
| **try/catch** | Ja | Ja |
| **Defer** | Ja | Ja |
| **Tail Call Optimization** | Ja | Nein |

## 12.2 Tree-Walker

Der **Tree-Walker** ist die Standard-Engine. Er evaluiert den AST
rekursiv — jeder Knoten wird direkt interpretiert.

### Vorteile
- **Alle Sprachfeatures** verfügbar
- **Tail Call Optimization** für tiefe Rekursion
- **Höhere Funktionen** (map, filter, reduce mit User-Funktionen)
- **for-in** Schleifen
- Einfachere Fehlermeldungen (direkt aus dem AST)

### Nachteile
- Langsamer (interpretiert, nicht kompiliert)
- Kein Bytecode-Cache

### Interner Ablauf
```
Quelltext → Lexer → Parser → AST → Evaluator → Ausgabe
```

## 12.3 Bytecode-VM

Die **VM** kompiliert den AST zu Bytecode und führt ihn auf einer
Stack-Maschine aus.

### Vorteile
- **Schneller bei rekursions- und aufruflastigem Code** — gemessen bis ~55× (fib(20)); einfache Schleifen und String-Ops vergleichbar oder langsamer (0,6×–3×)
- **Bytecode-Cache** (`.pipec`) — vermeidet wiederholtes Parsen/Kompilieren
- Geringerer Speicherverbrauch

### Nachteile
- **Nicht alle Features** in der VM implementiert:
  - `map`, `filter`, `reduce`, `each` mit benutzerdefinierten Funktionen
  - `for-in` Schleifen
  - `go` (nebenläufige Ausführung)
  - Tail Call Optimization
- `>>` Parallel-Pipeline funktioniert in der VM für Builtins (KI, I/O), fällt für Closures synchron zurück
- Komplexere Fehlermeldungen

### Interner Ablauf
```
Quelltext → Lexer → Parser → AST → Compiler → Bytecode → VM → Ausgabe
                                                        ↓
                                                   .pipec Cache
```

## 12.4 Wann welchen Modus?

| Anwendungsfall | Empfohlener Modus |
|----------------|-------------------|
| Entwicklung / Debugging | Tree-Walker |
| Kleine Skripte (< 100 LOC) | Tree-Walker |
| Produktions-Skripte | Bytecode-VM |
| Performance-kritische Anwendungen | Bytecode-VM |
| Scripting mit map/filter/reduce | Tree-Walker |
| Self-extracting Binary (`-build`) | Bytecode-VM |
| REPL interaktiv | Beide (umschaltbar) |

## 12.5 Performance-Vergleich

Gemessene Benchmark-Ergebnisse (aus `pipe -bench`, reine Ausführungszeit, Durchschnitt aus 5 Läufen):

| Benchmark | Tree-Walker | VM | Speedup |
|-----------|------------|-----|---------|
| fib(20) rekursiv | 366ms | 6.7ms | 55× |
| list sum 10000 | 4.5ms | 1.5ms | 3.0× |
| list push + sum 20000 | 15.7ms | 18.2ms | 0.9× |
| FizzBuzz 1-100 | 0.22ms | 0.34ms | 0.7× |
| String-Concat 20000 | 85ms | 150ms | 0.6× |

Die VM gewinnt deutlich bei rekursionslastigem Code (bis ~55×), aber einfache Schleifen, Listen-Aufbau und String-Concatenation sind vergleichbar oder sogar langsamer. Eigenen Workload messen mit:

```bash
./bin/pipe -bench
```

## 12.6 VM-Modus aktivieren

```bash
# Einmalig
./bin/pipe -vm script.pipe

# Ohne Bytecode-Debug-Ausgabe
./bin/pipe -vm -q script.pipe

# In der REPL umschalten
>>> :vm
  VM-Modus: ein
```

## 12.7 Feature-Matrix (vollständig)

| Feature | Tree-Walker | VM |
|---------|:---------:|:--:|
| Variablen | ✅ | ✅ |
| Compound Assignment | ✅ | ✅ |
| Arithmetik | ✅ | ✅ |
| Vergleiche | ✅ | ✅ |
| Logik (!, &&, \|\|) | ✅ | ✅ |
| Strings (++) | ✅ | ✅ |
| if/else | ✅ | ✅ |
| match | ✅ | ✅ |
| while | ✅ | ✅ |
| break/continue | ✅ | ✅ |
| for-in | ✅ | ❌ |
| Funktionen | ✅ | ✅ |
| Closures | ✅ | ✅ |
| Rekursion | ✅ | ✅ |
| Tail Call Optimization | ✅ | ❌ |
| Pipeline | ✅ | ✅ |
| `>>` Parallel-Pipeline | ✅ | 🟡 (Builtins) |
| Listen (Index, Slice) | ✅ | ✅ |
| Maps | ✅ | ✅ |
| Dot-Access | ✅ | ✅ |
| Anonyme Funktionen | ✅ | ✅ |
| map/filter/reduce/each (Builtin-Fn) | ✅ | ✅ |
| map/filter/reduce/each (User-Fn) | ✅ | ❌ |
| try/catch | ✅ | ✅ |
| return | ✅ | ✅ |
| defer | ✅ | ✅ |
| import/export | ✅ | ✅ |
| enum | ✅ | ✅ |
| Stack-Traces | ✅ | ✅ |
| go (Nebenläufig) | ✅ | ❌ |
