package ai

import "testing"

func TestParseMarkupToolCalls(t *testing.T) {
	// DeepSeek-style XML markup should be recognized and turned into real calls.
	markup := "<|tool_calls|>\n" +
		"<|invoke name=\"search_web\">\n" +
		"<|parameter name=\"query\">pipe language</parameter>\n" +
		"</invoke>\n" +
		"<|invoke name=\"__handoff__\">\n" +
		"<|parameter name=\"to\">analyst</parameter>\n" +
		"</invoke>\n" +
		"<|/tool_calls|>"

	calls, ok := parseMarkupToolCalls(markup)
	if !ok {
		t.Fatalf("expected markup to be recognized as tool calls, got ok=false")
	}
	if len(calls) != 2 {
		t.Fatalf("expected 2 calls, got %d: %+v", len(calls), calls)
	}
	if calls[0].Name != "search_web" {
		t.Errorf("call 0 name = %q, want search_web", calls[0].Name)
	}
	if calls[0].Arguments != `{"query":"pipe language"}` {
		t.Errorf("call 0 args = %s, want {\"query\":\"pipe language\"}", calls[0].Arguments)
	}
	if calls[1].Name != "__handoff__" || calls[1].Arguments != `{"to":"analyst"}` {
		t.Errorf("call 1 = %+v, want handoff to analyst", calls[1])
	}
}

func TestParseMarkupToolCallsIgnoresPlainText(t *testing.T) {
	// Ordinary conversational text without markup must not be treated as a call.
	calls, ok := parseMarkupToolCalls("You are a research coordinator. Transfer to the researcher.")
	if ok {
		t.Fatalf("plain text should not be recognized as tool calls, got %+v", calls)
	}
	if len(calls) != 0 {
		t.Fatalf("expected no calls, got %+v", calls)
	}
}

func TestParseMarkupToolCallsReportsUnknownTools(t *testing.T) {
	// Markup naming a tool that is not offered is still reported as a call; the
	// executor rejects it with an "unknown tool" result the model can react to.
	// This keeps stray markup from surfacing as a final answer.
	markup := "<|tool_calls|>\n<|invoke name=\"web_search\">\n<|parameter name=\"query\">x</parameter>\n</invoke>\n<|/tool_calls|>"
	calls, ok := parseMarkupToolCalls(markup)
	if !ok {
		t.Fatalf("expected markup to be reported, got ok=false")
	}
	if len(calls) != 1 || calls[0].Name != "web_search" {
		t.Fatalf("expected a reported web_search call, got %+v", calls)
	}
}

func TestParseMarkupToolCallsHandoffOnly(t *testing.T) {
	// A lone handoff emitted as markup (e.g. from a coordinator without real
	// tools) should be recognized so the swarm advances instead of stalling.
	markup := "<|tool_calls|>\n<|invoke name=\"__handoff__\">\n<|parameter name=\"to\">b</parameter>\n</invoke>\n<|/tool_calls|>"
	calls, ok := parseMarkupToolCalls(markup)
	if !ok {
		t.Fatalf("expected handoff markup to be recognized, got ok=false")
	}
	if len(calls) != 1 || calls[0].Name != "__handoff__" || calls[0].Arguments != `{"to":"b"}` {
		t.Fatalf("unexpected handoff parsing: %+v", calls)
	}
}
