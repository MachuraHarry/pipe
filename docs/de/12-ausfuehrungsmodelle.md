# 12. Ausführungsmodelle

Pipe hat **zwei Ausführungs-Engines**, die denselben Code ausführen können.

## 12.1 Übersicht

| Eigenschaft | Tree-Walker | Bytecode-VM |
|------------|-------------|-------------|
| **Befehl** | `./bin/pipe datei.pipe` | `./bin/pipe -vm datei.pipe` |
| **Geschwindigkeit** | Basis (~1×) | ~7× schneller |
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
- **~7× schneller** als der Tree-Walker
- **Bytecode-Cache** (`.pipec`) — vermeidet wiederholtes Parsen/Kompilieren
- Geringerer Speicherverbrauch

### Nachteile
- **Nicht alle Features** in der VM implementiert:
  - `map`, `filter`, `reduce`, `each` mit benutzerdefinierten Funktionen
  - `for-in` Schleifen
  - `go` (nebenläufige Ausführung)
  - Tail Call Optimization
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

Benchmark-Ergebnisse (typische Werte):

| Benchmark | Tree-Walker | VM | Speedup |
|-----------|------------|-----|---------|
| Fibonacci(20) | ~15ms | ~2ms | ~7.5× |
| FizzBuzz 1-100 | ~3ms | ~0.5ms | ~6× |
| List Sum 10K | ~8ms | ~1ms | ~8× |

```bash
# Benchmark selbst ausführen
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
