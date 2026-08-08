package object

import (
	"fmt"

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

		// entry.Def.Parameters is already a full JSON Schema {type, properties, required}.
		// Extract property names in deterministic order from the "required" list.
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

		// Build ordered param list for the handler
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
		// Fallback: iterate properties directly if no required list
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
			Schema:      entry.Def.Parameters, // Already a full JSON Schema from ai_tool
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
	if currentMCPServer == nil {
		return err("mcp_tools: no MCP server created. Call mcp_server first.")
	}
	tools := currentMCPServer.Tools()
	elems := make([]Object, len(tools))
	for i, t := range tools {
		elems[i] = &Map{Pairs: map[string]Object{
			"name":        &String{Value: t.Name},
			"description": &String{Value: t.Description},
		}}
	}
	return &List{Elements: elems}
}
