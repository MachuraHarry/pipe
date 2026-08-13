package object

import (
	"os"
	"strings"
	"testing"

	"github.com/MachuraHarry/pipe/pkg/ai"
)

func TestEmbedRealProviderBlockedByAIProfile(t *testing.T) {
	if os.Getenv("DEEPSEEK_API_KEY") == "" {
		t.Skip("DEEPSEEK_API_KEY not set")
	}
	prevCfg := ai.ActiveConfig
	defer func() { ai.ActiveConfig = prevCfg }()
	ai.SetProvider("deepseek")
	ai.SetAPIKey("deepseek", os.Getenv("DEEPSEEK_API_KEY"))

	defer withProfile(testProfile("emb-live-noai", FSFull, true, false, false, nil))()

	res := bEmbed(&String{Value: "hello"})
	if res.Type() != ERROR || !strings.Contains(res.Inspect(), "E_SANDBOX") {
		t.Fatalf("expected E_SANDBOX under ai:false with a live key, got %q", res.Inspect())
	}
}

func TestEmbedBatchRealProviderBlockedByAIProfile(t *testing.T) {
	if os.Getenv("DEEPSEEK_API_KEY") == "" {
		t.Skip("DEEPSEEK_API_KEY not set")
	}
	prevCfg := ai.ActiveConfig
	defer func() { ai.ActiveConfig = prevCfg }()
	ai.SetProvider("deepseek")
	ai.SetAPIKey("deepseek", os.Getenv("DEEPSEEK_API_KEY"))

	defer withProfile(testProfile("embb-live-noai", FSFull, true, false, false, nil))()

	items := &List{Elements: []Object{&String{Value: "a"}, &String{Value: "b"}}}
	res := bEmbedBatch(items)
	if res.Type() != ERROR || !strings.Contains(res.Inspect(), "E_SANDBOX") {
		t.Fatalf("expected E_SANDBOX under ai:false with a live key, got %q", res.Inspect())
	}
}
