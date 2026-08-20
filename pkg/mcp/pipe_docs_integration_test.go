package mcp

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestPipeDocsMCPIntegration tests the pipe-docs MCP server's tool dispatch
// at the protocol level: initialize, tools/list, and tools/call for each tool.
// This validates that the MCP layer correctly routes requests and returns
// well-formed responses — without needing git clone or AI keys.

func newPipeDocsTestServer(t *testing.T) *Server {
	t.Helper()
	s := NewServer("Pipe Docs", "1.0.0")

	// Tool: read_doc
	s.AddTool(ToolDef{
		Name:        "read_doc",
		Description: "Read a documentation file by relative path",
		Params:      []ParamDef{{Name: "path", Description: "Documentation file path"}},
	}, func(args map[string]interface{}) (string, error) {
		p, _ := args["path"].(string)
		if p == "" {
			return "", nil
		}
		if strings.Contains(p, "..") {
			return "", nil // would be blocked
		}
		return "# Test doc\n\nThis is a test.", nil
	})

	// Tool: list_docs
	s.AddTool(ToolDef{
		Name:        "list_docs",
		Description: "List available documentation files",
	}, func(args map[string]interface{}) (string, error) {
		return `{"en": ["docs/en/01-intro.md"], "de": [], "blog": []}`, nil
	})

	// Tool: search_code
	s.AddTool(ToolDef{
		Name:        "search_code",
		Description: "Find Go/Pipe declarations by name or keyword",
		Params:      []ParamDef{{Name: "query", Description: "Symbol or keyword to find"}},
	}, func(args map[string]interface{}) (string, error) {
		q, _ := args["query"].(string)
		if q == "" {
			return "[]", nil
		}
		return `[{"file":"pkg/compiler/compiler.go","kind":"func","symbol":"Compile","line":100}]`, nil
	})

	// Tool: read_source
	s.AddTool(ToolDef{
		Name:        "read_source",
		Description: "Read a source file with line numbers",
		Params:      []ParamDef{{Name: "path", Description: "Source file path"}},
	}, func(args map[string]interface{}) (string, error) {
		p, _ := args["path"].(string)
		if p == "" {
			return "", nil
		}
		if strings.Contains(p, "..") {
			return "", nil
		}
		return "1\tpackage main\n2\n3\tfunc main() {}\n", nil
	})

	// Tool: list_sources
	s.AddTool(ToolDef{
		Name:        "list_sources",
		Description: "List all source files",
	}, func(args map[string]interface{}) (string, error) {
		return `["pkg/compiler/compiler.go","cmd/pipe/main.go"]`, nil
	})

	// Tool: index_status
	s.AddTool(ToolDef{
		Name:        "index_status",
		Description: "Report index statistics",
	}, func(args map[string]interface{}) (string, error) {
		return `{"ready":true,"source_files":200,"code_symbols":5000}`, nil
	})

	return s
}

func TestPipeDocsMCPInitialize(t *testing.T) {
	s := newPipeDocsTestServer(t)
	resp := callRaw(t, s, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}`)

	var result struct {
		Result struct {
			ServerInfo struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			} `json:"serverInfo"`
			ProtocolVersion string `json:"protocolVersion"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, resp)
	}
	if result.Result.ServerInfo.Name != "Pipe Docs" {
		t.Errorf("expected server name 'Pipe Docs', got %q", result.Result.ServerInfo.Name)
	}
	if result.Result.ServerInfo.Version != "1.0.0" {
		t.Errorf("expected server version '1.0.0', got %q", result.Result.ServerInfo.Version)
	}
	if result.Result.ProtocolVersion == "" {
		t.Error("expected non-empty protocol version")
	}
}

func TestPipeDocsMCPToolsList(t *testing.T) {
	s := newPipeDocsTestServer(t)
	resp := callRaw(t, s, `{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`)

	var result struct {
		Result struct {
			Tools []struct {
				Name        string `json:"name"`
				Description string `json:"description"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, resp)
	}

	expectedTools := []string{"read_doc", "list_docs", "search_code", "read_source", "list_sources", "index_status"}
	toolNames := make(map[string]bool)
	for _, tool := range result.Result.Tools {
		toolNames[tool.Name] = true
		if tool.Description == "" {
			t.Errorf("tool %q has empty description", tool.Name)
		}
	}
	for _, name := range expectedTools {
		if !toolNames[name] {
			t.Errorf("missing tool: %s", name)
		}
	}
}

func TestPipeDocsMCPToolsCallReadDoc(t *testing.T) {
	s := newPipeDocsTestServer(t)
	resp := callRaw(t, s, `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"read_doc","arguments":{"path":"docs/en/01-intro.md"}}}`)

	var result struct {
		Result struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, resp)
	}
	if len(result.Result.Content) == 0 {
		t.Fatal("expected content in response")
	}
	if result.Result.Content[0].Text == "" {
		t.Error("expected non-empty text")
	}
}

func TestPipeDocsMCPToolsCallSearchCode(t *testing.T) {
	s := newPipeDocsTestServer(t)
	resp := callRaw(t, s, `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"search_code","arguments":{"query":"Compile"}}}`)

	var result struct {
		Result struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, resp)
	}
	if len(result.Result.Content) == 0 {
		t.Fatal("expected content in response")
	}
	text := result.Result.Content[0].Text
	if !strings.Contains(text, "Compile") {
		t.Errorf("expected result to contain 'Compile', got: %s", text)
	}
}

func TestPipeDocsMCPToolsCallListDocs(t *testing.T) {
	s := newPipeDocsTestServer(t)
	resp := callRaw(t, s, `{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"list_docs","arguments":{}}}`)

	var result struct {
		Result struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, resp)
	}
	if len(result.Result.Content) == 0 {
		t.Fatal("expected content in response")
	}
	if !strings.Contains(result.Result.Content[0].Text, "en") {
		t.Error("expected result to mention 'en' docs")
	}
}

func TestPipeDocsMCPToolsCallListSources(t *testing.T) {
	s := newPipeDocsTestServer(t)
	resp := callRaw(t, s, `{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"list_sources","arguments":{}}}`)

	var result struct {
		Result struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, resp)
	}
	if len(result.Result.Content) == 0 {
		t.Fatal("expected content in response")
	}
	if !strings.Contains(result.Result.Content[0].Text, "compiler.go") {
		t.Error("expected result to contain compiler.go")
	}
}

func TestPipeDocsMCPToolsCallIndexStatus(t *testing.T) {
	s := newPipeDocsTestServer(t)
	resp := callRaw(t, s, `{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"index_status","arguments":{}}}`)

	var result struct {
		Result struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, resp)
	}
	if len(result.Result.Content) == 0 {
		t.Fatal("expected content in response")
	}
	text := result.Result.Content[0].Text
	if !strings.Contains(text, "source_files") {
		t.Error("expected result to contain 'source_files'")
	}
}

func TestPipeDocsMCPUnknownTool(t *testing.T) {
	s := newPipeDocsTestServer(t)
	resp := callRaw(t, s, `{"jsonrpc":"2.0","id":8,"method":"tools/call","params":{"name":"nonexistent_tool","arguments":{}}}`)

	var result struct {
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, resp)
	}
	if result.Error == nil {
		t.Error("expected error for unknown tool")
	}
}
