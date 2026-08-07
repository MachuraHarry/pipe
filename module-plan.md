# Module Plan: HTTP-Client + jpipe

## Stand
Siehe auch `pipe-modules` Repo: https://github.com/MachuraHarry/pipe-modules

## Phase 1: Go-Builtin `http_request`

**Datei:** `pkg/object/object.go`

```go
// http_request(method, url, headers?, body?) -> {status, headers, body}
func bHttpRequest(args ...Object) Object
```

| Parameter | Typ | Pflicht |
|-----------|-----|---------|
| `method` | String ("GET","POST","PUT","DELETE","PATCH","HEAD") | Ja |
| `url` | String | Ja |
| `headers` | Map (optional, `{}` wenn fehlt) | Nein |
| `body` | String (optional) | Nein |

Rückgabe: `Map{status: int, headers: map, body: string}`

**Aufgaben:**
1. `bHttpRequest` implementieren (~60 Zeilen)
2. In `Builtins`-Array registrieren (Zeile ~299)
3. In `pkg/analysis/builtins.go` für LSP dokumentieren
4. In `pkg/gen/builtins.go` für Generator registrieren (optional)

---

## Phase 2: pipe-http Modul

**Datei:** `pipe-modules/pipe-http/module.pipe` (~250 Zeilen)

**Exports:**
```
export fn req(method, url, headers, body)    -- core: wraps http_request
export fn get(url, headers)                  -- GET convenience
export fn post(url, body, headers)           -- POST
export fn put(url, body, headers)            -- PUT
export fn delete(url, headers)               -- DELETE
export fn get_json(url, headers)             -- GET + parse response JSON
export fn post_json(url, body, headers)      -- POST + parse response JSON
export fn auth_header(type, credentials)     -- auth header builder
```

Response-Format: `{status: int, headers: map, body: string}`

**Features:**
- Alle HTTP-Methoden mit Custom Headers
- Auth-Helper für Basic/Bearer/API-Key
- Pipeline-kompatibel mit `>`-Operator
- Error-Wrapping mit Status-Code

**Test:** Integration-Test mit öffentlicher API

---

## Phase 3: jpipe Modul

**Datei:** `pipe-modules/jpipe/module.pipe` (~800 Zeilen)

### v1 — Path-Navigation
```
export fn jp(obj, path)               -- path evaluation
export fn jpick(obj, keys)            -- field picking
export fn jkeys(obj)                  -- all leaf paths
```

Path-Syntax: `.users[0].name`, `.items[*].id`

### v2 — Filter & Query
```
export fn jq(obj, query)              -- full query with filter
export fn jflatten(obj)               -- flatten nested
export fn jgroup(list, path)          -- group by key
```

Query-Syntax: `users[] | select(.active)`, `items[].price | map(* 1.19)`

**Architektur:**
```
Path-String ".users[0].name"
        │
        ▼
  Path Lexer     — zeichenweise Tokenisierung (~50 Zeilen)
        │
        ▼
  Path Parser    — recursive descent (~100 Zeilen)
        │
        ▼
  Path Evaluator — Object-Traversal (~150 Zeilen)
        │
        ▼
  Ergebnis (Map/List/Wert)
```

---

## Phase 4: Integration + Dokumentation

1. Demo: `examples/http_jpipe_demo.pipe` — API call → JSON query → SQLite
2. `docs/en/10-builtin-reference.md` — `http_request` Builtin dokumentieren
3. `docs/de/10-builtin-referenz.md` — dito
4. `pipe-modules` README updaten
5. `registry.json` updaten

---

## Geschätzte Zeilen

| Komponente | Zeilen |
|-----------|--------|
| `bHttpRequest` (Go) | ~60 |
| Registrierung + Tests (Go) | ~50 |
| `pipe-http/module.pipe` | ~250 |
| `jpipe/module.pipe` | ~800 |
| Tests | ~200 |
| Doku | ~100 |
| **Total** | **~1460** |
