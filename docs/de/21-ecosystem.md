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
pipe -get log-analyzer              # Neueste Version (implizit @latest)
pipe -get log-analyzer@1.0.0        # Bestimmte Version
pipe -get https://...               # Von URL installieren
```

Module werden in `~/.pipe/modules/` zwischengespeichert.

## Modul-Versionen

Mit `@version` wird ein Modul auf eine bestimmte Version festgelegt:

```pipe
-- exakte Version
import "log-analyzer@1.0[0]"
-- neueste Version
import "log-analyzer"
```

Ohne Version ermittelt Pipe automatisch `latest` aus der Registry.

## Module nutzen

Nach der Installation importierst du das Modul in dein Skript:

```pipe
import "log-analyzer@1.0[0]"

read_file "errors.log"
    > split "\n"
    > log_analyze
    > print
```

## Verfügbare Module (9)

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
| `parallel-runner` | Parallele KI-Abfragen via `>>` | `ask_many`, `summarize_many`, `translate_many` |

## Ein Modul beitragen

### Gerüst

```bash
pipe -init my-module    # erzeugt pipe.json + module.pipe + README.md
```

### Validieren

```bash
pipe -validate my-module          # prüft die Gültigkeit von pipe.json
pipe -gen-registry modules/       # Registry-Eintrag als Vorschau anzeigen
```

### Veröffentlichen via Pull Request

`pipe -publish` automatisiert den kompletten Beitrags-Workflow: Es validiert
dein Modul, prüft die Registry auf Versionskonflikte, klont das Registry-Repo,
staged dein Modul samt aktualisiertem `registry.json` und öffnet mit der
[gh CLI](https://cli.github.com) einen Pull Request:

```bash
pipe -publish my-module
```

Voraussetzungen:

- Die [gh CLI](https://cli.github.com) ist installiert und angemeldet (`gh auth login`)
- `pipe.json` hat einen `name` und eine semantische `version` (z. B. `1.2.0`)
- `module.pipe` existiert im Modul-Verzeichnis
- Die Version darf **nicht bereits** in der Registry existieren — sonst Version in `pipe.json` erhöhen

Der Workflow führt `gh pr create` gegen `MachuraHarry/pipe-modules` aus;
der Branch heißt `publish/<name>-<version>`.

### Manueller Weg

1. Forke [pipe-modules](https://github.com/MachuraHarry/pipe-modules)
2. Erstelle einen Ordner: `mkdir my-module`
3. Schreibe `module.pipe` mit exportierten Funktionen
4. Teste: `pipe -ast module.pipe`
5. Öffne einen Pull Request

Siehe [CONTRIBUTING.md](https://github.com/MachuraHarry/pipe-modules/blob/master/CONTRIBUTING.md) für Details.

## Datenschutz & DSGVO

Pipe ist **DSGVO-konform / GDPR-compliant by design**:

- **Zero Telemetrie & Analytics** — die Binary meldet sich nie nach Hause, nichts verlässt deine Maschine
- **Self-hosted Single-Binary** — läuft vollständig auf deiner Infrastruktur
- **Kein Cloud-Zwang** — kein Vendor-Server verarbeitet deine Daten
- **Open Source (MIT)** — vollständig auditierbar
- **Lokale KI** — mit Ollama verlässt kein einziges Byte dein Netzwerk; Cloud-Provider werden nur genutzt, wenn du einen konfigurierst (die Daten gehen dann direkt an den von dir gewählten Provider)
