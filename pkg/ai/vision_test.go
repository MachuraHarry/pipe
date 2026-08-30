package ai

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestVisionChatSendsTextAndImageContentBlocks(t *testing.T) {
	var capturedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("reading request body: %v", err)
		}
		if err := json.Unmarshal(body, &capturedBody); err != nil {
			t.Fatalf("decoding request body: %v", err)
		}
		writeChatContent(w, "a golden retriever sitting in grass")
	}))
	defer srv.Close()

	prev := ActiveConfig
	t.Cleanup(func() { ActiveConfig = prev })
	ActiveConfig = Config{Provider: "ollama", Model: "vision-model", Timeout: 5 * time.Second, APIHost: srv.URL}

	answer, err := VisionChat("What is in this image?", "https://example.com/dog.jpg", 0)
	if err != nil {
		t.Fatalf("VisionChat: unexpected error: %v", err)
	}
	if answer != "a golden retriever sitting in grass" {
		t.Errorf("answer = %q", answer)
	}

	messages, ok := capturedBody["messages"].([]interface{})
	if !ok || len(messages) != 1 {
		t.Fatalf("expected exactly 1 message, got %v", capturedBody["messages"])
	}
	msg := messages[0].(map[string]interface{})
	if msg["role"] != "user" {
		t.Errorf("role = %v, want user", msg["role"])
	}
	blocks, ok := msg["content"].([]interface{})
	if !ok || len(blocks) != 2 {
		t.Fatalf("expected 2 content blocks (text + image_url), got %v", msg["content"])
	}
	textBlock := blocks[0].(map[string]interface{})
	if textBlock["type"] != "text" || textBlock["text"] != "What is in this image?" {
		t.Errorf("text block = %v", textBlock)
	}
	imgBlock := blocks[1].(map[string]interface{})
	if imgBlock["type"] != "image_url" {
		t.Errorf("image block type = %v, want image_url", imgBlock["type"])
	}
	imgURL := imgBlock["image_url"].(map[string]interface{})
	if imgURL["url"] != "https://example.com/dog.jpg" {
		t.Errorf("image url = %v", imgURL["url"])
	}
}

func TestVisionChatIncludesMaxTokensWhenPositive(t *testing.T) {
	var capturedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		writeChatContent(w, "ok")
	}))
	defer srv.Close()

	prev := ActiveConfig
	t.Cleanup(func() { ActiveConfig = prev })
	ActiveConfig = Config{Provider: "ollama", Model: "vision-model", Timeout: 5 * time.Second, APIHost: srv.URL}

	if _, err := VisionChat("describe", "https://example.com/x.jpg", 128); err != nil {
		t.Fatalf("VisionChat: %v", err)
	}
	if mt, ok := capturedBody["max_tokens"].(float64); !ok || int(mt) != 128 {
		t.Errorf("max_tokens = %v, want 128", capturedBody["max_tokens"])
	}
}

func TestVisionChatBlockedByEgressGate(t *testing.T) {
	hitServer := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hitServer = true
	}))
	defer srv.Close()

	prev := ActiveConfig
	t.Cleanup(func() { ActiveConfig = prev })
	ActiveConfig = Config{Provider: "ollama", Model: "vision-model", Timeout: 5 * time.Second, APIHost: srv.URL}

	prevGate := egressGate
	blockErr := errors.New("blocked by test gate")
	SetEgressGate(func(info EgressInfo) error { return blockErr })
	t.Cleanup(func() { SetEgressGate(prevGate) })

	_, err := VisionChat("describe", "https://example.com/x.jpg", 0)
	if err == nil || !strings.Contains(err.Error(), "blocked") {
		t.Fatalf("expected a blocked error, got %v", err)
	}
	if hitServer {
		t.Fatal("VisionChat reached the network despite the egress gate blocking it")
	}
}
