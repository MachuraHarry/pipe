package ai

import (
	"fmt"
	"hash/fnv"
	"math"
	"sort"
	"strings"
	"sync"
	"unicode"
)

// ---- Embedding API ----

type EmbeddingRequest struct {
	Input string `json:"input"`
	Model string `json:"model"`
}

type EmbeddingResponse struct {
	Vector []float64
}

const localEmbedDim = 128

func localEmbed(text string) []float64 {
	vec := make([]float64, localEmbedDim)
	text = strings.ToLower(text)
	runes := []rune(text)

	for i := 0; i <= len(runes)-3; i++ {
		trigram := string(runes[i : i+3])
		h := fnv.New64a()
		h.Write([]byte(trigram))
		idx := h.Sum64() % uint64(localEmbedDim)
		vec[idx] += 1.0
	}

	words := strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	for _, word := range words {
		if len(word) == 0 {
			continue
		}
		h := fnv.New64a()
		h.Write([]byte(word))
		idx := h.Sum64() % uint64(localEmbedDim)
		vec[idx] += 1.0
	}

	norm := 0.0
	for _, v := range vec {
		norm += v * v
	}
	if norm > 0 {
		norm = math.Sqrt(norm)
		for i := range vec {
			vec[i] /= norm
		}
	}

	return vec
}

func providerSupportsEmbeddings() bool {
	switch ActiveConfig.Provider {
	case "openai", "ollama":
		return true
	default:
		return false
	}
}

func Embed(text string) ([]float64, error) {
	if !providerSupportsEmbeddings() {
		return localEmbed(text), nil
	}

	model := ActiveConfig.Model

	switch ActiveConfig.Provider {
	case "openai":
		model = "text-embedding-3-small"
	case "ollama":
		// Uses user-configured model, e.g., "nomic-embed-text"
	}

	apiKey := getProviderKey()
	if apiKey == "" {
		return nil, fmt.Errorf("%s API key not set", keyEnvName())
	}

	body := map[string]interface{}{
		"model": model,
		"input": text,
	}

	result, err := httpPostJSON(ActiveConfig.APIHost+"/v1/embeddings", apiKey, body, ActiveConfig.Timeout)
	if err != nil {
		return nil, fmt.Errorf("embed: %w", err)
	}

	return extractEmbedding(result)
}

func EmbedBatch(texts []string, concurrency int) ([][]float64, []error) {
	if concurrency <= 0 {
		concurrency = 4
	}

	type result struct {
		idx int
		vec []float64
		err error
	}

	results := make([][]float64, len(texts))
	errs := make([]error, len(texts))

	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	ch := make(chan result, len(texts))

	for i, text := range texts {
		wg.Add(1)
		go func(idx int, t string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			vec, err := Embed(t)
			ch <- result{idx: idx, vec: vec, err: err}
		}(i, text)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	for r := range ch {
		results[r.idx] = r.vec
		errs[r.idx] = r.err
	}

	return results, errs
}

func extractEmbedding(result map[string]interface{}) ([]float64, error) {
	data, ok := result["data"].([]interface{})
	if !ok || len(data) == 0 {
		return nil, fmt.Errorf("no embedding data in response")
	}
	first := data[0].(map[string]interface{})
	embedding, ok := first["embedding"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("no embedding vector in response")
	}

	vec := make([]float64, len(embedding))
	for i, v := range embedding {
		if f, ok := v.(float64); ok {
			vec[i] = f
		}
	}
	return vec, nil
}

func getProviderKey() string {
	switch ActiveConfig.Provider {
	case "openai":
		return getKeyWithOverride("OPENAI_API_KEY")
	case "anthropic":
		return getKeyWithOverride("ANTHROPIC_API_KEY")
	case "deepseek":
		return getKeyWithOverride("DEEPSEEK_API_KEY")
	case "ollama":
		return "ollama"
	}
	return ""
}

func keyEnvName() string {
	switch ActiveConfig.Provider {
	case "openai":
		return "OPENAI_API_KEY"
	case "anthropic":
		return "ANTHROPIC_API_KEY"
	case "deepseek":
		return "DEEPSEEK_API_KEY"
	case "ollama":
		return ""
	}
	return "API_KEY"
}

// ---- Vector Math (pure Go, no dependencies) ----

func DotProduct(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0
	}
	var sum float64
	for i := range a {
		sum += a[i] * b[i]
	}
	return sum
}

func Norm(vec []float64) float64 {
	return math.Sqrt(DotProduct(vec, vec))
}

func CosineSimilarity(a, b []float64) float64 {
	dot := DotProduct(a, b)
	normA := Norm(a)
	normB := Norm(b)
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (normA * normB)
}

type scoredDoc struct {
	index int
	score float64
}

func Nearest(query []float64, docs [][]float64, k int) []int {
	if k <= 0 || len(docs) == 0 {
		return nil
	}
	if k > len(docs) {
		k = len(docs)
	}

	scores := make([]scoredDoc, len(docs))
	for i, doc := range docs {
		scores[i] = scoredDoc{
			index: i,
			score: CosineSimilarity(query, doc),
		}
	}

	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	result := make([]int, k)
	for i := 0; i < k; i++ {
		result[i] = scores[i].index
	}
	return result
}
