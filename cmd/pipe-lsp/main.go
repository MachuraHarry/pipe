package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// rpcMessage is the JSON-RPC 2.0 envelope used over the wire.
type rpcMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

const (
	errParse    = -32700
	errInvalid  = -32600
	errMethod   = -32601
	errInternal = -32603
)

func main() {
	// vscode-languageclient launches stdio servers with a trailing "--stdio"
	// argument. The transport is already stdio, so accept and ignore it
	// instead of letting flag.Parse reject it and kill the process.
	for i := 1; i < len(os.Args); {
		if os.Args[i] == "--stdio" || os.Args[i] == "-stdio" {
			os.Args = append(os.Args[:i], os.Args[i+1:]...)
		} else {
			i++
		}
	}

	version := flag.Bool("version", false, "print version and exit")
	flag.Parse()
	if *version {
		fmt.Println("pipe-lsp 0.1.0")
		return
	}

	out := make(chan []byte, 64)
	server := NewServer(out)
	go func() {
		w := bufio.NewWriter(os.Stdout)
		for msg := range out {
			if err := writeMessage(w, msg); err != nil {
				return
			}
			w.Flush()
		}
	}()

	reader := bufio.NewReader(os.Stdin)
	shutdownReceived := false

	for {
		body, err := readMessage(reader)
		if err != nil {
			if err == io.EOF {
				return
			}
			// Protocol error: we can only report it if we can still talk.
			resp := rpcResponse{JSONRPC: "2.0", Error: &rpcError{Code: errParse, Message: err.Error()}}
			out <- marshalResponse(resp)
			continue
		}

		var msg rpcMessage
		if err := json.Unmarshal(body, &msg); err != nil {
			resp := rpcResponse{JSONRPC: "2.0", Error: &rpcError{Code: errParse, Message: "invalid JSON"}}
			out <- marshalResponse(resp)
			continue
		}
		if msg.JSONRPC != "2.0" {
			resp := rpcResponse{JSONRPC: "2.0", ID: msg.ID, Error: &rpcError{Code: errInvalid, Message: "unsupported jsonrpc version"}}
			out <- marshalResponse(resp)
			continue
		}

		isRequest := msg.ID != nil

		switch msg.Method {
		case "exit":
			if !shutdownReceived {
				os.Exit(1)
			}
			os.Exit(0)

		case "shutdown":
			shutdownReceived = true
			if isRequest {
				out <- marshalResponse(rpcResponse{JSONRPC: "2.0", ID: msg.ID, Result: nil})
			}

		case "initialize":
			if isRequest {
				result, err := server.initialize(msg.Params)
				out <- marshalResponse(rpcResponse{JSONRPC: "2.0", ID: msg.ID, Result: result, Error: errRPC(err)})
			}

		case "textDocument/didOpen":
			_ = server.didOpen(msg.Params)
		case "textDocument/didChange":
			_ = server.didChange(msg.Params)
		case "textDocument/didClose":
			_ = server.didClose(msg.Params)
		case "textDocument/didSave":
			_ = server.didSave(msg.Params)
		case "textDocument/completion":
			if isRequest {
				result, err := server.completion(msg.Params)
				out <- marshalResponse(rpcResponse{JSONRPC: "2.0", ID: msg.ID, Result: result, Error: errRPC(err)})
			}
		case "textDocument/hover":
			if isRequest {
				result, err := server.hover(msg.Params)
				out <- marshalResponse(rpcResponse{JSONRPC: "2.0", ID: msg.ID, Result: result, Error: errRPC(err)})
			}
		case "textDocument/signatureHelp":
			if isRequest {
				result, err := server.signatureHelp(msg.Params)
				out <- marshalResponse(rpcResponse{JSONRPC: "2.0", ID: msg.ID, Result: result, Error: errRPC(err)})
			}
		case "textDocument/definition":
			if isRequest {
				result, err := server.definition(msg.Params)
				out <- marshalResponse(rpcResponse{JSONRPC: "2.0", ID: msg.ID, Result: result, Error: errRPC(err)})
			}
		case "textDocument/references":
			if isRequest {
				result, err := server.references(msg.Params)
				out <- marshalResponse(rpcResponse{JSONRPC: "2.0", ID: msg.ID, Result: result, Error: errRPC(err)})
			}
		case "textDocument/rename":
			if isRequest {
				result, err := server.rename(msg.Params)
				out <- marshalResponse(rpcResponse{JSONRPC: "2.0", ID: msg.ID, Result: result, Error: errRPC(err)})
			}
		case "textDocument/formatting":
			if isRequest {
				result, err := server.formatting(msg.Params)
				out <- marshalResponse(rpcResponse{JSONRPC: "2.0", ID: msg.ID, Result: result, Error: errRPC(err)})
			}
		case "textDocument/semanticTokens/full":
			if isRequest {
				result, err := server.semanticTokens(msg.Params)
				out <- marshalResponse(rpcResponse{JSONRPC: "2.0", ID: msg.ID, Result: result, Error: errRPC(err)})
			}

		default:
			if isRequest {
				out <- marshalResponse(rpcResponse{JSONRPC: "2.0", ID: msg.ID, Error: &rpcError{Code: errMethod, Message: "method not found: " + msg.Method}})
			}
			// Unknown notifications are ignored per spec.
		}
	}
}

func errRPC(err error) *rpcError {
	if err == nil {
		return nil
	}
	return &rpcError{Code: errInternal, Message: err.Error()}
}

// readMessage reads one Content-Length framed JSON-RPC message from r.
func readMessage(r *bufio.Reader) ([]byte, error) {
	contentLength := -1
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "content-length:") {
			n, err := strconv.Atoi(strings.TrimSpace(line[len("Content-Length:"):]))
			if err != nil {
				return nil, errors.New("invalid Content-Length")
			}
			contentLength = n
		}
	}
	if contentLength < 0 {
		return nil, errors.New("missing Content-Length header")
	}
	body := make([]byte, contentLength)
	if _, err := io.ReadFull(r, body); err != nil {
		return nil, err
	}
	return body, nil
}

// marshal builds a valid JSON-RPC response: a response must carry exactly one
// of "result" or "error". A nil result (e.g. hover/definition with no hit) is
// still emitted as "result": null so that clients like vscode-jsonrpc do not
// treat the message as invalid.
func marshalResponse(resp rpcResponse) []byte {
	m := map[string]any{"jsonrpc": "2.0"}
	if resp.ID != nil {
		m["id"] = resp.ID
	} else {
		m["id"] = nil
	}
	if resp.Error != nil {
		m["error"] = resp.Error
	} else {
		m["result"] = resp.Result
	}
	b, err := json.Marshal(m)
	if err != nil {
		return []byte(`{"jsonrpc":"2.0","id":null,"error":{"code":-32603,"message":"internal marshal error"}}`)
	}
	return b
}

// writeMessage writes one Content-Length framed message to w.
func writeMessage(w io.Writer, body []byte) error {
	if _, err := fmt.Fprintf(w, "Content-Length: %d\r\n\r\n", len(body)); err != nil {
		return err
	}
	_, err := w.Write(body)
	return err
}
