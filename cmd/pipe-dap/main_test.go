package main

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/MachuraHarry/pipe/pkg/rpcframing"
)

// wireMsg is a decoded response or event received from serve().
type wireMsg struct {
	Type       string          `json:"type"`
	Command    string          `json:"command"`
	Event      string          `json:"event"`
	Success    bool            `json:"success"`
	Message    string          `json:"message"`
	RequestSeq int             `json:"request_seq"`
	Body       json.RawMessage `json:"body"`
}

// testClient drives one pipe-dap serve() instance over an io.Pipe in each
// direction, mirroring how a real DAP client talks to the adapter over
// stdio -- but without a real subprocess.
type testClient struct {
	t    *testing.T
	reqW *io.PipeWriter
	recv chan wireMsg
	seq  int
}

func startServer(t *testing.T) *testClient {
	t.Helper()
	reqR, reqW := io.Pipe()
	respR, respW := io.Pipe()

	go serve(reqR, respW)

	recv := make(chan wireMsg, 256)
	go func() {
		r := bufio.NewReader(respR)
		for {
			body, err := rpcframing.ReadMessage(r)
			if err != nil {
				close(recv)
				return
			}
			var m wireMsg
			if err := json.Unmarshal(body, &m); err != nil {
				continue
			}
			recv <- m
		}
	}()

	return &testClient{t: t, reqW: reqW, recv: recv}
}

func (c *testClient) send(command string, args any) {
	c.t.Helper()
	c.seq++
	argBytes, err := json.Marshal(args)
	if err != nil {
		c.t.Fatalf("marshal arguments for %s: %v", command, err)
	}
	req := map[string]any{
		"seq":       c.seq,
		"type":      "request",
		"command":   command,
		"arguments": json.RawMessage(argBytes),
	}
	b, err := json.Marshal(req)
	if err != nil {
		c.t.Fatalf("marshal request %s: %v", command, err)
	}
	if err := rpcframing.WriteMessage(c.reqW, b); err != nil {
		c.t.Fatalf("write request %s: %v", command, err)
	}
}

func (c *testClient) next() wireMsg {
	c.t.Helper()
	select {
	case m, ok := <-c.recv:
		if !ok {
			c.t.Fatal("server closed the connection unexpectedly")
		}
		return m
	case <-time.After(2 * time.Second):
		c.t.Fatal("timed out waiting for a message from the server")
		return wireMsg{}
	}
}

// expectResponse reads the next message, requiring it to be a response to
// the given command with the given success value.
func (c *testClient) expectResponse(command string, success bool) wireMsg {
	c.t.Helper()
	m := c.next()
	if m.Type != "response" || m.Command != command {
		c.t.Fatalf("expected a %q response, got %+v", command, m)
	}
	if m.Success != success {
		c.t.Fatalf("%s response: got success=%v (message %q), want %v", command, m.Success, m.Message, success)
	}
	return m
}

// expectEvent reads messages until it finds the named event (any responses
// or unrelated events in between are discarded), or times out.
func (c *testClient) expectEvent(name string) wireMsg {
	c.t.Helper()
	for i := 0; i < 32; i++ {
		m := c.next()
		if m.Type == "event" && m.Event == name {
			return m
		}
	}
	c.t.Fatalf("did not see event %q within 32 messages", name)
	return wireMsg{}
}

func writeTempProgram(t *testing.T, src string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "prog.pipe")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatalf("write temp program: %v", err)
	}
	return path
}

func TestDAPBreakpointStackTraceAndVariables(t *testing.T) {
	program := writeTempProgram(t, "x: 1\ny: 2\nz: x + y\nz\n")
	c := startServer(t)

	c.send("initialize", map[string]any{})
	c.expectResponse("initialize", true)
	c.expectEvent("initialized")

	c.send("launch", map[string]any{"program": program})
	c.expectResponse("launch", true)

	c.send("setBreakpoints", map[string]any{
		"source":      map[string]any{"path": program},
		"breakpoints": []map[string]any{{"line": 3}},
	})
	c.expectResponse("setBreakpoints", true)

	c.send("configurationDone", map[string]any{})
	c.expectResponse("configurationDone", true)

	stopped := c.expectEvent("stopped")
	var stoppedBody struct {
		Reason string `json:"reason"`
	}
	if err := json.Unmarshal(stopped.Body, &stoppedBody); err != nil {
		t.Fatalf("decode stopped body: %v", err)
	}
	if stoppedBody.Reason != "breakpoint" {
		t.Fatalf("stopped reason: got %q, want breakpoint", stoppedBody.Reason)
	}

	c.send("stackTrace", map[string]any{"threadId": 1})
	stResp := c.expectResponse("stackTrace", true)
	var stBody struct {
		StackFrames []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
			Line int    `json:"line"`
		} `json:"stackFrames"`
	}
	if err := json.Unmarshal(stResp.Body, &stBody); err != nil {
		t.Fatalf("decode stackTrace body: %v", err)
	}
	if len(stBody.StackFrames) != 1 {
		t.Fatalf("stackFrames: got %d frames, want 1: %+v", len(stBody.StackFrames), stBody.StackFrames)
	}
	if stBody.StackFrames[0].Line != 3 {
		t.Fatalf("top frame line: got %d, want 3", stBody.StackFrames[0].Line)
	}
	topFrameID := stBody.StackFrames[0].ID

	c.send("scopes", map[string]any{"frameId": topFrameID})
	scResp := c.expectResponse("scopes", true)
	var scBody struct {
		Scopes []struct {
			Name               string `json:"name"`
			VariablesReference int    `json:"variablesReference"`
		} `json:"scopes"`
	}
	if err := json.Unmarshal(scResp.Body, &scBody); err != nil {
		t.Fatalf("decode scopes body: %v", err)
	}
	var globalsRef int
	found := false
	for _, sc := range scBody.Scopes {
		if sc.Name == "Globals" {
			globalsRef = sc.VariablesReference
			found = true
		}
	}
	if !found {
		t.Fatalf("no Globals scope in %+v", scBody.Scopes)
	}

	c.send("variables", map[string]any{"variablesReference": globalsRef})
	varResp := c.expectResponse("variables", true)
	var varBody struct {
		Variables []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"variables"`
	}
	if err := json.Unmarshal(varResp.Body, &varBody); err != nil {
		t.Fatalf("decode variables body: %v", err)
	}
	got := map[string]string{}
	for _, v := range varBody.Variables {
		got[v.Name] = v.Value
	}
	if got["x"] != "1" || got["y"] != "2" {
		t.Fatalf("globals at breakpoint: got %+v, want x=1 y=2 (z not yet assigned)", got)
	}
	if _, hasZ := got["z"]; hasZ {
		t.Fatalf("globals at breakpoint: z should not be set yet, got %+v", got)
	}

	c.send("continue", map[string]any{"threadId": 1})
	c.expectResponse("continue", true)

	c.expectEvent("exited")
	c.expectEvent("terminated")
}

func TestDAPDisconnectUnblocksAPausedProgram(t *testing.T) {
	program := writeTempProgram(t, "x: 1\nx\n")
	c := startServer(t)

	c.send("initialize", map[string]any{})
	c.expectResponse("initialize", true)
	c.expectEvent("initialized")

	c.send("launch", map[string]any{"program": program})
	c.expectResponse("launch", true)

	c.send("setBreakpoints", map[string]any{
		"source":      map[string]any{"path": program},
		"breakpoints": []map[string]any{{"line": 1}},
	})
	c.expectResponse("setBreakpoints", true)

	c.send("configurationDone", map[string]any{})
	c.expectResponse("configurationDone", true)
	c.expectEvent("stopped")

	c.send("disconnect", map[string]any{})
	c.expectResponse("disconnect", true)

	c.expectEvent("exited")
	c.expectEvent("terminated")
}

func TestDAPStopOnEntry(t *testing.T) {
	program := writeTempProgram(t, "x: 1\nx\n")
	c := startServer(t)

	c.send("initialize", map[string]any{})
	c.expectResponse("initialize", true)
	c.expectEvent("initialized")

	c.send("launch", map[string]any{"program": program, "stopOnEntry": true})
	c.expectResponse("launch", true)

	c.send("configurationDone", map[string]any{})
	c.expectResponse("configurationDone", true)

	stopped := c.expectEvent("stopped")
	var body struct {
		Reason string `json:"reason"`
	}
	if err := json.Unmarshal(stopped.Body, &body); err != nil {
		t.Fatalf("decode stopped body: %v", err)
	}
	if body.Reason != "entry" {
		t.Fatalf("stopped reason: got %q, want entry", body.Reason)
	}

	c.send("continue", map[string]any{"threadId": 1})
	c.expectResponse("continue", true)
	c.expectEvent("exited")
	c.expectEvent("terminated")
}
