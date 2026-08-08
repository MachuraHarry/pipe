# 24. Discord Module

> **STATUS: IN DEVELOPMENT — NOT yet published to the pipe-modules registry.**
> Mock-tested against all functions. Live-testing requires a Discord webhook URL
> or bot token (both free — no payment, no app review).

The **Discord module** (`scripts/discord.pipe`) is a pure-Pipe client for the **Discord API** (v10). It supports two authentication models:

- **Webhook** — simplest: POST JSON to a webhook URL, no auth header needed. Ideal for notifications and alerts.
- **Bot Token** — full bot access: send messages, read channel history, edit/delete messages, and more via REST polling.

It is built entirely on the standard Pipe builtins: `http_request`, `to_json`, and `parse_json`.

**Limitation:** The Discord Gateway (WebSocket) is not supported, so real-time push events are unavailable. Message reads use REST polling (`GET /channels/{id}/messages`).

---

## 24.1 Setup

```pipe
import "discord.pipe" as d
```

Place `discord.pipe` next to your script, or expose it via `PIPE_PATH`:

```bash
export PIPE_PATH="$HOME/CODE/pipe/scripts"
```

The module uses one endpoint variable:

| Variable | Default |
|----------|---------|
| `DISCORD_API_BASE` | `https://discord.com/api/v10` |

To obtain a **webhook URL**: Discord → Server → Channel settings → Integrations → Webhooks → Create. Copy the URL (format: `https://discord.com/api/webhooks/ID/TOKEN`).

To obtain a **bot token**: [Discord Developer Portal](https://discord.com/developers/applications) → Create Application → Bot → Token.

Both are completely free — no payment, no app review required for personal use.

---

## 24.2 Webhook Functions

### `d_webhook_send` webhook_url payload → Ok {} | Err

Sends a message via webhook. `payload` is a map with Discord message fields:

```pipe
d.d_webhook_send WEBHOOK_URL {content: "Hello!", username: "MyBot"}
```

### `d_webhook_text` webhook_url content → Ok {} | Err

Convenience wrapper for plain text:

```pipe
d.d_webhook_text WEBHOOK_URL "Deploy finished successfully"
```

### `d_webhook_embed` webhook_url embed → Ok {} | Err

Sends a rich embed via webhook:

```pipe
d.d_webhook_embed WEBHOOK_URL {
    title: "Build #42",
    description: "All tests passed",
    color: 65280
}
```

---

## 24.3 Bot Token Functions

### `d_send` token channel_id payload → Ok message | Err

Sends a message as a bot. `payload` is a map with Discord `create-message` fields:

```pipe
d.d_send TOKEN CHANNEL_ID {content: "Hello from my bot!"}
```

### `d_send_text` token channel_id content → Ok message | Err

Plain-text convenience wrapper:

```pipe
d.d_send_text TOKEN CHANNEL_ID "Reminder: meeting in 10 minutes"
```

### `d_send_embed` token channel_id embed → Ok message | Err

Sends a rich embed as bot:

```pipe
d.d_send_embed TOKEN CHANNEL_ID {
    title: "Server Status",
    description: "All systems operational",
    color: 255
}
```

### `d_get_messages` token channel_id [limit] → Ok [message] | Err

Reads recent messages (polling). `limit` is optional (1–100, default 20):

```pipe
msgs: d.d_get_messages TOKEN CHANNEL_ID 10
if is_ok msgs
    for m in (unwrap msgs)
        print (get (get m "author") "username") ++ ": " ++ (get m "content")
```

### `d_get_channel` token channel_id → Ok channel | Err

Gets channel metadata (name, type, etc.):

```pipe
ch: d.d_get_channel TOKEN CHANNEL_ID
if is_ok ch
    print (get (unwrap ch) "name")
```

### `d_edit_message` token channel_id message_id payload → Ok message | Err

Edits a bot's own message:

```pipe
d.d_edit_message TOKEN CHANNEL_ID "123456789" {content: "Updated text"}
```

### `d_delete_message` token channel_id message_id → Ok {} | Err

Deletes a bot's own message:

```pipe
d.d_delete_message TOKEN CHANNEL_ID "123456789"
```

### `d_get_guilds` token → Ok [guild] | Err

Lists all servers the bot is a member of:

```pipe
guilds: d.d_get_guilds TOKEN
if is_ok guilds
    for g in (unwrap guilds)
        print (get g "name")
```

---

## 24.4 Error Handling

All functions return Pipe's `Ok`/`Err` results. Use `is_ok` and `is_err` to branch:

```pipe
res: d.d_send_text TOKEN CHANNEL_ID "Test"
if is_err res
    print "Failed: " ++ (to_str res)
else
    print "Message sent, id: " ++ (get (unwrap res) "id")
```

Discord-specific errors include the HTTP status code and the `message`/`code` from the JSON error body, e.g., `Discord-Fehler (HTTP 404): Not found (code 0)`.

---

## 24.5 Rate Limits

- **Webhooks:** ~5 messages per 5 seconds per channel (30/minute). 429 responses include `retry_after` (seconds).
- **Bot token:** bucket-based per route (read/write) — typically 5 req/s per server for sends, unlimited reads (exceeds at ~50+ splits).
- Pipe's `http_request` has a 30-second timeout per request; backoff on 429 must be handled by the caller.

---

## 24.6 Typical Use Cases

**Notification webhook (from CI/CD):**

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

**Bot reading recent messages and responding:**

```pipe
import "discord.pipe" as d

token: env "DISCORD_BOT_TOKEN"
ch: env "DISCORD_CHANNEL_ID"

msgs: d.d_get_messages token ch 5
if is_ok msgs
    for m in (unwrap msgs)
        d.d_send_text token ch ("Echo: " ++ (get m "content"))
```
