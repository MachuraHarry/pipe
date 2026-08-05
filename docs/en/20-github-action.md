# 20. GitHub Action

Run Pipe scripts directly in your GitHub Actions workflows — no installation needed.

## Basic Usage

```yaml
name: Pipe Demo
on: [push]
jobs:
  pipe:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: MachuraHarry/pipe/pipe-action@master
        with:
          script: |
            print "Hello from CI/CD!"
            print (2 + 2)
```

## Inputs

| Input | Required | Default | Description |
|-------|----------|---------|-------------|
| `script` | one of `script`/`file` | — | Inline Pipe code to execute |
| `file` | one of `script`/`file` | — | Path to a `.pipe` file in the repo to execute |
| `args` | ❌ | — | Additional arguments passed to the script (available as `$args`) |
| `flags` | ❌ | `-vm -q` | Extra CLI flags (e.g. `--sandbox`, `--sandbox-profile networked`) |
| `version` | ❌ | `latest` | Pipe version (release tag like `v0.7.0`) to download |
| `provider` | ❌ | — | AI provider (`openai`, `anthropic`, `deepseek`, `ollama`); injected via `--ai-provider` when `file` is used |

Sandbox flags (`--sandbox`, `--sandbox-profile networked`, `--allow-ai`) are passed via the `flags` input.

## Examples

### Hello World

```yaml
- uses: MachuraHarry/pipe/pipe-action@master
  with:
    script: print "hello from pipe-action"
```

### Process a File (after checkout)

```yaml
- uses: actions/checkout@v4
- uses: MachuraHarry/pipe/pipe-action@master
  with:
    script: |
      data: read_file "CHANGELOG.md"
      print data
```

### AI-Powered Release Notes

```yaml
- uses: MachuraHarry/pipe/pipe-action@master
  with:
    script: |
      log: exec "git log --oneline -20"
      print (get log "output")
  env:
    OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
```

### Classify with DeepSeek

```yaml
- uses: MachuraHarry/pipe/pipe-action@master
  with:
    script: |
      cats: ["bug", "feature", "question"]
      "the app crashes on submit"
          > classify cats
          > print
    ai-provider: deepseek
    sandbox: true
    allow-ai: true
  env:
    DEEPSEEK_API_KEY: ${{ secrets.DEEPSEEK_API_KEY }}
```

### Sandboxed

```yaml
- uses: MachuraHarry/pipe/pipe-action@master
  with:
    script: print "safe execution"
    flags: '--sandbox'
```

### Run a Script from the Repo with Arguments

```yaml
- uses: ./.github/actions/pipe-action
  with:
    file: examples/commit_summary.pipe
    args: '20 commit-summary.md'
  env:
    DEEPSEEK_API_KEY: ${{ secrets.DEEPSEEK_API_KEY }}
```

### German Commit Digest on Every Push

The repo ships a ready-made workflow (`.github/workflows/commit-digest.yml`) that summarizes the latest commits in German, uploads the digest as an artifact and comments it on pull requests:

```yaml
on:
  push:
    branches: [master]
  pull_request:
permissions:
  contents: read
  pull-requests: write
jobs:
  digest:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: ./.github/actions/pipe-action
        with:
          file: examples/commit_summary.pipe
          args: '20 commit-summary.md'
        env:
          DEEPSEEK_API_KEY: ${{ secrets.DEEPSEEK_API_KEY }}
      - uses: actions/upload-artifact@v4
        with:
          name: commit-summary-de
          path: commit-summary.md
```

## Source

`pipe-action/action.yml`, `pipe-action/run.sh`
