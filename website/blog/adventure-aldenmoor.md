[lang:en]# 🏰 Building a Keyless AI Text Adventure in Pure Pipe — "Die Ruine von Aldenmoor"[/lang]
[lang:de]# 🏰 Ein schlüsselloses KI-Textadventure in reinem Pipe — "Die Ruine von Aldenmoor"[/lang]

[lang:en]
**A complete, playable fantasy text adventure — tool-calling, RAG, SQLite state, `summarize`, and a locked-down sandbox — written in ~600 lines of Pipe and run entirely on the OpenCode Zen free tier. Zero API keys, zero cost. This post walks through every Pipe feature the game leans on, plus the two tricks we used to make a latency-heavy AI game feel instant.**

Text adventures are the perfect stress test for an agentic language: the game *needs* the LLM to narrate, but the *world* must be deterministic, persistent, and consistent. That tension is exactly what Pipe's toolkit is built for — `ai_with_tools` for agency, `embed`/`nearest` for memory, `db_exec` for state, `summarize` for compression, and `sandbox_profile` for safety. Aldenmoor is a demo that exercises all of them at once.

## What the game is

You wake at the entrance of a cursed ruin. A torch, seven rooms, a locked treasure chamber guarded by a monster, and a silent watchman named Ser Aldric who is bound by a curse. Your goal: return a piece of the unspoiled heritage of Aldenmoor to the watcher — freely, not by force — to break the curse.

It is a real RPG: movement, item pickup, combat, quests, and a small faction-reputation system. And every turn, an LLM acts as the *narrator*, choosing which tools to call.

```bash
cd adventure
/home/droid/pipe/bin/pipe adventure.pipe
# > nimm die fackel
# > geh nach norden
# > geh nach osten
# > greife den kammerling an
# > nimm die muenze
# > geh nach westen
# > gib dem waechter die muenze
```

The win condition is a pure-Pipe state check: when the `muenze` reaches the `waechter`, the curse breaks and the run ends in victory.

## Architecture: three layers

```
adventure.pipe   game loop, provider setup, feedback (spinner + fast-path)
tools.pipe       the ai_tool functions + registrations
state.pipe       sqlite schema, seed, save/load
lore.pipe        RAG over world lore
rooms.json       the 7-room world
world/          lore documents (history, factions, the watcher)
```

The split is deliberate. `state.pipe` and `tools.pipe` are imported by the test suite **without starting the game loop**, so every world rule is unit-testable in pure Pipe (30 tests, no AI calls).

## Feature 1 — Tool-calling without an SDK

The narrator is one `ai_with_tools` call per turn. You register ordinary Pipe functions as tools and let the model decide when to call them:

```pipe
ai_tool "move_to" "Bewege den Spieler durch einen Ausgang"
    {exit_name: "Himmelsrichtung, z.B. norden"} move_to
ai_tool "take_item" "Hebe einen Gegenstand auf"
    {item_name: "z.B. fackel"} take_item
ai_tool "give_item" "Biete einem NPC einen Gegenstand an"
    {item_name: "Name", npc: "z.B. waechter"} give_item
ai_tool "attack_enemy" "Greife ein Untier an"
    {target: "z.B. kammerling"} attack_enemy

-- one turn:
antwort: ai_with_tools NARRATOR_SYSTEM player_input
```

The model sees the tool *schemas* (name, description, typed parameters) and returns structured calls. Pipe executes them and feeds the results back. The functions themselves are plain deterministic Pipe — `move_to` just mutates the player's room in SQLite and returns the new description. **The LLM never holds the world state; it only narrates the consequences.**

## Feature 2 — RAG so the world stays consistent

Without memory, the narrator invents rooms, items, and lore on the fly. Aldenmoor instead builds a vector index over its own lore files once at startup:

```pipe
LORE_FILES: ["world/lore_history.txt", "world/lore_factions.txt", "world/npcs/waechter.txt"]

fn build_lore_index _
    docs: []
    for f in LORE_FILES
        push docs (read_file f)
    vecs: embed_batch docs
    set LORE_INDEX 0 {docs: docs, vectors: vecs}

fn lore_context query
    idx: at LORE_INDEX 0
    qv: embed query
    top: nearest qv (get idx "vectors") 1
    -- concatenate the matched lore snippets
```

When the player talks to Ser Aldric, the watcher's `talk_to` pulls the most relevant lore and feeds it as grounding context to `ask`. The result: the knight mentions the *real* history of Aldenmoor, not whatever the model dreamed up that turn.

**Keyless detail:** the OpenCode Zen provider has no embeddings endpoint, so `embed`/`embed_batch` transparently fall back to Pipe's local hash embedder. RAG still works — just keyword-ish in quality — with no API cost and no setup.

## Feature 3 — SQLite as the single source of truth

Everything that matters lives in SQLite: `player`, `rooms`, `monsters`, `quests`, `reputation`, `npc_memory`, and `flavor` (more on that later). A small helper escapes strings so lore with apostrophes can't break a query:

```pipe
fn sql_escape s
    replace (replace s "'" "''") "\"" "\\\""

fn db_query h sql
    unwrap (db_exec h sql)   -- pure-Pipe SQL engine, no driver
```

`save_game` serialises the whole world into a JSON blob and writes it to `.pipe_sandbox/saves/`; `load_game` replays it row by row. Because the engine is pure Pipe, saves work with zero dependencies.

## Feature 4 — summarize compresses NPC memory

Dialogues would otherwise grow the `npc_memory` field without bound. Every third exchange we compress it:

```pipe
turns: npc_turns npc
if (turns % 3) == 2
    mem: summarize (alt ++ "\nSpieler: " ++ last_message)
else
    mem: alt           -- keep as-is, but the latest line is still injected live
```

The trick: even on the two turns where we *skip* compression, the most recent player line is injected directly into the prompt — so the watcher never "forgets" what you just said, while the long-term summary stays cheap.

## Feature 5 — A sandbox that locks everything but the AI

The world files are read *before* the sandbox is raised, then the filesystem, shell, and free network are all removed:

```pipe
sandbox_profile "game" {fs: "temp-only", network: true,
    network_whitelist: ["opcode.ai"], exec: false, ai: true}
set_sandbox "game"
```

Now the LLM can only reach the OpenCode Zen endpoint (`ai: true` plus the whitelist), cannot spawn a shell, and can only write to temp. The agent is genuinely constrained — which matters the moment you let a model drive game logic.

## Feature 6 — Keyless free tier via OpenCode Zen

No API key, no credit card. The provider is selected in one line:

```pipe
ai_provider "opencode"
FREE_MODELS: ["x-preview-f-free", "laguna-s-2.1-free"]
```

A small probe picks the first model that answers; the turn loop rotates through the list on failure, so a 503 from one endpoint never blocks the player. The whole playthrough costs **$0**.

## Feature 7 — Making an AI game feel instant

This was the real engineering. A naive turn is three sequential AI calls (narrator + dialogue `ask` + `summarize`), and free endpoints can be slow. Two layers of work fixed the feel:

**a) Fast-path for common commands.** Direction, pickup, inventory, quests, and combat are parsed by pure Pipe — no LLM at all:

```pipe
fn fast_command cmd
    t: lower (trim cmd)
    w: split t " "
    erstes: at w 0
    if erstes == "norden" || erstes == "n" || (erstes == "geh" && contains t "norden")
        {handled: true, out: move_to "norden"}
    else if erstes == "nimm" && len w > 1
        {handled: true, out: take_item (rest_words t)}
    -- ... inventar, quests, greife, etc.
    else
        {handled: false, out: ""}   -- fall through to the LLM narrator
```

Typed `> nimm die fackel` and the torch is in your hand in under 5 ms. The LLM only runs for free text and dialogue.

**b) Pre-generated room atmosphere, in parallel.** At startup we generate an atmospheric description for every room at once with `ai_batch` (all 7 in one parallel round-trip), and store the result in the `flavor` table. During play, `look` returns the cached text instantly — no waiting for the narrator to describe a room you just walked into.

**c) A spinner that proves the game is alive.** Pipe has real concurrency (`spawn` + channels). While the narrator thinks, a background task animates dots:

```pipe
fn think_spinner ch
    v: try_recv ch
    while v == nil
        print_raw "."
        sleep 400
        v: try_recv ch
    print_raw "\n"

ch: chan 1
spawn think_spinner ch
antwort: ai_with_tools NARRATOR_SYSTEM text
send ch 1
```

The player sees `Der Wind trägt einen Hauch Glockenklang herüber. ................` instead of a frozen prompt.

## How it all fits together

One turn, in order:

1. `fast_command` checks for an instant local action. Hit → result printed in <5 ms, done.
2. Miss → a random ambient line prints, the spinner spawns, `ai_with_tools` runs the narrator.
3. The narrator calls tools (`move_to`, `take_item`, `attack_enemy`, `give_item`…); each mutates SQLite and returns factual text.
4. The narration wraps the tool results; the win flag is re-checked; if the curse broke, the epilogue prints.

## Why Pipe, not a Python notebook

- **One binary.** `bin/pipe` is ~8 MB; no venv, no `pip install`, no model server.
- **AI is a language primitive.** `ai_with_tools`, `embed_batch`, `summarize`, `ai_tool` are builtins — no LangChain, no SDK wiring.
- **Safety is structural.** `sandbox_profile` constrains filesystem, network, and shell *as a language construct*, so the agent can't escape into your machine.
- **It tests without the cloud.** The entire world logic runs under `pipe -test` with zero API calls — 30 green tests cover movement, combat, quests, reputation, and that the win condition fires.

## Try it

```bash
git clone https://github.com/MachuraHarry/pipe
cd pipe/adventure
../bin/pipe adventure.pipe
```

The full source — `adventure.pipe`, `state.pipe`, `tools.pipe`, `lore.pipe`, `rooms.json`, the `world/` lore, and the 30-test suite — is in the repository. It is a small, readable example of how far you can get with agentic AI, persistence, and a real sandbox, all expressed in one language and run for free.
[/lang]

[lang:de]
**Ein komplettes, spielbares Fantasy-Textadventure — Tool-Calling, RAG, SQLite-State, `summarize` und eine verriegelte Sandbox — geschrieben in ~600 Zeilen Pipe und komplett auf dem kostenlosen OpenCode-Zen-Tier laufend. Null API-Keys, null Kosten. Dieser Post geht durch jede Pipe-Funktion, auf die das Spiel baut, plus die zwei Tricks, mit denen wir ein latenzlastiges KI-Spiel augenblicklich wirken ließen.**

Textadventures sind der perfekte Stresstest für eine agentische Sprache: Das Spiel *braucht* die LLM zum Erzählen, aber die *Welt* muss deterministisch, persistent und konsistent bleiben. Genau diesen Spannungsbogen bedient Pipes Werkzeugkasten — `ai_with_tools` für Agentizität, `embed`/`nearest` für Gedächtnis, `db_exec` für State, `summarize` für Kompression und `sandbox_profile` für Sicherheit. Aldenmoor ist eine Demo, die all das auf einmal trainieren.

## Was das Spiel ist

Du erwachst am Eingang einer verfluchten Ruine. Eine Fackel, sieben Räume, eine verschlossene Schatzkammer, bewacht von einem Untier, und ein schweigsamer Wächter namens Ser Aldric, der an einen Fluch gebunden ist. Dein Ziel: ein Stück des unberührten Erbes von Aldenmoor dem Wächter *freiwillig* zurückzugeben — nicht mit Gewalt — um den Fluch zu brechen.

Es ist ein echtes RPG: Bewegung, Gegenstandsaufnahme, Kampf, Quests und ein kleines Fraktions-Reputationssystem. Und jede Runde agiert eine LLM als *Erzähler*, die selbst entscheidet, welche Werkzeuge sie aufruft.

```bash
cd adventure
/home/droid/pipe/bin/pipe adventure.pipe
# > nimm die fackel
# > geh nach norden
# > geh nach osten
# > greife den kammerling an
# > nimm die muenze
# > geh nach westen
# > gib dem waechter die muenze
```

Die Siegbedingung ist ein reiner Pipe-State-Check: erreicht die `muenze` den `waechter`, bricht der Fluch und der Lauf endet im Sieg.

## Architektur: drei Schichten

```
adventure.pipe   Spielschleife, Provider-Setup, Feedback (Spinner + Fast-Path)
tools.pipe       die ai_tool-Funktionen + Registrierungen
state.pipe       SQLite-Schema, Seed, Save/Load
lore.pipe        RAG über Welt-Lore
rooms.json       die 7-Räume-Welt
world/          Lore-Dokumente (Geschichte, Fraktionen, der Wächter)
```

Die Trennung ist bewusst. `state.pipe` und `tools.pipe` werden von der Testsuite importiert, **ohne die Spielschleife zu starten** — jede Weltregel ist damit in reinem Pipe unit-testbar (30 Tests, keine KI-Calls).

## Feature 1 — Tool-Calling ohne SDK

Der Erzähler ist pro Runde ein einziger `ai_with_tools`-Call. Du registrierst gewöhnliche Pipe-Funktionen als Tools und überlässt dem Modell die Entscheidung, wann es sie aufruft:

```pipe
ai_tool "move_to" "Bewege den Spieler durch einen Ausgang"
    {exit_name: "Himmelsrichtung, z.B. norden"} move_to
ai_tool "take_item" "Hebe einen Gegenstand auf"
    {item_name: "z.B. fackel"} take_item
ai_tool "give_item" "Biete einem NPC einen Gegenstand an"
    {item_name: "Name", npc: "z.B. waechter"} give_item
ai_tool "attack_enemy" "Greife ein Untier an"
    {target: "z.B. kammerling"} attack_enemy

-- eine Runde:
antwort: ai_with_tools NARRATOR_SYSTEM spieler_eingabe
```

Das Modell sieht die Tool-*Schemas* (Name, Beschreibung, typisierte Parameter) und liefert strukturierte Aufrufe. Pipe führt sie aus und füttert die Ergebnisse zurück. Die Funktionen selbst sind schlichtes, deterministisches Pipe — `move_to` mutiert nur den Raum des Spielers in SQLite und liefert die neue Beschreibung. **Die LLM hält niemals den Welt-State; sie erzählt nur die Konsequenzen.**

## Feature 2 — RAG, damit die Welt konsistent bleibt

Ohne Gedächtnis erfindet der Erzähler Räume, Gegenstände und Lore aus dem Stegreif. Aldenmoor baut stattdessen einmalig beim Start einen Vektor-Index über seine eigenen Loredateien:

```pipe
LORE_FILES: ["world/lore_history.txt", "world/lore_factions.txt", "world/npcs/waechter.txt"]

fn build_lore_index _
    docs: []
    for f in LORE_FILES
        push docs (read_file f)
    vecs: embed_batch docs
    set LORE_INDEX 0 {docs: docs, vectors: vecs}

fn lore_context query
    idx: at LORE_INDEX 0
    qv: embed query
    top: nearest qv (get idx "vectors") 1
    -- die passenden Lore-Schnipsel verketten
```

Spricht der Spieler mit Ser Aldric, zieht dessen `talk_to` die relevanteste Lore und füttert sie als Grounding-Kontext an `ask`. Ergebnis: der Ritter erwähnt die *echte* Geschichte Aldenmoors, nicht das, was das Modell in der Runde gerade träumt.

**Schlüsselloser Detail:** Der OpenCode-Zen-Provider hat keinen Embeddings-Endpoint, also fällt `embed`/`embed_batch` transparent auf Pipe's lokalen Hash-Embedder zurück. RAG funktioniert trotzdem — nur qualitativ eher keyword-artig — ohne API-Kosten und ohne Setup.

## Feature 3 — SQLite als alleinige Wahrheitsquelle

Alles, was zählt, lebt in SQLite: `player`, `rooms`, `monsters`, `quests`, `reputation`, `npc_memory` und `flavor` (dazu später). Ein kleiner Helper escaped Strings, damit Lore mit Apostrophen keine Query bricht:

```pipe
fn sql_escape s
    replace (replace s "'" "''") "\"" "\\\""

fn db_query h sql
    unwrap (db_exec h sql)   -- reine-Pipe-SQL-Engine, kein Treiber
```

`save_game` serialisiert die ganze Welt in einen JSON-Blob und schreibt ihn nach `.pipe_sandbox/saves/`; `load_game` spielt ihn zeilenweise zurück. Weil die Engine reines Pipe ist, funktionieren Saves ohne Abhängigkeiten.

## Feature 4 — summarize komprimiert NPC-Gedächtnis

Dialoge würden das `npc_memory`-Feld sonst ungebremst wachsen lassen. Jeden dritten Austausch komprimieren wir es:

```pipe
turns: npc_turns npc
if (turns % 3) == 2
    mem: summarize (alt ++ "\nSpieler: " ++ last_message)
else
    mem: alt           -- unverändert lassen, aber die letzte Zeile wird live injiziert
```

Der Trick: selbst in den zwei Runden, in denen wir die Kompression *überspringen*, wird die aktuellste Spielerzeile direkt in den Prompt injiziert — der Wächter „vergisst" also nie, was du gerade gesagt hast, während die Langzeit-Zusammenfassung günstig bleibt.

## Feature 5 — Eine Sandbox, die alles außer der KI sperrt

Die Weltdateien werden gelesen, *bevor* die Sandbox gehoben wird; danach werden Dateisystem, Shell und freies Netz entfernt:

```pipe
sandbox_profile "game" {fs: "temp-only", network: true,
    network_whitelist: ["opencode.ai"], exec: false, ai: true}
set_sandbox "game"
```

Jetzt kann die LLM nur noch den OpenCode-Zen-Endpoint erreichen (`ai: true` plus Whitelist), keine Shell spawnen und nur nach Temp schreiben. Der Agent ist echt constrainiert — was genau dann zählt, wenn du ein Modell Spiellogik treiben lässt.

## Feature 6 — Schlüsselloses Free-Tier via OpenCode Zen

Kein API-Key, keine Kreditkarte. Der Provider steht in einer Zeile:

```pipe
ai_provider "opencode"
FREE_MODELS: ["x-preview-f-free", "laguna-s-2.1-free"]
```

Eine kleine Sonde wählt das erste Modell, das antwortet; die Rundenschleife rotiert bei Fehlschlag durch die Liste, sodass ein 503 eines Endpunkts den Spieler nie blockiert. Der komplette Durchlauf kostet **$0**.

## Feature 7 — Ein KI-Spiel augenblicklich wirken lassen

Das war die eigentliche Ingenieursarbeit. Eine naive Runde sind drei sequenzielle KI-Calls (Erzähler + Dialog-`ask` + `summarize`), und Free-Endpunkte können langsam sein. Zwei Arbeitsschichten fixten das Gefühl:

**a) Fast-Path für Standardbefehle.** Richtung, Aufnehmen, Inventar, Quests und Kampf werden von reinem Pipe geparst — gar keine LLM:

```pipe
fn fast_command cmd
    t: lower (trim cmd)
    w: split t " "
    erstes: at w 0
    if erstes == "norden" || erstes == "n" || (erstes == "geh" && contains t "norden")
        {handled: true, out: move_to "norden"}
    else if erstes == "nimm" && len w > 1
        {handled: true, out: take_item (rest_words t)}
    -- ... inventar, quests, greife, etc.
    else
        {handled: false, out: ""}   -- an den LLM-Erzähler durchreichen
```

Tippt man `> nimm die fackel`, ist die Fackel in unter 5 ms im Beutel. Die LLM läuft nur für Freitext und Dialog.

**b) Vorgenerierte Raumatmosphäre, parallel.** Beim Start erzeugen wir mit `ai_batch` auf einmal eine atmosphärische Beschreibung für jeden Raum (alle 7 in einem parallelen Round-Trip) und speichern das Ergebnis in der `flavor`-Tabelle. Während des Spiels liefert `look` den gecachten Text sofort — kein Warten auf den Erzähler, der einen Raum beschreibt, den du gerade betreten hast.

**c) Ein Spinner, der beweist, dass das Spiel lebt.** Pipe hat echte Concurrency (`spawn` + Channels). Während der Erzähler denkt, animiert ein Hintergrundtask Punkte:

```pipe
fn think_spinner ch
    v: try_recv ch
    while v == nil
        print_raw "."
        sleep 400
        v: try_recv ch
    print_raw "\n"

ch: chan 1
spawn think_spinner ch
antwort: ai_with_tools NARRATOR_SYSTEM text
send ch 1
```

Der Spieler sieht `Der Wind trägt einen Hauch Glockenklang herüber. ................` statt eines eingefrorenen Prompts.

## Wie alles zusammenspielt

Eine Runde, in Reihenfolge:

1. `fast_command` prüft auf eine instant lokale Aktion. Treffer → Ergebnis in <5 ms ausgegeben, fertig.
2. Kein Treffer → eine zufällige Ambiente-Zeile wird ausgegeben, der Spinner startet, `ai_with_tools` lässt den Erzähler laufen.
3. Der Erzähler ruft Tools auf (`move_to`, `take_item`, `attack_enemy`, `give_item`…); jedes mutiert SQLite und liefert Faktentext.
4. Die Erzählung rahmt die Tool-Ergebnisse; die Sieg-Flagge wird neu geprüft; brach der Fluch, druckt die Epilog.

## Warum Pipe, nicht ein Python-Notebook

- **Eine Binary.** `bin/pipe` ist ~8 MB; kein venv, kein `pip install`, kein Modellserver.
- **KI ist ein Sprach-Primitive.** `ai_with_tools`, `embed_batch`, `summarize`, `ai_tool` sind Builtins — kein LangChain, kein SDK-Verkabeln.
- **Sicherheit ist strukturell.** `sandbox_profile` constrainiert Dateisystem, Netz und Shell *als Sprachkonstrukt*, sodass der Agent nicht in deine Maschine ausbrechen kann.
- **Es testet ohne Cloud.** Die gesamte Weltlogik läuft unter `pipe -test` mit null API-Calls — 30 grüne Tests decken Bewegung, Kampf, Quests, Reputation und dass die Siegbedingung feuert.

## Ausprobieren

```bash
git clone https://github.com/MachuraHarry/pipe
cd pipe/adventure
../bin/pipe adventure.pipe
```

Der komplette Source — `adventure.pipe`, `state.pipe`, `tools.pipe`, `lore.pipe`, `rooms.json`, die `world/`-Lore und die 30-Test-Suite — liegt im Repository. Es ist ein kleines, lesbares Beispiel dafür, wie weit man mit agentischer KI, Persistenz und echter Sandbox kommt, alles in einer Sprache ausgedrückt und kostenlos laufend.
[/lang]
