package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/MachuraHarry/pipe/pkg/eval"
	"github.com/MachuraHarry/pipe/pkg/lexer"
	"github.com/MachuraHarry/pipe/pkg/object"
	"github.com/MachuraHarry/pipe/pkg/parser"
)

var outputBuf strings.Builder
var pipeVersion = "v0.9.0"

func init() {
	object.PrintHook = func(args ...object.Object) {
		for i, arg := range args {
			if i > 0 {
				outputBuf.WriteByte(' ')
			}
			outputBuf.WriteString(arg.Inspect())
		}
		outputBuf.WriteByte('\n')
	}
}

func runCode(code string) string {
	outputBuf.Reset()

	l := lexer.New(code)
	p := parser.New(l)
	program := p.ParseProgram()

	if errs := p.Errors(); len(errs) > 0 {
		result := "Parse errors:\n"
		for _, e := range errs {
			result += "  " + e + "\n"
		}
		return result
	}

	env := object.NewEnvironment()
	ctx := eval.NewEvalContext("<api>")
	result := ctx.Eval(program, env)
	if result != nil && result.Type() == object.ERROR {
		outputBuf.WriteString("Error: " + result.Inspect() + "\n")
	}
	return outputBuf.String()
}

// ---- types ----

type RunRequest struct {
	Code string `json:"code"`
}

type RunResponse struct {
	Output  string `json:"output"`
	Version string `json:"version,omitempty"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// ---- middleware ----

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

// ---- handlers ----

func runHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "use POST"})
		return
	}

	var req RunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid JSON: " + err.Error()})
		return
	}

	if req.Code == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "missing 'code'"})
		return
	}

	if len(req.Code) > 50000 {
		writeJSON(w, http.StatusRequestEntityTooLarge, ErrorResponse{Error: "code exceeds 50,000 chars"})
		return
	}

	output := runCode(req.Code)
	writeJSON(w, http.StatusOK, RunResponse{Output: output, Version: pipeVersion})
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "version": pipeVersion})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// ---- main ----

func main() {
	addr := flag.String("addr", ":3001", "listen address")
	maxTime := flag.Duration("timeout", 10*time.Second, "max execution time per request")
	flag.Parse()

	if p := os.Getenv("PORT"); p != "" && *addr == ":3001" {
		*addr = ":" + p
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/run", runHandler)
	mux.HandleFunc("/health", healthHandler)

	handler := cors(withLogging(mux))
	handler = http.TimeoutHandler(handler, *maxTime, `{"error":"execution timed out"}`)

	server := &http.Server{
		Addr:         *addr,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: *maxTime + 2*time.Second,
	}

	fmt.Printf("pipe-api %s listening on %s (timeout: %s)\n", pipeVersion, *addr, *maxTime)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
