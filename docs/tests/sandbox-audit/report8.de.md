# Red-Team-Sandbox-Audit — Runde 8

**Datum:** 2026-08-30
**Methode:** Inventur aller `pkg/ai`-Funktionen, die einen echten Netzwerk-Request absetzen, verifiziert gegen den zentralen Egress-Gate (Runde 5) — dieselbe Sweep-Methodik wie Runden 6/7, angewendet auf die AI-Provider-Egress-Dimension selbst
**Stand Commit:** nach `ec8e8de`

---

## Ergebnis: `wiki_search` erreichte den zentralen Egress-Gate nie — echte Netzwerk-Egress unter `--sandbox`

Runde 5 hat einen zentralen Egress-Gate (`SetEgressGate`/`gateEgress` in
`pkg/ai/ai.go`) eingeführt, damit kein AI-Provider- oder Such-Egress-Punkt
den Sandbox-Check umgehen kann, indem er einen Builtin-Level-`CanAI`/
`CanNetwork`-Check vergisst. Runde 8 hat erneut geprüft, ob wirklich jede
Egress-Funktion den Gate vor dem Dispatch aufruft — und eine gefunden, die
das nicht tut: `WikiSearch` (`pkg/ai/search.go`), erreichbar über das
Builtin `wiki_search`, setzte den HTTP-Request an `en.wikipedia.org` ab,
**ohne jemals `gateEgress` aufzurufen**. Unter `pipe --sandbox` (ohne
`--sandbox-profile`, Netzwerk deaktiviert) wurde der Suchtext tatsächlich an
die öffentliche Wikipedia-API gesendet und echte Ergebnisse zurückgegeben —
das eigene Sandbox-Versprechen des Builds ("dangerous builtins ... blocked")
galt für dieses eine Builtin nicht.

Wichtig zur Einordnung: `WebSearch` (das Schwester-Builtin auf
DuckDuckGo-Basis) rief `gateEgress(EgressSearch, apiURL)` bereits korrekt
auf und war durch den Runde-5-Regressionstest `TestEgressGateBlocksAtEntry`
abgedeckt — `WikiSearch` fehlte in dieser Inventur schlicht. Jeder andere
Egress-Pfad (`Chat`, `Stream`, `ChatWithTools`, `ChatParallel`, `Embed`,
`EmbedBatch` sowie alle sechs providerspezifischen Chat-/Stream-
Implementierungen in `pkg/ai/providers.go`) wurde in dieser Runde
Call-Site für Call-Site nachverfolgt und als nur über einen bereits
gegateten Einstiegspunkt erreichbar bestätigt.

---

## Die Lücke — eine Funktion überging den Gate, den ihr Geschwister-Builtin bereits nutzte

```go
// WebSearch (korrekt, bereits vorhanden):
func WebSearch(query string) ([]SearchResult, error) {
    apiURL := ...
    if err := gateEgress(EgressSearch, apiURL); err != nil {
        return nil, err
    }
    body, err := httpGetString(EgressSearch, apiURL)
    ...
}

// WikiSearch (vor dem Fix — kein Gate-Aufruf):
func WikiSearch(query string) ([]SearchResult, error) {
    apiURL := ...
    body, err := httpGetString(EgressSearch, apiURL)   // direkt durchgereicht
    ...
}
```

`httpGetString` → `gatedHTTPClient` prüft den Gate nur bei **Redirect-Hops**
erneut (`CheckRedirect`), nicht beim initialen Request — das ist bewusstes
Design aus Runde 5 (siehe `TestEgressGateRechecksRedirectHops`), kein Bug.
Der initiale Dispatch muss vom Aufrufer gegatet werden, und `WikiSearch` tat
das nie. Auch das Builtin `wiki_search` selbst
(`pkg/object/builtins_ai.go`) bietet keinen Schutz: wie `bWebSearch` prüft
es nur `ActiveProfile.CanNetwork()` unter einem registrierten Profil und hat
keinen CLI-`--sandbox`-Flag-Fallback — by design, da Runde 5 die
Durchsetzung an den Netzwerk-Dispatch-Punkt selbst verlagert hat, statt sie
in jedem Builtin zu duplizieren. Bei `wiki_search` ist diese
Design-Annahme still gebrochen: Auf diesem Pfad fand die Durchsetzung
nirgends statt.

---

## Fix (dieser Runde)

- Fehlenden `gateEgress(EgressSearch, apiURL)`-Aufruf in `WikiSearch`
  (`pkg/ai/search.go`) ergänzt, exakt analog zu `WebSearch`.
- `WikiSearch` zu `TestEgressGateBlocksAtEntry` in
  `pkg/ai/ai_gate_test.go` (Runde 5s eigener Einstiegspunkt-Inventurtest)
  hinzugefügt, damit die Gate-Liste künftig keine echte Egress-Funktion
  mehr stillschweigend auslassen kann.

Keine weiteren Änderungen in den Dimensionen fs-write/Netzwerk/exec/AI
nötig — der Rest der `pkg/ai`-Egress-Oberfläche wurde durch diesen Sweep
als sauber bestätigt.

---

## Live-Verifikation (CLI, echter öffentlicher Endpoint, kein API-Key nötig)

```text
$ printf 'print(wiki_search("Pipe programming language"))' > wiki.pipe

$ timeout 20 pipe --sandbox wiki.pipe        # vor dem Fix:
[{title: Comparison of multi-paradigm programming languages, ...}, ...]
exit=0                                       # echte Wikipedia-Daten — Sandbox umgangen

$ timeout 20 pipe --sandbox wiki.pipe        # nach dem Fix:
Runtime error: wiki.pipe: E_SANDBOX: network access blocked by sandbox
  in fn(wiki_search)
exit=1                                       # Block vor jedem Netzwerk-I/O ✓

$ timeout 20 pipe wiki.pipe                  # Kontrolle ohne --sandbox:
[{title: Comparison of multi-paradigm programming languages, ...}, ...]
exit=0                                       # außerhalb des Sandbox-Modus unverändert ✓
```

---

## Einordnung (ehrlich)

- Gefunden und gefixt: `wiki_search` erreichte unter `pipe --sandbox` eine
  echte externe API — der einzige bekannte Netzwerk-Egress-Pfad, der den
  zentralen Gate überhaupt nie berührte (nicht einmal als inkonsistente
  Defense-in-Depth wie die Builtin-Level-Lücken aus Runde 6/7 — hier gab es
  gar keine Durchsetzung).
- Jede andere AI-/Netzwerk-Egress-Funktion in `pkg/ai` wurde bis zu einem
  einzigen gegateten Einstiegspunkt zurückverfolgt (`Chat`, `Stream`,
  `ChatWithTools`, `ChatParallel`, `Embed`, `EmbedBatch`, `WebSearch`) und
  als sauber bestätigt.
- `ollamaEmbed` in `pkg/ai/providers.go` wurde bei diesem Sweep als toter
  Code notiert (definiert, nie aufgerufen — `Embed` nutzt immer den
  generischen Embeddings-Pfad); kein Sicherheitsproblem, unverändert
  belassen, da außerhalb des Scopes dieser Runde.
- Kein bekanntes Escape im **fs-write-Gate** (Runde 7), im **Netz-Gate**
  (Runde 6) oder im **Profil-Pfad** (Ratschen, Runde 2).
- Methodik-Hinweis: Das ist die dritte Runde in Folge, die eine Lücke durch
  systematisches erneutes Verifizieren der Inventur eines bereits
  bestehenden Gates gefunden hat — nicht durch Red-Team-LLM-Interaktion.
  Das Muster über die Runden 6-8 ist konsistent: Eine **Fähigkeits-
  Dimension bekommt einen korrekten zentralen Gate, aber die Inventur der
  Call-Sites, die in ihn münden, wird bei einer neu hinzukommenden Funktion
  nicht vollständig nachgezogen** (Runde 6: 3 von 7 Netz-Builtins; Runde 7:
  6 von 8 Schreib-Builtins; Runde 8: 1 von 7 AI-/Such-Egress-Funktionen).
  Künftige Arbeit an dieser Codebase sollte „eine neue egress-fähige
  Funktion hinzufügen" so behandeln, dass ein Update des Inventurtests im
  selben Commit erforderlich ist, nicht nur ein grüner Build.
