# VSCode Extension

The Pipe VSCode extension provides syntax highlighting, auto-completion, auto-indentation, and code folding for `.pipe` files. It is bundled in the repository under `vscode/`.

## Manual Installation

### Option 1: Copy to Extensions Directory

```sh
cp -r vscode/ ~/.vscode/extensions/pipe-lang.pipe-syntax-0.1.0/
```

Then restart VSCode or reload the window (`Ctrl+Shift+P` → "Developer: Reload Window").

### Option 2: Via VSCode Command

1. Copy the `vscode/` directory to a location on your system
2. Open VSCode
3. Press `Ctrl+Shift+P` and run "Extensions: Install from VSIX..."
4. Unfortunately Pipe currently doesn't ship a `.vsix` file, so Option 1 is recommended

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
| **Engine** | VSCode `>=1.80.0` |
| **Language ID** | `pipe` |
| **File Extension** | `.pipe` |
| **Scope Name** | `source.pipe` |
| **Categories** | Programming Languages |
| **Icon** | `icons/pipe-icon.svg` |

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
| I/O | `support.function.builtin.io.pipe` | `print`, `input` |
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

```regex
^\s*(if|while|for|fn|match|try|catch|defer)\b(?!.*\b(else)\b)
```

Meaning: after typing `if`, `while`, `for`, `fn`, `match`, `try`, `catch`, or `defer` (as the first keyword on a line), the next line will be auto-indented.

Example:
```pipe
if x > 0
    print "positive"   ← auto-indented
```

### Decrease Indent
Triggered when a line begins with `else`, `else if`, or `catch`:

```regex
^\s*(else|else if|catch)\b
```

Example:
```pipe
if x > 0
    print "positive"
else                   ← auto-outdented
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

## Extension File Structure

```
vscode/
├── icons/
│   └── pipe-icon.svg              # Language icon (light and dark)
├── syntaxes/
│   └── pipe.tmLanguage.json       # TextMate grammar for syntax highlighting
├── language-configuration.json     # Bracket pairs, auto-indent, folding rules
├── package.json                    # Extension manifest
├── test-syntax.pipe                # Syntax test file (source)
└── test-syntax.pipec               # Compiled test file
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

Pipe does not yet have a Language Server Protocol (LSP) implementation, so features like go-to-definition, find-references, hover information, and diagnostics are not yet available. These are planned for a future VSCode 2.0 extension (see [Roadmap](18-roadmap.md)).
