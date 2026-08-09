package mcp

import (
	"encoding/json"
	"fmt"
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
	Tools *ToolsCapability `json:"tools,omitempty"`
}

type ToolsCapability struct {
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
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

type CallToolResult struct {
	Content []ContentItem `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

type ContentItem struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
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
