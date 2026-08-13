package object

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/MachuraHarry/pipe/pkg/ai"
)

func embeddingServer(hits *atomic.Int32) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":[{"embedding":[0.1,0.2,0.3]}]}`))
	}))
}

func TestEmbedBlockedByAIProfile(t *testing.T) {
	var hits atomic.Int32
	srv := embeddingServer(&hits)
	defer srv.Close()

	defer withProfile(testProfile("emb-noai", FSFull, true, false, false, nil))()

	res := bEmbed(&String{Value: "hello"})
	if res.Type() != ERROR || !strings.Contains(res.Inspect(), "E_SANDBOX") {
		t.Fatalf("expected E_SANDBOX error under ai:false, got %q", res.Inspect())
	}
	if hits.Load() != 0 {
		t.Fatalf("expected 0 AI requests, got %d", hits.Load())
	}
}

func TestEmbedBatchBlockedByAIProfile(t *testing.T) {
	var hits atomic.Int32
	srv := embeddingServer(&hits)
	defer srv.Close()

	defer withProfile(testProfile("embb-noai", FSFull, true, false, false, nil))()

	items := &List{Elements: []Object{&String{Value: "a"}, &String{Value: "b"}}}
	res := bEmbedBatch(items)
	if res.Type() != ERROR || !strings.Contains(res.Inspect(), "E_SANDBOX") {
		t.Fatalf("expected E_SANDBOX error under ai:false, got %q", res.Inspect())
	}
	if hits.Load() != 0 {
		t.Fatalf("expected 0 AI requests, got %d", hits.Load())
	}
}

func TestEmbedAllowedWithAI(t *testing.T) {
	var hits atomic.Int32
	srv := embeddingServer(&hits)
	defer srv.Close()

	prevCfg := ai.ActiveConfig
	defer func() { ai.ActiveConfig = prevCfg }()
	ai.ActiveConfig = ai.Config{Provider: "openai", Model: "text-embedding-3-small", APIHost: srv.URL}
	ai.SetAPIKey("OPENAI_API_KEY", "test-key")

	defer withProfile(testProfile("emb-ai", FSFull, true, false, true, nil))()

	res := bEmbed(&String{Value: "hello"})
	if res.Type() != LIST {
		t.Fatalf("expected a vector under ai:true, got %q", res.Inspect())
	}
	if hits.Load() != 1 {
		t.Fatalf("expected exactly 1 AI request, got %d", hits.Load())
	}
}

func TestImportURLBlockedByProfile(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Write([]byte("x"))
	}))
	defer srv.Close()

	defer withProfile(testProfile("imp-nonet", FSFull, false, false, true, nil))()

	_, _, err := ResolveImportFrom(srv.URL+"/mod.pipe", "")
	if err == nil || !strings.Contains(err.Error(), "E_SANDBOX") {
		t.Fatalf("expected E_SANDBOX error, got %v", err)
	}
	if hits.Load() != 0 {
		t.Fatalf("expected 0 network requests, got %d", hits.Load())
	}
}

func TestImportURLAllowedByWhitelist(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Write([]byte("42"))
	}))
	defer srv.Close()

	host := strings.TrimPrefix(srv.URL, "http://")
	whitelist := []string{host}
	defer withProfile(testProfile("imp-net", FSFull, true, false, true, whitelist))()

	_, content, err := ResolveImportFrom(srv.URL+"/mod.pipe", "")
	if err != nil {
		t.Fatalf("expected import to be allowed by whitelist, got %v", err)
	}
	if content != "42" {
		t.Fatalf("expected fetched content, got %q", content)
	}
	if hits.Load() != 1 {
		t.Fatalf("expected exactly 1 request, got %d", hits.Load())
	}
}
