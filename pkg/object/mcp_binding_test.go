package object

import (
	"testing"

	"github.com/MachuraHarry/pipe/pkg/ai"
)

// TestAiToolMCPBindingOrderDeterministic guards against nondeterministic
// positional argument binding for ai_tool-registered MCP tools: the "required"
// list stored by bAiTool must be consumable by the mcp_server bridge
// ([]interface{}), which otherwise falls back to random Go map iteration and
// binds arguments to parameters in arbitrary order.
func TestAiToolMCPBindingOrderDeterministic(t *testing.T) {
	defer resetMCPRegistries()
	resetMCPRegistries()

	var got []string
	probe := &BuiltinInfo{Name: "probe_fn", Fn: func(args ...Object) Object {
		for _, a := range args {
			s, ok := a.(*String)
			if !ok {
				t.Errorf("bound non-string argument: %s", a.Inspect())
				return NILOBJ
			}
			got = append(got, s.Value)
		}
		return &String{Value: "ok"}
	}}

	// Six schema keys in deliberately unsorted declaration order: a binding
	// order chosen at random would match sorted order only ~1/720 of runs.
	schema := MapFromGo(map[string]Object{
		"zulu":   &String{Value: "zulu desc"},
		"mike":   &String{Value: "mike desc"},
		"alpha":  &String{Value: "alpha desc"},
		"xray":   &String{Value: "xray desc"},
		"bravo":  &String{Value: "bravo desc"},
		"yankee": &String{Value: "yankee desc"},
	})
	if r := bAiTool(&String{Value: "probe"}, &String{Value: "probe tool"}, schema, probe); r.Type() == ERROR {
		t.Fatal("ai_tool failed: " + r.Inspect())
	}
	if r := bMcpServer(&String{Value: "srv"}, &String{Value: "1.0"}); r.Type() == ERROR {
		t.Fatal("mcp_server failed: " + r.Inspect())
	}

	reply := currentMCPServer.DispatchRaw([]byte(
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"probe","arguments":` +
			`{"zulu":"Z","mike":"M","alpha":"A","xray":"X","bravo":"B","yankee":"Y"}}}`))
	if reply == nil {
		t.Fatal("no tools/call response")
	}

	want := []string{"A", "B", "M", "X", "Y", "Z"} // alphabetical binding order
	if len(got) != len(want) {
		t.Fatalf("probe received %d args (%v), want %d (handler may have been rejected before invocation)", len(got), got, len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("binding order broken: got %v, want %v", got, want)
		}
	}
}

// TestOrderedToolArgsPreservesPositionWhenOptionalParamsOmitted guards
// against a second, related real bug found alongside the one
// TestParamOrderFromSchemaIncludesOptionalParams documents: even once
// paramOrderFromSchema returns the tool's full parameter list, a schema
// routinely declares MORE optional parameters than any one call actually
// uses (Gmail's send_gmail_message alone has ~13: to/subject/body plus
// cc/bcc/thread_id/reply_all/forward_message_id/...). orderedToolArgs used
// to silently COMPACT OUT every name the caller didn't provide, which
// shifted every later-in-schema-order argument left by one position for
// each earlier one omitted — so a call passing only
// {user_google_email, to, subject, body} landed those values on whatever
// parameter names happened to occupy positions 0-3 in the full schema
// order, not on user_google_email/to/subject/body. Reproduced live: the
// "to" value (an email address) arrived at the server as "body_format",
// and the real body text arrived as "attachments". Fixed by having
// orderedToolArgs emit exactly len(names) slots, NILOBJ for anything
// omitted, so position always matches paramOrderFromSchema's order.
func TestOrderedToolArgsPreservesPositionWhenOptionalParamsOmitted(t *testing.T) {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"user_google_email": map[string]interface{}{"type": "string"},
			"to":                map[string]interface{}{"type": "string"},
			"subject":           map[string]interface{}{"type": "string"},
			"body":              map[string]interface{}{"type": "string"},
			"cc":                map[string]interface{}{"type": "string"},
			"bcc":               map[string]interface{}{"type": "string"},
		},
		"required": []interface{}{"user_google_email"},
	}
	// paramOrderFromSchema order for this schema: user_google_email
	// (required), then sorted extras: bcc, body, cc, subject, to.
	entry := ToolEntry{Def: ai.ToolDef{Parameters: schema}}
	provided := map[string]interface{}{
		"user_google_email": "harry@example.com",
		"to":                "harry@example.com",
		"subject":           "Hallo",
		"body":              "Hallo Welt",
	}
	got := orderedToolArgs(entry, provided)
	names := toolParamNames(entry)
	if len(got) != len(names) {
		t.Fatalf("orderedToolArgs returned %d objects, want %d (one per declared param, matching paramOrderFromSchema order): %v", len(got), len(names), names)
	}
	for i, n := range names {
		want, wasProvided := provided[n]
		if !wasProvided {
			if got[i].Type() != NIL {
				t.Errorf("position %d (param %q, not provided by caller) = %s, want NIL", i, n, got[i].Inspect())
			}
			continue
		}
		s, ok := got[i].(*String)
		if !ok || s.Value != want {
			t.Errorf("position %d (param %q) = %s, want %q", i, n, got[i].Inspect(), want)
		}
	}
}

// TestParamOrderFromSchemaIncludesOptionalParams guards against a real bug:
// a schema like Gmail's send_gmail_message declares only user_google_email
// as required — to/subject/body are optional (Python defaults), but a caller
// routinely provides them anyway. Returning only the "required" names here
// (the old behavior) silently dropped every one of those genuinely-provided
// optional arguments on both sides of the MCP client bridge — orderedToolArgs
// (which arg values to pull out of the caller's map) and extractParamNames
// (which names to zip the resulting positional slice back onto) — so the
// remote tool received user_google_email alone and rejected the call as
// missing required fields, even though the caller passed everything correctly.
func TestParamOrderFromSchemaIncludesOptionalParams(t *testing.T) {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"user_google_email": map[string]interface{}{"type": "string"},
			"to":                map[string]interface{}{"type": "string"},
			"subject":           map[string]interface{}{"type": "string"},
			"body":              map[string]interface{}{"type": "string"},
		},
		"required": []interface{}{"user_google_email"},
	}
	got := paramOrderFromSchema(schema)
	want := []string{"user_google_email", "body", "subject", "to"} // required first, then sorted extras
	if len(got) != len(want) {
		t.Fatalf("paramOrderFromSchema = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("paramOrderFromSchema = %v, want %v", got, want)
		}
	}
}

// TestExtractParamNamesMatchesToolParamNames guards against the two halves of
// the MCP client bridge drifting apart again: orderedToolArgs (call site,
// keyed off toolParamNames/entry.Def.Parameters) zips argument VALUES into a
// positional slice by this order, and the registered tool's Fn (keyed off
// extractParamNames/bridge.paramNames, set once at registerMCPClient time)
// unzips that same slice back into argument NAMES by this order — if the two
// ever compute a different order for the same schema, arguments silently
// scramble or drop. Both now delegate to paramOrderFromSchema, so this test
// mostly documents the invariant, but keeps failing loudly if that changes.
func TestExtractParamNamesMatchesToolParamNames(t *testing.T) {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"user_google_email": map[string]interface{}{"type": "string"},
			"to":                map[string]interface{}{"type": "string"},
			"subject":           map[string]interface{}{"type": "string"},
			"body":              map[string]interface{}{"type": "string"},
		},
		"required": []interface{}{"user_google_email"},
	}
	entry := ToolEntry{Def: ai.ToolDef{Parameters: schema}}
	fromRegistration := extractParamNames(schema)
	fromCallSite := toolParamNames(entry)
	if len(fromRegistration) != len(fromCallSite) {
		t.Fatalf("extractParamNames = %v, toolParamNames = %v", fromRegistration, fromCallSite)
	}
	for i := range fromRegistration {
		if fromRegistration[i] != fromCallSite[i] {
			t.Fatalf("extractParamNames = %v, toolParamNames = %v", fromRegistration, fromCallSite)
		}
	}
}
