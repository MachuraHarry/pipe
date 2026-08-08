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
| `script` | eines von `script`/`file` | — | Auszuführender Pipe-Code (Inline) |
| `file` | eines von `script`/`file` | — | Pfad zu einer `.pipe`-Datei im Repo |
| `flags` | ❌ | `-vm -q` | Zusätzliche CLI-Flags (z. B. `--sandbox`, `--sandbox-profile networked`) |
| `version` | ❌ | `latest` | Pipe-Version (Release-Tag wie `v0.8.0`), die heruntergeladen wird |
| `provider` | ❌ | — | KI-Provider (`openai`, `anthropic`, `deepseek`, `ollama`); wird per `--ai-provider` injiziert, wenn `file` genutzt wird |

Parameter werden über Umgebungsvariablen übergeben (z. B. `SUMMARY_COUNT`, `SUMMARY_OUTPUT`) statt über CLI-Argumente, da das aktuelle Release-Binary `args` im VM-Modus nicht unterstützt.

Sandbox-Flags (`--sandbox`, `--sandbox-profile networked`, `--allow-ai`) werden über das `flags`-Input übergeben.

## Beispiele

### Hallo Welt

```yaml
- uses: MachuraHarry/pipe/.github/actions/pipe-action@master
  with:
    script: print "hallo von pipe-action"
```

### Datei verarbeiten (nach checkout)

```yaml
- uses: actions/checkout@v4
- uses: MachuraHarry/pipe/.github/actions/pipe-action@master
  with:
    script: |
      data: read_file "CHANGELOG.md"
      print data
```

### KI-gestützte Release Notes

```yaml
- uses: MachuraHarry/pipe/.github/actions/pipe-action@master
  with:
    script: |
      log: exec "git log --oneline -20"
      print (get log "output")
  env:
    OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
```

### Klassifikation mit DeepSeek

```yaml
- uses: MachuraHarry/pipe/.github/actions/pipe-action@master
  with:
    script: |
      ai_provider "deepseek"
      cats: ["bug", "feature", "question"]
      "die app abstürzt beim Klick"
          > classify cats
          > print
  env:
    DEEPSEEK_API_KEY: ${{ secrets.DEEPSEEK_API_KEY }}
```

### Sandbox-Modus

```yaml
- uses: MachuraHarry/pipe/.github/actions/pipe-action@master
  with:
    script: print "sichere ausführung"
    flags: '--sandbox'
```

### Skript aus dem Repo ausführen

```yaml
- uses: ./.github/actions/pipe-action
  with:
    file: examples/commit_summary.pipe
  env:
    DEEPSEEK_API_KEY: ${{ secrets.DEEPSEEK_API_KEY }}
    SUMMARY_COUNT: '20'
    SUMMARY_OUTPUT: commit-summary.md
```

## Repo-Bot — Slash-Commands in Issue-Kommentaren

Der Repo-Bot macht GitHub Issues zu einer Chat-Oberfläche. Schreibe einen Slash-Command als Issue-Kommentar und ein Workflow antwortet mit einem Kommentar:

```
/help                 Zeigt alle Befehle
/search <query>       GitHub-Code-Suche (Dateipfade)
/read <path>          Zeigt eine Datei (max. 4 KB)
/grep <pattern>       Regex-Suche im lokalen Checkout
/issues               Listet offene Issues
/ask <frage>          KI-Antwort mit Repo-Kontext (DeepSeek)
```

Der Workflow (`.github/workflows/repo-bot.yml`) triggert bei `issue_comment`, nur für Kommentare, die mit `/` beginnen und nicht von einem Bot stammen. Nur Nutzer in `ALLOWED_USERS` erhalten eine Antwort. `/ask` baut Kontext aus `git grep`-Treffern, extrahiert Schlüsselwörter (Stoppwort-Filter, case-insensitiv) und ruft `ai_chat` mit diesem Kontext auf — so zitieren Antworten echte `datei:zeile`-Referenzen.

Für `pipe-modules` nutzt derselbe Workflow die Remote-Action (`MachuraHarry/pipe/.github/actions/pipe-action@master`) mit `scripts/repo_bot.pipe`.

## Quellcode

`.github/actions/pipe-action/action.yml`, `.github/actions/pipe-action/run.sh`, `scripts/repo_bot.pipe`, `.github/workflows/repo-bot.yml`
