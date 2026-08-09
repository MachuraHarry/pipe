package mcp

import (
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestClientRealEverythingServer exercises the client against the official
// @modelcontextprotocol/server-everything reference server (via npx). It is
// skipped automatically when npx or the package is unavailable.
func TestClientRealEverythingServer(t *testing.T) {
	if _, err := exec.LookPath("npx"); err != nil {
		t.Skip("npx not available")
	}

	c, err := NewStdioClient("npx", []string{"-y", "@modelcontextprotocol/server-everything"}, nil)
	if err != nil {
		t.Skipf("could not start reference server: %v", err)
	}
	defer c.Close()
	c.SetCallTimeout(60 * time.Second)

	if _, err := c.Initialize(); err != nil {
		t.Skipf("reference server initialize failed (offline?): %v", err)
	}

	if !c.SupportsResources() || !c.SupportsPrompts() {
		t.Fatal("reference server should advertise resources and prompts")
	}

	resources, err := c.ListResources()
	if err != nil {
		t.Fatalf("ListResources: %v", err)
	}
	if len(resources) == 0 {
		t.Fatal("expected at least one resource from the reference server")
	}
	var staticURI string
	for _, r := range resources {
		if r.URI == "demo://resource/static/document/architecture.md" {
			staticURI = r.URI
			break
		}
	}
	if staticURI == "" {
		t.Fatalf("expected known resource URI, got %v", resources)
	}

	read, err := c.ReadResource(staticURI)
	if err != nil {
		t.Fatalf("ReadResource: %v", err)
	}
	if len(read.Contents) == 0 || !strings.Contains(read.Contents[0].Text, "#") {
		t.Fatalf("unexpected resource contents: %+v", read)
	}

	prompts, err := c.ListPrompts()
	if err != nil {
		t.Fatalf("ListPrompts: %v", err)
	}
	found := false
	for _, p := range prompts {
		if p.Name == "simple-prompt" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected simple-prompt in %v", prompts)
	}

	got, err := c.GetPrompt("simple-prompt", nil)
	if err != nil {
		t.Fatalf("GetPrompt: %v", err)
	}
	if len(got.Messages) == 0 || !strings.Contains(got.Messages[0].Content.Text, "simple prompt") {
		t.Fatalf("unexpected prompt result: %+v", got)
	}

	if _, err := c.GetPrompt("does-not-exist", nil); err == nil {
		t.Fatal("expected error for unknown prompt")
	}
}
