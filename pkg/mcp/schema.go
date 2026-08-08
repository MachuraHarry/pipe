package mcp

func paramsToSchema(params []ParamDef) map[string]interface{} {
	props := make(map[string]interface{})
	required := make([]string, 0, len(params))
	for _, p := range params {
		props[p.Name] = map[string]interface{}{
			"type":        "string",
			"description": p.Description,
		}
		required = append(required, p.Name)
	}
	return map[string]interface{}{
		"type":       "object",
		"properties": props,
		"required":   required,
	}
}
