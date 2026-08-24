# Worldcraft

KI-gesteuertes Textadventure mit **generierbaren, wiederverwendbaren und
wachsenden Welten** — komplett gebaut in [Pipe](https://github.com/MachuraHarry/pipe).
Läuft kostenlos und ohne API-Key über den OpenCode-Zen-Provider (Free-Tier).

Anders als das Referenzspiel `../adventure/` (feste Welt „Aldenmoor") beschreibst
du hier beim Start deine eigene Welt — Orte, Stimmung, Bewohner, Geschichte —
und die KI baut daraus ein spielbares Abenteuer. Die Spielmechanik kommt dabei
nicht aus der KI, sondern aus einem **festen Ressourcen-Paket**
(`pack_core.json`): Die KI wählt nur Einträge aus und schreibt Lore drumherum.
HP, Schaden, Preise und Loot bleiben fix → balanciert und deterministisch testbar.

## Spielen

```bash
# aus diesem Verzeichnis:
../bin/pipe worldcraft.pipe
```

Ablauf beim Start:

1. **Weltnamen eingeben** — existiert `worlds/<name>.json`, wird die Welt
   **ohne einen einzigen KI-Call** geladen (sofort spielbar). Existiert sie
   nicht, generiert die KI sie neu und speichert sie dort.
2. **Spieler-Profil**: Schwierigkeit (leicht/normal/schwer) und Spielstil
   (kaempfer/haendlerin/erkunderin/lore) — deterministische Start-Boni:
   | Stil | Bonus |
   |---|---|
   | kaempfer | Startet mit `schwert` |
   | haendlerin | +15 Startgold |
   | erkunderin | Startet mit `fackel` + `seil` |
   | lore | Extra-Lore-Hinweis + `fackel` |
   Schwierigkeit: leicht +15 Gold, schwer −5 Gold (min. 5).
3. **Welt-Briefing über mehrere Zeilen** (mit `fertig` abschließen; sofort
   `fertig` = Musterwelt). Aus dem Wunsch leitet die KI zusätzlich eigene
   NPC-Persönlichkeiten ab (Mechanik bleibt unangetastet).
4. Freitext spielen. `ende` beendet.

### Welt-Wachstum — die Welt wächst mit dir

Im Spiel jederzeit:

```
> erweitere welt einen verborgenen Wolkentempel
> erweitere welt einen Dungeon Richtung Westen
Die Welt wächst: 2 neue Räume (tpl_turm_2, tpl_dungeon_2) ...
```

Die KI generiert 1–3 neue Räume (freie Templates bevorzugt, sonst Klone wie
`tpl_dungeon_2`), hängt sie an freie Richtungen bestehender Räume, platziert
noch ungenutzte NPCs/Monster/Items aus dem Pack und kann **bis zu 2 neue
Aufgaben** aus unbenutzten Quest-Vorlagen mitbringen — alles validiert (BFS-
Konnektivität, erreichbare Ziele, keine Exit-Kollisionen, Raum-Cap 25) und
**direkt in die Welt-Datei zurückgespeichert**.

### Neues Abenteuer

Nach einem Sieg endet das Spiel nicht mehr. Mit einem Befehl gibt dir die Welt
eine frische, garantiert erreichbare Siegesbedingung (optional mit neuer Quest):

```
> neues abenteuer
Ein neues Abenteuer beginnt: 1 neue Aufgabe(n), mit einer frischen Quest im Log.
```

Quest-Kopplung: Erledigt der Spieler ein Objective, schließt sich automatisch
die zugehörige Quest im Log (Feld `quest` am Objective; alte Welten ohne das
Feld funktionieren unverändert).

## Zwei Ebenen: Welt vs. Spielstand

| Ebene | Artefakt | Effekt |
|---|---|---|
| **Welt** | `worlds/<name>.json` | Karte, NPCs, Monster, Quests, Objectives, Lore, Profil, Expansions-Zähler — wiederverwendbar, Laden ohne KI |
| **Spielstand** | `saves/<name>.json` | Player-, Inventar-, Quest-Zustand innerhalb einer Welt |

Die mitgelieferte `worlds/beispielwelt.json` macht das Spiel auch ganz ohne
Netzwerk/API testbar (dieselbe Welt erzeugt `fallback_world` auch automatisch,
wenn die KI nicht erreichbar ist oder wiederholt ungültige Karten liefert).

## Architektur

```
worldcraft.pipe      Entry: Provider-Setup -> Profil -> Welt laden ODER
                     generieren -> Game Loop (inkl. "erweitere"-Fastpath)
gen_world.pipe       Worldgen-Pipeline + Validierung + Fallback-Welt +
                     Expansions-Pipeline (Räume, Quests, Objectives) +
                     neues-abenteuer-Generator
state.pipe           sqlite-Spielzustand (In-Memory), datengetrieben,
                     Row-Level-Seeder, Objective→Quest-Kopplung
tools.pipe           ai_tool-Registrierungen inkl. Handel + DOC/PACK-Holder
lore.pipe            RAG-Layer über die generierte Welt-Lore
pack_core.json       festes Ressourcen-Paket: 16 Items, 8 NPCs, 10 Monster,
                     10 Raum-Templates, 6 Quest-Vorlagen, 3 Fraktionen
worlds/              gespeicherte Welten als JSON-Artefakte
worldcraft_test.pipe 40+ Tests via `pipe -test` (gegen In-Memory-DB, ohne KI)
```

### Worldgen-Pipeline (`gen_world.pipe`)

1. **Design** (`ai_chat_json`): Nutzerwunsch + Profil + kompakter Pack-Katalog
   (nur IDs/Namen/Tags — Token sparen) → Design-JSON mit ausgewählten IDs und
   Wunsch-Personas.
2. **Validierung** (pure Pipe): alle IDs existieren im Pack, Raumzahl 5–12;
   bei Verstoß max. 3 Versuche.
3. **Karte** (`ai_chat_json`): Raumgraph nur aus den freigegebenen IDs.
4. **Karten-Validierung** (pure Pipe): BFS-Konnektivität vom Startraum,
   Exits zeigen auf existierende Räume, Siegesbedingung erreichbar.
5. **Normalisierung** (pure Pipe): Monster-Stats, NPC-Rollen und Loot werden
   **immer** aus dem Pack überschrieben; Wunsch-Personas überschreiben nur Text.
6. **Profil-Anwendung** (pure Pipe): Start-Boni laut Spieler-Profil.
7. **Lore-Anreicherung** (`ai_batch`, parallel): Raum-Flavor + NPC-Backstories.
8. **Speichern** als `worlds/<name>.json`, Seed in sqlite, Sandbox-Lock.

### Expansions-Pipeline (`erweitere welt ...`)

Gleiches Muster zur Laufzeit: `gen_expansion` → `validate_expansion`
(Anker existiert, Richtung frei, IDs neu/unbenutzt, Ziele erreichbar,
Quest-Vorlagen frisch, Cap) → `merge_expansion` (inkrementelle DB-Seeds via
Row-Level-Seeder + Merge ins Doc inkl. gekoppelter Objectives) →
`save_current_world`. `neues abenteuer` nutzt dieselbe Pipeline mit
`gen_neue_objektive` + `reset_won`.

Prinzip: Das LLM erzählt und konfiguriert nur. **Jede Zustandsänderung läuft
über ein registriertes `ai_tool`** bzw. den Fastpath, der reine Pipe-Funktionen
aufruft, welche die sqlite-DB mutieren. Siegesbedingungen sind datengetriebene
`objectives` (kind: `give` / `kill` / `explore`) — keine Code-Änderungen nötig.

## Technik-Notizen

- **Provider**: `ai_provider "opencode"` — Free-Tier; Modell-Rotation bei 503/
  429/500 (`FREE_MODELS` im Game Loop, `GEN_MODELS` in der Generierung).
- **JSON-Robustheit**: Free-Modelle „denken laut" — Generierung nutzt deshalb
  `ai_chat` + `extract_json` (schneidet Prosa um das JSON heraus) statt `ai_chat_json`.
- **Sandbox**: `{fs: "full", network_whitelist: ["opencode.ai"], exec: false,
  ai: true}` — volles FS nötig, damit Erweiterungen die Welt-Datei persistent
  aktualisieren können; Datei-Builtins sind dem LLM nie als Tools exponiert.
- **SQL-Fallstrick**: Pipes Mini-SQL-Parser kennt kein `''`-Escaping —
  `sql_escape` ersetzt Apostrophe daher durch `’`. Spalten nicht `key`/`value`
  nennen.
- **Parser-Fallstrick**: Inline-Listen-Literale `[...]` als Funktionsargumente
  zuerst an eine Variable binden.

## Testen

```bash
../bin/pipe -test     # alle Tests, keine KI-Calls, keine Netzwerk-Nutzung
```

Abgedeckt: Pack-/Design-/Karten-Validierung, BFS, Profil-Boni, Wunsch-Personas,
Expansions-Merge (inkl. Backlinks & Konnektivität), Raum-Cap, Handels-Mathe,
Objectives aller drei Arten, Save/Load beider Ebenen, Rückwärtskompatibilität.
