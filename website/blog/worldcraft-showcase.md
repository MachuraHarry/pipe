[lang:en]# 🌍 Worldcraft — an AI Game Where Your World Grows Forever[/lang]
[lang:de]# 🌍 Worldcraft — ein KI-Spiel, dessen Welt für immer mitwächst[/lang]

[lang:en]
**The sequel to our Aldenmoor adventure goes one step further: you don't play a fixed world — you *describe* one. Worldcraft generates your personal text adventure from a free-text briefing plus a player profile, saves it as a reusable JSON artifact, grows it on demand while you play (`erweitere welt …`), and hands you a fresh win condition whenever the old one is done (`neues abenteuer`). All of it built in pure Pipe, all of it running on the OpenCode Zen free tier at exactly $0.00 per session.**

Where [Aldenmoor](/blog/adventure-aldenmoor.html) answered *"can Pipe host an AI game?"*, Worldcraft answers a harder question: **can an LLM be a game designer without breaking the game?** The answer is yes — but only because the language lets us draw a hard line between what the model decides and what stays deterministic.
[/lang]

[lang:de]
**Der Nachfolger unseres Aldenmoor-Abenteuers geht einen Schritt weiter: Du spielst keine feste Welt — du *beschreibst* sie. Worldcraft erzeugt aus einem Freitext-Briefing plus Spielerprofil dein persönliches Textadventure, speichert es als wiederverwendbares JSON-Artefakt, lässt es während des Spielens auf Wunsch wachsen (`erweitere welt …`) und liefert jederzeit eine neue Siegesbedingung nach (`neues abenteuer`). Komplett in reinem Pipe gebaut, komplett auf dem OpenCode-Zen-Free-Tier — exakt $0,00 pro Session.**

Wo [Aldenmoor](/blog/adventure-aldenmoor.html) die Frage *"kann Pipe ein KI-Spiel tragen?"* beantwortete, fragt Worldcraft härter: **Kann ein LLM Spieldesigner sein, ohne das Spiel zu kaputtmachen?** Die Antwort ist Ja — aber nur, weil die Sprache eine harte Grenze ziehen kann zwischen dem, was das Modell entscheidet, und dem, was deterministisch bleibt.
[/lang]

[lang:en]## What the player experiences[/lang]
[lang:de]## Was der Spieler erlebt[/lang]

[lang:en]
```bash
cd worldcraft
../bin/pipe worldcraft.pipe

Weltnamen eingeben: nebeltal
Profil — Schwierigkeit (leicht/normal/schwer): leicht
Profil — Spielstil (kaempfer/haendlerin/erkunderin/lore): erkunderin
Beschreibe deine Welt:
> Schwebende Inseln über einem Wolkenmeer.
> Ein alter Himmelshafen, in dem noch ein Luftschiff verrottet.
> fertig
```

About two minutes later you stand on a rotting sky-harbor pier — with a torch and rope in your inventory (profile bonus), a trader selling potions somewhere in the fog, monsters lurking in rooms the model picked to fit your theme, and atmospheric room texts written specifically about *your* world. Then:

```
> erweitere welt einen verborgenen Wolkentempel
Die Welt wächst: 2 neue Räume (tpl_turm_2, tpl_dungeon), neue Begegnungen erwartet dich.

> neues abenteuer        (after winning)
Ein neues Abenteuer beginnt: 1 neue Aufgabe(n), mit einer frischen Quest im Log.
```
[/lang]

[lang:de]
```bash
cd worldcraft
../bin/pipe worldcraft.pipe

Weltnamen eingeben: nebeltal
Profil — Schwierigkeit (leicht/normal/schwer): leicht
Profil — Spielstil (kaempfer/haendlerin/erkunderin/lore): erkunderin
Beschreibe deine Welt:
> Schwebende Inseln über einem Wolkenmeer.
> Ein alter Himmelshafen, in dem noch ein Luftschiff verrottet.
> fertig
```

Zwei Minuten später stehst du auf einem verrotteten Himmelshafen-Steg — mit Fackel und Seil im Inventar (Profil-Bonus), einer Händlerin irgendwo im Nebel, Monstern in Räumen, die das Modell zum Thema passt, und atmosphärischen Raumtexten, die extra über *deine* Welt geschrieben wurden. Und dann:

```
> erweitere welt einen verborgenen Wolkentempel
Die Welt wächst: 2 neue Räume (tpl_turm_2, tpl_dungeon), neue Begegnungen erwartet dich.

> neues abenteuer        (nach einem Sieg)
Ein neues Abenteuer beginnt: 1 neue Aufgabe(n), mit einer frischen Quest im Log.
```
[/lang]

[lang:en]## The core idea: a fixed resource pack meets a free-text wish[/lang]
[lang:de]## Die Kernidee: festes Ressourcen-Paket trifft freier Text[/lang]

[lang:en]
The failure mode of "AI-generated games" is well known: the model invents a sword that deals 999 damage, places a shopkeeper in a room that doesn't exist, or writes a quest whose target was never placed. Worldcraft solves this structurally, not by prompt-hoping:

1. **A fixed resource pack** (`pack_core.json`) holds every building block — 16 items, 8 NPCs, 10 monsters, 10 room templates, 6 quest templates, 3 factions. Every entry carries mechanical values (HP, defense, loot, prices) *and* semantic tags.
2. **The LLM only selects.** Its design call receives a compact catalog (ids, names, tags — token-cheap) plus the player's briefing and profile, and returns which ids to use. It may also derive NPC personalities from the wish — but only *text*.
3. **Deterministic validation and normalization.** Pure-Pipe functions check every id against the pack, then *overwrite* whatever stats the model produced with pack values. Hallucinations die here, before they ever reach the database.
4. **Map validation.** A BFS from the start room must reach every room; exits must point to existing rooms; the win condition must be reachable. Invalid maps trigger up to three regeneration attempts — then a fully deterministic fallback world takes over so the player is never stuck.
[/lang]

[lang:de]
Das bekannte Problem „KI-generierter Spiele": Das Modell erfindet ein Schwert mit 999 Schaden, platziert einen Händler in einen nicht existierenden Raum oder schreibt eine Quest, deren Ziel nie platziert wurde. Worldcraft löst das strukturell statt per Prompt-Gebet:

1. **Ein festes Ressourcen-Paket** (`pack_core.json`) hält alle Bausteine — 16 Items, 8 NPCs, 10 Monster, 10 Raum-Templates, 6 Quest-Vorlagen, 3 Fraktionen. Jeder Eintrag trägt Mechanik-Werte (HP, Defense, Loot, Preise) *und* semantische Tags.
2. **Das LLM wählt nur aus.** Sein Design-Call bekommt einen kompakten Katalog (IDs, Namen, Tags — Token-sparend) plus Briefing und Profil des Spielers und liefert die gewählten IDs. Persönlichkeiten darf es ableiten — aber nur als *Text*.
3. **Deterministische Validierung und Normalisierung.** Pure-Pipe-Funktionen prüfen jede ID gegen das Paket und *überschreiben* danach alle Modell-Werte mit Pack-Werten. Halluzinationen sterben hier — bevor sie je die Datenbank erreichen.
4. **Karten-Validierung.** Eine BFS vom Startraum muss jeden Raum erreichen, Exits müssen auf existierende Räume zeigen, die Siegesbedingung muss erreichbar sein. Bei ungültigen Karten: bis zu drei Regenerierungs-Versuche — dann übernimmt eine komplett deterministische Musterwelt, damit niemand festhängt.
[/lang]

[lang:en]## Feature 1 — Worlds are artifacts, not sessions[/lang]
[lang:de]## Feature 1 — Welten sind Artefakte, keine Sessions[/lang]

[lang:en]
Every generated world is serialized into `worlds/<name>.json`: rooms, NPC placements, monster stats, quests, objectives, lore, your profile, an expansion counter. Loading a saved world re-seeds the SQLite database **without a single AI call** — returning to your world is instant and offline-capable. This split also means two save layers that never interfere: the *world* (`worlds/`) and your *run* inside it (`saves/`).
[/lang]

[lang:de]
Jede generierte Welt wird nach `worlds/<name>.json` serialisiert: Räume, NPC-Platzierungen, Monster-Stats, Quests, Objectives, Lore, dein Profil, ein Expansions-Zähler. Das Laden reseeden die SQLite-Datenbank **ohne einen einzigen KI-Call** — die Rückkehr in deine Welt ist sofort und offline-fähig. Damit gibt es auch zwei Save-Ebenen, die sich nie in die Quere kommen: die *Welt* (`worlds/`) und dein *Durchlauf* darin (`saves/`).
[/lang]

[lang:en]## Feature 2 — `erweitere welt`: growth as a validated pipeline[/lang]
[lang:de]## Feature 2 — `erweitere welt`: Wachstum als validierte Pipeline[/lang]

[lang:en]
The same generation machinery runs again *during* play, in miniature. One typed sentence spawns up to three new rooms, wired into the graph through free exit directions only (with automatic backlinks), populated exclusively with still-unused pack entries — and optionally carrying up to two new objectives from unused quest templates. Then the validators fire: connectivity BFS over the *combined* graph, anchor directions must be free, ids must be fresh, targets must actually be reachable. Only after that does anything touch the live database or the world file.

Guardrails are data, too: max 3 new rooms per command, max 25 total, enforced in pure Pipe. And when the free-tier model has a bad day — we watched it produce a `schraufel` (sic!) item and a market in a nonexistent template during testing — the attempt simply fails cleanly: *"Der Nebel ist noch zu dicht."* Nothing breaks; try again.
[/lang]

[lang:de]
Dieselbe Generierungsmaschinerie läuft im Spiel noch einmal im Kleinen. Ein Satz genügt, und es entstehen bis zu drei neue Räume — angebunden ausschließlich über freie Exit-Richtungen (mit automatischem Rückweg!), bevölkert nur aus noch ungenutzten Paket-Einträgen, optional mit bis zu zwei neuen Aufgaben aus unbenutzten Quest-Vorlagen. Danach feuern die Validatoren: Konnektivitäts-BFS über den *kombinierten* Graphen, Anker-Richtungen müssen frei sein, IDs müssen neu sein, Ziele müssen wirklich erreichbar sein. Erst danach fasft etwas die laufende Datenbank oder die Welt-Datei an.

Auch die Guardrails sind Daten: maximal 3 neue Räume pro Befehl, maximal 25 insgesamt, erzwungen in pure Pipe. Und hat das Free-Tier-Modell einen schlechten Tag — wir haben beim Testen einen Gegenstand namens `schraufel` (sic!) und einen Markt in einem nicht existierenden Template gesehen — scheitert der Versuch einfach sauber: *„Der Nebel ist noch zu dicht."* Nichts bricht. Nochmal versuchen.
[/lang]

[lang:en]## Feature 3 — `neues abenteuer`: objectives as data rows[/lang]
[lang:de]## Feature 3 — `neues abenteuer`: Siegesbedingungen als Datenzeilen[/lang]

[lang:en]
Win conditions aren't code — they're rows in an `objectives` table with three kinds: `give` (hand item X to NPC Y), `kill` (defeat monster X), `explore` (reach room X). Complete them all and `won` flips to 1. That design is what makes infinite play trivial: after a victory, `neues abenteuer` asks the model for *one* new objective built strictly from resources that exist and can be reached, validates reachability (a kill target counts if it's a placed monster *or its loot*), resets the flag, and persists the chapter into the world file.

Objectives can carry a `quest` field — completing the objective automatically closes its quest in the log. Old worlds without the field keep working unchanged; backwards compatibility is covered by tests.
[/lang]

[lang:de]
Siegesbedingungen sind kein Code — sie sind Zeilen in einer `objectives`-Tabelle mit drei Arten: `give` (Item X an NPC Y), `kill` (Monster X besiegen), `explore` (Raum X erreichen). Sind alle erledigt, springt `won` auf 1. Genau dieses Design macht unendliches Spielen trivial: Nach einem Sieg fragt `neues abenteuer` das Modell nach *einer* neuen Aufgabe, gebaut strikt aus Ressourcen, die existieren und erreichbar sind — die Erreichbarkeit wird validiert (ein kill-Ziel zählt, wenn es ein platziertes Monster *oder dessen Loot* ist), das Flag wird zurückgesetzt, das Kapitel landet in der Welt-Datei.

Objectives können ein `quest`-Feld tragen — erledigt der Spieler das Objective, schließt sich die gekoppelte Quest automatisch im Log. Alte Welten ohne das Feld laufen unverändert weiter; Rückwärtskompatibilität ist getestet.
[/lang]

[lang:en]## Feature 4 — Engineering around unreliable free models[/lang]
[lang:de]## Feature 4 — Engineering gegen unzuverlässige Free-Models[/lang]

[lang:en]
This is where most of the real work went. Three free models rotate through every generation step; each attempt is wrapped in `try`, each response goes through a home-grown `extract_json` that strips chain-of-thought prose from around the JSON object (free models love to think out loud, and one of them truncates mid-object). We hit every failure mode live while building this: 503s, timeouts, invented ids, truncated JSON, even a generated room description containing `'Finns Fliegerbedarf'` — which taught us that Pipe's mini SQL tokenizer does not support `''` escaping, so `sql_escape` now replaces apostrophes typographically. Each discovery became either a retry path, a validator rule, or a fallback — the game degrades gracefully instead of crashing.
[/lang]

[lang:de]
Hier steckte die meiste echte Arbeit. Drei Free-Modelle rotieren durch jeden Generierungsschritt; jeder Versuch ist in `try` gepackt, jede Antwort läuft durch ein eigenes `extract_json`, das Chain-of-Thought-Prosa um das JSON-Objekt herum abschneidet (Free-Models denken gern laut, eines davon bricht mitten im Objekt ab). Wir haben beim Bauen alle Fehlermodi live getroffen: 503er, Timeouts, erfundene IDs, abgeschnittenes JSON — sogar eine generierte Raumbeschreibung mit `'Finns Fliegerbedarf'`, die uns lehrte, dass Pipes Mini-SQL-Tokenizer kein `''`-Escaping unterstützt. `sql_escape` ersetzt Apostrophe jetzt typografisch. Jede Entdeckung wurde zum Retry-Pfad, zur Validator-Regel oder zum Fallback — das Spiel degradiert graceful statt zu crashen.
[/lang]

[lang:en]## Feature 5 — tested without the cloud[/lang]
[lang:de]## Feature 5 — getestet ohne Cloud[/lang]

[lang:en]
Everything deterministic lives in importable modules separate from the game loop, so `../bin/pipe -test` runs 40+ assertions with zero API calls: pack/design/map validation, BFS reachability, profile bonuses, wish-persona overrides (text changes, mechanics don't), expansion merging including backlinks and caps, trade math, all three objective types, save/load round trips for both layers, and backwards compatibility with pre-2.0 world files. The AI-facing layer is thin by construction: registered `ai_tool`s wrap pure functions, so there's simply nothing nondeterministic to test.
[/lang]

[lang:de]
Alles Deterministische liegt in importierbaren Modulen getrennt vom Game Loop, deshalb laufen mit `../bin/pipe -test` über 40 Assertions ohne einen einzigen API-Call: Pack-/Design-/Karten-Validierung, BFS-Erreichbarkeit, Profil-Boni, Wunsch-Personas (Text ändert sich, Mechanik nicht), Expansions-Merge inklusive Backlinks und Caps, Handels-Mathe, alle drei Objective-Typen, Save/Load-Roundtrips beider Ebenen und Rückwärtskompatibilität mit Vor-2.0-Weltdateien. Die KI-Schicht ist konstruktiv dünn: registrierte `ai_tool`s wrappen reine Funktionen — es gibt schlicht nichts Nichtdeterministisches zu testen.
[/lang]

[lang:en]## What a session actually costs[/lang]
[lang:de]## Was eine Session wirklich kostet[/lang]

[lang:en]
World generation runs ~10–16 API calls (~13k–22k tokens); every later visit to the same world needs none; expansions and `neues abenteuer` add a few calls each. Every session ends with Pipe's cost trace reading `$0.000000`. No key, no account — the provider line is still just `ai_provider "opencode"`, with a sandbox whitelist locking the agent to that single host and no shell access at all.
[/lang]

[lang:de]
Die Weltgenerierung kostet ~10–16 API-Calls (~13k–22k Tokens); jeder spätere Besuch derselben Welt keinen einzigen; Erweiterungen und `neues abenteuer` je ein paar Calls. Am Ende jeder Session steht in Pipes Cost Trace `$0.000000`. Kein Key, kein Account — die Provider-Zeile ist weiter nur `ai_provider "opencode"`, mit einer Sandbox-Whitelist, die den Agenten auf diesen einen Host sperrt, ohne jede Shell.
[/lang]

[lang:en]## Why it matters as a showcase[/lang]
[lang:de]## Warum das ein Showcase ist[/lang]

[lang:en]
Aldenmoor proved an LLM can *narrate* a deterministic world. Worldcraft proves it can *design* one — safely — and keep redesigning it forever, because Pipe gives you the pieces such a contract needs:

| Need | Pipe primitive |
|---|---|
| Structured generation | `ai_chat` + own JSON extraction, model rotation |
| Agency under lock-down | `ai_tool` + `ai_with_tools` behind `sandbox_profile` |
| Persistent state | sqlite module, in-memory DB as single source of truth |
| Parallel atmosphere | `ai_batch` |
| Memory compression | `summarize` |
| Deterministic guarantees | pure-Pipe validators + 40+ tests via `pipe -test` |

The lesson generalizes beyond games: wherever you want an LLM to make open-ended decisions, give it a catalog, validate the selection, normalize against ground truth — and let the language, not the prompt, hold the boundary.
[/lang]

[lang:de]
Aldenmoor hat gezeigt, dass ein LLM eine deterministische Welt *erzählen* kann. Worldcraft zeigt, dass es eine *entwerfen* kann — gefahrlos — und sie für immer weiter entwerfen darf, weil Pipe genau die Bauteile für diesen Vertrag mitbringt:

| Bedürfnis | Pipe-Primitive |
|---|---|
| Strukturierte Generierung | `ai_chat` + eigene JSON-Extraktion, Modell-Rotation |
| Handlungsfähigkeit hinter Schloss | `ai_tool` + `ai_with_tools` hinter `sandbox_profile` |
| Persistenter Zustand | sqlite-Modul, In-Memory-DB als Single Source of Truth |
| Parallele Atmosphäre | `ai_batch` |
| Speicherkompression | `summarize` |
| Deterministische Garantien | Pure-Pipe-Validatoren + 40+ Tests via `pipe -test` |

Die Lehre reicht über Spiele hinaus: Wo immer ein LLM offene Entscheidungen treffen soll — gib ihm einen Katalog, validiere die Auswahl, normalisiere gegen die Ground Truth — und lass die Sprache, nicht den Prompt, die Grenze halten.
[/lang]

[lang:en]## Try it[/lang]
[lang:de]## Ausprobieren[/lang]

[lang:en]
```bash
git clone https://github.com/MachuraHarry/pipe && cd pipe && make build
cd worldcraft
../bin/pipe worldcraft.pipe
# Weltname: nebeltal   (oder "beispielwelt" für den Offline-Test)
```

Full documentation lives in `worldcraft/README.md`; the complete player's handbook covers profiles, commands, trading, combat, and world management. Existing worlds from older versions load unchanged — bring your own.
[/lang]

[lang:de]
```bash
git clone https://github.com/MachuraHarry/pipe && cd pipe && make build
cd worldcraft
../bin/pipe worldcraft.pipe
# Weltname: nebeltal   (oder "beispielwelt" für den Offline-Test)
```

Die komplette Doku liegt in `worldcraft/README.md`; das vollständige Spielerhandbuch deckt Profile, Befehle, Handel, Kampf und Weltverwaltung ab. Bestehende Welten aus älteren Versionen laden unverändert — bring deine eigene mit.
[/lang]
