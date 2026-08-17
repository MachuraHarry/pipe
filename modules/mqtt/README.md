# MQTT

A complete **MQTT 5.0** client for [Pipe](https://github.com/MachuraHarry/pipe), written in pure Pipe.

## Features

- MQTT 5.0 (protocol level 5), with all 27 property identifiers
- QoS 0, 1 and 2 (full PUBREC/PUBREL/PUBCOMP handshake, both directions)
- Transport: TCP and TLS (incl. self-signed certificates via `tls_insecure`)
- Will message, retain flag, clean start, keep-alive (PINGREQ)
- Topic wildcards `+` and `#` (handled broker-side, pass filters to subscribe)
- Username/password authentication
- Session expiry and the common MQTT 5.0 properties
- Input validation (client ID length, keep-alive range, topic rules)
- Server-side `DISCONNECT` handling (reason code is recorded)
- CONNACK properties (`server_keep_alive`, `assigned_client_identifier`)
- Runtime warning when `tls_insecure` is enabled

## Installation

```bash
pipe -get mqtt
```

Or import directly:

```pipe
import "mqtt" as mqtt
```

## Quick start

```pipe
import "mqtt" as mqtt

h: unwrap (mqtt.mqtt_connect "broker.emqx.io" 1883 {use_tls: false})

mqtt.mqtt_set_callback h (fn msg
    print ">> " ++ (get msg "topic") ++ " = " ++ (from_bytes (get msg "payload")))

mqtt.mqtt_subscribe h "some/topic" 1
mqtt.mqtt_publish h "some/topic" "hello" {qos: 1}
```

## API

All functions return `Ok(...)` or `Err(message)`. Check with `is_err` / `unwrap`.

### `mqtt_connect(host, port, options?) -> Ok(handle) | Err`

Connects to a broker and returns a connection handle.

`options` (all optional):

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `client_id` | string | auto | Client identifier (1–23 bytes; empty only allowed with `clean_start: true`) |
| `username` | string | nil | Username (plain) |
| `password` | string | nil | Password (plain) |
| `keepalive` | number | 60 | Keep-alive interval in seconds (0–65535, 0 = disabled) |
| `clean_start` | bool | true | Start a clean session |
| `use_tls` | bool | true | Use TLS (defaults to port 8883 style) |
| `tls_insecure` | bool | false | Skip certificate verification (prints a MITM warning) |
| `timeout` | number | 5000 | Connect timeout in ms |
| `will` | map | nil | `{topic, payload, qos, retain}` |
| `properties` | map | {} | CONNECT properties |

```pipe
will: {topic: "status/dev", payload: "offline", qos: 1, retain: true}
h: unwrap (mqtt.mqtt_connect "broker" 8883 {username: "user", password: "secret", will: will})
```

> Note: Pipe map literals must be written on a single line. For complex options,
> build the options map in a variable first, then pass it.

### `mqtt_disconnect(handle, graceful?) -> Ok(nil) | Err`

Disconnects. `graceful` defaults to `true` (sends DISCONNECT). Pass `false` to close
abruptly, which triggers the will message.

### `mqtt_publish(handle, topic, payload, options?) -> Ok(nil) | Err`

`options`: `{qos: 0|1|2, retain: bool, properties: {}}`.

### `mqtt_subscribe(handle, topic, qos?) -> Ok([granted_qos, ...]) | Err`

`topic` is a string or a list of strings. Returns the granted QoS per topic.

### `mqtt_unsubscribe(handle, topic) -> Ok(nil) | Err`

### `mqtt_ping(handle) -> Ok(nil) | Err`

### `mqtt_set_callback(handle, fn) -> Ok(nil) | Err`

Registers the message callback. `fn` receives a map:

```
{topic, payload (bytes), qos, retain, pid, properties}
```

Use `from_bytes` to convert `payload` to a string.

### `mqtt_connected(handle) -> bool`

Returns `true` while the connection is open.

### QoS constants

`QOS0`, `QOS1`, `QOS2` (exported enum members).

## Validation

The module validates inputs and returns `Err(...)` early:

- **Connect** — `client_id` must be a string of 1–23 bytes (empty only with `clean_start: true`); `keepalive` must be a number in `0..65535`.
- **Publish** — topic must be non-empty, ≤ 65535 bytes, and contain no wildcards (`#`/`+`).
- **Subscribe/Unsubscribe** — topic (or every topic in a list) must be non-empty and ≤ 65535 bytes.

When the server closes the connection (server-side `DISCONNECT`), the client marks
the handle closed and records the reason code; `mqtt_connected` then returns `false`.

## Requirements

Requires Pipe with the `tcp_connect_tls`, `tcp_read_bytes` and
`tcp_set_read_timeout` builtins (Pipe v0.9.4+).

## License

MIT
