package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
)

// startServer builds and launches the pipe-lsp binary as a subprocess.
func startServer(t *testing.T) (*exec.Cmd, io.WriteCloser, *bufio.Reader, func()) {
	t.Helper()
	cmd := exec.Command("go", "run", ".")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start server: %v", err)
	}
	go func() {
		data, _ := io.ReadAll(stderr)
		if len(data) > 0 {
			fmt.Fprintf(os.Stderr, "pipe-lsp stderr: %s", data)
		}
	}()
	return cmd, stdin, bufio.NewReader(stdout), func() {
		_ = stdin.Close()
		_ = cmd.Wait()
	}
}

func send(t *testing.T, w io.Writer, msg map[string]any) {
	t.Helper()
	body, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fmt.Fprintf(w, "Content-Length: %d\r\n\r\n", len(body)); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(body); err != nil {
		t.Fatal(err)
	}
}

func recv(t *testing.T, r *bufio.Reader) map[string]any {
	t.Helper()
	length := -1
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			t.Fatalf("read header: %v", err)
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		if strings.HasPrefix(strings.ToLower(line), "content-length:") {
			length, _ = strconv.Atoi(strings.TrimSpace(line[len("Content-Length:"):]))
		}
	}
	if length < 0 {
		t.Fatal("no content-length")
	}
	body := make([]byte, length)
	if _, err := io.ReadFull(r, body); err != nil {
		t.Fatalf("read body: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	return m
}

// TestStdioFlagTolerated guards against vscode-languageclient's convention of
// launching stdio servers with a trailing "--stdio" argument. The server must
// not die on that flag (flag.Parse would otherwise exit with code 2).
func TestStdioFlagTolerated(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping end-to-end in -short mode")
	}
	cmd := exec.Command("go", "run", ".", "--stdio")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start server: %v", err)
	}
	defer func() {
		_ = stdin.Close()
		_ = cmd.Wait()
	}()

	send(t, stdin, map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "initialize",
		"params": map[string]any{},
	})
	res := recv(t, bufio.NewReader(stdout))
	if res["error"] != nil {
		t.Fatalf("initialize with --stdio failed: %v", res["error"])
	}
}

func TestEndToEndStdio(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping end-to-end in -short mode")
	}
	cmd, stdin, reader, cleanup := startServer(t)
	defer cleanup()

	// 1. initialize
	send(t, stdin, map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "initialize",
		"params": map[string]any{},
	})
	res := recv(t, reader)
	if id := res["id"]; jsonNumber(id) != "1" {
		t.Fatalf("initialize response id = %v, want 1", id)
	}
	if res["error"] != nil {
		t.Fatalf("initialize error: %v", res["error"])
	}
	caps, ok := res["result"].(map[string]any)["capabilities"].(map[string]any)
	if !ok {
		t.Fatal("initialize result missing capabilities")
	}
	if caps["hoverProvider"] != true {
		t.Errorf("hoverProvider = %v, want true", caps["hoverProvider"])
	}

	// 2. initialized notification (no response expected)
	send(t, stdin, map[string]any{"jsonrpc": "2.0", "method": "initialized", "params": map[string]any{}})

	// 3. didOpen notification -> publishDiagnostics
	send(t, stdin, map[string]any{
		"jsonrpc": "2.0", "method": "textDocument/didOpen",
		"params": map[string]any{
			"textDocument": map[string]any{
				"uri": "file:///e2e.pipe", "languageId": "pipe", "version": 1,
				"text": "print missing_var\n",
			},
		},
	})
	notif := recv(t, reader)
	if notif["method"] != "textDocument/publishDiagnostics" {
		t.Fatalf("expected publishDiagnostics, got method %v", notif["method"])
	}
	diags := notif["params"].(map[string]any)["diagnostics"].([]any)
	found := false
	for _, d := range diags {
		if dm := d.(map[string]any); dm["code"] == "E001" {
			found = true
		}
	}
	if !found {
		t.Error("E001 diagnostic not published")
	}

	// 4. completion request
	send(t, stdin, map[string]any{
		"jsonrpc": "2.0", "id": 2, "method": "textDocument/completion",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": "file:///e2e.pipe"},
			"position":     map[string]any{"line": 1, "character": 0},
		},
	})
	res = recv(t, reader)
	items := res["result"].(map[string]any)["items"].([]any)
	if len(items) == 0 {
		t.Fatal("completion returned no items")
	}

	// 5. hover request on print
	send(t, stdin, map[string]any{
		"jsonrpc": "2.0", "id": 3, "method": "textDocument/hover",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": "file:///e2e.pipe"},
			"position":     map[string]any{"line": 0, "character": 0},
		},
	})
	res = recv(t, reader)
	contents := res["result"].(map[string]any)["contents"].(map[string]any)["value"].(string)
	if !strings.Contains(contents, "print") {
		t.Errorf("hover contents missing print: %q", contents)
	}

	// 6. unknown method -> error response
	send(t, stdin, map[string]any{"jsonrpc": "2.0", "id": 4, "method": "unknown/method", "params": map[string]any{}})
	res = recv(t, reader)
	if res["error"] == nil {
		t.Fatal("expected error for unknown method")
	}

	// 7. shutdown + exit
	send(t, stdin, map[string]any{"jsonrpc": "2.0", "id": 5, "method": "shutdown"})
	res = recv(t, reader)
	if res["result"] != nil {
		t.Fatalf("shutdown result = %v, want null", res["result"])
	}
	send(t, stdin, map[string]any{"jsonrpc": "2.0", "method": "exit"})
	if err := cmd.Wait(); err != nil {
		t.Fatalf("server did not exit cleanly: %v", err)
	}
	cleanup() // idempotent: stdin already closed by Wait
}

func jsonNumber(v any) string {
	switch x := v.(type) {
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case string:
		return x
	}
	return fmt.Sprint(v)
}
