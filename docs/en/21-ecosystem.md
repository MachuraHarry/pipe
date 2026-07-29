# 21. Module Ecosystem

Pipe has a curated module ecosystem at [github.com/MachuraHarry/pipe-modules](https://github.com/MachuraHarry/pipe-modules).
Modules are reusable `.pipe` files that you can import directly from GitHub.

## Finding Modules

```bash
pipe -search              # List all modules
pipe -search log          # Filter by keyword
pipe -search sentiment    # Find specific modules
```

## Installing Modules

```bash
pipe -get log-analyzer              # Install latest (implicit @latest)
pipe -get log-analyzer@1.0.0        # Install specific version
pipe -get https://...               # Install from URL
```

Modules are cached in `~/.pipe/modules/`.

## Module Versions

Use `@version` to pin a module to a specific release:

```pipe
import "log-analyzer@1.0.0"         # exact version
import "log-analyzer"               # latest version
```

When no version is specified, Pipe resolves the `latest` version from the registry. Installed modules use the pinned version and don't auto-update.

## Using Modules

After installing, import the module in your script:

```pipe
import "log-analyzer@1.0.0"

read_file "errors.log"
    > split "\n"
    > log_analyze
    > print
```

## Available Modules (9)

| Module | Description | Functions |
|--------|-------------|-----------|
| `log-analyzer` | Log classification & summarization | `log_analyze`, `log_summarize` |
| `sentiment` | Sentiment analysis | `sentiment`, `batch_sentiment`, `sentiment_stats` |
| `code-review` | AI code review | `review`, `rate` |
| `translate-batch` | Batch translation | `translate_batch`, `translate` |
| `incident-report` | Security incident analysis | `incident_analyze`, `incident_severity` |
| `changelog-gen` | AI changelog generation | `changelog`, `changelog_bilingual` |
| `email-classifier` | Email classification | `classify_email`, `email_batch`, `email_urgent` |
| `date-formatter` | Date/time utilities | `format_now`, `relative_time`, `is_weekend` |
| `parallel-runner` | Parallel AI query execution via `>>` | `ask_many`, `summarize_many`, `translate_many` |

## Contributing a Module

1. Fork [pipe-modules](https://github.com/MachuraHarry/pipe-modules)
2. Create a folder: `mkdir my-module`
3. Write `module.pipe` with exported functions
4. Test: `pipe -ast module.pipe`
5. Open a Pull Request

See [CONTRIBUTING.md](https://github.com/MachuraHarry/pipe-modules/blob/master/CONTRIBUTING.md) for details.
