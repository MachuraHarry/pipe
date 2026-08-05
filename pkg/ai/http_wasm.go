//go:build js && wasm

package ai

import (
	"encoding/json"
	"fmt"
	"strings"
	"syscall/js"
	"time"
)

func init() {
	httpPostJSONFn = httpPostJSONWasm
	httpGetStringFn = httpGetStringWasm
}

func wasmHTTPPost(url, apiKey, bodyJSON string) (string, error) {
	headers := js.Global().Get("Object").New()
	headers.Set("Content-Type", "application/json")
	if apiKey != "" {
		headers.Set("Authorization", "Bearer "+apiKey)
	}

	opts := js.Global().Get("Object").New()
	opts.Set("method", "POST")
	opts.Set("headers", headers)
	opts.Set("body", bodyJSON)

	resultChan := make(chan struct{})
	var resultText string
	var resultErr error

	success := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		resp := args[0]
		if !resp.Get("ok").Bool() {
			resultErr = fmt.Errorf("HTTP %d", resp.Get("status").Int())
			close(resultChan)
			return nil
		}
		resp.Call("text").Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			resultText = args[0].String()
			close(resultChan)
			return nil
		}))
		return nil
	})
	defer success.Release()

	fail := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		resultErr = fmt.Errorf("fetch error: %s", args[0].String())
		close(resultChan)
		return nil
	})
	defer fail.Release()

	js.Global().Call("fetch", url, opts).Call("then", success).Call("catch", fail)
	<-resultChan

	return resultText, resultErr
}

func httpPostJSONWasm(url, apiKey string, reqBody interface{}, timeout time.Duration) (map[string]interface{}, error) {
	bodyJSON, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	respText, err := wasmHTTPPost(url, apiKey, string(bodyJSON))
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(respText), &result); err != nil {
		snippet := respText
		if len(snippet) > 200 {
			snippet = snippet[:200]
		}
		return nil, fmt.Errorf("unmarshal: %w (body: %s)", err, snippet)
	}
	return result, nil
}

func httpGetStringWasm(url string) ([]byte, error) {
	resultChan := make(chan struct{})
	var resultText string
	var resultErr error

	success := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		resp := args[0]
		if !resp.Get("ok").Bool() {
			resultErr = fmt.Errorf("HTTP %d", resp.Get("status").Int())
			close(resultChan)
			return nil
		}
		resp.Call("text").Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			resultText = args[0].String()
			close(resultChan)
			return nil
		}))
		return nil
	})
	defer success.Release()

	fail := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		resultErr = fmt.Errorf("fetch error: %s", args[0].String())
		close(resultChan)
		return nil
	})
	defer fail.Release()

	js.Global().Call("fetch", url).Call("then", success).Call("catch", fail)
	<-resultChan

	if resultErr != nil {
		return nil, resultErr
	}
	return []byte(resultText), nil
}

func init() {
	_ = strings.TrimSpace
}
