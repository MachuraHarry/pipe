package object

import (
	"strings"
	"testing"
)

func resourceHandler(fn func(uri string) string) *BuiltinInfo {
	return &BuiltinInfo{
		Fn: func(args ...Object) Object {
			return &String{Value: fn(args[0].(*String).Value)}
		},
	}
}

func resetMCPRegistries() {
	resourceRegistry = nil
	templateRegistry = nil
	promptRegistry = nil
	currentMCPServer = nil
}

func TestMCPResourceBuiltins(t *testing.T) {
	defer resetMCPRegistries()
	resetMCPRegistries()

	r := bMcpResource(&String{Value: "docs://pipe"}, &String{Value: "Docs"}, &String{Value: "text/plain"},
		resourceHandler(func(uri string) string { return "doc for " + uri }))
	if r.Type() != NIL {
		t.Error("mcp_resource should return nil")
	}

	res := bMcpResources()
	list, ok := res.(*List)
	if !ok || len(list.Elements) != 1 {
		t.Fatalf("mcp_resources = %s, want 1 entry", res.Inspect())
	}
	entry := list.Elements[0].(*Map)
	if entry.Pairs["uri"].Inspect() != "docs://pipe" || entry.Pairs["source"].Inspect() != "local" {
		t.Fatalf("unexpected resource entry: %s", entry.Inspect())
	}

	got := bMcpReadResource(&String{Value: "docs://pipe"})
	if got.Inspect() != "doc for docs://pipe" {
		t.Errorf("mcp_read_resource = %s", got.Inspect())
	}

	if got := bMcpReadResource(&String{Value: "docs://nope"}).Type(); got != ERROR {
		t.Errorf("unknown resource should error, got %s", got)
	}
}

func TestMCPResourceTemplateBuiltin(t *testing.T) {
	defer resetMCPRegistries()
	resetMCPRegistries()

	bMcpResourceTemplate(&String{Value: "file:///{path}"}, &String{Value: "File"}, &String{Value: "text/plain"},
		resourceHandler(func(uri string) string { return "content: " + uri }))

	got := bMcpReadResource(&String{Value: "file:///a/b.txt"})
	if got.Inspect() != "content: file:///a/b.txt" {
		t.Errorf("mcp_read_resource template = %s", got.Inspect())
	}
}

func TestMCPPromptBuiltins(t *testing.T) {
	defer resetMCPRegistries()
	resetMCPRegistries()

	build := &BuiltinInfo{
		Fn: func(args ...Object) Object {
			m := args[0].(*Map)
			return &String{Value: "hello " + m.Pairs["name"].(*String).Value}
		},
	}
	if r := bMcpPrompt(&String{Value: "greet"}, &String{Value: "Greet"},
		&Map{Pairs: map[string]Object{"name": &String{Value: "The name"}}}, build); r.Type() != NIL {
		t.Error("mcp_prompt should return nil")
	}

	prompts := bMcpPrompts().(*List)
	if len(prompts.Elements) != 1 {
		t.Fatalf("mcp_prompts = %s, want 1 entry", prompts.Inspect())
	}

	got := bMcpPromptGet(&String{Value: "greet"}, &Map{Pairs: map[string]Object{"name": &String{Value: "World"}}})
	if got.Inspect() != "hello World" {
		t.Errorf("mcp_prompt_get = %s", got.Inspect())
	}

	if got := bMcpPromptGet(&String{Value: "nope"}, NILOBJ).Type(); got != ERROR {
		t.Errorf("unknown prompt should error, got %s", got)
	}
}

func TestMCPPromptRequiredArg(t *testing.T) {
	defer resetMCPRegistries()
	resetMCPRegistries()

	bMcpPrompt(&String{Value: "greet"}, &String{Value: "Greet"},
		&Map{Pairs: map[string]Object{"name": &String{Value: "The name"}}},
		&BuiltinInfo{Fn: func(args ...Object) Object { return &String{Value: "ok"} }})

	s := bMcpServer(&String{Value: "srv"}, &String{Value: "1.0"})
	if s.Type() == ERROR {
		t.Fatal("mcp_server failed: " + s.Inspect())
	}

	// Missing required argument must fail, even when bridged into the server.
	got := bMcpPromptGet(&String{Value: "greet"}, &Map{})
	if got.Type() != ERROR || !strings.Contains(got.Inspect(), "missing required argument") {
		t.Errorf("expected missing-arg error, got %s", got.Inspect())
	}
}

func TestMCPBuiltinsErrorArgs(t *testing.T) {
	defer resetMCPRegistries()
	resetMCPRegistries()

	if bMcpResource(&String{Value: "a"}).Type() != ERROR {
		t.Error("mcp_resource with 1 arg should error")
	}
	if bMcpResourceTemplate(&String{Value: "a"}).Type() != ERROR {
		t.Error("mcp_resource_template with 1 arg should error")
	}
	if bMcpPrompt(&String{Value: "a"}).Type() != ERROR {
		t.Error("mcp_prompt with 1 arg should error")
	}
	if bMcpReadResource().Type() != ERROR {
		t.Error("mcp_read_resource without args should error")
	}
	if bMcpPromptGet().Type() != ERROR {
		t.Error("mcp_prompt_get without args should error")
	}
	if bMcpResources().Type() != ERROR {
		t.Error("mcp_resources with empty registries should error")
	}
	if bMcpPrompts().Type() != ERROR {
		t.Error("mcp_prompts with empty registries should error")
	}
}
