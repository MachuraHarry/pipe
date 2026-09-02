package object

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
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

// TestAiSwarmParallelizesTwoIndependentToolCalls exercises the REAL,
// end-to-end wiring Feature 2 added — bAiSwarm -> runSwarm's batchExecutor
// closure -> ai.ChatSwarm -> runToolBatch -> the real toolRegistry entries
// registered via ai_tool -- not a hand-rolled fake, unlike the ai/swarm_test.go
// unit tests which drive ai.ChatSwarm directly. A model response with 2
// independent tool_calls (simulating two MCP-bridged tools, e.g. two
// werkzeug_<alias> calls in one round) must be dispatched concurrently, not
// one at a time.
func TestAiSwarmParallelizesTwoIndependentToolCalls(t *testing.T) {
	round := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		round++
		w.Header().Set("Content-Type", "application/json")
		if round == 1 {
			w.Write([]byte(`{"choices":[{"finish_reason":"tool_calls","message":{"tool_calls":[
				{"id":"call_1","type":"function","function":{"name":"e2e_tool_a","arguments":"{}"}},
				{"id":"call_2","type":"function","function":{"name":"e2e_tool_b","arguments":"{}"}}
			]}}]}`))
			return
		}
		w.Write([]byte(`{"choices":[{"finish_reason":"stop","message":{"content":"both done"}}]}`))
	}))
	defer srv.Close()
	openaiAt(t, srv)

	const sleepEach = 60 * time.Millisecond
	registerE2ETool := func(name string) func() {
		t.Helper()
		bi := &BuiltinInfo{Fn: func(args ...Object) Object {
			time.Sleep(sleepEach)
			return &String{Value: name + "-ok"}
		}}
		if r := bAiTool(&String{Value: name}, &String{Value: name}, NewMap(), bi); r.Type() == ERROR {
			t.Fatalf("ai_tool %q: %s", name, r.Inspect())
		}
		return func() { delete(toolRegistry, name) }
	}
	defer registerE2ETool("e2e_tool_a")()
	defer registerE2ETool("e2e_tool_b")()
	defer swarmAgentFor(t, "e2e_agent", "E2E", []string{"e2e_tool_a", "e2e_tool_b"}, nil)()

	start := time.Now()
	result := bAiSwarm(&String{Value: "do both"}, &String{Value: "e2e_agent"})
	elapsed := time.Since(start)

	if result.Type() == ERROR {
		t.Fatalf("ai_swarm: %s", result.Inspect())
	}
	s, ok := result.(*String)
	if !ok || s.Value != "both done" {
		t.Fatalf("ai_swarm result = %v, want %q", result, "both done")
	}
	if elapsed >= 2*sleepEach {
		t.Errorf("ai_swarm took %v end-to-end, want well under %v — the two tool calls should run concurrently through the real runSwarm/runToolBatch wiring, not sequentially", elapsed, 2*sleepEach)
	}
}

func TestAiSwarmUnknownEntryAgent(t *testing.T) {
	result := bAiSwarm(&String{Value: "hello"}, &String{Value: "does-not-exist"})
	if result.Type() != ERROR {
		t.Fatalf("expected an error for an unregistered entry agent, got %v", result)
	}
}
