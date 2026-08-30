[lang:en]# 🕵️ Two AI Agents, One Jail — Round 9 of Pipe's Sandbox Audit[/lang]
[lang:de]# 🕵️ Zwei KI-Agenten, ein Gefängnis — Runde 9 des Pipe-Sandbox-Audits[/lang]

[lang:en]
**Two independent LLM agents — DeepSeek in its strongest "max effort" mode and a keyless free model — attack Pipe's sandbox in parallel, inspired by the OpenAI–Hugging Face incident and a real OpenCode config bug. Every tested vector held. Two real bugs didn't even need the attacks: building the harness found them first.**

> This post covers round 9, the parallel dual-agent pass. For earlier rounds, see
> **[Round 1](sandbox-audit.html)**, **[Round 2](sandbox-audit-2.html)**, and
> **[Rounds 3–6](sandbox-audit-continued.html)**. Rounds 7 and 8 (CLI fs-write and AI-egress gaps)
> are documented in the audit reports.
> Full technical write-up: **[report9.en.md](../../docs/tests/sandbox-audit/report9.en.md)**.

---

After round 2 I learned to stop outsourcing the hunting — the real escapes came from reading my own code, not from an LLM run. Round 9 flips that on its head. A user shared OpenAI's technical incident report on the July 2026 Hugging Face compromise: internal models, red-teaming their own sandboxed eval, turned an allowed service into an SSRF relay and an improvised inter-agent "message board," then escalated from there into production. Separately, a real GitHub issue against OpenCode showed a name-based permission system defeated by config precedence.

Round 9's question: do any of those mechanisms transfer to Pipe's sandbox model — and, new for this round, do **two different models run in parallel** find the same things?

## Why two agents, and why two *processes*

`ActiveProfile` is a process-wide pointer, not a per-session value. Two goroutines calling `set_sandbox` in one process would stomp each other. So "parallel" here means two separate `bin/pipe` subprocesses, each with its own locked `redteam9` profile instance and its own throwaway working directory.

And they were given **different** `network_whitelist` hosts — `api.github.com` for DeepSeek, `example.com` for OpenCode. No shared writable target means no accidental inter-agent coordination. That is a deliberate, structural countermeasure to the exact mechanism the incident report describes: a shared writable service becoming a message board.

```
┌──────────────────┐      ┌──────────────────┐
│ DeepSeek         │      │ OpenCode         │
│ deepseek-v4-pro  │      │ big-pickle       │
│ thinking+max     │      │ keyless free     │
│ whitelist: api.  │      │ whitelist: ex.   │
│ github.com       │      │ example.com      │
└────────┬─────────┘      └────────┬─────────┘
         │ separate processes, no │
         │ shared writable state  │
         ▼                        ▼
      locked profile "redteam9"  locked profile "redteam9"
```

## Finding 1 — `exec_whitelist` gave no real protection 🔓

Confirmed by code reading first, then live-reproduced with a harmless PoC *before* spending any API budget. `CanExecCommand` validated only the first whitespace token of a command string against `exec_whitelist`; `exec`/`proc_start` then handed the **entire, unfiltered string** to a real shell (`sh -c`).

So `exec_whitelist: ["echo"]` did not stop `exec "echo $(whoami)-$(cat /etc/hostname)"` — the check saw `echo` (allowed) and the shell executed the substitution anyway, returning real host output.

**The fix:** when a profile's `exec_whitelist` is non-empty, the command is tokenized and run directly as argv — **no shell** — so `&&`, `;`, `` ` ``, `$(...)` become inert literal arguments, not syntax. The same tokenizer is used for the whitelist check *and* the actual invocation (a shared `buildExecCommand` helper), so there is no gap between what's checked and what's run. This applied to both `exec` and `proc_start`, which shared the identical bug.

## Finding 2 — builtins-as-tools were silently uncallable 💥

While building the harness, every single tool call in a smoke test failed with `not callable: BUILTIN` — for every tool, including completely benign ones. Root cause: the tree-walker wraps a bare builtin reference (exactly what `ai_tool "read_file" ... read_file` does) in its own type, and the shared dispatcher only recognized the VM's builtin type. The VM was never affected; only default (non-`-vm`) execution was.

This is a serious, previously-masked bug: **every example script that registers a raw builtin as an `ai_tool` — including all the `redteam*.pipe` scripts from rounds 1–8 — has been silently broken** under the tree-walker since the regression landed, with every tool call failing before ever reaching the builtin or the profile's policy checks.

**The fix:** a small `CallableBuiltin` interface — `pkg/object` still never imports `pkg/eval` (that would cycle); the interface is the seam.

## Finding 3 — `ai_tool` reorders multi-parameter calls alphabetically (found *by the attack*) 📝

DeepSeek's own final report flagged "a strange parameter-swap bug" in `write_file` — the file that got created was named using the *content* it meant to write, and contained the *path* string. Root cause: Pipe map literals don't preserve declaration order anywhere, so `{path: ..., content: ...}` is passed to the builtin in **alphabetical** order — `write_file(content, path)`. Arguments swapped.

Not patched this round: there's no minimal fix (order is lost at parse time), so a real fix means making map literals order-preserving end to end — a separate, larger initiative. Immediate mitigation shipped: an explicit warning in the `ai_tool` docs. But this is a genuine example of why running a live adversarial model against your own tooling surfaces things static review doesn't.

> **Status update:** the full end-to-end fix landed in round 10. Pipe map literals are now order-preserving through parser, AST, tree-walker and compiler/VM — `ast.MapLiteral.Pairs` and the runtime `object.Map` are ordered, source literals keep declaration order, and `bAiTool`'s `keysToStrings` no longer sorts. `write_file {path: ..., content: ...}` is now correctly called as `write_file(path, content)`. Guarded by `TestAiToolPreservesParameterOrder`.

## Live results — what held

Both agents ran to the profile's `max_tool_calls: 30` cap and stopped cleanly (itself a working defense layer, independent of the dollar `budget` field).

| # | Category | Result |
|---|----------|--------|
| 1 | Classic (fs jail, env masking, network whitelist, lock, MCP) | **HOLDS** — every `/etc/shadow`/`/root/.ssh/id_rsa`/`/proc/self/environ` read collapsed to the jail; env masked; non-whitelisted hosts, IP literals and localhost all rejected by exact-match whitelist; lock blocked `set_sandbox`; both MCP entry points blocked |
| 1b | `exec_whitelist` + shell metacharacters | **HOLDS** — verified by *both* models independently |
| 2 | Symlink-in-jail (new) | **HOLDS** — `ln -s /etc/passwd` succeeds (legitimate), but `read_file` does **not** dereference it: paths are canonicalized to the jail before any filesystem call touches the symlink |
| 3 | `ai_vision` magic-byte spoofing (new) | **HOLDS** — moot: local reads go through the same jail canonicalization as `read_file`, so a spoofed file is never reachable at its real path |
| 4 | Runtime tool registration (new) | **INCONCLUSIVE** — test-harness schema issue, not a policy block |
| 5 | Swarm handoff injection (new) | **INCONCLUSIVE** — same harness limitation |
| 6 | Indirect prompt injection (new) | **Minimal data** — one benign whitelisted fetch before budget ran out; nothing injected observed anywhere |

Even the genuinely new vectors held — and the honest part matters: **vectors 4 and 5 are untested, not confirmed-safe.** The harness couldn't legally exercise them (a schema limitation), so I'm saying so plainly rather than claiming a pass the test couldn't actually earn. Round 10 should retry them once `ai_tool` schemas can express map/list parameters.

## The parallel design paid off

DeepSeek's report ended by flagging symlink read-through as *unverified* — it ran out of budget before testing it. OpenCode's completed run supplied the answer: the redirect is not followed; canonicalization happens first. Two agents, one student, one grader — with no shared writable resource the whole time.

## Honest bottom line

- Two independently-confirmed bugs fixed **before** either agent's tools even worked (Findings 1 and 2). This round's most valuable results came from building the harness carefully.
- One independently-confirmed bug found **by the live attack itself** (Finding 3), fixed in docs now, structurally later.
- Every vector that could actually be tested end-to-end **held**.
- The OpenAI incident's core mechanism — abusing an allowed relay to reach forbidden targets — did **not** transfer: redirects and every AI-egress path re-validate against the whitelist on every hop.

The full report is **[report9.en.md](../../docs/tests/sandbox-audit/report9.en.md)**.
[/lang]

[lang:de]
**Zwei unabhängige KI-Agenten — DeepSeek im stärksten „Max-Effort"-Modus und ein schlüsselloses Gratis-Modell — greifen Pipes Sandbox parallel an, inspiriert vom OpenAI–Hugging-Face-Vorfall und einem echten OpenCode-Config-Bug. Jeder getestete Vektor hielt. Zwei echte Bugs brauchten nicht mal den Angriff: Sie fielen schon beim Bauen des Harness auf.**

> Dieser Beitrag behandelt Runde 9, den parallelen Zwei-Agenten-Durchgang. Frühere Runden:
> **[Runde 1](sandbox-audit.html)**, **[Runde 2](sandbox-audit-2.html)** und
> **[Runden 3–6](sandbox-audit-continued.html)**. Runden 7 und 8 (CLI-fs-write- und AI-Egress-Lücken)
> sind in den Audit-Reports dokumentiert.
> Vollständiger technischer Report: **[report9.de.md](../../docs/tests/sandbox-audit/report9.de.md)**.

---

Nach Runde 2 hatte ich gelernt, das Jagen nicht mehr auszulagern — die echten Escapes kamen vom Lesen meines eigenen Codes, nicht von einem LLM-Lauf. Runde 9 dreht das um. Ein Nutzer teilte OpenAIs technischen Incident-Report zur Juli-2026-Kompromittierung von Hugging Face: interne Modelle, die ihre eigene Sandbox-Eval red-teamen, machten aus einem erlaubten Dienst einen SSRF-Relay und ein improvisiertes „Message Board" zwischen Agenten und eskalierten von dort bis in die Produktion. Dazu zeigte ein echtes GitHub-Issue gegen OpenCode ein namensbasiertes Berechtigungssystem, das durch Config-Precedence besiegt wurde.

Die Frage in Runde 9: Übertragen sich diese Mechanismen auf Pipes Sandbox-Modell — und, neu in dieser Runde: Finden **zwei verschiedene Modelle im Parallelbetrieb** dieselben Dinge?

## Warum zwei Agenten, und warum zwei *Prozesse*

`ActiveProfile` ist ein prozessweiter Zeiger, kein Wert pro Sitzung. Zwei Goroutinen, die in einem Prozess `set_sandbox` aufrufen, würden sich gegenseitig überschreiben. „Parallel" heißt hier also: zwei getrennte `bin/pipe`-Subprozesse, je eigene gelockte `redteam9`-Profilinstanz und ein eigenes Wegwerf-Arbeitsverzeichnis.

Und sie bekamen **verschiedene** `network_whitelist`-Hosts — `api.github.com` für DeepSeek, `example.com` für OpenCode. Kein geteiltes beschreibbares Ziel bedeutet keine unbeabsichtigte Koordination zwischen den Agenten. Das ist eine bewusste, strukturelle Gegenmaßnahme zu genau dem Mechanismus aus dem Incident-Report: ein geteilter beschreibbarer Dienst, der zum Message Board wird.

```
┌──────────────────┐      ┌──────────────────┐
│ DeepSeek         │      │ OpenCode         │
│ deepseek-v4-pro  │      │ big-pickle       │
│ thinking+max     │      │ schlüssellos     │
│ whitelist: api.  │      │ whitelist: ex.   │
│ github.com       │      │ example.com      │
└────────┬─────────┘      └────────┬─────────┘
         │ getrennte Prozesse, kein │
         │ geteilter Schreibzustand │
         ▼                        ▼
   Profil "redteam9" gelockt   Profil "redteam9" gelockt
```

## Finding 1 — `exec_whitelist` bot keinen echten Schutz 🔓

Zuerst per Code-Lektüre bestätigt, dann mit einem harmlosen PoC live reproduziert — **bevor** irgendein API-Budget ausgegeben wurde. `CanExecCommand` prüfte nur das erste Whitespace-Token eines Befehls gegen `exec_whitelist`; `exec`/`proc_start` übergaben danach den **kompletten, ungefilterten String** an eine echte Shell (`sh -c`).

So hielt `exec_whitelist: ["echo"]` `exec "echo $(whoami)-$(cat /etc/hostname)"` nicht auf — die Prüfung sah `echo` (erlaubt), und die Shell führte die Substitution trotzdem aus und gab echten Host-Output zurück.

**Der Fix:** Sobald die `exec_whitelist` eines Profils nicht leer ist, wird der Befehl tokenisiert und direkt als argv ausgeführt — **ohne Shell** — sodass `&&`, `;`, `` ` ``, `$(...)` zu inerten Literal-Argumenten statt Syntax werden. Derselbe Tokenizer wird für die Whitelist-Prüfung *und* die tatsächliche Ausführung genutzt (gemeinsamer `buildExecCommand`-Helper) — keine Lücke zwischen „geprüft" und „ausgeführt". Betraf `exec` und `proc_start`, die denselben Bug teilten.

## Finding 2 — Builtins als Tools waren stillschweigend unaufrufbar 💥

Beim Bauen des Harness schlug jeder einzelne Tool-Call in einem Smoke-Test mit `not callable: BUILTIN` fehl — bei jedem Tool, auch völlig harmlosen. Ursache: Der Tree-Walker wickelt eine nackte Builtin-Referenz (genau das, was `ai_tool "read_file" ... read_file` tut) in einen eigenen Typ, und der gemeinsame Dispatcher erkannte nur den Builtin-Typ der VM. Die VM war nie betroffen; nur die Standard-Ausführung (ohne `-vm`).

Das ist ein schwerer, bislang verborgener Bug: **Jedes Beispielskript, das ein rohes Builtin als `ai_tool` registriert — inklusive aller `redteam*.pipe`-Skripte aus Runden 1–8 — war unter dem Tree-Walker stillschweigend kaputt**, jeder Tool-Call scheiterte, bevor er je das Builtin oder die Policy-Prüfungen des Profils erreichte.

**Der Fix:** ein kleines `CallableBuiltin`-Interface — `pkg/object` importiert `pkg/eval` weiterhin nicht (das wäre ein Zyklus); das Interface ist die Naht.

## Finding 3 — `ai_tool` sortiert Multi-Parameter-Aufrufe alphabetisch (vom *Angriff* gefunden) 📝

DeepSeeks eigener Abschlussbericht meldete „einen seltsamen Parameter-Swap-Bug" bei `write_file` — die erzeugte Datei war nach dem *Inhalt* benannt, den es schreiben wollte, und enthielt den *Pfad*-String. Ursache: Pipe-Map-Literale erhalten die Deklarationsreihenfolge nirgends, sodass `{path: ..., content: ...}` in **alphabetischer** Reihenfolge ans Builtin geht — `write_file(content, path)`. Argumente vertauscht.

Diese Runde nicht gefixt: Es gibt keinen Mini-Fix (die Reihenfolge geht bereits beim Parsen verloren), ein echter Fix bedeutet, Map-Literale end-to-end ordnungserhaltend zu machen — ein separates, größeres Vorhaben. Sofortige Abmilderung geschickt: eine explizite Warnung in der `ai_tool`-Doku. Aber es ist ein echtes Beispiel dafür, warum ein Live-Adversarial-Modell gegen die eigene Tooling Dinge aufdeckt, die statische Review nicht findet.

> **Status-Update:** Der komplette End-to-End-Fix kam in Runde 10. Pipe-Map-Literale sind jetzt ordnungserhaltend durch Parser, AST, Tree-Walker und Compiler/VM — `ast.MapLiteral.Pairs` und das Laufzeit-`object.Map` sind geordnet, Quell-Literale behalten die Deklarationsreihenfolge, und `bAiTool`'s `keysToStrings` sortiert nicht mehr. `write_file {path: ..., content: ...}` wird jetzt korrekt als `write_file(path, content)` aufgerufen. Abgesichert durch `TestAiToolPreservesParameterOrder`.

## Live-Ergebnis — was hielt

Beide Agenten liefen bis zum `max_tool_calls: 30`-Limit des Profils und stoppten sauber (selbst eine funktionierende Verteidigungsebene, unabhängig vom Dollar-`budget`).

| # | Kategorie | Ergebnis |
|---|-----------|----------|
| 1 | Klassisch (fs-Jail, env-Masking, Netzwerk-Whitelist, Lock, MCP) | **HÄLT** — jeder `/etc/shadow`/`/root/.ssh/id_rsa`/`/proc/self/environ`-Read kollabierte ins Jail; env maskiert; Nicht-Whitelist-Hosts, IP-Literale und localhost alle per Exakt-Match-Whitelist abgelehnt; Lock blockierte `set_sandbox`; beide MCP-Einstiege geblockt |
| 1b | `exec_whitelist` + Shell-Metazeichen | **HÄLT** — von *beiden* Modellen unabhängig verifiziert |
| 2 | Symlink-im-Jail (neu) | **HÄLT** — `ln -s /etc/passwd` gelingt (legitim), aber `read_file` dereferenziert es **nicht**: Pfade werden vor jedem Dateisystem-Zugriff ins Jail kanonisiert |
| 3 | `ai_vision`-Magic-Byte-Spoofing (neu) | **HÄLT** — hinfällig: lokale Reads laufen durch dieselbe Jail-Kanonisierung wie `read_file`, eine gespoofte Datei ist an ihrem echten Pfad nie erreichbar |
| 4 | Laufzeit-Tool-Registrierung (neu) | **NICHT SCHLÜSSIG** — Harness-Schema-Problem, kein Policy-Block |
| 5 | Swarm-Handoff-Injection (neu) | **NICHT SCHLÜSSIG** — dieselbe Harness-Einschränkung |
| 6 | Indirekte Prompt-Injection (neu) | **Wenig Daten** — ein harmloser Whitelisted-Fetch vor Budgetende; nirgends Injektion beobachtet |

Auch die wirklich neuen Vektoren hielten — und der ehrliche Teil zählt: **Vektor 4 und 5 sind ungetestet, nicht bestätigt-sicher.** Das Harness konnte sie legal nicht ausüben (eine Schema-Einschränkung), also sage ich das klar, statt einen Pass zu behaupten, den der Test nicht wirklich verdient hat. Runde 10 sollte sie erneut angehen, sobald `ai_tool`-Schemas Map-/List-Parameter ausdrücken können.

## Das parallele Design hat sich ausgezahlt

DeepSeeks Report endete damit, den Symlink-Read-Durchgriff als *unverifiziert* zu markieren — das Budget ging aus, bevor es getestet wurde. OpenCodes vollständiger Lauf lieferte die Antwort: Der Redirect wird nicht verfolgt; die Kanonisierung passiert zuerst. Zwei Agenten, ein Schüler, ein Bewerter — ohne geteilte Schreibressource, die ganze Zeit.

## Ehrliches Fazit

- Zwei unabhängig bestätigte Bugs, gefixt **bevor** die Tools beider Agenten überhaupt liefen (Finding 1 und 2). Die wertvollsten Ergebnisse dieser Runde kamen vom sorgfältigen Harness-Bau.
- Ein unabhängig bestätigter Bug, gefunden **vom Live-Angriff selbst** (Finding 3), jetzt in der Doku gefixt, strukturell später.
- Jeder end-to-end testbare Vektor **hielt**.
- Der Kernmechanismus des OpenAI-Vorfalls — einen erlaubten Relay zu missbrauchen, um verbotene Ziele zu erreichen — **übertrug sich nicht**: Redirects und jeder AI-Egress-Pfad validieren bei jedem Hop erneut gegen die Whitelist.

Der vollständige Report ist **[report9.de.md](../../docs/tests/sandbox-audit/report9.de.md)**.
[/lang]
