package ai

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

func getKey(envVar string) string {
	return os.Getenv(envVar)
}

var apiKeyOverrides = map[string]string{}

func SetAPIKey(name, key string) {
	apiKeyOverrides[name] = key
}

func getKeyWithOverride(envVar string) string {
	if key, ok := apiKeyOverrides[envVar]; ok && key != "" {
		return key
	}
	return os.Getenv(envVar)
}

func openAIChat(cfg Config, req ChatRequest) (ChatResponse, error) {
	apiKey := getKeyWithOverride("OPENAI_API_KEY")
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
	apiKey := getKeyWithOverride("DEEPSEEK_API_KEY")
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
	apiKey := getKeyWithOverride("ANTHROPIC_API_KEY")
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

// ---- Streaming Implementations ----

func openAIStream(cfg Config, req ChatRequest, onToken StreamCallback) error {
	apiKey := getKeyWithOverride("OPENAI_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("OPENAI_API_KEY not set")
	}

	type oaiMsg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	messages := make([]oaiMsg, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = oaiMsg{Role: m.Role, Content: m.Content}
	}

	body := map[string]interface{}{
		"model":    cfg.Model,
		"messages": messages,
		"stream":   true,
	}

	return httpPostStream(cfg.APIHost+"/v1/chat/completions", apiKey, body, cfg.Timeout, onToken)
}

func deepSeekStream(cfg Config, req ChatRequest, onToken StreamCallback) error {
	apiKey := getKeyWithOverride("DEEPSEEK_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("DEEPSEEK_API_KEY not set")
	}

	type oaiMsg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	messages := make([]oaiMsg, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = oaiMsg{Role: m.Role, Content: m.Content}
	}

	body := map[string]interface{}{
		"model":    cfg.Model,
		"messages": messages,
		"stream":   true,
	}

	return httpPostStream(cfg.APIHost+"/v1/chat/completions", apiKey, body, cfg.Timeout, onToken)
}

func anthropicStream(cfg Config, req ChatRequest, onToken StreamCallback) error {
	apiKey := getKeyWithOverride("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("ANTHROPIC_API_KEY not set")
	}

	tokens := req.MaxTokens
	if tokens == 0 {
		tokens = 4096
	}

	type antMsg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
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

	body := map[string]interface{}{
		"model":      cfg.Model,
		"max_tokens": tokens,
		"messages":   messages,
		"stream":     true,
	}
	if systemPrompt != "" {
		body["system"] = systemPrompt
	}

	return httpPostStreamAnthropic(cfg.APIHost+"/v1/messages", apiKey, body, cfg.Timeout, onToken)
}

func httpPostStreamAnthropic(url, apiKey string, reqBody interface{}, timeout time.Duration, callback StreamCallback) error {
	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("x-api-key", apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("read stream: %w", err)
		}
		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, "data: ") {
			continue
		}
		jsonStr := strings.TrimPrefix(line, "data: ")

		var event map[string]interface{}
		if err := json.Unmarshal([]byte(jsonStr), &event); err != nil {
			continue
		}

		eventType, _ := event["type"].(string)
		if eventType == "content_block_delta" {
			delta, ok := event["delta"].(map[string]interface{})
			if !ok {
				continue
			}
			text, ok := delta["text"].(string)
			if ok && text != "" {
				if err := callback(text); err != nil {
					return err
				}
			}
		}
	}
}

// ---- Ollama Provider (OpenAI-compatible API) ----

func ollamaChat(cfg Config, req ChatRequest) (ChatResponse, error) {
	type oaiMsg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	messages := make([]oaiMsg, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = oaiMsg{Role: m.Role, Content: m.Content}
	}

	body := map[string]interface{}{
		"model":    cfg.Model,
		"messages": messages,
		"stream":   false,
	}

	result, err := httpPostJSON(cfg.APIHost+"/v1/chat/completions", "ollama", body, cfg.Timeout)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("ollama: %w", err)
	}

	return extractOpenAIResult(result)
}

func ollamaStream(cfg Config, req ChatRequest, onToken StreamCallback) error {
	type oaiMsg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	messages := make([]oaiMsg, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = oaiMsg{Role: m.Role, Content: m.Content}
	}

	body := map[string]interface{}{
		"model":    cfg.Model,
		"messages": messages,
		"stream":   true,
	}

	return httpPostStream(cfg.APIHost+"/v1/chat/completions", "ollama", body, cfg.Timeout, onToken)
}

func ollamaEmbed(cfg Config, text string) ([]float64, error) {
	body := map[string]interface{}{
		"model": cfg.Model,
		"input": text,
	}

	result, err := httpPostJSON(cfg.APIHost+"/v1/embeddings", "ollama", body, cfg.Timeout)
	if err != nil {
		return nil, fmt.Errorf("ollama embed: %w", err)
	}

	return extractEmbedding(result)
}
