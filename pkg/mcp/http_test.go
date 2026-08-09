package mcp

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHTTPRoundTrip(t *testing.T) {
	s := newTestServer(t, nil)
	ts := httptest.NewServer(s.HTTPHandler())
	defer ts.Close()

	c, err := NewHTTPClient(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	initResult, err := c.Initialize()
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}
	if initResult.ProtocolVersion != LATEST_PROTOCOL_VERSION {
		t.Fatalf("protocol version: got %q, want %q", initResult.ProtocolVersion, LATEST_PROTOCOL_VERSION)
	}

	tools, err := c.ListTools()
	if err != nil {
		t.Fatalf("listTools: %v", err)
	}
	if len(tools) != 1 || tools[0].Name != "hello" {
		t.Fatalf("unexpected tools: %+v", tools)
	}

	if err := c.Ping(); err != nil {
		t.Fatalf("ping: %v", err)
	}

	res, err := c.CallTool("hello", map[string]interface{}{"name": "World"})
	if err != nil {
		t.Fatalf("callTool: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected error result: %+v", res)
	}
	if !strings.Contains(resultTextOf(res), "Hello, World") {
		t.Fatalf("unexpected result: %+v", res)
	}
}

func TestHTTPMissingRequiredArg(t *testing.T) {
	s := newTestServer(t, nil)
	ts := httptest.NewServer(s.HTTPHandler())
	defer ts.Close()

	c, _ := NewHTTPClient(ts.URL)
	defer c.Close()
	if _, err := c.Initialize(); err != nil {
		t.Fatal(err)
	}

	res, err := c.CallTool("hello", map[string]interface{}{})
	if err != nil {
		t.Fatalf("expected isError result, got error: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected isError=true, got %+v", res)
	}
}

func TestHTTPCallTimeout(t *testing.T) {
	s := NewServer("slow", "1.0")
	s.AddTool(ToolDef{Name: "slow_tool"}, func(args map[string]interface{}) (string, error) {
		time.Sleep(3 * time.Second)
		return "done", nil
	})
	ts := httptest.NewServer(s.HTTPHandler())
	defer ts.Close()

	c, _ := NewHTTPClient(ts.URL)
	c.SetCallTimeout(300 * time.Millisecond)
	defer c.Close()
	if _, err := c.Initialize(); err != nil {
		t.Fatal(err)
	}

	start := time.Now()
	_, err := c.CallTool("slow_tool", nil)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("expected timeout error, got: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("timeout did not fire promptly: %v", elapsed)
	}
}

func TestNegotiationOverHTTP(t *testing.T) {
	s := NewServer("test", "1.0")
	ts := httptest.NewServer(s.HTTPHandler())
	defer ts.Close()

	req := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}`
	resp, err := rawPOSTCapture(ts.URL, req)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(resp, `"protocolVersion":"2025-06-18"`) {
		t.Fatalf("expected negotiated 2025-06-18, got: %s", resp)
	}
}

func TestHTTPDeleteClosesSession(t *testing.T) {
	s := NewServer("test", "1.0")
	ts := httptest.NewServer(s.HTTPHandler())
	defer ts.Close()

	c, _ := NewHTTPClient(ts.URL)
	defer c.Close()
	if _, err := c.Initialize(); err != nil {
		t.Fatal(err)
	}

	// Session id should now be set; DELETE it and ensure no error.
	sid := c.sessionID
	if sid == "" {
		t.Fatal("expected session id after initialize")
	}
	req, err := http.NewRequest(http.MethodDelete, ts.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Mcp-Session-Id", sid)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on DELETE, got %d", resp.StatusCode)
	}
}

func resultTextOf(res *CallToolResult) string {
	var sb strings.Builder
	for _, c := range res.Content {
		if c.Type == "text" {
			sb.WriteString(c.Text)
		}
	}
	return sb.String()
}

func rawPOSTCapture(url, body string) (string, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader([]byte(body)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(b), "\n")
	var out []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "data:") {
			out = append(out, strings.TrimPrefix(line, "data:"))
		}
	}
	if len(out) > 0 {
		return strings.Join(out, ""), nil
	}
	return string(b), nil
}
