# 9. Module und Importe

## 9.1 import

Mit `import` wird Code aus anderen Dateien geladen:

**lib.pipe:**
```pipe
fn quadrat x
    x * x

fn kubik x
    x * (quadrat x)
```

**main.pipe:**
```pipe
import "lib.pipe"

-- 49
print (quadrat 7)
-- 27
print (kubik 3)
```

Der Pfad ist **relativ zur aufrufenden Datei**.

### Import-Caching

Importierte Dateien werden nur **einmal** geparst und evaluiert —
auch wenn sie mehrfach importiert werden. Die Symbole werden in den
aktuellen Scope injiziert.

### Namespace-Import (as)

```pipe
import "lib.pipe" as lib

-- 49
print (lib.quadrat 7)
-- 27
print (lib.kubik 3)
```

Mit `as name` werden alle exportierten Symbole in eine Map verpackt,
auf die über den Alias zugegriffen wird.

## 9.2 export

Standardmäßig sind alle Symbole einer Datei beim Import sichtbar.
Mit `export` kann man explizit machen, welche Symbole exportiert werden:

**math.pipe:**
```pipe
export fn quadrat x
    x * x

export fn kubik x
    x * (quadrat x)

-- Nicht exportiert, von außen unsichtbar
fn intern x
    x + 1
```

**main.pipe:**
```pipe
import "math.pipe"

-- 25
print (quadrat 5)
-- 27
print (kubik 3)
-- print (intern 5)   -- FEHLER: nicht sichtbar
```

**Regel:** Wenn mindestens ein `export` in einer Datei existiert,
sind **nur** die exportierten Symbole von außen sichtbar.

## 9.3 PIPE_PATH

Die Umgebungsvariable `PIPE_PATH` definiert zusätzliche Suchpfade
für Importe (ähnlich wie `PYTHONPATH`):

```bash
export PIPE_PATH="/home/user/pipe-libs:/usr/local/share/pipe"
```

```pipe
-- Sucht in aktuellen Verzeichnis UND in PIPE_PATH
import "meine_lib.pipe"
```

Die Pfade werden mit `:` getrennt (wie bei `PATH`/`PYTHONPATH`).

## 9.4 Modul-Registry und Versionen

### 9.4.1 Registry

Pipe hat eine zentrale Modul-Registry: `github.com/MachuraHarry/pipe-modules`. Module können mit `pipe -search` entdeckt werden:

```bash
pipe -search log       -- log-Module finden
pipe -search ai        -- KI-Module finden
```

### 9.4.2 Module installieren

```bash
pipe -get log-analyzer               -- neueste Version
pipe -get log-analyzer@1.0.0         -- bestimmte Version
pipe -get https://.../module.pipe    -- via URL
```

Installierte Module liegen in `~/.pipe/modules/` und können per Name importiert werden.

### 9.4.3 Versionierte Imports

Mit `@version` wird eine bestimmte Version eines Moduls angefordert:

```pipe
-- exakte Version
import "log-analyzer@1.0[0]"
-- neueste (implizit @latest)
import "log-analyzer"
-- Version mit Alias
import "sentiment@0.9[0]" as s
```

Falls die lokale Datei nicht existiert, fragt Pipe die Registry nach der Versions-URL, lädt das Modul herunter und speichert es im Cache.

### 9.4.4 Registry-Format

```json
{
  "log-analyzer": {
    "latest": "1.0.0",
    "versions": {
      "1.0.0": "https://raw.../v1.0.0/module.pipe",
      "0.9.0": "https://raw.../v0.9.0/module.pipe"
    }
  }
}
```

- `latest` zeigt auf die empfohlene Version
- `versions` mappt Versions-Tags auf Modul-URLs
- Ohne `@version` wird automatisch `latest` verwendet
- Legacy-Format mit nur `url`-Feld wird ebenfalls unterstützt

## 9.5 Modul-Struktur

Empfohlene Projektstruktur:

```
mein_projekt/
├── main.pipe              -- Hauptprogramm
├── lib/
│   ├── math.pipe          -- Mathematische Hilfsfunktionen
│   ├── strings.pipe       -- String-Utilities
│   └── http.pipe          -- HTTP-Helfer
└── tests/
    └── test_math.pipe     -- Tests
```

## 9.6 Best Practices

### 1. Exportiere nur, was gebraucht wird

```pipe
-- math.pipe
export fn add a b
    a + b

export fn sub a b
    a - b

-- Interne Hilfsfunktionen nicht exportieren
fn validate x
    x > 0
```

### 2. Verwende Namespace-Imports bei großen Modulen

```pipe
import "http_utils.pipe" as http

print (http.get "https://example.com")
print (http.post "https://example.com" "{json}")
```

### 3. Gruppiere Imports am Dateianfang

```pipe
import "lib/math.pipe"
import "lib/strings.pipe"
import "lib/http.pipe" as http

-- Dann der eigentliche Code
print (quadrat 7)
```
