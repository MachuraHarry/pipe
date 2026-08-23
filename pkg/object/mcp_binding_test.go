package object

import (
	"testing"
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
	schema := &Map{Pairs: map[string]Object{
		"zulu":   &String{Value: "zulu desc"},
		"mike":   &String{Value: "mike desc"},
		"alpha":  &String{Value: "alpha desc"},
		"xray":   &String{Value: "xray desc"},
		"bravo":  &String{Value: "bravo desc"},
		"yankee": &String{Value: "yankee desc"},
	}}
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
