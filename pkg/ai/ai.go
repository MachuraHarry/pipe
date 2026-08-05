package ai

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Messages  []Message `json:"messages"`
	MaxTokens int       `json:"max_tokens,omitempty"`
}

type ChatResponse struct {
	Content string `json:"content"`
}

type Provider interface {
	Chat(req ChatRequest) (ChatResponse, error)
	Name() string
}

type Config struct {
	Provider string
	Model    string
	Timeout  time.Duration
	APIHost  string
}

var DefaultConfig = Config{
	Provider: "openai",
	Model:    "gpt-4o-mini",
	Timeout:  60 * time.Second,
	APIHost:  "https://api.openai.com",
}

var ActiveConfig = DefaultConfig

func SetProvider(name string) {
	ActiveConfig.Provider = name
	switch name {
	case "openai":
		ActiveConfig.APIHost = "https://api.openai.com"
		ActiveConfig.Model = "gpt-4o-mini"
	case "anthropic":
		ActiveConfig.APIHost = "https://api.anthropic.com"
		ActiveConfig.Model = "claude-3-5-sonnet-20241022"
	case "deepseek":
		ActiveConfig.APIHost = "https://api.deepseek.com"
		ActiveConfig.Model = "deepseek-v4-pro"
	case "ollama":
		ActiveConfig.APIHost = "http://localhost:11434"
		ActiveConfig.Model = "llama3.1:8b"
	}
}

func SetHost(host string) {
	ActiveConfig.APIHost = host
}

func SetModel(model string) {
	ActiveConfig.Model = model
}

func SetTimeout(seconds int) {
	ActiveConfig.Timeout = time.Duration(seconds) * time.Second
}

func Chat(req ChatRequest) (ChatResponse, error) {
	systemPrompt, userPrompt := extractPromptStrings(req)
	key := cacheKey(ActiveConfig.Provider, ActiveConfig.Model, systemPrompt, userPrompt)

	if cached, ok := cacheGet(key); ok {
		return ChatResponse{Content: cached}, nil
	}

	var resp ChatResponse
	var err error

	switch ActiveConfig.Provider {
	case "openai":
		resp, err = openAIChat(ActiveConfig, req)
	case "anthropic":
		resp, err = anthropicChat(ActiveConfig, req)
	case "deepseek":
		resp, err = deepSeekChat(ActiveConfig, req)
	case "ollama":
		resp, err = ollamaChat(ActiveConfig, req)
	default:
		return ChatResponse{}, fmt.Errorf("unknown AI provider: %s", ActiveConfig.Provider)
	}

	if err == nil {
		cacheSet(key, resp.Content)
	}

	return resp, err
}

func extractPromptStrings(req ChatRequest) (systemPrompt, userPrompt string) {
	for _, m := range req.Messages {
		switch m.Role {
		case "system":
			systemPrompt = m.Content
		case "user":
			userPrompt = m.Content
		}
	}
	return
}

// StreamCallback receives each token as it arrives from the model.
// Return an error to abort the stream early.
type StreamCallback func(token string) error

func Stream(req ChatRequest, onToken StreamCallback) error {
	switch ActiveConfig.Provider {
	case "openai":
		return openAIStream(ActiveConfig, req, onToken)
	case "anthropic":
		return anthropicStream(ActiveConfig, req, onToken)
	case "deepseek":
		return deepSeekStream(ActiveConfig, req, onToken)
	case "ollama":
		return ollamaStream(ActiveConfig, req, onToken)
	default:
		return fmt.Errorf("unknown AI provider: %s", ActiveConfig.Provider)
	}
}

func httpPostJSON(url, apiKey string, reqBody interface{}, timeout time.Duration) (map[string]interface{}, error) {
	if httpPostJSONFn != nil {
		return httpPostJSONFn(url, apiKey, reqBody, timeout)
	}
	return httpPostJSONNative(url, apiKey, reqBody, timeout)
}

var httpPostJSONFn func(string, string, interface{}, time.Duration) (map[string]interface{}, error)

func httpPostJSONNative(url, apiKey string, reqBody interface{}, timeout time.Duration) (map[string]interface{}, error) {
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	}

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	return result, nil
}

func httpPostStream(url, apiKey string, reqBody interface{}, timeout time.Duration, callback StreamCallback) error {
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
	if apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	}

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
	var contentBuf strings.Builder

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("read stream: %w", err)
		}
		line = strings.TrimSpace(line)
		if line == "" || line == "data: [DONE]" {
			continue
		}
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		jsonStr := strings.TrimPrefix(line, "data: ")

		var chunk map[string]interface{}
		if err := json.Unmarshal([]byte(jsonStr), &chunk); err != nil {
			continue
		}
		choices, ok := chunk["choices"].([]interface{})
		if !ok || len(choices) == 0 {
			continue
		}
		choice := choices[0].(map[string]interface{})
		delta, ok := choice["delta"].(map[string]interface{})
		if !ok {
			continue
		}
		content, ok := delta["content"].(string)
		if !ok {
			continue
		}
		contentBuf.WriteString(content)
		if err := callback(content); err != nil {
			return err
		}
	}
	return nil
}

// ---- Parallel Execution ----

var (
	rateLimiter   *time.Ticker
	rateLimitMu   sync.Mutex
	rateLimitStop chan struct{}
)

func SetRateLimit(callsPerSecond int) {
	rateLimitMu.Lock()
	defer rateLimitMu.Unlock()

	if rateLimitStop != nil {
		close(rateLimitStop)
	}
	if callsPerSecond <= 0 {
		rateLimiter = nil
		return
	}
	interval := time.Second / time.Duration(callsPerSecond)
	rateLimiter = time.NewTicker(interval)
	rateLimitStop = make(chan struct{})
}

func waitRateLimit() {
	rateLimitMu.Lock()
	limiter := rateLimiter
	rateLimitMu.Unlock()
	if limiter != nil {
		<-limiter.C
	}
}

type ParallelResult struct {
	Index  int
	Result ChatResponse
	Err    error
}

func ChatParallel(requests []ChatRequest, concurrency int) ([]ChatResponse, []error) {
	if concurrency <= 0 {
		concurrency = runtime.NumCPU() * 2
	}
	if concurrency > len(requests) {
		concurrency = len(requests)
	}

	results := make([]ChatResponse, len(requests))
	errs := make([]error, len(requests))

	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for i, req := range requests {
		wg.Add(1)
		go func(idx int, r ChatRequest) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			waitRateLimit()

			resp, err := Chat(r)
			results[idx] = resp
			errs[idx] = err
		}(i, req)
	}

	wg.Wait()
	return results, errs
}
