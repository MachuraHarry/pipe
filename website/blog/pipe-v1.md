[lang:en]# Pipe v1.0.0 — From Prototype to Production[/lang]
[lang:de]# Pipe v1.0.0 — Vom Prototypen zur Produktion[/lang]

[lang:en]
**Today Pipe reaches v1.0.0. After months of hardening — sandbox audits, bytecode optimization, concurrency primitives, and 232 builtins — Pipe is production-ready. This is the release that consolidates everything into a single, stable, ~8 MB binary.**
[/lang]

[lang:de]
**Heute erreicht Pipe v1.0.0. Nach monatelanger Härterung — Sandbox-Audits, Bytecode-Optimierung, Concurrency-Primitiven und 232 Builtins — ist Pipe production-ready. Dieses Release konsolidiert alles in einer einzigen, stabilen, ~8 MB Binary.**
[/lang]

---

[lang:en]## What's New in v1.0[/lang]
[lang:de]## Was ist neu in v1.0[/lang]

[lang:en]
### Guard Clauses

Match expressions now support guard clauses — conditions that refine patterns:

```pipe
fn classify severity
    match severity
        | s if s > 9 -> "critical"
        | s if s > 5 -> "warning"
        | _ -> "info"
```

### Concurrency Primitives

Channels, mutex, and counting semaphore — true concurrency without leaving the language:

```pipe
ch: chan 3
go { send ch "hello" }
go { send ch "world" }
print (recv ch) ++ " " ++ (recv ch)
```

### Bytecode VM Improvements

- **Constant folding** — literal expressions evaluated at compile time
- **Alias import namespaces** — cleaner module imports
- **Bytecode cache** — faster startup for repeated runs

### MQTT 5.0 Module

A complete MQTT client in pure Pipe — QoS 0/1/2, TLS, will, retain, keep-alive, and the full MQTT 5.0 property system. No C library, no `paho`, nothing but the standard library.

```pipe
import "mqtt" as mqtt

h: unwrap (mqtt.mqtt_connect "broker.emqx.io" 1883 {use_tls: false})
mqtt.mqtt_subscribe h "sensors/#" 1
mqtt.mqtt_publish h "sensors/temp" "23.5" {qos: 1, retain: true}
```

### docs-pipe — RAG for Documentation

A documentation-native RAG module with heading-aware chunking and web dashboard:

```pipe
import "docs-pipe" as dp

idx: dp.doc_index "https://docs.example.com"
dp.doc_add idx "README.md"
dp.doc_search idx "installation" 3 > each print
```

### Test Framework Improvements

- Setup/teardown hooks
- `assert_near` and `assert_contains`
- VM test blocks with `OpTestResult`/`OpTestAbortIfError`

### Hardened Sandbox

Six rounds of security audits, deterministic env masking, central egress gate for AI provider HTTP, and input validation across all modules.
[/lang]

[lang:de]
### Guard Clauses

Match-Ausdrücke unterstützen jetzt Guard Clauses — Bedingungen, die Patterns verfeinern:

```pipe
fn classify severity
    match severity
        | s if s > 9 -> "critical"
        | s if s > 5 -> "warning"
        | _ -> "info"
```

### Concurrency-Primitiven

Channels, Mutex und Counting Semaphore — echte Concurrency, die Sprache nicht verlässt:

```pipe
ch: chan 3
go { send ch "hello" }
go { send ch "world" }
print (recv ch) ++ " " ++ (recv ch)
```

### Bytecode-VM-Verbesserungen

- **Constant Folding** — Literal-Ausdrücke werden zur Compile-Zeit ausgewertet
- **Alias-Import-Namespaces** — sauberere Modul-Imports
- **Bytecode-Cache** — schnellere Startzeiten bei wiederholten Ausführungen

### MQTT 5.0 Modul

Ein vollständiger MQTT-Client in reinem Pipe — QoS 0/1/2, TLS, Will, Retain, Keep-Alive und das vollständige MQTT 5.0 Property-System. Keine C-Bibliothek, kein `paho`, nichts als die Standardbibliothek.

```pipe
import "mqtt" as mqtt

h: unwrap (mqtt.mqtt_connect "broker.emqx.io" 1883 {use_tls: false})
mqtt.mqtt_subscribe h "sensors/#" 1
mqtt.mqtt_publish h "sensors/temp" "23.5" {qos: 1, retain: true}
```

### docs-pipe — RAG für Dokumentation

Ein dokumentations-natives RAG-Modul mit heading-aware Chunking und Web-Dashboard:

```pipe
import "docs-pipe" as dp

idx: dp.doc_index "https://docs.example.com"
dp.doc_add idx "README.md"
dp.doc_search idx "installation" 3 > each print
```

### Test-Framework-Verbesserungen

- Setup/Teardown-Hooks
- `assert_near` und `assert_contains`
- VM-Test-Blöcke mit `OpTestResult`/`OpTestAbortIfError`

### Gehärtete Sandbox

Sechs Runden Security-Audits, deterministische Env-Maskierung, zentraler Egress-Gate für AI-Provider-HTTP und Input-Validierung in allen Modulen.
[/lang]

---

[lang:en]## The Numbers[/lang]
[lang:de]## Die Zahlen[/lang]

| | v0.9.4 | v1.0.0 |
|---|---|---|
| **Builtins** | 226 | 232 |
| **Modules** | 22 | 23 |
| **Tests** | 640 | 643 |
| **Binary size** | ~8 MB | ~8 MB |
| **Dependencies** | 0 | 0 |

---

[lang:en]## Install[/lang]
[lang:de]## Installation[/lang]

[lang:en]
```sh
curl -fsSL https://pipe-lang.com/install.sh | bash
```
[/lang]

[lang:de]
```sh
curl -fsSL https://pipe-lang.com/install.sh | bash
```
[/lang]

[lang:en]## What's Next[/lang]
[lang:de]## Was kommt als Nächstes[/lang]

[lang:en]
- **Type annotations** — optional type checking at compile time
- **REPL improvements** — tab completion, multi-line editing
- **Additional builtins** — regex capture groups, binary data, crypto functions
- **Plugin system** — extend Pipe with Go modules
[/lang]

[lang:de]
- **Type Annotations** — optionale Typprüfung zur Compile-Zeit
- **REPL-Verbesserungen** — Tab-Vervollständigung, Multi-Line-Editing
- **Weitere Builtins** — Regex-Capture-Gruppen, Binärdaten, Crypto-Funktionen
- **Plugin-System** — Pipe mit Go-Modulen erweitern
[/lang]

[lang:en]
---

**Pipe v1.0.0 is the language's first stable release.** From IoT with MQTT to AI agents with MCP, from sandboxed execution to parallel pipelines — Pipe is ready for production.

[Star Pipe on GitHub](https://github.com/MachuraHarry/pipe) · [Read the Docs](https://pipe-lang.com/docs.html) · [Try the Playground](https://pipe-lang.com/playground.html) · [Join Discord](https://discord.gg/kdjce8hnYw)
[/lang]

[lang:de]
---

**Pipe v1.0.0 ist das erste stabile Release der Sprache.** Von IoT mit MQTT bis hin zu AI-Agents mit MCP, von sandboxed Execution bis zu parallelen Pipelines — Pipe ist bereit für die Produktion.

[Pipe auf GitHub starren](https://github.com/MachuraHarry/pipe) · [Dokumentation lesen](https://pipe-lang.com/docs.html) · [Playground ausprobieren](https://pipe-lang.com/playground.html) · [Discord beitreten](https://discord.gg/kdjce8hnYw)
[/lang]
