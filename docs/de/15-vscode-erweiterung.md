# 15. VSCode-Erweiterung

Pipe enthält eine **VSCode-Erweiterung** im Verzeichnis `vscode/` für
Syntax-Highlighting und Sprachunterstützung.

## 15.1 Installation

### Manuelle Installation

```bash
# In das VSCode-Erweiterungsverzeichnis kopieren
cp -r vscode/ ~/.vscode/extensions/pipe-lang.pipe-syntax-0.1.0/
```

### Installation als VSIX

```bash
make vsix   # paketiert vscode/pipe-syntax-0.1.0.vsix
```

Danach in VSCode `Ctrl+Shift+P` → "Extensions: Install from VSIX..." und die
Datei `vscode/pipe-syntax-0.1.0.vsix` auswählen.

### Entwicklung

In VSCode:
1. `Ctrl+Shift+P` → "Developer: Install Extension from Location..."
2. `vscode/`-Verzeichnis auswählen

### Extension-Details

| Eigenschaft | Wert |
|------------|------|
| Name | `pipe-syntax` |
| Publisher | `pipe-lang` |
| Version | 0.1.0 |
| VSCode Engine | ^1.85.0 |
| Language ID | `pipe` |
| Dateiendung | `.pipe` |

## 15.2 Features

### Syntax-Highlighting

Die Extension erkennt und färbt:

- **Kommentare:** `-- ...` (grau/grün)
- **Strings:** `"..."` und `` `...` `` (orange)
- **Keywords:** `fn`, `match`, `if`, `else`, `while`, `for`, `in`, `try`, `catch`, `return`, `break`, `continue`, `defer`, `import`, `export`, `enum`
- **Konstanten:** `true`, `false`, `nil`, `_` (blau)
- **Builtins:** Alle 115 Funktionen in Kategorien:
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
    -- code...
-- endregion
```

### Wort-Muster

Erkennt Pipe-Identifier: `[A-Za-z_][A-Za-z0-9_]*`

## 15.3 IntelliSense (Language Server)

Seit Version 0.1.0 bringt die Extension einen LSP-Client mit, der sich mit dem
`pipe-lsp`-Server verbindet und volle IntelliSense für `.pipe`-Dateien liefert:

- **Auto-Vervollständigung** — eigene Funktionen, Variablen, Parameter, alle Builtins, Keywords und Snippets
- **Hover-Dokumentation** — Signaturen und Beschreibungen für Builtins und eigenen Code
- **Signatur-Hilfe** — Parameterlisten während der Eingabe eines Aufrufs
- **Gehe zu Definition / Referenzen / Umbenennen** — für eigene Symbole
- **Diagnosen** — Parse-Fehler, undefinierte Variablen (E001), ungenutzte Variablen (E007)
- **Semantische Hervorhebung** — Tokens zusätzlich zur TextMate-Grammatik klassifiziert
- **Dokument formatieren** — formatiert die gesamte Datei (`pipe formatter`)

### Server bauen

Das `pipe-lsp`-Binary einmalig bauen:

```sh
make lsp          # oder: go build -o bin/pipe-lsp ./cmd/pipe-lsp
```

Die Extension sucht das Binary in dieser Reihenfolge:

1. die Einstellung `pipe.lspPath`
2. `bin/pipe-lsp` im Extension-Ordner
3. `<workspace>/bin/pipe-lsp`
4. `pipe-lsp` auf `PATH`

### Konfiguration

| Einstellung | Standard | Beschreibung |
|-------------|----------|--------------|
| `pipe.lspPath` | `""` | Absoluter Pfad zum `pipe-lsp`-Binary |
| `pipe.lsp.enabled` | `true` | Auf `false` setzen, um den Language Server zu deaktivieren |

## 15.4 Dateien der Extension

```
vscode/
├── package.json                     -- Extension-Manifest
├── language-configuration.json      -- Sprach-Konfiguration
├── src/
│   └── extension.ts                 -- LSP-Client-Bootstrap
├── syntaxes/
│   └── pipe.tmLanguage.json        -- TextMate-Grammatik
├── icons/
│   ├── pipe-icon.png               -- Extension- und Sprach-Icon
│   └── pipe-icon.svg               -- SVG-Quelle
├── README.md                        -- Marketplace-Listing
├── LICENSE                          -- MIT-Lizenz
├── .vscodeignore                    -- Dateien, die aus dem VSIX ausgeschlossen sind
├── tsconfig.json                    -- TypeScript-Konfiguration
└── test-syntax.pipe                 -- Syntax-Testdatei
```

## 15.5 Getestete Features

Die Extension wurde mit einer `test-syntax.pipe` Datei getestet, die alle
Sprachfeatures abdeckt.

Pipe besitzt seit Version 0.1.0 eine LSP-Implementierung (`cmd/pipe-lsp`,
reine Standardbibliothek, ohne externe Abhängigkeiten). Die Extension verbindet
sich automatisch damit; siehe [15.3 IntelliSense (Language Server)](#153-intellisense-language-server).
