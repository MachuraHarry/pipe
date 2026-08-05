package ai

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type SearchResult struct {
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
	URL     string `json:"url"`
}

type ddgResponse struct {
	AbstractText   string `json:"AbstractText"`
	AbstractURL    string `json:"AbstractURL"`
	AbstractSource string `json:"AbstractSource"`
	Heading        string `json:"Heading"`
	Answer         string `json:"Answer"`
	Definition     string `json:"Definition"`
	RelatedTopics  []struct {
		Text     string `json:"Text"`
		FirstURL string `json:"FirstURL"`
	} `json:"RelatedTopics"`
	Results []struct {
		Text     string `json:"Text"`
		FirstURL string `json:"FirstURL"`
	} `json:"Results"`
}

func WebSearch(query string) ([]SearchResult, error) {
	encoded := url.QueryEscape(strings.TrimSpace(query))
	apiURL := fmt.Sprintf("https://api.duckduckgo.com/?q=%s&format=json&no_html=1&skip_disambig=1", encoded)

	body, err := httpGetString(apiURL)
	if err != nil {
		return nil, fmt.Errorf("web_search: %w", err)
	}

	var resp ddgResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("web_search: parse error: %w", err)
	}

	var results []SearchResult

	if resp.AbstractText != "" {
		snippet := resp.AbstractText
		if len(snippet) > 300 {
			snippet = snippet[:300] + "..."
		}
		source := resp.AbstractSource
		if source == "" {
			source = resp.AbstractURL
		}
		results = append(results, SearchResult{
			Title:   resp.Heading,
			Snippet: snippet,
			URL:     resp.AbstractURL,
		})
	}

	if resp.Answer != "" {
		results = append(results, SearchResult{
			Title:   "Answer",
			Snippet: resp.Answer,
			URL:     "",
		})
	}

	if resp.Definition != "" {
		results = append(results, SearchResult{
			Title:   "Definition",
			Snippet: resp.Definition,
			URL:     "",
		})
	}

	for _, t := range resp.RelatedTopics {
		if t.Text == "" {
			continue
		}
		text := t.Text
		if len(text) > 300 {
			text = text[:300] + "..."
		}
		results = append(results, SearchResult{
			Title:   extractTitle(text),
			Snippet: text,
			URL:     t.FirstURL,
		})
	}

	for _, r := range resp.Results {
		if r.Text == "" {
			continue
		}
		text := r.Text
		if len(text) > 300 {
			text = text[:300] + "..."
		}
		results = append(results, SearchResult{
			Title:   extractTitle(text),
			Snippet: text,
			URL:     r.FirstURL,
		})
	}

	if len(results) == 0 {
		return []SearchResult{{
			Title:   "No results",
			Snippet: fmt.Sprintf("No results found for \"%s\".", query),
			URL:     fmt.Sprintf("https://duckduckgo.com/?q=%s", encoded),
		}}, nil
	}

	return results, nil
}

func extractTitle(text string) string {
	parts := strings.SplitN(text, " — ", 2)
	if len(parts) == 2 && len(parts[0]) < 80 {
		return strings.TrimSpace(parts[0])
	}
	if len(text) > 80 {
		return text[:77] + "..."
	}
	return text
}

var httpGetStringFn func(string) ([]byte, error)

func httpGetString(url string) ([]byte, error) {
	if httpGetStringFn != nil {
		return httpGetStringFn(url)
	}
	return httpGetStringNative(url)
}

func httpGetStringNative(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}
