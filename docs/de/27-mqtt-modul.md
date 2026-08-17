# MQTT-Modul — Reiner-Pipe-MQTT-5.0-Client

**Status:** Verfügbar als externes Paket im [`pipe-modules`](https://github.com/MachuraHarry/pipe-modules)-Repository. Installation mit `pipe -get mqtt`. Ein vollständiger MQTT-**5.0**-Client, komplett in Pipe geschrieben — keine C-Bibliothek, kein `paho`, nur die Standardbibliothek und drei kleine TCP-Builtins.

## Funktionen

- **MQTT 5.0** (Protokoll-Level 5) mit dem vollen Property-System (Encode **und** Decode aller 27 Property-Identifier)
- **QoS 0, 1 und 2** — der komplette `PUBREC`/`PUBREL`/`PUBCOMP`-Handshake, in beide Richtungen
- **TCP und TLS** — inklusive `tls_insecure` für selbst-signierte Zertifikate
- **Will-Nachricht**, **Retain**-Flag, **Clean Start**, **Keep-alive** (`PINGREQ`)
- Topic-Wildcards `+` und `#` (an den Broker durchgereicht)
- Username/Passwort-Authentifizierung

## Architektur

Das Modul liegt unter `modules/mqtt/module.pipe` im Registry und wird per `pipe -get mqtt` oder `import "mqtt"` geladen. Es sind ~880 Zeilen reines Pipe.

Drei TCP-Builtins machen es möglich (alle in der Standard-Binary):

- `tcp_connect_tls(host, port, servername?, insecure?)` — TLS-Transport
- `tcp_read_bytes(conn, n)` — liest exakt `n` Bytes als `bytes`
- `tcp_set_read_timeout(conn, ms)` — Lese-Deadline für den Connect-Handshake

Alles andere baut auf den vorhandenen Byte-Primitiven auf: `to_bytes`, `int_to_bytes`, `bytes_to_int`, `bytes_append`, `bit_and`, `bit_rshift`, `at`, `slice`. Die MQTT-„Remaining Length" (ein Base-128-Varint) wird mit Bit-Shifts statt Division kodiert: `n % 128` → `bit_and n 127`, `n / 128` → `bit_rshift n 7`.

### Nebenläufigkeitsmodell

Eine Hintergrund-Goroutine (`go receive_loop`) ist der **einzige Leser** des Sockets. Sie rahmt Pakete, leitet `PUBLISH` an den User-Callback weiter und routet `PUBACK`/`SUBACK`/`PUBREC`/`PUBCOMP` über gepufferte Channels an den wartenden Aufrufer zurück. Eine zweite Goroutine (`go ping_loop`) sendet `PINGREQ` im Keep-alive-Intervall.

Der gesamte geteilte Zustand (das Verbindungs-Registry und die Per-Verbindungs-State-Maps) wird von einem einzigen globalen Mutex geschützt. Callbacks werden außerhalb des Locks aufgerufen, damit ein langsamer Handler den Receive-Loop nicht blockieren kann.

## API

Alle Funktionen geben `Ok(...)` / `Err(meldung)` zurück.

| Funktion | Beschreibung |
|----------|-------------|
| `mqtt_connect(host, port, optionen?)` | Verbinden, gibt `Ok(handle)` zurück |
| `mqtt_disconnect(handle, graceful?)` | Trennen; `graceful: false` löst das Will aus |
| `mqtt_publish(handle, topic, payload, optionen?)` | Publizieren (Optionen: `qos`, `retain`, `properties`) |
| `mqtt_subscribe(handle, topic, qos?)` | Abonnieren (Topic oder Liste), gibt gewährtes QoS zurück |
| `mqtt_unsubscribe(handle, topic)` | Abbestellen |
| `mqtt_ping(handle)` | `PINGREQ` senden |
| `mqtt_set_callback(handle, fn)` | Nachrichten-Callback `fn(msg)` registrieren |
| `mqtt_connected(handle)` | Verbindungsstatus |

Der Callback erhält eine Map: `{topic, payload (bytes), qos, retain, pid, properties}`.

## Beispiele

```pipe
import "mqtt" as mqtt

h: unwrap (mqtt.mqtt_connect "broker.emqx.io" 1883 {use_tls: false})

mqtt.mqtt_set_callback h (fn msg
    print ">> " ++ (get msg "topic") ++ " = " ++ (from_bytes (get msg "payload")))

mqtt.mqtt_subscribe h "sensoren/#" 1
mqtt.mqtt_publish h "sensoren/temperatur" "23.5" {qos: 1, retain: true}
```

```pipe
-- TLS mit selbst-signiertem Zertifikat und Will-Nachricht
will: {topic: "status/geraet", payload: "offline", qos: 1, retain: true}
h: unwrap (mqtt.mqtt_connect "192.168.1.10" 8883 {tls_insecure: true, will: will})
```

## Bekannte Grenzen

- **Kein WebSocket-Transport** — nur TCP/TLS (keine Browser-Clients).
- **Kein automatischer Reconnect** — bricht der Broker die Verbindung ab, feuert der Callback einfach nicht mehr; `mqtt_connected` liefert `false`. Die Wiederverbindung bleibt dem Aufrufer überlassen.
- **Kein QoS-2-Nachrichtenspeicher** — In-Flight-QoS-2-Zustand wird im Speicher gehalten, nicht über Prozess-Neustarts hinweg persistiert.

Weitere Beispiele: `modules/mqtt/example.pipe`.
