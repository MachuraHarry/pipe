# 17. Migration von anderen Sprachen

## 17.1 Pipe vs Python

### Syntax-Vergleich

| Konzept | Python | Pipe |
|---------|--------|------|
| Kommentar | `#` | `--` |
| Variable | `x = 42` | `x: 42` |
| Funktion | `def f(x):` | `fn f x` |
| Funktionsaufruf | `print("Hallo")` | `print "Hallo"` |
| Return | `return x` | `x` (letzter Ausdruck) |
| Bedingung | `if x > 0:` | `if x > 0` |
| Pattern Matching | `match x:` (3.10+) | `match x` |
| Pipeline | ❌ (nur mit Libraries) | `> f > g > print` |
| Listen | `[1, 2, 3]` | `[1, 2, 3]` |
| Maps/Dicts | `{"a": 1}` | `{a: 1}` |
| Schleife | `for i in range(5):` | `for i in (range 5)` |
| Fehler | `try: ... except e:` | `try: ... catch e:` |
| Module | `import x` | `import "x.pipe"` |
| Klassen/OOP | ✅ | ❌ |
| List Comprehension | `[x*2 for x in l]` | `map l double` |
| Decorators | ✅ | ❌ |
| Async/Await | ✅ | ❌ (aber `go` für Nebenläufigkeit) |

### Side-by-Side

**Python:**
```python
def fib(n):
    if n <= 1:
        return n
    return fib(n-1) + fib(n-2)

print(fib(10))
```

**Pipe:**
```pipe
fn fib n
    match n
        | 0  -> 0
        | 1  -> 1
        | _  -> fib(n - 1) + fib(n - 2)

print (fib 10)
```

### Wann Pipe statt Python?

- Single-Binary-Deployment ohne venv/pip
- Pipeline-artige Datenverarbeitung
- Eingebettete Skripte in Go-Projekten
- Keine Notwendigkeit für OOP/Klassen

### Wann Python statt Pipe?

- Großes Ökosystem (400k+ Packages)
- Machine Learning/Data Science (NumPy, Pandas)
- Web-Frameworks (Django, Flask)
- Team-Projekte mit vielen Entwicklern

---

## 17.2 Pipe vs Lua

### Syntax-Vergleich

| Konzept | Lua | Pipe |
|---------|-----|------|
| Kommentar | `--` | `--` |
| Variable | `local x = 42` | `x: 42` |
| Funktion | `function f(x) end` | `fn f x` |
| Return | `return x` | `x` (letzter Ausdruck) |
| Blöcke | `then ... end`, `do ... end` | Einrückung |
| Tables/Maps | `{a = 1}` | `{a: 1}` |
| Pipeline | ❌ | `> f > g` |
| HTTP | ❌ (externes luasocket) | ✅ (eingebaut) |
| JSON | ❌ (extern) | ✅ (eingebaut) |
| Regex | ❌ (extern) | ✅ (eingebaut) |
| Metatabellen | ✅ | ❌ |
| Coroutinen | ✅ | ❌ (`go` als Alternative) |
| Binary-Größe | ~300 KB | ~10 MB |

### Side-by-Side

**Lua:**
```lua
function fib(n)
    if n == 0 then return 0
    elseif n == 1 then return 1
    else return fib(n-1) + fib(n-2)
    end
end

print(fib(10))
```

**Pipe:**
```pipe
fn fib n
    match n
        | 0  -> 0
        | 1  -> 1
        | _  -> fib(n - 1) + fib(n - 2)

print (fib 10)
```

### Wann Pipe statt Lua?

- Mehr Builtins ohne externe Abhängigkeiten
- Modernere Syntax (Einrückung statt `end`)
- HTTP/JSON/Regex direkt verfügbar
- Pipeline als First-Class Feature

### Wann Lua statt Pipe?

- Embedded Systems (minimaler Footprint, ~300 KB)
- C-Integration (Lua wurde dafür entworfen)
- LuaJIT-Performance (extrem schnell)
- Spiele-Entwicklung (LÖVE, Roblox)

---

## 17.3 Pipe vs Bash

### Syntax-Vergleich

| Konzept | Bash | Pipe |
|---------|------|------|
| Datenstrukturen | Nur Strings/Arrays | Listen, Maps, Strings, Numbers |
| Funktionen | `f() { ... }` | `fn f x` |
| JSON | ❌ (braucht `jq`) | ✅ (`parse_json`) |
| HTTP | ❌ (braucht `curl`) | ✅ (`http_get`) |
| Regex | 🟡 (`grep`/`sed`) | ✅ (`regex_match`) |
| Fehlerbehandlung | `set -e`, `trap` | `try`/`catch`, Result-Typ |
| Pipeline | Text-Streams | Strukturierte Werte |
| CLI-Befehle | Native | `exec` Funktion |

### Wann Pipe statt Bash?

- Komplexe Logik jenseits von "ein paar Befehle aneinanderreihen"
- Echte Datenstrukturen statt Text-Parsing
- JSON-APIs verarbeiten
- Fehlerbehandlung nötig
- Cross-Platform (keine Unix-Abhängigkeit)

### Wann Bash statt Pipe?

- Einfache Shell-Skripte (3-10 Zeilen)
- Direkte Pipe `|` zwischen Unix-Befehlen
- Überall verfügbar (kein zusätzlicher Interpreter nötig)

---

## 17.4 Pipe vs JavaScript/Node.js

### Syntax-Vergleich

| Konzept | JavaScript | Pipe |
|---------|-----------|------|
| Variable | `let x = 42` | `x: 42` |
| Funktion | `function f(x) { }` | `fn f x` |
| Arrow | `(x) => x * 2` | `double: fn x`<br>`    x * 2` |
| Objekte | `{a: 1, b: 2}` | `{a: 1, b: 2}` |
| Pipeline | ❌ | `> f > g > print` |
| Async | `async/await` | ❌ (`go` nur Tree-Walker) |
| NPM | ✅ (Mio. Packages) | ❌ |
| Pattern Matching | ❌ | ✅ (`match`) |
| Blöcke | `{ }` | Einrückung |
| Binary-Größe | ~80 MB (Node) | ~10 MB |

### Wann Pipe statt Node.js?

- Kleine bis mittlere Tools ohne NPM-Overhead
- Single-Binary-Deployment
- Einfachere Syntax für Skripte

### Wann Node.js statt Pipe?

- Web-Server mit Async-I/O (Express, Fastify)
- Riesiges Ökosystem
- Frontend/Backend mit gleicher Sprache
- Event-driven Architektur

---

## 17.5 Zusammenfassung

| Qualität | Pipe | Python | Lua | Bash | Node.js |
|----------|------|--------|-----|------|---------|
| Einfachheit | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐ | ⭐⭐⭐ |
| Ausdruckskraft | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐ | ⭐⭐⭐⭐ |
| Builtins | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐ | ⭐⭐ | ⭐⭐ |
| Performance | ⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐ | ⭐⭐⭐⭐ |
| Portabilität | ⭐⭐⭐⭐⭐ | ⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ |
| Ökosystem | ⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| Binärgröße | ~10 MB | ~30 MB | ~300 KB | ~1 MB | ~80 MB |
