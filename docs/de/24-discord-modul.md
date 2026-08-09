# 24. Discord-Modul

> **STATUS: IN ENTWICKLUNG — NOCH NICHT im pipe-modules-Registry veröffentlicht.**
> Mock-getestet gegen alle Funktionen. Live-Tests erfordern eine Discord-Webhook-URL
> oder einen Bot-Token (beides kostenlos — keine Zahlung, kein App-Review).

Das **Discord-Modul** (`scripts/discord.pipe`) ist ein reiner-Pipe-Client für die **Discord API** (v10). Es unterstützt zwei Authentifizierungsmodelle:

- **Webhook** — am einfachsten: JSON per POST an eine Webhook-URL, kein Auth-Header nötig. Ideal für Benachrichtigungen und Alerts.
- **Bot-Token** — voller Bot-Zugriff: Nachrichten senden, Channel-Verlauf lesen, Nachrichten bearbeiten/löschen und mehr via REST-Polling.

Es basiert vollständig auf den Standard-Pipe-Builtins: `http_request`, `to_json` und `parse_json`.

**Einschränkung:** Das Discord Gateway (WebSocket) wird nicht unterstützt, daher sind Echtzeit-Push-Events nicht verfügbar. Nachrichten-Lesen erfolgt über REST-Polling (`GET /channels/{id}/messages`).

---

## 24.1 Einrichtung

```pipe
import "discord.pipe" as d
```

Lege `discord.pipe` neben dein Skript oder stelle es über `PIPE_PATH` bereit:

```bash
export PIPE_PATH="$HOME/CODE/pipe/scripts"
```

Das Modul verwendet eine Endpoint-Variable:

| Variable | Standard |
|----------|----------|
| `DISCORD_API_BASE` | `https://discord.com/api/v10` |

**Webhook-URL erhalten:** Discord → Server → Kanaleinstellungen → Integrationen → Webhooks → Erstellen. URL kopieren (Format: `https://discord.com/api/webhooks/ID/TOKEN`).

**Bot-Token erhalten:** [Discord Developer Portal](https://discord.com/developers/applications) → Create Application → Bot → Token.

Beides ist komplett kostenlos — keine Zahlung, kein App-Review für den persönlichen Gebrauch.

---

## 24.2 Webhook-Funktionen

### `d_webhook_send` webhook_url payload → Ok {} | Err

Sendet eine Nachricht per Webhook. `payload` ist eine Map mit Discord-Nachrichtenfeldern:

```pipe
d.d_webhook_send WEBHOOK_URL {content: "Hallo!", username: "MeinBot"}
```

### `d_webhook_text` webhook_url content → Ok {} | Err

Komfort-Wrapper für reinen Text:

```pipe
d.d_webhook_text WEBHOOK_URL "Deploy erfolgreich abgeschlossen"
```

### `d_webhook_embed` webhook_url embed → Ok {} | Err

Sendet ein Rich-Embed per Webhook:

```pipe
d.d_webhook_embed WEBHOOK_URL {
    title: "Build #42",
    description: "Alle Tests bestanden",
    color: 65280
}
```

---

## 24.3 Bot-Token-Funktionen

### `d_send` token channel_id payload → Ok message | Err

Sendet eine Nachricht als Bot. `payload` ist eine Map mit den Discord-`create-message`-Feldern:

```pipe
d.d_send TOKEN CHANNEL_ID {content: "Hallo von meinem Bot!"}
```

### `d_send_text` token channel_id content → Ok message | Err

Komfort-Wrapper für reinen Text:

```pipe
d.d_send_text TOKEN CHANNEL_ID "Erinnerung: Meeting in 10 Minuten"
```

### `d_send_embed` token channel_id embed → Ok message | Err

Sendet ein Rich-Embed als Bot:

```pipe
d.d_send_embed TOKEN CHANNEL_ID {
    title: "Server-Status",
    description: "Alle Systeme betriebsbereit",
    color: 255
}
```

### `d_get_messages` token channel_id [limit] → Ok [message] | Err

Liest aktuelle Nachrichten (Polling). `limit` ist optional (1–100, Standard 20):

```pipe
msgs: d.d_get_messages TOKEN CHANNEL_ID 10
if is_ok msgs
    for m in (unwrap msgs)
        print (get (get m "author") "username") ++ ": " ++ (get m "content")
```

### `d_get_channel` token channel_id → Ok channel | Err

Holt Channel-Metadaten (Name, Typ, usw.):

```pipe
ch: d.d_get_channel TOKEN CHANNEL_ID
if is_ok ch
    print (get (unwrap ch) "name")
```

### `d_edit_message` token channel_id message_id payload → Ok message | Err

Bearbeitet eine eigene Nachricht des Bots:

```pipe
d.d_edit_message TOKEN CHANNEL_ID "123456789" {content: "Aktualisierter Text"}
```

### `d_delete_message` token channel_id message_id → Ok {} | Err

Löscht eine eigene Nachricht des Bots:

```pipe
d.d_delete_message TOKEN CHANNEL_ID "123456789"
```

### `d_get_guilds` token → Ok [guild] | Err

Listet alle Server auf, in denen der Bot Mitglied ist:

```pipe
guilds: d.d_get_guilds TOKEN
if is_ok guilds
    for g in (unwrap guilds)
        print (get g "name")
```

---

## 24.4 Fehlerbehandlung

Alle Funktionen liefern Pipe-`Ok`/`Err`-Results. Verzweige mit `is_ok` und `is_err`:

```pipe
res: d.d_send_text TOKEN CHANNEL_ID "Test"
if is_err res
    print "Fehlgeschlagen: " ++ (to_str res)
else
    print "Nachricht gesendet, id: " ++ (get (unwrap res) "id")
```

Discord-spezifische Fehler enthalten den HTTP-Statuscode sowie `message`/`code` aus dem JSON-Fehlerbody, z. B. `Discord-Fehler (HTTP 404): Not found (code 0)`.

---

## 24.5 Rate Limits

- **Webhooks:** ~5 Nachrichten pro 5 Sekunden pro Channel (30/Minute). 429-Antworten enthalten `retry_after` (Sekunden).
- **Bot-Token:** Bucket-basiert pro Route (read/write) — typischerweise 5 req/s pro Server für Sends, unbegrenzt für Reads (bricht bei ~50+ Splits).
- Pipe's `http_request` hat ein 30-Sekunden-Timeout pro Request; Backoff bei 429 muss der Aufrufer selbst behandeln.

---

## 24.6 Typische Anwendungsfälle

**Benachrichtigungs-Webhook (aus CI/CD):**

```pipe
import "discord.pipe" as d

webhook: env "DISCORD_WEBHOOK_URL"
status: env "BUILD_STATUS"
d.d_webhook_embed webhook {
    title: "CI Build",
    description: "Status: " ++ status,
    color: 65280
}
```

**Bot liest aktuelle Nachrichten und antwortet:**

```pipe
import "discord.pipe" as d

token: env "DISCORD_BOT_TOKEN"
ch: env "DISCORD_CHANNEL_ID"

msgs: d.d_get_messages token ch 5
if is_ok msgs
    for m in (unwrap msgs)
        d.d_send_text token ch ("Echo: " ++ (get m "content"))
```
