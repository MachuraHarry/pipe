# 20. GitHub Action

Run Pipe scripts directly in your GitHub Actions workflows — no installation needed.

## How It Works

1. GitHub checks out your code
2. The action builds Pipe from source (or downloads the binary)
3. Your Pipe script executes
4. Output appears in the workflow logs

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
| `script` | ✅ | — | Pipe code to execute |
| `flags` | ❌ | `-vm -q` | CLI flags |
| `provider` | ❌ | — | AI provider (openai, anthropic, deepseek, ollama) |

## Examples

### Git Log Summary

```yaml
- uses: MachuraHarry/pipe/.github/actions/pipe-action@master
  with:
    script: |
      log: exec "git log --oneline -20"
      print (get log "output")
```

### AI-Powered Release Notes

```yaml
- uses: MachuraHarry/pipe/.github/actions/pipe-action@master
  with:
    provider: 'openai'
    script: |
      log: exec "git log --oneline -20"
      summarize (get log "output") > save "CHANGELOG.md"
  env:
    OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
```

### Safe Sandboxed Execution

```yaml
- uses: MachuraHarry/pipe/.github/actions/pipe-action@master
  with:
    flags: '--sandbox -vm -q'
    script: |
      print "Running in sandbox — no shell, no network, no file writes."
```

## How It Works Internally

See the source: `.github/actions/pipe-action/`
- `action.yml` — composite action definition
- `run.sh` — bash wrapper that builds pipe and executes your script
