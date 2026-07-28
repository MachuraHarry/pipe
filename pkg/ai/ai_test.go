package ai

import (
	"testing"
	"time"
)

func TestSetProvider(t *testing.T) {
	tests := []struct {
		name         string
		provider     string
		wantModel    string
		wantAPIHost  string
	}{
		{"openai", "openai", "gpt-4o-mini", "https://api.openai.com"},
		{"anthropic", "anthropic", "claude-3-5-sonnet-20241022", "https://api.anthropic.com"},
		{"deepseek", "deepseek", "deepseek-v4-pro", "https://api.deepseek.com"},
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
