# 15. VSCode-Erweiterung

Pipe enthält eine **VSCode-Erweiterung** im Verzeichnis `vscode/` für
Syntax-Highlighting und Sprachunterstützung.

## 15.1 Installation

### Manuelle Installation

```bash
# In das VSCode-Erweiterungsverzeichnis kopieren
cp -r vscode/ ~/.vscode/extensions/pipe-lang.pipe-syntax-0.1.0/
```

Oder in VSCode:
1. `Ctrl+Shift+P` → "Developer: Install Extension from Location..."
2. `vscode/`-Verzeichnis auswählen

### Extension-Details

| Eigenschaft | Wert |
|------------|------|
| Name | `pipe-syntax` |
| Publisher | `pipe-lang` |
| Version | 0.1.0 |
| VSCode Engine | ^1.80.0 |
| Language ID | `pipe` |
| Dateiendung | `.pipe` |

## 15.2 Features

### Syntax-Highlighting

Die Extension erkennt und färbt:

- **Kommentare:** `-- ...` (grau/grün)
- **Strings:** `"..."` und `` `...` `` (orange)
- **Keywords:** `fn`, `match`, `if`, `else`, `while`, `for`, `in`, `try`, `catch`, `return`, `break`, `continue`, `defer`, `import`, `export`, `enum`
- **Konstanten:** `true`, `false`, `nil`, `_` (blau)
- **Builtins:** Alle 80+ Funktionen in Kategorien:
  - IO: `print`, `input`
  - Dateisystem: `read_file`, `write_file`, `append_file`, `read_lines`, `file_exists`, `file_delete`, `file_move`, `file_copy`, `file_size`, `file_type`, `list_dir`, `make_dir`, `remove_dir`
  - Pfad: `path_join`, `path_base`, `path_dir`, `path_ext`
  - Strings: `upper`, `lower`, `trim`, `split`, `join`, `contains`
  - Listen: `len`, `push`, `pop`, `at`, `sort`, `range`, `map`, `filter`, `reduce`
  - Maps: `get`, `set`, `keys`, `values`
  - Mathematik: `abs`, `min`, `max`, `pow`, `sqrt`, `round`
  - HTTP: `http_get`, `http_post`, `http_get_json`, `parse_json`, `to_json`
  - TCP: `tcp_listen`, `tcp_connect`, `tcp_accept`, `tcp_read`, `tcp_write`, `tcp_close`
  - Sonstige: `exec`, `env`, `sleep`, `now`, `format_time`, `random`, `random_range`, `base64_encode`, `base64_decode`, `regex_match`, `regex_replace`, `type_of`, `is_num`, `is_str`, `is_list`, `is_map`, `is_nil`, `to_str`, `to_num`
- **Variablenzuweisung:** `name:` Muster
- **Zahlen:** Integer und Float Literale
- **Pipeline:** `>` am Zeilenanfang mit Einrückung
- **Match-Muster:** `|` und `->` Syntax
- **Operatoren:** `+`, `-`, `*`, `/`, `%`, `**`, `==`, `!=`, `<`, `>`, `<=`, `>=`, `!`, `&&`, `||`, `++`, `..`, `+=`, `-=`, `*=`, `/=`, `%=`

### Auto-Vervollständigung

- **Klammern:** `()`, `[]`, `{}`
- **Anführungszeichen:** `""`, `\`\``
- **Umrahmende Paare** für alle Klammern und Anführungszeichen

### Auto-Einrückung

Die Extension erhöht die Einrückung nach:
`if`, `while`, `for`, `fn`, `match`, `try`, `catch`, `defer`

(Ausnahme: `else` und `else if` erhöhen nicht)

Verringert Einrückung nach:
`else`, `else if`, `catch`

### Code-Faltung

Unterstützt Folding-Marker:
```pipe
-- region Beschreibung
...code...
-- endregion
```

### Wort-Muster

Erkennt Pipe-Identifier: `[A-Za-z_][A-Za-z0-9_]*`

## 15.3 Dateien der Extension

```
vscode/
├── package.json                     -- Extension-Manifest
├── language-configuration.json      -- Sprach-Konfiguration
├── syntaxes/
│   └── pipe.tmLanguage.json        -- TextMate-Grammatik
├── icons/
│   └── pipe-icon.svg               -- Extension-Icon
├── test-syntax.pipe                 -- Syntax-Testdatei
└── test-syntax.pipec               -- Kompilierte Testdatei
```

## 15.4 Getestete Features

Die Extension wurde mit einer `test-syntax.pipe` Datei getestet, die alle
Sprachfeatures abdeckt. Eine kompilierte Version (`test-syntax.pipec`) liegt bei.
