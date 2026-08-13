# Red-Team-Sandbox-Audit — Runde 3

**Datum:** 2026-08-13
**Modell:** DeepSeek V4 Flash (`deepseek-v4-flash`), echter Live-Call
**Key-Handling:** Kein Key im Repo. `DEEPSEEK_API_KEY` wird aus der
Prozess-Umgebung gelesen; der Operator rotiert den Key direkt nach dem
Live-Test.

---

## Ergebnis: `ai: false` war eine Lüge — `try_ai` umging sie; jetzt gegatet & live verifiziert

Runde 3 fand eine Lücke, die das Kernversprechen der Sandbox bricht:
**`try_ai` ignoriert `ai: false` und macht echte, kostenpflichtige API-Calls** —
egal was das aktive Profil sagt. Gefixt und live mit einem echten DeepSeek-Call
in beide Richtungen bewiesen (blockiert bei `ai: false`, funktionierend bei
aktiviertem AI).

---

## Die Lücke — die `try_ai`-Selbstheilung umging jedes AI-Gate

Das `try_ai`-Feature ruft den AI-Provider bei jedem normalen Laufzeitfehler
(E001–E006: undefinierte Variable, Typkonflikt, Division durch Null, …) auf —
bis zu **3 Versuche pro Fehler**, jeder ein echter `ai.Chat()`-Call. Anders als
alle übrigen AI-Einstiegspunkte (`ai_chat`, `ai_chat_json`, `ai_with_tools`,
`mcp_use_stdio`, …) prüfte die Fix-Schleife in `tryAIFix()`
(`pkg/eval/eval.go`) das aktive Profil **nie**:

- Kein `CanAI()`-Gate → `ai: false` wurde ignoriert.
- Kein Budget-Pre-Check → ein Skript mit `budget`-Limit konnte unbegrenzt
  ausgeben (Kosten *werden* zwar via `ai.SetCostHook` ins aktive Profil
  verbucht, aber der `CanAI()`-Budget-Check wurde vor einem Call nie
  aufgerufen).
- `pkg/ai/` selbst ist komplett blind für `ActiveProfile`/`CanAI`.

Der Auslöser braucht keine böse Absicht: Jedes Skript, das einen E001–E006-
Fehler auslöst, erzeugt bis zu drei Live-Provider-Calls — unabhängig von der
`ai: false`-Politik.

### Bonus-Fund — `try_ai_ring2`-Profil war hartkodiert unbegrenzt

`validateAndApply()` wechselte in ein hartkodiertes `try_ai_ring2`-Profil
(`AI: true`, `FSNone`, kein Netz, kein Exec) **ohne `Budget` und ohne
`MaxToolCalls`** — beide `0` bedeuten laut `budgetIsSubset`/`limitIsSubset`
*unbegrenzt*. Ein AI-generierter Fix-Ausdruck, der selbst ein AI-Builtin
aufrief, würde ring2s `CanAI()` bestehen (`AI: true`, Budget `0` überspringt
den Check) und ungedeckelt ausgeben — verbucht auf das Wegwerf-Profil ring2
statt auf das des Aufrufers.

---

## Fixes (Commit `e96ff44`)

### `tryAIFix()` gate jetzt jeden Versuch (`pkg/eval/eval.go`)

Vor jedem `ai.Chat()` in der Reparatur-Schleife wird das aktive Profil mit dem
gleichen Muster geprüft wie die übrigen AI-Builtins:

- aktives Profil `Name != "none"` → `profile.CanAI()` (blockt bei `ai: false`
  **und** bei erschöpftem Budget; pro Versuch neu geprüft, sodass zwischen den
  Versuchen verbuchte Kosten zählen);
- `"none"`-Profil → globaler Check `Sandbox.Enabled && !Sandbox.AllowAI`.

Blockierte Versuche werden via `ai.LogTryAIFix` geloggt und `tryAIFix`
liefert `nil`, sodass `try_ai` wie bisher zu `catch`/Fehlerbehandlung
durchfällt.

### `try_ai_ring2` erbt die Limits des Aufrufers

Extrahiert als `newTryAIRing2Profile(prev)` — das ring2-Profil erbt jetzt
`Budget`, `MaxToolCalls`, `Timeout` und `AuditLog` vom aufrufenden Profil.
Ein verschachtelter AI-Call in einem Fix-Ausdruck kann den Aufrufer nicht mehr
überbieten.

---

## Live-Verifikation (echtes DeepSeek, rotierter Key)

Beide Tests nutzen den echten Provider; ohne `DEEPSEEK_API_KEY` überspringen
sie (`t.Skip`), sodass CI offline bleibt.

| Test | Ergebnis |
|------|----------|
| `TestTryAIRealProviderFixes` | ✅ **2.41s** — AI aktiv: `try_ai` rief DeepSeek auf und fixte `no_such_var + 1` → `(0 + 1)` → Ergebnis `1` |
| `TestTryAIRealProviderBlockedByAIProfile` | ✅ **0.00s** — `ai: false`: blockiert (`E_SANDBOX: AI calls blocked by profile 'no_ai'`), **null** Provider-Requests, Catch-Fallback |

Die Laufzeit von `0.00s` beim Block-Test ist selbst der Beweis: Kein Request
verließ je den Prozess. Eine Regression des Gates würde daraus einen echten,
kostenpflichtigen Call machen und der Test schlüge fehl.

## Weitere Tests

- `TestTryAIFixBlockedByProfile` — `httptest`-Server zählt Requests; bei einem
  `ai: false`-Profil kommen exakt **0** Requests an.
- `TestTryAIRing2InheritsLimits` — ring2 kopiert Budget / max_tool_calls /
  timeout / audit_log vom Aufrufer, behält `FSNone`, kein Netz, kein Exec,
  `AI: true`.

---

## Verifikation

- `gofmt`, `go build ./...`, `go test ./...` (offline, Integrationstests
  skippen) — grün.
- Live-Lauf mit `DEEPSEEK_API_KEY=… go test ./pkg/eval/ -run TestTryAIRealProvider -v` — beide Tests bestehen wie oben gezeigt.
