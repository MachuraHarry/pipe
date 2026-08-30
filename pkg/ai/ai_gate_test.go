package ai

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestEgressGateBlocksAtEntry proves every egress entry point consults the
// central gate before any provider/network logic runs.
func TestEgressGateBlocksAtEntry(t *testing.T) {
	want := errors.New("E_SANDBOX: blocked")
	prevGate := egressGate
	SetEgressGate(func(info EgressInfo) error { return want })
	t.Cleanup(func() { SetEgressGate(prevGate) })

	req := ChatRequest{Messages: []Message{{Role: "user", Content: "x"}}}

	if _, err := Chat(req); !errors.Is(err, want) {
		t.Fatalf("Chat: got %v, want %v", err, want)
	}
	if err := Stream(req, func(string) error { return nil }); !errors.Is(err, want) {
		t.Fatalf("Stream: got %v, want %v", err, want)
	}
	if _, err := ChatWithTools("s", "u", nil, nil, 1); !errors.Is(err, want) {
		t.Fatalf("ChatWithTools: got %v, want %v", err, want)
	}
	if _, errs := ChatParallel([]ChatRequest{req}, 1); len(errs) != 1 || !errors.Is(errs[0], want) {
		t.Fatalf("ChatParallel: got %v, want %v", errs, want)
	}
	if _, err := Embed("x"); !errors.Is(err, want) {
		t.Fatalf("Embed: got %v, want %v", err, want)
	}
	if _, errs := EmbedBatch([]string{"x"}, 1); len(errs) != 1 || !errors.Is(errs[0], want) {
		t.Fatalf("EmbedBatch: got %v, want %v", errs, want)
	}
	if _, err := WebSearch("x"); !errors.Is(err, want) {
		t.Fatalf("WebSearch: got %v, want %v", err, want)
	}
	if _, err := WikiSearch("x"); !errors.Is(err, want) {
		t.Fatalf("WikiSearch: got %v, want %v", err, want)
	}
	swarmAgents := map[string]SwarmAgentSpec{"a": {SystemPrompt: "s"}}
	if _, err := ChatSwarm("a", swarmAgents, "u", nil, 1); !errors.Is(err, want) {
		t.Fatalf("ChatSwarm: got %v, want %v", err, want)
	}
	if _, err := VisionChat("describe", "https://example.invalid/x.jpg", 0); !errors.Is(err, want) {
		t.Fatalf("VisionChat: got %v, want %v", err, want)
	}
}

// TestEgressGateNoopByDefault ensures a nil gate permits all egress.
func TestEgressGateNoopByDefault(t *testing.T) {
	prevGate := egressGate
	SetEgressGate(nil)
	t.Cleanup(func() { SetEgressGate(prevGate) })

	if err := gateEgress(EgressChat, "https://example.invalid"); err != nil {
		t.Fatalf("expected nil gate to allow egress, got %v", err)
	}
}

// TestEgressGateRechecksRedirectHops proves provider HTTP clients re-run the
// central gate for every redirect hop, so a 30x to a target the profile blocks
// is refused instead of silently followed.
func TestEgressGateRechecksRedirectHops(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("redirect target must never be reached")
	}))
	defer target.Close()

	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	defer redirector.Close()

	prevGate := egressGate
	SetEgressGate(func(info EgressInfo) error {
		if info.URL == target.URL {
			return errors.New("E_SANDBOX: redirect target blocked")
		}
		return nil
	})
	t.Cleanup(func() { SetEgressGate(prevGate) })

	_, err := httpPostJSON(EgressChat, redirector.URL, "k", map[string]interface{}{"x": 1}, time.Second)
	if err == nil || !strings.Contains(err.Error(), "redirect target blocked") {
		t.Fatalf("expected redirect hop to be re-checked and blocked, got %v", err)
	}
}
