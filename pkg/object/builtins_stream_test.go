package object

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestURLEncoding(t *testing.T) {
	enc := bURLEncode(&String{Value: "from:elonmusk lang:en"})
	if enc.Type() == ERROR {
		t.Fatalf("url_encode error: %s", enc.Inspect())
	}
	if got := enc.(*String).Value; got != "from%3Aelonmusk%20lang%3Aen" {
		t.Errorf("url_encode = %q, want spaces as %%20", got)
	}

	dec := bURLDecode(&String{Value: "from%3Aelonmusk%20lang%3Aen"})
	if dec.Type() == ERROR {
		t.Fatalf("url_decode error: %s", dec.Inspect())
	}
	if got := dec.(*String).Value; got != "from:elonmusk lang:en" {
		t.Errorf("url_decode = %q", got)
	}
}

func TestBase64URL(t *testing.T) {
	enc := bBase64URLEncode(&String{Value: "hello"})
	if enc.Type() == ERROR {
		t.Fatalf("base64url_encode error: %s", enc.Inspect())
	}
	if got := enc.(*String).Value; got != "aGVsbG8" {
		t.Errorf("base64url_encode = %q, want 'aGVsbG8' (no padding)", got)
	}

	dec := bBase64URLDecode(&String{Value: "aGVsbG8"})
	if dec.Type() == ERROR {
		t.Fatalf("base64url_decode error: %s", dec.Inspect())
	}
	if got := dec.(*String).Value; got != "hello" {
		t.Errorf("base64url_decode = %q, want 'hello'", got)
	}
}

func TestBase64URLBytes(t *testing.T) {
	// PKCE: base64url of raw bytes (no padding)
	enc := bBase64URLEncode(&Bytes{Value: []byte("hello")})
	if enc.Type() == ERROR {
		t.Fatalf("base64url_encode(bytes) error: %s", enc.Inspect())
	}
	if got := enc.(*String).Value; got != "aGVsbG8" {
		t.Errorf("base64url_encode(bytes) = %q, want 'aGVsbG8'", got)
	}
}

func TestSha256Bytes(t *testing.T) {
	// RFC 7636 Appendix B PKCE vector:
	// verifier "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	// challenge "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"
	verifier := []byte("dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk")
	h := bSha256(&Bytes{Value: verifier})
	if h.Type() == ERROR {
		t.Fatalf("sha256(bytes) error: %s", h.Inspect())
	}
	b, ok := h.(*Bytes)
	if !ok {
		t.Fatalf("sha256(bytes) returned %s, want Bytes", h.Type())
	}
	challenge := bBase64URLEncode(b)
	if challenge.Type() == ERROR {
		t.Fatalf("base64url_encode error: %s", challenge.Inspect())
	}
	want := "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"
	if got := challenge.(*String).Value; got != want {
		t.Errorf("PKCE challenge = %q, want %q", got, want)
	}

	// String path unchanged: hex
	sh := bSha256(&String{Value: "hello"})
	if sh.Type() == ERROR {
		t.Fatalf("sha256(string) error: %s", sh.Inspect())
	}
	if got := sh.(*String).Value; got != "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824" {
		t.Errorf("sha256('hello') = %q", got)
	}
}

func TestHmacSha1(t *testing.T) {
	// RFC 2202 test vector 2
	got := bHmacSha1(&String{Value: "Jefe"}, &String{Value: "what do ya want for nothing?"})
	if got.Type() == ERROR {
		t.Fatalf("hmac_sha1 error: %s", got.Inspect())
	}
	want := "effcdf6ae5eb2fa2d27416d5f184df9c259a7c79"
	if got.(*String).Value != want {
		t.Errorf("hmac_sha1 = %q, want %q", got.(*String).Value, want)
	}
}

func TestHTTPStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("event: tweets/data\ndata: {\"id\":\"1\",\"text\":\"hello\"}\n\n"))
		w.Write([]byte("data: {\"id\":\"2\",\"text\":\"world\"}\n\n"))
	}))
	defer server.Close()

	headers := MapFromGo(map[string]Object{"Authorization": &String{Value: "Bearer test"}})
	h := bHttpStreamOpen(&String{Value: server.URL}, headers)
	if h.Type() == ERROR {
		t.Fatalf("http_stream_open error: %s", h.Inspect())
	}
	handle := h.(*Integer).Value

	var lines []string
	for {
		line := bHttpStreamReadLine(&Integer{Value: handle})
		if line.Type() == ERROR {
			t.Fatalf("http_stream_read_line error: %s", line.Inspect())
		}
		if line == NILOBJ {
			break
		}
		lines = append(lines, line.(*String).Value)
	}

	joined := strings.Join(lines, "\n")
	for _, want := range []string{"event: tweets/data", `data: {"id":"1","text":"hello"}`, `data: {"id":"2","text":"world"}`} {
		if !strings.Contains(joined, want) {
			t.Errorf("stream missing %q; got:\n%s", want, joined)
		}
	}

	closeResult := bHttpStreamClose(&Integer{Value: handle})
	if closeResult.Type() == ERROR {
		t.Fatalf("http_stream_close error: %s", closeResult.Inspect())
	}

	// Handle should now be invalid
	if r := bHttpStreamClose(&Integer{Value: handle}); r.Type() != ERROR {
		t.Error("expected error closing already-closed handle")
	}
}

func TestHTTPStreamReadChunk(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("abcdefghij"))
	}))
	defer server.Close()

	h := bHttpStreamOpen(&String{Value: server.URL})
	if h.Type() == ERROR {
		t.Fatalf("http_stream_open error: %s", h.Inspect())
	}
	handle := h.(*Integer).Value

	var total string
	for {
		chunk := bHttpStreamRead(&Integer{Value: handle})
		if chunk.Type() == ERROR {
			t.Fatalf("http_stream_read error: %s", chunk.Inspect())
		}
		if chunk == NILOBJ {
			break
		}
		total += chunk.(*String).Value
	}
	if total != "abcdefghij" {
		t.Errorf("http_stream_read total = %q, want 'abcdefghij'", total)
	}
	bHttpStreamClose(&Integer{Value: handle})
}

func TestHTTPStreamErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusBadRequest)
	}))
	defer server.Close()

	if r := bHttpStreamOpen(&String{Value: server.URL}); r.Type() != ERROR {
		t.Errorf("expected error for non-200, got %s", r.Inspect())
	}
	if r := bHttpStreamOpen(&String{Value: "not a url"}); r.Type() != ERROR {
		t.Errorf("expected error for bad url, got %s", r.Inspect())
	}
	if r := bHttpStreamRead(&Integer{Value: 9999}); r.Type() != ERROR {
		t.Errorf("expected error for invalid handle, got %s", r.Inspect())
	}
}
