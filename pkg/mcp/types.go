package mcp

import (
	"encoding/json"
	"fmt"
)

// --- Protocol versions ---

const LATEST_PROTOCOL_VERSION = "2025-11-25"

var ValidProtocolVersions = []string{
	"2025-11-25",
	"2025-06-18",
	"2025-03-26",
	"2024-11-05",
}

// --- JSON-RPC error codes (see JSON-RPC 2.0 + MCP spec) ---

const (
	ErrCodeParseError     = -32700
	ErrCodeInvalidRequest = -32600
	ErrCodeMethodNotFound = -32601
	ErrCodeInvalidParams  = -32602
	ErrCodeInternalError  = -32603
)

// --- JSON-RPC 2.0 ---

type JSONRPCRequest struct {
	Jsonrpc string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonrpcResponse struct {
	Jsonrpc string `json:"jsonrpc"`
	ID      any    `json:"id,omitempty"`
	Result  any    `json:"result,omitempty"`
}

type jsonrpcError struct {
	Jsonrpc string         `json:"jsonrpc"`
	ID      any            `json:"id,omitempty"`
	Error   rpcErrorDetail `json:"error"`
}

type rpcErrorDetail struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func makeResponse(id any, result any) jsonrpcResponse {
	return jsonrpcResponse{Jsonrpc: "2.0", ID: id, Result: result}
}

func makeError(id any, code int, msg string) jsonrpcError {
	return jsonrpcError{
		Jsonrpc: "2.0",
		ID:      id,
		Error:   rpcErrorDetail{Code: code, Message: msg},
	}
}

// --- MCP Initialize ---

type InitializeParams struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ClientCapabilities `json:"capabilities"`
	ClientInfo      struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"clientInfo"`
}

type ClientCapabilities struct {
}

type InitializeResult struct {
	ProtocolVersion string       `json:"protocolVersion"`
	Capabilities    Capabilities `json:"capabilities"`
	ServerInfo      ServerInfo   `json:"serverInfo"`
}

type Capabilities struct {
	Tools     *ToolsCapability     `json:"tools,omitempty"`
	Resources *ResourcesCapability `json:"resources,omitempty"`
	Prompts   *PromptsCapability   `json:"prompts,omitempty"`
}

type ToolsCapability struct {
	ListChanged bool `json:"listChanged"`
}

type ResourcesCapability struct {
	Subscribe   bool `json:"subscribe"`
	ListChanged bool `json:"listChanged"`
}

type PromptsCapability struct {
	ListChanged bool `json:"listChanged"`
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// --- MCP Tools ---

type Tool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema any    `json:"inputSchema"`
}

type CallToolParams struct {
	Name string `json:"name"`
	// No omitempty: some MCP servers (notably ones using the official SDK's
	// zod-based tool schemas, e.g. mcp-docker-server, time-mcp) call
	// schema.parse(request.params.arguments) directly and reject a missing
	// "arguments" key with "expected object, received undefined" — even for
	// zero-parameter tools where an empty object would validate fine. The
	// caller (builtins_mcp.go) always builds a non-nil map, so this never
	// regresses a genuine "no arguments at all" case into an accidental
	// "arguments: null".
	Arguments map[string]interface{} `json:"arguments"`
}

type CallToolResult struct {
	Content []ContentItem `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

type ContentItem struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// --- MCP Resources ---

type Resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MIMEType    string `json:"mimeType,omitempty"`
	Size        int64  `json:"size,omitempty"`
}

type ResourceTemplate struct {
	URITemplate string `json:"uriTemplate"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MIMEType    string `json:"mimeType,omitempty"`
}

type ReadResourceParams struct {
	URI string `json:"uri"`
}

type ResourceContents struct {
	URI      string `json:"uri"`
	MIMEType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
}

type ReadResourceResult struct {
	Contents []ResourceContents `json:"contents"`
}

// --- MCP Prompts ---

type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

type Prompt struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Arguments   []PromptArgument `json:"arguments,omitempty"`
}

type GetPromptParams struct {
	Name string `json:"name"`
	// Same "omitempty drops the key entirely, some servers reject a missing
	// arguments field" issue as CallToolParams above — kept consistent.
	Arguments map[string]string `json:"arguments"`
}

type PromptMessage struct {
	Role    string      `json:"role"`
	Content ContentItem `json:"content"`
}

type GetPromptResult struct {
	Description string          `json:"description,omitempty"`
	Messages    []PromptMessage `json:"messages"`
}

// --- Completion ---

type CompleteRequest struct {
	Ref struct {
		Type string `json:"type"`
		Name string `json:"name"`
		URI  string `json:"uri"`
	} `json:"ref"`
	Argument struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	} `json:"argument"`
}

type CompleteResult struct {
	Completion struct {
		Values  []string `json:"values"`
		HasMore bool     `json:"hasMore,omitempty"`
		Total   int      `json:"total,omitempty"`
	} `json:"completion"`
}

// --- Tool Registry (caller-provided) ---

type ParamDef struct {
	Name        string
	Description string
}

type ToolDef struct {
	Name        string
	Description string
	Params      []ParamDef
	Schema      any // raw JSON Schema for inputSchema (optional; if nil, paramsToSchema is used)
}

// ToolHandler receives parsed arguments and returns text result + optional error.
// If error is non-nil, the tool result is marked isError: true.
// args are passed in the same order as ToolDef.Params.
type ToolHandler func(args map[string]interface{}) (string, error)

// ResourceHandler returns the text contents of a resource for a concrete URI.
type ResourceHandler func(uri string) (string, error)

// PromptHandler renders a prompt from its arguments (map argument name -> value).
type PromptHandler func(args map[string]string) (string, error)

func NewTextResult(text string) *CallToolResult {
	return &CallToolResult{
		Content: []ContentItem{{Type: "text", Text: text}},
	}
}

func NewErrorResult(msg string) *CallToolResult {
	return &CallToolResult{
		Content: []ContentItem{{Type: "text", Text: msg}},
		IsError: true,
	}
}

func formatArgs(args map[string]interface{}) string {
	data, err := json.Marshal(args)
	if err != nil {
		return fmt.Sprintf("{%v}", args)
	}
	return string(data)
}
