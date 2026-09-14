[lang:en]# Pipe v1.3.0 — Live Swarm Control, and Four Real Bugs Found Shipping a Bot as a Single Binary[/lang]
[lang:de]# Pipe v1.3.0 — Live-Swarm-Steuerung, und vier echte Bugs beim Shippen eines Bots als Single Binary[/lang]

[lang:en]
**v1.3.0 adds live observation and mid-run control for AI swarms (`ai_swarm_stream`), the `elif` keyword, direct LLM-free tool invocation (`tool_call`), cross-process file locking (`file_lock`/`file_unlock`), and parallel tool-call batches (`parallel_safe`). But the bulk of this release came from an unusual source: trying to ship a real, working Telegram bot (Muninn) as a single self-extracting `pipe -build` binary surfaced four separate bytecode-VM bugs, a UPX corruption bug, and a genuine multi-agent oscillation loop — all fixed, all with minimal reproductions below.**
[/lang]

[lang:de]
**v1.3.0 bringt Live-Beobachtung und Eingriffsmöglichkeiten mitten in laufende AI-Swarms (`ai_swarm_stream`), das `elif`-Schlüsselwort, direkten LLM-freien Tool-Aufruf (`tool_call`), prozessübergreifendes File-Locking (`file_lock`/`file_unlock`) und parallele Tool-Call-Batches (`parallel_safe`). Der Großteil dieses Releases kam aber aus einer ungewöhnlichen Quelle: der Versuch, einen echten, funktionierenden Telegram-Bot (Muninn) als einzelne self-extracting `pipe -build`-Binary zu shippen, hat vier separate Bytecode-VM-Bugs, einen UPX-Korruptionsbug und eine echte Multi-Agent-Oszillationsschleife zutage gefördert — alle gefixt, alle mit minimalen Repros weiter unten.**
[/lang]

---

[lang:en]## What's New in v1.3[/lang]
[lang:de]## Was ist neu in v1.3[/lang]

[lang:en]
### ai_swarm_stream — Live Progress and Mid-Run Control

`ai_swarm`/`ai_swarm_trace` (v1.2.0) run a multi-agent swarm to completion and hand you back the result. `ai_swarm_stream` adds two optional closures on top of the same run, for callers that need more than a final answer.

The 4th argument, `on_progress`, fires after every step:

```pipe
fn on_progress agent event detail args_json round max_rounds
    print agent ++ " [" ++ event ++ "] " ++ detail

ai_swarm_stream "My bill looks wrong" "triage" 5 on_progress
```

`event` is one of `start`, `reasoning` (the model's raw chain-of-thought, when the provider returns one — e.g. DeepSeek reasoner models, previously silently discarded), `tool`, `handoff` (now carrying the handoff tool's full raw arguments, so you can pull out an agent's own stated `reason` for the transfer — `"triage -> billing: verify the invoice total"` instead of just "handed off to billing"), `final`, and `inject`.

The 5th argument, `round_check`, is called with no arguments at the start of every round — synchronously, same goroutine, no new concurrency — and can return `{abort: true, abort_reason: "..."}` to stop the run immediately, or `{inject: "..."}` to append a fresh instruction to the live conversation before the round proceeds. This is what lets something like a Telegram bot poll for a `/stop` command mid-swarm-run without touching the swarm's internals at all.

### Parallel Tool Calls, Where It's Actually Safe

Before this release, `ChatSwarm` ran every tool call an LLM requested in a single round strictly one after another — even when two calls were completely independent (say, two separate MCP tool invocations). The fix required a real safety analysis first, not just flipping to `go func()`:

- Tools bridged from an external MCP server (`*BuiltinInfo`, pure Go, only ever talks to a subprocess over stdio/HTTP) never touch the Pipe VM at all — already safe to run concurrently.
- Tools written in Pipe itself (`*Closure`) run on the shared VM instance, and in practice nearly all of them end up calling into the pure-Pipe sqlite module, whose handle registry is an ordinary, unsynchronized global list — running two of those in parallel would race on that list.

So `ai_tool` gained a 5th, optional `parallel_safe` boolean (default `false`) that a tool author sets explicitly once they've verified their own tool touches no shared mutable state:

```pipe
fn get_weather city
    -- pure function, touches nothing shared
    match city
        | "Berlin" -> "22°C, sunny"
        | _ -> "no data"

ai_tool "get_weather" "Get weather for a city" {city: "City name"} get_weather true
```

A batch of ≥2 non-handoff tool calls from the same round now runs `*BuiltinInfo` tools concurrently under a shared `sync.WaitGroup` automatically, and `*Closure` tools that opted in via `parallel_safe` alongside them; anything not opted in still runs synchronously inline, same as before. A round with 0 or 1 tool call — the overwhelming majority — takes the exact same code path it always did, bit-for-bit. Verified end-to-end against the real `bAiSwarm → runSwarm → ChatSwarm → runToolBatch → toolRegistry` path with `go test -race`, not just in isolated units.

### Breaking Handoff Oscillation Loops

Live-reproduced against a real swarm pair (Muninn's `kritiker`/`faktenwaechter` — a critic and a fact-checker): two agents that legitimately keep handing a conversation back to each other, each judging the other's work incomplete every single time, would burn through all of `max_rounds` and return an error instead of any answer. Neither agent was buggy — every individual handoff was a fair, correct decision — the *pair* was just stuck.

`ChatSwarm` now detects an exact `A, B, A, B` tail in the swarm's handoff path — two full round trips between the same two agents with no one else stepping in between. One back-and-forth is completely normal; an exact second repeat is a cheap, reliable signal that the pair isn't converging. When it fires, that round's handoff tool is withdrawn — the current agent sees only its own tools, no escape hatch — so a well-behaved model just answers directly instead of bouncing the conversation again. The swarm now returns *some* answer well before exhausting `max_rounds`, instead of none. Live-verified against 6 previously-flaky prompts that had triggered the loop; all completed, one visibly hit the new oscillation break exactly where expected.

### elif Keyword

```pipe
grade: if score >= 90
  "A"
elif score >= 80
  "B"
elif score >= 70
  "C"
else
  "F"
```

`elif` is sugar for `else if` — the lexer emits a dedicated `ELIF` token that dispatches back into the same `parseIfExpression` prefix-parse slot as `if`, so it recurses to arbitrary chain depth and still composes with a trailing plain `else`. Picked up two side effects along the way: the formatter no longer silently deletes every comment in a file when reformatting it (it now leaves a file untouched if it contains any whole-line `--`/`--!` comments, since `pipe fmt` rebuilds source purely from the AST, which has no comment representation — better to do nothing than delete your comments), and `import X as Y` no longer loses its alias when reformatted.

### tool_call — Deterministic Execution, No LLM Involved

```pipe
schema: {city: "Name of the city"}
ai_tool "get_weather" "Get weather for a city" schema get_weather

-- no model call, no non-determinism — just runs the tool directly:
print (tool_call "get_weather" {city: "Berlin"})
```

Same `toolRegistry` as `ai_tool`/`ai_with_tools`, same `executeTool` dispatch, same sandbox gating (`max_tool_calls`, `audit_log`) — `tool_call` just skips the model deciding *whether* to call it. Works identically for a local Pipe tool or one bridged in from `mcp_use_stdio`/`mcp_use_sse`, since both live in the same registry. This makes `tool_call` the building block for a deterministic executor: let an LLM plan once, then replay or retry individual steps without paying for (or risking the non-determinism of) another model call per step.

### file_lock / file_unlock — Real Cross-Process Locking

```pipe
lock: file_lock "shared.db"
-- ... exclusive access to shared.db, even across separate pipe processes ...
file_unlock lock
```

An OS-level exclusive advisory lock — `flock(2)` on Unix, `LockFileEx` on Windows, a no-op on `js`/wasm where there's no cross-process concept to begin with. `file_lock` blocks until it acquires the lock; if the holding process crashes without calling `file_unlock`, the OS releases the lock automatically, so a crash can't deadlock every other process out of the file forever.

The motivating case: Pipe's own pure-Pipe `sqlite` module loads a full snapshot into memory on `db_open` and atomically rewrites the whole file on `db_close` — with no locking of its own, two separate `pipe` processes touching the same database file concurrently would silently lose whichever one closed first, since the last writer's in-memory (now stale) snapshot simply overwrites the other's committed change. There was previously no way for Pipe code to coordinate across processes at all; `file_lock`/`file_unlock` close that gap without requiring every stateful module to grow its own locking scheme.
[/lang]

[lang:de]
### ai_swarm_stream — Live-Fortschritt und Eingriff mitten im Lauf

`ai_swarm`/`ai_swarm_trace` (v1.2.0) führen einen Multi-Agent-Swarm bis zum Ende aus und liefern das Ergebnis zurück. `ai_swarm_stream` legt zwei optionale Closures über denselben Lauf, für Aufrufer, die mehr als nur eine finale Antwort brauchen.

Das 4. Argument, `on_progress`, feuert nach jedem Schritt:

```pipe
fn on_progress agent event detail args_json round max_rounds
    print agent ++ " [" ++ event ++ "] " ++ detail

ai_swarm_stream "Meine Rechnung stimmt nicht" "triage" 5 on_progress
```

`event` ist eines von `start`, `reasoning` (die rohe Gedankenkette des Modells, sofern der Provider eine liefert — z. B. DeepSeek-Reasoner-Modelle, vorher stillschweigend verworfen), `tool`, `handoff` (trägt jetzt die vollständigen rohen Argumente des Handoff-Tools, sodass man den vom Agenten selbst angegebenen `reason` für die Übergabe extrahieren kann — `"triage -> billing: Rechnungssumme prüfen"` statt nur "an billing übergeben"), `final` und `inject`.

Das 5. Argument, `round_check`, wird ohne Argumente zu Beginn jeder Runde aufgerufen — synchron, gleiche Goroutine, keine neue Nebenläufigkeit — und kann `{abort: true, abort_reason: "..."}` zurückgeben, um den Lauf sofort zu stoppen, oder `{inject: "..."}`, um der laufenden Konversation eine neue Anweisung anzuhängen, bevor die Runde fortfährt. Genau das erlaubt es z. B. einem Telegram-Bot, mitten in einem Swarm-Lauf auf einen `/stop`-Befehl zu pollen, ohne die Interna des Swarms überhaupt anzufassen.

### Parallele Tool-Calls, wo es tatsächlich sicher ist

Vor diesem Release führte `ChatSwarm` jeden Tool-Call, den ein LLM in einer Runde angefordert hat, strikt nacheinander aus — auch wenn zwei Aufrufe völlig unabhängig voneinander waren (z. B. zwei separate MCP-Tool-Aufrufe). Der Fix brauchte zuerst eine echte Sicherheitsanalyse, nicht einfach ein `go func()`:

- Aus einem externen MCP-Server gebrückte Tools (`*BuiltinInfo`, reiner Go-Code, spricht nur über stdio/HTTP mit einem Subprozess) berühren die Pipe-VM überhaupt nicht — schon heute sicher parallelisierbar.
- In Pipe selbst geschriebene Tools (`*Closure`) laufen auf der gemeinsamen VM-Instanz, und praktisch alle davon greifen letztlich auf das reine Pipe-sqlite-Modul zu, dessen Handle-Registry eine gewöhnliche, unsynchronisierte globale Liste ist — zwei davon parallel laufen zu lassen würde auf dieser Liste racen.

Deshalb bekam `ai_tool` ein 5., optionales `parallel_safe`-Boolean (Standard `false`), das ein Tool-Autor explizit setzt, sobald er verifiziert hat, dass sein eigenes Tool keinen gemeinsamen veränderlichen Zustand anfasst:

```pipe
fn get_wetter stadt
    -- reine Funktion, fasst nichts Gemeinsames an
    match stadt
        | "Berlin" -> "22°C, sonnig"
        | _ -> "keine Daten"

ai_tool "get_wetter" "Wetter für eine Stadt" {stadt: "Stadtname"} get_wetter true
```

Eine Charge von ≥2 Nicht-Handoff-Tool-Calls aus derselben Runde lässt `*BuiltinInfo`-Tools jetzt automatisch parallel unter einer gemeinsamen `sync.WaitGroup` laufen, und `*Closure`-Tools, die per `parallel_safe` opt-in gemacht haben, gleich mit; alles ohne Opt-in läuft weiterhin synchron inline wie bisher. Eine Runde mit 0 oder 1 Tool-Call — die überwältigende Mehrheit — nimmt exakt denselben Codepfad wie immer, bitgenau identisch. End-to-end gegen den echten Pfad `bAiSwarm → runSwarm → ChatSwarm → runToolBatch → toolRegistry` mit `go test -race` verifiziert, nicht nur in isolierten Einheiten.

### Handoff-Oszillationsschleifen durchbrechen

Live reproduziert an einem echten Swarm-Paar (Muninns `kritiker`/`faktenwaechter`): Zwei Agenten, die sich eine Konversation legitim immer wieder gegenseitig zuschieben, weil jeder die Arbeit des anderen jedes Mal als unvollständig bewertet, würden das gesamte `max_rounds`-Budget verbrennen und einen Fehler statt irgendeiner Antwort zurückgeben. Keiner der beiden Agenten war fehlerhaft — jeder einzelne Handoff war eine faire, korrekte Entscheidung — nur das *Paar* steckte fest.

`ChatSwarm` erkennt jetzt einen exakten `A, B, A, B`-Schwanz im Handoff-Pfad des Swarms — zwei komplette Hin-und-her-Runden zwischen denselben zwei Agenten, ohne dass jemand anders dazwischen einsteigt. Ein einmaliges Hin und Her ist völlig normal; eine exakte zweite Wiederholung ist ein billiges, verlässliches Signal, dass das Paar nicht konvergiert. Tritt das ein, wird das Handoff-Tool dieser Runde zurückgezogen — der aktuelle Agent sieht nur seine eigenen Tools, kein Fluchtweg — sodass ein wohlerzogenes Modell direkt antwortet statt die Konversation erneut zurückzuschieben. Der Swarm liefert jetzt *irgendeine* Antwort deutlich vor Erschöpfen von `max_rounds`, statt keine. Live gegen 6 vorher flackernde Prompts verifiziert, die die Schleife ausgelöst hatten — alle liefen durch, einer traf sichtbar genau am erwarteten Punkt auf den neuen Oszillations-Break.

### elif-Schlüsselwort

```pipe
note: if punkte >= 90
  "A"
elif punkte >= 80
  "B"
elif punkte >= 70
  "C"
else
  "F"
```

`elif` ist syntaktischer Zucker für `else if` — der Lexer erzeugt ein eigenes `ELIF`-Token, das über denselben Prefix-Parse-Slot wie `if` zurück in `parseIfExpression` springt, sodass es beliebig tief verkettet und trotzdem mit einem abschließenden reinen `else` komponiert. Zwei Nebeneffekte fielen dabei ab: Der Formatter löscht beim Reformatieren nicht mehr stillschweigend jeden Kommentar in einer Datei (er lässt eine Datei jetzt unangetastet, wenn sie irgendeinen ganzzeiligen `--`/`--!`-Kommentar enthält, da `pipe fmt` den Quellcode rein aus dem AST rekonstruiert, das keine Kommentar-Repräsentation kennt — lieber nichts tun, als Kommentare zu löschen), und `import X as Y` verliert beim Reformatieren nicht mehr seinen Alias.

### tool_call — Deterministische Ausführung ohne LLM

```pipe
schema: {stadt: "Name der Stadt"}
ai_tool "get_wetter" "Wetter für eine Stadt" schema get_wetter

-- kein Modell-Call, keine Nichtdeterminismus — führt das Tool direkt aus:
print (tool_call "get_wetter" {stadt: "Berlin"})
```

Dieselbe `toolRegistry` wie `ai_tool`/`ai_with_tools`, derselbe `executeTool`-Dispatch, dasselbe Sandbox-Gating (`max_tool_calls`, `audit_log`) — `tool_call` überspringt nur die Entscheidung des Modells, *ob* es aufgerufen wird. Funktioniert identisch für ein lokales Pipe-Tool oder eines, das über `mcp_use_stdio`/`mcp_use_sse` gebrückt wurde, da beide in derselben Registry liegen. Das macht `tool_call` zum Baustein für einen deterministischen Executor: Ein LLM plant einmal, danach lassen sich einzelne Schritte wiederholen oder erneut versuchen, ohne für (oder das Nichtdeterminismus-Risiko) einen weiteren Modell-Call pro Schritt zu zahlen.

### file_lock / file_unlock — Echtes prozessübergreifendes Locking

```pipe
lock: file_lock "shared.db"
-- ... exklusiver Zugriff auf shared.db, auch über separate pipe-Prozesse hinweg ...
file_unlock lock
```

Ein Betriebssystem-Level-Exklusiv-Lock — `flock(2)` unter Unix, `LockFileEx` unter Windows, ein No-Op unter `js`/wasm, wo es ohnehin kein Prozessübergreifend-Konzept gibt. `file_lock` blockiert, bis der Lock erworben ist; stürzt der haltende Prozess ab, ohne `file_unlock` aufzurufen, gibt das Betriebssystem den Lock automatisch frei — ein Absturz kann also nicht alle anderen Prozesse für immer aus der Datei aussperren.

Der auslösende Fall: Pipes eigenes reines Pipe-`sqlite`-Modul lädt bei `db_open` einen kompletten Snapshot in den Speicher und schreibt bei `db_close` die gesamte Datei atomar neu — ohne eigenes Locking würden zwei separate `pipe`-Prozesse, die dieselbe Datenbankdatei gleichzeitig anfassen, stillschweigend den Verlust dessen riskieren, wer zuerst schließt, da der zuletzt schreibende (jetzt veraltete) Speicher-Snapshot einfach die committete Änderung des anderen überschreibt. Bisher gab es für Pipe-Code keine Möglichkeit, sich prozessübergreifend zu koordinieren; `file_lock`/`file_unlock` schließen diese Lücke, ohne dass jedes zustandsbehaftete Modul ein eigenes Locking-Schema bräuchte.
[/lang]

---

[lang:en]## Shipping Muninn as a Single Binary: pipe -build Goes Multi-File[/lang]
[lang:de]## Muninn als Single Binary shippen: pipe -build wird multi-file-fähig[/lang]

[lang:en]
Most of this release wasn't planned as a feature list — it fell out of actually trying to use Pipe the way we tell people to. [Muninn](https://github.com/MachuraHarry/muninn) is a real Telegram bot: a main script plus 9 files under `modules/`, cross-importing each other via relative paths, using `ai_swarm`, sqlite, and a poll loop that runs `try`/`catch` once per cycle. The goal: package the whole thing as one self-extracting `pipe -build` binary — the same feature the README has advertised since v0.6. It didn't work. Here's everything that was actually wrong, in the order we found it.

### Bug 1: a builtin's table position, baked into stale bytecode

First symptom, nothing to do with `-build` yet: a totally unrelated example script started hanging under `-vm` after `ai_swarm`'s new builtins were added in the *middle* of the builtin table instead of at the end. The compiler bakes each builtin's position into the bytecode as a raw integer index — insert one anywhere but the end, and every later builtin's index silently shifts. A stale `.pipec` disk cache, compiled against the old table, still looked "valid" (same source hash, same cache-version byte) and fed the VM bytecode that resolved a call to the *wrong function entirely*.

The fix in the previous release (v1.2.0) bumped a cache-version constant for that one case. This release made the entire bug class impossible: the cache's dependency hash now includes a fingerprint of the ordered builtin-name table itself, so any future insertion, removal, or reorder self-invalidates every `.pipec` on disk automatically — no longer dependent on a human remembering to bump anything.

### Bug 2: a bare import can permanently break a builtin for everyone else

Muninn's test file bare-imports `"sqlite"` (for direct DB fixtures) and separately imports a Docker helper module as `dt`, which just calls the real `exec` builtin for `docker pull`/`docker run` — nothing to do with sqlite. `pipe -vm -test` refused to even compile: `undefined variable: exec`.

A bare `import "sqlite"` flattens sqlite's own exports — including its own `exec(handle, sql)` — directly into the importer's symbol table. That's by design, so a user's own top-level redefinition of a builtin works. But it *mutates the lookup table itself*, not just a local shadow: the original builtin entry at that slot is gone, permanently, for that table. A later aliased import (`dt`) resolves its own identifiers by walking the importer's ancestor scope chain — and since the entry it finds along the way is no longer a builtin, it reports `exec` as undefined, even though `dt.pipe` never touched sqlite at all.

Fix: a static, never-mutated name→index lookup, consulted only as a last-resort fallback after the normal scope walk fails — mirroring how the tree-walker already resolves builtins from a persistent registry independent of any mutable scope chain. Zero behavior change for the non-shadowed case.

### Bugs 3 & 4: a leaking stack slot, and two for-loops sharing one set of bookkeeping variables

Past that compiler error, `pipe -vm -test` on Muninn hit runtime corruption — wrong values, garbled errors, eventually a hard out-of-memory crash. Bisected down to two independent minimal repros.

The first: a `catch` block whose *last* statement is a variable assignment (`catch e { print(...); tools: [] }`) got compiled through the generic one-size-fits-all path, which already fully consumes the assignment's own pushed value — and then an *extra* `OpPop` was emitted on top of that, matching the (wrong, for this case) assumption that any non-expression statement leaves one value still needing popping. For a local variable, that extra pop walked the VM's stack pointer one slot into the function's reserved-locals region, silently corrupting an unrelated sibling local.

The second, and the actual root cause of the OOM crash: `for x in y` compiles its iteration bookkeeping into two hardcoded, fixed symbol names — `__list__` and `__idx__`. A for-loop body doesn't open its own scope, so a *nested* for-loop compiles the exact same two names into the exact same scope as its enclosing loop, and symbol definition reuses an existing name rather than allocating a fresh slot. Both loops silently shared one pair of slots. The inner loop's own init (reset index to 0, list to its own) stomped the outer loop's in-progress state mid-body; when control returned to the outer loop's condition check, it read back the *inner* loop's post-loop state instead of its own and exited after exactly one iteration, regardless of how many elements it actually had:

```pipe
for o in ["a","b","c"]
    print o
    for i in ["x"]
        print i
-- prints "a"/"x" only — never reaches "b" or "c"
```

Muninn's own `mcp.discovered_tools_by_source` has exactly this nested shape, so it always returned after one outer iteration no matter the input size — and elsewhere in the codebase, a similarly-shaped loop over an unbounded collection spun long enough retrying a corrupted "iteration" to exhaust memory before the test harness's own timeout could step in.

Fixed with a per-`Compiler` counter that gives every compiled for-loop its own unique bookkeeping names (`__list__1`, `__idx__1`, `__list__2`, ...) — no two loops, nested or sequential, can collide on the same slots again. A separate, related bug (also found via Muninn, fixed in the same investigation): a *multi-statement* `try` block leaked exactly one operand-stack slot per non-final statement on every no-error run — invisible for a one-off `try`/`catch`, fatal for Muninn's own main poll loop, which ran one multi-statement `try` block per cycle and crashed with a stack overflow after about a minute:

```pipe
i: 0
while i < 2000
    try
        a: 1
        b: 2
        c: a + b
    catch e
        print "err"
    i: i + 1
-- -vm: "stack overflow: recursion too deep (operand stack exhausted)"
-- well before i reaches 2000
```

All four of these are `-vm`-only — the tree-walker was correct throughout, which is exactly why they'd never surfaced before: `pipe -build` binaries *always* run through the bytecode VM (`cmd/pipe/main.go`'s `runEmbedded` never touches the tree-walker), so a project that only ever ran under the default tree-walker had no way to hit any of them until it tried to become a `-build` binary.

### Multi-file projects, and a working directory that stays put

With the VM correct, `pipe -build` itself still couldn't produce a working Muninn binary. Two separate gaps:

`pkg/build/build.go` stored every `--embed-file` argument under just its base name, discarding any subdirectory — `modules/telegram.pipe` got embedded and later re-extracted as bare `telegram.pipe`, no `modules/` directory at all, so every one of Muninn's own `import "modules/telegram.pipe"` statements failed once the binary actually ran. Fixed by preserving the given path (slash-normalized) for anything that stays within the project, while still falling back to the flattened bare-name behavior — unchanged from before — for anything containing `..` or an absolute path, so a stray or malicious `--embed-file` argument still can't bake a traversal path into the binary.

Separately, `runEmbedded` used to `os.Chdir` the *entire process* into the extracted-files temp directory so relative imports would resolve — which also silently redirected Muninn's own `.env` config loading and its sqlite database file into that throwaway temp directory instead of wherever the user actually launched the binary from. Fixed by adding the extraction directory to `PIPE_PATH` (the existing import search-path variable) instead of changing the working directory — imports resolve exactly the same way, and the running program's actual working directory is left untouched.

With both fixed: `pipe -build muninn.pipe -o muninn --embed-file modules/*.pipe` produced a single ~9 MB binary that resolved all 9 modules, read its real `.env` from the real working directory, created its sqlite database there too, and reached "Muninn started." — a full startup, end to end, from one executable.

### And then -upx corrupted it

`-build`'s `-upx` flag (advertised in `pipe -h` as "~60% smaller") turned out to have two separate, real bugs, only visible once the multi-file work above actually worked. First: the UPX invocation itself (`upx -q outPath -o outPath`) rejected pointing `-o` at the same path as the input on the UPX version in use — `pipe -build` reported success anyway (the error only went to stderr), silently leaving the output uncompressed. Second, and worse: UPX repacks the *entire* file it's given and does not preserve arbitrary bytes appended after what it considers the executable's end. The build flow ran UPX as a step *after* the self-extracting binary was already fully assembled (interpreter + markers + embedded source and files) — so UPX happily corrupted the appended payload along with everything else. On a trivial one-line script this printed garbage instead of running; on the full Muninn build it was worse: no error at all, just silent immediate exit, because the marker search that locates the appended payload found nothing to run.

Fixed by compressing a copy of just the interpreter *first*, before it's ever copied into the output file or has anything appended to it — compression now only ever touches the interpreter, and the payload is never seen by UPX at all. Verified end to end: trivial script builds identically compressed and uncompressed (3.7 MB vs 8.8 MB, ~58% smaller); full Muninn build (9 embedded modules) went from 9.1 MB to 4.0 MB (~56% smaller) and, started in web mode with a real API key, returned a correct, non-cached answer through its own dashboard endpoint.
[/lang]

[lang:de]
Der Großteil dieses Releases war nicht als Feature-Liste geplant — er fiel dabei ab, Pipe tatsächlich so zu benutzen, wie wir es Leuten empfehlen. [Muninn](https://github.com/MachuraHarry/muninn) ist ein echter Telegram-Bot: ein Hauptskript plus 9 Dateien unter `modules/`, die sich gegenseitig über relative Pfade importieren, mit `ai_swarm`, sqlite und einer Poll-Schleife, die einmal pro Zyklus `try`/`catch` ausführt. Das Ziel: das Ganze als eine einzige self-extracting `pipe -build`-Binary zu paketieren — genau das Feature, das das README seit v0.6 bewirbt. Es hat nicht funktioniert. Hier ist alles, was tatsächlich kaputt war, in der Reihenfolge, in der wir es gefunden haben.

### Bug 1: Die Tabellenposition eines Builtins, eingebacken in veralteten Bytecode

Erstes Symptom, noch nichts mit `-build` zu tun: Ein völlig unabhängiges Beispielskript fing an, unter `-vm` zu hängen, nachdem `ai_swarm`s neue Builtins in der *Mitte* der Builtin-Tabelle eingefügt wurden statt am Ende. Der Compiler bäckt die Position jedes Builtins als rohen Integer-Index in den Bytecode ein — fügt man eines irgendwo außer am Ende ein, verschieben sich die Indizes aller nachfolgenden Builtins still und leise. Ein veralteter `.pipec`-Cache, kompiliert gegen die alte Tabelle, sah weiterhin "gültig" aus (gleicher Quell-Hash, gleiches Cache-Versions-Byte) und fütterte die VM mit Bytecode, der einen Aufruf auf die *komplett falsche Funktion* auflöste.

Der Fix im vorherigen Release (v1.2.0) hat für genau diesen einen Fall eine Cache-Versions-Konstante hochgezählt. Dieses Release macht die gesamte Bugklasse unmöglich: Der Dependency-Hash des Caches enthält jetzt einen Fingerprint der geordneten Builtin-Namen-Tabelle selbst, sodass jedes künftige Einfügen, Entfernen oder Umsortieren automatisch alle `.pipec`-Dateien auf der Platte invalidiert — unabhängig davon, ob ein Mensch daran denkt, etwas hochzuzählen.

### Bug 2: Ein bare Import kann ein Builtin dauerhaft für alle anderen kaputt machen

Muninns Testdatei importiert `"sqlite"` bare (für direkte DB-Fixtures) und separat ein Docker-Hilfsmodul als `dt`, das nur das echte `exec`-Builtin für `docker pull`/`docker run` aufruft — nichts mit sqlite zu tun. `pipe -vm -test` weigerte sich, überhaupt zu kompilieren: `undefined variable: exec`.

Ein bare `import "sqlite"` flacht sqlites eigene Exports — inklusive dessen eigenem `exec(handle, sql)` — direkt in die Symboltabelle des Importierenden ab. Das ist Absicht, damit die eigene Top-Level-Neudefinition eines Builtins durch den Nutzer funktioniert. Aber es *mutiert die Lookup-Tabelle selbst*, nicht nur einen lokalen Schatten: Der ursprüngliche Builtin-Eintrag an dieser Stelle ist für diese Tabelle dauerhaft weg. Ein späterer aliasierter Import (`dt`) löst seine eigenen Bezeichner auf, indem er die Ancestor-Scope-Kette des Importierenden entlangläuft — und da der dabei gefundene Eintrag kein Builtin mehr ist, meldet er `exec` als undefiniert, obwohl `dt.pipe` sqlite nie berührt hat.

Fix: eine statische, nie mutierte Name-zu-Index-Lookup-Tabelle, nur als letzter Fallback konsultiert, nachdem der normale Scope-Walk scheitert — spiegelt, wie der Tree-Walker Builtins bereits aus einer persistenten, von jeder veränderlichen Scope-Kette unabhängigen Registry auflöst. Keine Verhaltensänderung für den nicht-überschatteten Fall.

### Bugs 3 & 4: Ein leckender Stack-Slot, und zwei For-Schleifen, die sich ein Set Buchhaltungsvariablen teilen

Nach diesem Compiler-Fehler traf `pipe -vm -test` auf Muninn zur Laufzeit auf Korruption — falsche Werte, verstümmelte Fehler, schließlich ein harter Out-of-Memory-Crash. Per Bisektion auf zwei unabhängige minimale Repros heruntergebrochen.

Der erste: Ein `catch`-Block, dessen *letzte* Anweisung eine Variablenzuweisung ist (`catch e { print(...); tools: [] }`), wurde über den generischen Einheitspfad kompiliert, der den gepushten Wert der Zuweisung bereits vollständig konsumiert — und danach wurde ein *zusätzliches* `OpPop` obendrauf emittiert, passend zur (für diesen Fall falschen) Annahme, dass jede Nicht-Ausdrucks-Anweisung noch einen zu poppenden Wert hinterlässt. Bei einer lokalen Variable lief dieser zusätzliche Pop den VM-Stack-Pointer einen Slot in den reservierten Locals-Bereich der Funktion hinein und korrumpierte still eine unbeteiligte benachbarte Lokale.

Der zweite, und die eigentliche Ursache des OOM-Crashs: `for x in y` kompiliert seine Iterations-Buchhaltung in zwei fest hartcodierte Symbolnamen — `__list__` und `__idx__`. Ein For-Schleifen-Body öffnet keinen eigenen Scope, sodass eine *verschachtelte* For-Schleife exakt dieselben zwei Namen in exakt denselben Scope wie ihre umgebende Schleife kompiliert, und die Symboldefinition einen bestehenden Namen wiederverwendet statt einen frischen Slot zu allozieren. Beide Schleifen teilten sich still ein Paar Slots. Die Init der inneren Schleife (Index auf 0 zurücksetzen, Liste auf ihre eigene) überschrieb den laufenden Zustand der äußeren Schleife mitten im Body; als die Kontrolle zur Bedingungsprüfung der äußeren Schleife zurückkehrte, las sie den Nachzustand der *inneren* Schleife statt ihres eigenen und beendete sich nach genau einer Iteration, egal wie viele Elemente sie tatsächlich hatte:

```pipe
for o in ["a","b","c"]
    print o
    for i in ["x"]
        print i
-- druckt nur "a"/"x" — erreicht "b" und "c" nie
```

Muninns eigenes `mcp.discovered_tools_by_source` hat genau diese verschachtelte Form, sodass es unabhängig von der Eingabegröße immer nach einer äußeren Iteration zurückkehrte — und an anderer Stelle im Code drehte eine ähnlich geformte Schleife über eine unbeschränkte Sammlung lange genug im Kreis, eine korrumpierte "Iteration" erneut zu versuchen, um den Speicher zu erschöpfen, bevor das Timeout des Test-Harness selbst eingreifen konnte.

Gefixt mit einem Zähler pro `Compiler`, der jeder kompilierten For-Schleife eigene, eindeutige Buchhaltungsnamen gibt (`__list__1`, `__idx__1`, `__list__2`, ...) — keine zwei Schleifen, verschachtelt oder sequenziell, können mehr auf denselben Slots kollidieren. Ein separater, verwandter Bug (ebenfalls über Muninn gefunden, in derselben Untersuchung gefixt): Ein *mehrzeiliger* `try`-Block leckte bei jedem fehlerfreien Durchlauf genau einen Operand-Stack-Slot pro Nicht-letzter-Anweisung — unsichtbar für ein einmaliges `try`/`catch`, fatal für Muninns eigene Poll-Hauptschleife, die einen mehrzeiligen `try`-Block pro Zyklus ausführte und nach etwa einer Minute mit Stack-Overflow abstürzte:

```pipe
i: 0
while i < 2000
    try
        a: 1
        b: 2
        c: a + b
    catch e
        print "err"
    i: i + 1
-- -vm: "stack overflow: recursion too deep (operand stack exhausted)"
-- deutlich bevor i 2000 erreicht
```

Alle vier dieser Bugs betreffen nur `-vm` — der Tree-Walker war die ganze Zeit korrekt, weshalb sie vorher nie aufgetaucht sind: `pipe -build`-Binaries laufen *immer* über die Bytecode-VM (`runEmbedded` in `cmd/pipe/main.go` berührt den Tree-Walker nie), sodass ein Projekt, das bisher nur unter dem Standard-Tree-Walker lief, keine Chance hatte, einen davon zu treffen — bis es versuchte, eine `-build`-Binary zu werden.

### Multi-File-Projekte, und ein Arbeitsverzeichnis, das an Ort und Stelle bleibt

Selbst mit korrekter VM konnte `pipe -build` noch keine funktionierende Muninn-Binary erzeugen. Zwei separate Lücken:

`pkg/build/build.go` speicherte jedes `--embed-file`-Argument nur unter seinem Basisnamen, ohne jedes Unterverzeichnis — `modules/telegram.pipe` wurde eingebettet und später als nacktes `telegram.pipe` wieder extrahiert, ganz ohne `modules/`-Verzeichnis, sodass jede von Muninns `import "modules/telegram.pipe"`-Anweisungen fehlschlug, sobald die Binary tatsächlich lief. Gefixt, indem der angegebene Pfad (slash-normalisiert) für alles innerhalb des Projekts erhalten bleibt, während für alles mit `..` oder einem absoluten Pfad weiterhin das alte, abgeflachte Verhalten greift — unverändert wie zuvor —, sodass ein verirrtes oder böswilliges `--embed-file`-Argument weiterhin keinen Traversal-Pfad in die Binary einbacken kann.

Separat davon hat `runEmbedded` bisher den *gesamten Prozess* per `os.Chdir` in das temporäre Extraktionsverzeichnis gewechselt, damit relative Imports auflösen — was auch Muninns eigenes `.env`-Config-Laden und seine sqlite-Datenbankdatei still in dieses Wegwerf-Temp-Verzeichnis umgeleitet hat, statt dorthin, von wo der Nutzer die Binary tatsächlich gestartet hat. Gefixt, indem das Extraktionsverzeichnis zu `PIPE_PATH` (der bestehenden Import-Suchpfad-Variable) hinzugefügt wird, statt das Arbeitsverzeichnis zu wechseln — Imports lösen exakt gleich auf, und das tatsächliche Arbeitsverzeichnis des laufenden Programms bleibt unangetastet.

Mit beidem gefixt: `pipe -build muninn.pipe -o muninn --embed-file modules/*.pipe` erzeugte eine einzelne ~9-MB-Binary, die alle 9 Module auflöste, ihr echtes `.env` aus dem echten Arbeitsverzeichnis las, dort auch ihre sqlite-Datenbank anlegte und "Muninn started." erreichte — ein vollständiger Start, Ende zu Ende, aus einer einzigen ausführbaren Datei.

### Und dann hat -upx sie korrumpiert

`-build`s `-upx`-Flag (in `pipe -h` als "~60% kleiner" beworben) hatte zwei separate, echte Bugs, die erst sichtbar wurden, nachdem die Multi-File-Arbeit oben tatsächlich funktionierte. Erstens: Der UPX-Aufruf selbst (`upx -q outPath -o outPath`) lehnte es auf der verwendeten UPX-Version ab, `-o` auf denselben Pfad wie den Input zeigen zu lassen — `pipe -build` meldete trotzdem Erfolg (der Fehler ging nur nach stderr), und ließ die Ausgabe still unkomprimiert. Zweitens, und schlimmer: UPX packt die *gesamte* Datei, die man ihm gibt, neu und bewahrt keine beliebigen Bytes, die nach dem angehängt sind, was es als Ende der ausführbaren Datei betrachtet. Der Build-Ablauf führte UPX als Schritt aus, *nachdem* die self-extracting Binary bereits vollständig zusammengesetzt war (Interpreter + Marker + eingebetteter Quellcode und Dateien) — UPX korrumpierte also fröhlich die angehängte Payload mit allem anderen. Bei einem trivialen Einzeiler-Skript druckte das Müll statt zu laufen; beim vollständigen Muninn-Build war es schlimmer: gar kein Fehler, nur stiller sofortiger Exit, weil die Marker-Suche, die die angehängte Payload findet, nichts zum Ausführen fand.

Gefixt, indem eine Kopie *nur* des Interpreters komprimiert wird — *bevor* er überhaupt in die Ausgabedatei kopiert wird oder irgendetwas angehängt bekommt — Kompression berührt jetzt nur noch den Interpreter, und die Payload bekommt UPX nie zu Gesicht. Ende-zu-Ende verifiziert: Ein triviales Skript baut identisch komprimiert und unkomprimiert (3,7 MB vs. 8,8 MB, ~58% kleiner); der vollständige Muninn-Build (9 eingebettete Module) ging von 9,1 MB auf 4,0 MB (~56% kleiner) und lieferte, im Web-Modus mit einem echten API-Key gestartet, über seinen eigenen Dashboard-Endpoint eine korrekte, nicht gecachte Antwort zurück.
[/lang]

---

[lang:en]## Sandbox Audit Rounds 9-11[/lang]
[lang:de]## Sandbox-Audit-Runden 9-11[/lang]

[lang:en]
The red-team audits continued past v1.2.0's rounds 7-8:

- **Round 9 — `exec_whitelist` shell injection.** `exec_whitelist` only ever inspected the *first token* of a command string, then let the full, unfiltered string reach a real shell (`sh -c`). A whitelist of `["echo"]` still fully ran `echo hi && cat /etc/hostname`. Fixed by tokenizing with a shell-word splitter and exec'ing directly as argv, no shell involved — metacharacters (`&& ; | \` $() > <`) become inert literal arguments. Traded off deliberately: `cd x && git ...`-style composition no longer works under a whitelist at all, by design, since there's no shell left to interpret it.
- **Round 10 — coordinated agents, one shared host.** Two isolated agents allowed to share a single network host turn it into an inter-agent message board in minutes — the sandbox can't stop that by design, but every byte still lands in the audit log. ([full report](round-10-interagent-channel.html))
- **Round 11 — a swarm adds nothing, but `ln` does.** Round 11 asked the obvious follow-up question after v1.2.0 shipped `ai_swarm`: can a *coordinated* swarm escape where a single agent cannot? Two parallel, fully isolated runs (DeepSeek and OpenCode) confirmed it can't — every swarm member shares the same executor, `ActiveProfile`, and budget pool, with identical per-tool gates. But the single-agent baseline surfaced a real, independently-verified escape: a whitelisted `exec ln` has no filesystem isolation of its own, so a hard link planted inside the temp-only jail points straight at a host inode — and `read_file`'s path-only canonicalization dereferences it, handing back the real `/etc/shadow`/`/etc/passwd`. Reproduced with a harmless stand-in file; the actual fix (running `exec` under real filesystem isolation) is documented as a follow-up rather than patched in this release. ([full report](round-11-swarm-escape.html))
[/lang]

[lang:de]
Die Red-Team-Audits liefen nach v1.2.0s Runden 7-8 weiter:

- **Runde 9 — `exec_whitelist`-Shell-Injection.** `exec_whitelist` prüfte bisher nur das *erste Token* eines Befehlsstrings und ließ den vollständigen, ungefilterten String dann eine echte Shell erreichen (`sh -c`). Eine Whitelist von `["echo"]` führte weiterhin vollständig `echo hi && cat /etc/hostname` aus. Gefixt durch Tokenisierung mit einem Shell-Wort-Splitter und direktem Exec als argv, ganz ohne Shell — Metazeichen (`&& ; | \` $() > <`) werden zu inerten Literal-Argumenten. Bewusst in Kauf genommen: `cd x && git ...`-artige Komposition funktioniert unter einer Whitelist jetzt gar nicht mehr, by design, da keine Shell mehr übrig ist, die sie interpretieren könnte.
- **Runde 10 — koordinierte Agenten, ein geteilter Host.** Zwei isolierte Agenten, die sich einen Netzwerk-Host teilen dürfen, verwandeln ihn in Minuten in ein Inter-Agent-Message-Board — die Sandbox kann das by design nicht verhindern, aber jedes Byte landet trotzdem im Audit-Log. ([vollständiger Bericht](round-10-interagent-channel.html))
- **Runde 11 — ein Swarm bringt nichts, aber `ln` schon.** Runde 11 stellte die naheliegende Folgefrage, nachdem v1.2.0 `ai_swarm` gebracht hatte: Kann ein *koordinierter* Swarm entkommen, wo ein einzelner Agent es nicht kann? Zwei parallele, vollständig isolierte Läufe (DeepSeek und OpenCode) bestätigten: nein — jedes Swarm-Mitglied teilt sich denselben Executor, dasselbe `ActiveProfile` und denselben Budget-Pool, mit identischen Per-Tool-Gates. Die Single-Agent-Baseline förderte aber einen echten, unabhängig verifizierten Escape zutage: Ein gewhitelistetes `exec ln` hat keine eigene Dateisystem-Isolation, sodass ein im Temp-only-Jail platzierter Hardlink direkt auf einen Host-Inode zeigt — und die reine Pfad-Kanonisierung von `read_file` löst ihn auf und liefert den echten `/etc/shadow`/`/etc/passwd` zurück. Mit einer harmlosen Ersatzdatei reproduziert; der eigentliche Fix (`exec` unter echter Dateisystem-Isolation) ist als Follow-up dokumentiert statt in diesem Release gepatcht. ([vollständiger Bericht](round-11-swarm-escape.html))
[/lang]

---

[lang:en]## Smaller Fixes Worth Knowing About[/lang]
[lang:de]## Kleinere Fixes, die man kennen sollte[/lang]

[lang:en]
- **`spawn`/`await`:** `push` and `set` no longer eagerly resolve `Future` arguments before dispatch. This silently serialized the common "spawn N tasks, push their Futures into a list, await them afterwards" pattern — each `push` blocked on its own Future before the loop could move on, so the spawned closures never actually ran concurrently. `push`/`set` now get the same unresolved-argument exemption `await` already had, in both the tree-walker and the VM.
- **`pipe -get` module cache invalidation** ([#4](https://github.com/MachuraHarry/pipe/issues/4)): the module URL disk cache was written but never invalidated, so an explicit `pipe -get <name>` served a stale copy forever — live-observed serving a pre-`ALTER` copy of a module that had already been republished. `pipe -get` now explicitly invalidates the cache before resolving; ordinary runtime imports still cache normally, to avoid a network hit on every single program run.
- **`ai_cost` cache hit/miss tracking**: `cache_hits`/`cache_misses` were always 0 — the provider's real prompt-cache usage from the response was never actually read.
- **MCP client fixes**: the client bridge dropped or scrambled optional tool arguments; the client omitted `"arguments"` entirely for empty tool/prompt calls; stdio subprocesses (and their own children) leaked on every restart; a watchdog process was referenced in the code but never actually implemented; the stdio client discarded captured stderr output on a crash that happened before any response came back.
[/lang]

[lang:de]
- **`spawn`/`await`:** `push` und `set` lösen `Future`-Argumente vor dem Dispatch nicht mehr eager auf. Das serialisierte still das gängige Muster "N Tasks spawnen, ihre Futures in eine Liste pushen, danach awaiten" — jeder `push` blockierte auf sein eigenes Future, bevor die Schleife weiterlaufen konnte, sodass die gespawnten Closures nie tatsächlich parallel liefen. `push`/`set` bekommen jetzt dieselbe Ausnahme für unaufgelöste Argumente, die `await` schon hatte, sowohl im Tree-Walker als auch in der VM.
- **`pipe -get`-Modul-Cache-Invalidierung** ([#4](https://github.com/MachuraHarry/pipe/issues/4)): Der Modul-URL-Disk-Cache wurde geschrieben, aber nie invalidiert, sodass ein explizites `pipe -get <name>` für immer eine veraltete Kopie auslieferte — live beobachtet beim Ausliefern einer Vor-`ALTER`-Kopie eines bereits neu veröffentlichten Moduls. `pipe -get` invalidiert den Cache jetzt explizit vor dem Auflösen; gewöhnliche Runtime-Imports cachen weiterhin normal, um nicht bei jedem einzelnen Programmlauf einen Netzwerk-Hit zu riskieren.
- **`ai_cost`-Cache-Hit/Miss-Tracking**: `cache_hits`/`cache_misses` standen immer auf 0 — die echte Prompt-Cache-Nutzung des Providers aus der Antwort wurde nie tatsächlich gelesen.
- **MCP-Client-Fixes**: Die Client-Brücke verlor oder verwürfelte optionale Tool-Argumente; der Client ließ `"arguments"` bei leeren Tool-/Prompt-Aufrufen komplett weg; stdio-Subprozesse (und ihre eigenen Kindprozesse) leakten bei jedem Neustart; ein Watchdog-Prozess wurde im Code referenziert, aber nie tatsächlich implementiert; der stdio-Client verwarf mitgeschnittene stderr-Ausgabe bei einem Absturz, der passierte, bevor irgendeine Antwort zurückkam.
[/lang]

---

[lang:en]## The Numbers[/lang]
[lang:de]## Die Zahlen[/lang]

| | v1.2.0 | v1.3.0 |
|---|---|---|
| **Builtins** | 242 | 246 |
| **AI Builtins** | 39 | 41 |
| **Tests** | 711 | 752 |
| **Sandbox audit rounds** | 8 | 11 |
| **Binary size** | ~8 MB | ~8 MB |
| **Dependencies** | 0 | 0 |

---

[lang:en]## Install[/lang]
[lang:de]## Installation[/lang]

[lang:en]
```sh
curl -fsSL https://pipe-lang.com/install.sh | bash
```

Already on Pipe? Just update in place:

```sh
pipe --update
```
[/lang]

[lang:de]
```sh
curl -fsSL https://pipe-lang.com/install.sh | bash
```

Schon auf Pipe? Einfach an Ort und Stelle aktualisieren:

```sh
pipe --update
```
[/lang]

[lang:en]## What's Next[/lang]
[lang:de]## Was kommt als Nächstes[/lang]

[lang:en]
- **`exec` under real filesystem isolation** — closing round 11's hard-link escape properly, not just documenting it
- **Coroutines and `select`** over the existing channel/mutex/semaphore primitives
- **Sets** — a dedicated unique-collection type
- **VSCode 2.0** — debugger support, richer snippets
[/lang]

[lang:de]
- **`exec` unter echter Dateisystem-Isolation** — Runde 11s Hardlink-Escape sauber schließen, nicht nur dokumentieren
- **Coroutines und `select`** über den bestehenden Channel-/Mutex-/Semaphore-Primitiven
- **Sets** — ein eigener Unique-Collection-Typ
- **VSCode 2.0** — Debugger-Unterstützung, umfangreichere Snippets
[/lang]

[lang:en]
---

**Pipe v1.3.0 is what fell out of trying to run a real, working bot on it.** From live swarm control to four VM bugs that only a real multi-module project could have found — this release exists because we tried to eat our own dog food, and it didn't go down easy the first time.

[Star Pipe on GitHub](https://github.com/MachuraHarry/pipe) · [Read the Docs](https://pipe-lang.com/docs.html) · [Try the Playground](https://pipe-lang.com/playground.html) · [Join Discord](https://discord.gg/kdjce8hnYw)
[/lang]

[lang:de]
---

**Pipe v1.3.0 ist das, was dabei herauskam, einen echten, funktionierenden Bot damit zu betreiben.** Von Live-Swarm-Steuerung bis zu vier VM-Bugs, die nur ein echtes Multi-Modul-Projekt hätte finden können — dieses Release existiert, weil wir unser eigenes Hundefutter gegessen haben, und das ist beim ersten Mal nicht reibungslos runtergegangen.

[Pipe auf GitHub starren](https://github.com/MachuraHarry/pipe) · [Dokumentation lesen](https://pipe-lang.com/docs.html) · [Playground ausprobieren](https://pipe-lang.com/playground.html) · [Discord beitreten](https://discord.gg/kdjce8hnYw)
[/lang]
