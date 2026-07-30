# Pipe Action

Run [Pipe](https://pipe-lang.com) code directly in your GitHub workflow.

## Usage

```yaml
jobs:
  analyze:
    runs-on: ubuntu-latest
    steps:
      - uses: MachuraHarry/pipe@master
        with:
          script: |
            read_file "report.txt"
              > summarize
              > translate "de"
              > print
        env:
          OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
```

## Inputs

| Input | Default | Description |
|---|---|---|
| `script` | – | Pipe code to execute (required) |
| `sandbox` | `false` | Block AI, exec, net, FS builtins |
| `allow-ai` | `false` | Re-enable AI builtins in sandbox |
| `timeout` | `30` | Max execution time in seconds |

## Examples

### Hello World
```yaml
- uses: MachuraHarry/pipe@master
  with:
    script: print "hello from pipe-action"
```

### Classify an Issue
```yaml
- uses: MachuraHarry/pipe@master
  with:
    script: |
      read_file "${{ github.event.issue.body }}"
        > classify ["bug","feature","question"]
        > print
  env:
    OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
```

### Sandboxed Execution
```yaml
- uses: MachuraHarry/pipe@master
  with:
    script: print "safe execution"
    sandbox: true
    timeout: 10
```
