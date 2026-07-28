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
pipe -get log-analyzer    # Install by name
pipe -get https://...     # Install from URL
```

Modules are cached in `~/.pipe/modules/`.

## Using Modules

After installing, import the module in your script:

```pipe
import "https://raw.githubusercontent.com/MachuraHarry/pipe-modules/master/log-analyzer/module.pipe"

read_file "errors.log"
    > split "\n"
    > log_analyze
    > print
```

## Available Modules (8)

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

## Contributing a Module

1. Fork [pipe-modules](https://github.com/MachuraHarry/pipe-modules)
2. Create a folder: `mkdir my-module`
3. Write `module.pipe` with exported functions
4. Test: `pipe -ast module.pipe`
5. Open a Pull Request

See [CONTRIBUTING.md](https://github.com/MachuraHarry/pipe-modules/blob/master/CONTRIBUTING.md) for details.
