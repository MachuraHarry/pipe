# MQTT Module — Pure-Pipe MQTT 5.0 Client

**Status:** Available as an external package in the [`pipe-modules`](https://github.com/MachuraHarry/pipe-modules) repository. Install with `pipe -get mqtt`. A complete MQTT **5.0** client written entirely in Pipe — no C library, no `paho`, nothing but the standard library and three small TCP builtins.

## Features

- **MQTT 5.0** (protocol level 5) with the full property system (encode *and* decode of all 26 property identifiers)
- **QoS 0, 1 and 2** — the full `PUBREC`/`PUBREL`/`PUBCOMP` handshake, in both directions
- **TCP and TLS** — including `tls_insecure` for self-signed certificates
- **Will message**, **retain** flag, **clean start**, **keep-alive** (`PINGREQ`)
- Topic wildcards `+` and `#` (passed through to the broker)
- Username/password authentication

## Architecture

The module lives under `modules/mqtt/module.pipe` in the registry and is loaded via `pipe -get mqtt` or `import "mqtt"`. It is ~880 lines of pure Pipe.

Three TCP builtins make it possible (all in the standard binary):

- `tcp_connect_tls(host, port, servername?, insecure?)` — TLS transport
- `tcp_read_bytes(conn, n)` — read exactly `n` bytes as `bytes`
- `tcp_set_read_timeout(conn, ms)` — read deadline for the connect handshake

Everything else is built on the existing byte primitives: `to_bytes`, `int_to_bytes`, `bytes_to_int`, `bytes_append`, `bit_and`, `bit_rshift`, `at`, `slice`. The MQTT "Remaining Length" (a base-128 varint) is encoded with bit shifts instead of division: `n % 128` → `bit_and n 127`, `n / 128` → `bit_rshift n 7`.

### Concurrency model

A background goroutine (`go receive_loop`) is the **sole reader** of the socket. It frames packets, dispatches `PUBLISH` to the user callback, and routes `PUBACK`/`SUBACK`/`PUBREC`/`PUBCOMP` back to the waiting caller through buffered channels. A second goroutine (`go ping_loop`) sends `PINGREQ` at the keep-alive interval.

All shared state (the connection registry and per-connection state maps) is guarded by a single global mutex. Callbacks are invoked outside the lock, so a slow handler cannot stall the receive loop.

## API

All functions return `Ok(...)` / `Err(message)`.

| Function | Description |
|----------|-------------|
| `mqtt_connect(host, port, options?)` | Connect, returns `Ok(handle)` |
| `mqtt_disconnect(handle, graceful?)` | Disconnect; `graceful: false` triggers the will |
| `mqtt_publish(handle, topic, payload, options?)` | Publish (options: `qos`, `retain`, `properties`) |
| `mqtt_subscribe(handle, topic, qos?)` | Subscribe (topic or list), returns granted QoS |
| `mqtt_unsubscribe(handle, topic)` | Unsubscribe |
| `mqtt_ping(handle)` | Send `PINGREQ` |
| `mqtt_set_callback(handle, fn)` | Register message callback `fn(msg)` |
| `mqtt_connected(handle)` | Connection state |

The callback receives a map: `{topic, payload (bytes), qos, retain, pid, properties}`.

## Examples

```pipe
import "mqtt" as mqtt

h: unwrap (mqtt.mqtt_connect "broker.emqx.io" 1883 {use_tls: false})

mqtt.mqtt_set_callback h (fn msg
    print ">> " ++ (get msg "topic") ++ " = " ++ (from_bytes (get msg "payload")))

mqtt.mqtt_subscribe h "sensors/#" 1
mqtt.mqtt_publish h "sensors/temp" "23.5" {qos: 1, retain: true}
```

```pipe
-- TLS with a self-signed certificate and a will message
will: {topic: "status/dev", payload: "offline", qos: 1, retain: true}
h: unwrap (mqtt.mqtt_connect "192.168.1.10" 8883 {tls_insecure: true, will: will})
```

## Known Limits

- **No WebSocket transport** — TCP/TLS only (no browser clients).
- **No automatic reconnect** — if the broker drops the connection, the callback simply stops firing; `mqtt_connected` returns `false`. Reconnection is left to the caller.
- **No QoS-2 message store** — in-flight QoS-2 state is held in memory, not persisted across process restarts.

More examples: `modules/mqtt/example.pipe`.
