# Red-Team-Sandbox-Audit — Runde 6

**Datum:** 2026-08-13
**Methode:** statische Egress-Inventur + Verifikation gegen das CLI-Flag-Set
**Stand Commit:** nach `f6879f8`

---

## Ergebnis: `--sandbox` ohne Profil ließ `http_post`/`tcp_connect`/`tcp_listen` durch — jetzt gegatet

Runde 6 hat eine Lücke im **CLI-Flag-Pfad** gefunden: Der Kommandozeilen-Sandbox-Modus
`pipe --sandbox <script>` (ohne `--sandbox-profile`) blockierte laut Hilfe „dangerous
builtins (exec, tcp, http, ai, fs-write)" — aber drei netzwerkfähige Builtins
**ignorierten das Netz-Verbot** und machten weiterhin echte Netzwerk-Egress
(HTTP-POST und TCP-Verbindungen).

Wichtig zur Einordnung: Der **AI-Provider-Egress** (zentraler Gate aus Runde 5) und
der **Profil-Pfad** (`strict`/`noexec`/`isolated`/custom mit `network: false`) waren
nicht betroffen — die Lücke lag ausschließlich im „`none`-Profil + Sandbox-Flags"-Pfad.

---

## Die Lücke — der `if Name != "none"`-Pfad vergaß die CLI-Flags

Das CLI-`--sandbox`-Flag setzt `Sandbox.Enabled = true` und lässt `ActiveProfile`
auf `"none"` (das „alles erlaubt"-Profil). Die Builtin-Gates folgen seit Runde 1
diesem Muster:

```go
if ActiveProfile.Load().Name != "none" {
    if canErr := ActiveProfile.Load().CanNetwork(); canErr != nil {
        return err(canErr.Error())
    }
} else if Sandbox.Enabled && !Sandbox.AllowNet {
    return sandboxBlock("... (network)")
}
```

Der zweite Zweig (`else if Sandbox …`) war in **drei** netzwerkfähigen Builtins
vergessen worden — sie prüften nur das Profil und ließen im `"none"`-Pfad alles
durch:

- `http_post` (`pkg/object/builtins_net.go`)
- `tcp_connect` (`pkg/object/builtins_net.go`)
- `tcp_listen` (`pkg/object/builtins_net.go`)

Abgedeckt waren hingegen `http_get`, `http_request`, `http_stream_open` und
`http_server`. Ein red-teamed Skript hätte unter `pipe --sandbox` weiterhin Daten
per HTTP-POST exfiltrieren oder TCP-Verbindungen aufbauen können — exakt die
Egress-Oberfläche, die der Sandbox-Modus verspricht zu unterbinden.

---

## Fix (Branch `network-gate-6`)

- **Zentraler Helper** `checkNetworkAccess(feature)` in
  `pkg/object/builtins_sandbox.go`: kapselt beide Zweige (Profil-`CanNetwork`
  **und** `Sandbox.Enabled && !Sandbox.AllowNet`) an **einer** Stelle.
- Alle **sieben** netzwerkfähigen Builtins nutzen ihn jetzt:
  `http_get`, `http_post`, `http_request`, `http_stream_open`, `http_server`,
  `tcp_connect`, `tcp_listen`. Die Whitelist-Checks (`CanNetworkTo`) bleiben
  zusätzlich bestehen (Defense-in-Depth).
- Damit ist die Fehlerklasse „vergessener `AllowNet`-Branch" strukturell nicht
  mehr reproduzierbar: Ein künftiges Builtin, das Netz-I/O macht, muss nur
  `checkNetworkAccess` aufrufen.

### Regressionstests (`pkg/object/network_gate_test.go`)

| Test | prüft |
|------|-------|
| `TestNetworkBuiltinsBlockedBySandboxFlag` | alle 7 Builtins blockiert unter `Sandbox.Enabled=true, AllowNet=false` (kein Netz-I/O, Block vor Dial/Request) |
| `TestNetworkBuiltinsAllowedBySandboxFlag` | `http_get`/`http_post` funktionieren gegen `httptest`-Server unter `AllowNet=true` |
| `TestNetworkBuiltinsBlockedByProfile` | blockiert unter Profil mit `network: false` |
| `TestNetworkBuiltinsAllowedByProfile` | `http_post` funktioniert unter Profil mit `network: true` |

---

## Live-Verifikation (CLI, keine echten Remote-Hosts)

```text
$ printf 'http_post "http://127.0.0.1:9/" "{}"' > net.pipe

$ timeout 20 pipe --sandbox net.pipe
Runtime error: net.pipe: SANDBOX: http_post (network) is disabled in sandbox mode
exit=1                                   # Block vor jedem Netzwerk-I/O ✓

$ timeout 20 pipe net.pipe               # Kontrolle ohne --sandbox:
Runtime error: http_post: Post ... dial tcp 127.0.0.1:9: connect: connection refused
exit=1                                   # erreicht den Dial → Block ist sandbox-spezifisch ✓
```

Der Block passiert **vor** jedem Netz-I/O (kein `net.Dial`/`client.Post`), wie die
Kontrollgruppe belegt, die bis zum `connection refused` kommt.

---

## Einordnung (ehrlich)

- Kein bekanntes Escape im **AI-Egress** (zentraler Gate, Runde 5) und im
  **Profil-Pfad** (Ratchet, Runde 2).
- Gefunden und gefixt: `--sandbox`-Flag-Pfad ließ `http_post`/`tcp_connect`/
  `tcp_listen` durch. Die Lücke ist durch Tests für beide Richtungen (blockiert
  und erlaubt) abgesichert.
- Verbleibende bekannte Limitationen: unverändert aus den Runden 4/5
  (globales `ActiveProfile`, Datei-I/O-Inventur, Layer-2 ohne Redirect-Re-Check).
