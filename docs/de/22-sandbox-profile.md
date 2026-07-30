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

write_file "/tmp/test.txt" "blockiert"   -- E_SANDBOX: blockiert
read_file "/tmp/test.txt"                -- erlaubt (read-only)
http_get "https://example.com"           -- E_SANDBOX: blockiert
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
set_sandbox "strict"     -- Alle folgenden Operationen nutzen dieses Profil
set_sandbox "none"       -- Zurücksetzen auf uneingeschränkt
```

Ändert das aktive Profil global. Alle nachfolgenden Builtin-Aufrufe werden gegen dieses Profil geprüft.

### `with_sandbox` — Temporäres Profil

```pipe
fn gefaehrliche_arbeit
    exec "gefährlicher Befehl"
    http_get "https://unbekannte-seite.com"

with_sandbox "strict" gefaehrliche_arbeit   -- vom Sandbox blockiert
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
    -- Sandbox erlaubt Netzwerk → OK

fn delete_logs pattern
    file_delete "/var/log/" ++ pattern
    -- Sandbox blockiert Schreiben → Fehler an LLM

ai_tool "get_weather" "Wetter abrufen" {stadt: "Stadtname"} get_weather
ai_tool "delete_logs" "Log-Dateien löschen" {pattern: "glob"} delete_logs

set_sandbox "agent-safe"
ai_with_tools "Du bist ein hilfreicher Assistent" "Wie ist das Wetter? Lösche alte Logs."
```

Das LLM kann `get_weather` erfolgreich aufrufen, aber `delete_logs` gibt einen Sandbox-Fehler zurück. Das LLM sieht den Fehler und kann sein Verhalten anpassen.

---

## 22.9 Rückwärtskompatibilität

Die alten `--sandbox` / `--allow-ai` CLI-Flags funktionieren weiterhin. Wenn kein Profil aktiv ist (`"none"`), werden die alten Sandbox-Flags wie zuvor geprüft. Benutzerdefinierte Profile haben Vorrang vor dem Legacy-System, wenn `ActiveProfile.Name != "none"` ist.
