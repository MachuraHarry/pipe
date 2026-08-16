[lang:en]# 🌍 MicroPipe — Pipe on a Microcontroller (Preview)[/lang]
[lang:de]# 🌍 MicroPipe — Pipe auf dem Mikrocontroller (Preview)[/lang]

[lang:en]
**Pipe's pitch is one language for every layer — WASM in the browser, an ~8 MB binary on servers and Raspberry Pi. MicroPipe closes the last gap: a faithful re-implementation of the core that runs on a $5 ESP32.**

MicroPipe is a dependency-free Python re-implementation of the Pipe language core, verified **against the real `pipe` v0.9.4 binary** with a side-by-side comparison harness. It targets CPython (for tests and development) and **MicroPython** (for the ESP32 and other microcontrollers). No pip dependencies — the stdlib is all it uses.

What hangs on a microcontroller, you drive with the same language as your pipelines: read a sensor, call an MCP server over the LAN, toggle a GPIO. Same syntax, same pipelines, same `mcp_use_sse`.

### What works today

- **The faithful core**: precedence, vertical pipelines, closures, recursion, `if`/`while`/`for`, lists/maps/slicing, `>>` parsed — every behaviour checked against pipe v0.9.4.
- **~59 curated builtins** — the ones that make sense on a small device: `map`, `filter`, `reduce`, `sum`, `unique`, `parse_json`, `sleep`, file I/O, plus the MCP bridge.
- **MCP client over HTTP and stdio**: `mcp_use_sse "http://192.168.178.78:8000/mcp"` exposes every server tool as `mcp0_<tool>` inside your Pipe program.
- **MicroPython compatible**: identifier checks that survive the missing `str.isalnum`, a `urequests`-or-raw-socket HTTP fallback, and an import layout that works as a flat package on the device.

```pipe
out: mcp_use_sse "http://192.168.178.78:8000/mcp"
print out

reading: parse_json (mcp0_sensor_read 1)
print "sensor:" (reading["value"]) (reading["unit"])

state: if (reading["value"]) > 23.0
    "on"
else
    "off"
print "led:" (mcp0_led_set state)
```

Verified on real hardware — an ESP32-D0WD-V3 running MicroPython v1.28.0:

```
wifi connected: 192.168.178.81
MicroPipe ESP32 demo started
[pipe] connected 5 tools from http://192.168.178.78:8000/mcp (prefix: mcp0_)
[pipe] temp: 27.5
[pipe] led: on
```

### What it is not — yet

Honest scope, because a preview should say so:

- **No AI builtins.** `summarize`, `ask`, `embed` need a server that doesn't fit on a device. The intended bridge: run Pipe itself as the MCP server and call *it*.
- **No sandbox, no bytecode VM, no `pipe -build`.** It's a tree-walker.
- **`>>` is parsed but executed sequentially.** No real on-device parallelism — yet.
- **MCP client only** — the server side stays in the Go binary.

Think of MicroPipe as the **embedded edition**: the same language for all the things that aren't servers.

### Try it

```bash
pip install git+https://github.com/MachuraHarry/micropipe
```

Examples in the repo: a Fibonacci, a `map`/`filter`/`reduce` pipeline, and the MCP sensor demo above (`examples/*.mpipe`). The README covers flashing an ESP32, WiFi, and the MCP server setup.

It's a preview, it's MIT, and it's the natural next target for `>>` parallelism and the embedded MCP client. [Star it on GitHub](https://github.com/MachuraHarry/micropipe) or open an issue.
[/lang]

[lang:de]
**Pipes Versprechen ist eine Sprache für jede Ebene — WASM im Browser, eine ~8-MB-Binary auf Servern und Raspberry Pi. MicroPipe schließt die letzte Lücke: eine treue Neuimplementierung des Kerns, die auf einem 5-Dollar-ESP32 läuft.**

MicroPipe ist eine abhängigkeitsfreie Python-Neuimplementierung des Pipe-Sprachkerns, **gegen die echte `pipe`-v0.9.4-Binary verifiziert** (Side-by-Side-Vergleichs-Harness). Zielplattformen: CPython (für Tests und Entwicklung) und **MicroPython** (für den ESP32 und andere Mikrocontroller). Keine pip-Abhängigkeiten — nur die Standardbibliothek.

Was an einem Mikrocontroller hängt, steuerst du mit derselben Sprache wie deine Pipelines: Sensor lesen, MCP-Server im LAN ansprechen, GPIO schalten. Gleiche Syntax, gleiche Pipelines, gleiches `mcp_use_sse`.

### Was heute funktioniert

- **Der treue Kern**: Präzedenz, vertikale Pipelines, Closures, Rekursion, `if`/`while`/`for`, Listen/Maps/Slicing, `>>` geparst — jedes Verhalten gegen pipe v0.9.4 geprüft.
- **~59 kuratierte Builtins** — die, die auf einem kleinen Gerät Sinn ergeben: `map`, `filter`, `reduce`, `sum`, `unique`, `parse_json`, `sleep`, Datei-I/O, dazu die MCP-Brücke.
- **MCP-Client über HTTP und stdio**: `mcp_use_sse "http://192.168.178.78:8000/mcp"` macht jedes Server-Tool als `mcp0_<tool>` in deinem Pipe-Programm verfügbar.
- **MicroPython-kompatibel**: Identifier-Checks ohne das fehlende `str.isalnum`, ein `urequests`-oder-Raw-Socket-HTTP-Fallback und ein Import-Layout, das auch als flaches Paket auf dem Gerät funktioniert.

```pipe
out: mcp_use_sse "http://192.168.178.78:8000/mcp"
print out

reading: parse_json (mcp0_sensor_read 1)
print "sensor:" (reading["value"]) (reading["unit"])

state: if (reading["value"]) > 23.0
    "on"
else
    "off"
print "led:" (mcp0_led_set state)
```

Verifiziert auf echter Hardware — einem ESP32-D0WD-V3 mit MicroPython v1.28.0:

```
wifi connected: 192.168.178.81
MicroPipe ESP32 demo started
[pipe] connected 5 tools from http://192.168.178.78:8000/mcp (prefix: mcp0_)
[pipe] temp: 27.5
[pipe] led: on
```

### Was es (noch) nicht ist

Ehrlicher Scope — ein Preview soll das sagen:

- **Keine KI-Builtins.** `summarize`, `ask`, `embed` brauchen einen Server, der nicht auf ein Gerät passt. Die vorgesehene Brücke: Pipe selbst als MCP-Server laufen lassen und *den* aufrufen.
- **Kein Sandbox, keine Bytecode-VM, kein `pipe -build`.** Es ist ein Tree-Walker.
- **`>>` wird geparst, aber sequenziell ausgeführt.** Noch keine echte Parallelität on-device.
- **Nur MCP-Client** — die Server-Seite bleibt in der Go-Binary.

Denk also an MicroPipe als **Embedded-Edition**: dieselbe Sprache für alles, was kein Server ist.

### Ausprobieren

```bash
pip install git+https://github.com/MachuraHarry/micropipe
```

Beispiele im Repo: ein Fibonacci, eine `map`/`filter`/`reduce`-Pipeline und die MCP-Sensor-Demo von oben (`examples/*.mpipe`). Die README behandelt ESP32-Flashing, WLAN und das MCP-Server-Setup.

Es ist ein Preview, es ist MIT, und es ist das nächste natürliche Ziel für `>>`-Parallelismus und den eingebetteten MCP-Client. [Auf GitHub markieren](https://github.com/MachuraHarry/micropipe) oder ein Issue eröffnen.
[/lang]
