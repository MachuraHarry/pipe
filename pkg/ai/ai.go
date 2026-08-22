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
	Content          string  `json:"content"`
	PromptTokens     int     `json:"prompt_tokens,omitempty"`
	CompletionTokens int     `json:"completion_tokens,omitempty"`
	TotalTokens      int     `json:"total_tokens,omitempty"`
	CostUSD          float64 `json:"cost_usd,omitempty"`
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
	// ExtraBody holds additional request fields that are merged verbatim into
	// the JSON body of every provider request. Provider-agnostic passthrough
	// for capabilities like reasoning/thinking controls.
	ExtraBody map[string]interface{}
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
	ActiveConfig.ExtraBody = nil
	switch name {
	case "openai":
		ActiveConfig.APIHost = "https://api.openai.com"
		ActiveConfig.Model = "gpt-4o-mini"
	case "anthropic":
		ActiveConfig.APIHost = "https://api.anthropic.com"
		ActiveConfig.Model = "claude-3-5-haiku-20241022"
	case "deepseek":
		ActiveConfig.APIHost = "https://api.deepseek.com"
		ActiveConfig.Model = "deepseek-v4-flash"
	case "ollama":
		ActiveConfig.APIHost = "http://localhost:11434"
		ActiveConfig.Model = "llama3.1:8b"
	case "openrouter":
		ActiveConfig.APIHost = "https://openrouter.ai/api"
		ActiveConfig.Model = "openrouter/free"
	}
}

func SetHost(host string) {
	ActiveConfig.APIHost = host
}

func SetModel(model string) {
	ActiveConfig.Model = model
}

// SetExtraBody replaces the provider-agnostic extra request body fields (e.g.
// reasoning/thinking controls). Passed through verbatim by every provider.
func SetExtraBody(extra map[string]interface{}) {
	ActiveConfig.ExtraBody = extra
}

// ThinkingConfig describes provider-specific reasoning controls.
type ThinkingConfig struct {
	// Enabled toggles thinking mode. nil leaves the provider default in place.
	Enabled *bool
	// Effort is the requested reasoning effort: "low", "medium", "high",
	// "xhigh", "max", or "none". "none" disables thinking mode. "" = unset.
	Effort string
}

// SetThinking applies reasoning controls (thinking toggle, effort) to the
// active config's ExtraBody, merging with any existing extra fields. Only the
// deepseek provider understands these controls today.
func SetThinking(provider string, tc ThinkingConfig) error {
	if provider != "deepseek" {
		return fmt.Errorf("provider '%s' does not support thinking/effort controls (only deepseek)", provider)
	}
	extra, err := buildDeepSeekThinking(tc)
	if err != nil {
		return err
	}
	if len(extra) == 0 {
		return nil
	}
	merged := make(map[string]interface{}, len(ActiveConfig.ExtraBody)+len(extra))
	for k, v := range ActiveConfig.ExtraBody {
		merged[k] = v
	}
	for k, v := range extra {
		merged[k] = v
	}
	SetExtraBody(merged)
	return nil
}

// buildDeepSeekThinking maps DeepSeek V4 thinking controls to request body
// fields. The API performs the final effort mapping server-side (low/medium to
// high on pro, xhigh to max), so values are forwarded verbatim. "none"
// disables thinking entirely per the official thinking-mode guide.
func buildDeepSeekThinking(tc ThinkingConfig) (map[string]interface{}, error) {
	var extra map[string]interface{}
	if tc.Enabled != nil {
		typ := "enabled"
		if !*tc.Enabled {
			typ = "disabled"
		}
		extra = map[string]interface{}{"thinking": map[string]interface{}{"type": typ}}
	}
	if tc.Effort == "" {
		return extra, nil
	}
	switch tc.Effort {
	case "low", "medium", "high", "xhigh", "max":
		if extra == nil {
			extra = map[string]interface{}{}
		}
		extra["reasoning_effort"] = tc.Effort
	case "none":
		if extra == nil {
			extra = map[string]interface{}{}
		}
		extra["thinking"] = map[string]interface{}{"type": "disabled"}
		delete(extra, "reasoning_effort")
	default:
		return nil, fmt.Errorf("invalid reasoning effort '%s' (use low, medium, high, xhigh, max, or none)", tc.Effort)
	}
	return extra, nil
}

func SetTimeout(seconds int) {
	ActiveConfig.Timeout = time.Duration(seconds) * time.Second
}

func Chat(req ChatRequest) (ChatResponse, error) {
	if err := gateEgress(EgressChat, ActiveConfig.APIHost); err != nil {
		return ChatResponse{}, err
	}
	systemPrompt, userPrompt := extractPromptStrings(req)
	key := cacheKey(ActiveConfig.Provider, ActiveConfig.Model, systemPrompt, userPrompt)

	if cached, ok := cacheGet(key); ok {
		resp := ChatResponse{Content: cached}
		recordCostHit()
		return resp, nil
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
	case "openrouter":
		resp, err = openrouterChat(ActiveConfig, req)
	default:
		return ChatResponse{}, fmt.Errorf("unknown AI provider: %s", ActiveConfig.Provider)
	}

	if err == nil {
		cacheSet(key, resp.Content)
		recordCost(resp)
	} else {
		recordCostMiss()
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
	if err := gateEgress(EgressStream, ActiveConfig.APIHost); err != nil {
		return err
	}
	switch ActiveConfig.Provider {
	case "openai":
		return openAIStream(ActiveConfig, req, onToken)
	case "anthropic":
		return anthropicStream(ActiveConfig, req, onToken)
	case "deepseek":
		return deepSeekStream(ActiveConfig, req, onToken)
	case "ollama":
		return ollamaStream(ActiveConfig, req, onToken)
	case "openrouter":
		return openrouterStream(ActiveConfig, req, onToken)
	default:
		return fmt.Errorf("unknown AI provider: %s", ActiveConfig.Provider)
	}
}

func httpPostJSON(kind EgressKind, url, apiKey string, reqBody interface{}, timeout time.Duration) (map[string]interface{}, error) {
	if httpPostJSONFn != nil {
		return httpPostJSONFn(kind, url, apiKey, reqBody, timeout)
	}
	return httpPostJSONNative(kind, url, apiKey, reqBody, timeout)
}

var httpPostJSONFn func(EgressKind, string, string, interface{}, time.Duration) (map[string]interface{}, error)

func httpPostJSONNative(kind EgressKind, url, apiKey string, reqBody interface{}, timeout time.Duration) (map[string]interface{}, error) {
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

	client := gatedHTTPClient(kind, timeout)
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

func httpPostStream(kind EgressKind, url, apiKey string, reqBody interface{}, timeout time.Duration, callback StreamCallback) error {
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
	if err := gateEgress(EgressChat, ActiveConfig.APIHost); err != nil {
		errs := make([]error, len(requests))
		for i := range errs {
			errs[i] = err
		}
		return make([]ChatResponse, len(requests)), errs
	}
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

// ---- Cost Tracking ----

var (
	costMu            sync.Mutex
	totalCost         float64
	totalTokens       int
	totalPromptTokens int
	totalCompTokens   int
	costHits          int
	costMisses        int
	callsRecorded     int
	costHistory       []CostEntry
)

type CostEntry struct {
	Provider         string
	Model            string
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	CostUSD          float64
	Cached           bool
}

var costHook func(CostEntry)

func SetCostHook(fn func(CostEntry)) {
	costMu.Lock()
	defer costMu.Unlock()
	costHook = fn
}

// EgressKind classifies AI-provider egress for the central sandbox gate.
type EgressKind int

const (
	// EgressChat is a chat/completion call (ai_chat, ai_with_tools, try_ai, ...).
	EgressChat EgressKind = iota
	// EgressStream is a streaming chat call.
	EgressStream
	// EgressEmbed is an embeddings call (embed, embed_batch).
	EgressEmbed
	// EgressSearch is a web-search call.
	EgressSearch
)

// EgressInfo describes a single provider egress that is about to happen.
type EgressInfo struct {
	Kind EgressKind
	URL  string
}

var (
	egressGate   func(EgressInfo) error
	egressGateMu sync.Mutex
)

// SetEgressGate installs a central sandbox gate that every AI-provider egress
// must pass before a network call is made. It is invoked by the egress entry
// points (Chat, Stream, ChatParallel, ChatWithTools, Embed, EmbedBatch,
// WebSearch) and can reject a call, e.g. when the active profile has ai:false
// or an exhausted budget. A nil gate permits all egress.
func SetEgressGate(fn func(EgressInfo) error) {
	egressGateMu.Lock()
	defer egressGateMu.Unlock()
	egressGate = fn
}

// gateEgress invokes the installed gate, if any, for the given egress kind and
// target URL. A nil error means the call is allowed.
func gateEgress(kind EgressKind, url string) error {
	egressGateMu.Lock()
	fn := egressGate
	egressGateMu.Unlock()
	if fn == nil {
		return nil
	}
	return fn(EgressInfo{Kind: kind, URL: url})
}

// gatedHTTPClient returns an http.Client whose redirect hops are re-checked
// against the central egress gate. A provider endpoint could answer with a
// redirect to a target the profile would block; re-checking per hop keeps that
// from being silently followed.
func gatedHTTPClient(kind EgressKind, timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return gateEgress(kind, req.URL.String())
		},
	}
}

func recordCost(resp ChatResponse) {
	costMu.Lock()
	defer costMu.Unlock()

	tokens := resp.TotalTokens
	if tokens == 0 {
		tokens = resp.PromptTokens + resp.CompletionTokens
	}
	totalTokens += tokens
	totalPromptTokens += resp.PromptTokens
	totalCompTokens += resp.CompletionTokens
	callsRecorded++

	cost := resp.CostUSD
	if cost == 0 {
		cost = estimateCost(ActiveConfig.Provider, ActiveConfig.Model, resp.PromptTokens, resp.CompletionTokens)
	}
	totalCost += cost

	entry := CostEntry{
		Provider:         ActiveConfig.Provider,
		Model:            ActiveConfig.Model,
		PromptTokens:     resp.PromptTokens,
		CompletionTokens: resp.CompletionTokens,
		TotalTokens:      tokens,
		CostUSD:          cost,
	}
	costHistory = append(costHistory, entry)

	if costHook != nil {
		costHook(entry)
	}
}

func recordCostHit() {
	costMu.Lock()
	defer costMu.Unlock()
	costHits++
}

func recordCostMiss() {
	costMu.Lock()
	defer costMu.Unlock()
	costMisses++
}

func GetCostMetrics() (totalCostUSD float64, totalTokensUsed int, calls int, hits int, misses int) {
	costMu.Lock()
	defer costMu.Unlock()
	return totalCost, totalTokens, callsRecorded, costHits, costMisses
}

func GetCostHistory() []CostEntry {
	costMu.Lock()
	defer costMu.Unlock()
	cp := make([]CostEntry, len(costHistory))
	copy(cp, costHistory)
	return cp
}

func GetTotalCost() float64 {
	costMu.Lock()
	defer costMu.Unlock()
	return totalCost
}

func ResetCostMetrics() {
	costMu.Lock()
	defer costMu.Unlock()
	totalCost = 0
	totalTokens = 0
	totalPromptTokens = 0
	totalCompTokens = 0
	costHits = 0
	costMisses = 0
	callsRecorded = 0
	costHistory = nil
}

func estimateCost(provider, model string, promptTokens, completionTokens int) float64 {
	switch provider {
	case "openai":
		if strings.Contains(model, "gpt-4o-mini") {
			return float64(promptTokens)*0.00015/1000 + float64(completionTokens)*0.0006/1000
		}
		if strings.Contains(model, "gpt-4") {
			return float64(promptTokens)*0.03/1000 + float64(completionTokens)*0.06/1000
		}
		return float64(promptTokens)*0.005/1000 + float64(completionTokens)*0.015/1000
	case "deepseek":
		if strings.Contains(model, "flash") {
			return float64(promptTokens)*0.00014/1000 + float64(completionTokens)*0.00028/1000
		}
		return float64(promptTokens)*0.000435/1000 + float64(completionTokens)*0.00087/1000
	case "anthropic":
		if strings.Contains(model, "haiku") {
			return float64(promptTokens)*0.0008/1000 + float64(completionTokens)*0.004/1000
		}
		if strings.Contains(model, "sonnet") {
			return float64(promptTokens)*0.003/1000 + float64(completionTokens)*0.015/1000
		}
		return float64(promptTokens)*0.015/1000 + float64(completionTokens)*0.075/1000
	case "ollama":
		return 0
	case "openrouter":
		switch {
		case strings.Contains(model, "gpt-4o-mini"):
			return float64(promptTokens)*0.00015/1000 + float64(completionTokens)*0.0006/1000
		case strings.Contains(model, "gpt-4"):
			return float64(promptTokens)*0.03/1000 + float64(completionTokens)*0.06/1000
		case strings.Contains(model, "haiku"):
			return float64(promptTokens)*0.0008/1000 + float64(completionTokens)*0.004/1000
		case strings.Contains(model, "sonnet"):
			return float64(promptTokens)*0.003/1000 + float64(completionTokens)*0.015/1000
		case strings.Contains(model, "deepseek"):
			return float64(promptTokens)*0.000435/1000 + float64(completionTokens)*0.00087/1000
		default:
			return float64(promptTokens)*0.005/1000 + float64(completionTokens)*0.015/1000
		}
	default:
		return 0
	}
}

// EstimateMaxCost returns a conservative upper bound for the cost of a single
// AI call that may produce up to maxTokens completion tokens. A prompt buffer
// is included to cover the request context. It is used for budget
// pre-checking before a call is issued.
func EstimateMaxCost(maxTokens int) float64 {
	if maxTokens <= 0 {
		maxTokens = 4096
	}
	return estimateCost(ActiveConfig.Provider, ActiveConfig.Model, 2000, maxTokens)
}
