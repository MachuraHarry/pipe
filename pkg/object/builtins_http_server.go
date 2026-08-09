package object

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"
)

type HttpServer struct {
	Addr    string
	srv     *http.Server
	ln      net.Listener
	mu      sync.Mutex
	handler Object
}

func (hs *HttpServer) Type() ObjectType { return "HTTP_SERVER" }
func (hs *HttpServer) Inspect() string  { return fmt.Sprintf("http-server:%s", hs.Addr) }

var (
	httpServersMu sync.Mutex
	httpServers   = make(map[*HttpServer]bool)
)

func bHttpServer(args ...Object) Object {
	if ActiveProfile.Load().Name != "none" {
		if canErr := ActiveProfile.Load().CanNetwork(); canErr != nil {
			return err(canErr.Error())
		}
	} else if Sandbox.Enabled && !Sandbox.AllowNet {
		return sandboxBlock("http_server (network)")
	}
	if len(args) != 2 {
		return err("http_server expects 2 arguments (addr, handler)")
	}
	addr, ok := args[0].(*String)
	if !ok {
		return err("http_server: addr must be a string")
	}
	handler := args[1]

	ln, lnErr := net.Listen("tcp", addr.Value)
	if lnErr != nil {
		return err("http_server: " + lnErr.Error())
	}

	hs := &HttpServer{
		Addr:    addr.Value,
		ln:      ln,
		handler: handler,
	}

	srv := &http.Server{
		Addr:              addr.Value,
		ReadHeaderTimeout: 10 * time.Second,
		Handler:           http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hs.serveHTTP(w, r) }),
	}
	hs.srv = srv

	httpServersMu.Lock()
	httpServers[hs] = true
	httpServersMu.Unlock()

	go func() {
		srv.Serve(ln)
		httpServersMu.Lock()
		delete(httpServers, hs)
		httpServersMu.Unlock()
	}()

	return hs
}

func (hs *HttpServer) serveHTTP(w http.ResponseWriter, r *http.Request) {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	reqMap := hs.buildRequestMap(r)
	resp := hs.callHandler(reqMap)

	hs.writeResponse(w, resp)
}

func (hs *HttpServer) buildRequestMap(r *http.Request) *Map {
	bodyBytes, _ := io.ReadAll(r.Body)
	r.Body.Close()

	headers := make(map[string]Object)
	for k, vals := range r.Header {
		if len(vals) > 0 {
			headers[k] = &String{Value: vals[0]}
		}
	}

	query := make(map[string]Object)
	for k, vals := range r.URL.Query() {
		if len(vals) > 0 {
			query[k] = &String{Value: vals[0]}
		}
	}

	pairs := map[string]Object{
		"method":  &String{Value: r.Method},
		"path":    &String{Value: r.URL.Path},
		"query":   &Map{Pairs: query},
		"headers": &Map{Pairs: headers},
		"body":    &String{Value: string(bodyBytes)},
	}
	return &Map{Pairs: pairs}
}

func (hs *HttpServer) callHandler(req *Map) Object {
	if callUserFn != nil {
		return callUserFn(hs.handler, req)
	}
	if bi, ok := hs.handler.(*BuiltinInfo); ok {
		return bi.Fn(req)
	}
	return err("http_server: handler not callable")
}

func (hs *HttpServer) writeResponse(w http.ResponseWriter, resp Object) {
	if resp == nil || resp.Type() == NIL {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if errObj, ok := resp.(*Error); ok {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(errObj.Message))
		return
	}

	respMap, ok := resp.(*Map)
	if !ok {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(resp.Inspect()))
		return
	}

	statusCode := http.StatusOK
	if s, ok := respMap.Pairs["status"]; ok {
		if n, ok := ToInt(s); ok {
			statusCode = int(n)
		}
	}

	if h, ok := respMap.Pairs["headers"]; ok {
		if hm, ok := h.(*Map); ok {
			for k, v := range hm.Pairs {
				w.Header().Set(k, v.Inspect())
			}
		}
	}

	body := []byte{}
	if b, ok := respMap.Pairs["body"]; ok {
		body = []byte(b.Inspect())
	}

	w.WriteHeader(statusCode)
	if len(body) > 0 {
		w.Write(body)
	}
}

func bHttpClose(args ...Object) Object {
	if len(args) != 1 {
		return err("http_close expects 1 argument (server)")
	}
	hs, ok := args[0].(*HttpServer)
	if !ok {
		return err("http_close expects an HTTP server handle")
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	_ = hs.srv.Shutdown(shutdownCtx)
	hs.ln.Close()

	httpServersMu.Lock()
	delete(httpServers, hs)
	httpServersMu.Unlock()

	return NILOBJ
}
