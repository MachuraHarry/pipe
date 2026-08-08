package mcp

import (
	"encoding/json"
	"fmt"
)

type Server struct {
	name    string
	version string
	tools   []ToolInfo
	toolMap map[string]*ToolInfo
}

type ToolInfo struct {
	Def     ToolDef
	Handler ToolHandler
	Schema  any
}

func NewServer(name, version string) *Server {
	return &Server{
		name:    name,
		version: version,
		tools:   make([]ToolInfo, 0),
		toolMap: make(map[string]*ToolInfo),
	}
}

func (s *Server) AddTool(def ToolDef, handler ToolHandler) {
	info := ToolInfo{Def: def, Handler: handler}
	s.tools = append(s.tools, info)
	s.toolMap[def.Name] = &s.tools[len(s.tools)-1]
}

func (s *Server) SetToolSchema(name string, schema any) {
	if info, ok := s.toolMap[name]; ok {
		info.Schema = schema
	}
}

func (s *Server) Tools() []ToolDef {
	result := make([]ToolDef, len(s.tools))
	for i, ti := range s.tools {
		result[i] = ti.Def
	}
	return result
}

func (s *Server) DispatchRaw(data []byte) any {
	return s.dispatch(data)
}

func (s *Server) dispatch(data []byte) any {
	var req JSONRPCRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return makeError(nil, -32700, "Parse error: "+err.Error())
	}

	if req.Method == "" {
		if req.ID != nil {
			return makeError(req.ID, -32600, "Invalid Request: missing method")
		}
		return nil
	}

	switch req.Method {
	case "initialize":
		return s.handleInitialize(req.ID)
	case "tools/list":
		return s.handleToolsList(req.ID)
	case "tools/call":
		return s.handleToolsCall(req.ID, req.Params)
	case "notifications/initialized":
		return nil
	default:
		if req.ID != nil {
			// Only respond if it was a request (has ID), not a notification.
			// Check if it IS a notification — method starts with "notifications/"
			if len(req.Method) > 14 && req.Method[:14] == "notifications/" {
				return nil
			}
			return makeError(req.ID, -32601, fmt.Sprintf("Method not found: %s", req.Method))
		}
		return nil
	}
}

func (s *Server) handleInitialize(id any) jsonrpcResponse {
	result := InitializeResult{
		ProtocolVersion: "2025-11-25",
		Capabilities: Capabilities{
			Tools: &ToolsCapability{ListChanged: false},
		},
		ServerInfo: ServerInfo{
			Name:    s.name,
			Version: s.version,
		},
	}
	return makeResponse(id, result)
}

func (s *Server) handleToolsList(id any) jsonrpcResponse {
	tools := make([]Tool, len(s.tools))
	for i, ti := range s.tools {
		schema := ti.Def.Schema
		if schema == nil {
			schema = paramsToSchema(ti.Def.Params)
		}
		tools[i] = Tool{
			Name:        ti.Def.Name,
			Description: ti.Def.Description,
			InputSchema: schema,
		}
	}
	return makeResponse(id, map[string]interface{}{"tools": tools})
}

func (s *Server) handleToolsCall(id any, raw json.RawMessage) any {
	var params CallToolParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return makeError(id, -32602, "Invalid params: "+err.Error())
	}

	info, ok := s.toolMap[params.Name]
	if !ok {
		return makeError(id, -32602, fmt.Sprintf("Unknown tool: %s", params.Name))
	}

	if params.Arguments == nil {
		params.Arguments = make(map[string]interface{})
	}

	text, err := info.Handler(params.Arguments)
	if err != nil {
		result := NewErrorResult(err.Error())
		return makeResponse(id, result)
	}
	return makeResponse(id, NewTextResult(text))
}
