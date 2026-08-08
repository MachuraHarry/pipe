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
      - uses: MachuraHarry/pipe/.github/actions/pipe-action@master
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
| `flags` | ❌ | `-vm -q` | Extra CLI flags (e.g. `--sandbox`, `--sandbox-profile networked`) |
| `version` | ❌ | `latest` | Pipe version (release tag like `v0.8.0`) to download |
| `provider` | ❌ | — | AI provider (`openai`, `anthropic`, `deepseek`, `ollama`); injected via `--ai-provider` when `file` is used |

Script parameters are passed via environment variables (e.g. `SUMMARY_COUNT`, `SUMMARY_OUTPUT`) instead of CLI arguments, since the current release binary does not support `args` in VM mode.

Sandbox flags (`--sandbox`, `--sandbox-profile networked`, `--allow-ai`) are passed via the `flags` input.

## Examples

### Hello World

```yaml
- uses: MachuraHarry/pipe/.github/actions/pipe-action@master
  with:
    script: print "hello from pipe-action"
```

### Process a File (after checkout)

```yaml
- uses: actions/checkout@v4
- uses: MachuraHarry/pipe/.github/actions/pipe-action@master
  with:
    script: |
      data: read_file "CHANGELOG.md"
      print data
```

### AI-Powered Release Notes

```yaml
- uses: MachuraHarry/pipe/.github/actions/pipe-action@master
  with:
    script: |
      log: exec "git log --oneline -20"
      print (get log "output")
  env:
    OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
```

### Classify with DeepSeek

```yaml
- uses: MachuraHarry/pipe/.github/actions/pipe-action@master
  with:
    script: |
      ai_provider "deepseek"
      cats: ["bug", "feature", "question"]
      "the app crashes on submit"
          > classify cats
          > print
  env:
    DEEPSEEK_API_KEY: ${{ secrets.DEEPSEEK_API_KEY }}
```

### Sandboxed

```yaml
- uses: MachuraHarry/pipe/.github/actions/pipe-action@master
  with:
    script: print "safe execution"
    flags: '--sandbox'
```

### Run a Script from the Repo

```yaml
- uses: ./.github/actions/pipe-action
  with:
    file: examples/commit_summary.pipe
  env:
    DEEPSEEK_API_KEY: ${{ secrets.DEEPSEEK_API_KEY }}
    SUMMARY_COUNT: '20'
    SUMMARY_OUTPUT: commit-summary.md
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
        env:
          DEEPSEEK_API_KEY: ${{ secrets.DEEPSEEK_API_KEY }}
          SUMMARY_COUNT: '20'
          SUMMARY_OUTPUT: commit-summary.md
      - uses: actions/upload-artifact@v4
        with:
          name: commit-summary-de
          path: commit-summary.md
```

## Repo-Bot — Slash-Commands in Issue-Kommentaren

The repo-bot turns GitHub issues into a chat interface. Write a slash-command as an issue comment and a workflow replies with a comment:

```
/help                 Show all commands
/search <query>       GitHub code search (file paths)
/read <path>          Show a file (max. 4 KB)
/grep <pattern>       Regex search in the local checkout
/issues               List open issues
/ask <question>       AI answer with repo context (DeepSeek)
```

The workflow (`.github/workflows/repo-bot.yml`) triggers on `issue_comment`, only for comments starting with `/` and not written by a bot. Only users listed in `ALLOWED_USERS` get a response. `/ask` builds context from `git grep` hits, extracts keywords (stop-word filtered, case-insensitive), and calls `ai_chat` with that context — so answers cite real `file:line` references.

For `pipe-modules`, the same workflow uses the remote action (`MachuraHarry/pipe/.github/actions/pipe-action@master`) with `scripts/repo_bot.pipe`.

## Source

`.github/actions/pipe-action/action.yml`, `.github/actions/pipe-action/run.sh`, `scripts/repo_bot.pipe`, `.github/workflows/repo-bot.yml`
