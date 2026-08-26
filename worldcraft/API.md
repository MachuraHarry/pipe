# Worldcraft Server — API-Spezifikation (v1.1)

HTTP/JSON-Backend für Worldcraft. Jede Spiel-Session läuft in einem **isolierten
OS-Prozess** mit eigener In-Memory-Welt — mehrere Spieler/Session stören sich nie.
Langsame KI-Operationen laufen asynchron über Jobs; idle Sessions beenden sich
selbst (mit Persistenz) und lassen sich per `resume` wiederbeleben.

Server starten (aus `worldcraft/`):

```bash
../bin/pipe server.pipe                    # Port 8080
WORLDCRAFT_TOKEN=admin123 ../bin/pipe server.pipe   # mit Admin-Gate
```

## Auth-Modell (zwei Ebenen)

| Token | Wofür | Woher |
|---|---|---|
| **Admin-Token** (`WORLDCRAFT_TOKEN`, optional) | Nur `POST /sessions` und `GET /worlds` | Server-Betreiber — steckt **nicht** in der App |
| **Session-Secret** (~48 Hex-Zeichen) | Alle Aufrufe zu genau einer Session (`Authorization: Bearer <secret>`) | Wird bei Session-Erstellung **einmalig** mitgeliefert; App muss es persistieren |

Falsches/fehlendes Secret → `401 unauthorized`. Session-IDs sind nicht erratbar
geschützt, weil ohne Secret jeder Call abgelehnt wird.

---

## Konventionen

- Alle Antworten sind JSON. Fehler immer im Format:
  `{"error": {"code": "...", "message": "..."}}`
- Alle Responses enthalten CORS-Header (`*`) — Browser-Clients funktionieren direkt.
- **Async-Pattern**: Züge und Weltoperationen antworten mit `202 {job_id, status:"running"}`.
  Ergebnis abholen per Polling auf dem Job-Pfad (Fastpath-Kommandos sind meist
  nach einer Poll sofort `done`). Ein Session hat max. **einen** aktiven Job;
  zweiter Versuch → `409 {error:{code:"busy"}}`.
- **Idle-Timeout**: Ohne Requests beendet ein Worker sich selbst nach
  standardmäßig 10 Minuten (pro Session konfigurierbar via `idle_minutes`),
  nachdem er den Spielstand persistiert hat. Der Zustand geht nicht verloren:
  `/resume` belebt die Session jederzeit wieder.

---

## Endpunkte

### `GET /api/v1/worlds`
Verfügbare Welten aus `worlds/`.
```json
{"worlds": ["beispielwelt", "nebeltal", "wachstum"]}
```

### `POST /api/v1/sessions`
Startet einen Worker-Prozess und lädt/generiert die Welt.

Variante A — existierende Welt (sofort spielbar):
```json
{"world": "nebeltal"}
```
Variante B — neue Welt generieren (dauert ~2 Min., siehe Boot-Job unten):
```json
{"world": "meinewelt", "briefing": "Schwebende Inseln...", "profil": {"stil": "kaempfer", "schwierigkeit": "normal"}, "idle_minutes": 10}
```

Antwort `201`:
```json
{
  "session_id": "s1787730632_1",
  "world": "beispielwelt",
  "secret": "3c092da094c2...",
  "ready": true,
  "state": { ...siehe State-Objekt... },
  "options": ["geh nach norden", "nimm fackel", "..."]
}
```
`ready: false` → Welt generiert noch; solange `GET /sessions/{id}/state` pollen,
bis das State-Objekt gefüllt ist. **Das `secret` muss die App speichern** —
es ist der einzige Schlüssel zu dieser Session.

### `POST /api/v1/sessions/{id}/resume`
Belebt eine Session wieder — etwa nachdem die App gekillt wurde, das Handy
eingeschlafen ist oder der Server neu gestartet wurde. Bearer = Session-Secret.

- Läuft der Worker noch → `200` mit aktuellem State.
- Ist er weg (idle-tod, Crash, Supervisor-Restart) → Worker wird aus der
  persistierten Meta-Datei (`port`, `world`) neu gestartet, der Spielstand
  (`saves/session_<id>.json`) automatisch wiederhergestellt → `200` mit dem
  exakten Stand vor dem Tod (Position, HP, Gold, Inventar).

### `GET /api/v1/sessions/{id}/state`
Voller Zustand + Intro-Narration:
```json
{
  "narration": "== tpl_eingang == ...",
  "dialog": null,
  "state": {
    "world": "Die Musterwelt",
    "room": "tpl_eingang",
    "room_description": "== tpl_eingang == ...",
    "exits": ["norden"],
    "items": ["fackel"],
    "npcs": [{"id": "weiser_aldric", "name": "Weiser Aldric", "role": "weiser"}],
    "monsters": [{"id": "kammerling", "name": "Kammerling", "hp": 5, "max_hp": 5}],
    "hp": 10, "gold": 20, "inventory": [],
    "quests": [{"id": "q_ungewuerm", "name": "Das Ungewürm", "state": "aktiv"}],
    "won": false
  },
  "options": ["geh nach norden", "nimm fackel", "inventar", "quests"]
}
```

### `POST /api/v1/sessions/{id}/command` `{input: string}`
Ein Zug — Freitext oder Fastpath-Befehl. → `202 {job_id}`

Ergebnis nach Polling:
```json
{
  "job_id": "j2",
  "status": "done",
  "result": {
    "type": "narrator",
    "narration": "Der Weise hebt langsam den Kopf...",
    "dialog": {
      "speaker": "Weiser Aldric",
      "npc": "weiser_aldric",
      "player_line": "rede mit weiser_aldric ueber das erbe",
      "reply": "„Das Erbe... ist wie ein Brunnen, der nie versiegt...“",
      "choices": ["Was willst du wissen?", "Biete ihm etwas.", "Weiterziehen, ohne zu vertrauen."]
    },
    "state": { ... },
    "options": [ ... ]
  }
}
```

| Feld | Bedeutung |
|---|---|
| `type` | `fast` (ohne KI, <10 ms) · `narrator` (KI-Erzähler) · `expand` · `new_adventure` |
| `dialog` | Nur gesetzt, wenn der Zug ein NPC-Gespräch war. `choices` = 3 KI-Vorschläge für nächste Befehle — ideal als Buttons in der App |
| `options` | Deterministische Aktionsvorschläge aus dem Zustand (Exits, Items, NPCs, Handel, Kampf) |

Polling-Pfad: `GET /api/v1/sessions/{id}/jobs/{job_id}` → `{status:"running"|"done", result?, error?}`

### `POST /api/v1/sessions/{id}/expand` `{wunsch: string}`
Welt wachsen lassen. → `202 {job_id}`, Result-Narration fasst zusammen:
*„Die Welt wächst: 2 neue Räume …"*. Bei Fehlschlag (Free-Tier) ist die Welt
unverändert — einfach wiederholen.

### `POST /api/v1/sessions/{id}/new-adventure`
Neue Siegesbedingung nach einem Sieg (`won: true`). → `202 {job_id}`.

### `DELETE /api/v1/sessions/{id}`
Beendet den Worker **und löscht Spielstand + Metadaten** (bewusstes Beenden =
Neuanfang möglich). Nur der Idle-Timeout/Resume erhält den Stand.

---

## State-Objekt (Referenz)

| Feld | Typ | Inhalt |
|---|---|---|
| `world` | str | Anzeigename der Welt |
| `room`, `room_description` | str | aktueller Raum (+ atmosphärischer Text) |
| `exits` | list | mögliche Richtungen |
| `items` | list | Items am Boden |
| `npcs` | list of obj | `{id, name, role}` |
| `monsters` | list of obj | `{id, name, hp, max_hp}` |
| `hp`, `gold` | num | Vitalwerte |
| `inventory` | list | getragene Items (IDs) |
| `quests` | list of obj | `{id, name, state}` (`aktiv`/`erledigt`) |
| `won` | bool | Hauptziel erfüllt |

## Fehlercodes

| Code | HTTP | Bedeutung |
|---|---|---|
| `unauthorized` | 401 | Token falsch/fehlt |
| `missing_world` | 400 | Weder `world` noch `briefing` |
| `busy` | 409 | Session hat schon einen laufenden Job |
| `booting` | 503 | Welt generiert noch (state pollen) |
| `boot_failed` / `worker_start_failed` / `worker_unreachable` | 500/502 | Prozessproblem — Session löschen und neu anlegen |
| `unknown_session` / `unknown_job` / `not_found` | 404 | —

## Architektur-Notizen

- **Supervisor** (`server.pipe`): Routing, Registry, startet pro Session einen
  Worker via `proc_start` auf Ports 8100+. Ein Catch-All verhindert, dass
  Handler-Fehler den Supervisor beenden.
- **Worker** (`worldd.pipe`): eigene sqlite-Welt, eigener HTTP-Port, Job-Queue
  (ein aktiver Job), Persistenz via `saves/session_<id>.json` bei jedem Zug und
  beim Shutdown. Neustart derselben Session-ID stellt den Stand wieder her.
- Die Spiellogik ist identisch zum Terminal (`turns.pipe`) — beide Frontends
  spielen exakt dasselbe Spiel.
