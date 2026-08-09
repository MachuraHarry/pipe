# 25. MCP (Model Context Protocol)

> **STATUS: IN ENTWICKLUNG — NOCH NICHT im pipe-modules-Registry veröffentlicht.**
> E2E-verifiziert: MCP Server + Client mit JSON-RPC 2.0 über stdio.
> HTTP/SSE-Transport in Arbeit.

Pipe implementiert das **[Model Context Protocol (MCP)](https://modelcontextprotocol.io)** — sowohl als Server (um Tools für externe Clients wie Claude Desktop bereitzustellen) als auch als Client (um externe MCP-Server in `ai_with_tools` zu nutzen). Die Implementierung ist **dependency-frei** und verwendet nur Go's Standardbibliothek.

---

## 25.1 Konzepte

MCP verwendet **JSON-RPC 2.0** über zwei Transporte:

| Transport | Anwendungsfall |
|-----------|---------------|
| **stdio** | Subprocess-Pipe (Claude Desktop, CLI-Tools). Pipe liest von stdin, schreibt auf stdout. |
| **Streamable HTTP** | Netzwerkbasiert (POST + SSE). *Geplant, noch nicht implementiert.* |

Das Protokoll definiert drei Primitive:
- **Tools** — Funktionen, die das KI-Modell aufrufen kann (`tools/list`, `tools/call`)
- **Resources** — Daten, die dem Modell ausgesetzt werden (in Pipe noch nicht implementiert)
- **Prompts** — Wiederverwendbare Vorlagen (in Pipe noch nicht implementiert)

---

## 25.2 MCP-Server — Pipe-Tools bereitstellen

Pipe kann als MCP-Server agieren und alle über `ai_tool` registrierten Funktionen jedem MCP-kompatiblen Client bereitstellen.

### Builtins

| Builtin | Beschreibung |
|---------|--------------|
| `mcp_server(name, version)` | Erstellt einen MCP-Server. Bridgt automatisch alle `ai_tool`-Einträge als MCP-Tools. |
| `mcp_serve_stdio` | Startet den Server auf stdin/stdout (blockierend). Für Claude Desktop, Cursor usw. |
| `mcp_serve_sse(addr)` | *(Noch nicht implementiert)* Startet einen HTTP/SSE-Server. |
| `mcp_tools` | Listet alle registrierten Tools (lokal + remote). |

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

---

## 25.3 MCP-Client — Externe Tools nutzen

Pipe kann sich mit externen MCP-Servern verbinden und deren Tools in `ai_with_tools` verwenden.

### Builtins

| Builtin | Beschreibung |
|---------|--------------|
| `mcp_use_stdio(command, args...)` | Startet einen Subprocess und verbindet sich über stdio. Entdeckt Tools und registriert sie im Tool-Registry mit einem `mcp0_`-, `mcp1_`-, …-Präfix. |
| `mcp_use_sse(url)` | Verbindet sich per POST mit einem Streamable-HTTP-MCP-Server. |

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
| `pkg/mcp/types.go` | JSON-RPC-2.0- und MCP-Message-Typen |
| `pkg/mcp/server.go` | Server-Dispatch: `initialize`, `tools/list`, `tools/call` |
| `pkg/mcp/stdio.go` | stdio-Transport (`bufio.Scanner` + `fmt.Println`) |
| `pkg/mcp/client.go` | Client: `exec.Cmd` (stdio) oder `net/http` (HTTP), nebenläufigkeitssicher |
| `pkg/mcp/schema.go` | JSON-Schema-Konverter |
| `pkg/object/builtins_mcp.go` | 7 Pipe-Builtins + Brücke zum `ai_tool`-Registry |

---

## 25.6 Fehlerbehandlung

Alle Builtins liefern Pipe's standard `Ok`/`Err`-Results oder einfache Strings. MCP-Protokollfehler (unbekannte Tools, Parse-Fehler) werden als JSON-RPC-Fehlerantworten an den Client zurückgegeben.

```pipe
r: mcp_use_stdio "./my-server"
if (type_of r) == "ERROR"
    print "MCP-Verbindung fehlgeschlagen: " ++ (to_str r)
```
