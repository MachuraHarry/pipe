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

## Verfügbare Module (22)

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
| `pipe-http` | HTTP-Client mit Auth | `hget`, `hpost`, `hput`, `hdelete`, `req`, `auth_bearer`, `auth_basic`, `auth_apikey` |
| `jpipe` | JSON-Path-Abfragen | `jp`, `jpick`, `jkeys`, `jflatten` |
| `pipe-cli` | CLI-Framework | `app`, `command`, `flag`, `handler`, `run` |
| `pipe-date` | Datum/Zeit-Hilfsfunktionen | `parse`, `format`, `add_days`, `diff_days`, `relative` |
| `pipe-test` | Bessere Test-Assertions | `expect_eq`, `expect_truthy`, `expect_contains`, `with_hooks` |
| `pipe-tpl` | Template-Engine | `render`, `render_file` |
| `pipe-validate` | Schema-Validierung | `validate`, `is_valid` |
| `pipe-orm` | ORM für Pipe+SQLite | `table`, `col`, `migrate`, `insert`, `select`, `all`, `first`, `count`, `update`, `delete` |
| `pipe-web` | Web-Framework (ASP.NET-/Express-Stil) | `app`, `route_get`, `post`, `put`, `delete`, `use`, `listen`, `serve`, `json`, `ok`, `text`, `html`, `redirect`, `not_found` |
| `sqlite` | Reine-Pipe-SQL-Datenbank-Engine | `db_open`, `db_close`, `db_exec`, `db_query`, `q`, `exec` |
| `rag-pipe` | RAG-Pipeline | `index_create`, `index_add`, `index_search`, `index_ask` |
| `docs-pipe` | Dokumentations-natives RAG (heading-bewusstes Chunking, hybride Keyword- + semantische Suche, zitierte Antworten, inkrementelles Indexieren) | `doc_index`, `doc_index_status`, `doc_search`, `doc_ask`, `doc_reindex`, `doc_close` |
| `telegram-bot` | Telegram-Bot-API-Client (Long Polling) | `tg_bot`, `tg_me`, `tg_get_updates`, `tg_send_text`, `tg_send_md`, `tg_send_html`, `tg_send_mdv2`, `tg_reply_text`, `tg_send_photo_url`, `tg_send_photo_file_id`, `tg_send_buttons`, `tg_send_chat_action`, `tg_edit_text`, `tg_edit_md`, `tg_edit_html`, `tg_delete_message`, `tg_forward_message`, `tg_set_reaction`, `tg_answer_callback_query`, `tg_get_chat`, `tg_get_chat_member` |

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
