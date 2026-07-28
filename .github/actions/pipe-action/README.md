# Pipe GitHub Action

Run [Pipe (SPR)](https://github.com/MachuraHarry/pipe) scripts directly in your GitHub Actions workflows.

## Quick Start

```yaml
- uses: MachuraHarry/pipe/.github/actions/pipe-action@master
  with:
    script: |
      print "Hello from Pipe!"
      print (2 + 2)
```

## AI-Powered CI/CD

```yaml
- uses: MachuraHarry/pipe/.github/actions/pipe-action@master
  with:
    provider: 'deepseek'
    script: |
      log: exec "git log --oneline -20"
      summarize (get log "output") > save "RELEASE_NOTES.md"
  env:
    DEEPSEEK_API_KEY: ${{ secrets.DEEPSEEK_API_KEY }}
```

## Inputs

| Input | Required | Default | Description |
|-------|----------|---------|-------------|
| `script` | ✅ | — | Pipe code to execute |
| `version` | ❌ | `latest` | Pipe version to use |
| `flags` | ❌ | `-vm -q` | CLI flags (VM mode, quiet) |
| `provider` | ❌ | — | AI provider: openai, anthropic, deepseek, ollama |

## Examples

### Generate Release Notes

```yaml
- uses: MachuraHarry/pipe/.github/actions/pipe-action@master
  with:
    provider: 'openai'
    script: |
      log: exec "git log --oneline v1.0..HEAD"
      summarize (get log "output")
        > translate "de"
        > save "CHANGELOG_DE.md"
```

### Code Review on PR

```yaml
- uses: MachuraHarry/pipe/.github/actions/pipe-action@master
  with:
    provider: 'deepseek'
    script: |
      diff: exec "git diff origin/main"
      ai_stream "You are a senior reviewer. Analyze this diff." (get diff "output")
```

### Safe Execution

```yaml
- uses: MachuraHarry/pipe/.github/actions/pipe-action@master
  with:
    flags: '--sandbox -vm -q'
    script: |
      print "Running in sandbox — no shell, no network, no file writes."
```

## API Keys

Set secrets in your repo settings:
- `OPENAI_API_KEY`
- `ANTHROPIC_API_KEY`
- `DEEPSEEK_API_KEY`
