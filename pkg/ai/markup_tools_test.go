package ai

import (
	"encoding/json"
	"testing"
)

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

func TestParseMarkupToolCallsRecognizesDSMLLeakedTokens(t *testing.T) {
	// Reproduces the live-observed DeepSeek failure: a garbled leaked-special-
	// token dump instead of a structured tool_calls field.
	markup := "<｜｜DSML｜｜ calls>" +
		"<｜｜DSML｜｜ invoke name=\"__handoff__\">" +
		"<｜｜DSML｜｜ parameter name=\"to\" string=\"true\">billing</｜｜DSML｜｜ parameter>" +
		"<｜｜DSML｜｜ parameter name=\"reason\" string=\"true\">verify the invoice</｜｜DSML｜｜ parameter>" +
		"</｜｜DSML｜｜ invoke></｜｜DSML｜｜ calls>"

	calls, ok := parseMarkupToolCalls(markup)
	if !ok {
		t.Fatalf("expected DSML-style markup to be recognized as tool calls, got ok=false")
	}
	if len(calls) != 1 || calls[0].Name != "__handoff__" {
		t.Fatalf("expected 1 handoff call, got %+v", calls)
	}
	var args map[string]string
	if err := json.Unmarshal([]byte(calls[0].Arguments), &args); err != nil {
		t.Fatalf("unmarshaling arguments: %v", err)
	}
	if args["to"] != "billing" || args["reason"] != "verify the invoice" {
		t.Errorf("args = %+v, want to=billing reason=\"verify the invoice\"", args)
	}
}

func TestParseMarkupToolCallsRecognizesGenericPipeWrappedMarker(t *testing.T) {
	// A DIFFERENT marker spelling than "DSML" must still be recognized — the
	// regex matches the pipe-wrapped-marker SHAPE, not a hardcoded string.
	markup := "<｜tool▁call▁begin｜ invoke name=\"search_web\">" +
		"<｜tool▁call▁begin｜ parameter name=\"query\">pipe language</｜tool▁call▁begin｜ parameter>" +
		"</｜tool▁call▁begin｜ invoke>"
	calls, ok := parseMarkupToolCalls(markup)
	if !ok || len(calls) != 1 || calls[0].Name != "search_web" {
		t.Fatalf("expected search_web call recognized regardless of marker spelling, got ok=%v calls=%+v", ok, calls)
	}
}
