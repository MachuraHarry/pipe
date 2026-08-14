[lang:en]# Unique? Powerful? What Pipe Actually Does Differently[/lang]
[lang:de]# Einmalig? Mächtig? Was Pipe wirklich anders macht[/lang]

[lang:en]
**A sandbox that lives in the language, an MCP protocol that is a first-class
sandbox citizen, and an honest look at where it ends.**
[/lang]
[lang:de]
**Eine Sandbox, die in der Sprache lebt, ein MCP-Protokoll als Bürger erster
Klasse der Sandbox — und ein ehrlicher Blick darauf, wo das aufhört.**
[/lang]

---

[lang:en]
When people ask whether Pipe's sandbox + MCP combo is "unique and powerful,"
the honest answer is: the *isolation* isn't unique — but *where the control
lives* is. Here's the landscape, then the difference.
[/lang]
[lang:de]
Wenn man fragt, ob Pipes Sandbox+MCP-Kombination „einmalig und mächtig" ist,
ist die ehrliche Antwort: die *Isolation* ist es nicht — aber *wo die Kontrolle
lebt*, sehr wohl. Erst die Landschaft, dann der Unterschied.
[/lang]

---

[lang:en]
## The Landscape: Isolation Has a Maturity Ladder 🪜
[/lang]
[lang:de]
## Die Landschaft: Isolation hat eine Reifeskala 🪜
[/lang]

| [lang:en]Layer[/lang][lang:de]Ebene[/lang] | [lang:en]Example[/lang][lang:de]Beispiel[/lang] | [lang:en]Boundary[/lang][lang:de]Grenze[/lang] |
|---------|----------|----------|
| [lang:en]MicroVM[/lang][lang:de]MicroVM[/lang] | Firecracker, E2B, AWS Lambda | [lang:en]own kernel, hardware-enforced[/lang][lang:de]eigener Kernel, hardware-gesichert[/lang] |
| [lang:en]User-space kernel[/lang][lang:de]Userspace-Kernel[/lang] | gVisor, Modal | [lang:en]syscalls reimplemented, host kernel never sees them directly[/lang][lang:de]Syscalls neuimplementiert, Host-Kernel sieht sie nie direkt[/lang] |
| [lang:en]Container[/lang][lang:de]Container[/lang] | Docker | [lang:en]shared kernel — explicitly not a security boundary[/lang][lang:de]geteilter Kernel — ausdrücklich keine Sicherheitsgrenze[/lang] |
| [lang:en]Permission system[/lang][lang:de]Permission-System[/lang] | Claude Code | [lang:en]tool-level allow/ask/deny, OS sandbox only for Bash[/lang][lang:de]Tool-Ebene allow/ask/deny, OS-Sandbox nur für Bash[/lang] |

[lang:en]
OWASP's Top 10 for Agentic Applications (ASI05) is blunt: *software-only
sandboxing is insufficient; LLM-generated code must run isolated*. Fair — for
untrusted code, put a microVM under Pipe. But the question "where is the
control" isn't answered by any of these layers.
[/lang]
[lang:de]
Der OWASP-Top-10-Bericht für Agentic Applications (ASI05) ist unmissverständlich:
*Software-Sandboxing reicht nicht; LLM-generierter Code muss isoliert laufen*.
Berechtigt — für unvertrauten Code legt man eine MicroVM unter Pipe. Aber die
Frage „wo lebt die Kontrolle" beantwortet keine dieser Ebenen.
[/lang]

---

[lang:en]
## Where Pipe Differs: Control Lives in the Language 🗝️
[/lang]
[lang:de]
## Wo Pipe sich unterscheidet: Kontrolle lebt in der Sprache 🗝️
[/lang]

[lang:en]
Pipe's sandbox is a **declarative language construct**, not an external layer.
You write the agent *and* the profile in one program:
[/lang]
[lang:de]
Pipes Sandbox ist ein **deklaratives Sprachkonstrukt**, keine äußere Schicht.
Du schreibst Agent *und* Profil in einem Programm:
[/lang]

```pipe
sandbox_profile "cell" {fs: "temp-only", network: true, network_whitelist: ["api.deepseek.com"], exec: false, ai: true, budget: 0.5, max_tool_calls: 25, audit_log: true}
set_sandbox "cell"
```

[lang:en]
Three properties matter:
[/lang]
[lang:de]
Drei Eigenschaften zählen:
[/lang]

[lang:en]
1. **Ratchet semantics** — once a restricted profile is active, a script may only switch to profiles granting *same or fewer* rights (`IsSubsetOf`). Even the agent can't free itself.
2. **Structural gates** — a central egress gate and a central network helper check every network-capable builtin, so a future builtin can't silently bypass `--sandbox`. Bug classes become impossible, not just rarer.
3. **Budget + audit log** — `budget`, `max_tool_calls`, `timeout`, and a per-event audit trail are part of the profile, not bolted on.
[/lang]
[lang:de]
1. **Ratchet-Semantik** — ist erst ein restriktives Profil aktiv, darf ein Skript nur in Profile wechseln, die *gleich viele oder weniger* Rechte gewähren (`IsSubsetOf`). Nicht einmal der Agent kann sich selbst befreien.
2. **Strukturelle Gates** — ein zentraler Egress-Gate und ein zentraler Netz-Helper prüfen jedes netzfähige Builtin, damit ein künftiges Builtin `--sandbox` nicht stillschweigend umgeht. Fehlerklassen werden unmöglich, nicht nur seltener.
3. **Budget + Audit-Log** — `budget`, `max_tool_calls`, `timeout` und ein lückenloser Audit-Trail gehören zum Profil, sind nicht nachträglich aufgesetzt.
[/lang]

[lang:en]
## The Real Clou: MCP Is a Sandbox Citizen 🔌
[/lang]
[lang:de]
## Der eigentliche Clou: MCP ist Sandbox-Bürger 🔌
[/lang]

[lang:en]
Pipe is simultaneously an **MCP server and an MCP client**. Locally registered
tools (`ai_tool`) and remotely discovered tools (bridged with an `mcp0_` prefix)
run under the **same profile**. In Claude Code, by contrast, MCP servers run
*unconstrained on the host* — the permission system gates calls, but the server
process itself is outside the sandbox.
[/lang]
[lang:de]
Pipe ist zugleich **MCP-Server und MCP-Client**. Lokal registrierte Tools
(`ai_tool`) und entfernt entdeckte Tools (gebridged mit `mcp0_`-Präfix) laufen
unter **demselben Profil**. Bei Claude Code laufen MCP-Server dagegen
*uneingeschränkt auf dem Host* — das Permission-System sperrt Aufrufe, aber der
Serverprozess selbst liegt außerhalb der Sandbox.
[/lang]

[lang:en]
That single detail is the difference: an external client like Cursor or Claude
Desktop connects to a Pipe `serve` session, gets the same five sandboxed tools,
and every `tools/call` executes under the profile. One sandbox, two entry
points, same guarantee.
[/lang]
[lang:de]
Genau dieses Detail ist der Unterschied: Ein externer Client wie Cursor oder
Claude Desktop verbindet sich mit einer Pipe-`serve`-Session, bekommt dieselben
fünf sandboxed Tools, und jeder `tools/call` läuft unter dem Profil. Eine
Sandbox, zwei Einstiegspunkte, dieselbe Garantie.
[/lang]

---

[lang:en]
## The Honest Limits ⚠️
[/lang]
[lang:de]
## Die ehrlichen Grenzen ⚠️
[/lang]

[lang:en]
Pipe's sandbox is **software-level, inside the process** — no own kernel, no
hardware boundary. It's superb for orchestrating agents quickly and safely
without infrastructure; it's not a hardened boundary for adversarial code. No
TLS inspection, so domain fronting is theoretically possible. But because the
profile is a *script-level* construct, you can always run Pipe itself inside a
Docker container or microVM and keep both layers.
[/lang]
[lang:de]
Pipes Sandbox ist **auf Software-Ebene, im Prozess** — kein eigener Kernel, keine
Hardware-Grenze. Sie ist hervorragend, um Agenten schnell und sicher *ohne*
Infrastruktur zu orchestrieren; keine gehärtete Grenze für adversariellen Code.
Keine TLS-Inspektion, Domain-Fronting ist theoretisch möglich. Aber weil das
Profil ein *Skript-Ebenen*-Konstrukt ist, kannst du Pipe selbst jederzeit in
einen Docker-Container oder eine MicroVM legen und beide Ebenen behalten.
[/lang]

---

[lang:en]
## Try It 🚀
[/lang]
[lang:de]
## Probier es 🚀
[/lang]

[lang:en]
The demo shows the whole idea in one file:
[/lang]
[lang:de]
Die Demo zeigt die ganze Idee in einer Datei:
[/lang]

```bash
DEEPSEEK_API_KEY=sk-... pipe examples/mcp_sandbox_agent.pipe agent
```

[lang:en]
See the deep dive: **[The MCP Cell](mcp-cell.html)** — and the source at
[`examples/mcp_sandbox_agent.pipe`](https://github.com/MachuraHarry/pipe/blob/master/examples/mcp_sandbox_agent.pipe).
[/lang]
[lang:de]
Zum Eintauchen: **[Die MCP-Zelle](mcp-cell.html)** — und der Quelltext unter
[`examples/mcp_sandbox_agent.pipe`](https://github.com/MachuraHarry/pipe/blob/master/examples/mcp_sandbox_agent.pipe).
[/lang]
