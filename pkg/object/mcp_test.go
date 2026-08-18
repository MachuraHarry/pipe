package object

import (
	"testing"

	"github.com/MachuraHarry/pipe/pkg/mcp"
)

func TestMCPServerDispatch(t *testing.T) {
	s := mcp.NewServer("test", "1.0")
	called := false
	s.AddTool(mcp.ToolDef{
		Name:        "hello",
		Description: "Say hello",
		Params: []mcp.ParamDef{
			{Name: "name", Description: "Who to greet"},
		},
	}, func(args map[string]interface{}) (string, error) {
		called = true
		return "Hello, " + args["name"].(string) + "!", nil
	})

	reply := s.DispatchRaw([]byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","clientInfo":{"name":"t","version":"1"}}}`))
	if reply == nil {
		t.Fatal("expected initialize response")
	}

	reply = s.DispatchRaw([]byte(`{"jsonrpc":"2.0","method":"notifications/initialized"}`))
	if reply != nil {
		t.Fatal("expected nil for notification")
	}

	reply = s.DispatchRaw([]byte(`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`))
	if reply == nil {
		t.Fatal("expected tools/list response")
	}

	reply = s.DispatchRaw([]byte(`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"hello","arguments":{"name":"World"}}}`))
	if reply == nil {
		t.Fatal("expected tools/call response")
	}
	if !called {
		t.Fatal("tool handler was not called")
	}

	reply = s.DispatchRaw([]byte(`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"unknown","arguments":{}}}`))
	if reply == nil {
		t.Fatal("expected error for unknown tool")
	}
}

func TestMCPServerEmpty(t *testing.T) {
	s := mcp.NewServer("empty", "1.0")
	tools := s.Tools()
	if len(tools) != 0 {
		t.Fatal("expected 0 tools")
	}
}

func TestValidAlias(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"github", true},
		{"postgres", true},
		{"_private", true},
		{"with_underscore123", true},
		{"WithCaps", true},
		{"", false},
		{"1startsdigit", false},
		{"has space", false},
		{"has-dash", false},
		{"has.dot", false},
	}
	for _, c := range cases {
		if got := validAlias(c.in); got != c.want {
			t.Errorf("validAlias(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestResolvePrefix(t *testing.T) {
	// Save and restore global client state for isolation.
	orig := mcpClients
	defer func() { mcpClients = orig }()

	t.Run("empty alias uses registration order", func(t *testing.T) {
		mcpClients = []*mcpClientEntry{}
		p, err := resolvePrefix("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p != "mcp0_" {
			t.Fatalf("prefix = %q, want mcp0_", p)
		}
		mcpClients = []*mcpClientEntry{{prefix: "mcp0_"}}
		p, _ = resolvePrefix("")
		if p != "mcp1_" {
			t.Fatalf("prefix = %q, want mcp1_", p)
		}
	})

	t.Run("alias maps to alias_", func(t *testing.T) {
		mcpClients = []*mcpClientEntry{}
		p, err := resolvePrefix("github")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p != "github_" {
			t.Fatalf("prefix = %q, want github_", p)
		}
	})

	t.Run("invalid alias errors", func(t *testing.T) {
		mcpClients = []*mcpClientEntry{}
		if _, err := resolvePrefix("1bad"); err == nil {
			t.Fatal("expected error for invalid alias")
		}
	})

	t.Run("collision errors", func(t *testing.T) {
		mcpClients = []*mcpClientEntry{{prefix: "github_"}}
		if _, err := resolvePrefix("github"); err == nil {
			t.Fatal("expected collision error")
		}
	})
}
