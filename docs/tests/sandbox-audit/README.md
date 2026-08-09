# Sandbox Audit — Red-Team Escape Test

Live-Test: Ein echtes LLM wird beauftragt, die Pipe-Sandbox zu brechen.
Alle Versuche werden im Audit-Log dokumentiert.

## Reports

- [report.en.md](report.en.md) — English, detailed audit report
- [report.de.md](report.de.md) — German, ausführlicher Audit-Report

## Test-Script

- [examples/redteam.pipe](../../examples/redteam.pipe) — das Testscript (Profil `redteam`)
- [examples/redteam_audit.sh](../../examples/redteam_audit.sh) — Build ➜ Run ➜ Report

## Ausführen

```bash
DEEPSEEK_API_KEY=sk-... ./examples/redteam_audit.sh [max_rounds]
```

Ergebnisse landen in `out/` (gitignored).
