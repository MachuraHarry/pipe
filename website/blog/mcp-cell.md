[lang:en]# 🧱 The MCP Cell — an AI That Can Only Talk to Pipe's Own MCP Server[/lang]
[lang:de]# 🧱 Die MCP-Zelle — eine KI, die nur mit Pipes eigenem MCP-Server sprechen kann[/lang]

[lang:en]
**The smallest demo of the strongest idea: a sandboxed AI agent whose entire
universe is Pipe's own MCP server. No free network, no free filesystem, no
shell — just five tools behind a whitelist.**

Everyone asks the same question about sandboxes: *"But if the agent has tools,
aren't the tools the attack surface?"*

Fair. So this time I flipped it. Instead of giving the agent a toolbox and
hoping the sandbox holds, I built a profile where the **only** things the
agent can do are the MCP tools that *I* chose to expose — and nothing else.
The AI's entire world is one MCP server, and that server lives inside the
sandbox too.

The example lives in [`examples/mcp_sandbox_agent.pipe`](https://github.com/MachuraHarry/pipe/blob/master/examples/mcp_sandbox_agent.pipe).
It has two modes: an in-process agent mode (`agent`) and a real MCP server on
stdio (`serve`) that external clients like Claude Desktop or Cursor can plug
into. Both modes give the exact same five tools.
[/lang]
[lang:de]
**Die kleinste Demo der stärksten Idee: eine KI in der Sandbox, deren ganzes
Universum Pipes eigener MCP-Server ist. Kein freies Netz, kein freies
Dateisystem, keine Shell — nur fünf Tools hinter einer Whitelist.**

Jeder fragt bei Sandboxen dasselbe: *"Aber wenn der Agent Werkzeuge hat, ist
der Werkzeugsatz nicht die Angriffsfläche?"*

Fair. Also habe ich es diesmal umgedreht. Statt dem Agenten einen Werkzeugkasten
zu geben und zu hoffen, dass die Sandbox hält, habe ich ein Profil gebaut, in
dem der Agent **ausschließlich** die MCP-Tools nutzen kann, die *ich* freigebe —
und sonst nichts. Die ganze Welt der KI ist ein einziger MCP-Server, und dieser
Server lebt ebenfalls in der Sandbox.

Das Beispiel liegt in [`examples/mcp_sandbox_agent.pipe`](https://github.com/MachuraHarry/pipe/blob/master/examples/mcp_sandbox_agent.pipe).
Es hat zwei Modi: einen In-Process-Agenten (`agent`) und einen echten MCP-Server
auf stdio (`serve`), in den sich externe Clients wie Claude Desktop oder Cursor
einklinken. Beide Modi geben exakt dieselben fünf Tools.
[/lang]

---

[lang:en]
## The Cell 🏗️
[/lang]
[lang:de]
## Die Zelle 🏗️
[/lang]

[lang:en]
The profile is the wall:
[/lang]
[lang:de]
Das Profil ist die Wand:
[/lang]

```pipe
sandbox_profile "mcp-cell" {
    fs:                "temp-only",       # every write lands in ./.pipe_sandbox
    network:           true,
    network_whitelist: [provider_host],   # the ONLY network target is the AI provider
    exec:              false,             # no shell, period
    ai:                true,
    budget:            0.5,
    max_tool_calls:    25,
    audit_log:         true,
    timeout:           30
}
```

[lang:en]
Inside that wall the agent can call exactly five tools, defined with ordinary
Pipe functions and registered with `ai_tool`:
[/lang]
[lang:de]
Innerhalb dieser Wand kann der Agent genau fünf Tools aufrufen, definiert als
ganz normale Pipe-Funktionen und registriert mit `ai_tool`:
[/lang]

| Tool | [lang:en]What it does[/lang][lang:de]Was es tut[/lang] |
|------|--------------|
| `sb_write` | [lang:en]Writes a file — redirected into `.pipe_sandbox`[/lang][lang:de]Schreibt eine Datei — umgeleitet nach `.pipe_sandbox`[/lang] |
| `sb_read` | [lang:en]Reads a file from the sandbox[/lang][lang:de]Liest eine Datei aus der Sandbox[/lang] |
| `sb_list` | [lang:en]Lists the sandbox directory[/lang][lang:de]Listet das Sandbox-Verzeichnis[/lang] |
| `sb_note` | [lang:en]Stores an in-memory note (a plain Pipe map)[/lang][lang:de]Legt eine In-Memory-Notiz an (eine Pipe-Map)[/lang] |
| `sb_ping` | [lang:en]Liveness check[/lang][lang:de]Lebenszeichen-Check[/lang] |

```pipe
fn sb_note key value
    set __cell_notes key value
    "Notiz gesetzt: " ++ key ++ " = " ++ value

ai_tool "sb_note" "Legt eine In-Memory-Notiz an" {key: "Schluessel", value: "Wert"} sb_note
```

[lang:en]
Then the sandbox is locked in with `set_sandbox "mcp-cell"`, and the server is
started. Because `mcp_server` bridges whatever `ai_tool` registered, the
client-facing tool list is *the same five functions* — and every single
`tools/call` executes under the active profile.
[/lang]
[lang:de]
Danach wird die Sandbox mit `set_sandbox "mcp-cell"` festgeschaltet und der
Server gestartet. Da `mcp_server` alles bridged, was `ai_tool` registriert hat,
ist die client-seitige Toolliste *dieselben fünf Funktionen* — und jeder
`tools/call` läuft unter dem aktiven Profil.
[/lang]

---

[lang:en]
## What the Agent Actually Did 🤖
[/lang]
[lang:de]
## Was der Agent wirklich tat 🤖
[/lang]

[lang:en]
In `agent` mode the task was simple: create three notes, write a summary file,
list the sandbox. DeepSeek did it in six tool calls:
[/lang]
[lang:de]
Im `agent`-Modus war die Aufgabe simpel: drei Notizen anlegen, eine
Zusammenfassung schreiben, die Sandbox auflisten. DeepSeek erledigte das in
sechs Tool-Calls:
[/lang]

```
tool_call | sb_note
tool_call | sb_note
tool_call | sb_note
tool_call | sb_write
tool_call | sb_list
```

[lang:en]
That's the whole audit log. No `exec`, no foreign `http_get`, no writes
outside the cell. The agent *couldn't* have done more even if it had tried —
the profile blocks it at the builtin level, not at the prompt level.
[/lang]
[lang:de]
Das ist das gesamte Audit-Log. Kein `exec`, kein fremder `http_get`, keine
Schreibzugriffe außerhalb der Zelle. Der Agent *hätte* nicht mehr tun können,
selbst wenn er es versucht hätte — das Profil blockt auf Builtin-Ebene, nicht
auf Prompt-Ebene.
[/lang]

---

[lang:en]
## The Real Proof: Trying to Break Out 💥
[/lang]
[lang:de]
## Der echte Beweis: Der Ausbruchsversuch 💥
[/lang]

[lang:en]
A demo that only shows success proves nothing. So I tried to escape, using the
same builtins an attacker would reach for inside the cell:
[/lang]
[lang:de]
Eine Demo, die nur Erfolg zeigt, beweist nichts. Also habe ich versucht
auszubrechen — mit genau den Builtins, nach denen ein Angreifer in der Zelle
greifen würde:
[/lang]

| [lang:en]Attempt[/lang][lang:de]Versuch[/lang] | [lang:en]Result[/lang][lang:de]Ergebnis[/lang] |
|---------|--------|
| `exec "id"` | `E_SANDBOX: exec blocked by profile 'mcp-cell'` |
| `http_get "https://www.google.com"` | `E_SANDBOX: network target not in whitelist` |
| `write_file "/tmp/escape.txt"` | [lang:en]silently redirected to `.pipe_sandbox/escape.txt`[/lang][lang:de]stillschweigend umgeleitet nach `.pipe_sandbox/escape.txt`[/lang] |

[lang:en]
The third one is the interesting case: temp-only **doesn't fail**, it
*redirects*. The agent writes wherever it wants and believes it wrote to
`/tmp/escape.txt` — the file actually lands inside the cell. The attacker gets
a consistent illusion, and the host stays clean.
[/lang]
[lang:de]
Der dritte Fall ist der interessante: temp-only **scheitert nicht**, es
*leitet um*. Der Agent schreibt, wohin er will, und glaubt, er habe nach
`/tmp/escape.txt` geschrieben — die Datei landet tatsächlich in der Zelle. Der
Angreifer bekommt eine konsistente Illusion, und der Host bleibt sauber.
[/lang]

---

[lang:en]
## One Real Bug Found Along the Way 🐛
[/lang]
[lang:de]
## Ein echter Bug, der dabei auffiel 🐛
[/lang]

[lang:en]
Building this demo surfaced an actual bug in Pipe. With `fs: "temp-only"` and
the default `workingDir: "."`, listing the sandbox directory broke:
[/lang]
[lang:de]
Beim Bauen dieser Demo fiel ein echter Bug in Pipe auf. Mit `fs: "temp-only"`
und dem Standard-`workingDir: "."` brach das Auflisten des Sandbox-Verzeichnisses:
[/lang]

```
list_dir: open /tmp/.pipe_sandbox/tmp: no such file or directory
```

[lang:en]
The `filepath.Rel` call in the redirect logic can't relate an absolute path
to a *relative* base — so `list_dir "."` resolved to the wrong place. The fix
was to make the profile's working directory absolute everywhere
([`sandbox.go`](https://github.com/MachuraHarry/pipe/blob/master/pkg/object/sandbox.go)):
[/lang]
[lang:de]
Der `filepath.Rel`-Aufruf in der Redirect-Logik kann einen absoluten Pfad
nicht auf eine *relative* Basis beziehen — deshalb landete `list_dir "."` an
der falschen Stelle. Der Fix: Das Arbeitsverzeichnis des Profils wird überall
absolut gemacht
([`sandbox.go`](https://github.com/MachuraHarry/pipe/blob/master/pkg/object/sandbox.go)):
[/lang]

```go
func currentDir() string {
    d, err := os.Getwd()
    if err != nil || d == "" {
        return "."
    }
    return d
}
```

[lang:en]
Now every profile starts with an absolute working directory, and temp-only
redirects behave correctly — including `list_dir "."`.
[/lang]
[lang:de]
Jetzt startet jedes Profil mit absolutem Arbeitsverzeichnis, und temp-only-
Redirects verhalten sich korrekt — inklusive `list_dir "."`.
[/lang]

---

[lang:en]
## Try It Yourself 🚀
[/lang]
[lang:de]
## Probier es selbst 🚀
[/lang]

```bash
# DeepSeek
DEEPSEEK_API_KEY=sk-... pipe examples/mcp_sandbox_agent.pipe agent

# local, no key needed
pipe examples/mcp_sandbox_agent.pipe serve ollama
```

[lang:en]
Then point Claude Desktop or Cursor at the stdio server:
[/lang]
[lang:de]
Danach Claude Desktop oder Cursor auf den stdio-Server zeigen lassen:
[/lang]

```json
{
  "mcpServers": {
    "pipe-cell": {
      "command": "pipe",
      "args": ["examples/mcp_sandbox_agent.pipe", "serve"]
    }
  }
}
```

[lang:en]
The same `mcp-cell` profile wraps both the in-process agent and every external
client call. One sandbox, two entry points, same guarantee.
[/lang]
[lang:de]
Dasselbe `mcp-cell`-Profil umschließt sowohl den In-Process-Agenten als auch
jeden externen Client-Aufruf. Eine Sandbox, zwei Einstiegspunkte, dieselbe
Garantie.
[/lang]
