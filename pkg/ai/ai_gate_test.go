package ai

import (
	"errors"
	"testing"
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
