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
      - uses: MachuraHarry/pipe/.github/actions/pipe-action@master
        with:
          script: |
            print "Hallo aus CI/CD!"
            print (2 + 2)
```

## Inputs

| Input | Erforderlich | Standard | Beschreibung |
|-------|-------------|----------|-------------|
| `script` | ✅ | — | Auszuführender Pipe-Code |
| `flags` | ❌ | `-vm -q` | CLI-Flags |
| `provider` | ❌ | — | KI-Provider |

## Beispiele

### Git-Log-Zusammenfassung

```yaml
- uses: MachuraHarry/pipe/.github/actions/pipe-action@master
  with:
    script: |
      log: exec "git log --oneline -20"
      print (get log "output")
```

### KI-gestützte Release Notes

```yaml
- uses: MachuraHarry/pipe/.github/actions/pipe-action@master
  with:
    provider: 'deepseek'
    script: |
      log: exec "git log --oneline -20"
      summarize (get log "output") > save "CHANGELOG.md"
  env:
    DEEPSEEK_API_KEY: ${{ secrets.DEEPSEEK_API_KEY }}
```
