package object

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func withSandboxAIFlags(enabled, allowAI bool) func() {
	prevEnabled, prevAllowAI := Sandbox.Enabled, Sandbox.AllowAI
	prevProfile := ActiveProfile.Load()
	Sandbox.Enabled, Sandbox.AllowAI = enabled, allowAI
	ActiveProfile.Store(profileRegistry["none"])
	return func() {
		Sandbox.Enabled, Sandbox.AllowAI = prevEnabled, prevAllowAI
		ActiveProfile.Store(prevProfile)
	}
}

// The CLI --sandbox flag keeps ActiveProfile at "none" and governs AI calls via
// Sandbox.AllowAI. ai_swarm must honor that flag before ai.ChatSwarm ever reaches
// the network — mirroring the round-6/7/8 audit discipline for this new builtin.
func TestAiSwarmBlockedBySandboxFlag(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Write([]byte(`{"choices":[{"message":{"content":"should never be reached"}}]}`))
	}))
	defer srv.Close()
	openaiAt(t, srv)

	defer swarmAgentFor(t, "solo", "SOLO", nil, nil)()
	defer withSandboxAIFlags(true, false)()

	result := bAiSwarm(&String{Value: "hello"}, &String{Value: "solo"})
	assertSandboxBlocked(t, "ai_swarm", result)

	if hits.Load() != 0 {
		t.Fatalf("ai_swarm reached the network under --sandbox: %d requests", hits.Load())
	}
}

func TestAiSwarmAllowedBySandboxFlag(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"choices":[{"finish_reason":"stop","message":{"content":"hi there"}}]}`))
	}))
	defer srv.Close()
	openaiAt(t, srv)

	defer swarmAgentFor(t, "solo", "SOLO", nil, nil)()
	defer withSandboxAIFlags(true, true)()

	result := bAiSwarm(&String{Value: "hello"}, &String{Value: "solo"})
	if result.Type() == ERROR {
		t.Fatalf("ai_swarm: expected success under AllowAI=true, got %s", result.Inspect())
	}
	s, ok := result.(*String)
	if !ok || s.Value != "hi there" {
		t.Fatalf("ai_swarm: unexpected result %v", result)
	}
}

// swarmAgentFor registers a swarm_agent for the duration of a test and returns a
// cleanup func that removes it, so tests don't leak entries into the global
// swarmRegistry.
func swarmAgentFor(t *testing.T, name, system string, tools, handoff []string) func() {
	t.Helper()
	toolElems := make([]Object, len(tools))
	for i, tn := range tools {
		toolElems[i] = &String{Value: tn}
	}
	handoffElems := make([]Object, len(handoff))
	for i, hn := range handoff {
		handoffElems[i] = &String{Value: hn}
	}
	config := MapFromGo(map[string]Object{
		"system":  &String{Value: system},
		"tools":   &List{Elements: toolElems},
		"handoff": &List{Elements: handoffElems},
	})
	if res := bSwarmAgent(&String{Value: name}, config); res.Type() == ERROR {
		t.Fatalf("swarm_agent %q: %s", name, res.Inspect())
	}
	return func() {
		swarmRegistryMu.Lock()
		delete(swarmRegistry, name)
		swarmRegistryMu.Unlock()
	}
}

func TestSwarmAgentRejectsMissingSystem(t *testing.T) {
	result := bSwarmAgent(&String{Value: "bad"}, MapFromGo(map[string]Object{}))
	if result.Type() != ERROR || !strings.Contains(result.(*Error).Message, "system") {
		t.Fatalf("expected an error about the missing 'system' key, got %v", result)
	}
}

func TestAiSwarmUnknownEntryAgent(t *testing.T) {
	result := bAiSwarm(&String{Value: "hello"}, &String{Value: "does-not-exist"})
	if result.Type() != ERROR {
		t.Fatalf("expected an error for an unregistered entry agent, got %v", result)
	}
}
