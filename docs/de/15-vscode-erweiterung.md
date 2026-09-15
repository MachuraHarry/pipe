# 15. VSCode-Erweiterung

Pipe enthält eine **VSCode-Erweiterung** im Verzeichnis `vscode/` für
Syntax-Highlighting und Sprachunterstützung.

## 15.1 Installation

### Manuelle Installation

```bash
# In das VSCode-Erweiterungsverzeichnis kopieren
cp -r vscode/ ~/.vscode/extensions/pipe-lang.pipe-syntax-1.3.0/
```

### Installation als VSIX

```bash
make vsix   # paketiert vscode/pipe-syntax-1.3.0.vsix
```

Danach in VSCode `Ctrl+Shift+P` → "Extensions: Install from VSIX..." und die
Datei `vscode/pipe-syntax-1.3.0.vsix` auswählen.

### Entwicklung

In VSCode:
1. `Ctrl+Shift+P` → "Developer: Install Extension from Location..."
2. `vscode/`-Verzeichnis auswählen

### Extension-Details

| Eigenschaft | Wert |
|------------|------|
| Name | `pipe-syntax` |
| Publisher | `pipe-lang` |
| Version | 1.3.0 |
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
- **Builtins:** Alle 168 Funktionen in Kategorien:
  - IO: `print`, `input`
  - Dateisystem: `read_file`, `write_file`, `append_file`, `read_lines`, `file_exists`, `file_delete`, `file_move`, `file_copy`, `file_size`, `file_type`, `list_dir`, `make_dir`, `remove_dir`
  - Pfad: `path_join`, `path_base`, `path_dir`, `path_ext`
  - Strings: `upper`, `lower`, `trim`, `split`, `join`, `contains`
  - Listen: `len`, `push`, `pop`, `at`, `sort`, `range`, `map`, `filter`, `reduce`
  - Maps: `get`, `set`, `keys`, `values`
  - Mathematik: `abs`, `min`, `max`, `pow`, `sqrt`, `round`
  - HTTP: `http_get`, `http_post`, `http_get_json`, `parse_json`, `to_json`
  - TCP: `tcp_listen`, `tcp_connect`, `tcp_connect_tls`, `tcp_accept`, `tcp_read`, `tcp_read_bytes`, `tcp_write`, `tcp_close`
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

Die Extension bringt einen LSP-Client mit, der sich mit dem
`pipe-lsp`-Server verbindet und volle IntelliSense für `.pipe`-Dateien liefert:

- **Auto-Vervollständigung** — eigene Funktionen, Variablen, Parameter, alle Builtins, Keywords und Snippets
- **Hover-Dokumentation** — Signaturen und Beschreibungen für Builtins und eigenen Code
- **Signatur-Hilfe** — Parameterlisten während der Eingabe eines Aufrufs
- **Gehe zu Definition / Referenzen / Umbenennen** — für eigene Symbole
- **Diagnosen** — Parse-Fehler, undefinierte Variablen (E001), ungenutzte Variablen (E007)
- **Semantische Hervorhebung** — Tokens zusätzlich zur TextMate-Grammatik klassifiziert
- **Dokument formatieren** — formatiert die gesamte Datei (`pipe formatter`)

### Bei jedem Speichern formatieren

`pipe-lsp` implementiert LSP-Dokumentformatierung, daher funktioniert **Format Document** (`Shift+Alt+F`) ohne weitere Einrichtung. Um das automatisch bei jedem Speichern auszulösen, eine sprachspezifische VSCode-Einstellung ergänzen (eine eingebaute Editor-Einstellung, keine Pipe-spezifische):

```json
{
  "[pipe]": {
    "editor.formatOnSave": true
  }
}
```

## 15.3a Snippets

Die Extension liefert Tab-vervollständigbare Snippets für häufige Pipe-Muster: `fn`, `fnlit`, `if`, `ifelse`, `match`, `forin`, `while`, `trycatch`, `aitool` (registriert ein `ai_tool`), `mcpserver` (`mcp_server` + `mcp_serve_stdio`), `swarmagent` (`swarm_agent`-Registrierung) und `sandboxprofile` (ein `sandbox_profile`-Block). Präfix in einer `.pipe`-Datei eingeben und den Vorschlag annehmen, oder mit `Tab` expandieren.

## 15.3c Debugger

Die Extension registriert einen `pipe`-Debug-Adapter (`cmd/pipe-dap`, reine Standardbibliothek, spricht das Debug Adapter Protocol über stdio), sodass VS Codes eingebaute Run-and-Debug-Ansicht für `.pipe`-Dateien funktioniert: Breakpoints per Klick am Rand setzen, `F5` drücken, und die üblichen Continue/Step-Over/Step-Into/Step-Out-Steuerelemente verwenden. Das Variablen-Panel zeigt lokale Variablen (inklusive Parameter) und Globals namentlich; das Call-Stack-Panel zeigt Frame-Namen und Zeilen.

Debugging läuft immer über die Bytecode-VM (dasselbe Backend wie `Pipe: Run File (VM Mode)`), da nur die VM eine präzise Quellzeile pro Instruktion nachverfolgt.

**Umfang und bekannte Einschränkungen (Stage 1):**
- Nur die eine Top-Level-VM eines Debug-Launches wird pausiert/gesteppt. Mit `spawn` gestarteter Code läuft in einer eigenen, unabhängigen VM und ist **nicht** debugbar — er läuft ungehindert weiter, auch während der Hauptthread an einem Breakpoint pausiert ist.
- Noch keine bedingten Breakpoints, Logpoints oder Watch-Ausdrücke.
- Keine `.vscode/launch.json` nötig für den Normalfall — `F5` auf einer offenen `.pipe`-Datei bietet eine Standardkonfiguration "Run current Pipe file". Für Anpassungen (z.B. `stopOnEntry`) einen `pipe`-Eintrag in `.vscode/launch.json` ergänzen:

  ```jsonc
  {
    "type": "pipe",
    "request": "launch",
    "name": "Run current Pipe file",
    "program": "${file}",
    "stopOnEntry": false
  }
  ```

Das Debug-Adapter-Binary einmalig bauen:

```sh
make dap          # oder: go build -o bin/pipe-dap ./cmd/pipe-dap
```

Die Extension löst es genauso auf wie das `pipe`-CLI (es wird nicht mit der Extension ausgeliefert): die Einstellung `pipe.dapPath`, dann `<workspace>/bin/pipe-dap`, dann `pipe-dap` auf `PATH`.

## 15.3b Befehle

Über die Befehlspalette (`Ctrl+Shift+P` / `Cmd+Shift+P`), Kategorie "Pipe":

| Befehl | Standard-Tastenkürzel | Beschreibung |
|--------|------------------------|--------------|
| `Pipe: Run File` | `Ctrl+Alt+R` / `Cmd+Alt+R` | Speichert und führt die aktive `.pipe`-Datei mit dem Tree-Walker in einem integrierten Terminal aus |
| `Pipe: Run File (VM Mode)` | — | Wie oben, aber mit `pipe -vm` (Bytecode-VM) |
| `Pipe: Open REPL` | — | Öffnet ein integriertes Terminal mit der `pipe`-REPL |

Diese Befehle lösen das `pipe`-CLI-Binary unabhängig vom `pipe-lsp`-Server auf — siehe `pipe.cliPath` unten.

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

`Pipe: Run File`/`Pipe: Open REPL` lösen das `pipe`-CLI-Binary separat auf (es wird nicht mit der Extension ausgeliefert): die Einstellung `pipe.cliPath`, dann `<workspace>/bin/pipe`, dann `pipe` auf `PATH`.

### Konfiguration

| Einstellung | Standard | Beschreibung |
|-------------|----------|--------------|
| `pipe.lspPath` | `""` | Absoluter Pfad zum `pipe-lsp`-Binary |
| `pipe.lsp.enabled` | `true` | Auf `false` setzen, um den Language Server zu deaktivieren |
| `pipe.cliPath` | `""` | Absoluter Pfad zum `pipe`-CLI-Binary, genutzt von `Pipe: Run File`/`Pipe: Open REPL`. Leer = automatische Erkennung (`<workspace>/bin/pipe`, dann `PATH`) |
| `pipe.dapPath` | `""` | Absoluter Pfad zum `pipe-dap`-Debug-Adapter-Binary. Leer = automatische Erkennung (`<workspace>/bin/pipe-dap`, dann `PATH`) |

## 15.4 Dateien der Extension

```
vscode/
├── package.json                     -- Extension-Manifest
├── language-configuration.json      -- Sprach-Konfiguration
├── src/
│   ├── extension.ts                 -- Aktivierung: LSP-Client + Befehls- + Debug-Adapter-Registrierung
│   ├── serverPath.ts                -- Auflösung des pipe-lsp-Binaries
│   ├── cliPath.ts                   -- Auflösung des pipe-CLI-Binaries (Run File/Open REPL)
│   ├── dapPath.ts                   -- Auflösung des pipe-dap-Binaries (Debugger)
│   ├── debugAdapter.ts              -- Registrierung der DebugAdapterDescriptorFactory
│   └── commands.ts                  -- Run File / Run File (VM) / Open REPL
├── snippets/
│   └── pipe.json                    -- Tab-vervollständigbare Snippets
├── syntaxes/
│   └── pipe.tmLanguage.json        -- TextMate-Grammatik
├── test/
│   ├── serverPath.test.ts           -- vitest: pipe-lsp-Auflösung
│   ├── cliPath.test.ts              -- vitest: pipe-CLI-Auflösung
│   ├── dapPath.test.ts              -- vitest: pipe-dap-Auflösung
│   └── snippets.test.ts             -- vitest: Snippet-JSON-Form
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

Pipe besitzt eine LSP-Implementierung (`cmd/pipe-lsp`,
reine Standardbibliothek, ohne externe Abhängigkeiten). Die Extension verbindet
sich automatisch damit; siehe [15.3 IntelliSense (Language Server)](#153-intellisense-language-server).

Der Debug-Adapter (`cmd/pipe-dap`) hat automatisierte Protokoll-Abdeckung — ein Go-Test-Harness
spielt rohes DAP-JSON über ein `io.Pipe()` gegen seinen Request-Handler und deckt Breakpoint-Treffer,
Stack-Traces, Variableninspektion, Stepping, `stopOnEntry` und Disconnect ab — aber ein manueller
Test über VS Codes tatsächliche Run-and-Debug-Oberfläche (`F5`) wurde nicht durchgeführt, da dies in
einer Headless-Umgebung ohne verfügbare VS-Code-Instanz gebaut wurde. Das `F5`-Erlebnis gilt als
ungetestet, bis es jemand manuell verifiziert.
