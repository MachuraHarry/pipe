[lang:en]# 📡 MQTT 5.0 in Pure Pipe — a Full Client in ~880 Lines[/lang]
[lang:de]# 📡 MQTT 5.0 in reinem Pipe — ein vollständiger Client in ~880 Zeilen[/lang]

[lang:en]
**MQTT — the de-facto standard for IoT messaging — is now a single Pipe module: QoS 0/1/2, TLS, will, retain, keep-alive, and the full MQTT 5.0 property system. No C library, no `paho`, nothing but the standard library and three small TCP builtins.**

Every IoT project eventually needs MQTT. And every MQTT client in a scripting language means dragging in a native library: `paho-mqtt` on Python, `MQTT.js` on Node, `eclipse/paho` on Go. Then a build step, a dependency tree, and a binary that's no longer a single file.

The Pipe answer is the same as always: **make it a module.** A pure-Pipe file that runs on the existing ~8 MB binary. Here's the whole thing:

```bash
pipe -get mqtt
```

```pipe
import "mqtt" as mqtt

h: unwrap (mqtt.mqtt_connect "broker.emqx.io" 1883 {use_tls: false})

mqtt.mqtt_set_callback h (fn msg
    print ">> " ++ (get msg "topic") ++ " = " ++ (from_bytes (get msg "payload")))

mqtt.mqtt_subscribe h "sensors/#" 1
mqtt.mqtt_publish h "sensors/temp" "23.5" {qos: 1, retain: true}
```

That's a live subscriber and publisher — tested against public brokers over TCP and TLS, including the full QoS-2 handshake in both directions.

## Three builtins, nothing else

MQTT is a binary protocol, so it needs three things Pipe didn't have: TLS dial, exact-byte reads, and a read deadline. Three additions, ~90 lines of Go total:

- `tcp_connect_tls(host, port, servername?, insecure?)` — TLS transport, with `insecure` for self-signed IoT certificates
- `tcp_read_bytes(conn, n)` — read exactly `n` bytes as `bytes` (the framing primitive every binary protocol wants)
- `tcp_set_read_timeout(conn, ms)` — read deadline for the connect handshake

Everything else is pure Pipe on top of existing byte primitives. MQTT's "Remaining Length" is a base-128 varint; instead of integer division (`n / 128`), it's written with bit ops that Pipe already has: `bit_and n 127` and `bit_rshift n 7`. Topic strings are length-prefixed UTF-8, built from `int_to_bytes` + `bytes_append`. The whole encoder/decoder is ~250 lines of `bytes` shuffling.

## The concurrency model

MQTT is a long-lived, bidirectional connection, so the interesting part is the runtime, not the packet parsing. Two goroutines do the work:

- **`receive_loop`** is the *sole reader* of the socket. It frames packets, hands `PUBLISH` to your callback, and routes `PUBACK`/`SUBACK`/`PUBREC`/`PUBCOMP` back to the waiting caller through buffered channels.
- **`ping_loop`** sends `PINGREQ` at the keep-alive interval so the broker doesn't drop you.

All shared state — the connection registry and per-connection maps — sits behind a single global mutex, and callbacks are invoked *outside* the lock so a slow handler can't stall the receive loop. A synchronous `mqtt_subscribe` registers a channel, writes `SUBSCRIBE`, then polls for `SUBACK` with a timeout. QoS 2 does the same dance twice: `PUBLISH` → `PUBREC` → `PUBREL` → `PUBCOMP`.

The two bugs we caught while building it are the ones that always bite concurrent code: a Go-map data race (two connections mutating the same registry map), and a Pipe scoping subtlety where `NEXT_ID: NEXT_ID + 1` reassigns a *local* instead of the module counter — so every connection got id 1 and clobbered the previous one. Both fixed; the module is green against `broker.emqx.io`, `broker.hivemq.com`, and a local test broker.

## Will, retain, and the full 5.0 property set

Because it's MQTT **5.0** (not 3.1.1), the module also speaks the modern dialect: all 27 property identifiers are encoded *and* decoded — `session_expiry_interval`, `user_property`, `content_type`, `correlation_data`, and the rest. Will messages work (drop the connection with `graceful: false` and the broker publishes your will). Retained messages come back on fresh subscriptions.

```pipe
-- TLS + will message + self-signed cert
will: {topic: "status/dev", payload: "offline", qos: 1, retain: true}
h: unwrap (mqtt.mqtt_connect "192.168.1.10" 8883 {tls_insecure: true, will: will})
```

## Honest limits

- **No WebSocket transport** — TCP/TLS only, so no browser clients.
- **No automatic reconnect** — if the broker drops you, `mqtt_connected` flips to `false` and reconnecting is on you.
- **No QoS-2 message store** — in-flight state lives in memory, not across restarts.

For the typical "publish telemetry, subscribe to commands" IoT workload, none of that matters. If you need reconnect or persistence, the hooks are there — the module is plain Pipe you can read and extend.

## Try it

- Module: [`pipe-modules/mqtt`](https://github.com/MachuraHarry/pipe-modules/tree/master/modules/mqtt) — ~880 lines of pure Pipe.
- Docs: [MQTT Module](https://github.com/MachuraHarry/pipe/blob/master/docs/en/27-mqtt-module.md)

```bash
pipe -get mqtt
# or clone and run the example
git clone https://github.com/MachuraHarry/pipe
cd pipe && ./bin/pipe modules/mqtt/example.pipe
```
[/lang]

[lang:de]
**MQTT — der De-facto-Standard für IoT-Messaging — ist jetzt ein einziges Pipe-Modul: QoS 0/1/2, TLS, Will, Retain, Keep-alive und das volle MQTT-5.0-Property-System. Keine C-Bibliothek, kein `paho`, nur die Standardbibliothek und drei kleine TCP-Builtins.**

Jedes IoT-Projekt braucht irgendwann MQTT. Und jeder MQTT-Client in einer Skriptsprache bedeutet, eine native Bibliothek mitzuschleppen: `paho-mqtt` unter Python, `MQTT.js` unter Node, `eclipse/paho` unter Go. Dazu ein Build-Schritt, ein Abhängigkeitsbaum und eine Binary, die nicht mehr eine einzige Datei ist.

Pipes Antwort ist wie immer dieselbe: **mach ein Modul daraus.** Eine reine-Pipe-Datei, die auf der bestehenden ~8-MB-Binary läuft. Das Ganze in einem Rutsch:

```bash
pipe -get mqtt
```

```pipe
import "mqtt" as mqtt

h: unwrap (mqtt.mqtt_connect "broker.emqx.io" 1883 {use_tls: false})

mqtt.mqtt_set_callback h (fn msg
    print ">> " ++ (get msg "topic") ++ " = " ++ (from_bytes (get msg "payload")))

mqtt.mqtt_subscribe h "sensors/#" 1
mqtt.mqtt_publish h "sensors/temp" "23.5" {qos: 1, retain: true}
```

Das ist ein lebender Subscriber und Publisher — getestet gegen öffentliche Broker über TCP und TLS, inklusive des vollen QoS-2-Handshakes in beide Richtungen.

## Drei Builtins, sonst nichts

MQTT ist ein Binärprotokoll, also braucht es drei Dinge, die Pipe nicht hatte: TLS-Dial, Byte-genaue Reads und eine Read-Deadline. Drei Ergänzungen, insgesamt ~90 Zeilen Go:

- `tcp_connect_tls(host, port, servername?, insecure?)` — TLS-Transport, mit `insecure` für selbst-signierte IoT-Zertifikate
- `tcp_read_bytes(conn, n)` — liest exakt `n` Bytes als `bytes` (die Framing-Primitive, die jedes Binärprotokoll will)
- `tcp_set_read_timeout(conn, ms)` — Lese-Deadline für den Connect-Handshake

Alles andere ist reines Pipe auf den vorhandenen Byte-Primitiven. MQTTs „Remaining Length" ist ein Base-128-Varint; statt Integer-Division (`n / 128`) wird es mit Bit-Operationen geschrieben, die Pipe bereits hat: `bit_and n 127` und `bit_rshift n 7`. Topic-Strings sind längen-präfixiertes UTF-8, gebaut aus `int_to_bytes` + `bytes_append`. Der ganze Encoder/Decoder sind ~250 Zeilen `bytes`-Geschubse.

## Das Nebenläufigkeitsmodell

MQTT ist eine langlebige, bidirektionale Verbindung, daher ist der interessante Teil die Laufzeit, nicht das Paket-Parsing. Zwei Goroutinen erledigen die Arbeit:

- **`receive_loop`** ist der *einzige Leser* des Sockets. Er rahmt Pakete, übergibt `PUBLISH` an deinen Callback und routet `PUBACK`/`SUBACK`/`PUBREC`/`PUBCOMP` über gepufferte Channels an den wartenden Aufrufer zurück.
- **`ping_loop`** sendet `PINGREQ` im Keep-alive-Intervall, damit der Broker dich nicht abwirft.

Der gesamte geteilte Zustand — das Verbindungs-Registry und die Per-Verbindungs-Maps — liegt hinter einem einzigen globalen Mutex, und Callbacks werden *außerhalb* des Locks aufgerufen, damit ein langsamer Handler den Receive-Loop nicht blockieren kann. Ein synchrones `mqtt_subscribe` registriert einen Channel, schreibt `SUBSCRIBE` und pollt dann mit Timeout auf `SUBACK`. QoS 2 tanzt denselben Tanz zweimal: `PUBLISH` → `PUBREC` → `PUBREL` → `PUBCOMP`.

Die zwei Bugs, die wir beim Bauen einfingen, sind die, die nebenläufigen Code immer beißen: eine Go-Map-Data-Race (zwei Verbindungen mutieren dieselbe Registry-Map) und eine Pipe-Scoping-Subtilität, bei der `NEXT_ID: NEXT_ID + 1` eine *lokale* Variable neu zuweist statt den Modul-Zähler — sodass jede Verbindung die ID 1 bekam und die vorherige überschrieb. Beides behoben; das Modul ist grün gegen `broker.emqx.io`, `broker.hivemq.com` und einen lokalen Test-Broker.

## Will, Retain und das volle 5.0-Property-Set

Weil es MQTT **5.0** ist (nicht 3.1.1), spricht das Modul auch den modernen Dialekt: alle 27 Property-Identifier werden kodiert *und* dekodiert — `session_expiry_interval`, `user_property`, `content_type`, `correlation_data` und der Rest. Will-Nachrichten funktionieren (Verbindung mit `graceful: false` fallen lassen, und der Broker publiziert dein Will). Retained-Nachrichten kommen bei frischen Subscriptions zurück.

```pipe
-- TLS + Will-Nachricht + selbst-signiertes Zertifikat
will: {topic: "status/geraet", payload: "offline", qos: 1, retain: true}
h: unwrap (mqtt.mqtt_connect "192.168.1.10" 8883 {tls_insecure: true, will: will})
```

## Ehrliche Grenzen

- **Kein WebSocket-Transport** — nur TCP/TLS, also keine Browser-Clients.
- **Kein automatischer Reconnect** — wirft der Broker dich ab, springt `mqtt_connected` auf `false`, und die Wiederverbindung liegt bei dir.
- **Kein QoS-2-Nachrichtenspeicher** — In-Flight-Zustand lebt im Speicher, nicht über Neustarts hinweg.

Für den typischen IoT-Workload „Telemetrie publizieren, Befehle abonnieren" spielt davon nichts eine Rolle. Brauchst du Reconnect oder Persistenz, sind die Hooks da — das Modul ist lesbares, erweiterbares Pipe.

## Ausprobieren

- Modul: [`pipe-modules/mqtt`](https://github.com/MachuraHarry/pipe-modules/tree/master/modules/mqtt) — ~880 Zeilen reines Pipe.
- Doku: [MQTT-Modul](https://github.com/MachuraHarry/pipe/blob/master/docs/de/27-mqtt-modul.md)

```bash
pipe -get mqtt
# oder klonen und das Beispiel ausführen
git clone https://github.com/MachuraHarry/pipe
cd pipe && ./bin/pipe modules/mqtt/example.pipe
```
[/lang]
