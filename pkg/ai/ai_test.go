package ai

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestSetProvider(t *testing.T) {
	tests := []struct {
		name        string
		provider    string
		wantModel   string
		wantAPIHost string
	}{
		{"openai", "openai", "gpt-4o-mini", "https://api.openai.com"},
		{"anthropic", "anthropic", "claude-3-5-haiku-20241022", "https://api.anthropic.com"},
		{"deepseek", "deepseek", "deepseek-v4-flash", "https://api.deepseek.com"},
		{"openrouter", "openrouter", "openrouter/free", "https://openrouter.ai/api"},
		{"opencode", "opencode", "big-pickle", "https://opencode.ai/zen"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetProvider(tt.provider)

			if ActiveConfig.Provider != tt.provider {
				t.Errorf("provider = %s, want %s", ActiveConfig.Provider, tt.provider)
			}
			if ActiveConfig.Model != tt.wantModel {
				t.Errorf("model = %s, want %s", ActiveConfig.Model, tt.wantModel)
			}
			if ActiveConfig.APIHost != tt.wantAPIHost {
				t.Errorf("apiHost = %s, want %s", ActiveConfig.APIHost, tt.wantAPIHost)
			}
		})
	}

	ActiveConfig = DefaultConfig
}

func TestSetModel(t *testing.T) {
	SetModel("gpt-4o")
	if ActiveConfig.Model != "gpt-4o" {
		t.Errorf("model = %s, want gpt-4o", ActiveConfig.Model)
	}

	ActiveConfig = DefaultConfig
}

func TestSetTimeout(t *testing.T) {
	SetTimeout(120)
	if ActiveConfig.Timeout != 120*time.Second {
		t.Errorf("timeout = %v, want 120s", ActiveConfig.Timeout)
	}

	ActiveConfig = DefaultConfig
}

func TestDefaultConfig(t *testing.T) {
	if DefaultConfig.Provider != "openai" {
		t.Errorf("default provider = %s, want openai", DefaultConfig.Provider)
	}
	if DefaultConfig.Model != "gpt-4o-mini" {
		t.Errorf("default model = %s, want gpt-4o-mini", DefaultConfig.Model)
	}
	if DefaultConfig.Timeout != 60*time.Second {
		t.Errorf("default timeout = %v, want 60s", DefaultConfig.Timeout)
	}
	if DefaultConfig.APIHost != "https://api.openai.com" {
		t.Errorf("default apiHost = %s, want https://api.openai.com", DefaultConfig.APIHost)
	}
}

func TestChatUnknownProvider(t *testing.T) {
	prev := ActiveConfig
	defer func() { ActiveConfig = prev }()

	ActiveConfig = Config{
		Provider: "unknown",
		Model:    "test",
		Timeout:  1 * time.Second,
		APIHost:  "localhost",
	}

	_, err := Chat(ChatRequest{
		Messages: []Message{{Role: "user", Content: "hello"}},
	})

	if err == nil {
		t.Error("expected error for unknown provider")
	}
}

func TestChatNoAPIKey(t *testing.T) {
	prev := ActiveConfig
	defer func() { ActiveConfig = prev }()

	tests := []struct {
		name     string
		provider string
		envVar   string
	}{
		{"openai missing key", "openai", "OPENAI_API_KEY"},
		{"anthropic missing key", "anthropic", "ANTHROPIC_API_KEY"},
		{"deepseek missing key", "deepseek", "DEEPSEEK_API_KEY"},
		{"openrouter missing key", "openrouter", "OPENROUTER_API_KEY"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldVal := os.Getenv(tt.envVar)
			os.Unsetenv(tt.envVar)
			defer os.Setenv(tt.envVar, oldVal)

			SetProvider(tt.provider)

			_, err := Chat(ChatRequest{
				Messages: []Message{{Role: "user", Content: "hello"}},
			})

			if err == nil {
				t.Error("expected error for missing API key")
			}
		})
	}

	ActiveConfig = DefaultConfig
}

func TestMessageStruct(t *testing.T) {
	msg := Message{Role: "system", Content: "You are helpful."}
	if msg.Role != "system" {
		t.Errorf("role = %s, want system", msg.Role)
	}
	if msg.Content != "You are helpful." {
		t.Errorf("content = %s, want You are helpful.", msg.Content)
	}
}

func TestChatRequestStruct(t *testing.T) {
	req := ChatRequest{
		Messages: []Message{
			{Role: "system", Content: "Be precise."},
			{Role: "user", Content: "Hello"},
		},
		MaxTokens: 100,
	}

	if len(req.Messages) != 2 {
		t.Errorf("len(messages) = %d, want 2", len(req.Messages))
	}
	if req.MaxTokens != 100 {
		t.Errorf("maxTokens = %d, want 100", req.MaxTokens)
	}
	if req.Messages[0].Role != "system" {
		t.Errorf("messages[0].role = %s, want system", req.Messages[0].Role)
	}
	if req.Messages[1].Content != "Hello" {
		t.Errorf("messages[1].content = %s, want Hello", req.Messages[1].Content)
	}
}

func TestChatResponseStruct(t *testing.T) {
	resp := ChatResponse{Content: "Hello, world!"}
	if resp.Content != "Hello, world!" {
		t.Errorf("content = %s, want Hello, world!", resp.Content)
	}
}

func TestEstimateCostFreeModels(t *testing.T) {
	// Free-tier OpenRouter models must cost exactly 0: they fall through to
	// the default price table otherwise, and EstimateMaxCost would burn the
	// Budget sandbox limit with phantom costs for $0 calls.
	SetProvider("openrouter")
	defer func() { SetProvider("openai"); SetModel("gpt-4o-mini") }()
	models := []string{
		"nvidia/nemotron-3-super-120b-a12b:free",
		"deepseek/deepseek-chat-v3.1:free",
	}
	for _, m := range models {
		if got := estimateCost("openrouter", m, 1000000, 1000000); got != 0 {
			t.Errorf("estimateCost(%q) = %v, want 0", m, got)
		}
	}
	SetModel("nvidia/nemotron-3-super-120b-a12b:free")
	if got := EstimateMaxCost(4096); got != 0 {
		t.Errorf("EstimateMaxCost with free default model = %v, want 0", got)
	}
	// Paid models keep their estimates.
	if got := estimateCost("openrouter", "openai/gpt-4o-mini", 1000, 1000); got <= 0 {
		t.Errorf("paid model estimate should be > 0, got %v", got)
	}
}

func TestEstimateCostOpencodeFreeModels(t *testing.T) {
	// Zen free-tier models ("-free" suffix and the big-pickle stealth alias)
	// must cost exactly 0, mirroring the OpenRouter :free handling.
	models := []string{
		"big-pickle",
		"deepseek-v4-flash-free",
		"mimo-v2.5-free",
		"nemotron-3-ultra-free",
	}
	for _, m := range models {
		if got := estimateCost("opencode", m, 1000000, 1000000); got != 0 {
			t.Errorf("estimateCost(opencode, %q) = %v, want 0", m, got)
		}
	}
	if got := estimateCost("opencode", "claude-sonnet-4-5", 1000, 1000); got <= 0 {
		t.Errorf("paid zen model estimate should be > 0, got %v", got)
	}
}

func TestOpencodeHeaders(t *testing.T) {
	prev := ActiveConfig
	defer func() { ActiveConfig = prev }()
	SetProvider("opencode")

	oldVal := os.Getenv("OPENCODE_API_KEY")
	os.Unsetenv("OPENCODE_API_KEY")
	defer os.Setenv("OPENCODE_API_KEY", oldVal)

	h1 := opencodeHeaders()
	if h1 == nil {
		t.Fatal("keyless mode must produce auth headers, got nil")
	}
	if h1["Authorization"] != "Bearer public" {
		t.Errorf("Authorization = %q, want Bearer public", h1["Authorization"])
	}
	for _, k := range []string{"User-Agent", "x-opencode-client", "x-opencode-project", "x-opencode-request", "x-opencode-session"} {
		if h1[k] == "" {
			t.Errorf("header %q missing in keyless mode", k)
		}
	}

	h2 := opencodeHeaders()
	if h1["x-opencode-request"] == h2["x-opencode-request"] {
		t.Error("x-opencode-request IDs must be unique per call")
	}
	if !strings.HasPrefix(h1["x-opencode-request"], "msg_") || !strings.HasPrefix(h1["x-opencode-session"], "ses_") {
		t.Errorf("ID prefixes wrong: request=%q session=%q", h1["x-opencode-request"], h1["x-opencode-session"])
	}
}

func TestOpencodeHeadersWithKey(t *testing.T) {
	prev := ActiveConfig
	defer func() { ActiveConfig = prev }()
	SetProvider("opencode")

	oldVal := os.Getenv("OPENCODE_API_KEY")
	os.Setenv("OPENCODE_API_KEY", "sk-test")
	defer func() {
		os.Setenv("OPENCODE_API_KEY", oldVal)
		delete(apiKeyOverrides, "OPENCODE_API_KEY")
	}()

	if h := opencodeHeaders(); h != nil {
		t.Errorf("with an API key no extra headers should be sent, got %v", h)
	}
}
