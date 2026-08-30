package object

import (
	"reflect"
	"testing"

	"github.com/MachuraHarry/pipe/pkg/ai"
)

func TestAiProviderBuiltin(t *testing.T) {
	result := bAiProvider(&String{Value: "openai"})
	s, ok := result.(*String)
	if !ok || s.Value != "provider set to openai" {
		t.Errorf("ai_provider returned %v, want 'provider set to openai'", result.Inspect())
	}
}

func TestAiProviderThinkingConfig(t *testing.T) {
	ai.SetProvider("deepseek")
	ai.SetExtraBody(nil)

	result := bAiProvider(&String{Value: "deepseek"}, MapFromGo(map[string]Object{
		"thinking": &Boolean{Value: true},
		"effort":   &String{Value: "max"},
	}))
	if _, ok := result.(*Error); ok {
		t.Fatalf("thinking/effort config failed: %s", result.Inspect())
	}
	if ai.ActiveConfig.ExtraBody == nil {
		t.Fatal("ExtraBody is nil after thinking/effort config")
	}
	th, ok := ai.ActiveConfig.ExtraBody["thinking"].(map[string]interface{})
	if !ok || th["type"] != "enabled" {
		t.Errorf("thinking = %v, want enabled", ai.ActiveConfig.ExtraBody["thinking"])
	}
	if eff, ok := ai.ActiveConfig.ExtraBody["reasoning_effort"].(string); !ok || eff != "max" {
		t.Errorf("reasoning_effort = %v, want max", ai.ActiveConfig.ExtraBody["reasoning_effort"])
	}
}

func TestAiProviderThinkingDisabled(t *testing.T) {
	ai.SetProvider("deepseek")
	ai.SetExtraBody(nil)

	bAiProvider(&String{Value: "deepseek"}, MapFromGo(map[string]Object{
		"thinking": &Boolean{Value: false},
	}))
	th, ok := ai.ActiveConfig.ExtraBody["thinking"].(map[string]interface{})
	if !ok || th["type"] != "disabled" {
		t.Errorf("thinking = %v, want disabled", ai.ActiveConfig.ExtraBody["thinking"])
	}
	if _, exists := ai.ActiveConfig.ExtraBody["reasoning_effort"]; exists {
		t.Error("reasoning_effort must be absent when thinking is disabled")
	}
}

func TestAiProviderEffortNoneDisablesThinking(t *testing.T) {
	ai.SetProvider("deepseek")
	ai.SetExtraBody(nil)

	bAiProvider(&String{Value: "deepseek"}, MapFromGo(map[string]Object{
		"effort": &String{Value: "none"},
	}))
	th, ok := ai.ActiveConfig.ExtraBody["thinking"].(map[string]interface{})
	if !ok || th["type"] != "disabled" {
		t.Errorf("thinking = %v, want disabled for effort none", ai.ActiveConfig.ExtraBody["thinking"])
	}
	if _, exists := ai.ActiveConfig.ExtraBody["reasoning_effort"]; exists {
		t.Error("reasoning_effort must be absent when effort is none")
	}
}

func TestSetThinkingMergesExtraBody(t *testing.T) {
	ai.SetProvider("deepseek")
	ai.SetExtraBody(map[string]interface{}{"temperature": 0.0})

	if err := ai.SetThinking("deepseek", ai.ThinkingConfig{Enabled: boolPtr(true)}); err != nil {
		t.Fatalf("SetThinking failed: %v", err)
	}
	if got := ai.ActiveConfig.ExtraBody["temperature"]; got != 0.0 {
		t.Errorf("existing extra field temperature lost: %v", got)
	}
	if _, ok := ai.ActiveConfig.ExtraBody["thinking"]; !ok {
		t.Error("thinking not merged into ExtraBody")
	}
}

func boolPtr(b bool) *bool { return &b }

func TestAiProviderThinkingInvalidEffort(t *testing.T) {
	ai.SetProvider("deepseek")
	ai.SetExtraBody(nil)

	result := bAiProvider(&String{Value: "deepseek"}, MapFromGo(map[string]Object{
		"effort": &String{Value: "extreme"},
	}))
	if _, ok := result.(*Error); !ok {
		t.Errorf("invalid effort should return error, got %s", result.Inspect())
	}
}

func TestAiProviderThinkingTypeCheck(t *testing.T) {
	ai.SetProvider("deepseek")
	result := bAiProvider(&String{Value: "deepseek"}, MapFromGo(map[string]Object{
		"thinking": &String{Value: "yes"},
	}))
	if _, ok := result.(*Error); !ok {
		t.Errorf("thinking as string should return error, got %s", result.Inspect())
	}

	result = bAiProvider(&String{Value: "deepseek"}, MapFromGo(map[string]Object{
		"effort": &Integer{Value: 3},
	}))
	if _, ok := result.(*Error); !ok {
		t.Errorf("effort as integer should return error, got %s", result.Inspect())
	}
}

func TestAiProviderThinkingNonDeepSeek(t *testing.T) {
	ai.SetProvider("openai")
	result := bAiProvider(&String{Value: "openai"}, MapFromGo(map[string]Object{
		"thinking": &Boolean{Value: true},
	}))
	if _, ok := result.(*Error); !ok {
		t.Errorf("thinking on non-deepseek provider should error, got %s", result.Inspect())
	}
}

func TestSetThinkingEffortMapping(t *testing.T) {
	tests := []struct {
		effort string
		want   map[string]interface{}
	}{
		{"low", map[string]interface{}{"reasoning_effort": "low"}},
		{"medium", map[string]interface{}{"reasoning_effort": "medium"}},
		{"high", map[string]interface{}{"reasoning_effort": "high"}},
		{"xhigh", map[string]interface{}{"reasoning_effort": "xhigh"}},
		{"max", map[string]interface{}{"reasoning_effort": "max"}},
		{"none", map[string]interface{}{"thinking": map[string]interface{}{"type": "disabled"}}},
	}
	for _, tt := range tests {
		t.Run(tt.effort, func(t *testing.T) {
			ai.SetProvider("deepseek")
			ai.SetExtraBody(nil)
			if err := ai.SetThinking("deepseek", ai.ThinkingConfig{Effort: tt.effort}); err != nil {
				t.Fatalf("SetThinking failed: %v", err)
			}
			if !reflect.DeepEqual(ai.ActiveConfig.ExtraBody, tt.want) {
				t.Errorf("SetThinking(%q) ExtraBody = %v, want %v", tt.effort, ai.ActiveConfig.ExtraBody, tt.want)
			}
		})
	}
}

func TestSetThinkingInvalidEffort(t *testing.T) {
	ai.SetProvider("deepseek")
	if err := ai.SetThinking("deepseek", ai.ThinkingConfig{Effort: "extreme"}); err == nil {
		t.Error("invalid effort should return error")
	}
}

func TestSetThinkingNonDeepSeek(t *testing.T) {
	ai.SetProvider("openai")
	if err := ai.SetThinking("openai", ai.ThinkingConfig{Effort: "high"}); err == nil {
		t.Error("non-deepseek provider should return error")
	}
}

func TestAiProviderInvalidArg(t *testing.T) {
	result := bAiProvider(&Integer{Value: 42})
	_, ok := result.(*Error)
	if !ok {
		t.Error("ai_provider(42) should return error")
	}
}

func TestAiModelBuiltin(t *testing.T) {
	result := bAiModel(&String{Value: "gpt-4o"})
	s, ok := result.(*String)
	if !ok || s.Value != "model set to gpt-4o" {
		t.Errorf("ai_model returned %v, want 'model set to gpt-4o'", result.Inspect())
	}
}

func TestAiTimeoutBuiltin(t *testing.T) {
	result := bAiTimeout(&Integer{Value: 120})
	if result.Type() != NIL {
		t.Errorf("ai_timeout should return nil, got %s", result.Type())
	}
}

func TestAiTimeoutNoArg(t *testing.T) {
	result := bAiTimeout()
	_, ok := result.(*Error)
	if !ok {
		t.Error("ai_timeout() should return error")
	}
}

func TestAiConfigArgsError(t *testing.T) {
	tests := []struct {
		name string
		fn   func(args ...Object) Object
	}{
		{"ai_provider no args", bAiProvider},
		{"ai_model no args", bAiModel},
		{"ai_chat too few args", bAiChat},
		{"ai_chat_json too few args", bAiChatJSON},
		{"summarize no args", bSummarize},
		{"translate too few args", bTranslate},
		{"classify too few args", bClassify},
		{"extract too few args", bExtract},
		{"generate no args", bGenerate},
		{"ask no args", bAsk},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.fn()
			_, ok := result.(*Error)
			if !ok {
				t.Errorf("%s() should return error, got %s", tt.name, result.Type())
			}
		})
	}
}

func TestMaxTokensArg(t *testing.T) {
	tests := []struct {
		name  string
		args  []Object
		want  int
		isErr bool
	}{
		{"no third arg", []Object{&String{Value: "s"}, &String{Value: "u"}}, 0, false},
		{"valid max_tokens", []Object{&String{Value: "s"}, &String{Value: "u"}, &Integer{Value: 300}}, 300, false},
		{"non-integer", []Object{&String{Value: "s"}, &String{Value: "u"}, &String{Value: "x"}}, 0, true},
		{"zero", []Object{&String{Value: "s"}, &String{Value: "u"}, &Integer{Value: 0}}, 0, true},
		{"negative", []Object{&String{Value: "s"}, &String{Value: "u"}, &Integer{Value: -5}}, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := maxTokensArg(tt.args, "ai_chat")
			if tt.isErr {
				if res.err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if res.err != nil {
				t.Fatalf("unexpected error: %s", res.err.Inspect())
			}
			if res.tokens != tt.want {
				t.Errorf("tokens = %d, want %d", res.tokens, tt.want)
			}
		})
	}
}

func TestConvertJSONRoundtrip(t *testing.T) {
	tests := []struct {
		name string
		json string
	}{
		{"string", `"hello"`},
		{"integer", `42`},
		{"float", `3.14`},
		{"bool", `true`},
		{"null", `null`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := jsonToObject(tt.json)
			if obj == nil {
				t.Errorf("jsonToObject(%s) returned nil", tt.json)
			}
			if obj.Type() == ERROR {
				t.Errorf("jsonToObject(%s) returned error: %s", tt.json, obj.Inspect())
			}
		})
	}
}

func TestConvertJSONNested(t *testing.T) {
	json := `{"name":"Pipe","version":1,"tags":["ai","pipeline"]}`

	obj := jsonToObject(json)
	m, ok := obj.(*Map)
	if !ok {
		t.Fatalf("expected Map, got %s", obj.Type())
	}
	if len(m.Pairs) != 3 {
		t.Errorf("expected 3 pairs, got %d", len(m.Pairs))
	}

	nameObj, _ := m.Get("name")
	name, ok := nameObj.(*String)
	if !ok || name.Value != "Pipe" {
		t.Errorf("name = %v, want Pipe", nameObj.Inspect())
	}

	verObj, _ := m.Get("version")
	ver, ok := verObj.(*Integer)
	if !ok || ver.Value != 1 {
		t.Errorf("version = %v, want 1", verObj.Inspect())
	}

	tagsObj, _ := m.Get("tags")
	tags, ok := tagsObj.(*List)
	if !ok || len(tags.Elements) != 2 {
		t.Errorf("tags = %v", tagsObj.Inspect())
	}
}

func TestClassifyWithList(t *testing.T) {
	result := bClassify(
		&String{Value: "Meeting at 10am tomorrow"},
		&List{Elements: []Object{
			&String{Value: "work"},
			&String{Value: "personal"},
			&String{Value: "spam"},
		}},
	)
	// Will fail due to no API key, which is expected
	err, isErr := result.(*Error)
	if !isErr {
		return // API might be available (unlikely in test)
	}
	if err != nil {
		t.Logf("classify error (expected without API key): %s", err.Message)
	}
}

func TestClassifyWithString(t *testing.T) {
	result := bClassify(
		&String{Value: "Meeting at 10am tomorrow"},
		&String{Value: "work, personal, spam"},
	)
	_, isErr := result.(*Error)
	if isErr {
		t.Log("classify error (expected without API key)")
	}
}

// TestAiToolPreservesParameterOrder guards round 10's fix: ai_tool must bind
// multi-parameter tools in the user's declaration order, not alphabetically.
// A schema of {path, content} used to be re-ordered to {content, path}.
func TestAiToolPreservesParameterOrder(t *testing.T) {
	name := "write_file_order_test"
	schema := NewMap()
	schema.Set("path", &String{Value: "the file path"})
	schema.Set("content", &String{Value: "the file content"})
	fn := &BuiltinInfo{Fn: func(args ...Object) Object { return NILOBJ }}

	r := bAiTool(&String{Value: name}, &String{Value: "write a file"}, schema, fn)
	if r.Type() == ERROR {
		t.Fatalf("ai_tool returned error: %v", r)
	}

	entry, ok := toolRegistry[name]
	if !ok {
		t.Fatalf("tool %q not registered", name)
	}
	required, _ := entry.Def.Parameters["required"].([]interface{})
	if len(required) != 2 || required[0] != "path" || required[1] != "content" {
		t.Fatalf("required order = %v, want [path content] (declaration order, not alphabetical)", required)
	}

	names := toolParamNames(entry)
	if len(names) != 2 || names[0] != "path" || names[1] != "content" {
		t.Fatalf("toolParamNames = %v, want [path content]", names)
	}
}
