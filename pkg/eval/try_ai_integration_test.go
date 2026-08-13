package eval

import (
	"os"
	"testing"

	"github.com/MachuraHarry/pipe/pkg/ai"
	"github.com/MachuraHarry/pipe/pkg/object"
)

func setupRealDeepSeek(t *testing.T) {
	t.Helper()
	if os.Getenv("DEEPSEEK_API_KEY") == "" {
		t.Skip("DEEPSEEK_API_KEY not set")
	}
	prevCfg := ai.ActiveConfig
	t.Cleanup(func() { ai.ActiveConfig = prevCfg })
	ai.SetProvider("deepseek")
	ai.SetAPIKey("deepseek", os.Getenv("DEEPSEEK_API_KEY"))
}

func TestTryAIRealProviderFixes(t *testing.T) {
	setupRealDeepSeek(t)
	prev := object.ActiveProfile.Load()
	defer object.ActiveProfile.Store(prev)
	prof := object.NewSandboxProfile("real_ai")
	prof.AI = true
	object.ActiveProfile.Store(prof)

	result := parseAndEval(t, "try_ai\n    no_such_var + 1\ncatch e\n    \"fallback\"")
	if result == nil || result.Type() == object.ERROR {
		t.Fatalf("expected a fixed value, got %v", result)
	}
	got := result.Inspect()
	t.Logf("AI-fixed result: %s", got)
	if got == "fallback" {
		t.Fatal("fix did not apply (AI returned UNFIXABLE or a still-broken fix)")
	}
}

func TestTryAIRealProviderBlockedByAIProfile(t *testing.T) {
	setupRealDeepSeek(t)
	prev := object.ActiveProfile.Load()
	defer object.ActiveProfile.Store(prev)
	prof := object.NewSandboxProfile("no_ai")
	prof.AI = false
	object.ActiveProfile.Store(prof)

	result := parseAndEval(t, "try_ai\n    no_such_var + 1\ncatch e\n    \"fallback\"")
	if result == nil || result.Inspect() != "fallback" {
		t.Fatalf("expected fallback with ai:false, got %v", result)
	}
}
