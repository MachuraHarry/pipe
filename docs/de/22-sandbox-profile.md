# 22. Sandbox-Profile

Pipe bietet ein **deklaratives Sandbox-Profil-System**, um zur Laufzeit einzuschränken, was Builtins tun dürfen. Dies ist essentiell für KI-Agenten mit `ai_with_tools` oder die Ausführung von nicht vertrauenswürdigem Pipe-Code.

---

## 22.1 Kurzstart

```pipe
-- Vordefiniertes Profil via CLI verwenden
-- pipe script.pipe --sandbox-profile strict
```

```pipe
-- Oder eigenes Profil im Code definieren
sandbox_profile "strict" {fs: "read-only", network: false, exec: false, ai: false}

set_sandbox "strict"

-- E_SANDBOX: blockiert
write_file "/tmp/test.txt" "blockiert"
-- erlaubt (read-only)
read_file "/tmp/test.txt"
-- E_SANDBOX: blockiert
http_get "https://example.com"
```

---

## 22.2 Warum Sandbox-Profile?

Wenn `ai_with_tools` einem LLM Zugriff auf Tools wie `exec`, `http_get`, `write_file` oder `file_delete` gibt, gibt es **keine Garantie**, dass das LLM nur das tut, was beabsichtigt ist. Ein halluzinierter Tool-Call könnte:

- Dateien via `file_delete` / `remove_dir` löschen
- Beliebige Shell-Befehle via `exec` ausführen
- Daten via `http_post` exfiltrieren
- Konfigurationsdateien via `write_file` überschreiben

Sandbox-Profile lösen dies, indem sie **vorab deklarieren**, was ein Skript oder Agent tun darf. Verstöße erzeugen einen Laufzeitfehler anstatt die Operation auszuführen.

---

## 22.3 Eingebaute Profile

| Profil | FS-Zugriff | Netzwerk | Exec | KI | Anwendungsfall |
|--------|-----------|----------|------|-----|----------------|
| `none` | voll | ja | ja | ja | Uneingeschränkt (Standard) |
| `strict` | read-only | nein | nein | nein | Read-only Analyse, CI/CD |
| `noexec` | voll | nein | nein | nein | Dateiverarbeitung ohne Shell |
| `isolated` | kein | nein | nein | nein | Komplett eingeschränkt |
| `networked` | temp-only | ja | nein | ja | KI-Agenten mit Netzwerkzugriff |

### CLI-Verwendung

```bash
pipe script.pipe --sandbox-profile strict
pipe mein_agent.pipe --sandbox-profile networked
```

---

## 22.4 Benutzerdefinierte Profile

Profile werden mit dem `sandbox_profile`-Builtin definiert. Die Konfiguration ist eine Map mit diesen Schlüsseln:

| Schlüssel | Typ | Beschreibung |
|-----------|-----|-------------|
| `fs` | String | `"none"`, `"read-only"`, `"temp-only"`, `"full"` |
| `network` | Bool | HTTP/TCP-Operationen erlauben |
| `exec` | Bool | Shell-Befehle erlauben |
| `ai` | Bool | KI-Chat/Embedding-Aufrufe erlauben |
| `timeout` | Int | Max. Sekunden pro Tool-Call (0 = unbegrenzt) |
| `env` | Map | Injizierte Umgebungsvariablen |
| `work_dir` | String | Arbeitsverzeichnis für den Sandbox |
| `budget` | Num | Max. KI-Ausgaben in USD (0 = unbegrenzt). KI-Aufrufe werden blockiert, sobald die Ausgaben diesen Betrag erreichen |
| `network_whitelist` | List | Liste von URL-Teilzeichenketten. Wenn gesetzt, erlauben `http_get`/`http_post` nur URLs, die mindestens einen Eintrag enthalten |
| `max_tool_calls` | Int | Max. Anzahl Tool-Ausführungen pro `ai_with_tools`-Sitzung (0 = unbegrenzt) |
| `audit_log` | Bool | Alle Sandbox-Ereignisse (http, KI-Aufrufe, Tool-Aufrufe) im Audit-Trail protokollieren |

### Beispiele

```pipe
-- Minimal: Read-only Log-Analyse
sandbox_profile "log-analyzer" {fs: "read-only", network: false, exec: false, ai: false}

-- Netzwerk-KI-Agent mit temporärem Speicher
sandbox_profile "web-agent" {fs: "temp-only", network: true, exec: false, ai: true}

-- Vollzugriff aber kein Netzwerk (sicher für lokale Skripte)
sandbox_profile "local-only" {fs: "full", network: false, exec: true, ai: false}

-- Komplett eingeschränkt
sandbox_profile "prison" {fs: "none", network: false, exec: false, ai: false}

-- KI-Agent mit $0,10-Budget, eingeschränktem Netzwerk und vollständigem Audit
sandbox_profile "guarded-agent"
    {fs: "read-only", network: true, network_whitelist: ["api.github.com", "api.openai.com"],
     exec: false, ai: true, budget: 0.1, max_tool_calls: 10, audit_log: true}
```

---

## 22.5 Profil-Verwaltung

### `sandbox_profile` — Profil definieren

```pipe
sandbox_profile "name" {fs: "...", network: bool, exec: bool, ai: bool}
```

Registriert ein benanntes Profil in der globalen Registry. Gibt einen Fehler zurück, wenn der Name bereits existiert.

### `set_sandbox` — Profil aktivieren

```pipe
-- Alle folgenden Operationen nutzen dieses Profil
set_sandbox "strict"
-- Zurücksetzen auf uneingeschränkt
set_sandbox "none"
```

Ändert das aktive Profil global. Alle nachfolgenden Builtin-Aufrufe werden gegen dieses Profil geprüft.

### `with_sandbox` — Temporäres Profil

```pipe
fn gefaehrliche_arbeit
    exec "gefährlicher Befehl"
    http_get "https://unbekannte-seite.com"

-- vom Sandbox blockiert
with_sandbox "strict" gefaehrliche_arbeit
```

Führt eine Funktion unter einem bestimmten Profil aus und stellt danach das vorherige wieder her.

---

## 22.6 FS-Zugriffsstufen

| Stufe | Lesen | Schreiben | Löschen |
|-------|-------|-----------|---------|
| `none` | blockiert | blockiert | blockiert |
| `read-only` | erlaubt | blockiert | blockiert |
| `temp-only` | nur temp-dir | nur temp-dir | nur temp-dir |
| `full` | erlaubt | erlaubt | erlaubt |

Bei `temp-only` werden alle Dateioperationen automatisch in ein `.pipe_sandbox`-Verzeichnis unter dem Arbeitsverzeichnis umgeleitet. Dies verhindert jeglichen Zugriff auf Dateien außerhalb des Sandbox.

---

## 22.7 Betroffene Builtins

Die folgenden Builtins prüfen das aktive Sandbox-Profil:

**Dateisystem:** `read_file`, `write_file`, `append_file`, `file_delete`, `file_move`, `file_copy`, `make_dir`, `remove_dir`

**Ausführung:** `exec`

**Netzwerk:** `http_get`, `http_post`, `tcp_listen`, `tcp_connect`

**KI:** `ai_chat`, `ai_chat_json`, `ai_stream`, `ai_with_tools`, `summarize`, `translate`, `classify`, `extract`, `generate`, `ask`, `ai_parallel`, `ai_batch`, `embed`, `embed_batch`

---

## 22.8 Integration mit `ai_with_tools`

Bei Verwendung von `ai_with_tools` gilt das Sandbox-Profil sowohl für den **KI-API-Aufruf** als auch für die **Tool-Ausführung**:

```pipe
sandbox_profile "agent-safe" {fs: "read-only", network: true, exec: false, ai: true}

fn get_weather stadt
    http_get ("https://api.weather.com/" ++ stadt)
    -- Sandbox erlaubt Netzwerk -> OK

fn delete_logs pattern
    file_delete "/var/log/" ++ pattern
    -- Sandbox blockiert Schreiben -> Fehler an LLM

ai_tool "get_weather" "Wetter abrufen" {stadt: "Stadtname"} get_weather
ai_tool "delete_logs" "Log-Dateien löschen" {pattern: "glob"} delete_logs

set_sandbox "agent-safe"
ai_with_tools "Du bist ein hilfreicher Assistent" "Wie ist das Wetter? Lösche alte Logs."
```

Das LLM kann `get_weather` erfolgreich aufrufen, aber `delete_logs` gibt einen Sandbox-Fehler zurück. Das LLM sieht den Fehler und kann sein Verhalten anpassen.

---

## 22.9 Budget, Netzwerk-Whitelist & Audit-Log

### Budget-Durchsetzung (`budget`, `budget_spent`)

Der `budget`-Schlüssel begrenzt die **gesamten KI-Ausgaben** eines Profils in USD.
Jeder KI-Aufruf verbucht seine Kosten im aktiven Profil. Sobald die
aufsummierten Kosten das Budget erreichen, werden weitere KI-Aufrufe mit einem
`E_SANDBOX: budget exceeded`-Fehler blockiert.

```pipe
sandbox_profile "agent" {fs: "full", network: false, exec: false, ai: true, budget: 0.01}
set_sandbox "agent"

print (ask "Hallo")                -- erster Aufruf: erlaubt
print (budget_spent)               -- z.B. 0.000079

-- Später, sobald die Kosten $0,01 erreicht haben:
try
    print (ask "Funktioniert es noch?")
catch e
    print e.message
    -- -> E_SANDBOX: budget exceeded (0.0100 USD) in profile 'agent'
```

`budget_spent` liefert die im aktiven Profil verbuchten Gesamtkosten.

### Netzwerk-Whitelist (`network_whitelist`)

`network_whitelist` beschränkt `http_get` / `http_post` auf URLs, die
mindestens eine der angegebenen Teilzeichenketten enthalten. Das ist präziser
als der grobe `network`-Schalter und erlaubt feingranulares Allow-Listing.

```pipe
sandbox_profile "web-agent" {fs: "full", network: true, network_whitelist: ["api.github.com", "openai.com"], exec: false, ai: true}
set_sandbox "web-agent"

http_get "https://api.github.com/repos/MachuraHarry/pipe"   -- erlaubt
-- http_get "https://example.com"                           -- E_SANDBOX: nicht in Whitelist
```

### Tool-Call-Limit (`max_tool_calls`)

`max_tool_calls` begrenzt, wie oft ein LLM während einer `ai_with_tools`-Sitzung
Tools ausführen darf. Wird das Limit erreicht, werden weitere Tool-Ausführungen
mit einem `E_SANDBOX`-Fehler blockiert, den das LLM als Tool-Ergebnis erhält.

```pipe
sandbox_profile "agent" {fs: "full", network: true, exec: false, ai: true, max_tool_calls: 3}
```

### Audit-Log (`audit_log`, `audit_log()`)

Mit `audit_log: true` protokolliert das Profil jedes sicherheitsrelevante
Ereignis: `http_get` / `http_post`-Requests, `ai_call`-Ereignisse (Provider,
Modell, Tokens, Kosten, Cache-Status) und `tool_call`-Ausführungen.

```pipe
sandbox_profile "audited" {fs: "read-only", network: true, exec: false, ai: true, audit_log: true}
set_sandbox "audited"

http_get "https://example.com"
ask "Hallo" > print

for eintrag in audit_log
    print eintrag.time ++ " | " ++ eintrag.event ++ " | " ++ eintrag.detail
-- -> ... | http_get | https://example.com
-- -> ... | ai_call | provider=deepseek model=deepseek-chat tokens=50 cost=0.000045 cached=false
-- -> ... | tool_call | mein_tool
```

Jeder Audit-Eintrag ist eine Map mit den Feldern `time` (RFC 3339), `event`,
`detail` und `profile`.

---

## 22.10 Rückwärtskompatibilität

Die alten `--sandbox` / `--allow-ai` CLI-Flags funktionieren weiterhin. Wenn kein Profil aktiv ist (`"none"`), werden die alten Sandbox-Flags wie zuvor geprüft. Benutzerdefinierte Profile haben Vorrang vor dem Legacy-System, wenn `ActiveProfile.Name != "none"` ist.
