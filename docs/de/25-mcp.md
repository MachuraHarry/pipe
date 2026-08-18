# 25. MCP (Model Context Protocol)

> **STATUS: Production-ready — eingebaut in Pipe v1.0.0.**
> E2E-verifiziert: MCP Server + Client mit JSON-RPC 2.0 über stdio UND Streamable HTTP (SSE).

Pipe implementiert das **[Model Context Protocol (MCP)](https://modelcontextprotocol.io)** — sowohl als Server (um Tools für externe Clients wie Claude Desktop bereitzustellen) als auch als Client (um externe MCP-Server in `ai_with_tools` zu nutzen). Die Implementierung ist **dependency-frei** und verwendet nur Go's Standardbibliothek.

---

## 25.1 Konzepte

MCP verwendet **JSON-RPC 2.0** über zwei Transporte:

| Transport | Anwendungsfall |
|-----------|---------------|
| **stdio** | Subprocess-Pipe (Claude Desktop, CLI-Tools). Pipe liest von stdin, schreibt auf stdout. |
| **Streamable HTTP** | Netzwerkbasiert (POST + SSE, session-verwaltet via `Mcp-Session-Id`). |

Das Protokoll definiert drei Primitive:
- **Tools** — Funktionen, die das KI-Modell aufrufen kann (`tools/list`, `tools/call`)
- **Resources** — Daten, die dem Modell ausgesetzt werden (`resources/list`, `resources/read`, inkl. URI-Templates)
- **Prompts** — Wiederverwendbare Nachrichtenvorlagen (`prompts/list`, `prompts/get`)

Der Server verhandelt während `initialize` die **Protokollversion** (neueste unterstützte: `2025-11-25`, abwärtskompatibel mit `2025-06-18`, `2025-03-26`, `2024-11-05`) und beantwortet `ping`-Keep-Alive-Anfragen.

---

## 25.2 MCP-Server — Pipe-Tools bereitstellen

Pipe kann als MCP-Server agieren und alle über `ai_tool` registrierten Funktionen jedem MCP-kompatiblen Client bereitstellen.

### Builtins

| Builtin | Beschreibung |
|---------|--------------|
| `mcp_server(name, version)` | Erstellt einen MCP-Server. Bridgt automatisch alle `ai_tool`-Einträge als MCP-Tools. |
| `mcp_serve_stdio` | Startet den Server auf stdin/stdout (blockierend). Für Claude Desktop, Cursor usw. |
| `mcp_serve_sse(addr)` | Startet einen Streamable-HTTP-Server auf `addr` (z. B. `:9090`, blockierend). Clients verbinden sich per `POST` + SSE. |
| `mcp_tools` | Listet alle registrierten Tools (lokal + remote). |
| `mcp_resource(uri, name, mime, read_fn)` | Registriert eine statische Resource. `read_fn(uri)` liefert den Resourcentext. |
| `mcp_resource_template(template, name, mime, read_fn)` | Registriert eine URI-Template-Resource, z. B. `file:///{path}`. `read_fn(uri)` wird mit der konkreten URI aufgerufen. |
| `mcp_prompt(name, description, args_map, build_fn)` | Registriert eine Prompt-Vorlage. `args_map` bildet Argumentnamen auf eine Beschreibung ab (oder auf eine Map mit `description`/`required`). `build_fn(args)` liefert den gerenderten Text. |
| `mcp_resources` | Listet alle Ressourcen (lokal + remote). |
| `mcp_read_resource(uri)` | Liest eine Resource vom lokalen Server oder einem verbundenen Client. |
| `mcp_prompts` | Listet alle Prompts (lokal + remote). |
| `mcp_prompt_get(name, args?)` | Rendert einen Prompt vom lokalen Server oder einem verbundenen Client. |

> **Hinweis:** Fehlende Pflicht-Argumente werden mit `isError: true` abgelehnt; Tool-Namen müssen dem Spec-Muster `^[a-zA-Z0-9_-]{1,64}$` entsprechen (ungültige Namen werden übersprungen). Tools, die eine Map/Liste zurückgeben, werden als pretty JSON serialisiert, damit strukturierte Daten den Roundtrip überleben.

### Beispiel: Claude-Desktop-Integration

```pipe
--- agent.pipe — Als MCP-Server für Claude Desktop ausführen

fn get_weather city
    match city
        | "Berlin" -> "Berlin: 22°C, sonnig"
        | "London" -> "London: 15°C, regnerisch"
        | _ -> city ++ ": keine Daten"

fn search_docs query
    "Ergebnisse für: " ++ query

ai_tool "get_weather" "Aktuelles Wetter für eine Stadt abrufen" {city: "Stadtname"} get_weather
ai_tool "search_docs" "Dokumentation durchsuchen" {query: "Suchbegriff"} search_docs

mcp_server "Pipe Agent" "1.0.0"
mcp_serve_stdio
```

Dann in `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "pipe-agent": {
      "command": "/tmp/pipe",
      "args": ["/path/to/agent.pipe"]
    }
  }
}
```

Claude Desktop entdeckt `get_weather` und `search_docs` automatisch als verfügbare Tools.

### E2E-verifizierter MCP-Ablauf

```
Client                  Pipe (MCP Server)
  │                         │
  ├─ initialize ──────────>│   → liefert Protokollversion, Capabilities
  ├─ notifications/init'ed─>│   (keine Antwort)
  ├─ tools/list ───────────>│   → liefert Tool-Schemas (JSON Schema)
  ├─ tools/call ───────────>│   → führt Pipe-Funktion aus, liefert Result
  │                         │
```

### Beispiel: Streamable-HTTP-Server

```pipe
--- http_agent.pipe — Tools über HTTP für Netzwerk-Clients bereitstellen

fn get_weather city
    match city
        | "Berlin" -> "Berlin: 22°C, sonnig"
        | _ -> city ++ ": keine Daten"

ai_tool "get_weather" "Aktuelles Wetter für eine Stadt abrufen" {city: "Stadtname"} get_weather

mcp_server "Pipe HTTP Agent" "1.0.0"
mcp_serve_sse ":9090"
```

Clients POSTen JSON-RPC-Nachrichten an `http://host:9090/` mit
`Accept: application/json, text/event-stream`. Der Server verwaltet Sessions über den
`Mcp-Session-Id`-Header; `DELETE` beendet eine Session.

### Ressourcen & Prompts

Ressourcen und Prompts werden mit den folgenden Builtins registriert und beim
Aufruf von `mcp_server` automatisch in den Server gebridged. Die Lese-/Render-
Builtins funktionieren auch ohne laufenden Server (Fallback auf die lokalen
Registries).

```pipe
--- resources.pipe — Ressourcen + Prompts über MCP bereitstellen

--- Statische Resource
fn read_docs uri
    "Dokumentation zu " ++ uri ++ "\n\n# Pipe\nDependency-freie Sprache."

--- URI-Template-Resource: jede file:///{path} matcht
fn read_tmp uri
    path: replace uri "file:///" ""
    content: read_file ("/tmp/" ++ path)
    content

mcp_resource "docs://pipe" "Pipe-Doku" "text/markdown" read_docs
mcp_resource_template "file:///{path}" "Datei" "text/plain" read_tmp

--- Prompt mit Pflicht-Argument
fn build_summary args
    "Bitte zusammenfassen: " ++ (get args "text")

mcp_prompt "summarize" "Fasse den gegebenen Text zusammen" {text: "Der zu fassende Text"} build_summary

--- Ohne laufenden Server prüfen (mcp_read_resource / mcp_prompt_get
--- funktionieren eigenständig; bei aktivem mcp_serve_stdio NICHT print()en,
--- da stdout das JSON-RPC-Protokoll trägt)
print (mcp_resources)
print (mcp_read_resource "file:///hello.txt")
print (mcp_prompt_get "summarize" {text: "Ein langer Artikel ..."})

mcp_server "Pipe Resources" "1.0.0"
mcp_serve_stdio
```

`args_map`-Einträge können einfache Strings (eine Beschreibung) oder Maps mit
`description` und optionalem `required`-Boolean (Standard `true`) sein. Remote-
Ressourcen/-Prompts, die über `mcp_use_*` entdeckt werden, erscheinen in
`mcp_resources` / `mcp_prompts` und können per `mcp_read_resource` /
`mcp_prompt_get` gelesen werden.

> Ein lauffähiges Beispiel mit Tools, Ressourcen und Prompts liegt unter
> `examples/mcp_resources.pipe` (direkt ausführbar oder als stdio-Server in
> Claude Desktop konfigurierbar). Weitere Kombinationen: `examples/mcp_server.pipe`,
> `examples/mcp_filesystem.pipe`, `examples/mcp_github.pipe`,
> `examples/mcp_combined.pipe`.

---

## 25.3 MCP-Client — Externe Tools nutzen

Pipe kann sich mit externen MCP-Servern verbinden und deren Tools in `ai_with_tools` verwenden.

### Builtins

| Builtin | Beschreibung |
|---------|--------------|
| `mcp_use_stdio(command, args..., env?, alias?)` | Startet einen Subprocess und verbindet sich über stdio. Entdeckt Tools und registriert sie im Tool-Registry mit einem `mcp0_`-, `mcp1_`-, …-Präfix, oder mit eigenem Präfix bei Angabe von `alias`. |
| `mcp_use_sse(url, alias?)` | Verbindet sich per POST + SSE mit einem Streamable-HTTP-MCP-Server (session-verwaltet), optional mit eigenem `alias`-Präfix. |

> **Sandbox:** MCP-Client-Verbindungen unterliegen dem aktiven
> Sandbox-Profil. `mcp_use_stdio` startet einen Subprocess und benötigt daher
> `exec: true`; `mcp_use_sse` führt HTTP-Anfragen aus und benötigt
> `network: true` — die Endpunkt-URL wird gegen die `network_whitelist` des
> Profils geprüft, ebenso jede weitere Anfrage (inkl. Redirect-Ziele). Unter
> dem Profil `none` greift kein Sandbox-Gate.

> **Hinweis:** Client-Calls haben ein konfigurierbares Timeout (Standard 120 s, via
> `client.SetCallTimeout`); Subprocess-stderr wird erfasst (letzte 64 KB) und in
> Fehlermeldungen einbezogen. Remote-Tool-Argumente unterstützen verschachtelte
> Maps/Listen (rekursiv nach JSON konvertiert).

Entdeckte Tools werden automatisch im Tool-Registry von Pipe registriert und stehen damit `ai_with_tools` neben lokal definierten Tools zur Verfügung.

### Beispiel: Einen Remote-MCP-Server nutzen

```pipe
ai_provider "deepseek"
ai_set_key "deepseek" (env "DEEPSEEK_API_KEY")

--- Verbindung zu einem externen MCP-Filesystem-Server
mcp_use_stdio "npx" "-y" "@modelcontextprotocol/server-filesystem" "/tmp"

--- Nun dessen Tools via ai_with_tools nutzen
result: ai_with_tools
    "Du verwaltest ein Dateisystem. Nutze verfügbare Tools, um Fragen zu beantworten."
    "Liste alle Dateien in /tmp auf und sage mir, welche am größten ist."
print result
```

### Tool-Namensgebung

Remote-Tools erhalten ein Präfix, um Namenskollisionen zwischen mehreren Servern zu vermeiden:

| Verbindung | Präfix | Beispiel |
|------------|--------|----------|
| 1. `mcp_use_stdio` | `mcp0_` | `mcp0_read_file` |
| 2. `mcp_use_stdio` | `mcp1_` | `mcp1_search` |
| 1. `mcp_use_sse` | `mcp2_` | `mcp2_query_db` |

Lokale `ai_tool`-Tools behalten ihre ursprünglichen Namen (kein Präfix).

Statt des automatischen Präfixes in Registrierungsreihenfolge kannst du einer Verbindung einen expliziten Alias geben, damit ihre Tools einen stabilen, aussagekräftigen Namen erhalten:

```pipe
mcp_use_stdio "npx" "-y" "@modelcontextprotocol/server-github" {GITHUB_TOKEN: (env "GITHUB_TOKEN")} "github"
mcp_use_sse "http://localhost:9090/mcp" "postgres"

--- github_list_issues, postgres_query, ...
```

Der Alias muss ein gültiger Identifier-Fragment sein und darf nicht mit dem Präfix eines anderen Clients kollidieren.

---

## 25.4 Kombiniert: Server + Client

Pipe kann als MCP-Hub dienen — eigene Tools bereitstellen UND Tools externer Server aggregieren:

```pipe
--- hub.pipe — Pipe als MCP-Aggregations-Hub

ai_provider "deepseek"

--- Eigene Tools
fn local_query q
    "Lokales Ergebnis für: " ++ q

ai_tool "local_query" "Lokale Wissensdatenbank durchsuchen" {q: "Query"} local_query

--- Verbindung zu externen MCP-Servern
mcp_use_stdio "node" "./filesystem-server.js"
mcp_use_stdio "python" "./database-server.py"

--- Alles (eigene + remote Tools) externen Clients bereitstellen
mcp_server "Pipe MCP Hub" "1.0.0"
mcp_serve_stdio
```

Claude Desktop sieht: `local_query`, `mcp0_read_file`, `mcp0_write_file`, `mcp1_query`, …

---

## 25.5 Architektur (Dependency-frei)

Die gesamte MCP-Funktionalität ist in reinem Go mit Standardbibliothek implementiert — keine externen Abhängigkeiten.

| Package | Verantwortlichkeit |
|---------|-------------------|
| `pkg/mcp/types.go` | JSON-RPC-2.0- und MCP-Message-Typen, Protokollversionen, Fehlercodes |
| `pkg/mcp/server.go` | Server-Dispatch: `initialize` (Versions-Negotiation), `ping`, `tools/list`, `tools/call`, `resources/list` + `resources/read`, `prompts/list` + `prompts/get` |
| `pkg/mcp/stdio.go` | stdio-Transport (`bufio.Scanner` + `fmt.Println`) |
| `pkg/mcp/http_server.go` | Streamable-HTTP-Server (POST + SSE, Session-Verwaltung, `DELETE`) |
| `pkg/mcp/client.go` | Client: stdio (`exec.Cmd`) + HTTP (SSE-Read-Loop, Session-Id), Call-Timeouts, stderr-Capture, nebenläufigkeitssicher |
| `pkg/mcp/schema.go` | JSON-Schema-Konverter |
| `pkg/object/builtins_mcp.go` | 13 Pipe-Builtins + Brücke zum `ai_tool`-Registry |

---

## 25.6 Fehlerbehandlung

Alle Builtins liefern Pipe's standard `Ok`/`Err`-Results oder einfache Strings. MCP-Protokollfehler (unbekannte Tools, Parse-Fehler) werden als JSON-RPC-Fehlerantworten an den Client zurückgegeben. Fehler auf Tool-Ebene (inkl. fehlender Pflicht-Argumente oder Remote-Tool-Fehler) werden als `isError: true`-Results zurückgegeben, damit Clients einen fehlgeschlagenen von einem erfolgreichen Call unterscheiden können.

```pipe
r: mcp_use_stdio "./my-server"
if (type_of r) == "ERROR"
    print "MCP-Verbindung fehlgeschlagen: " ++ (to_str r)
```
