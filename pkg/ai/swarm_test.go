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
