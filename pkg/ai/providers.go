package ai

import (
	"fmt"
	"os"
	"time"
)

func getKey(envVar string) string {
	return os.Getenv(envVar)
}

func openAIChat(cfg Config, req ChatRequest) (ChatResponse, error) {
	apiKey := getKey("OPENAI_API_KEY")
	if apiKey == "" {
		return ChatResponse{}, fmt.Errorf("OPENAI_API_KEY not set")
	}

	type oaiMsg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	type oaiReq struct {
		Model    string   `json:"model"`
		Messages []oaiMsg `json:"messages"`
	}
	messages := make([]oaiMsg, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = oaiMsg{Role: m.Role, Content: m.Content}
	}

	oai := oaiReq{
		Model:    cfg.Model,
		Messages: messages,
	}

	result, err := httpPostJSON(cfg.APIHost+"/v1/chat/completions", apiKey, oai, cfg.Timeout)
	if err != nil {
		return ChatResponse{}, err
	}

	return extractOpenAIResult(result)
}

func extractOpenAIResult(result map[string]interface{}) (ChatResponse, error) {
	choices, ok := result["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return ChatResponse{}, fmt.Errorf("no choices in OpenAI response")
	}
	choice := choices[0].(map[string]interface{})
	msg, ok := choice["message"].(map[string]interface{})
	if !ok {
		return ChatResponse{}, fmt.Errorf("no message in choice")
	}
	content, ok := msg["content"].(string)
	if !ok {
		return ChatResponse{}, fmt.Errorf("no content in message")
	}
	return ChatResponse{Content: content}, nil
}

func deepSeekChat(cfg Config, req ChatRequest) (ChatResponse, error) {
	apiKey := getKey("DEEPSEEK_API_KEY")
	if apiKey == "" {
		return ChatResponse{}, fmt.Errorf("DEEPSEEK_API_KEY not set")
	}

	type dsMsg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	type dsReq struct {
		Model    string  `json:"model"`
		Messages []dsMsg `json:"messages"`
	}
	messages := make([]dsMsg, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = dsMsg{Role: m.Role, Content: m.Content}
	}

	ds := dsReq{
		Model:    cfg.Model,
		Messages: messages,
	}

	result, err := httpPostJSON(cfg.APIHost+"/v1/chat/completions", apiKey, ds, cfg.Timeout)
	if err != nil {
		return ChatResponse{}, err
	}

	return extractOpenAIResult(result)
}

func anthropicChat(cfg Config, req ChatRequest) (ChatResponse, error) {
	apiKey := getKey("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return ChatResponse{}, fmt.Errorf("ANTHROPIC_API_KEY not set")
	}

	type antMsg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	type antReq struct {
		Model     string   `json:"model"`
		MaxTokens int      `json:"max_tokens"`
		Messages  []antMsg `json:"messages"`
	}

	tokens := req.MaxTokens
	if tokens == 0 {
		tokens = 1024
	}

	systemPrompt := ""
	var messages []antMsg
	for _, m := range req.Messages {
		if m.Role == "system" {
			systemPrompt = m.Content
		} else {
			messages = append(messages, antMsg{Role: m.Role, Content: m.Content})
		}
	}

	ant := antReq{
		Model:     cfg.Model,
		MaxTokens: tokens,
		Messages:  messages,
	}

	body := map[string]interface{}{
		"model":      ant.Model,
		"max_tokens": ant.MaxTokens,
		"messages":   ant.Messages,
	}
	if systemPrompt != "" {
		body["system"] = systemPrompt
	}

	result, err := httpPostJSON(cfg.APIHost+"/v1/messages", apiKey, body, cfg.Timeout)
	if err != nil {
		return ChatResponse{}, err
	}

	contentList, ok := result["content"].([]interface{})
	if !ok || len(contentList) == 0 {
		return ChatResponse{}, fmt.Errorf("no content in Anthropic response")
	}
	firstContent := contentList[0].(map[string]interface{})
	text, ok := firstContent["text"].(string)
	if !ok {
		return ChatResponse{}, fmt.Errorf("no text in Anthropic content")
	}
	return ChatResponse{Content: text}, nil
}

func httpPostJSONStream(url, apiKey string, reqBody interface{}, timeout time.Duration, callback func(string)) error {
	respCh, errCh := make(chan string), make(chan error)

	go func() {
		defer close(respCh)
		defer close(errCh)
		err := httpPostStream(url, apiKey, reqBody, timeout, callback)
		if err != nil {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-time.After(timeout):
		return fmt.Errorf("stream timeout after %v", timeout)
	case <-respCh:
		return nil
	}
}
