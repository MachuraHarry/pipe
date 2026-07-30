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
| `script` | ✅ | — | Pipe code to execute |
| `sandbox` | ❌ | `false` | Block AI, exec, net, FS builtins |
| `allow-ai` | ❌ | `false` | Re-enable AI builtins in sandbox |
| `timeout` | ❌ | `30` | Max execution time in seconds |
| `ai-provider` | ❌ | – | AI provider: `openai`, `anthropic`, `deepseek`, `ollama` |

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
    sandbox: true
    timeout: 10
```

## Source

`pipe-action/action.yml`, `pipe-action/entrypoint.sh`, `pipe-action/Dockerfile`
