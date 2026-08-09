package mcp

import (
	"encoding/json"
	"strings"
	"testing"
)

func newTestServer(t *testing.T, handler ToolHandler) *Server {
	t.Helper()
	s := NewServer("test", "1.0")
	s.AddTool(ToolDef{
		Name:        "hello",
		Description: "Say hello",
		Params: []ParamDef{
			{Name: "name", Description: "Who to greet"},
		},
	}, func(args map[string]interface{}) (string, error) {
		if handler != nil {
			return handler(args)
		}
		return "Hello, " + args["name"].(string) + "!", nil
	})
	return s
}

func callRaw(t *testing.T, s *Server, raw string) string {
	t.Helper()
	reply := s.DispatchRaw([]byte(raw))
	if reply == nil {
		t.Fatal("expected response")
	}
	data, err := json.Marshal(reply)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestVersionNegotiation(t *testing.T) {
	s := NewServer("test", "1.0")

	cases := []struct {
		clientVersion string
		want          string
	}{
		{"2025-11-25", "2025-11-25"},
		{"2025-06-18", "2025-06-18"},
		{"2025-03-26", "2025-03-26"},
		{"2024-11-05", "2024-11-05"},
		{"9999-01-01", "2025-11-25"},
		{"", "2025-11-25"},
	}

	for _, c := range cases {
		var out struct {
			Result struct {
				ProtocolVersion string `json:"protocolVersion"`
			} `json:"result"`
		}
		resp := callRaw(t, s, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"`+c.clientVersion+`"}}`)
		if err := json.Unmarshal([]byte(resp), &out); err != nil {
			t.Fatalf("client version %q: %v", c.clientVersion, err)
		}
		if out.Result.ProtocolVersion != c.want {
			t.Errorf("client version %q: got %q, want %q", c.clientVersion, out.Result.ProtocolVersion, c.want)
		}
	}
}

func TestPing(t *testing.T) {
	s := NewServer("test", "1.0")
	resp := callRaw(t, s, `{"jsonrpc":"2.0","id":5,"method":"ping"}`)
	if !strings.Contains(resp, `"id":5`) {
		t.Fatalf("ping response should echo id, got %s", resp)
	}
}

func TestMissingRequiredArg(t *testing.T) {
	called := false
	s := NewServer("test", "1.0")
	s.AddTool(ToolDef{
		Name: "hello",
		Params: []ParamDef{
			{Name: "name", Description: "Who to greet"},
		},
	}, func(args map[string]interface{}) (string, error) {
		called = true
		return "ok", nil
	})

	resp := callRaw(t, s, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"hello","arguments":{}}}`)
	var out struct {
		Result struct {
			Content []ContentItem `json:"content"`
			IsError bool          `json:"isError"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(resp), &out); err != nil {
		t.Fatal(err)
	}
	if !out.Result.IsError {
		t.Fatalf("expected isError for missing required arg, got %s", resp)
	}
	if called {
		t.Fatal("handler must not be called when required arg is missing")
	}
}

func TestInvalidToolNameSkipped(t *testing.T) {
	s := NewServer("test", "1.0")
	s.AddTool(ToolDef{Name: "bad name!", Params: nil}, func(args map[string]interface{}) (string, error) { return "", nil })
	s.AddTool(ToolDef{Name: "good-name_1", Params: nil}, func(args map[string]interface{}) (string, error) { return "", nil })
	if got := len(s.Tools()); got != 1 {
		t.Fatalf("expected 1 valid tool, got %d", got)
	}
}

func TestStructuredResultPassthrough(t *testing.T) {
	// Structured results are produced by the Pipe layer; here we just verify
	// a handler can return JSON text untouched (escaped inside text content).
	s := NewServer("test", "1.0")
	s.AddTool(ToolDef{Name: "json_tool"}, func(args map[string]interface{}) (string, error) {
		return `{"a": 1, "b": [1,2,3]}`, nil
	})
	resp := callRaw(t, s, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"json_tool","arguments":{}}}`)
	var out struct {
		Result struct {
			Content []ContentItem `json:"content"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(resp), &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Result.Content) != 1 || out.Result.Content[0].Text != `{"a": 1, "b": [1,2,3]}` {
		t.Fatalf("expected JSON text passthrough, got %s", resp)
	}
}

func TestServerResourcesAndPrompts(t *testing.T) {
	s := NewServer("test", "1.0")
	s.AddResource(Resource{URI: "docs://pipe", Name: "Pipe Docs", MIMEType: "text/markdown"},
		func(uri string) (string, error) { return "docs for " + uri, nil })
	s.AddResourceTemplate(ResourceTemplate{URITemplate: "file:///{path}", Name: "File", MIMEType: "text/plain"},
		func(uri string) (string, error) { return "content of " + uri, nil })
	s.AddPrompt(Prompt{Name: "greet", Description: "Greet", Arguments: []PromptArgument{
		{Name: "name", Description: "Who", Required: true},
		{Name: "tone", Description: "Optional tone", Required: false},
	}}, func(args map[string]string) (string, error) {
		return "Hello, " + args["name"] + "!", nil
	}, nil)

	// resources/list lists static resources only.
	resp := callRaw(t, s, `{"jsonrpc":"2.0","id":1,"method":"resources/list"}`)
	if !strings.Contains(resp, `docs://pipe`) || strings.Contains(resp, "file:///{path}") {
		t.Fatalf("resources/list unexpected: %s", resp)
	}

	// resources/templates/list lists URI templates.
	resp = callRaw(t, s, `{"jsonrpc":"2.0","id":2,"method":"resources/templates/list"}`)
	if !strings.Contains(resp, `file:///{path}`) {
		t.Fatalf("resources/templates/list unexpected: %s", resp)
	}

	// resources/read: static resource.
	resp = callRaw(t, s, `{"jsonrpc":"2.0","id":3,"method":"resources/read","params":{"uri":"docs://pipe"}}`)
	if !strings.Contains(resp, "docs for docs://pipe") {
		t.Fatalf("resources/read static unexpected: %s", resp)
	}

	// resources/read: template match.
	resp = callRaw(t, s, `{"jsonrpc":"2.0","id":4,"method":"resources/read","params":{"uri":"file:///a/b.txt"}}`)
	if !strings.Contains(resp, "content of file:///a/b.txt") {
		t.Fatalf("resources/read template unexpected: %s", resp)
	}

	// resources/read: unknown resource -> isError.
	resp = callRaw(t, s, `{"jsonrpc":"2.0","id":5,"method":"resources/read","params":{"uri":"docs://nope"}}`)
	if !strings.Contains(resp, `"isError":true`) {
		t.Fatalf("resources/read unknown should be error: %s", resp)
	}

	// prompts/list lists the registered prompt with arguments.
	resp = callRaw(t, s, `{"jsonrpc":"2.0","id":6,"method":"prompts/list"}`)
	if !strings.Contains(resp, `greet`) || !strings.Contains(resp, `"required":true`) {
		t.Fatalf("prompts/list unexpected: %s", resp)
	}

	// prompts/get renders the prompt.
	resp = callRaw(t, s, `{"jsonrpc":"2.0","id":7,"method":"prompts/get","params":{"name":"greet","arguments":{"name":"World"}}}`)
	if !strings.Contains(resp, "Hello, World!") {
		t.Fatalf("prompts/get unexpected: %s", resp)
	}

	// prompts/get: missing required argument -> isError.
	resp = callRaw(t, s, `{"jsonrpc":"2.0","id":8,"method":"prompts/get","params":{"name":"greet","arguments":{}}}`)
	if !strings.Contains(resp, "Missing required argument: name") || !strings.Contains(resp, `"isError":true`) {
		t.Fatalf("prompts/get missing arg unexpected: %s", resp)
	}

	// prompts/get: unknown prompt -> isError.
	resp = callRaw(t, s, `{"jsonrpc":"2.0","id":9,"method":"prompts/get","params":{"name":"nope","arguments":{}}}`)
	if !strings.Contains(resp, "Unknown prompt: nope") {
		t.Fatalf("prompts/get unknown unexpected: %s", resp)
	}
}
