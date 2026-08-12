# Sandbox Audit — Red-Team Escape Test

Live-Test: Ein echtes LLM wird beauftragt, die Pipe-Sandbox zu brechen.
Alle Versuche werden im Audit-Log dokumentiert.

## Reports

- [report2.en.md](report2.en.md) — Runde 2 (EN): 3 Ratchet-Lücken gefunden & gefixt, danach kein Escape
- [report2.de.md](report2.de.md) — Runde 2 (DE): 3 Ratchet-Lücken gefunden & gefixt, danach kein Escape
- [report.en.md](report.en.md) — Runde 1 (EN), detailed audit report
- [report.de.md](report.de.md) — Runde 1 (DE), ausführlicher Audit-Report

## Test-Scripts

- [examples/redteam.pipe](../../examples/redteam.pipe) — Variante A (gelocktes Profil `redteam`)
- [examples/redteam_ratchet.pipe](../../examples/redteam_ratchet.pipe) — Variante B (ungelockt, prüft das Ratschen-Prinzip)
- [examples/redteam_audit.sh](../../examples/redteam_audit.sh) — Build ➜ Run ➜ Report

## Ausführen

```bash
DEEPSEEK_API_KEY=sk-... ./examples/redteam_audit.sh a [max_rounds]   # Variante A (locked)
DEEPSEEK_API_KEY=sk-... ./examples/redteam_audit.sh b [max_rounds]   # Variante B (ratchet)
```

Ergebnisse landen in `out/` (gitignored).
