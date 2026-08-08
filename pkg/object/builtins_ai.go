package object

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/MachuraHarry/pipe/pkg/ai"
)

func bAiProvider(args ...Object) Object {
	if len(args) < 1 {
		return err("ai_provider expects a provider name (openai, anthropic, deepseek, ollama)")
	}
	s, ok := args[0].(*String)
	if !ok {
		return err("ai_provider: argument must be a string")
	}
	ai.SetProvider(s.Value)

	if len(args) >= 2 {
		config, ok := args[1].(*Map)
		if !ok {
			return err("ai_provider: optional second argument must be a block {model: ..., host: ..., timeout: ...}")
		}
		for key, val := range config.Pairs {
			switch key {
			case "model":
				m, ok := val.(*String)
				if !ok {
					return err("ai_provider: model must be a string")
				}
				ai.SetModel(m.Value)
			case "host":
				h, ok := val.(*String)
				if !ok {
					return err("ai_provider: host must be a string")
				}
				ai.SetHost(h.Value)
			case "timeout":
				t, ok := ToInt(val)
				if !ok {
					return err("ai_provider: timeout must be a number (seconds)")
				}
				ai.SetTimeout(int(t))
			default:
				return err("ai_provider: unknown option '" + key + "'. Use model, host, or timeout")
			}
		}
	}

	return &String{Value: "provider set to " + s.Value}
}

func bAiModel(args ...Object) Object {
	if len(args) != 1 {
		return err("ai_model expects 1 argument (model name)")
	}
	s, ok := args[0].(*String)
	if !ok {
		return err("ai_model: argument must be a string")
	}
	ai.SetModel(s.Value)
	return &String{Value: "model set to " + s.Value}
}

func bAiTimeout(args ...Object) Object {
	if len(args) < 1 {
		return err("ai_timeout expects 1 argument (seconds)")
	}
	v, ok := ToInt(args[0])
	if !ok {
		return err("ai_timeout: argument must be a number")
	}
	ai.SetTimeout(int(v))
	return NILOBJ
}

func bAiHost(args ...Object) Object {
	if len(args) != 1 {
		return err("ai_host expects 1 argument (url)")
	}
	s, ok := args[0].(*String)
	if !ok {
		return err("ai_host: argument must be a string (e.g. 'http://localhost:11434')")
	}
	ai.SetHost(s.Value)
	return &String{Value: "host set to " + s.Value}
}

func bAiCache(args ...Object) Object {
	if len(args) < 1 {
		return err("ai_cache expects 1 argument (on|off|ttl in minutes)")
	}

	switch v := args[0].(type) {
	case *Boolean:
		ai.SetCacheEnabled(v.Value)
		if v.Value {
			return &String{Value: "ai cache enabled (ttl: 10 min)"}
		}
		return &String{Value: "ai cache disabled"}
	case *Integer:
		ttl := int(v.Value)
		if ttl <= 0 {
			ai.SetCacheEnabled(false)
			return &String{Value: "ai cache disabled"}
		}
		ai.SetCacheEnabled(true)
		ai.SetCacheTTL(ttl)
		return &String{Value: fmt.Sprintf("ai cache enabled (ttl: %d min)", ttl)}
	case *String:
		if v.Value == "clear" {
			ai.ClearCache()
			return &String{Value: "ai cache cleared"}
		}
		if v.Value == "stats" {
			h, m := ai.CacheStats()
			return &String{Value: fmt.Sprintf("cache hits: %d, misses: %d", h, m)}
		}
		if v.Value == "on" {
			ai.SetCacheEnabled(true)
			return &String{Value: "ai cache enabled (ttl: 10 min)"}
		}
		if v.Value == "off" {
			ai.SetCacheEnabled(false)
			return &String{Value: "ai cache disabled"}
		}
		return err("ai_cache: unknown option '" + v.Value + "'. Use 'on', 'off', 'clear', 'stats', or a number (minutes)")
	default:
		return err("ai_cache expects true/false, a number (minutes), or 'on'/'off'/'clear'/'stats'")
	}
}

func bAiSetKey(args ...Object) Object {
	if len(args) < 2 {
		return err("ai_set_key expects 2 arguments (provider, key)")
	}
	provider, ok := args[0].(*String)
	if !ok {
		return err("ai_set_key: first argument must be a string (provider: 'openai', 'deepseek', 'anthropic')")
	}
	key, ok := args[1].(*String)
	if !ok {
		return err("ai_set_key: second argument must be a string (API key)")
	}

	switch provider.Value {
	case "openai":
		ai.SetAPIKey("OPENAI_API_KEY", key.Value)
	case "deepseek":
		ai.SetAPIKey("DEEPSEEK_API_KEY", key.Value)
	case "anthropic":
		ai.SetAPIKey("ANTHROPIC_API_KEY", key.Value)
	default:
		return err("ai_set_key: unknown provider '" + provider.Value + "'. Use 'openai', 'deepseek', or 'anthropic'.")
	}
	return &String{Value: "key set for " + provider.Value}
}

func bWebSearch(args ...Object) Object {
	if ActiveProfile.Name != "none" {
		if canErr := ActiveProfile.CanNetwork(); canErr != nil {
			return err(canErr.Error())
		}
	}

	if len(args) < 1 {
		return err("web_search expects 1 argument (query)")
	}

	query, ok := args[0].(*String)
	if !ok {
		return err("web_search: argument must be a string")
	}

	results, searchErr := ai.WebSearch(query.Value)
	if searchErr != nil {
		return err(searchErr.Error())
	}

	elems := make([]Object, len(results))
	for i, r := range results {
		elems[i] = &Map{Pairs: map[string]Object{
			"title":   &String{Value: r.Title},
			"snippet": &String{Value: r.Snippet},
			"url":     &String{Value: r.URL},
		}}
	}
	return &List{Elements: elems}
}

func bWikiSearch(args ...Object) Object {
	if ActiveProfile.Name != "none" {
		if canErr := ActiveProfile.CanNetwork(); canErr != nil {
			return err(canErr.Error())
		}
	}

	if len(args) < 1 {
		return err("wiki_search expects 1 argument (query)")
	}

	query, ok := args[0].(*String)
	if !ok {
		return err("wiki_search: argument must be a string")
	}

	results, searchErr := ai.WikiSearch(query.Value)
	if searchErr != nil {
		return err(searchErr.Error())
	}

	elems := make([]Object, len(results))
	for i, r := range results {
		elems[i] = &Map{Pairs: map[string]Object{
			"title":   &String{Value: r.Title},
			"snippet": &String{Value: r.Snippet},
			"url":     &String{Value: r.URL},
		}}
	}
	return &List{Elements: elems}
}

func bAiCost(args ...Object) Object {
	cost, _, calls, hits, misses := ai.GetCostMetrics()
	if len(args) > 0 {
		if str, ok := args[0].(*String); ok && str.Value == "reset" {
			ai.ResetCostMetrics()
			return &String{Value: "cost metrics reset"}
		}
	}
	return &Map{Pairs: map[string]Object{
		"cost_usd":     &Float{Value: cost},
		"calls":        &Integer{Value: int64(calls)},
		"cache_hits":   &Integer{Value: int64(hits)},
		"cache_misses": &Integer{Value: int64(misses)},
	}}
}

func bAiTokens(args ...Object) Object {
	_, tokens, _, _, _ := ai.GetCostMetrics()
	return &Integer{Value: int64(tokens)}
}

func bAiCacheHits(args ...Object) Object {
	_, _, _, hits, _ := ai.GetCostMetrics()
	return &Integer{Value: int64(hits)}
}

func bAiCacheMisses(args ...Object) Object {
	_, _, _, _, misses := ai.GetCostMetrics()
	return &Integer{Value: int64(misses)}
}

type maxTokensResult struct {
	tokens int
	err    Object
}

func maxTokensArg(args []Object, name string) maxTokensResult {
	if len(args) < 3 {
		return maxTokensResult{}
	}
	mt, ok := args[2].(*Integer)
	if !ok {
		return maxTokensResult{err: err(name + ": optional third argument must be an integer (max_tokens)")}
	}
	if mt.Value < 1 {
		return maxTokensResult{err: err(name + ": max_tokens must be >= 1")}
	}
	return maxTokensResult{tokens: int(mt.Value)}
}

func bAiChat(args ...Object) Object {
	if ActiveProfile.Name != "none" {
		if canErr := ActiveProfile.CanAI(); canErr != nil {
			return err(canErr.Error())
		}
	} else if Sandbox.Enabled && !Sandbox.AllowAI {
		return sandboxBlock("ai_chat (AI calls)")
	}
	if len(args) < 2 {
		return err("ai_chat expects at least 2 arguments (system_prompt, user_prompt)")
	}
	sp, ok := args[0].(*String)
	if !ok {
		return err("ai_chat: first argument must be a string (system prompt)")
	}
	up, ok := args[1].(*String)
	if !ok {
		return err("ai_chat: second argument must be a string (user prompt)")
	}

	req := ai.ChatRequest{
		Messages: []ai.Message{
			{Role: "system", Content: sp.Value},
			{Role: "user", Content: up.Value},
		},
	}
	if mt := maxTokensArg(args, "ai_chat"); mt.err != nil {
		return mt.err
	} else if mt.tokens > 0 {
		req.MaxTokens = mt.tokens
	}

	resp, respErr := ai.Chat(req)
	if respErr != nil {
		return err("ai_chat: " + respErr.Error())
	}
	return &String{Value: resp.Content}
}

func bAiChatJSON(args ...Object) Object {
	if ActiveProfile.Name != "none" {
		if canErr := ActiveProfile.CanAI(); canErr != nil {
			return err(canErr.Error())
		}
	}
	if len(args) < 2 {
		return err("ai_chat_json expects at least 2 arguments (system_prompt, user_prompt)")
	}
	sp, ok := args[0].(*String)
	if !ok {
		return err("ai_chat_json: first argument must be a string")
	}
	up, ok := args[1].(*String)
	if !ok {
		return err("ai_chat_json: second argument must be a string")
	}

	sysPrompt := sp.Value + "\nYou must respond with valid JSON only. No markdown, no explanation."

	req := ai.ChatRequest{
		Messages: []ai.Message{
			{Role: "system", Content: sysPrompt},
			{Role: "user", Content: up.Value},
		},
	}
	if mt := maxTokensArg(args, "ai_chat_json"); mt.err != nil {
		return mt.err
	} else if mt.tokens > 0 {
		req.MaxTokens = mt.tokens
	}

	resp, respErr := ai.Chat(req)
	if respErr != nil {
		return err("ai_chat_json: " + respErr.Error())
	}

	var parsed interface{}
	if jsonErr := json.Unmarshal([]byte(resp.Content), &parsed); jsonErr != nil {
		return err("ai_chat_json: invalid JSON response: " + resp.Content)
	}
	return convertJSON(parsed)
}

func bSummarize(args ...Object) Object {
	if ActiveProfile.Name != "none" {
		if canErr := ActiveProfile.CanAI(); canErr != nil {
			return err(canErr.Error())
		}
	}
	if len(args) < 1 {
		return err("summarize expects at least 1 argument (text)")
	}
	t, ok := args[0].(*String)
	if !ok {
		return err("summarize: argument must be a string")
	}

	sysPrompt := "You are a precise summarizer. Summarize the given text concisely in 2-3 sentences. Respond only with the summary."

	req := ai.ChatRequest{
		Messages: []ai.Message{
			{Role: "system", Content: sysPrompt},
			{Role: "user", Content: "Summarize this:\n\n" + t.Value},
		},
	}

	resp, respErr := ai.Chat(req)
	if respErr != nil {
		return err("summarize: " + respErr.Error())
	}
	return &String{Value: resp.Content}
}

func bTranslate(args ...Object) Object {
	if ActiveProfile.Name != "none" {
		if canErr := ActiveProfile.CanAI(); canErr != nil {
			return err(canErr.Error())
		}
	}
	if len(args) < 2 {
		return err("translate expects 2 arguments (text, target_language)")
	}
	t, ok := args[0].(*String)
	if !ok {
		return err("translate: first argument must be a string (text)")
	}
	lang, ok := args[1].(*String)
	if !ok {
		return err("translate: second argument must be a string (target language)")
	}

	sysPrompt := "You are a translator. Translate the given text to " + lang.Value + ". Respond only with the translated text."

	req := ai.ChatRequest{
		Messages: []ai.Message{
			{Role: "system", Content: sysPrompt},
			{Role: "user", Content: t.Value},
		},
	}

	resp, respErr := ai.Chat(req)
	if respErr != nil {
		return err("translate: " + respErr.Error())
	}
	return &String{Value: resp.Content}
}

func bClassify(args ...Object) Object {
	if ActiveProfile.Name != "none" {
		if canErr := ActiveProfile.CanAI(); canErr != nil {
			return err(canErr.Error())
		}
	}
	if len(args) < 2 {
		return err("classify expects 2 arguments (text, categories)")
	}
	t, ok := args[0].(*String)
	if !ok {
		return err("classify: first argument must be a string (text)")
	}

	var categories string
	switch a := args[1].(type) {
	case *String:
		categories = a.Value
	case *List:
		parts := make([]string, len(a.Elements))
		for i, e := range a.Elements {
			parts[i] = e.Inspect()
		}
		categories = strings.Join(parts, ", ")
	default:
		return err("classify: second argument must be a string or list of categories")
	}

	sysPrompt := "Classify the given text into EXACTLY ONE of the following categories. Respond with only the category name.\nCategories: " + categories

	req := ai.ChatRequest{
		Messages: []ai.Message{
			{Role: "system", Content: sysPrompt},
			{Role: "user", Content: t.Value},
		},
	}

	resp, respErr := ai.Chat(req)
	if respErr != nil {
		return err("classify: " + respErr.Error())
	}
	return &String{Value: strings.TrimSpace(resp.Content)}
}

func bExtract(args ...Object) Object {
	if ActiveProfile.Name != "none" {
		if canErr := ActiveProfile.CanAI(); canErr != nil {
			return err(canErr.Error())
		}
	}
	if len(args) < 2 {
		return err("extract expects 2 arguments (text, schema)")
	}
	t, ok := args[0].(*String)
	if !ok {
		return err("extract: first argument must be a string (text)")
	}
	schema, ok := args[1].(*String)
	if !ok {
		return err("extract: second argument must be a string (schema description)")
	}

	sysPrompt := "Extract the requested information from the text. Respond ONLY with valid JSON. No markdown, no explanation.\nSchema: " + schema.Value

	req := ai.ChatRequest{
		Messages: []ai.Message{
			{Role: "system", Content: sysPrompt},
			{Role: "user", Content: t.Value},
		},
	}

	resp, respErr := ai.Chat(req)
	if respErr != nil {
		return err("extract: " + respErr.Error())
	}

	var parsed interface{}
	if jsonErr := json.Unmarshal([]byte(resp.Content), &parsed); jsonErr != nil {
		return err("extract: invalid JSON response: " + resp.Content)
	}
	return convertJSON(parsed)
}

func bGenerate(args ...Object) Object {
	if ActiveProfile.Name != "none" {
		if canErr := ActiveProfile.CanAI(); canErr != nil {
			return err(canErr.Error())
		}
	}
	if len(args) < 1 {
		return err("generate expects at least 1 argument (prompt)")
	}
	p, ok := args[0].(*String)
	if !ok {
		return err("generate: argument must be a string")
	}

	req := ai.ChatRequest{
		Messages: []ai.Message{
			{Role: "user", Content: p.Value},
		},
	}

	resp, respErr := ai.Chat(req)
	if respErr != nil {
		return err("generate: " + respErr.Error())
	}
	return &String{Value: resp.Content}
}

func bGenerateJSON(args ...Object) Object {
	if ActiveProfile.Name != "none" {
		if canErr := ActiveProfile.CanAI(); canErr != nil {
			return err(canErr.Error())
		}
	}
	if len(args) < 2 {
		return err("generate_json expects 2 arguments (instruction, schema)")
	}
	instruction, ok := args[0].(*String)
	if !ok {
		return err("generate_json: first argument must be a string (instruction)")
	}
	schema, ok := args[1].(*String)
	if !ok {
		return err("generate_json: second argument must be a string (schema)")
	}

	sysPrompt := "You are a JSON generator. Generate data matching the schema. Respond ONLY with valid JSON. No markdown, no explanation, no extra text.\nSchema: " + schema.Value

	req := ai.ChatRequest{
		Messages: []ai.Message{
			{Role: "system", Content: sysPrompt},
			{Role: "user", Content: instruction.Value},
		},
	}

	resp, respErr := ai.Chat(req)
	if respErr != nil {
		return err("generate_json: " + respErr.Error())
	}

	var parsed interface{}
	if jsonErr := json.Unmarshal([]byte(resp.Content), &parsed); jsonErr != nil {
		return err("generate_json: invalid JSON response: " + resp.Content)
	}
	return convertJSON(parsed)
}

func bAsk(args ...Object) Object {
	if ActiveProfile.Name != "none" {
		if canErr := ActiveProfile.CanAI(); canErr != nil {
			return err(canErr.Error())
		}
	}
	if len(args) < 1 {
		return err("ask expects at least 1 argument (question)")
	}
	q, ok := args[0].(*String)
	if !ok {
		return err("ask: argument must be a string")
	}

	sysPrompt := "You are a helpful assistant. Answer the question concisely and accurately."

	req := ai.ChatRequest{
		Messages: []ai.Message{
			{Role: "system", Content: sysPrompt},
			{Role: "user", Content: q.Value},
		},
	}

	resp, respErr := ai.Chat(req)
	if respErr != nil {
		return err("ask: " + respErr.Error())
	}
	return &String{Value: resp.Content}
}

func bAiStream(args ...Object) Object {
	if ActiveProfile.Name != "none" {
		if canErr := ActiveProfile.CanAI(); canErr != nil {
			return err(canErr.Error())
		}
	}
	if len(args) < 2 {
		return err("ai_stream expects at least 2 arguments (system_prompt, user_prompt)")
	}
	sp, ok := args[0].(*String)
	if !ok {
		return err("ai_stream: first argument must be a string (system prompt)")
	}
	up, ok := args[1].(*String)
	if !ok {
		return err("ai_stream: second argument must be a string (user prompt)")
	}

	req := ai.ChatRequest{
		Messages: []ai.Message{
			{Role: "system", Content: sp.Value},
			{Role: "user", Content: up.Value},
		},
	}

	var fullText strings.Builder
	streamErr := ai.Stream(req, func(token string) error {
		fmt.Print(token)
		fullText.WriteString(token)
		return nil
	})
	fmt.Println()

	if streamErr != nil {
		return err("ai_stream: " + streamErr.Error())
	}
	return &String{Value: fullText.String()}
}

func bAiRateLimit(args ...Object) Object {
	if len(args) < 1 {
		return err("ai_rate_limit expects 1 argument (calls_per_second)")
	}
	v, ok := ToInt(args[0])
	if !ok {
		return err("ai_rate_limit: argument must be a number")
	}
	ai.SetRateLimit(int(v))
	return NILOBJ
}

func bAiParallel(args ...Object) Object {
	if ActiveProfile.Name != "none" {
		if canErr := ActiveProfile.CanAI(); canErr != nil {
			return err(canErr.Error())
		}
	}
	if len(args) < 3 {
		return err("ai_parallel expects 3 arguments (concurrency, system_prompt, items)")
	}

	concurrency, ok := ToInt(args[0])
	if !ok {
		return err("ai_parallel: first argument must be a number (concurrency)")
	}

	sp, ok := args[1].(*String)
	if !ok {
		return err("ai_parallel: second argument must be a string (system prompt)")
	}

	items, ok := args[2].(*List)
	if !ok {
		return err("ai_parallel: third argument must be a list of strings")
	}

	requests := make([]ai.ChatRequest, len(items.Elements))
	for i, elem := range items.Elements {
		s, ok := elem.(*String)
		if !ok {
			s = &String{Value: elem.Inspect()}
		}
		requests[i] = ai.ChatRequest{
			Messages: []ai.Message{
				{Role: "system", Content: sp.Value},
				{Role: "user", Content: s.Value},
			},
		}
	}

	results, errs := ai.ChatParallel(requests, int(concurrency))

	elems := make([]Object, len(results))
	for i := range results {
		if errs[i] != nil {
			elems[i] = err("ai_parallel[" + fmt.Sprintf("%d", i) + "]: " + errs[i].Error())
		} else {
			elems[i] = &String{Value: results[i].Content}
		}
	}
	return &List{Elements: elems}
}

func bAiBatch(args ...Object) Object {
	if ActiveProfile.Name != "none" {
		if canErr := ActiveProfile.CanAI(); canErr != nil {
			return err(canErr.Error())
		}
	}
	if len(args) < 2 {
		return err("ai_batch expects 2 arguments (system_prompt, items)")
	}

	sp, ok := args[0].(*String)
	if !ok {
		return err("ai_batch: first argument must be a string (system prompt)")
	}

	items, ok := args[1].(*List)
	if !ok {
		return err("ai_batch: second argument must be a list of strings")
	}

	requests := make([]ai.ChatRequest, len(items.Elements))
	for i, elem := range items.Elements {
		s, ok := elem.(*String)
		if !ok {
			s = &String{Value: elem.Inspect()}
		}
		requests[i] = ai.ChatRequest{
			Messages: []ai.Message{
				{Role: "system", Content: sp.Value},
				{Role: "user", Content: s.Value},
			},
		}
	}

	results, errs := ai.ChatParallel(requests, 0)

	elems := make([]Object, len(results))
	for i := range results {
		if errs[i] != nil {
			elems[i] = err("ai_batch[" + fmt.Sprintf("%d", i) + "]: " + errs[i].Error())
		} else {
			elems[i] = &String{Value: results[i].Content}
		}
	}
	return &List{Elements: elems}
}

func bEmbed(args ...Object) Object {
	if ActiveProfile.Name != "none" {
		if canErr := ActiveProfile.CanNetwork(); canErr != nil {
			return err(canErr.Error())
		}
	}
	if len(args) < 1 {
		return err("embed expects 1 argument (text)")
	}
	t, ok := args[0].(*String)
	if !ok {
		return err("embed: argument must be a string")
	}

	vec, vecErr := ai.Embed(t.Value)
	if vecErr != nil {
		return err("embed: " + vecErr.Error())
	}

	elems := make([]Object, len(vec))
	for i, v := range vec {
		elems[i] = &Float{Value: v}
	}
	return &List{Elements: elems}
}

func bEmbedBatch(args ...Object) Object {
	if len(args) < 1 {
		return err("embed_batch expects 1 argument (list of texts)")
	}
	items, ok := args[0].(*List)
	if !ok {
		return err("embed_batch: argument must be a list of strings")
	}

	texts := make([]string, len(items.Elements))
	for i, elem := range items.Elements {
		s, okElem := elem.(*String)
		if !okElem {
			s = &String{Value: elem.Inspect()}
		}
		texts[i] = s.Value
	}

	vectors, errs := ai.EmbedBatch(texts, 4)

	elems := make([]Object, len(vectors))
	for i := range vectors {
		if errs[i] != nil {
			elems[i] = err("embed_batch[" + fmt.Sprintf("%d", i) + "]: " + errs[i].Error())
		} else {
			vecElems := make([]Object, len(vectors[i]))
			for j, v := range vectors[i] {
				vecElems[j] = &Float{Value: v}
			}
			elems[i] = &List{Elements: vecElems}
		}
	}
	return &List{Elements: elems}
}

func bCosineSim(args ...Object) Object {
	if len(args) < 2 {
		return err("cosine_sim expects 2 arguments (vector_a, vector_b)")
	}
	vecA, okA := args[0].(*List)
	vecB, okB := args[1].(*List)
	if !okA || !okB {
		return err("cosine_sim: arguments must be lists of numbers")
	}

	a := listToFloats(vecA)
	b := listToFloats(vecB)

	return &Float{Value: ai.CosineSimilarity(a, b)}
}

func bDotProduct(args ...Object) Object {
	if len(args) < 2 {
		return err("dot_product expects 2 arguments (vector_a, vector_b)")
	}
	vecA, okA := args[0].(*List)
	vecB, okB := args[1].(*List)
	if !okA || !okB {
		return err("dot_product: arguments must be lists of numbers")
	}

	a := listToFloats(vecA)
	b := listToFloats(vecB)

	return &Float{Value: ai.DotProduct(a, b)}
}

func bNearest(args ...Object) Object {
	if len(args) < 3 {
		return err("nearest expects 3 arguments (query_vec, doc_vecs, k)")
	}
	query, okQ := args[0].(*List)
	docs, okD := args[1].(*List)
	if !okQ || !okD {
		return err("nearest: first two arguments must be lists")
	}
	k, okK := ToInt(args[2])
	if !okK {
		return err("nearest: third argument must be a number (k)")
	}

	q := listToFloats(query)

	docVectors := make([][]float64, len(docs.Elements))
	for i, elem := range docs.Elements {
		docList, ok := elem.(*List)
		if !ok {
			return err("nearest: document vectors must be lists of numbers")
		}
		docVectors[i] = listToFloats(docList)
	}

	indices := ai.Nearest(q, docVectors, int(k))

	elems := make([]Object, len(indices))
	for i, idx := range indices {
		elems[i] = &Integer{Value: int64(idx)}
	}
	return &List{Elements: elems}
}

func listToFloats(list *List) []float64 {
	floats := make([]float64, len(list.Elements))
	for i, elem := range list.Elements {
		if f, ok := elem.(*Float); ok {
			floats[i] = f.Value
		} else if n, ok := elem.(*Integer); ok {
			floats[i] = float64(n.Value)
		}
	}
	return floats
}

// ---- Tool Registry ----

type ToolEntry struct {
	Def ai.ToolDef
	Fn  Object
}

var toolRegistry = map[string]ToolEntry{}

func bAiTool(args ...Object) Object {
	if len(args) < 4 {
		return err("ai_tool expects 4 arguments (name, description, parameters, function)")
	}
	name, ok := args[0].(*String)
	if !ok {
		return err("ai_tool: first argument must be a string (tool name)")
	}
	desc, ok := args[1].(*String)
	if !ok {
		return err("ai_tool: second argument must be a string (description)")
	}
	params, ok := args[2].(*Map)
	if !ok {
		return err("ai_tool: third argument must be a map (parameter schema)")
	}

	fn := args[3]

	paramSchema := make(map[string]interface{})
	for k, v := range params.Pairs {
		if s, ok := v.(*String); ok {
			paramSchema[k] = map[string]interface{}{
				"type":        "string",
				"description": s.Value,
			}
		} else if m, ok := v.(*Map); ok {
			inner := make(map[string]interface{})
			for ik, iv := range m.Pairs {
				if is, ok := iv.(*String); ok {
					inner[ik] = is.Value
				}
			}
			paramSchema[k] = inner
		}
	}

	toolRegistry[name.Value] = ToolEntry{
		Def: ai.ToolDef{
			Name:        name.Value,
			Description: desc.Value,
			Parameters: map[string]interface{}{
				"type":       "object",
				"properties": paramSchema,
				"required":   keysToStrings(params),
			},
		},
		Fn: fn,
	}

	return NILOBJ
}

func keysToStrings(m *Map) []string {
	keys := make([]string, 0, len(m.Pairs))
	for k := range m.Pairs {
		keys = append(keys, k)
	}
	return keys
}

func bAiWithTools(args ...Object) Object {
	if len(args) < 2 {
		return err("ai_with_tools expects at least 2 arguments (system_prompt, user_prompt)")
	}
	sp, ok := args[0].(*String)
	if !ok {
		return err("ai_with_tools: first argument must be a string (system prompt)")
	}
	up, ok := args[1].(*String)
	if !ok {
		return err("ai_with_tools: second argument must be a string (user prompt)")
	}

	maxRounds := 5
	argIdx := 2
	if len(args) >= 3 {
		if n, ok := ToInt(args[2]); ok {
			maxRounds = int(n)
			argIdx = 3
		}
	}

	// Optional sandbox name from a string arg
	profile := ActiveProfile
	if len(args) > argIdx {
		if s, ok := args[argIdx].(*String); ok {
			if p, pErr := GetProfile(s.Value); pErr == nil {
				profile = p
			}
		}
	}

	if profile.Name != "none" {
		if canErr := profile.CanAI(); canErr != nil {
			return err(canErr.Error())
		}
		if canErr := profile.CanNetwork(); canErr != nil {
			return err(canErr.Error())
		}
	}

	tools := make([]ai.ToolDef, 0, len(toolRegistry))
	for _, entry := range toolRegistry {
		tools = append(tools, entry.Def)
	}

	if len(tools) == 0 {
		return err("ai_with_tools: no tools registered. Use ai_tool first.")
	}

	executor := func(toolName string, args map[string]interface{}) (string, error) {
		entry, exists := toolRegistry[toolName]
		if !exists {
			return "", fmt.Errorf("unknown tool: %s", toolName)
		}

		if profile.Name != "none" {
			if canErr := profile.CanExec(); canErr != nil {
				return "", fmt.Errorf("E_SANDBOX: tool '%s' execution blocked by profile '%s'", toolName, profile.Name)
			}
			if canErr := profile.CanToolCall(); canErr != nil {
				return "", fmt.Errorf("E_SANDBOX: %s", canErr.Error())
			}
			profile.Audit("tool_call", toolName)
		}

		argObjects := make([]Object, 0, len(args))
		for _, v := range args {
			switch val := v.(type) {
			case string:
				argObjects = append(argObjects, &String{Value: val})
			case float64:
				if val == float64(int64(val)) {
					argObjects = append(argObjects, &Integer{Value: int64(val)})
				} else {
					argObjects = append(argObjects, &Float{Value: val})
				}
			case bool:
				argObjects = append(argObjects, NativeBoolToBoolean(val))
			default:
				argObjects = append(argObjects, &String{Value: fmt.Sprintf("%v", val)})
			}
		}

		if callUserFn != nil {
			result := callUserFn(entry.Fn, argObjects...)
			return result.Inspect(), nil
		}

		if bi, ok := entry.Fn.(*BuiltinInfo); ok {
			result := bi.Fn(argObjects...)
			return result.Inspect(), nil
		}

		return "", fmt.Errorf("tool function not callable")
	}

	result, chatErr := ai.ChatWithTools(sp.Value, up.Value, tools, executor, maxRounds)
	if chatErr != nil {
		return err("ai_with_tools: " + chatErr.Error())
	}

	return &String{Value: result}
}

// ---- AI — Agents ----

func bAgent(args ...Object) Object {
	if len(args) < 2 {
		return err("agent expects 2 arguments (name, system_prompt)")
	}
	name, ok := args[0].(*String)
	if !ok {
		return err("agent: first argument must be a string (name)")
	}
	prompt, ok := args[1].(*String)
	if !ok {
		return err("agent: second argument must be a string (system prompt)")
	}

	ai.CreateAgent(name.Value, prompt.Value)
	return &String{Value: "agent '" + name.Value + "' created"}
}

func bAgentAsk(args ...Object) Object {
	if ActiveProfile.Name != "none" {
		if canErr := ActiveProfile.CanAI(); canErr != nil {
			return err(canErr.Error())
		}
	}

	if len(args) < 2 {
		return err("agent_ask expects 2 arguments (name, message)")
	}
	name, ok := args[0].(*String)
	if !ok {
		return err("agent_ask: first argument must be a string (agent name)")
	}
	msg, ok := args[1].(*String)
	if !ok {
		return err("agent_ask: second argument must be a string (message)")
	}

	ag, exists := ai.GetAgent(name.Value)
	if !exists {
		return err("agent_ask: agent '" + name.Value + "' not found. Create it with agent first.")
	}

	resp, askErr := ag.Ask(msg.Value)
	if askErr != nil {
		return err("agent_ask: " + askErr.Error())
	}

	return &String{Value: resp}
}

func bAgentClear(args ...Object) Object {
	if len(args) < 1 {
		return err("agent_clear expects 1 argument (name)")
	}
	name, ok := args[0].(*String)
	if !ok {
		return err("agent_clear: argument must be a string (agent name)")
	}

	ag, exists := ai.GetAgent(name.Value)
	if !exists {
		return err("agent_clear: agent '" + name.Value + "' not found")
	}

	ag.Clear()
	return &String{Value: "agent '" + name.Value + "' history cleared"}
}

func bTryAILog(args ...Object) Object {
	logs := ai.GetTryAILog()
	elems := make([]Object, len(logs))
	for i, entry := range logs {
		elems[i] = &Map{Pairs: map[string]Object{
			"time":     &Integer{Value: entry.Time},
			"code":     &String{Value: entry.Code},
			"original": &String{Value: entry.Original},
			"fixed":    &String{Value: entry.Fixed},
			"attempt":  &Integer{Value: int64(entry.Attempt)},
			"success":  &Boolean{Value: entry.Success},
		}}
	}
	return &List{Elements: elems}
}
