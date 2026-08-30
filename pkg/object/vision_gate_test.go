package object

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

// The CLI --sandbox flag keeps ActiveProfile at "none" and governs AI calls
// via Sandbox.AllowAI. ai_vision must honor that flag before ai.VisionChat
// ever reaches the network or reads the image off disk.
func TestAiVisionBlockedBySandboxFlag(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Write([]byte(`{"choices":[{"message":{"content":"should never be reached"}}]}`))
	}))
	defer srv.Close()
	openaiAt(t, srv)
	defer withSandboxAIFlags(true, false)()

	result := bAiVision(&String{Value: "https://example.com/x.jpg"}, &String{Value: "describe"})
	assertSandboxBlocked(t, "ai_vision", result)

	if hits.Load() != 0 {
		t.Fatalf("ai_vision reached the network under --sandbox: %d requests", hits.Load())
	}
}

func TestAiVisionAllowedBySandboxFlag(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"choices":[{"finish_reason":"stop","message":{"content":"a red circle"}}]}`))
	}))
	defer srv.Close()
	openaiAt(t, srv)
	defer withSandboxAIFlags(true, true)()

	result := bAiVision(&String{Value: "https://example.com/x.jpg"}, &String{Value: "describe"})
	if result.Type() == ERROR {
		t.Fatalf("ai_vision: expected success under AllowAI=true, got %s", result.Inspect())
	}
	s, ok := result.(*String)
	if !ok || s.Value != "a red circle" {
		t.Fatalf("ai_vision: unexpected result %v", result)
	}
}

// A 1x1 transparent PNG, the smallest fixture that content-sniffs as
// image/png.
var tinyPNG = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
	0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4, 0x89, 0x00, 0x00, 0x00,
	0x0a, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00,
	0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00, 0x00, 0x00, 0x00, 0x49,
	0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
}

func TestResolveImageURLHTTPPassesThrough(t *testing.T) {
	url, rerr := resolveImageURL(&String{Value: "https://example.com/photo.jpg"})
	if rerr != nil {
		t.Fatalf("unexpected error: %v", rerr)
	}
	if url != "https://example.com/photo.jpg" {
		t.Errorf("url = %q, want passthrough", url)
	}
}

func TestResolveImageURLFromLocalFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tiny.png")
	if err := os.WriteFile(path, tinyPNG, 0644); err != nil {
		t.Fatal(err)
	}

	url, rerr := resolveImageURL(&String{Value: path})
	if rerr != nil {
		t.Fatalf("unexpected error: %v", rerr)
	}
	if !strings.HasPrefix(url, "data:image/png;base64,") {
		t.Fatalf("url = %q, want a data:image/png;base64,... URL", url)
	}
	encoded := strings.TrimPrefix(url, "data:image/png;base64,")
	decoded, decErr := base64.StdEncoding.DecodeString(encoded)
	if decErr != nil || string(decoded) != string(tinyPNG) {
		t.Fatalf("decoded data URL does not round-trip the original bytes")
	}
}

func TestResolveImageURLFromBytes(t *testing.T) {
	url, rerr := resolveImageURL(&Bytes{Value: tinyPNG})
	if rerr != nil {
		t.Fatalf("unexpected error: %v", rerr)
	}
	if !strings.HasPrefix(url, "data:image/png;base64,") {
		t.Fatalf("url = %q, want a data:image/png;base64,... URL", url)
	}
}

func TestResolveImageURLRejectsUnsupportedType(t *testing.T) {
	_, rerr := resolveImageURL(&Bytes{Value: []byte("not an image, just plain text bytes")})
	if rerr == nil || !strings.Contains(rerr.Message, "unsupported image type") {
		t.Fatalf("expected an unsupported-type error, got %v", rerr)
	}
}

func TestResolveImageURLRejectsWrongArgType(t *testing.T) {
	_, rerr := resolveImageURL(&Integer{Value: 42})
	if rerr == nil {
		t.Fatal("expected an error for a non-string, non-bytes image argument")
	}
}
