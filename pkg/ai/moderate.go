package ai

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ModerationResult reports a fixed, safety-oriented classification of a piece
// of text -- distinct from the user-defined categories classify() uses.
type ModerationResult struct {
	Flagged    bool               `json:"flagged"`
	Categories map[string]bool    `json:"categories"`
	Scores     map[string]float64 `json:"scores"`
}

// moderationCategories is the fixed taxonomy used by the generic
// prompted-JSON fallback (openAIModerate uses OpenAI's own taxonomy
// instead, returned verbatim by the API).
var moderationCategories = []string{
	"violence", "self-harm", "hate", "sexual", "harassment",
}

// Moderate classifies text against a fixed safety taxonomy. OpenAI has a
// dedicated, free /v1/moderations endpoint -- used directly when it's the
// active provider, since it's cheaper, faster, and more reliable than a
// prompted chat completion. Every other provider falls back to a
// prompted-JSON chat call against the same general taxonomy.
func Moderate(text string) (ModerationResult, error) {
	if ActiveConfig.Provider == "openai" {
		return openAIModerate(ActiveConfig, text)
	}
	return promptedModerate(text)
}

func openAIModerate(cfg Config, text string) (ModerationResult, error) {
	apiKey := getKeyWithOverride("OPENAI_API_KEY")
	if apiKey == "" {
		return ModerationResult{}, fmt.Errorf("OPENAI_API_KEY not set")
	}

	type modReq struct {
		Input string `json:"input"`
	}

	result, err := httpPostJSON(EgressChat, cfg.APIHost+"/v1/moderations", apiKey, modReq{Input: text}, cfg.Timeout)
	if err != nil {
		return ModerationResult{}, err
	}

	results, ok := result["results"].([]interface{})
	if !ok || len(results) == 0 {
		return ModerationResult{}, fmt.Errorf("no results in OpenAI moderation response")
	}
	entry, ok := results[0].(map[string]interface{})
	if !ok {
		return ModerationResult{}, fmt.Errorf("malformed OpenAI moderation result")
	}

	mr := ModerationResult{
		Categories: map[string]bool{},
		Scores:     map[string]float64{},
	}
	if flagged, ok := entry["flagged"].(bool); ok {
		mr.Flagged = flagged
	}
	if cats, ok := entry["categories"].(map[string]interface{}); ok {
		for k, v := range cats {
			if b, ok := v.(bool); ok {
				mr.Categories[k] = b
			}
		}
	}
	if scores, ok := entry["category_scores"].(map[string]interface{}); ok {
		for k, v := range scores {
			if f, ok := v.(float64); ok {
				mr.Scores[k] = f
			}
		}
	}
	return mr, nil
}

func promptedModerate(text string) (ModerationResult, error) {
	sysPrompt := "You are a content safety classifier. Assess the given text against these categories: " +
		strings.Join(moderationCategories, ", ") +
		". Respond ONLY with valid JSON: {\"flagged\": bool, \"categories\": {category: bool, ...}, \"scores\": {category: number 0-1, ...}}. No markdown, no explanation."

	resp, err := Chat(ChatRequest{
		Messages: []Message{
			{Role: "system", Content: sysPrompt},
			{Role: "user", Content: text},
		},
	})
	if err != nil {
		return ModerationResult{}, err
	}

	var mr ModerationResult
	if err := json.Unmarshal([]byte(resp.Content), &mr); err != nil {
		return ModerationResult{}, fmt.Errorf("invalid JSON response: %s", resp.Content)
	}
	if mr.Categories == nil {
		mr.Categories = map[string]bool{}
	}
	if mr.Scores == nil {
		mr.Scores = map[string]float64{}
	}
	return mr, nil
}
