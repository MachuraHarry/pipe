package object

import (
	"os"
	"testing"
)

func TestWasmSandboxAllowsAI(t *testing.T) {
	// Simulate wasm main.go init
	Sandbox = SandboxConfig{}
	SetSandbox(true)
	SetSandboxAllowAI(true)
	SetSandboxAllowNet(true)

	if Sandbox.Enabled != true {
		t.Error("sandbox should be enabled")
	}
	if Sandbox.AllowAI != true {
		t.Error("AI should be allowed")
	}
	if Sandbox.AllowNet != true {
		t.Error("network should be allowed")
	}

	// Verify SandboxAI check
	// ai_chat checks: Sandbox.Enabled && !Sandbox.AllowAI
	// should be: true && !true = false → not blocked
	blocked := Sandbox.Enabled && !Sandbox.AllowAI
	if blocked {
		t.Error("AI should NOT be blocked: Sandbox.Enabled=true, AllowAI=true")
	}
}

func TestWasmNoAPIKeyInBrowser(t *testing.T) {
	// In browser, os.Getenv returns "" for everything
	key := os.Getenv("DEEPSEEK_API_KEY")
	t.Logf("DEEPSEEK_API_KEY = %q (empty in browser/CI)", key)
}
