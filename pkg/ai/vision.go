package ai

import "fmt"

// VisionChat sends a single text+image message and returns the model's text
// answer. It mirrors chatWithToolsRaw's raw-map request building rather than
// widening the typed Message/ChatRequest path: a vision request's content is
// a multi-part array (text + image_url blocks), not a plain string, and that
// shape would otherwise have to be threaded through every provider struct in
// providers.go and extractPromptStrings' cache-key logic in ai.go for a
// capability that (like ChatWithTools) only works with OpenAI-compatible
// providers to begin with. The response shape is unaffected — a vision
// completion still returns plain-string content, so extractOpenAIResult is
// reused as-is.
//
// imageURL is expected to already be a ready-to-send URL (an http(s) URL or a
// data: URL) — building that from a file path or raw bytes is the object
// layer's job (pkg/object/builtins_vision.go), which needs filesystem/sandbox
// access this package intentionally does not have.
func VisionChat(prompt, imageURL string, maxTokens int) (string, error) {
	if err := gateEgress(EgressChat, ActiveConfig.APIHost); err != nil {
		return "", err
	}
	apiKey := getProviderKey()
	if apiKey == "" {
		return "", fmt.Errorf("%s API key not set", keyEnvName())
	}

	body := map[string]interface{}{
		"model": ActiveConfig.Model,
		"messages": []map[string]interface{}{{
			"role": "user",
			"content": []map[string]interface{}{
				{"type": "text", "text": prompt},
				{"type": "image_url", "image_url": map[string]interface{}{"url": imageURL}},
			},
		}},
	}
	if maxTokens > 0 {
		body["max_tokens"] = maxTokens
	}

	result, err := httpPostJSON(EgressChat, ActiveConfig.APIHost+"/v1/chat/completions", apiKey, body, ActiveConfig.Timeout)
	if err != nil {
		return "", fmt.Errorf("vision: %w", err)
	}
	resp, err := extractOpenAIResult(result)
	if err != nil {
		return "", fmt.Errorf("vision: %w", err)
	}
	return resp.Content, nil
}
