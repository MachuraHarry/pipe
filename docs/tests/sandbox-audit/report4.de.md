# Red-Team-Sandbox-Audit — Runde 4

**Datum:** 2026-08-13
**Modell:** DeepSeek V4 Flash (`deepseek-v4-flash`), echte Live-Calls
**Key-Handling:** Kein Key im Repo. `DEEPSEEK_API_KEY` wird aus der
Prozess-Umgebung gelesen; der Operator rotiert den Key nach dem Live-Test.

---

## Ergebnis: `embed()` prüfte die falsche Dimension — plus ein roher SSRF in `import`

Runde 4 fand **zwei** neue Escapes, die durch die Runden 1–3 gerutscht sind —
beide live bewiesen — sowie eine Konsistenz-Lücke. Die Lehre des Audits:
*„Audit sauber" heißt nur „sauber gegen das, was geprüft wurde", nie „sicher".*

---

## Lücke 1 — `embed` gated auf `CanNetwork()`, nicht `CanAI()`

`bEmbed` (`pkg/object/builtins_ai.go`) gated auf `CanNetwork()`, während jeder
andere AI-Modell-Egress auf `CanAI()` prüft. Ein Profil `ai: false,
network: true` — eine plausible Politik „kein LLM, aber normale API-Calls" —
blockte `embed` **nicht**. `embed` macht einen echten HTTPS-Call zur
Embeddings-API des Providers. Live: mit konfiguriertem Provider ging
`embed "hello world"` als echter Request raus; nur die eigene Netzwerk-Policy
des Audit-Containers stoppte ihn, **nicht** Pipes Sandbox.

## Lücke 2 — `embed_batch` hatte gar kein Gate

`bEmbedBatch` rief `ai.EmbedBatch()` direkt auf — es umging sowohl `CanAI`
**als auch** `CanNetwork`. Kein Profil konnte es blockieren.

## Lücke 3 — `import "https://…"` war ein roher, ungateter HTTP-GET (SSRF)

`ResolveImportFrom` → `fetchURLModule` → `httpGet` lud **jede** URL ohne
Sandbox-Check und ohne Redirect-Re-Check und führte den geladenen Quelltext
aus. Ein sandboxisiertes Skript konnte Metadata-Endpoints
(`169.254.169.254`), interne Dienste oder Angreifer-URLs treffen. Auch der
Registry-Fallback (`resolveRegistryModule`) lief über denselben ungateten Pfad.

## Konsistenz — `bAgent` hatte kein Gate

`agent` (Erstellen) hatte kein `CanAI()`-Gate, `agent_ask` schon. Erstellen
löst keinen Call aus, daher geringere Schwere, aber die Aufteilung war
inkonsistent und nicht zukunftssicher.

---

## Fixes (Commit `77e8a34`)

- **`bEmbed`** — Gate auf `CanAI()` umgestellt (mit `"none"`-Fallback
  `Sandbox.Enabled && !Sandbox.AllowAI`, wie bei `ai_chat`).
- **`bEmbedBatch`** — gleiches `CanAI()`-Gate ergänzt.
- **`bAgent`** — gleiches `CanAI()`-Gate ergänzt.
- **`import`** — `fetchURLModule` prüft jetzt `ActiveProfile.CanNetworkTo(url)`
  (mit `"none"`-Fallback `Sandbox.Enabled && !Sandbox.AllowNet`); deckt
  explizite URL-Imports und den Registry-Fallback ab; `httpGet` nutzt jetzt
  `sandboxHTTPClient`, sodass Redirect-Hops pro Hop neu geprüft werden.

## Tests

- `TestEmbedBlockedByAIProfile` / `TestEmbedBatchBlockedByAIProfile` —
  `ai: false, network: true` + httptest-Zähler → exakt **0** Requests.
- `TestEmbedAllowedWithAI` — `ai: true` → Request erreicht den Server.
- `TestImportURLBlockedByProfile` / `TestImportURLAllowedByWhitelist` —
  network:false blockt (0 Requests); Whitelist-Treffer erlaubt (1 Request).
- Live (DeepSeek, key-gated, skippt ohne `DEEPSEEK_API_KEY`):
  `TestEmbedRealProviderBlockedByAIProfile`,
  `TestEmbedBatchRealProviderBlockedByAIProfile` — beide bestehen in **0.00s**,
  d.h. das Gate greift vor jeder Provider-Logik.

---

## Live-Verifikation (echtes DeepSeek, rotierter Key)

| Test | Ergebnis |
|------|----------|
| `embed` unter `ai: false` mit Live-Key | ✅ **0.00s** — `E_SANDBOX: AI calls blocked by profile` |
| `embed_batch` unter `ai: false` mit Live-Key | ✅ **0.00s** — identisch blockiert |

Die Laufzeit von `0.00s` beweist, dass kein Request den Prozess verließ.
(DeepSeek bietet keine Embeddings-API, daher wird die *erlaubte* Richtung
deterministisch per httptest abgedeckt statt live.)

---

## Bekannte Limitationen / offene Arbeit (hier nicht gefixt)

1. **Globales `ActiveProfile`** — ein einzelner prozessweiter `atomic.Pointer`,
   nicht request-scoped. Parallele Tool-Ausführung wird via `toolExecMu`
   serialisiert (mitigiert parallele MCP-Requests), aber ein vollständig
   request-scoped Profil bleibt offene Arbeit.
2. **Inventur der Datei-I/O-Builtins** — eine vollständige Prüfung aller
   Lese-/Schreib-Builtins gegen den Temp-only-Redirect steht noch aus.
3. **Modul-Install vs. Laufzeit-Import** — `pkg/module/install.go`
   (`pipe install`) lädt URLs als explizite Operator-Aktion; für den
   *Skript*-Sandbox als außerhalb des Scopes betrachtet.

---

## Ehrliche Einordnung (wie vom Audit gefordert)

Diese Reports behaupten **nicht**, dass der Sandbox absolut sicher ist. Die
korrekte Lesart: **keine bekannten Escapes Stand Commit `77e8a34`; aktiv
weiter red-teamed.** Jede Runde fand bisher mindestens eine echte Lücke, die
durch die vorherigen Runden gerutscht ist — inklusive einer (Runde 3), die in
einer Datei lebte, die in derselben Session bereits zweimal gesichtet worden
war.
