# 23. X (Twitter) Module

> **STATUS: IN DEVELOPMENT — NOT yet published to the pipe-modules registry.**
> Live-verified: PKCE auth, `x_token_from_code`, `users/me`.
> Credit-based endpoints (posting tweets, search, lookups) require purchased
> credits in the X Developer Console (pay-per-use since Feb 2026).

The **X module** (`scripts/x.pipe`) is a pure-Pipe client for the **X API v2** (Twitter). It handles the OAuth 2.0 flows (PKCE for user context, client credentials for app-only) and provides convenience wrappers for common endpoints — including the **filtered stream** (SSE) for real-time tweets.

It is built entirely on the standard Pipe builtins: `http_request`, `http_stream_*`, `sha256`, `base64url_encode`, `secure_random_bytes`, and `url_encode`.

---

## 23.1 Setup

```pipe
import "x.pipe" as x
```

Place `x.pipe` next to your script, or install it via `PIPE_PATH`:

```bash
export PIPE_PATH="$HOME/CODE/pipe/scripts"
```

The module uses three endpoint variables, overridable via environment variables (useful for testing/staging):

| Variable | Default |
|----------|---------|
| `X_API_BASE` | `https://api.x.com/2` |
| `X_TOKEN_URL` | `https://api.x.com/2/oauth2/token` |
| `X_AUTH_URL` | `https://x.com/i/oauth2/authorize` |

---

## 23.2 OAuth 2.0 — User Context (PKCE)

The API lets you read and write. For **user-context** access (posting, likes, `x_me`), X uses OAuth 2.0 with PKCE.

### Step 1 — Build the authorize URL

```pipe
auth: x.x_auth_url "YOUR_CLIENT_ID" "https://yourapp.com/callback" "tweet.read tweet.write users.read"
print auth.url          -- open this URL in a browser
print auth.verifier     -- save this; you need it in step 2
```

`x_auth_url` returns a map with three keys:

- `url` — the authorization URL (contains `code_challenge`, `state`, scopes)
- `verifier` — the generated PKCE `code_verifier` (43 chars)
- `state` — CSRF state value

### Step 2 — Exchange the code

After the user approves, X redirects to your `redirect_uri` with `?code=...`. Exchange it:

```pipe
tokens: x.x_token_from_code "YOUR_CLIENT_ID" "https://yourapp.com/callback" code auth.verifier
```

On success you get `Ok` with a map containing `access_token`, `refresh_token`, `expires_in`, and `scope`. Use `unwrap` to access it.

### Step 3 — Refresh tokens

```pipe
fresh: x.x_token_refresh "YOUR_CLIENT_ID" refresh_token
```

---

## 23.3 OAuth 2.0 — App-Only (read)

For read-only access you only need a bearer token, obtained via client credentials:

```pipe
app: x.x_app_token "YOUR_CLIENT_ID" "YOUR_CLIENT_SECRET"
bearer: get (unwrap app) "access_token"
```

---

## 23.4 Endpoints

All API functions take the **bearer token** as the first argument and return an `Ok`/`Err` **Result** (use `is_ok`, `is_err`, `unwrap`, `unwrap_or`).

| Function | Description |
|----------|-------------|
| `x_request bearer method path body?` | Generic call; `body` is a JSON string or `nil` |
| `x_tweet bearer text` | Post a tweet |
| `x_delete_tweet bearer id` | Delete a tweet |
| `x_get_tweet bearer id` | Fetch one tweet |
| `x_search bearer query max_results?` | Recent search (`max_results` 10–100) |
| `x_user_by_username bearer username` | User lookup |
| `x_me bearer` | Current user (needs user-context token) |
| `x_like bearer user_id tweet_id` | Like a tweet |
| `x_unlike bearer user_id tweet_id` | Remove a like |
| `x_stream_open bearer` | Open the filtered stream (SSE handle) |

Examples:

```pipe
-- Post a tweet
tw: x.x_tweet bearer "Hello from Pipe!"
if is_ok tw
    print (get (get (unwrap tw) "data") "id")

-- Search recent tweets mentioning elonmusk
res: x.x_search bearer "from:elonmusk lang:en" 10
if is_ok res
    for t in (get (unwrap res) "data")
        print (get t "text")
```

---

## 23.5 Filtered Stream (SSE)

`x_stream_open` returns a stream handle. Read lines with `http_stream_read_line`:

- Non-empty string → an SSE line (`event: ...`, `data: ...`)
- Empty string → a blank separator line (SSE heartbeat)
- `nil` → stream ended

```pipe
-- Set up filter rules first (via x_request), then:
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

## 23.6 Error Handling

The module returns `Err` for HTTP errors (status ≥ 400), invalid JSON, and failed token exchanges:

```pipe
res: x.x_search bearer "lang:en" 10
if is_err res
    print "Search failed"
else
    print (to_str (get (unwrap res) "status"))
```

The `Ok`/`Err` **Result** type is distinct from builtin `*Error` objects. Inside the module, `type_of` is used to distinguish them; you only interact with `Ok`/`Err` from outside.
