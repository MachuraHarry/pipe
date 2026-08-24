# Die Ruine von Aldenmoor

Ein KI-gesteuertes Textadventure — komplett gebaut in [Pipe](https://github.com/MachuraHarry/pipe).
Läuft **kostenlos und ohne API-Key** über den OpenCode-Zen-Provider (Free-Tier).

## Spielen

```bash
# aus diesem Verzeichnis:
../bin/pipe adventure.pipe
```

Freitext-Eingaben auf Deutsch, z.B.:

```
nimm die fackel
geh nach norden
rede mit dem waechter
wie heißt du?
gib dem waechter die muenze    <- so gewinnt man
speichere als run1
lade run1
ende
```

**Ziel:** Dem verfluchten Wächter in der Halle die Goldmünze aus der
Schatzkammer freiwillig zurückbringen — sein Fluch bricht nur durch eine
Gabe, nie durch Nehmen. (Die Lösung ist bewusst in der Lore versteckt.)

## Architektur

```
adventure.pipe       Game Loop (while/input/match) + Modell-Rotation + Sandbox
state.pipe           sqlite-Spielzustand (In-Memory), sql_escape, Save/Load
lore.pipe            RAG-Layer: embed_batch/nearest über die World-Lore
tools.pipe           ai_tool-Registrierungen: look, move_to, take_item,
                     give_item, show_inventory, talk_to, save_game, load_game
rooms.json           Raum-Seed (beim Start in sqlite importiert)
world/*.txt          Lore-Dokumente für den RAG-Kontext
adventure_test.pipe  13 Tests via `pipe -test` (gegen In-Memory-DB, ohne KI)
```

Prinzip: Das LLM erzählt nur. **Jede Zustandsänderung läuft über ein
registriertes `ai_tool`**, das reine Pipe-Funktionen aufruft, welche die
sqlite-DB mutieren. Die Wahrheit steht immer in der DB, nie im LLM-Kontext.
NPC-Gedächtnis wird pro Turn per `summarize` komprimiert (konstante Kosten).

## Technik-Notizen

- **Provider**: `ai_provider "opencode"` — Free-Tier-Modelle rotieren monatlich;
  `FREE_MODELS` in adventure.pipe listet Kandidaten, bei 503/429 wird automatisch
  zum nächsten gewechselt.
- **Embeddings**: OpenCode bietet keinen Embeddings-Endpunkt — `embed` fällt
  transparent auf Pipes lokalen Hash-Embedder zurück (RAG funktioniert trotzdem).
- **Sandbox**: Nach dem Welt-Aufbau wird `{fs: "temp-only", network_whitelist:
  ["opencode.ai"], exec: false, ai: true}` aktiviert; Saves landen im
  `.pipe_sandbox/saves/`-Verzeichnis.
- **Hinweis**: Free-Tier-Prompts dürfen laut OpenCode zum Modelltraining
  verwendet werden — bei einem Fantasy-Spiel irrelevant, für sensible Daten
  diesen Provider nicht verwenden.

## Engine-Fixes (in dieser Session am Pipe-Core)

1. `input` liest jetzt ganze Zeilen statt nur bis zum ersten Leerzeichen
   (`pkg/object/builtins_io.go`, vorher `fmt.Scanln`).
2. `_` als Platzhalter-Parameter erzeugt keinen "unused variable"-Warning mehr
   (`pkg/analysis/diagnostics.go`) — nötig, weil Pipe keine nullstelligen
   Funktionsaufrufe in der Grammatik hat und parameterlose ai_tools daher
   einen Dummy-Parameter brauchen.
