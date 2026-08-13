package object

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/MachuraHarry/pipe/pkg/ai"
)

func egressChatServer(hits *atomic.Int32) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"hello"}}]}`))
	}))
}

func openaiAt(t *testing.T, srv *httptest.Server) {
	t.Helper()
	prevCfg := ai.ActiveConfig
	t.Cleanup(func() { ai.ActiveConfig = prevCfg })
	ai.SetProvider("openai")
	ai.ActiveConfig.APIHost = srv.URL
	ai.SetAPIKey("OPENAI_API_KEY", "test-key")
}

func TestEgressGateBlocksChatWithoutBuiltin(t *testing.T) {
	var hits atomic.Int32
	srv := egressChatServer(&hits)
	defer srv.Close()
	openaiAt(t, srv)

	defer withProfile(testProfile("egress-noai", FSFull, true, false, false, nil))()

	_, err := ai.Chat(ai.ChatRequest{Messages: []ai.Message{
		{Role: "user", Content: "egress-gate-chat-blocked"},
	}})
	if err == nil || !strings.Contains(err.Error(), "E_SANDBOX") {
		t.Fatalf("expected E_SANDBOX from the central gate, got %v", err)
	}
	if hits.Load() != 0 {
		t.Fatalf("expected 0 requests, got %d", hits.Load())
	}
}

func TestEgressGateBlocksEmbedWithoutBuiltin(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":[{"embedding":[0.1,0.2,0.3]}]}`))
	}))
	defer srv.Close()
	openaiAt(t, srv)

	defer withProfile(testProfile("egress-noai", FSFull, true, false, false, nil))()

	_, err := ai.Embed("egress-gate-embed-blocked")
	if err == nil || !strings.Contains(err.Error(), "E_SANDBOX") {
		t.Fatalf("expected E_SANDBOX from the central gate, got %v", err)
	}
	if hits.Load() != 0 {
		t.Fatalf("expected 0 requests, got %d", hits.Load())
	}
}

func TestEgressGateBlocksCachedChatUnderAIFalse(t *testing.T) {
	var hits atomic.Int32
	srv := egressChatServer(&hits)
	defer srv.Close()
	openaiAt(t, srv)

	prompt := "egress-gate-cache-probe"
	req := ai.ChatRequest{Messages: []ai.Message{{Role: "user", Content: prompt}}}

	restore1 := withProfile(testProfile("egress-ai", FSFull, true, false, true, nil))
	defer restore1()

	resp, err := ai.Chat(req)
	if err != nil {
		t.Fatalf("priming call under ai:true failed: %v", err)
	}
	if resp.Content != "hello" || hits.Load() != 1 {
		t.Fatalf("priming call: content=%q hits=%d, want hello/1", resp.Content, hits.Load())
	}

	restore2 := withProfile(testProfile("egress-noai", FSFull, true, false, false, nil))
	defer restore2()

	if _, err := ai.Chat(req); err == nil || !strings.Contains(err.Error(), "E_SANDBOX") {
		t.Fatalf("expected cached chat blocked under ai:false, got %v", err)
	}
	if hits.Load() != 1 {
		t.Fatalf("cache hit must still be gated: hits=%d, want 1", hits.Load())
	}
}

func TestEgressGateAllowsChat(t *testing.T) {
	var hits atomic.Int32
	srv := egressChatServer(&hits)
	defer srv.Close()
	openaiAt(t, srv)

	defer withProfile(testProfile("egress-ai", FSFull, true, false, true, nil))()

	resp, err := ai.Chat(ai.ChatRequest{Messages: []ai.Message{
		{Role: "user", Content: "egress-gate-chat-allowed"},
	}})
	if err != nil {
		t.Fatalf("chat under ai:true failed: %v", err)
	}
	if resp.Content != "hello" || hits.Load() != 1 {
		t.Fatalf("content=%q hits=%d, want hello/1", resp.Content, hits.Load())
	}
}
