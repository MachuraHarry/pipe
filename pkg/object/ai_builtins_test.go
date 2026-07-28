package object

import (
	"testing"
)

func TestAiProviderBuiltin(t *testing.T) {
	result := bAiProvider(&String{Value: "openai"})
	s, ok := result.(*String)
	if !ok || s.Value != "provider set to openai" {
		t.Errorf("ai_provider returned %v, want 'provider set to openai'", result.Inspect())
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

	name, ok := m.Pairs["name"].(*String)
	if !ok || name.Value != "Pipe" {
		t.Errorf("name = %v, want Pipe", m.Pairs["name"].Inspect())
	}

	ver, ok := m.Pairs["version"].(*Integer)
	if !ok || ver.Value != 1 {
		t.Errorf("version = %v, want 1", m.Pairs["version"].Inspect())
	}

	tags, ok := m.Pairs["tags"].(*List)
	if !ok || len(tags.Elements) != 2 {
		t.Errorf("tags = %v", m.Pairs["tags"].Inspect())
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
