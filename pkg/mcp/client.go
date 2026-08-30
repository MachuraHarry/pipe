package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const (
	defaultCallTimeout = 120 * time.Second
	closeWaitTimeout   = 5 * time.Second
	stderrMaxBytes     = 64 * 1024
)

type Client struct {
	name         string
	mu           sync.Mutex
	nextID       int
	stdin        io.WriteCloser
	stdout       *bufio.Scanner
	stderr       *limitedBuffer
	cmd          *exec.Cmd
	mode         string // "stdio" or "http"
	httpURL      string
	httpHeaders  map[string]string
	sessionID    string
	capabilities Capabilities
	pendRes      map[int]chan json.RawMessage
	pendMu       sync.Mutex
	closed       bool
	callTimeout  time.Duration

	// NetworkGate, when set, is consulted before every HTTP request of this
	// client (including redirect targets). It is how the sandbox applies its
	// network profile to MCP connections without pkg/mcp importing the
	// sandbox package. A nil gate means no restriction.
	NetworkGate func(url string) error
}

func NewStdioClient(command string, args []string, env map[string]string) (*Client, error) {
	cmd := exec.Command(command, args...)
	if len(env) > 0 {
		cmd.Env = os.Environ()
		for k, v := range env {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("mcp client stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("mcp client stdout: %w", err)
	}
	stderr := &limitedBuffer{max: stderrMaxBytes}
	cmd.Stderr = stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("mcp client start: %w", err)
	}

	c := &Client{
		name:        command + " " + strings.Join(args, " "),
		mode:        "stdio",
		stdin:       stdin,
		stdout:      bufio.NewScanner(stdout),
		stderr:      stderr,
		cmd:         cmd,
		pendRes:     make(map[int]chan json.RawMessage),
		callTimeout: defaultCallTimeout,
	}
	c.stdout.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)
	go c.readLoop()
	return c, nil
}

func NewHTTPClient(url string) (*Client, error) {
	c := &Client{
		name:        url,
		mode:        "http",
		httpURL:     url,
		httpHeaders: make(map[string]string),
		pendRes:     make(map[int]chan json.RawMessage),
		callTimeout: defaultCallTimeout,
	}
	return c, nil
}

// SetHeader adds an extra HTTP header (e.g. Authorization) to every request
// of an HTTP-based client. Only valid before Initialize.
func (c *Client) SetHeader(name, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.httpHeaders == nil {
		c.httpHeaders = make(map[string]string)
	}
	c.httpHeaders[name] = value
}

// SetCallTimeout overrides the per-call timeout (default 120s).
func (c *Client) SetCallTimeout(d time.Duration) {
	if d <= 0 {
		d = defaultCallTimeout
	}
	c.mu.Lock()
	c.callTimeout = d
	c.mu.Unlock()
}

func (c *Client) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	stdin := c.stdin
	cmd := c.cmd
	c.mu.Unlock()

	if stdin != nil {
		stdin.Close()
	}
	if cmd != nil {
		done := make(chan struct{})
		go func() {
			cmd.Wait()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(closeWaitTimeout):
			if cmd.Process != nil {
				cmd.Process.Kill()
				<-done
			}
		}
	}
	return nil
}

func (c *Client) Initialize() (*InitializeResult, error) {
	req := JSONRPCRequest{
		Jsonrpc: "2.0",
		Method:  "initialize",
		ID:      1,
		Params: jsonToRaw(InitializeParams{
			ProtocolVersion: LATEST_PROTOCOL_VERSION,
			Capabilities:    ClientCapabilities{},
			ClientInfo: struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			}{Name: "pipe-mcp-client", Version: "1.0"},
		}),
	}
	resp, err := c.sendRequest(req)
	if err != nil {
		return nil, err
	}
	var result InitializeResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("parse initialize result: %w", err)
	}
	c.capabilities = result.Capabilities

	// Send initialized notification
	initialized := jsonrpcNotification{
		Jsonrpc: "2.0",
		Method:  "notifications/initialized",
	}
	raw, _ := json.Marshal(initialized)
	c.sendRaw(raw)

	return &result, nil
}

func (c *Client) ListTools() ([]Tool, error) {
	req := JSONRPCRequest{
		Jsonrpc: "2.0",
		Method:  "tools/list",
		ID:      2,
	}
	resp, err := c.sendRequest(req)
	if err != nil {
		return nil, err
	}
	var list struct {
		Tools []Tool `json:"tools"`
	}
	if err := json.Unmarshal(resp.Result, &list); err != nil {
		return nil, fmt.Errorf("parse tools/list: %w", err)
	}
	return list.Tools, nil
}

func (c *Client) CallTool(name string, args map[string]interface{}) (*CallToolResult, error) {
	if args == nil {
		args = map[string]interface{}{}
	}
	params, _ := json.Marshal(CallToolParams{Name: name, Arguments: args})
	req := JSONRPCRequest{
		Jsonrpc: "2.0",
		Method:  "tools/call",
		Params:  params,
	}
	resp, err := c.sendRequest(req)
	if err != nil {
		return nil, err
	}
	var result CallToolResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("parse tools/call: %w", err)
	}
	return &result, nil
}

// Ping sends a keep-alive check to the server.
func (c *Client) Ping() error {
	req := JSONRPCRequest{
		Jsonrpc: "2.0",
		Method:  "ping",
		ID:      3,
	}
	_, err := c.sendRequest(req)
	return err
}

// SupportsResources reports whether the remote server advertised the
// resources capability during initialize.
func (c *Client) SupportsResources() bool {
	return c.capabilities.Resources != nil
}

// SupportsPrompts reports whether the remote server advertised the prompts
// capability during initialize.
func (c *Client) SupportsPrompts() bool {
	return c.capabilities.Prompts != nil
}

func (c *Client) ListResources() ([]Resource, error) {
	req := JSONRPCRequest{Jsonrpc: "2.0", Method: "resources/list", ID: 10}
	resp, err := c.sendRequest(req)
	if err != nil {
		return nil, err
	}
	var list struct {
		Resources []Resource `json:"resources"`
	}
	if err := json.Unmarshal(resp.Result, &list); err != nil {
		return nil, fmt.Errorf("parse resources/list: %w", err)
	}
	return list.Resources, nil
}

func (c *Client) ReadResource(uri string) (*ReadResourceResult, error) {
	params, _ := json.Marshal(ReadResourceParams{URI: uri})
	req := JSONRPCRequest{Jsonrpc: "2.0", Method: "resources/read", Params: params}
	resp, err := c.sendRequest(req)
	if err != nil {
		return nil, err
	}
	var result ReadResourceResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("parse resources/read: %w", err)
	}
	return &result, nil
}

func (c *Client) ListPrompts() ([]Prompt, error) {
	req := JSONRPCRequest{Jsonrpc: "2.0", Method: "prompts/list", ID: 11}
	resp, err := c.sendRequest(req)
	if err != nil {
		return nil, err
	}
	var list struct {
		Prompts []Prompt `json:"prompts"`
	}
	if err := json.Unmarshal(resp.Result, &list); err != nil {
		return nil, fmt.Errorf("parse prompts/list: %w", err)
	}
	return list.Prompts, nil
}

func (c *Client) GetPrompt(name string, args map[string]string) (*GetPromptResult, error) {
	if args == nil {
		args = map[string]string{}
	}
	params, _ := json.Marshal(GetPromptParams{Name: name, Arguments: args})
	req := JSONRPCRequest{Jsonrpc: "2.0", Method: "prompts/get", Params: params}
	resp, err := c.sendRequest(req)
	if err != nil {
		return nil, err
	}
	var result GetPromptResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("parse prompts/get: %w", err)
	}
	return &result, nil
}

// --- internal ---

type jsonrpcNotification struct {
	Jsonrpc string `json:"jsonrpc"`
	Method  string `json:"method"`
}

type rawResponse struct {
	Jsonrpc string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcErrorDetail `json:"error,omitempty"`
}

func (c *Client) sendRequest(req JSONRPCRequest) (*rawResponse, error) {
	id := c.nextID
	c.nextID++
	req.ID = id

	raw, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	ch := make(chan json.RawMessage, 1)
	c.pendMu.Lock()
	c.pendRes[id] = ch
	c.pendMu.Unlock()

	if err := c.sendRaw(raw); err != nil {
		c.pendMu.Lock()
		delete(c.pendRes, id)
		c.pendMu.Unlock()
		return nil, err
	}

	timeout := c.callTimeout
	if timeout <= 0 {
		timeout = defaultCallTimeout
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	var data []byte
	select {
	case data = <-ch:
	case <-timer.C:
		c.pendMu.Lock()
		delete(c.pendRes, id)
		c.pendMu.Unlock()
		return nil, fmt.Errorf("call timed out after %s%s", timeout, c.stderrSuffix())
	}

	var resp rawResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("rpc error %d: %s", resp.Error.Code, resp.Error.Message)
	}

	return &resp, nil
}

func (c *Client) stderrSuffix() string {
	if c.stderr != nil {
		if tail := c.stderr.String(); tail != "" {
			return "; server stderr: " + tail
		}
	}
	return ""
}

func (c *Client) sendRaw(raw []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return fmt.Errorf("client closed")
	}
	switch c.mode {
	case "stdio":
		_, err := c.stdin.Write(append(raw, '\n'))
		return err
	case "http":
		return c.httpPostLocked(raw)
	default:
		return fmt.Errorf("unknown mode: %s", c.mode)
	}
}

// httpPostLocked performs a single Streamable HTTP POST and routes the
// resulting SSE (or JSON) response events to their pending requests. Called
// with c.mu held.
func (c *Client) httpPostLocked(raw []byte) error {
	body := bytes.NewReader(raw)
	httpReq, err := http.NewRequest("POST", c.httpURL, body)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json, text/event-stream")
	if c.sessionID != "" {
		httpReq.Header.Set("Mcp-Session-Id", c.sessionID)
	}
	for k, v := range c.httpHeaders {
		httpReq.Header.Set(k, v)
	}

	if c.NetworkGate != nil {
		if gateErr := c.NetworkGate(httpReq.URL.String()); gateErr != nil {
			return gateErr
		}
	}

	timeout := c.callTimeout
	if timeout <= 0 {
		timeout = defaultCallTimeout
	}
	client := &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if c.NetworkGate == nil {
				return nil
			}
			return c.NetworkGate(req.URL.String())
		},
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		if ne, ok := err.(net.Error); ok && ne.Timeout() {
			return fmt.Errorf("call timed out after %s: %w", timeout, err)
		}
		return fmt.Errorf("http post: %w", err)
	}
	defer resp.Body.Close()

	if sid := resp.Header.Get("Mcp-Session-Id"); sid != "" {
		c.sessionID = sid
	}

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("http %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/event-stream") {
		return c.readSSELocked(resp.Body)
	}

	// Plain JSON-RPC response body (some servers only return JSON).
	return c.routeRawMessage(resp.Body)
}

// readSSELocked parses an SSE stream and routes each data event. It stops
// once a response for a pending request has been delivered.
func (c *Client) readSSELocked(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	delivered := false
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimPrefix(line, "data:")
		payload = strings.TrimSpace(payload)
		if payload == "" {
			continue
		}
		if delivered && c.noPending() {
			break
		}
		if done := c.deliver(payload); done {
			delivered = true
		}
	}
	return nil
}

// routeRawMessage reads a single JSON-RPC body and routes it.
func (c *Client) routeRawMessage(r io.Reader) error {
	body, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}
	if len(bytes.TrimSpace(body)) > 0 {
		c.deliver(string(body))
	}
	return nil
}

// deliver routes a raw JSON-RPC message to a pending request by ID. Returns
// true if it was a response (as opposed to a notification).
func (c *Client) deliver(raw string) bool {
	var hdr struct {
		ID     any    `json:"id"`
		Method string `json:"method"`
	}
	if err := json.Unmarshal([]byte(raw), &hdr); err != nil {
		return false
	}
	if hdr.ID == nil {
		return false // notification, ignore for now
	}
	id, ok := anyToInt(hdr.ID)
	if !ok {
		return false
	}
	c.pendMu.Lock()
	ch, exists := c.pendRes[id]
	if exists {
		delete(c.pendRes, id)
	}
	c.pendMu.Unlock()
	if exists {
		ch <- []byte(raw)
		return true
	}
	return false
}

func (c *Client) noPending() bool {
	c.pendMu.Lock()
	defer c.pendMu.Unlock()
	return len(c.pendRes) == 0
}

func (c *Client) readLoop() {
	for c.stdout.Scan() {
		line := c.stdout.Bytes()
		if len(line) == 0 {
			continue
		}
		c.deliver(string(line))
	}
	// Close all pending channels on EOF
	c.pendMu.Lock()
	for id, ch := range c.pendRes {
		close(ch)
		delete(c.pendRes, id)
	}
	c.pendMu.Unlock()
}

func anyToInt(v any) (int, bool) {
	switch x := v.(type) {
	case float64:
		return int(x), true
	case int:
		return x, true
	case int64:
		return int(x), true
	case string:
		n := 0
		_, err := fmt.Sscanf(x, "%d", &n)
		return n, err == nil
	default:
		return 0, false
	}
}

// limitedBuffer keeps only the last max bytes written to it.
type limitedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
	max int
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	n := len(p)
	if over := b.buf.Len() + len(p) - b.max; over > 0 {
		if over >= b.buf.Len() {
			b.buf.Reset()
		} else {
			b.buf.Next(over)
		}
	}
	b.buf.Write(p)
	return n, nil
}

func (b *limitedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func jsonToRaw(v any) json.RawMessage {
	data, _ := json.Marshal(v)
	return json.RawMessage(data)
}
