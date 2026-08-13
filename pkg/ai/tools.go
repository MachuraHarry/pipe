package ai

import (
	"encoding/json"
	"fmt"
	"strings"
)

type ToolDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

type ToolResult struct {
	ToolCallID string
	Content    string
}

type toolResponse struct {
	Content          string
	ReasoningContent string
	ToolCalls        []ToolCall
	IsToolCall       bool
}

type ToolExecutor func(toolName string, args map[string]interface{}) (string, error)

func ChatWithTools(
	systemPrompt string,
	userPrompt string,
	tools []ToolDef,
	executor ToolExecutor,
	maxRounds int,
) (string, error) {
	if err := gateEgress(EgressChat, ActiveConfig.APIHost); err != nil {
		return "", err
	}
	if maxRounds <= 0 {
		maxRounds = 5
	}

	messages := []map[string]interface{}{}
	if systemPrompt != "" {
		messages = append(messages, map[string]interface{}{
			"role":    "system",
			"content": systemPrompt,
		})
	}
	messages = append(messages, map[string]interface{}{
		"role":    "user",
		"content": userPrompt,
	})

	for round := 0; round < maxRounds; round++ {
		resp, err := chatWithToolsRaw(messages, tools)
		if err != nil {
			return "", fmt.Errorf("tool chat round %d: %w", round, err)
		}

		if !resp.IsToolCall {
			return resp.Content, nil
		}

		if len(resp.ToolCalls) == 0 {
			return resp.Content, nil
		}

		assistantMsg := map[string]interface{}{
			"role": "assistant",
		}
		if resp.ReasoningContent != "" {
			assistantMsg["reasoning_content"] = resp.ReasoningContent
		}

		var openaiToolCalls []map[string]interface{}
		for _, tc := range resp.ToolCalls {
			tcMap := map[string]interface{}{
				"id":   tc.ID,
				"type": "function",
				"function": map[string]interface{}{
					"name":      tc.Name,
					"arguments": tc.Arguments,
				},
			}
			openaiToolCalls = append(openaiToolCalls, tcMap)
		}
		assistantMsg["tool_calls"] = openaiToolCalls
		messages = append(messages, assistantMsg)

		for _, tc := range resp.ToolCalls {
			var args map[string]interface{}
			if err := json.Unmarshal([]byte(tc.Arguments), &args); err != nil {
				args = map[string]interface{}{"raw": tc.Arguments}
			}

			result, execErr := executor(tc.Name, args)
			content := result
			if execErr != nil {
				content = "Error: " + execErr.Error()
			}

			messages = append(messages, map[string]interface{}{
				"role":         "tool",
				"tool_call_id": tc.ID,
				"content":      content,
			})
		}
	}

	return "", fmt.Errorf("max tool rounds (%d) exceeded without final response", maxRounds)
}

func chatWithToolsRaw(messages []map[string]interface{}, tools []ToolDef) (toolResponse, error) {
	apiKey := getProviderKey()
	if apiKey == "" {
		return toolResponse{}, fmt.Errorf("%s API key not set", keyEnvName())
	}

	openaiTools := make([]map[string]interface{}, len(tools))
	for i, t := range tools {
		openaiTools[i] = map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        t.Name,
				"description": t.Description,
				"parameters":  t.Parameters,
			},
		}
	}

	body := map[string]interface{}{
		"model":    ActiveConfig.Model,
		"messages": messages,
		"tools":    openaiTools,
	}
	for k, v := range ActiveConfig.ExtraBody {
		body[k] = v
	}

	result, err := httpPostJSON(ActiveConfig.APIHost+"/v1/chat/completions", apiKey, body, ActiveConfig.Timeout)
	if err != nil {
		return toolResponse{}, fmt.Errorf("chat: %w", err)
	}

	choices, ok := result["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return toolResponse{}, fmt.Errorf("no choices in response")
	}
	choice := choices[0].(map[string]interface{})
	msg, ok := choice["message"].(map[string]interface{})
	if !ok {
		return toolResponse{}, fmt.Errorf("no message in choice")
	}

	finishReason, _ := choice["finish_reason"].(string)
	reasoningContent, _ := msg["reasoning_content"].(string)

	if tc, ok := msg["tool_calls"].([]interface{}); ok && len(tc) > 0 {
		var calls []ToolCall
		for _, raw := range tc {
			tcMap := raw.(map[string]interface{})
			fn := tcMap["function"].(map[string]interface{})

			calls = append(calls, ToolCall{
				ID:        tcMap["id"].(string),
				Name:      fn["name"].(string),
				Arguments: fn["arguments"].(string),
			})
		}
		return toolResponse{IsToolCall: true, ToolCalls: calls, ReasoningContent: reasoningContent}, nil
	}

	content, _ := msg["content"].(string)

	_ = finishReason

	if content == "" && len(strings.TrimSpace(content)) == 0 {
		return toolResponse{Content: "", ReasoningContent: reasoningContent}, nil
	}

	return toolResponse{Content: content, ReasoningContent: reasoningContent}, nil
}
