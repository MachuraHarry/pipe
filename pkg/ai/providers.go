package ai

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/hex"
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
		Model     string   `json:"model"`
		Messages  []oaiMsg `json:"messages"`
		MaxTokens int      `json:"max_tokens,omitempty"`
	}
	messages := make([]oaiMsg, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = oaiMsg{Role: m.Role, Content: m.Content}
	}

	oai := oaiReq{
		Model:     cfg.Model,
		Messages:  messages,
		MaxTokens: req.MaxTokens,
	}

	oaiBody, err := bodyWithExtra(oai, cfg.ExtraBody)
	if err != nil {
		return ChatResponse{}, err
	}

	result, err := httpPostJSON(EgressChat, cfg.APIHost+"/v1/chat/completions", apiKey, oaiBody, cfg.Timeout)
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
	resp := ChatResponse{Content: content}
	if usage, ok := result["usage"].(map[string]interface{}); ok {
		if pt, ok := usage["prompt_tokens"].(float64); ok {
			resp.PromptTokens = int(pt)
		}
		if ct, ok := usage["completion_tokens"].(float64); ok {
			resp.CompletionTokens = int(ct)
		}
		if tt, ok := usage["total_tokens"].(float64); ok {
			resp.TotalTokens = int(tt)
		}
		// DeepSeek-specific fields reporting its automatic server-side
		// prompt-prefix cache. Absent (and thus zero) for providers that
		// don't support it.
		if ht, ok := usage["prompt_cache_hit_tokens"].(float64); ok {
			resp.PromptCacheHitTokens = int(ht)
		}
		if mt, ok := usage["prompt_cache_miss_tokens"].(float64); ok {
			resp.PromptCacheMissTokens = int(mt)
		}
	}
	return resp, nil
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
		Model     string  `json:"model"`
		Messages  []dsMsg `json:"messages"`
		MaxTokens int     `json:"max_tokens,omitempty"`
	}
	messages := make([]dsMsg, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = dsMsg{Role: m.Role, Content: m.Content}
	}

	ds := dsReq{
		Model:     cfg.Model,
		Messages:  messages,
		MaxTokens: req.MaxTokens,
	}

	dsBody, err := bodyWithExtra(ds, cfg.ExtraBody)
	if err != nil {
		return ChatResponse{}, err
	}

	result, err := httpPostJSON(EgressChat, cfg.APIHost+"/v1/chat/completions", apiKey, dsBody, cfg.Timeout)
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
	for k, v := range cfg.ExtraBody {
		body[k] = v
	}

	result, err := httpPostJSON(EgressChat, cfg.APIHost+"/v1/messages", apiKey, body, cfg.Timeout)
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
	resp := ChatResponse{Content: text}
	if usage, ok := result["usage"].(map[string]interface{}); ok {
		if it, ok := usage["input_tokens"].(float64); ok {
			resp.PromptTokens = int(it)
		}
		if ot, ok := usage["output_tokens"].(float64); ok {
			resp.CompletionTokens = int(ot)
		}
		resp.TotalTokens = resp.PromptTokens + resp.CompletionTokens
	}
	return resp, nil
}

// bodyWithExtra marshals base into a map and merges the ExtraBody fields on
// top of it, so arbitrary provider capabilities (reasoning/thinking controls,
// etc.) can be passed through verbatim.
func bodyWithExtra(base interface{}, extra map[string]interface{}) (interface{}, error) {
	if len(extra) == 0 {
		return base, nil
	}
	b, err := json.Marshal(base)
	if err != nil {
		return nil, fmt.Errorf("marshal request body: %w", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("unmarshal request body: %w", err)
	}
	for k, v := range extra {
		m[k] = v
	}
	return m, nil
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
	if req.MaxTokens > 0 {
		body["max_tokens"] = req.MaxTokens
	}
	for k, v := range cfg.ExtraBody {
		body[k] = v
	}

	return httpPostStream(EgressStream, cfg.APIHost+"/v1/chat/completions", apiKey, body, cfg.Timeout, onToken)
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
	if req.MaxTokens > 0 {
		body["max_tokens"] = req.MaxTokens
	}
	for k, v := range cfg.ExtraBody {
		body[k] = v
	}

	return httpPostStream(EgressStream, cfg.APIHost+"/v1/chat/completions", apiKey, body, cfg.Timeout, onToken)
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
	for k, v := range cfg.ExtraBody {
		body[k] = v
	}

	return httpPostStreamAnthropic(EgressStream, cfg.APIHost+"/v1/messages", apiKey, body, cfg.Timeout, onToken)
}

func httpPostStreamAnthropic(kind EgressKind, url, apiKey string, reqBody interface{}, timeout time.Duration, callback StreamCallback) error {
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

	client := gatedHTTPClient(kind, timeout)
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
	for k, v := range cfg.ExtraBody {
		body[k] = v
	}

	result, err := httpPostJSON(EgressChat, cfg.APIHost+"/v1/chat/completions", "ollama", body, cfg.Timeout)
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
	for k, v := range cfg.ExtraBody {
		body[k] = v
	}

	return httpPostStream(EgressStream, cfg.APIHost+"/v1/chat/completions", "ollama", body, cfg.Timeout, onToken)
}

func ollamaEmbed(cfg Config, text string) ([]float64, error) {
	body := map[string]interface{}{
		"model": cfg.Model,
		"input": text,
	}

	result, err := httpPostJSON(EgressEmbed, cfg.APIHost+"/v1/embeddings", "ollama", body, cfg.Timeout)
	if err != nil {
		return nil, fmt.Errorf("ollama embed: %w", err)
	}

	return extractEmbedding(result)
}

func openrouterChat(cfg Config, req ChatRequest) (ChatResponse, error) {
	apiKey := getKeyWithOverride("OPENROUTER_API_KEY")
	if apiKey == "" {
		return ChatResponse{}, fmt.Errorf("OPENROUTER_API_KEY not set")
	}

	type oaiMsg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	type oaiReq struct {
		Model     string   `json:"model"`
		Messages  []oaiMsg `json:"messages"`
		MaxTokens int      `json:"max_tokens,omitempty"`
	}
	messages := make([]oaiMsg, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = oaiMsg{Role: m.Role, Content: m.Content}
	}

	oai := oaiReq{
		Model:     cfg.Model,
		Messages:  messages,
		MaxTokens: req.MaxTokens,
	}

	oaiBody, err := bodyWithExtra(oai, cfg.ExtraBody)
	if err != nil {
		return ChatResponse{}, err
	}

	result, err := httpPostJSON(EgressChat, cfg.APIHost+"/v1/chat/completions", apiKey, oaiBody, cfg.Timeout)
	if err != nil {
		return ChatResponse{}, err
	}

	return extractOpenAIResult(result)
}

func openrouterStream(cfg Config, req ChatRequest, onToken StreamCallback) error {
	apiKey := getKeyWithOverride("OPENROUTER_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("OPENROUTER_API_KEY not set")
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
	if req.MaxTokens > 0 {
		body["max_tokens"] = req.MaxTokens
	}
	for k, v := range cfg.ExtraBody {
		body[k] = v
	}

	return httpPostStream(EgressStream, cfg.APIHost+"/v1/chat/completions", apiKey, body, cfg.Timeout, onToken)
}

// ---- OpenCode Zen Provider ----

// opencodeZenHost is the base URL of the OpenCode Zen gateway.
const opencodeZenHost = "https://opencode.ai/zen"

// zenUserAgent mimics the OpenCode CLI so the keyless public tier accepts
// requests. The exact version string is what the CLI sends; the API rejects
// unknown clients with AuthError.
const zenUserAgent = "opencode/1.15.0 ai-sdk/provider-utils/4.0.23 runtime/bun/1.3.13"

// opencodeAPIKey returns the user's Zen API key, if one is configured.
func opencodeAPIKey() string {
	return getKeyWithOverride("OPENCODE_API_KEY")
}

// isOpencodeFreeModel reports whether a Zen model id belongs to the free tier.
func isOpencodeFreeModel(model string) bool {
	return model == "big-pickle" || strings.HasSuffix(model, "-free")
}

// randHex returns n random bytes hex-encoded, falling back to a
// nanosecond timestamp if the system entropy source fails.
func randHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// opencodeHeaders builds the per-request HTTP headers for the Zen gateway.
//
// With an API key, no extra headers are needed (the caller sends the plain
// Authorization bearer). Without a key the public tier requires the request to
// look like it originates from the OpenCode CLI: a fixed bearer token plus
// x-opencode-* identification headers whose request/session IDs must be
// unique per call — hence fresh IDs on every invocation (safe for parallel
// requests).
func opencodeHeaders() map[string]string {
	if opencodeAPIKey() != "" {
		return nil
	}
	ts := time.Now().UnixMilli()
	suffix := randHex(8)
	return map[string]string{
		"Authorization":      "Bearer public",
		"User-Agent":         zenUserAgent,
		"x-opencode-client":  "cli",
		"x-opencode-project": "global",
		"x-opencode-request": fmt.Sprintf("msg_%d_%s", ts, suffix),
		"x-opencode-session": fmt.Sprintf("ses_%d_%s", ts, suffix),
	}
}

// activeExtraHeaders returns provider-specific headers for the active
// provider, used by generic paths such as tool calling.
func activeExtraHeaders() map[string]string {
	if ActiveConfig.Provider == "opencode" {
		return opencodeHeaders()
	}
	return nil
}

func opencodeChat(cfg Config, req ChatRequest) (ChatResponse, error) {
	type zenMsg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	messages := make([]zenMsg, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = zenMsg{Role: m.Role, Content: m.Content}
	}

	// Free-tier models are reasoning models: part of the completion budget is
	// consumed by thinking tokens before any visible content is produced. A
	// generous default keeps short answers from coming back empty.
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 2048
	}

	body := map[string]interface{}{
		"model":      cfg.Model,
		"messages":   messages,
		"max_tokens": maxTokens,
	}
	for k, v := range cfg.ExtraBody {
		body[k] = v
	}

	result, err := httpPostJSON(EgressChat, cfg.APIHost+"/v1/chat/completions", opencodeAPIKey(), body, cfg.Timeout, opencodeHeaders())
	if err != nil {
		if strings.Contains(err.Error(), "AuthError") || strings.Contains(err.Error(), "FreeUsageLimitError") {
			return ChatResponse{}, fmt.Errorf("opencode zen rejected the request (%w); free tier rotates and rate-limits models — set OPENCODE_API_KEY or try another -free model", err)
		}
		return ChatResponse{}, err
	}

	resp, extractErr := extractOpenAIResult(result)
	if extractErr != nil {
		return ChatResponse{}, extractErr
	}
	if strings.TrimSpace(resp.Content) == "" && hasReasoningOnlyResponse(result) {
		return ChatResponse{}, fmt.Errorf("opencode zen returned only reasoning tokens without content — increase max_tokens (current: %d)", maxTokens)
	}
	return resp, nil
}

// hasReasoningOnlyResponse detects the failure mode where the model spent the
// entire completion budget on reasoning_content/reasoning and produced no
// visible text.
func hasReasoningOnlyResponse(result map[string]interface{}) bool {
	choices, ok := result["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return false
	}
	choice, ok := choices[0].(map[string]interface{})
	if !ok {
		return false
	}
	msg, ok := choice["message"].(map[string]interface{})
	if !ok {
		return false
	}
	if rc, ok := msg["reasoning_content"].(string); ok && rc != "" {
		return true
	}
	if r, ok := msg["reasoning"].(string); ok && r != "" {
		return true
	}
	return false
}

func opencodeStream(cfg Config, req ChatRequest, onToken StreamCallback) error {
	type zenMsg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	messages := make([]zenMsg, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = zenMsg{Role: m.Role, Content: m.Content}
	}

	body := map[string]interface{}{
		"model":    cfg.Model,
		"messages": messages,
		"stream":   true,
	}
	// Reasoning models burn budget on invisible thinking tokens first; keep a
	// generous floor so short streams don't end with nothing but reasoning.
	body["max_tokens"] = 2048
	if req.MaxTokens > 0 {
		body["max_tokens"] = req.MaxTokens
	}
	for k, v := range cfg.ExtraBody {
		body[k] = v
	}

	return httpPostStream(EgressStream, cfg.APIHost+"/v1/chat/completions", opencodeAPIKey(), body, cfg.Timeout, onToken, opencodeHeaders())
}
