package mcp

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

type Server struct {
	name    string
	version string
	tools   []ToolInfo
	toolMap map[string]*ToolInfo

	resources         []Resource
	resourceHandlers  map[string]ResourceHandler
	resourceTemplates []ResourceTemplate
	templateHandlers  map[string]ResourceHandler

	prompts        []Prompt
	promptHandlers map[string]PromptHandler
	promptComplete map[string]func(argName, value string) []string
}

type ToolInfo struct {
	Def     ToolDef
	Handler ToolHandler
	Schema  any
}

func NewServer(name, version string) *Server {
	return &Server{
		name:             name,
		version:          version,
		tools:            make([]ToolInfo, 0),
		toolMap:          make(map[string]*ToolInfo),
		resourceHandlers: make(map[string]ResourceHandler),
		templateHandlers: make(map[string]ResourceHandler),
		promptHandlers:   make(map[string]PromptHandler),
		promptComplete:   make(map[string]func(argName, value string) []string),
	}
}

func (s *Server) AddTool(def ToolDef, handler ToolHandler) {
	if !validToolName(def.Name) {
		return
	}
	info := ToolInfo{Def: def, Handler: handler}
	s.tools = append(s.tools, info)
	s.toolMap[def.Name] = &s.tools[len(s.tools)-1]
}

// validToolName enforces the MCP spec's tool name pattern
// ^[a-zA-Z0-9_-]{1,64}$.
func validToolName(name string) bool {
	if len(name) == 0 || len(name) > 64 {
		return false
	}
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			continue
		}
		return false
	}
	return true
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

// AddResource registers a static resource.
func (s *Server) AddResource(r Resource, handler ResourceHandler) {
	if r.URI == "" {
		return
	}
	s.resources = append(s.resources, r)
	s.resourceHandlers[r.URI] = handler
}

// AddResourceTemplate registers a URI-template resource.
func (s *Server) AddResourceTemplate(t ResourceTemplate, handler ResourceHandler) {
	if t.URITemplate == "" {
		return
	}
	s.resourceTemplates = append(s.resourceTemplates, t)
	s.templateHandlers[t.URITemplate] = handler
}

// AddPrompt registers a prompt template. complete is optional and used for
// argument completion (may be nil).
func (s *Server) AddPrompt(p Prompt, handler PromptHandler, complete func(argName, value string) []string) {
	if p.Name == "" {
		return
	}
	s.prompts = append(s.prompts, p)
	s.promptHandlers[p.Name] = handler
	if complete != nil {
		s.promptComplete[p.Name] = complete
	}
}

func (s *Server) Resources() []Resource {
	return s.resources
}

func (s *Server) Prompts() []Prompt {
	return s.prompts
}

func (s *Server) DispatchRaw(data []byte) any {
	return s.dispatch(data)
}

func (s *Server) dispatch(data []byte) any {
	var req JSONRPCRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return makeError(nil, ErrCodeParseError, "Parse error: "+err.Error())
	}

	if req.Method == "" {
		if req.ID != nil {
			return makeError(req.ID, ErrCodeInvalidRequest, "Invalid Request: missing method")
		}
		return nil
	}

	switch req.Method {
	case "initialize":
		return s.handleInitialize(req.ID, req.Params)
	case "ping":
		return makeResponse(req.ID, map[string]interface{}{})
	case "tools/list":
		return s.handleToolsList(req.ID)
	case "tools/call":
		return s.handleToolsCall(req.ID, req.Params)
	case "resources/list":
		return s.handleResourcesList(req.ID)
	case "resources/templates/list":
		return s.handleResourceTemplatesList(req.ID)
	case "resources/read":
		return s.handleResourcesRead(req.ID, req.Params)
	case "prompts/list":
		return s.handlePromptsList(req.ID)
	case "prompts/get":
		return s.handlePromptsGet(req.ID, req.Params)
	case "completion/complete":
		return s.handleComplete(req.ID, req.Params)
	case "notifications/initialized":
		return nil
	default:
		if req.ID != nil {
			// Only respond if it was a request (has ID), not a notification.
			// Check if it IS a notification — method starts with "notifications/"
			if len(req.Method) > 14 && req.Method[:14] == "notifications/" {
				return nil
			}
			return makeError(req.ID, ErrCodeMethodNotFound, fmt.Sprintf("Method not found: %s", req.Method))
		}
		return nil
	}
}

// negotiateVersion picks the highest protocol version supported by both the
// client and this server. Unknown client versions fall back to the latest
// supported server version (per the MCP version negotiation rules).
func negotiateVersion(clientVersion string) string {
	if clientVersion != "" {
		for _, v := range ValidProtocolVersions {
			if v == clientVersion {
				return v
			}
		}
	}
	return LATEST_PROTOCOL_VERSION
}

func (s *Server) handleInitialize(id any, raw json.RawMessage) jsonrpcResponse {
	clientVersion := ""
	if len(raw) > 0 {
		var params InitializeParams
		if err := json.Unmarshal(raw, &params); err == nil {
			clientVersion = params.ProtocolVersion
		}
	}
	result := InitializeResult{
		ProtocolVersion: negotiateVersion(clientVersion),
		Capabilities: Capabilities{
			Tools:     &ToolsCapability{ListChanged: false},
			Resources: &ResourcesCapability{Subscribe: false, ListChanged: false},
			Prompts:   &PromptsCapability{ListChanged: false},
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
		return makeError(id, ErrCodeInvalidParams, "Invalid params: "+err.Error())
	}

	info, ok := s.toolMap[params.Name]
	if !ok {
		return makeError(id, ErrCodeInvalidParams, fmt.Sprintf("Unknown tool: %s", params.Name))
	}

	if params.Arguments == nil {
		params.Arguments = make(map[string]interface{})
	}

	for _, reqName := range requiredNames(info.Def) {
		if _, ok := params.Arguments[reqName]; !ok {
			return makeResponse(id, NewErrorResult(fmt.Sprintf("Missing required argument: %s", reqName)))
		}
	}

	text, err := info.Handler(params.Arguments)
	if err != nil {
		result := NewErrorResult(err.Error())
		return makeResponse(id, result)
	}
	return makeResponse(id, NewTextResult(text))
}

// requiredNames returns the required parameter names for a tool, preferring
// the raw JSON Schema's "required" list when present.
func requiredNames(def ToolDef) []string {
	if schema, ok := def.Schema.(map[string]interface{}); ok {
		if required, ok := schema["required"].([]interface{}); ok && len(required) > 0 {
			names := make([]string, 0, len(required))
			for _, r := range required {
				if s, ok := r.(string); ok {
					names = append(names, s)
				}
			}
			return names
		}
	}
	names := make([]string, 0, len(def.Params))
	for _, p := range def.Params {
		names = append(names, p.Name)
	}
	return names
}

// --- Resources ---

func (s *Server) handleResourcesList(id any) jsonrpcResponse {
	return makeResponse(id, map[string]interface{}{"resources": s.resources})
}

func (s *Server) handleResourceTemplatesList(id any) jsonrpcResponse {
	return makeResponse(id, map[string]interface{}{"resourceTemplates": s.resourceTemplates})
}

func (s *Server) handleResourcesRead(id any, raw json.RawMessage) any {
	var params ReadResourceParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return makeError(id, ErrCodeInvalidParams, "Invalid params: "+err.Error())
	}
	result, err := s.ReadResource(params.URI)
	if err != nil {
		return makeResponse(id, NewErrorResult(err.Error()))
	}
	return makeResponse(id, result)
}

// ReadResource resolves a URI (static resource or URI template) and returns
// its text contents.
func (s *Server) ReadResource(uri string) (*ReadResourceResult, error) {
	if uri == "" {
		return nil, fmt.Errorf("Missing required argument: uri")
	}

	handler, ok := s.resourceHandlers[uri]
	mime := ""
	if !ok {
		// Try template match (URI templates with {param} placeholders).
		for _, t := range s.resourceTemplates {
			if URITemplateMatches(uri, t.URITemplate) {
				handler = s.templateHandlers[t.URITemplate]
				mime = t.MIMEType
				ok = true
				break
			}
		}
	} else {
		for _, r := range s.resources {
			if r.URI == uri {
				mime = r.MIMEType
				break
			}
		}
	}

	if !ok {
		return nil, fmt.Errorf("Unknown resource: %s", uri)
	}

	text, err := handler(uri)
	if err != nil {
		return nil, err
	}
	return &ReadResourceResult{
		Contents: []ResourceContents{{
			URI:      uri,
			MIMEType: mime,
			Text:     text,
		}},
	}, nil
}

// URITemplateMatches reports whether uri matches a URI template like
// "file:///{path}" where {param} segments match any value.
func URITemplateMatches(uri, template string) bool {
	if !strings.Contains(template, "{") {
		return uri == template
	}
	var sb strings.Builder
	sb.WriteString("^")
	inVar := false
	for _, r := range template {
		switch {
		case r == '{':
			sb.WriteString("(?s:.*?)")
			inVar = true
		case r == '}':
			inVar = false
		case inVar:
			// skip variable name
		default:
			sb.WriteString(regexp.QuoteMeta(string(r)))
		}
	}
	sb.WriteString("$")
	matched, err := regexp.MatchString(sb.String(), uri)
	return err == nil && matched
}

// --- Prompts ---

func (s *Server) handlePromptsList(id any) jsonrpcResponse {
	return makeResponse(id, map[string]interface{}{"prompts": s.prompts})
}

func (s *Server) handlePromptsGet(id any, raw json.RawMessage) any {
	var params GetPromptParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return makeError(id, ErrCodeInvalidParams, "Invalid params: "+err.Error())
	}
	result, err := s.GetPrompt(params.Name, params.Arguments)
	if err != nil {
		return makeResponse(id, NewErrorResult(err.Error()))
	}
	return makeResponse(id, result)
}

// GetPrompt renders a prompt from its arguments.
func (s *Server) GetPrompt(name string, args map[string]string) (*GetPromptResult, error) {
	prompt, ok := s.findPrompt(name)
	if !ok {
		return nil, fmt.Errorf("Unknown prompt: %s", name)
	}

	if args == nil {
		args = make(map[string]string)
	}
	for _, a := range prompt.Arguments {
		if a.Required {
			if _, exists := args[a.Name]; !exists {
				return nil, fmt.Errorf("Missing required argument: %s", a.Name)
			}
		}
	}

	text, err := s.promptHandlers[name](args)
	if err != nil {
		return nil, err
	}
	return &GetPromptResult{
		Description: prompt.Description,
		Messages: []PromptMessage{{
			Role:    "user",
			Content: ContentItem{Type: "text", Text: text},
		}},
	}, nil
}

func (s *Server) findPrompt(name string) (Prompt, bool) {
	for _, p := range s.prompts {
		if p.Name == name {
			return p, true
		}
	}
	return Prompt{}, false
}

// --- Completion ---

func (s *Server) handleComplete(id any, raw json.RawMessage) any {
	var params CompleteRequest
	if err := json.Unmarshal(raw, &params); err != nil {
		return makeError(id, ErrCodeInvalidParams, "Invalid params: "+err.Error())
	}

	var values []string
	switch params.Ref.Type {
	case "ref/prompt":
		if complete, ok := s.promptComplete[params.Ref.Name]; ok {
			values = complete(params.Argument.Name, params.Argument.Value)
		}
	case "ref/resource":
		// No resource completion support yet.
	}

	if values == nil {
		values = []string{}
	}
	result := CompleteResult{}
	result.Completion.Values = values
	return makeResponse(id, result)
}
