# 20. GitHub Action

Führe Pipe-Skripte direkt in deinen GitHub Actions Workflows aus — keine Installation nötig.

## Verwendung

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
            print "Hallo aus CI/CD!"
            print (2 + 2)
```

## Inputs

| Input | Erforderlich | Standard | Beschreibung |
|-------|-------------|----------|-------------|
| `script` | ✅ | — | Auszuführender Pipe-Code |
| `sandbox` | ❌ | `false` | Blockiert AI, exec, net, FS Builtins |
| `allow-ai` | ❌ | `false` | AI Builtins im Sandbox-Modus erlauben |
| `timeout` | ❌ | `30` | Maximale Ausführungszeit in Sekunden |
| `ai-provider` | ❌ | – | KI-Provider: `openai`, `anthropic`, `deepseek`, `ollama` |

## Beispiele

### Hallo Welt

```yaml
- uses: MachuraHarry/pipe/pipe-action@master
  with:
    script: print "hallo von pipe-action"
```

### Datei verarbeiten (nach checkout)

```yaml
- uses: actions/checkout@v4
- uses: MachuraHarry/pipe/pipe-action@master
  with:
    script: |
      data: read_file "CHANGELOG.md"
      print data
```

### KI-gestützte Release Notes

```yaml
- uses: MachuraHarry/pipe/pipe-action@master
  with:
    script: |
      log: exec "git log --oneline -20"
      print (get log "output")
  env:
    OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
```

### Klassifikation mit DeepSeek

```yaml
- uses: MachuraHarry/pipe/pipe-action@master
  with:
    script: |
      cats: ["bug", "feature", "question"]
      "die app abstürzt beim Klick"
          > classify cats
          > print
    ai-provider: deepseek
    sandbox: true
    allow-ai: true
  env:
    DEEPSEEK_API_KEY: ${{ secrets.DEEPSEEK_API_KEY }}
```

### Sandbox-Modus

```yaml
- uses: MachuraHarry/pipe/pipe-action@master
  with:
    script: print "sichere ausführung"
    sandbox: true
    timeout: 10
```

## Quellcode

`pipe-action/action.yml`, `pipe-action/entrypoint.sh`, `pipe-action/Dockerfile`
