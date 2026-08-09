package object

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/MachuraHarry/pipe/pkg/ai"
	"github.com/MachuraHarry/pipe/pkg/mcp"
)

var currentMCPServer *mcp.Server

// --- Local resource & prompt registries (bridged into the server) ---

type mcpResourceDef struct {
	def     mcp.Resource
	handler mcp.ResourceHandler
}

type mcpTemplateDef struct {
	def     mcp.ResourceTemplate
	handler mcp.ResourceHandler
}

type mcpPromptDef struct {
	def     mcp.Prompt
	handler mcp.PromptHandler
}

var resourceRegistry []mcpResourceDef
var templateRegistry []mcpTemplateDef
var promptRegistry []mcpPromptDef

func bMcpServer(args ...Object) Object {
	if len(args) < 2 {
		return err("mcp_server expects 2 arguments (name, version)")
	}
	name, ok := args[0].(*String)
	if !ok {
		return err("mcp_server: first argument must be a string (server name)")
	}
	version, ok := args[1].(*String)
	if !ok {
		return err("mcp_server: second argument must be a string (version)")
	}

	s := mcp.NewServer(name.Value, version.Value)

	// Bridge all locally registered resources.
	for _, r := range resourceRegistry {
		s.AddResource(r.def, r.handler)
	}
	for _, t := range templateRegistry {
		s.AddResourceTemplate(t.def, t.handler)
	}
	for _, p := range promptRegistry {
		s.AddPrompt(p.def, p.handler, nil)
	}

	for _, entry := range toolRegistry {
		entry := entry

		props := make(map[string]string)
		if properties, ok := entry.Def.Parameters["properties"].(map[string]interface{}); ok {
			for pn, pv := range properties {
				if pm, ok := pv.(map[string]interface{}); ok {
					if d, ok := pm["description"]; ok {
						if ds, ok := d.(string); ok {
							props[pn] = ds
						} else {
							props[pn] = pn
						}
					} else {
						props[pn] = pn
					}
				} else {
					props[pn] = pn
				}
			}
		}

		paramNames := make([]string, 0, len(props))
		params := make([]mcp.ParamDef, 0, len(props))
		if required, ok := entry.Def.Parameters["required"].([]interface{}); ok {
			for _, r := range required {
				if rn, ok := r.(string); ok {
					desc := props[rn]
					if desc == "" {
						desc = rn
					}
					paramNames = append(paramNames, rn)
					params = append(params, mcp.ParamDef{Name: rn, Description: desc})
				}
			}
		}
		if len(params) == 0 {
			for pn, desc := range props {
				paramNames = append(paramNames, pn)
				params = append(params, mcp.ParamDef{Name: pn, Description: desc})
			}
		}

		def := mcp.ToolDef{
			Name:        entry.Def.Name,
			Description: entry.Def.Description,
			Params:      params,
			Schema:      entry.Def.Parameters,
		}
		s.AddTool(def, func(args map[string]interface{}) (string, error) {
			argObjects := make([]Object, 0, len(paramNames))
			for _, pn := range paramNames {
				v, found := args[pn]
				if !found {
					continue
				}
				argObjects = append(argObjects, interfaceToObject(v))
			}

			toolExecMu.Lock()
			defer toolExecMu.Unlock()

			if callUserFn != nil {
				result := callUserFn(entry.Fn, argObjects...)
				if e, isErr := result.(*Error); isErr {
					return e.Message, fmt.Errorf("%s", e.Message)
				}
				return toolResultText(result), nil
			}

			return "", fmt.Errorf("tool execution not available")
		})
	}

	currentMCPServer = s
	return NILOBJ
}

func bMcpServeStdio(args ...Object) Object {
	if currentMCPServer == nil {
		return err("mcp_serve_stdio: no MCP server created. Call mcp_server first.")
	}
	if len(currentMCPServer.Tools()) == 0 {
		return err("mcp_serve_stdio: no tools registered. Call mcp_server after registering tools with ai_tool.")
	}
	if serveErr := currentMCPServer.ServeStdio(); serveErr != nil {
		return err("mcp_serve_stdio: " + serveErr.Error())
	}
	return NILOBJ
}

func bMcpServeSSE(args ...Object) Object {
	if len(args) < 1 {
		return err("mcp_serve_sse expects 1 argument (addr, e.g. ':9090')")
	}
	addr, ok := args[0].(*String)
	if !ok {
		return err("mcp_serve_sse: argument must be a string")
	}
	if currentMCPServer == nil {
		return err("mcp_serve_sse: no MCP server created. Call mcp_server first.")
	}
	if len(currentMCPServer.Tools()) == 0 {
		return err("mcp_serve_sse: no tools registered. Call mcp_server after registering tools with ai_tool.")
	}
	if serveErr := currentMCPServer.ServeHTTP(addr.Value); serveErr != nil {
		return err("mcp_serve_sse: " + serveErr.Error())
	}
	return NILOBJ
}

func bMcpTools(args ...Object) Object {
	elems := make([]Object, 0)

	// Local MCP server tools
	if currentMCPServer != nil {
		for _, t := range currentMCPServer.Tools() {
			elems = append(elems, &Map{Pairs: map[string]Object{
				"name":        &String{Value: t.Name},
				"description": &String{Value: t.Description},
				"source":      &String{Value: "local"},
			}})
		}
	}

	// MCP client tools
	for _, entry := range mcpClients {
		for localName, bridge := range entry.tools {
			elems = append(elems, &Map{Pairs: map[string]Object{
				"name":        &String{Value: localName},
				"description": &String{Value: "remote: " + bridge.remoteName},
				"source":      &String{Value: entry.prefix},
			}})
		}
	}

	if len(elems) == 0 {
		return err("mcp_tools: no tools available. Create a local server with mcp_server or connect to a remote one with mcp_use_stdio/mcp_use_sse.")
	}
	return &List{Elements: elems}
}

// --- MCP Resources & Prompts (local registration) ---

// mcp_resource(uri, name, mime, read_fn) registers a static resource.
// read_fn receives the requested URI and returns the resource text.
func bMcpResource(args ...Object) Object {
	if len(args) < 4 {
		return err("mcp_resource expects 4 arguments (uri, name, mime, read_fn)")
	}
	uri, ok1 := args[0].(*String)
	name, ok2 := args[1].(*String)
	mime, ok3 := args[2].(*String)
	if !ok1 || !ok2 || !ok3 {
		return err("mcp_resource: uri, name and mime must be strings")
	}

	resourceRegistry = append(resourceRegistry, mcpResourceDef{
		def: mcp.Resource{URI: uri.Value, Name: name.Value, MIMEType: mime.Value},
		handler: func(u string) (string, error) {
			return runResourceFn(args[3], u)
		},
	})
	return NILOBJ
}

// mcp_resource_template(uri_template, name, mime, read_fn) registers a
// URI-template resource, e.g. "file:///{path}". read_fn receives the concrete
// URI.
func bMcpResourceTemplate(args ...Object) Object {
	if len(args) < 4 {
		return err("mcp_resource_template expects 4 arguments (uri_template, name, mime, read_fn)")
	}
	tmpl, ok1 := args[0].(*String)
	name, ok2 := args[1].(*String)
	mime, ok3 := args[2].(*String)
	if !ok1 || !ok2 || !ok3 {
		return err("mcp_resource_template: uri_template, name and mime must be strings")
	}

	templateRegistry = append(templateRegistry, mcpTemplateDef{
		def: mcp.ResourceTemplate{URITemplate: tmpl.Value, Name: name.Value, MIMEType: mime.Value},
		handler: func(u string) (string, error) {
			return runResourceFn(args[3], u)
		},
	})
	return NILOBJ
}

// mcp_prompt(name, description, args_map, build_fn) registers a prompt
// template. args_map maps argument names to a description (or to a map with
// "description" and optional "required"). build_fn receives the argument map
// and returns the rendered prompt text.
func bMcpPrompt(args ...Object) Object {
	if len(args) < 4 {
		return err("mcp_prompt expects 4 arguments (name, description, args_map, build_fn)")
	}
	name, ok1 := args[0].(*String)
	desc, ok2 := args[1].(*String)
	argsMap, ok3 := args[2].(*Map)
	if !ok1 || !ok2 || !ok3 {
		return err("mcp_prompt: name, description and args_map are required")
	}

	arguments := make([]mcp.PromptArgument, 0, len(argsMap.Pairs))
	for k, v := range argsMap.Pairs {
		pa := mcp.PromptArgument{Name: k, Required: true}
		if s, ok := v.(*String); ok {
			pa.Description = s.Value
		} else if m, ok := v.(*Map); ok {
			if d, ok := m.Pairs["description"]; ok {
				if ds, ok := d.(*String); ok {
					pa.Description = ds.Value
				}
			}
			if r, ok := m.Pairs["required"]; ok {
				if rb, ok := r.(*Boolean); ok {
					pa.Required = rb.Value
				}
			}
		}
		arguments = append(arguments, pa)
	}

	promptRegistry = append(promptRegistry, mcpPromptDef{
		def: mcp.Prompt{Name: name.Value, Description: desc.Value, Arguments: arguments},
		handler: func(a map[string]string) (string, error) {
			pairs := make(map[string]Object, len(a))
			for k, v := range a {
				pairs[k] = &String{Value: v}
			}
			toolExecMu.Lock()
			defer toolExecMu.Unlock()
			result := callFunctionObject(args[3], &Map{Pairs: pairs})
			if e, isErr := result.(*Error); isErr {
				return e.Message, fmt.Errorf("%s", e.Message)
			}
			return toolResultText(result), nil
		},
	})
	return NILOBJ
}

func runResourceFn(fn Object, uri string) (string, error) {
	toolExecMu.Lock()
	defer toolExecMu.Unlock()
	result := callFunctionObject(fn, &String{Value: uri})
	if e, isErr := result.(*Error); isErr {
		return e.Message, fmt.Errorf("%s", e.Message)
	}
	return toolResultText(result), nil
}

// callFunctionObject invokes a user function (or a test-level *BuiltinInfo)
// through the eval context.
func callFunctionObject(fn Object, args ...Object) Object {
	if callUserFn != nil {
		return callUserFn(fn, args...)
	}
	if bi, ok := fn.(*BuiltinInfo); ok {
		return bi.Fn(args...)
	}
	return err("function not callable")
}

func registryResourceResult(handler mcp.ResourceHandler, uri string) Object {
	text, err := handler(uri)
	if err != nil {
		return &Error{Message: "mcp_read_resource: " + err.Error()}
	}
	return &String{Value: text}
}

func uriMatchesTemplate(uri, tmpl string) bool {
	return mcp.URITemplateMatches(uri, tmpl)
}

// mcp_resources lists all registered resources (local + remote).
func bMcpResources(args ...Object) Object {
	elems := make([]Object, 0)

	if currentMCPServer != nil {
		for _, r := range currentMCPServer.Resources() {
			elems = append(elems, &Map{Pairs: map[string]Object{
				"uri":         &String{Value: r.URI},
				"name":        &String{Value: r.Name},
				"mimeType":    &String{Value: r.MIMEType},
				"description": &String{Value: r.Description},
				"source":      &String{Value: "local"},
			}})
		}
	} else {
		// No server active: list the local registries directly.
		for _, r := range resourceRegistry {
			elems = append(elems, &Map{Pairs: map[string]Object{
				"uri":         &String{Value: r.def.URI},
				"name":        &String{Value: r.def.Name},
				"mimeType":    &String{Value: r.def.MIMEType},
				"description": &String{Value: r.def.Description},
				"source":      &String{Value: "local"},
			}})
		}
		for _, t := range templateRegistry {
			elems = append(elems, &Map{Pairs: map[string]Object{
				"uri":         &String{Value: t.def.URITemplate},
				"name":        &String{Value: t.def.Name},
				"mimeType":    &String{Value: t.def.MIMEType},
				"description": &String{Value: t.def.Description},
				"source":      &String{Value: "local"},
			}})
		}
	}
	for _, entry := range mcpClients {
		for _, r := range entry.resources {
			elems = append(elems, &Map{Pairs: map[string]Object{
				"uri":         &String{Value: r.URI},
				"name":        &String{Value: r.Name},
				"mimeType":    &String{Value: r.MIMEType},
				"description": &String{Value: r.Description},
				"source":      &String{Value: entry.prefix},
			}})
		}
	}

	if len(elems) == 0 {
		return err("mcp_resources: no resources registered.")
	}
	return &List{Elements: elems}
}

// mcp_read_resource(uri) reads a resource from the local server or a remote
// MCP client.
func bMcpReadResource(args ...Object) Object {
	if len(args) < 1 {
		return err("mcp_read_resource expects 1 argument (uri)")
	}
	uri, ok := args[0].(*String)
	if !ok {
		return err("mcp_read_resource: uri must be a string")
	}

	if currentMCPServer != nil {
		if result, err := currentMCPServer.ReadResource(uri.Value); err == nil {
			return &String{Value: resourceResultText(result)}
		}
	}
	// Fall back to the local registries (usable without a running server).
	for _, r := range resourceRegistry {
		if r.def.URI == uri.Value {
			return registryResourceResult(r.handler, uri.Value)
		}
	}
	for _, t := range templateRegistry {
		if uriMatchesTemplate(uri.Value, t.def.URITemplate) {
			return registryResourceResult(t.handler, uri.Value)
		}
	}
	for _, entry := range mcpClients {
		for _, r := range entry.resources {
			if r.URI == uri.Value {
				result, err := entry.client.ReadResource(uri.Value)
				if err != nil {
					return &Error{Message: "mcp_read_resource: " + err.Error()}
				}
				return &String{Value: resourceResultText(result)}
			}
		}
	}
	return &Error{Message: "mcp_read_resource: unknown resource: " + uri.Value}
}

// mcp_prompts lists all registered prompts (local + remote).
func bMcpPrompts(args ...Object) Object {
	elems := make([]Object, 0)

	if currentMCPServer != nil {
		for _, p := range currentMCPServer.Prompts() {
			elems = append(elems, &Map{Pairs: map[string]Object{
				"name":        &String{Value: p.Name},
				"description": &String{Value: p.Description},
				"source":      &String{Value: "local"},
			}})
		}
	} else {
		for _, p := range promptRegistry {
			elems = append(elems, &Map{Pairs: map[string]Object{
				"name":        &String{Value: p.def.Name},
				"description": &String{Value: p.def.Description},
				"source":      &String{Value: "local"},
			}})
		}
	}
	for _, entry := range mcpClients {
		for _, p := range entry.prompts {
			elems = append(elems, &Map{Pairs: map[string]Object{
				"name":        &String{Value: p.Name},
				"description": &String{Value: p.Description},
				"source":      &String{Value: entry.prefix},
			}})
		}
	}

	if len(elems) == 0 {
		return err("mcp_prompts: no prompts registered.")
	}
	return &List{Elements: elems}
}

// mcp_prompt_get(name, args_map?) renders a prompt from the local server or a
// remote MCP client.
func bMcpPromptGet(args ...Object) Object {
	if len(args) < 1 {
		return err("mcp_prompt_get expects at least 1 argument (name)")
	}
	name, ok := args[0].(*String)
	if !ok {
		return err("mcp_prompt_get: name must be a string")
	}

	argMap := make(map[string]string)
	if len(args) >= 2 {
		if m, ok := args[1].(*Map); ok {
			for k, v := range m.Pairs {
				if s, ok := v.(*String); ok {
					argMap[k] = s.Value
				} else {
					argMap[k] = v.Inspect()
				}
			}
		}
	}

	if currentMCPServer != nil {
		if result, err := currentMCPServer.GetPrompt(name.Value, argMap); err == nil {
			return &String{Value: promptResultText(result)}
		}
	}
	// Fall back to the local registries (usable without a running server).
	for _, p := range promptRegistry {
		if p.def.Name == name.Value {
			for _, a := range p.def.Arguments {
				if a.Required {
					if _, exists := argMap[a.Name]; !exists {
						return &Error{Message: "mcp_prompt_get: missing required argument: " + a.Name}
					}
				}
			}
			text, err := p.handler(argMap)
			if err != nil {
				return &Error{Message: "mcp_prompt_get: " + err.Error()}
			}
			return &String{Value: text}
		}
	}
	for _, entry := range mcpClients {
		for _, p := range entry.prompts {
			if p.Name == name.Value {
				result, err := entry.client.GetPrompt(name.Value, argMap)
				if err != nil {
					return &Error{Message: "mcp_prompt_get: " + err.Error()}
				}
				return &String{Value: promptResultText(result)}
			}
		}
	}
	return &Error{Message: "mcp_prompt_get: unknown prompt: " + name.Value}
}

// --- MCP Client ---

var mcpClients []*mcpClientEntry

// toolExecMu serializes tool execution. The Pipe interpreter/VM is not
// thread-safe, so concurrent sessions (e.g. multiple HTTP clients) must not
// call user functions in parallel.
var toolExecMu sync.Mutex

type mcpClientEntry struct {
	client    *mcp.Client
	prefix    string
	tools     map[string]*mcpBridgeInfo
	resources []mcp.Resource
	prompts   []mcp.Prompt
}

type mcpBridgeInfo struct {
	remoteName string
	paramNames []string
}

// interfaceToObject converts a JSON-decoded value into a Pipe object.
func interfaceToObject(v interface{}) Object {
	switch val := v.(type) {
	case string:
		return &String{Value: val}
	case float64:
		if val == float64(int64(val)) {
			return &Integer{Value: int64(val)}
		}
		return &Float{Value: val}
	case bool:
		return NativeBoolToBoolean(val)
	case nil:
		return NILOBJ
	case []interface{}:
		elems := make([]Object, 0, len(val))
		for _, e := range val {
			elems = append(elems, interfaceToObject(e))
		}
		return &List{Elements: elems}
	case map[string]interface{}:
		pairs := make(map[string]Object, len(val))
		for k, ev := range val {
			pairs[k] = interfaceToObject(ev)
		}
		return &Map{Pairs: pairs}
	default:
		return &String{Value: fmt.Sprintf("%v", val)}
	}
}

// objectToInterface recursively converts a Pipe object into a plain JSON
// value for use in MCP tool arguments.
func objectToInterface(obj Object) interface{} {
	switch v := obj.(type) {
	case *String:
		return v.Value
	case *Integer:
		return v.Value
	case *Float:
		return v.Value
	case *Boolean:
		return v.Value
	case *Map:
		out := make(map[string]interface{}, len(v.Pairs))
		for k, val := range v.Pairs {
			out[k] = objectToInterface(val)
		}
		return out
	case *List:
		out := make([]interface{}, 0, len(v.Elements))
		for _, e := range v.Elements {
			out = append(out, objectToInterface(e))
		}
		return out
	default:
		return obj.Inspect()
	}
}

// toolResultText renders a tool result for the MCP client. Maps and Lists are
// serialized as pretty JSON so structured data survives the round trip.
func toolResultText(obj Object) string {
	switch obj.(type) {
	case *Map, *List:
		data, err := json.MarshalIndent(objToJSON(obj), "", "  ")
		if err == nil {
			return string(data)
		}
	}
	return obj.Inspect()
}

// objToJSON converts a Pipe object into a JSON-encodable value.
func objToJSON(obj Object) interface{} {
	switch v := obj.(type) {
	case *Map:
		out := make(map[string]interface{}, len(v.Pairs))
		for k, val := range v.Pairs {
			out[k] = objToJSON(val)
		}
		return out
	case *List:
		out := make([]interface{}, 0, len(v.Elements))
		for _, e := range v.Elements {
			out = append(out, objToJSON(e))
		}
		return out
	default:
		return objectToInterface(obj)
	}
}

func registerMCPClient(client *mcp.Client, prefix string) error {
	_, initErr := client.Initialize()
	if initErr != nil {
		client.Close()
		return fmt.Errorf("mcp client initialize: %w", initErr)
	}

	tools, listErr := client.ListTools()
	if listErr != nil {
		client.Close()
		return fmt.Errorf("mcp client tools/list: %w", listErr)
	}

	entry := &mcpClientEntry{
		client: client,
		prefix: prefix,
		tools:  make(map[string]*mcpBridgeInfo),
	}
	mcpClients = append(mcpClients, entry)

	// Discover resources and prompts if the server advertises them.
	if client.SupportsResources() {
		if resources, resErr := client.ListResources(); resErr == nil {
			entry.resources = resources
		}
	}
	if client.SupportsPrompts() {
		if prompts, pErr := client.ListPrompts(); pErr == nil {
			entry.prompts = prompts
		}
	}

	for _, tool := range tools {
		remoteName := tool.Name
		localName := prefix + remoteName

		// Extract param names in order from the JSON Schema
		paramNames := extractParamNames(tool.InputSchema)
		bridge := &mcpBridgeInfo{
			remoteName: remoteName,
			paramNames: paramNames,
		}
		entry.tools[localName] = bridge

		bi := &BuiltinInfo{
			Name: localName,
			Fn: func(args ...Object) Object {
				// Convert positional args to named map
				callArgs := make(map[string]interface{})
				for i, arg := range args {
					if i < len(bridge.paramNames) {
						callArgs[bridge.paramNames[i]] = objectToInterface(arg)
					}
				}
				result, callErr := entry.client.CallTool(bridge.remoteName, callArgs)
				if callErr != nil {
					return &Error{Message: "mcp call " + bridge.remoteName + ": " + callErr.Error()}
				}
				if result.IsError {
					return &Error{Message: "mcp call " + bridge.remoteName + ": " + resultText(result)}
				}
				return &String{Value: resultText(result)}
			},
		}

		toolRegistry[localName] = ToolEntry{
			Def: ai.ToolDef{
				Name:        localName,
				Description: tool.Description,
				Parameters:  schemaToParams(tool.InputSchema),
			},
			Fn: bi,
		}
	}

	return nil
}

func extractParamNames(inputSchema interface{}) []string {
	schema, ok := inputSchema.(map[string]interface{})
	if !ok {
		return nil
	}
	if required, ok := schema["required"].([]interface{}); ok {
		names := make([]string, 0, len(required))
		for _, r := range required {
			if s, ok := r.(string); ok {
				names = append(names, s)
			}
		}
		return names
	}
	return nil
}

func schemaToParams(inputSchema interface{}) map[string]interface{} {
	// ai.ToolDef uses the same format — return as-is
	schema, ok := inputSchema.(map[string]interface{})
	if !ok {
		return map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
			"required":   []interface{}{},
		}
	}
	return schema
}

func resultText(result *mcp.CallToolResult) string {
	if len(result.Content) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, c := range result.Content {
		if c.Type == "text" {
			sb.WriteString(c.Text)
		}
	}
	return sb.String()
}

func resourceResultText(result *mcp.ReadResourceResult) string {
	if len(result.Contents) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, c := range result.Contents {
		sb.WriteString(c.Text)
	}
	return sb.String()
}

func promptResultText(result *mcp.GetPromptResult) string {
	if len(result.Messages) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, m := range result.Messages {
		if m.Content.Type == "text" {
			sb.WriteString(m.Content.Text)
		}
	}
	return sb.String()
}

func bMcpUseStdio(args ...Object) Object {
	if len(args) < 1 {
		return err("mcp_use_stdio expects at least 1 argument (command, args..., env?)")
	}
	command, ok := args[0].(*String)
	if !ok {
		return err("mcp_use_stdio: first argument must be a string (command)")
	}
	cmdArgs := make([]string, 0)
	envVars := make(map[string]string)
	argEnd := len(args)

	// Last argument can be a Map for environment variables
	if len(args) >= 2 {
		if envMap, ok := args[len(args)-1].(*Map); ok {
			argEnd = len(args) - 1
			for k, v := range envMap.Pairs {
				if s, ok := v.(*String); ok {
					envVars[k] = s.Value
				} else {
					envVars[k] = v.Inspect()
				}
			}
		}
	}

	for _, a := range args[1:argEnd] {
		if s, ok := a.(*String); ok {
			cmdArgs = append(cmdArgs, s.Value)
		} else {
			cmdArgs = append(cmdArgs, a.Inspect())
		}
	}

	client, clientErr := mcp.NewStdioClient(command.Value, cmdArgs, envVars)
	if clientErr != nil {
		return err("mcp_use_stdio: " + clientErr.Error())
	}

	prefix := fmt.Sprintf("mcp%d_", len(mcpClients))
	if regErr := registerMCPClient(client, prefix); regErr != nil {
		return err("mcp_use_stdio: " + regErr.Error())
	}

	return &String{Value: fmt.Sprintf("connected %d tools from %s (prefix: %s)", len(mcpClients[len(mcpClients)-1].tools), command.Value, prefix)}
}

func bMcpUseSSE(args ...Object) Object {
	if len(args) < 1 {
		return err("mcp_use_sse expects 1 argument (url)")
	}
	url, ok := args[0].(*String)
	if !ok {
		return err("mcp_use_sse: argument must be a string (url)")
	}

	client, clientErr := mcp.NewHTTPClient(url.Value)
	if clientErr != nil {
		return err("mcp_use_sse: " + clientErr.Error())
	}

	prefix := fmt.Sprintf("mcp%d_", len(mcpClients))
	if regErr := registerMCPClient(client, prefix); regErr != nil {
		return err("mcp_use_sse: " + regErr.Error())
	}

	return &String{Value: fmt.Sprintf("connected %d tools from %s (prefix: %s)", len(mcpClients[len(mcpClients)-1].tools), url.Value, prefix)}
}
