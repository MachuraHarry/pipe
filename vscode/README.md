# Pipe Language Support

Syntax highlighting and full IntelliSense (LSP) for the **Pipe** scripting language (`.pipe` files).

> **Pipe** is a Semantic Pipeline Runtime (SPR) — a pipeline-based execution environment where AI operations (`summarize`, `translate`, `classify`) are language primitives. See the [Pipe repository](https://github.com/MachuraHarry/pipe) for the language itself, the compiler, and the full documentation.

## Features

- **Syntax highlighting** — TextMate grammar covering comments, strings, keywords, operators, pipelines, match patterns and categorized builtins
- **IntelliSense (language server)** — auto-completion, hover docs, signature help, go-to-definition, find references, rename
- **Diagnostics** — parse errors, undefined and unused variables
- **Semantic highlighting** — token classification on top of the grammar
- **Format document** — reformats the whole file (see [Format on Save](#format-on-save) to run it automatically)
- **Snippets** — `fn`, `fnlit`, `if`, `ifelse`, `match`, `forin`, `while`, `trycatch`, `aitool`, `mcpserver`, `swarmagent`, `sandboxprofile`
- **Commands** — `Pipe: Run File`, `Pipe: Run File (VM Mode)`, `Pipe: Open REPL` (Command Palette; `Run File` also bound to `Ctrl+Alt+R` / `Cmd+Alt+R`)
- **Auto-completion** — brackets `() [] {}`, quotes `"` and `` ` ``, auto-indent after control keywords, outdent on `else`/`catch`
- **Code folding** — via `-- region` / `-- endregion` markers

## Requirements

- VSCode `^1.85.0`
- The `pipe-lsp` server binary (ships prebuilt for **linux x64** in this package)

On other platforms, build the server once and point the extension to it:

```sh
make lsp   # from the Pipe repository, or: go build -o bin/pipe-lsp ./cmd/pipe-lsp
```

## Configuration

| Setting | Default | Description |
|---------|---------|-------------|
| `pipe.lspPath` | `""` | Absolute path to the `pipe-lsp` binary. Empty = auto-detect. |
| `pipe.lsp.enabled` | `true` | Set to `false` to disable the language server. |
| `pipe.cliPath` | `""` | Absolute path to the `pipe` CLI binary, used by **Run File**/**Open REPL**. Empty = auto-detect. |

Auto-detection order for the server binary:

1. the `pipe.lspPath` setting
2. `bin/pipe-lsp` inside the extension folder
3. `<workspace>/bin/pipe-lsp`
4. `pipe-lsp` on `PATH`

Auto-detection order for the `pipe` CLI binary (used by Run File/Open REPL only — the CLI itself isn't bundled with the extension):

1. the `pipe.cliPath` setting
2. `<workspace>/bin/pipe`
3. `pipe` on `PATH`

### Format on Save

`pipe-lsp` already implements LSP document formatting, so **Format Document** (`Shift+Alt+F`) works out of the box. To run it automatically on save, add a language-scoped setting — this is a built-in VS Code setting, not something this extension configures for you:

```json
{
  "[pipe]": {
    "editor.formatOnSave": true
  }
}
```

## Building from Source

```sh
git clone https://github.com/MachuraHarry/pipe.git
cd pipe
make lsp                        # build the server
cd vscode
npm install
npm run compile                 # compile src/ to out/
npm run build:server            # build bin/pipe-lsp into the extension
```

Then run VSCode with `--extensionDevelopmentPath=path/to/pipe/vscode` (F5 in the `vscode/` folder).

## Known Limitations

- Prebuilt `pipe-lsp` binaries are **linux x64 only**; other platforms must build from source (Go standard library only, no external dependencies).
- Rename requires a valid identifier (Pipe validation rules apply).

## License

MIT — see [LICENSE](LICENSE).
