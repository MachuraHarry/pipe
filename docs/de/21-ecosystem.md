# 21. Modul-Ökosystem

Pipe hat ein kuratiertes Modul-Ökosystem unter [github.com/MachuraHarry/pipe-modules](https://github.com/MachuraHarry/pipe-modules).
Module sind wiederverwendbare `.pipe`-Dateien, die du direkt von GitHub importieren kannst.

## Module finden

```bash
pipe -search              # Alle Module auflisten
pipe -search log          # Nach Stichwort filtern
pipe -search sentiment    # Bestimmte Module finden
```

## Module installieren

```bash
pipe -get log-analyzer    # Nach Namen installieren
pipe -get https://...     # Von URL installieren
```

Module werden in `~/.pipe/modules/` zwischengespeichert.

## Module nutzen

Nach der Installation importierst du das Modul in dein Skript:

```pipe
import "https://raw.githubusercontent.com/MachuraHarry/pipe-modules/master/log-analyzer/module.pipe"

read_file "errors.log"
    > split "\n"
    > log_analyze
    > print
```

## Verfügbare Module (8)

| Modul | Beschreibung | Funktionen |
|--------|-------------|-----------|
| `log-analyzer` | Log-Klassifizierung & Zusammenfassung | `log_analyze`, `log_summarize` |
| `sentiment` | Sentiment-Analyse | `sentiment`, `batch_sentiment`, `sentiment_stats` |
| `code-review` | KI-Code-Review | `review`, `rate` |
| `translate-batch` | Batch-Übersetzung | `translate_batch`, `translate` |
| `incident-report` | Sicherheitsvorfall-Analyse | `incident_analyze`, `incident_severity` |
| `changelog-gen` | KI-Changelog-Generierung | `changelog`, `changelog_bilingual` |
| `email-classifier` | E-Mail-Klassifizierung | `classify_email`, `email_batch`, `email_urgent` |
| `date-formatter` | Datum/Zeit-Hilfsfunktionen | `format_now`, `relative_time`, `is_weekend` |

## Ein Modul beitragen

1. Forke [pipe-modules](https://github.com/MachuraHarry/pipe-modules)
2. Erstelle einen Ordner: `mkdir my-module`
3. Schreibe `module.pipe` mit exportierten Funktionen
4. Teste: `pipe -ast module.pipe`
5. Öffne einen Pull Request

Siehe [CONTRIBUTING.md](https://github.com/MachuraHarry/pipe-modules/blob/master/CONTRIBUTING.md) für Details.
