# 23. X-Modul (Twitter)

> **STATUS: IN ENTWICKLUNG — NOCH NICHT im pipe-modules-Registry veröffentlicht.**
> Live-verifiziert: PKCE-Auth, `x_token_from_code`, `users/me`.
> Credit-basierte Endpoints (Tweet posten, Suche, Lookups) erfordern gekaufte
> Credits in der X Developer Console (Pay-per-Use seit Feb 2026).

Das **X-Modul** (`scripts/x.pipe`) ist ein reiner-Pipe-Client für die **X API v2** (Twitter). Es behandelt die OAuth-2.0-Flows (PKCE für den User-Context, Client-Credentials für App-only) und bietet komfortable Wrapper für gängige Endpoints — inklusive des **Filtered Stream** (SSE) für Echtzeit-Tweets.

Es basiert vollständig auf den Standard-Pipe-Builtins: `http_request`, `http_stream_*`, `sha256`, `base64url_encode`, `secure_random_bytes` und `url_encode`.

---

## 23.1 Einrichtung

```pipe
import "x.pipe" as x
```

Lege `x.pipe` neben dein Skript oder installiere es via `PIPE_PATH`:

```bash
export PIPE_PATH="$HOME/CODE/pipe/scripts"
```

Das Modul verwendet drei Endpoint-Variablen, die per Umgebungsvariablen überschrieben werden können (nützlich für Test/Staging):

| Variable | Standard |
|----------|----------|
| `X_API_BASE` | `https://api.x.com/2` |
| `X_TOKEN_URL` | `https://api.x.com/2/oauth2/token` |
| `X_AUTH_URL` | `https://x.com/i/oauth2/authorize` |

---

## 23.2 OAuth 2.0 — User-Context (PKCE)

Die API erlaubt Lesen und Schreiben. Für **User-Context**-Zugriff (Posten, Likes, `x_me`) verwendet X OAuth 2.0 mit PKCE.

### Schritt 1 — Authorize-URL bauen

```pipe
auth: x.x_auth_url "YOUR_CLIENT_ID" "https://yourapp.com/callback" "tweet.read tweet.write users.read"
print auth.url          -- diese URL im Browser öffnen
print auth.verifier     -- speichern; wird in Schritt 2 benötigt
```

`x_auth_url` liefert eine Map mit drei Schlüsseln:

- `url` — die Authorisierungs-URL (enthält `code_challenge`, `state`, Scopes)
- `verifier` — der generierte PKCE-`code_verifier` (43 Zeichen)
- `state` — der CSRF-State-Wert

### Schritt 2 — Code austauschen

Nach Zustimmung des Benutzers leitet X mit `?code=...` zu deiner `redirect_uri` weiter. Tausche den Code:

```pipe
tokens: x.x_token_from_code "YOUR_CLIENT_ID" "https://yourapp.com/callback" code auth.verifier
```

Bei Erfolg erhältst du `Ok` mit einer Map, die `access_token`, `refresh_token`, `expires_in` und `scope` enthält. Verwende `unwrap` für den Zugriff.

### Schritt 3 — Tokens erneuern

```pipe
fresh: x.x_token_refresh "YOUR_CLIENT_ID" refresh_token
```

---

## 23.3 OAuth 2.0 — App-Only (Lesen)

Für reinen Lesezugriff brauchst du nur ein Bearer-Token, das per Client-Credentials bezogen wird:

```pipe
app: x.x_app_token "YOUR_CLIENT_ID" "YOUR_CLIENT_SECRET"
bearer: get (unwrap app) "access_token"
```

---

## 23.4 Endpoints

Alle API-Funktionen nehmen das **Bearer-Token** als erstes Argument und liefern ein `Ok`/`Err`-**Result** (nutze `is_ok`, `is_err`, `unwrap`, `unwrap_or`).

| Funktion | Beschreibung |
|----------|--------------|
| `x_request bearer method path body?` | Generischer Aufruf; `body` ist ein JSON-String oder `nil` |
| `x_tweet bearer text` | Tweet posten |
| `x_delete_tweet bearer id` | Tweet löschen |
| `x_get_tweet bearer id` | Einen Tweet abrufen |
| `x_search bearer query max_results?` | Rezente Suche (`max_results` 10–100) |
| `x_user_by_username bearer username` | User-Lookup |
| `x_me bearer` | Aktueller User (benötigt User-Context-Token) |
| `x_like bearer user_id tweet_id` | Tweet liken |
| `x_unlike bearer user_id tweet_id` | Like entfernen |
| `x_stream_open bearer` | Filtered Stream öffnen (SSE-Handle) |

Beispiele:

```pipe
-- Tweet posten
tw: x.x_tweet bearer "Hello from Pipe!"
if is_ok tw
    print (get (get (unwrap tw) "data") "id")

-- Rezente Tweets von elonmusk suchen
res: x.x_search bearer "from:elonmusk lang:en" 10
if is_ok res
    for t in (get (unwrap res) "data")
        print (get t "text")
```

---

## 23.5 Filtered Stream (SSE)

`x_stream_open` liefert ein Stream-Handle. Zeilen mit `http_stream_read_line` lesen:

- Nicht-leerer String → eine SSE-Zeile (`event: ...`, `data: ...`)
- Leerer String → eine leere Trennzeile (SSE-Heartbeat)
- `nil` → Stream beendet

```pipe
-- Zuerst Filter-Regeln setzen (via x_request), dann:
h: x.x_stream_open bearer
while true
    line: http_stream_read_line h
    if line == nil
        break
    if (len line) > 0
        print line
http_stream_close h
```

---

## 23.6 Fehlerbehandlung

Das Modul liefert `Err` bei HTTP-Fehlern (Status ≥ 400), ungültigem JSON und fehlgeschlagenen Token-Austauschen:

```pipe
res: x.x_search bearer "lang:en" 10
if is_err res
    print "Suche fehlgeschlagen"
else
    print (to_str (get (unwrap res) "status"))
```

Der `Ok`/`Err`-**Result**-Typ ist von Builtin-`*Error`-Objekten zu unterscheiden. Im Modul wird `type_of` verwendet, um sie zu unterscheiden; von außen interagierst du nur mit `Ok`/`Err`.
