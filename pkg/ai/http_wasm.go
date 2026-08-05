//go:build js && wasm

package ai

import (
	"encoding/json"
	"fmt"
	"syscall/js"
	"time"
)

func init() {
	httpPostJSONFn = httpPostJSONWasm
	httpGetStringFn = httpGetStringWasm
}

func httpPostJSONWasm(url, apiKey string, reqBody interface{}, timeout time.Duration) (map[string]interface{}, error) {
	bodyJSON, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	headers := js.Global().Get("Object").New()
	headers.Set("Content-Type", "application/json")
	if apiKey != "" {
		headers.Set("Authorization", "Bearer "+apiKey)
	}

	opts := js.Global().Get("Object").New()
	opts.Set("method", "POST")
	opts.Set("headers", headers)
	opts.Set("body", string(bodyJSON))

	resp := js.Global().Call("pipeFetchSync", url, opts)
	if resp.Type() != js.TypeObject {
		return nil, fmt.Errorf("pipeFetchSync returned non-object: %v", resp.Type())
	}

	errMsg := resp.Get("error").String()
	if errMsg != "" {
		return nil, fmt.Errorf("http: %s", errMsg)
	}

	body := resp.Get("body").String()
	var result map[string]interface{}
	if json.Unmarshal([]byte(body), &result) != nil {
		snippet := body
		if len(snippet) > 200 {
			snippet = snippet[:200]
		}
		return nil, fmt.Errorf("unmarshal failed: %s", snippet)
	}
	return result, nil
}

func httpGetStringWasm(url string) ([]byte, error) {
	resp := js.Global().Call("pipeFetchSync", url, nil)
	if resp.Type() != js.TypeObject {
		return nil, fmt.Errorf("pipeFetchSync returned non-object")
	}
	errMsg := resp.Get("error").String()
	if errMsg != "" {
		return nil, fmt.Errorf("http: %s", errMsg)
	}
	return []byte(resp.Get("body").String()), nil
}
