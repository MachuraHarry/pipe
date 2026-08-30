package object

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestHttpServerBasic(t *testing.T) {
	handlerResp := MapFromGo(map[string]Object{
		"status": &Integer{Value: 201},
		"headers": MapFromGo(map[string]Object{
			"X-Custom": &String{Value: "hello"},
		}),
		"body": &String{Value: "Hello from Pipe"},
	})
	handler := &BuiltinInfo{
		Name: "test_handler",
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return err("expected 1 arg")
			}
			req, ok := args[0].(*Map)
			if !ok {
				return err("expected map")
			}
			method, _ := req.Get("method")
			path, _ := req.Get("path")
			if method.Inspect() != "GET" {
				return err("expected GET")
			}
			if path.Inspect() != "/api/test" {
				return err("expected /api/test")
			}
			return handlerResp
		},
	}

	port := "127.0.0.1:19876"
	result := bHttpServer(&String{Value: port}, handler)
	if result.Type() == ERROR {
		t.Fatalf("http_server failed: %s", result.Inspect())
	}
	server := result.(*HttpServer)
	defer bHttpClose(server)

	time.Sleep(50 * time.Millisecond)

	resp, err := http.Get("http://" + port + "/api/test")
	if err != nil {
		t.Fatalf("http.Get failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		t.Errorf("expected status 201, got %d", resp.StatusCode)
	}
	if resp.Header.Get("X-Custom") != "hello" {
		t.Errorf("expected X-Custom header 'hello', got '%s'", resp.Header.Get("X-Custom"))
	}
}

func TestHttpServerError(t *testing.T) {
	result := bHttpServer(&String{Value: "missing_handler"})
	if result.Type() != ERROR {
		t.Error("expected error for missing handler arg")
	}
}

func TestHttpCloseInvalid(t *testing.T) {
	result := bHttpClose(NILOBJ)
	if result.Type() != ERROR {
		t.Error("expected error for http_close with nil")
	}
}

func TestHttpServerNilHandler(t *testing.T) {
	handler := &BuiltinInfo{
		Name: "nil_handler",
		Fn: func(args ...Object) Object {
			return NILOBJ
		},
	}

	port := "127.0.0.1:19877"
	result := bHttpServer(&String{Value: port}, handler)
	if result.Type() == ERROR {
		t.Fatalf("http_server failed: %s", result.Inspect())
	}
	server := result.(*HttpServer)
	defer bHttpClose(server)

	time.Sleep(50 * time.Millisecond)

	resp, err := http.Get("http://" + port + "/anything")
	if err != nil {
		t.Fatalf("http.Get failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("expected status 204 for nil response, got %d", resp.StatusCode)
	}
}

func TestHttpServerErrorHandler(t *testing.T) {
	handler := &BuiltinInfo{
		Name: "error_handler",
		Fn: func(args ...Object) Object {
			return err("something broke")
		},
	}

	port := "127.0.0.1:19878"
	result := bHttpServer(&String{Value: port}, handler)
	if result.Type() == ERROR {
		t.Fatalf("http_server failed: %s", result.Inspect())
	}
	server := result.(*HttpServer)
	defer bHttpClose(server)

	time.Sleep(50 * time.Millisecond)

	resp, err := http.Get("http://" + port + "/fail")
	if err != nil {
		t.Fatalf("http.Get failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", resp.StatusCode)
	}
}

func TestHttpServerNonMapResponse(t *testing.T) {
	handler := &BuiltinInfo{
		Name: "text_handler",
		Fn: func(args ...Object) Object {
			return &String{Value: "plain text"}
		},
	}

	port := "127.0.0.1:19879"
	result := bHttpServer(&String{Value: port}, handler)
	if result.Type() == ERROR {
		t.Fatalf("http_server failed: %s", result.Inspect())
	}
	server := result.(*HttpServer)
	defer bHttpClose(server)

	time.Sleep(50 * time.Millisecond)

	resp, err := http.Get("http://" + port + "/text")
	if err != nil {
		t.Fatalf("http.Get failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestHttpServerType(t *testing.T) {
	hs := &HttpServer{Addr: ":8080"}
	if hs.Type() != "HTTP_SERVER" {
		t.Errorf("expected HTTP_SERVER, got %s", hs.Type())
	}
	if hs.Inspect() != "http-server::8080" {
		t.Errorf("expected 'http-server::8080', got '%s'", hs.Inspect())
	}
}

func TestHttpServerMultipleRequests(t *testing.T) {
	calls := 0
	handler := &BuiltinInfo{
		Name: "count_handler",
		Fn: func(args ...Object) Object {
			calls++
			return MapFromGo(map[string]Object{
				"status": &Integer{Value: 200},
				"body":   &String{Value: fmt.Sprintf("call %d", calls)},
			})
		},
	}

	port := "127.0.0.1:19880"
	result := bHttpServer(&String{Value: port}, handler)
	if result.Type() == ERROR {
		t.Fatalf("http_server failed: %s", result.Inspect())
	}
	server := result.(*HttpServer)
	defer bHttpClose(server)

	time.Sleep(50 * time.Millisecond)

	for i := 0; i < 3; i++ {
		resp, err := http.Get("http://" + port + "/count")
		if err != nil {
			t.Fatalf("request %d failed: %v", i, err)
		}
		if resp.StatusCode != 200 {
			t.Errorf("request %d: expected 200, got %d", i, resp.StatusCode)
		}
		resp.Body.Close()
	}

	if calls != 3 {
		t.Errorf("expected 3 handler calls, got %d", calls)
	}
}
