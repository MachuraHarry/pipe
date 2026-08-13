# VSCode Extension

The Pipe VSCode extension provides syntax highlighting, auto-completion, auto-indentation, and code folding for `.pipe` files. It is bundled in the repository under `vscode/`.

## Manual Installation

### Option 1: Copy to Extensions Directory

```sh
cp -r vscode/ ~/.vscode/extensions/pipe-lang.pipe-syntax-0.1.0/
```

Then restart VSCode or reload the window (`Ctrl+Shift+P` → "Developer: Reload Window").

### Option 2: Install from VSIX

1. Package the extension (see [Building the Extension](#building-the-extension)):

   ```sh
   make vsix
   ```

2. Press `Ctrl+Shift+P` and run "Extensions: Install from VSIX...", then select `vscode/pipe-syntax-0.9.5.vsix`.

### Option 3: Symlink (for development)

```sh
ln -s $(pwd)/vscode ~/.vscode/extensions/pipe-lang.pipe-syntax-0.1.0/
```

This keeps the extension in sync with the repository during development.

## Extension Details

| Property | Value |
|----------|-------|
| **Name** | `pipe-syntax` |
| **Display Name** | Pipe Language Support |
| **Publisher** | `pipe-lang` |
| **Version** | `0.1.0` |
| **Engine** | VSCode `>=1.85.0` |
| **Language ID** | `pipe` |
| **File Extension** | `.pipe` |
| **Scope Name** | `source.pipe` |
| **Categories** | Programming Languages |
| **Icon** | `icons/pipe-icon.png` |

## Syntax Highlighting Features

The TextMate grammar (`pipe.tmLanguage.json`) provides scoped highlighting for all Pipe language constructs.

### Comments
- `-- line comment` → `comment.line.double-dash.pipe`
- All text from `--` to end of line is colored as a comment

### Strings
- `"double-quoted strings"` → `string.quoted.double.pipe`
- `` `backtick strings` `` → `string.quoted.other.backtick.pipe`
- Escape sequences inside double-quoted strings (`\n`, `\t`, `\\`, `\"`) → `constant.character.escape.pipe`

### Keywords
- **Conditional**: `if`, `else if` → `keyword.control.conditional.pipe`
- **Control flow**: `else`, `match`, `while`, `for`, `in`, `try`, `catch`, `return`, `break`, `continue`, `defer`, `import` → `keyword.control.pipe`

### Constants
- `true`, `false` → `constant.language.boolean.pipe`
- `nil` → `constant.language.nil.pipe`
- `_` (wildcard in match patterns, after `|`) → `constant.language.wildcard.pipe`

### Built-in Functions (categorized by scope)

| Category | Scope Name | Functions |
|----------|-----------|-----------|
| I/O | `support.function.builtin.io.pipe` | `print`, `print_raw`, `input` |
| File | `support.function.builtin.file.pipe` | `read_file`, `write_file`, `append_file`, `read_lines`, `file_exists`, `file_delete`, `file_move`, `file_copy`, `file_size`, `file_type`, `list_dir`, `make_dir`, `remove_dir` |
| Path | `support.function.builtin.path.pipe` | `path_join`, `path_base`, `path_dir`, `path_ext` |
| String | `support.function.builtin.string.pipe` | `upper`, `lower`, `trim`, `split`, `join`, `contains` |
| List | `support.function.builtin.list.pipe` | `len`, `push`, `pop`, `at`, `sort`, `range`, `map`, `filter`, `reduce` |
| Map | `support.function.builtin.map.pipe` | `get`, `set`, `keys`, `values` |
| Math | `support.function.builtin.math.pipe` | `abs`, `min`, `max`, `pow`, `sqrt`, `round` |
| HTTP | `support.function.builtin.http.pipe` | `http_get`, `http_post`, `http_get_json`, `parse_json`, `to_json` |
| TCP | `support.function.builtin.tcp.pipe` | `tcp_listen`, `tcp_connect`, `tcp_accept`, `tcp_read`, `tcp_write`, `tcp_close` |
| System | `support.function.builtin.system.pipe` | `exec`, `env`, `sleep`, `now`, `format_time`, `random`, `random_range`, `base64_encode`, `base64_decode`, `type_of`, `is_num`, `is_str`, `is_list`, `is_map`, `is_nil`, `to_str`, `to_num`, `regex_match`, `regex_replace` |

### Variables
- **Assignment**: `name: value` → variable name is `variable.other.pipe`, colon is `keyword.operator.assignment.pipe`
- **Compound assignment**: `name += value` → variable name is `variable.other.pipe`, operator is `keyword.operator.assignment.compound.pipe`

### Numbers
- Integers: `42` → `constant.numeric.integer.pipe`
- Floats: `3.14` → `constant.numeric.float.pipe`

### Pipeline
- Vertical pipeline `> function` (at line start, after indentation) → `keyword.operator.pipeline.pipe`

### Match Patterns
- Arrow: `->` → `keyword.operator.arrow.pipe`
- Pattern separator: `|` (single) → `keyword.operator.pattern.pipe`

### Operators
- **Arithmetic**: `**`, `*`, `/`, `%`, `+`, `-` → `keyword.operator.arithmetic.pipe`
- **Comparison**: `==`, `!=`, `<=`, `>=`, `<`, `>` → `keyword.operator.comparison.pipe`
- **Logical**: `not`, `!`, `&&`, `||` → `keyword.operator.logical.pipe`
- **Concatenation**: `++` → `keyword.operator.concat.pipe`
- **Range**: `..` → `keyword.operator.range.pipe`

## Auto-Completion

Configured in `language-configuration.json`:

### Brackets
- `( )` — parentheses
- `[ ]` — square brackets
- `{ }` — curly braces

### Auto-Closing Pairs
These pairs automatically insert the closing character:
- `"..."` (not auto-closed inside existing strings)
- `` `...` `` (not auto-closed inside existing strings)
- `(...)` — parentheses
- `[...]` — square brackets
- `{...}` — curly braces

### Surrounding Pairs
When text is selected and a bracket key is pressed, the selection is wrapped:
- `(...)`, `[...]`, `{...}`, `"..."`, `` `...` ``

## Auto-Indentation Rules

### Increase Indent
Triggered when a line begins with one of these keywords (unless `else` follows on the same line):

```text
^\s*(if|while|for|fn|match|try|catch|defer)\b(?!.*\b(else)\b)
```

Meaning: after typing `if`, `while`, `for`, `fn`, `match`, `try`, `catch`, or `defer` (as the first keyword on a line), the next line will be auto-indented.

Example:
```pipe
if x > 0
    print "positive"   <- auto-indented
```

### Decrease Indent
Triggered when a line begins with `else`, `else if`, or `catch`:

```text
^\s*(else|else if|catch)\b
```

Example:
```pipe
if x > 0
    print "positive"
else                   -- auto-outdented
    print "not positive"
```

## Code Folding

The extension supports region-based code folding using comment markers:

- **Start region**: `-- region [name]`
- **End region**: `-- endregion [name]`

These markers follow the pattern:
- Start: `^\s*--\s*region`
- End: `^\s*--\s*endregion`

Example:
```pipe
-- region Initialization
x: 0
y: 1
z: 2
-- endregion Initialization
```

The entire block between `-- region` and `-- endregion` can be collapsed using VSCode's folding controls.

## Word Pattern

The word pattern for navigation (double-click selection, word-based cursor movement) is:

```
[A-Za-z_][A-Za-z0-9_]*
```

This means double-clicking on an identifier like `my_var123` selects the entire identifier, including underscores.

## IntelliSense (Language Server)

Since version 0.1.0 the extension ships an LSP client that connects to the
`pipe-lsp` server, providing full IntelliSense for `.pipe` files:

- **Auto-completion** — user functions, variables, parameters, all builtins, keywords and snippets
- **Hover documentation** — signatures and descriptions for builtins and user code
- **Signature help** — parameter lists while typing a call
- **Go-to-definition / Find references / Rename** — for user symbols
- **Diagnostics** — parse errors, undefined variables (E001), unused variables (E007)
- **Semantic highlighting** — tokens classified on top of the TextMate grammar
- **Format document** — reformats the whole file (`pipe formatter`)

### Building the Server

Build the `pipe-lsp` binary once:

```sh
make lsp          # or: go build -o bin/pipe-lsp ./cmd/pipe-lsp
```

The extension looks for the binary in this order:

1. the `pipe.lspPath` setting
2. `bin/pipe-lsp` inside the extension folder
3. `<workspace>/bin/pipe-lsp`
4. `pipe-lsp` on `PATH`

### Configuration

| Setting | Default | Description |
|---------|---------|-------------|
| `pipe.lspPath` | `""` | Absolute path to the `pipe-lsp` binary |
| `pipe.lsp.enabled` | `true` | Set to `false` to disable the language server |

## Extension File Structure

```
vscode/
├── icons/
│   ├── pipe-icon.png              # Extension and language icon
│   └── pipe-icon.svg              # Source vector icon
├── src/
│   └── extension.ts               # LSP client bootstrap
├── syntaxes/
│   └── pipe.tmLanguage.json       # TextMate grammar for syntax highlighting
├── language-configuration.json     # Bracket pairs, auto-indent, folding rules
├── package.json                    # Extension manifest
├── tsconfig.json                   # TypeScript compiler config
├── .vscodeignore                   # Files excluded from the .vsix package
├── README.md                       # Marketplace listing
├── LICENSE                         # MIT license
├── test-syntax.pipe                # Syntax test file (source)
└── pipe-syntax-0.9.5.vsix          # Packaged extension (built with `make vsix`)
```

## Building the Extension

```sh
cd vscode
npm install
npm run compile          # compiles src/ to out/
npm run build:server     # builds bin/pipe-lsp inside the extension
```

To produce an installable `.vsix` package (runs the prepublish build, then packages):

```sh
make vsix                # or: cd vscode && npx @vscode/vsce package
```

## Tested Features

The extension has been tested with the following VSCode features and confirmed working:

- **Syntax highlighting**: All token categories render with appropriate colors in VSCode's default themes
- **Bracket matching**: `()`, `[]`, `{}` pairs highlight correctly
- **Auto-closing pairs**: Brackets and quotes auto-close as expected
- **Auto-indentation**: Indent increase after control keywords, decrease on `else`/`catch`
- **Code folding**: `-- region` / `-- endregion` markers create foldable regions
- **Comment toggling**: `Ctrl+/` toggles `--` line comments
- **Word selection**: Double-click selects full identifiers including underscores
- **Bracket content**: Content inside `()`, `[]`, `{}` is not affected by INDENT/DEDENT logic

Pipe ships a Language Server Protocol (LSP) server (`cmd/pipe-lsp`, standard library only, no external dependencies). The VSCode extension connects to it automatically; see [IntelliSense (Language Server)](#intellisense-language-server) above.
