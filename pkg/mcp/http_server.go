package mcp

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ServeHTTP starts a Streamable HTTP MCP server on addr (e.g. ":9090").
// It blocks until the server is shut down.
func (s *Server) ServeHTTP(addr string) error {
	return http.ListenAndServe(addr, s.HTTPHandler())
}

// HTTPHandler returns the Streamable HTTP handler for this server. Each call
// returns a handler with its own session registry.
func (s *Server) HTTPHandler() http.Handler {
	sessionMu := &sync.Mutex{}
	sessions := make(map[string]bool)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			s.handleHTTPPost(w, r, sessionMu, sessions)
		case http.MethodDelete:
			s.handleHTTPDelete(w, r, sessionMu, sessions)
		default:
			w.Header().Set("Allow", "POST, DELETE")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

func (s *Server) handleHTTPDelete(w http.ResponseWriter, r *http.Request, sessionMu *sync.Mutex, sessions map[string]bool) {
	sid := r.Header.Get("Mcp-Session-Id")
	sessionMu.Lock()
	delete(sessions, sid)
	sessionMu.Unlock()
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleHTTPPost(w http.ResponseWriter, r *http.Request, sessionMu *sync.Mutex, sessions map[string]bool) {
	sid := r.Header.Get("Mcp-Session-Id")
	sessionMu.Lock()
	if sid == "" || !sessions[sid] {
		sid = newSessionID()
		sessions[sid] = true
	}
	sessionMu.Unlock()

	w.Header().Set("Mcp-Session-Id", sid)

	content, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, makeError(nil, ErrCodeParseError, "Parse error: "+err.Error()))
		return
	}

	var reply any
	if len(content) == 0 {
		reply = makeError(nil, ErrCodeInvalidRequest, "Invalid Request: empty body")
	} else {
		reply = s.safeDispatch(content)
	}

	if strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
		writeSSE(w, reply)
		return
	}
	writeJSON(w, reply)
}

func (s *Server) safeDispatch(data []byte) (reply any) {
	defer func() {
		if r := recover(); r != nil {
			reply = makeError(nil, ErrCodeInternalError, fmt.Sprintf("Internal error: %v", r))
		}
	}()
	return s.dispatch(data)
}

func writeSSE(w http.ResponseWriter, reply any) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, reply)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	if reply == nil {
		// Notifications produce no reply.
		flusher.Flush()
		return
	}
	data, err := json.Marshal(reply)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "event: message\ndata: %s\n\n", data)
	flusher.Flush()
}

func writeJSON(w http.ResponseWriter, reply any) {
	w.Header().Set("Content-Type", "application/json")
	if reply == nil {
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte("{}"))
		return
	}
	data, err := json.Marshal(reply)
	if err != nil {
		http.Error(w, `{"error":{"code":-32603,"message":"internal error"}}`, http.StatusInternalServerError)
		return
	}
	w.Write(data)
}

func newSessionID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("sess-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
