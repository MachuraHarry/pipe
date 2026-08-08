package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
)

type Client struct {
	name     string
	mu       sync.Mutex
	nextID   int
	stdin    io.WriteCloser
	stdout   *bufio.Scanner
	cmd      *exec.Cmd
	mode     string   // "stdio" or "http"
	httpURL  string
	pendRes  map[int]chan json.RawMessage
	pendMu   sync.Mutex
	closed   bool
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
	cmd.Stderr = nil // discard stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("mcp client start: %w", err)
	}

	c := &Client{
		name:   command + " " + strings.Join(args, " "),
		mode:   "stdio",
		stdin:  stdin,
		stdout: bufio.NewScanner(stdout),
		cmd:    cmd,
		pendRes: make(map[int]chan json.RawMessage),
	}
	c.stdout.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)
	go c.readLoop()
	return c, nil
}

func NewHTTPClient(url string) (*Client, error) {
	c := &Client{
		name:   url,
		mode:   "http",
		httpURL: url,
		pendRes: make(map[int]chan json.RawMessage),
	}
	return c, nil
}

func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	if c.stdin != nil {
		c.stdin.Close()
	}
	if c.cmd != nil {
		c.cmd.Wait()
	}
	return nil
}

func (c *Client) Initialize() (*InitializeResult, error) {
	req := JSONRPCRequest{
		Jsonrpc: "2.0",
		Method:  "initialize",
		ID:      1,
		Params: jsonToRaw(InitializeParams{
			ProtocolVersion: "2025-11-25",
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

	data, ok := <-ch
	if !ok {
		return nil, fmt.Errorf("client closed")
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
		resp, err := http.Post(c.httpURL, "application/json", strings.NewReader(string(raw)+"\n"))
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 400 {
			return fmt.Errorf("http %d", resp.StatusCode)
		}
		return nil
	default:
		return fmt.Errorf("unknown mode: %s", c.mode)
	}
}

func (c *Client) readLoop() {
	for c.stdout.Scan() {
		line := c.stdout.Bytes()
		if len(line) == 0 {
			continue
		}
		var hdr struct {
			ID     any  `json:"id"`
			Result json.RawMessage `json:"result"`
			Method string `json:"method"`
		}
		if err := json.Unmarshal(line, &hdr); err != nil {
			continue
		}

		// Check if it's a response (has id but no method) vs notification (has method)
		if hdr.ID != nil && hdr.Method == "" {
			c.pendMu.Lock()
			if ch, ok := c.pendRes[_anyToInt(hdr.ID)]; ok {
				ch <- line
				delete(c.pendRes, _anyToInt(hdr.ID))
			}
			c.pendMu.Unlock()
		}
	}
	// Close all pending channels on EOF
	c.pendMu.Lock()
	for id, ch := range c.pendRes {
		close(ch)
		delete(c.pendRes, id)
	}
	c.pendMu.Unlock()
}

func _anyToInt(v any) int {
	switch x := v.(type) {
	case float64:
		return int(x)
	case int:
		return x
	default:
		return 0
	}
}

func jsonToRaw(v any) json.RawMessage {
	data, _ := json.Marshal(v)
	return json.RawMessage(data)
}
