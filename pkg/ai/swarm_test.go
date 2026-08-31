package ai

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// swarmSystemPrompt pulls the system message's content out of a raw chat-completions
// request body, so the mock server below can decide which agent is "speaking" this
// round without the test needing any state beyond what ChatSwarm itself sends.
func swarmSystemPrompt(t *testing.T, r *http.Request) string {
	t.Helper()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("reading request body: %v", err)
	}
	var decoded struct {
		Messages []map[string]interface{} `json:"messages"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("decoding request body: %v", err)
	}
	for _, m := range decoded.Messages {
		if m["role"] == "system" {
			s, _ := m["content"].(string)
			return s
		}
	}
	return ""
}

func writeChatContent(w http.ResponseWriter, content string) {
	json.NewEncoder(w).Encode(map[string]interface{}{
		"choices": []interface{}{
			map[string]interface{}{
				"finish_reason": "stop",
				"message":       map[string]interface{}{"content": content},
			},
		},
	})
}

func writeToolCall(w http.ResponseWriter, callID, toolName, argsJSON string) {
	json.NewEncoder(w).Encode(map[string]interface{}{
		"choices": []interface{}{
			map[string]interface{}{
				"finish_reason": "tool_calls",
				"message": map[string]interface{}{
					"tool_calls": []interface{}{
						map[string]interface{}{
							"id":   callID,
							"type": "function",
							"function": map[string]interface{}{
								"name":      toolName,
								"arguments": argsJSON,
							},
						},
					},
				},
			},
		},
	})
}

func withMockChatServer(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	prev := ActiveConfig
	t.Cleanup(func() { ActiveConfig = prev })
	ActiveConfig = Config{
		Provider: "ollama", // needs no API key env var
		Model:    "test-model",
		Timeout:  5 * time.Second,
		APIHost:  server.URL,
	}
}

func TestChatSwarmHandoffSwitchesAgent(t *testing.T) {
	round := 0
	withMockChatServer(t, func(w http.ResponseWriter, r *http.Request) {
		round++
		system := swarmSystemPrompt(t, r)
		switch {
		case strings.Contains(system, "TRIAGE"):
			writeToolCall(w, "call_1", swarmHandoffTool, `{"to":"billing"}`)
		case strings.Contains(system, "BILLING"):
			writeChatContent(w, "Your invoice is 49.90 EUR.")
		default:
			t.Fatalf("unexpected system prompt: %q", system)
		}
	})

	agents := map[string]SwarmAgentSpec{
		"triage":  {SystemPrompt: "TRIAGE: route billing questions to billing.", HandoffTo: []string{"billing"}},
		"billing": {SystemPrompt: "BILLING: answer invoice questions."},
	}

	result, err := ChatSwarm("triage", agents, "What do I owe?", nil, 5, nil)
	if err != nil {
		t.Fatalf("ChatSwarm: unexpected error: %v", err)
	}
	if result.Content != "Your invoice is 49.90 EUR." {
		t.Errorf("Content = %q, want the billing agent's answer", result.Content)
	}
	if want := []string{"triage", "billing"}; !equalStrSlices(result.Path, want) {
		t.Errorf("Path = %v, want %v", result.Path, want)
	}
	if result.Rounds != 2 {
		t.Errorf("Rounds = %d, want 2", result.Rounds)
	}
	if round != 2 {
		t.Errorf("server received %d requests, want 2", round)
	}
}

func TestChatSwarmToolCallUsesExecutor(t *testing.T) {
	round := 0
	withMockChatServer(t, func(w http.ResponseWriter, r *http.Request) {
		round++
		if round == 1 {
			writeToolCall(w, "call_1", "get_invoice", `{"customer":"acme"}`)
			return
		}
		writeChatContent(w, "Invoice lookup done.")
	})

	var executedName string
	var executedArgs map[string]interface{}
	executor := func(name string, args map[string]interface{}) (string, error) {
		executedName = name
		executedArgs = args
		return "invoice #4471", nil
	}

	agents := map[string]SwarmAgentSpec{
		"billing": {
			SystemPrompt: "BILLING",
			Tools:        []ToolDef{{Name: "get_invoice", Description: "look up an invoice"}},
		},
	}

	result, err := ChatSwarm("billing", agents, "What's on my invoice?", executor, 5, nil)
	if err != nil {
		t.Fatalf("ChatSwarm: unexpected error: %v", err)
	}
	if executedName != "get_invoice" {
		t.Errorf("executor received tool name %q, want get_invoice", executedName)
	}
	if executedArgs["customer"] != "acme" {
		t.Errorf("executor received args %v, want customer=acme", executedArgs)
	}
	if result.Content != "Invoice lookup done." {
		t.Errorf("Content = %q", result.Content)
	}
	if want := []string{"billing"}; !equalStrSlices(result.Path, want) {
		t.Errorf("Path = %v, want %v (no handoff happened)", result.Path, want)
	}
}

func TestChatSwarmInvokesProgressCallback(t *testing.T) {
	round := 0
	withMockChatServer(t, func(w http.ResponseWriter, r *http.Request) {
		round++
		switch round {
		case 1:
			writeToolCall(w, "call_1", swarmHandoffTool, `{"to":"billing"}`)
		case 2:
			writeToolCall(w, "call_2", "get_invoice", `{"customer":"acme"}`)
		default:
			writeChatContent(w, "Invoice lookup done.")
		}
	})

	agents := map[string]SwarmAgentSpec{
		"triage": {SystemPrompt: "TRIAGE", HandoffTo: []string{"billing"}},
		"billing": {
			SystemPrompt: "BILLING",
			Tools:        []ToolDef{{Name: "get_invoice", Description: "look up an invoice"}},
		},
	}
	executor := func(name string, args map[string]interface{}) (string, error) {
		return "invoice #4471", nil
	}

	type event struct{ agent, kind, detail, args string }
	var got []event
	onProgress := func(agent, kind, detail, argsJSON string) {
		got = append(got, event{agent, kind, detail, argsJSON})
	}

	result, err := ChatSwarm("triage", agents, "What's on my invoice?", executor, 5, onProgress)
	if err != nil {
		t.Fatalf("ChatSwarm: unexpected error: %v", err)
	}
	if result.Content != "Invoice lookup done." {
		t.Errorf("Content = %q", result.Content)
	}

	want := []event{
		{"triage", "start", "", ""},
		{"triage", "handoff", "billing", ""},
		{"billing", "start", "", ""},
		{"billing", "tool", "get_invoice", `{"customer":"acme"}`},
		{"billing", "start", "", ""},
		{"billing", "final", "", ""},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d progress events, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("event[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestChatSwarmRejectsDisallowedHandoffTarget(t *testing.T) {
	round := 0
	withMockChatServer(t, func(w http.ResponseWriter, r *http.Request) {
		round++
		if round == 1 {
			// "tech" is not in triage's HandoffTo — must be rejected without switching agents.
			writeToolCall(w, "call_1", swarmHandoffTool, `{"to":"tech"}`)
			return
		}
		system := swarmSystemPrompt(t, r)
		if !strings.Contains(system, "TRIAGE") {
			t.Fatalf("agent switched despite disallowed handoff target; system = %q", system)
		}
		writeChatContent(w, "I can only help with billing here.")
	})

	agents := map[string]SwarmAgentSpec{
		"triage":  {SystemPrompt: "TRIAGE", HandoffTo: []string{"billing"}},
		"billing": {SystemPrompt: "BILLING"},
		"tech":    {SystemPrompt: "TECH"},
	}

	result, err := ChatSwarm("triage", agents, "Reboot my router", nil, 5, nil)
	if err != nil {
		t.Fatalf("ChatSwarm: unexpected error: %v", err)
	}
	if want := []string{"triage"}; !equalStrSlices(result.Path, want) {
		t.Errorf("Path = %v, want %v (rejected handoff must not switch agents)", result.Path, want)
	}
}

func TestChatSwarmMaxRoundsExceeded(t *testing.T) {
	withMockChatServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeToolCall(w, "call_x", swarmHandoffTool, `{"to":"billing"}`)
	})

	agents := map[string]SwarmAgentSpec{
		"triage":  {SystemPrompt: "TRIAGE", HandoffTo: []string{"billing"}},
		"billing": {SystemPrompt: "BILLING", HandoffTo: []string{"triage"}},
	}

	_, err := ChatSwarm("triage", agents, "loop forever", nil, 2, nil)
	if err == nil || !strings.Contains(err.Error(), "max rounds") {
		t.Fatalf("expected a max-rounds error, got %v", err)
	}
}

// TestChatSwarmBreaksOscillationInsteadOfExhaustingRounds guards against a
// live-observed failure mode distinct from TestChatSwarmMaxRoundsExceeded's
// deliberate infinite loop: two agents that legitimately keep handing back
// to each other because each judges the other's output still incomplete
// (e.g. a critic sending work back to a fact-checker with nothing new to
// add, which sends it right back) — a real, valid handoff each time, not a
// rejected one, so nothing here is a swarm-config bug to fix at the prompt
// level. Without oscillation detection this burns every remaining round and
// returns no answer at all (reproduced live with kritiker/faktenwaechter,
// max_rounds=18, on a plain small-talk message). isOscillating should kick
// in on the second full A,B,A,B repeat and withdraw the handoff tool for
// that round, so the swarm returns SOME answer well before max rounds.
func TestChatSwarmBreaksOscillationInsteadOfExhaustingRounds(t *testing.T) {
	round := 0
	withMockChatServer(t, func(w http.ResponseWriter, r *http.Request) {
		round++
		if round <= 3 {
			// Rounds 1-3: kritiker<->faktenwaechter hand off to each other,
			// same as a real "still incomplete" judgment each time.
			to := "faktenwaechter"
			if round%2 == 0 {
				to = "kritiker"
			}
			writeToolCall(w, "call_"+string(rune('0'+round)), swarmHandoffTool, `{"to":"`+to+`"}`)
			return
		}
		// Round 4: the handoff tool should no longer be offered (oscillation
		// detected), so a well-behaved model just answers directly.
		system := swarmSystemPrompt(t, r)
		if !strings.Contains(system, "FAKTENWAECHTER") {
			t.Fatalf("expected round 4 to still be faktenwaechter's turn (forced to answer, not switched agent); system = %q", system)
		}
		writeChatContent(w, "Done despite the loop.")
	})

	agents := map[string]SwarmAgentSpec{
		"kritiker":       {SystemPrompt: "KRITIKER", HandoffTo: []string{"faktenwaechter"}},
		"faktenwaechter": {SystemPrompt: "FAKTENWAECHTER", HandoffTo: []string{"kritiker"}},
	}

	result, err := ChatSwarm("kritiker", agents, "hi", nil, 18, nil)
	if err != nil {
		t.Fatalf("ChatSwarm: unexpected error (oscillation should have broken the loop, not exhausted rounds): %v", err)
	}
	if result.Content != "Done despite the loop." {
		t.Errorf("Content = %q", result.Content)
	}
	wantPath := []string{"kritiker", "faktenwaechter", "kritiker", "faktenwaechter"}
	if !equalStrSlices(result.Path, wantPath) {
		t.Errorf("Path = %v, want %v", result.Path, wantPath)
	}
	if round != 4 {
		t.Errorf("server received %d requests, want 4 (loop should break well before max_rounds=18)", round)
	}
}

func TestChatSwarmUnknownEntryAgent(t *testing.T) {
	_, err := ChatSwarm("nope", map[string]SwarmAgentSpec{}, "hi", nil, 5, nil)
	if err == nil {
		t.Fatal("expected an error for an unknown entry agent")
	}
}

func equalStrSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
