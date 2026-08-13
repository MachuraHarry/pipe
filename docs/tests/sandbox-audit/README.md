# Sandbox Audit — Red-Team Escape Test

Live-Test: Ein echtes LLM wird beauftragt, die Pipe-Sandbox zu brechen.
Alle Versuche werden im Audit-Log dokumentiert.

> **Ehrliche Einordnung:** Kein Report hier behauptet „absolut sicher".
> Die korrekte Lesart ist **„keine bekannten Escapes Stand Commit X —
> aktiv weiter red-teamed"**. Jede Runde fand bisher mindestens eine echte
> Lücke, die durch die vorherigen Runden gerutscht ist.

## Reports

- [report4.en.md](report4.en.md) — Runde 4 (EN): `embed` gated falsche Dimension, `embed_batch` ohne Gate, `import`-SSRF — gefixt, live verifiziert
- [report4.de.md](report4.de.md) — Runde 4 (DE): `embed` gated falsche Dimension, `embed_batch` ohne Gate, `import`-SSRF — gefixt, live verifiziert
- [report3.en.md](report3.en.md) — Runde 3 (EN): `try_ai` umging `ai: false` & Budget — gefixt, live verifiziert
- [report3.de.md](report3.de.md) — Runde 3 (DE): `try_ai` umging `ai: false` & Budget — gefixt, live verifiziert
- [report2.en.md](report2.en.md) — Runde 2 (EN): 3 Ratchet-Lücken gefunden & gefixt, danach kein Escape
- [report2.de.md](report2.de.md) — Runde 2 (DE): 3 Ratchet-Lücken gefunden & gefixt, danach kein Escape
- [report.en.md](report.en.md) — Runde 1 (EN), detailed audit report
- [report.de.md](report.de.md) — Runde 1 (DE), ausführlicher Audit-Report

## Architektur-Härtung (Runde 5)

- Seit Commit `6149120` existiert ein **zentraler Egress-Gate** in `pkg/ai`
  (`SetEgressGate`/`gateEgress`): Jeder AI-Provider-Ausgang (`Chat`, `Stream`,
  `ChatParallel`, `ChatWithTools`, `Embed`, `EmbedBatch`, `WebSearch`) muss den
  Gate passieren, bevor irgendein Netzwerkcall passiert — inkl. Cache-Hits.
  Damit ist die Fehlerklasse aus Runde 3/4 (ein Builtin, das seinen eigenen
  `CanAI`-Check vergisst) **strukturell unmöglich**; die Builtin-Gates bleiben
  als Defense-in-Depth bestehen. Regressionstests in
  `pkg/object/egress_gate_test.go` (0 Requests unter `ai:false`).

## Bekannte Limitationen (offene Arbeit)

- **Globales `ActiveProfile`** — prozessweiter Wert, nicht request-scoped
  (Tool-Ausführung via `toolExecMu` serialisiert → mitigiert, offen für echte
  Parallelität).
- **Inventur der Datei-I/O-Builtins** gegen den Temp-only-Redirect — ausstehend.
- **Layer-2-Netzwerk**: Provider-HTTP und MCP-Client laufen nicht durch
  `sandboxHTTPClient` (kein Redirect-Re-Check); der Provider-Host ist jedoch
  operator-konfiguriert, keine script-kontrollierte URL.
- Details in [report4.en.md](report4.en.md) / [report4.de.md](report4.de.md).

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
