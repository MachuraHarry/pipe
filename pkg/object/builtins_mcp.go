package object

import (
	"fmt"
	"strings"

	"github.com/MachuraHarry/pipe/pkg/ai"
	"github.com/MachuraHarry/pipe/pkg/mcp"
)

var currentMCPServer *mcp.Server

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
					argObjects = append(argObjects, &String{Value: ""})
					continue
				}
				switch val := v.(type) {
				case string:
					argObjects = append(argObjects, &String{Value: val})
				case float64:
					if val == float64(int64(val)) {
						argObjects = append(argObjects, &Integer{Value: int64(val)})
					} else {
						argObjects = append(argObjects, &Float{Value: val})
					}
				case bool:
					argObjects = append(argObjects, NativeBoolToBoolean(val))
				default:
					argObjects = append(argObjects, &String{Value: fmt.Sprintf("%v", val)})
				}
			}

			if callUserFn != nil {
				result := callUserFn(entry.Fn, argObjects...)
				if e, isErr := result.(*Error); isErr {
					return e.Message, fmt.Errorf("%s", e.Message)
				}
				return result.Inspect(), nil
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
	_ = addr
	return err("mcp_serve_sse: HTTP/SSE transport not yet implemented")
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

// --- MCP Client ---

var mcpClients []*mcpClientEntry

type mcpClientEntry struct {
	client *mcp.Client
	prefix string
	tools  map[string]*mcpBridgeInfo
}

type mcpBridgeInfo struct {
	remoteName string
	paramNames []string
}

func objectToInterface(obj Object) interface{} {
	switch v := obj.(type) {
	case *String:
		return v.Value
	case *Integer:
		return float64(v.Value)
	case *Float:
		return v.Value
	case *Boolean:
		return v.Value
	default:
		return obj.Inspect()
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
