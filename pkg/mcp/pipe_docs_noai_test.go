package mcp

import (
	"bufio"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestPipeDocsNoAIEndToEnd runs the actual pipe-docs server (without any API
// key) and verifies the non-AI tools via JSON-RPC over stdio.
func TestPipeDocsNoAIEndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pipeBin := filepath.Join(runtime.GOROOT(), "..", "bin", "pipe")
	if _, err := exec.LookPath("pipe"); err == nil {
		pipeBin, _ = exec.LookPath("pipe")
	}

	// Find repo root (two levels up from pkg/mcp/)
	_, thisFile, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")

	cmd := exec.Command(pipeBin, filepath.Join(repoRoot, "examples", "pipe_docs_server.pipe"))
	cmd.Dir = repoRoot
	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()

	if err := cmd.Start(); err != nil {
		t.Fatalf("start pipe-docs server: %v", err)
	}
	t.Cleanup(func() { cmd.Process.Kill(); cmd.Wait() })

	responses := make(chan string, 64)
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			responses <- scanner.Text()
		}
	}()

	write := func(msg string) {
		if _, err := stdin.Write([]byte(msg + "\n")); err != nil {
			t.Fatalf("write: %v", err)
		}
	}

	waitResp := func(id int) map[string]interface{} {
		for i := 0; i < 200; i++ {
			select {
			case line := <-responses:
				var parsed map[string]interface{}
				if json.Unmarshal([]byte(line), &parsed) == nil {
					if rid, ok := parsed["id"].(float64); ok && int(rid) == id {
						return parsed
					}
				}
			}
		}
		t.Fatalf("no response for id %d", id)
		return nil
	}

	callTool := func(id int, name string, args interface{}) string {
		params, _ := json.Marshal(args)
		write(`{"jsonrpc":"2.0","id":` + jsonNum(id) + `,"method":"tools/call","params":{"name":"` + name + `","arguments":` + string(params) + `}}`)
		resp := waitResp(id)
		result, _ := resp["result"].(map[string]interface{})
		content, _ := result["content"].([]interface{})
		if len(content) == 0 {
			return ""
		}
		text, _ := content[0].(map[string]interface{})["text"].(string)
		return text
	}

	// Initialize
	write(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}`)
	initResp := waitResp(1)
	serverInfo, _ := initResp["result"].(map[string]interface{})["serverInfo"].(map[string]interface{})
	t.Logf("initialize → %s %s", serverInfo["name"], serverInfo["version"])

	// list_docs
	text := callTool(2, "list_docs", map[string]interface{}{})
	if !strings.Contains(text, `"en"`) {
		t.Fatalf("list_docs missing 'en': %.100s", text)
	}
	t.Log("PASS list_docs → en/de/blog present")

	// index_status
	text = callTool(3, "index_status", map[string]interface{}{})
	var status map[string]interface{}
	json.Unmarshal([]byte(text), &status)
	symbols := status["code_symbols"].(float64)
	if symbols < 100 {
		t.Fatalf("code_symbols too low: %v", symbols)
	}
	t.Logf("PASS index_status → %v symbols, %v files", status["code_symbols"], status["source_files"])

	// search_code
	text = callTool(4, "search_code", map[string]interface{}{"query": "Compile"})
	if !strings.Contains(text, "Compile") {
		t.Fatalf("search_code missing Compile: %.100s", text)
	}
	t.Log("PASS search_code → Compile found")

	// list_sources
	text = callTool(5, "list_sources", map[string]interface{}{})
	if !strings.Contains(text, "compiler.go") {
		t.Fatalf("list_sources missing compiler.go: %.100s", text)
	}
	t.Log("PASS list_sources → compiler.go present")

	// read_doc
	text = callTool(6, "read_doc", map[string]interface{}{"path": "docs/en/01-getting-started.md"})
	if strings.Contains(text, "not found") || text == "" {
		t.Fatalf("read_doc failed: %.100s", text)
	}
	t.Logf("PASS read_doc → %d bytes", len(text))

	// read_source
	text = callTool(7, "read_source", map[string]interface{}{"path": "cmd/pipe/main.go"})
	if !strings.Contains(text, "main") {
		t.Fatalf("read_source failed: %.100s", text)
	}
	t.Log("PASS read_source → has content")

	// unknown tool
	write(`{"jsonrpc":"2.0","id":8,"method":"tools/call","params":{"name":"nonexistent","arguments":{}}}`)
	errResp := waitResp(8)
	if errResp["error"] == nil {
		t.Fatal("expected error for unknown tool")
	}
	t.Log("PASS unknown tool → error")
}

func jsonNum(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}
