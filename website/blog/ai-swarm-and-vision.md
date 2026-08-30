[lang:en]# 🐝👁️ ai_swarm and ai_vision — Handoff Multi-Agent Swarms and Image Understanding in Pure Pipe[/lang]
[lang:de]# 🐝👁️ ai_swarm und ai_vision — Handoff-Multi-Agent-Swarms und Bildverständnis in reinem Pipe[/lang]

[lang:en]
**Two new AI builtins: `ai_swarm` wires named agents together with the handoff pattern OpenAI's original "Swarm" library popularized — one agent active at a time, full shared conversation, transfer via a reserved tool call. `ai_vision` answers questions about images (URL, local file, or raw bytes) against DeepSeek's new vision model. Plus the sneaky `.pipec` bytecode cache bug we found — and fixed for good — while building them.**

Pipe already had two separate AI building blocks that never quite met: `agent`/`agent_ask` gives you a named, stateful conversation, but no tools. `ai_tool`/`ai_with_tools` gives you tool-calling, but no persistent identity. Neither lets one agent hand a conversation to another. That's the gap `ai_swarm` closes.

## The gap: agents that can't talk to each other

A swarm is a set of named agents, each with its own system prompt and tool set, that can transfer control to one another mid-conversation — while the **full message history** carries forward, so nothing gets lost at the handoff.

```pipe
ai_provider "deepseek"

fn get_invoice customer
    "Invoice #4471, 49.90 EUR, due 2026-09-15."

ai_tool "get_invoice" "Look up a customer's latest invoice" {customer: "Customer name"} get_invoice

swarm_agent "triage" {system: "Route billing questions to 'billing'. Handle anything else yourself.", handoff: ["billing"]}
swarm_agent "billing" {system: "You handle billing questions using get_invoice.", tools: ["get_invoice"], handoff: ["triage"]}

"What's on my latest invoice?"
    > ai_swarm "triage"
    > print
```

Run against a real DeepSeek key, that returns the billing agent's answer after a clean handoff — `ai_swarm_trace` gives you the same result plus `{content, path, rounds}` so you can see exactly who handled the request: `["triage", "billing"]`.

## How handoff actually works

There's no magic router. When an agent declares `handoff` targets, `ChatSwarm` (the new Go-side loop in `pkg/ai/swarm.go`) synthesizes one extra tool for that round — a reserved `__handoff__(to: enum[...])` the model can call like any other tool. When it does, the loop doesn't run your executor: it swaps the system message for the target agent's prompt and continues the *same* message array into the next round. The conversation history — including the transferring agent's own turns — is never touched, so the new agent has full context without a summary or a second prompt.

`ChatSwarm` mirrors `ChatWithTools`'s round loop almost line for line, on purpose: it's the same proven request/response shape, just with one more branch for the reserved tool name. Tool execution during a swarm run goes through the exact same `executeTool`/`toolRegistry` machinery `ai_with_tools` already uses — a swarm agent's `tools` field is just a list of names already registered with `ai_tool`.

## ai_vision: content blocks, not a new Message type

DeepSeek [shipped a vision model](https://api-docs.deepseek.com/guides/vision/) — `deepseek-v4-flash-vision-exp` — using the same OpenAI-compatible `/v1/chat/completions` shape Pipe already speaks everywhere else. The only difference: a user message's `content` is an array of `{type: "text", ...}` / `{type: "image_url", ...}` blocks instead of a plain string.

The tempting move is widening `ai.Message.Content` from `string` to something richer. We didn't do that. It's used as a plain string in six separate provider structs (`pkg/ai/providers.go`) *and* in the response-cache key logic — a capability that (like `ai_with_tools`) only works with OpenAI-compatible providers to begin with isn't worth a wide, repetitive change to a typed path five other builtins depend on staying string-shaped.

Instead, `ai.VisionChat` is a small, self-contained function that builds its own raw JSON body directly — the same pattern `ChatWithTools`'s internals already use for tool-call messages. Zero changes to `Message`, `ChatRequest`, or any of the six provider implementations.

```pipe
ai_provider "deepseek" {model: "deepseek-v4-flash-vision-exp"}

"https://raw.githubusercontent.com/github/explore/main/topics/go/go.png"
    > ai_vision "What does this logo depict?"
    > print
-- -> "This logo depicts the Go programming language (also commonly
--     known as Golang)..."
```

`image` accepts three forms: an `http(s)` URL passed straight through (the provider's servers fetch it, not Pipe), a local file path read through the same sandbox read-gate as `read_file`, or raw `bytes`. Local files and raw bytes get content-sniffed with Go's stdlib `http.DetectContentType` (no hand-written magic-byte table, no third-party dependency) and base64-encoded into a `data:` URL. Both the URL path and the local-file path are live-verified against a real DeepSeek key — same correct answer either way.

## Sandbox gating: nothing new

Both builtins reuse gates that already existed rather than inventing a third one. `ai_swarm`/`ai_vision` get the same two-branch check as `ai_chat`: `profile.CanAI()` under a registered profile, the CLI `--sandbox` flag's `Sandbox.AllowAI` otherwise. The real backstop is `gateEgress(EgressChat, ...)` inside `ChatSwarm`/`VisionChat` themselves — the same central sandbox gate every `Chat`/`Stream`/`Embed` call has gone through since [round 5 of our sandbox audits](sandbox-audit-2.html). Reading a local image path goes through the exact same fs-read gate `read_file` uses, unaffected by `--sandbox` (which only restricts writes). Nothing here needed a new audit round — everything routes through gates we'd already hardened.

## The bug we found: builtins move, bytecode caches don't know

Adding `ai_swarm`'s three new builtins in the middle of Pipe's builtin table — not at the end — quietly broke an unrelated example. `xor_cipher.pipe` started hanging instead of running, with the VM printing `encrypt: key must be 16, 24, or 32 bytes` even though the script never calls `encrypt`.

The cause: the compiler bakes each builtin's *position* in the table directly into the bytecode as an integer index (`BuiltinScope`). Insert a builtin anywhere but the end, and every later builtin's index shifts. A `.pipec` disk cache compiled against the old table still looked "valid" — same source hash, same `CacheVersion` byte — and fed the VM bytecode that resolved `OpGetBuiltin` to the *wrong function*. A leftover local cache from before our change called `encrypt` where the script meant something else entirely, and looped instead of erroring cleanly.

The comment already sitting next to `CacheVersion` even predicted this exact failure mode — it just depends on a human remembering to bump a constant for a change that has nothing to do with bytecode *format*. So instead of bumping it once, we made the class of bug impossible: the cache's dependency hash now includes a fingerprint of the ordered builtin-name table itself, so *any* future insertion, removal, or reorder self-invalidates every `.pipec` on disk automatically.

```go
// pkg/cache/cache.go — depsHash now also covers the builtin table
for _, b := range object.Builtins {
    h.Write([]byte(b.Name))
    h.Write([]byte{0})
}
```

A new regression test (`TestLoadOrCompileInvalidatesOnBuiltinTableChange`) inserts a builtin mid-table and asserts the cache misses. Builtin position is now provably irrelevant to cache correctness — which is also why `ai_vision`'s registration didn't need any special placement thought at all.

## Honest limits

- **OpenAI-compatible providers only** — `openai`, `deepseek`, `ollama`, `openrouter`, `opencode`. `anthropic` uses a different tool-call and image-block shape and isn't supported by either builtin, the same inherited constraint `ai_with_tools` already has.
- **Single image per `ai_vision` call** — DeepSeek's API allows up to 600. A `list` of images is a straightforward extension of the same request shape if we need it later; not built now.
- **No provider/model validation** — same hands-off approach as the rest of `ai_provider`/`ai_model`. Point `ai_vision` at a non-vision model and you get the provider's own error, not a Pipe-side check.
- **No shared state across parallel swarm runs** — each `ai_swarm` call owns its own message history; nothing is shared between concurrent `>>` swarm calls by design.

## Try it

```bash
DEEPSEEK_API_KEY="sk-..." pipe examples/swarm_demo.pipe
DEEPSEEK_API_KEY="sk-..." pipe examples/vision_demo.pipe
```

- Docs: [AI Builtins §19.14–19.15](https://github.com/MachuraHarry/pipe/blob/master/docs/en/19-ai-builtins.md)
- Source: [`pkg/ai/swarm.go`](https://github.com/MachuraHarry/pipe/blob/master/pkg/ai/swarm.go), [`pkg/ai/vision.go`](https://github.com/MachuraHarry/pipe/blob/master/pkg/ai/vision.go), [`pkg/object/builtins_swarm.go`](https://github.com/MachuraHarry/pipe/blob/master/pkg/object/builtins_swarm.go), [`pkg/object/builtins_vision.go`](https://github.com/MachuraHarry/pipe/blob/master/pkg/object/builtins_vision.go)
[/lang]

[lang:de]
**Zwei neue KI-Builtins: `ai_swarm` verdrahtet benannte Agenten mit dem Handoff-Pattern, das OpenAIs ursprüngliche „Swarm"-Bibliothek populär gemacht hat — ein aktiver Agent zur Zeit, komplett geteilter Gesprächsverlauf, Übergabe per reserviertem Tool-Call. `ai_vision` beantwortet Fragen zu Bildern (URL, lokale Datei oder rohe Bytes) gegen DeepSeeks neues Vision-Modell. Dazu der hinterhältige `.pipec`-Bytecode-Cache-Bug, den wir dabei gefunden — und dauerhaft gefixt — haben.**

Pipe hatte bereits zwei getrennte KI-Bausteine, die sich nie ganz trafen: `agent`/`agent_ask` gibt dir eine benannte, zustandsbehaftete Konversation, aber keine Tools. `ai_tool`/`ai_with_tools` gibt dir Tool-Calling, aber keine dauerhafte Identität. Keins von beiden lässt einen Agenten eine Konversation an einen anderen übergeben. Genau diese Lücke schließt `ai_swarm`.

## Die Lücke: Agenten, die nicht miteinander reden können

Ein Swarm ist eine Menge benannter Agenten, jeder mit eigenem System-Prompt und eigenen Tools, die sich mitten in der Konversation die Kontrolle zuschieben können — während der **komplette Gesprächsverlauf** mitwandert, sodass beim Handoff nichts verloren geht.

```pipe
ai_provider "deepseek"

fn get_invoice kunde
    "Rechnung Nr. 4471, 49.90€, fällig 15.09.2026."

ai_tool "get_invoice" "Aktuelle Rechnung eines Kunden abrufen" {kunde: "Kundenname"} get_invoice

swarm_agent "triage" {system: "Leite Rechnungsfragen an 'billing' weiter. Alles andere beantwortest du selbst.", handoff: ["billing"]}
swarm_agent "billing" {system: "Du beantwortest Rechnungsfragen mit get_invoice.", tools: ["get_invoice"], handoff: ["triage"]}

"Was steht auf meiner letzten Rechnung?"
    > ai_swarm "triage"
    > print
```

Gegen einen echten DeepSeek-Key ausgeführt liefert das die Antwort des Billing-Agenten nach einem sauberen Handoff — `ai_swarm_trace` gibt dasselbe Ergebnis plus `{content, path, rounds}` zurück, sodass du genau siehst, wer die Anfrage bearbeitet hat: `["triage", "billing"]`.

## Wie Handoff tatsächlich funktioniert

Es gibt keinen magischen Router. Wenn ein Agent `handoff`-Ziele deklariert, baut `ChatSwarm` (der neue Go-Loop in `pkg/ai/swarm.go`) für diese Runde ein zusätzliches Tool zusammen — ein reserviertes `__handoff__(to: enum[...])`, das das Modell wie jedes andere Tool aufrufen kann. Tut es das, führt der Loop nicht deinen Executor aus: Er tauscht die System-Message gegen den Prompt des Zielagenten aus und führt dasselbe Nachrichten-Array in der nächsten Runde fort. Der Gesprächsverlauf — inklusive der eigenen Züge des übergebenden Agenten — bleibt unangetastet, sodass der neue Agent vollen Kontext hat, ohne Zusammenfassung oder zweiten Prompt.

`ChatSwarm` spiegelt `ChatWithTools`s Rundenlauf fast Zeile für Zeile — absichtlich: dieselbe bewährte Request/Response-Form, nur mit einem zusätzlichen Zweig für den reservierten Tool-Namen. Tool-Ausführung während eines Swarm-Laufs läuft über exakt dieselbe `executeTool`/`toolRegistry`-Maschinerie, die `ai_with_tools` bereits nutzt — die `tools`-Liste eines Swarm-Agenten sind einfach Namen, die bereits per `ai_tool` registriert sind.

## ai_vision: Content-Blöcke statt neuem Message-Typ

DeepSeek hat [ein Vision-Modell veröffentlicht](https://api-docs.deepseek.com/guides/vision/) — `deepseek-v4-flash-vision-exp` — im selben OpenAI-kompatiblen `/v1/chat/completions`-Format, das Pipe überall sonst schon spricht. Der einzige Unterschied: Der `content` einer User-Message ist ein Array aus `{type: "text", ...}`/`{type: "image_url", ...}`-Blöcken statt eines reinen Strings.

Der naheliegende Schritt wäre, `ai.Message.Content` von `string` auf etwas Reichhaltigeres zu erweitern. Haben wir nicht gemacht. Er wird als reiner String in sechs separaten Provider-Structs (`pkg/ai/providers.go`) **und** in der Response-Cache-Key-Logik verwendet — eine Fähigkeit, die (wie `ai_with_tools`) ohnehin nur mit OpenAI-kompatiblen Providern funktioniert, rechtfertigt keine breite, sich wiederholende Änderung an einem typisierten Pfad, auf dessen String-Form fünf andere Builtins angewiesen sind.

Stattdessen ist `ai.VisionChat` eine kleine, in sich geschlossene Funktion, die ihren eigenen rohen JSON-Body direkt baut — dasselbe Muster, das `ChatWithTools` intern schon für Tool-Call-Nachrichten nutzt. Null Änderungen an `Message`, `ChatRequest` oder einer der sechs Provider-Implementierungen.

```pipe
ai_provider "deepseek" {model: "deepseek-v4-flash-vision-exp"}

"https://raw.githubusercontent.com/github/explore/main/topics/go/go.png"
    > ai_vision "Was zeigt dieses Logo?"
    > print
-- -> "Dieses Logo zeigt die Programmiersprache Go
--     (auch bekannt als Golang)..."
```

`image` akzeptiert drei Formen: eine `http(s)`-URL, die unverändert durchgereicht wird (der Server des Providers holt sie, nicht Pipe), einen lokalen Dateipfad, gelesen über dasselbe Sandbox-Lese-Gate wie `read_file`, oder rohe `bytes`. Lokale Dateien und rohe Bytes werden mit Gos Standardbibliothek `http.DetectContentType` inhaltlich erkannt (keine handgeschriebene Magic-Byte-Tabelle, keine Drittanbieter-Abhängigkeit) und als `data:`-URL base64-kodiert. Beide Wege — URL und lokale Datei — sind live gegen einen echten DeepSeek-Key verifiziert, mit derselben korrekten Antwort.

## Sandbox-Gating: nichts Neues

Beide Builtins nutzen bereits bestehende Gates wieder, statt ein drittes zu erfinden. `ai_swarm`/`ai_vision` bekommen dieselbe Zwei-Zweig-Prüfung wie `ai_chat`: `profile.CanAI()` unter einem registrierten Profil, sonst das `Sandbox.AllowAI` des CLI-`--sandbox`-Flags. Der eigentliche Rückhalt ist `gateEgress(EgressChat, ...)` innerhalb von `ChatSwarm`/`VisionChat` selbst — derselbe zentrale Sandbox-Gate, den jeder `Chat`/`Stream`/`Embed`-Call seit [Runde 5 unserer Sandbox-Audits](sandbox-audit-2.html) durchläuft. Das Lesen eines lokalen Bildpfads läuft über exakt dasselbe fs-Lese-Gate wie `read_file`, unberührt von `--sandbox` (das nur Schreibzugriffe einschränkt). Dafür brauchte es keine neue Audit-Runde — alles läuft über bereits gehärtete Gates.

## Der Bug, den wir gefunden haben: Builtins ziehen um, Bytecode-Caches wissen es nicht

`ai_swarm`s drei neue Builtins in der Mitte von Pipes Builtin-Tabelle einzufügen — nicht am Ende — hat still und leise ein unabhängiges Beispiel kaputtgemacht. `xor_cipher.pipe` hing plötzlich, statt zu laufen, und die VM druckte `encrypt: key must be 16, 24, or 32 bytes`, obwohl das Skript `encrypt` gar nicht aufruft.

Die Ursache: Der Compiler bäckt die *Position* jedes Builtins in der Tabelle direkt als Integer-Index (`BuiltinScope`) in den Bytecode ein. Fügt man ein Builtin irgendwo außer am Ende ein, verschieben sich die Indizes aller nachfolgenden Builtins. Ein `.pipec`-Cache auf der Platte, kompiliert gegen die alte Tabelle, sah weiterhin „gültig" aus — gleicher Quell-Hash, gleiches `CacheVersion`-Byte — und fütterte die VM mit Bytecode, der `OpGetBuiltin` auf die *falsche Funktion* auflöste. Ein liegengebliebener lokaler Cache von vor unserer Änderung rief `encrypt` auf, wo das Skript etwas ganz anderes meinte, und lief in eine Schleife statt sauber zu fehlern.

Der Kommentar, der direkt neben `CacheVersion` steht, hatte genau dieses Fehlerbild sogar schon vorhergesagt — es hängt nur davon ab, dass ein Mensch daran denkt, eine Konstante für eine Änderung hochzuzählen, die nichts mit dem Bytecode-*Format* zu tun hat. Statt sie einmal hochzuzählen, haben wir die Fehlerklasse unmöglich gemacht: Der Dependency-Hash des Caches enthält jetzt einen Fingerprint der geordneten Builtin-Namen-Tabelle selbst, sodass **jedes** künftige Einfügen, Entfernen oder Umsortieren automatisch alle `.pipec`-Dateien auf der Platte invalidiert.

```go
// pkg/cache/cache.go — depsHash deckt jetzt auch die Builtin-Tabelle ab
for _, b := range object.Builtins {
    h.Write([]byte(b.Name))
    h.Write([]byte{0})
}
```

Ein neuer Regressionstest (`TestLoadOrCompileInvalidatesOnBuiltinTableChange`) fügt ein Builtin mitten in die Tabelle ein und prüft, dass der Cache verfehlt wird. Die Position eines Builtins ist jetzt nachweislich irrelevant für die Cache-Korrektheit — weshalb auch `ai_vision`s Registrierung keinerlei besondere Überlegung zur Platzierung brauchte.

## Ehrliche Grenzen

- **Nur OpenAI-kompatible Provider** — `openai`, `deepseek`, `ollama`, `openrouter`, `opencode`. `anthropic` nutzt ein anderes Tool-Call- und Bild-Block-Format und wird von keinem der beiden Builtins unterstützt — dieselbe geerbte Einschränkung, die `ai_with_tools` schon hat.
- **Nur ein Bild pro `ai_vision`-Aufruf** — DeepSeeks API erlaubt bis zu 600. Eine `list` von Bildern wäre eine naheliegende Erweiterung derselben Request-Form, falls später gebraucht — jetzt nicht gebaut.
- **Keine Provider-/Modell-Validierung** — derselbe zurückhaltende Ansatz wie beim Rest von `ai_provider`/`ai_model`. Zeigt `ai_vision` auf ein Nicht-Vision-Modell, bekommt man den Fehler des Providers, keinen Pipe-seitigen Check.
- **Kein geteilter Zustand über parallele Swarm-Läufe hinweg** — jeder `ai_swarm`-Aufruf besitzt seinen eigenen Gesprächsverlauf; zwischen gleichzeitigen `>>`-Swarm-Calls wird absichtlich nichts geteilt.

## Ausprobieren

```bash
DEEPSEEK_API_KEY="sk-..." pipe examples/swarm_demo.pipe
DEEPSEEK_API_KEY="sk-..." pipe examples/vision_demo.pipe
```

- Doku: [KI-Builtins §19.14–19.15](https://github.com/MachuraHarry/pipe/blob/master/docs/de/19-ki-builtins.md)
- Quellcode: [`pkg/ai/swarm.go`](https://github.com/MachuraHarry/pipe/blob/master/pkg/ai/swarm.go), [`pkg/ai/vision.go`](https://github.com/MachuraHarry/pipe/blob/master/pkg/ai/vision.go), [`pkg/object/builtins_swarm.go`](https://github.com/MachuraHarry/pipe/blob/master/pkg/object/builtins_swarm.go), [`pkg/object/builtins_vision.go`](https://github.com/MachuraHarry/pipe/blob/master/pkg/object/builtins_vision.go)
[/lang]
